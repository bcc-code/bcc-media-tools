package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/bcc-code/bcc-media-flows/services/vidispine"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vscommon"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EditorialAPI backs the editorial approval tool: review sessions of markers
// that get accepted/rejected for publishing. Sessions are persisted in SQLite;
// markers can be imported from Mediabanken (Vidispine) chapters.
type EditorialAPI struct {
	store     *editorial.Store
	vidispine vidispine.Client
}

func NewEditorialAPI(store *editorial.Store, vs vidispine.Client) *EditorialAPI {
	return &EditorialAPI{store: store, vidispine: vs}
}

// requireEditorial authenticates the caller and checks editorial access. When
// needEdit is true the caller must have edit rights (add/remove/edit markers);
// otherwise plain tool access (see/accept/reject) is enough. Returns the email.
func requireEditorial[T any](req *connect.Request[T], needEdit bool) (string, error) {
	email := getEmail(req)
	if email == "" {
		return "", connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	perms := PermissionsForEmail(email)
	if needEdit {
		if !perms.CanEditorialEdit() {
			return "", connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized to edit editorial sessions"))
		}
	} else if !perms.CanEditorial() {
		return "", connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized to use the editorial tool"))
	}
	return email, nil
}

// editorialErr maps store errors to appropriate connect codes.
func editorialErr(err error) error {
	if errors.Is(err, editorial.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func (e EditorialAPI) ListEditorialSessions(ctx context.Context, req *connect.Request[apiv1.Void]) (*connect.Response[apiv1.ListEditorialSessionsResponse], error) {
	if _, err := requireEditorial(req, false); err != nil {
		return nil, err
	}
	sessions, err := e.store.ListSessions(ctx)
	if err != nil {
		return nil, editorialErr(err)
	}
	resp := &apiv1.ListEditorialSessionsResponse{}
	for i := range sessions {
		resp.Sessions = append(resp.Sessions, editorialSessionToProto(&sessions[i]))
	}
	return connect.NewResponse(resp), nil
}

func (e EditorialAPI) CreateEditorialSession(ctx context.Context, req *connect.Request[apiv1.CreateEditorialSessionRequest]) (*connect.Response[apiv1.EditorialSession], error) {
	email, err := requireEditorial(req, true)
	if err != nil {
		return nil, err
	}
	vxID := req.Msg.GetVXID()
	if vxID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing VXID"))
	}
	sess, err := e.store.CreateSession(ctx, vxID, req.Msg.GetTitle(), email)
	if err != nil {
		return nil, editorialErr(err)
	}
	return connect.NewResponse(editorialSessionToProto(sess)), nil
}

func (e EditorialAPI) GetEditorialSession(ctx context.Context, req *connect.Request[apiv1.GetEditorialSessionRequest]) (*connect.Response[apiv1.EditorialSession], error) {
	if _, err := requireEditorial(req, false); err != nil {
		return nil, err
	}
	sess, err := e.store.GetSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(err)
	}
	return connect.NewResponse(editorialSessionToProto(sess)), nil
}

func (e EditorialAPI) SaveEditorialSession(ctx context.Context, req *connect.Request[apiv1.SaveEditorialSessionRequest]) (*connect.Response[apiv1.EditorialSession], error) {
	if _, err := requireEditorial(req, true); err != nil {
		return nil, err
	}
	markers := make([]editorial.Marker, 0, len(req.Msg.GetMarkers()))
	for _, m := range req.Msg.GetMarkers() {
		markers = append(markers, protoToEditorialMarker(m))
	}
	sess, err := e.store.SaveSession(ctx, req.Msg.GetId(), req.Msg.GetTitle(), markers)
	if err != nil {
		return nil, editorialErr(err)
	}
	return connect.NewResponse(editorialSessionToProto(sess)), nil
}

func (e EditorialAPI) SetEditorialPublish(ctx context.Context, req *connect.Request[apiv1.SetEditorialPublishRequest]) (*connect.Response[apiv1.Void], error) {
	if _, err := requireEditorial(req, false); err != nil {
		return nil, err
	}
	if req.Msg.GetSessionId() == "" || req.Msg.GetMarkerId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing session_id or marker_id"))
	}
	if err := e.store.SetPublish(ctx, req.Msg.GetSessionId(), req.Msg.GetMarkerId(), req.Msg.GetPublish()); err != nil {
		return nil, editorialErr(err)
	}
	return connect.NewResponse(&apiv1.Void{}), nil
}

func (e EditorialAPI) DeleteEditorialSession(ctx context.Context, req *connect.Request[apiv1.DeleteEditorialSessionRequest]) (*connect.Response[apiv1.Void], error) {
	if _, err := requireEditorial(req, true); err != nil {
		return nil, err
	}
	if err := e.store.DeleteSession(ctx, req.Msg.GetId()); err != nil {
		return nil, editorialErr(err)
	}
	return connect.NewResponse(&apiv1.Void{}), nil
}

// ImportEditorialMarkers pulls chapter markers from Mediabanken (Vidispine) for
// the session's asset and returns them as candidate rows. It does NOT save; the
// client merges them into the table and saves explicitly.
func (e EditorialAPI) ImportEditorialMarkers(ctx context.Context, req *connect.Request[apiv1.ImportEditorialMarkersRequest]) (*connect.Response[apiv1.ImportEditorialMarkersResponse], error) {
	if _, err := requireEditorial(req, true); err != nil {
		return nil, err
	}
	sess, err := e.store.GetSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(err)
	}

	markers, err := e.importFromVidispine(sess.VXID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("import markers: %w", err))
	}
	return connect.NewResponse(&apiv1.ImportEditorialMarkersResponse{Markers: markers}), nil
}

// importFromVidispine mirrors the export tool's chapter extraction
// (export.go getSubclips): fetch the asset's clips, then their chapter metadata,
// mapping each chapter to an editorial marker (title → name, subclip-type →
// type, timecodes → start/end ms).
func (e EditorialAPI) importFromVidispine(vxID string) ([]*apiv1.EditorialMarker, error) {
	exportData, err := vidispine.GetDataForExport(e.vidispine, vxID, nil, nil, "", false)
	if err != nil {
		return nil, err
	}
	rawChapters, err := vidispine.GetChapterMetaForClips(e.vidispine, exportData.Clips)
	if err != nil {
		return nil, err
	}

	var out []*apiv1.EditorialMarker
	for _, c := range rawChapters {
		titleFields := c.Meta.Terse["title"]
		if len(titleFields) == 0 {
			continue
		}
		startSec, _ := vscommon.TCToSeconds(titleFields[0].Start)
		endSec, _ := vscommon.TCToSeconds(titleFields[0].End)
		out = append(out, &apiv1.EditorialMarker{
			Name:    c.Meta.Get(vscommon.FieldTitle, ""),
			Type:    c.Meta.Get(vscommon.FieldSubclipType, ""),
			StartMs: int64(startSec * 1000),
			EndMs:   int64(endSec * 1000),
			Source:  editorial.SourceImport,
		})
	}
	return out, nil
}

// ExportEditorialSession renders the review to a CSV file (the deliverable that
// replaces the manual Excel sheet) and marks the session as exported. The file
// bytes are returned inline; the client saves them.
func (e EditorialAPI) ExportEditorialSession(ctx context.Context, req *connect.Request[apiv1.ExportEditorialSessionRequest]) (*connect.Response[apiv1.ExportEditorialSessionResponse], error) {
	if _, err := requireEditorial(req, false); err != nil {
		return nil, err
	}
	sess, err := e.store.GetSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(err)
	}

	csvBytes, err := editorialSessionToCSV(sess)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build csv: %w", err))
	}

	if _, err := e.store.MarkExported(ctx, sess.ID); err != nil {
		return nil, editorialErr(err)
	}

	return connect.NewResponse(&apiv1.ExportEditorialSessionResponse{
		Filename:    editorialExportFilename(sess),
		ContentType: "text/csv",
		Data:        csvBytes,
	}), nil
}

// editorialSessionToCSV renders a session's markers as a semicolon-separated CSV
// (Norwegian Excel locale uses ';'), with a UTF-8 BOM so Excel reads æ/ø/å
// correctly. Columns mirror the review table.
func editorialSessionToCSV(s *editorial.Session) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM so Excel reads æ/ø/å correctly
	w := csv.NewWriter(&buf)
	w.Comma = ';'

	header := []string{"Hvem eller hva", "Type", "Start", "Slutt", "Varighet", "Publiseres"}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, m := range s.Markers {
		publish := "Nei"
		if m.Publish {
			publish = "Ja"
		}
		row := []string{
			m.Name,
			m.Type,
			formatMillisTC(m.StartMS),
			formatMillisTC(m.EndMS),
			formatMillisTC(m.EndMS - m.StartMS),
			publish,
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// formatMillisTC renders a duration/offset in milliseconds as HH:MM:SS.
func formatMillisTC(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	totalSec := ms / 1000
	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	sec := totalSec % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
}

// editorialExportFilename builds a safe CSV filename from the session title
// (falling back to the VXID).
func editorialExportFilename(s *editorial.Session) string {
	base := s.Title
	if base == "" {
		base = s.VXID
	}
	if base == "" {
		base = "editorial"
	}
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		case r == ' ':
			return '-'
		default:
			return -1
		}
	}, base)
	if safe == "" {
		safe = "editorial"
	}
	return safe + ".csv"
}

func editorialSessionToProto(s *editorial.Session) *apiv1.EditorialSession {
	out := &apiv1.EditorialSession{
		Id:        s.ID,
		VXID:      s.VXID,
		Title:     s.Title,
		Status:    s.Status,
		CreatedBy: s.CreatedBy,
		CreatedAt: timestamppb.New(s.CreatedAt),
		UpdatedAt: timestamppb.New(s.UpdatedAt),
	}
	if s.ExportedAt != nil {
		out.ExportedAt = timestamppb.New(*s.ExportedAt)
	}
	for _, m := range s.Markers {
		out.Markers = append(out.Markers, editorialMarkerToProto(m))
	}
	return out
}

func editorialMarkerToProto(m editorial.Marker) *apiv1.EditorialMarker {
	return &apiv1.EditorialMarker{
		Id:        m.ID,
		SortOrder: m.SortOrder,
		Name:      m.Name,
		Type:      m.Type,
		StartMs:   m.StartMS,
		EndMs:     m.EndMS,
		Publish:   m.Publish,
		Source:    m.Source,
	}
}

// protoToEditorialMarker maps an incoming marker for a save. SortOrder is
// assigned by the store from list position, so any incoming value is ignored.
func protoToEditorialMarker(m *apiv1.EditorialMarker) editorial.Marker {
	return editorial.Marker{
		ID:      m.GetId(),
		Name:    m.GetName(),
		Type:    m.GetType(),
		StartMS: m.GetStartMs(),
		EndMS:   m.GetEndMs(),
		Publish: m.GetPublish(),
		Source:  m.GetSource(),
	}
}
