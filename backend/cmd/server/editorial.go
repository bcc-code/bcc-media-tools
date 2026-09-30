package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/bcc-code/bcc-media-flows/services/cantemo"
	"github.com/bcc-code/bcc-media-flows/services/vidispine"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vscommon"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// errEditorialNotFound is returned when a session (or a marker within it) does
// not exist.
var errEditorialNotFound = errors.New("editorial: session not found")

// EditorialAPI backs the editorial approval tool: review sessions of markers
// that get accepted/rejected for publishing. Sessions are persisted in SQLite;
// markers can be imported from Mediabanken (Vidispine) chapters.
type EditorialAPI struct {
	db        *sql.DB
	queries   *editorial.Queries
	vidispine vidispine.Client
	cantemo   *cantemo.Client
}

func NewEditorialAPI(db *sql.DB, vs vidispine.Client, cantemoClient *cantemo.Client) *EditorialAPI {
	return &EditorialAPI{db: db, queries: editorial.New(db), vidispine: vs, cantemo: cantemoClient}
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

// editorialErr maps DB errors to appropriate connect codes.
func editorialErr(err error) error {
	if errors.Is(err, errEditorialNotFound) || errors.Is(err, sql.ErrNoRows) {
		return connect.NewError(connect.CodeNotFound, errEditorialNotFound)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func (e EditorialAPI) ListEditorialSessions(ctx context.Context, req *connect.Request[apiv1.Void]) (*connect.Response[apiv1.ListEditorialSessionsResponse], error) {
	if _, err := requireEditorial(req, false); err != nil {
		return nil, err
	}
	sessions, err := e.queries.ListSessions(ctx)
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: list sessions: %w", err))
	}
	resp := &apiv1.ListEditorialSessionsResponse{}
	for _, s := range sessions {
		resp.Sessions = append(resp.Sessions, editorialSessionToProto(s, nil))
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
	now := time.Now().UnixMilli()
	sess := editorial.Session{
		ID:        uuid.NewString(),
		Vxid:      vxID,
		Title:     req.Msg.GetTitle(),
		Status:    editorial.StatusDraft,
		CreatedBy: email,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := e.queries.CreateSession(ctx, editorial.CreateSessionParams(sess)); err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: create session: %w", err))
	}
	return connect.NewResponse(editorialSessionToProto(sess, nil)), nil
}

func (e EditorialAPI) GetEditorialSession(ctx context.Context, req *connect.Request[apiv1.GetEditorialSessionRequest]) (*connect.Response[apiv1.EditorialSession], error) {
	if _, err := requireEditorial(req, false); err != nil {
		return nil, err
	}
	out, err := e.loadSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(err)
	}
	// The session (which the caller is authorized to see) is the authorization
	// for its preview — resolve the URL here so the client never asks for a
	// video by VXID. Best-effort: a preview failure shouldn't block the review.
	if url, err := e.cantemo.GetPreviewUrl(out.VXID); err == nil {
		out.PreviewUrl = url
	}
	return connect.NewResponse(out), nil
}

func (e EditorialAPI) SaveEditorialSession(ctx context.Context, req *connect.Request[apiv1.SaveEditorialSessionRequest]) (*connect.Response[apiv1.EditorialSession], error) {
	if _, err := requireEditorial(req, true); err != nil {
		return nil, err
	}
	if err := e.saveSession(ctx, req.Msg.GetId(), req.Msg.GetTitle(), req.Msg.GetMarkers()); err != nil {
		return nil, editorialErr(err)
	}
	out, err := e.loadSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(err)
	}
	return connect.NewResponse(out), nil
}

func (e EditorialAPI) SetEditorialPublish(ctx context.Context, req *connect.Request[apiv1.SetEditorialPublishRequest]) (*connect.Response[apiv1.Void], error) {
	if _, err := requireEditorial(req, false); err != nil {
		return nil, err
	}
	if req.Msg.GetSessionId() == "" || req.Msg.GetMarkerId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing session_id or marker_id"))
	}
	sessionID, markerID := req.Msg.GetSessionId(), req.Msg.GetMarkerId()
	err := e.updateMarker(ctx, sessionID, func(qtx *editorial.Queries, now int64) (int64, error) {
		return qtx.SetMarkerPublish(ctx, editorial.SetMarkerPublishParams{
			PublishBmm: req.Msg.GetPublishBmm(),
			PublishBcc: req.Msg.GetPublishBcc(),
			UpdatedAt:  now,
			ID:         markerID,
			SessionID:  sessionID,
		})
	})
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: set publish: %w", err))
	}
	return connect.NewResponse(&apiv1.Void{}), nil
}

func (e EditorialAPI) SetEditorialComment(ctx context.Context, req *connect.Request[apiv1.SetEditorialCommentRequest]) (*connect.Response[apiv1.Void], error) {
	if _, err := requireEditorial(req, false); err != nil {
		return nil, err
	}
	if req.Msg.GetSessionId() == "" || req.Msg.GetMarkerId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing session_id or marker_id"))
	}
	sessionID, markerID := req.Msg.GetSessionId(), req.Msg.GetMarkerId()
	err := e.updateMarker(ctx, sessionID, func(qtx *editorial.Queries, now int64) (int64, error) {
		return qtx.SetMarkerComment(ctx, editorial.SetMarkerCommentParams{
			Comment:   req.Msg.GetComment(),
			UpdatedAt: now,
			ID:        markerID,
			SessionID: sessionID,
		})
	})
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: set comment: %w", err))
	}
	return connect.NewResponse(&apiv1.Void{}), nil
}

func (e EditorialAPI) SetEditorialName(ctx context.Context, req *connect.Request[apiv1.SetEditorialNameRequest]) (*connect.Response[apiv1.Void], error) {
	if _, err := requireEditorial(req, true); err != nil {
		return nil, err
	}
	if req.Msg.GetSessionId() == "" || req.Msg.GetMarkerId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing session_id or marker_id"))
	}
	sessionID, markerID := req.Msg.GetSessionId(), req.Msg.GetMarkerId()
	err := e.updateMarker(ctx, sessionID, func(qtx *editorial.Queries, now int64) (int64, error) {
		return qtx.SetMarkerName(ctx, editorial.SetMarkerNameParams{
			Name:      req.Msg.GetName(),
			UpdatedAt: now,
			ID:        markerID,
			SessionID: sessionID,
		})
	})
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: set name: %w", err))
	}
	return connect.NewResponse(&apiv1.Void{}), nil
}

func (e EditorialAPI) DeleteEditorialSession(ctx context.Context, req *connect.Request[apiv1.DeleteEditorialSessionRequest]) (*connect.Response[apiv1.Void], error) {
	if _, err := requireEditorial(req, true); err != nil {
		return nil, err
	}
	// Markers go with it via ON DELETE CASCADE.
	n, err := e.queries.DeleteSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: delete session: %w", err))
	}
	if n == 0 {
		return nil, editorialErr(errEditorialNotFound)
	}
	return connect.NewResponse(&apiv1.Void{}), nil
}

// loadSession returns a session with its markers ordered by sort_order.
func (e EditorialAPI) loadSession(ctx context.Context, id string) (*apiv1.EditorialSession, error) {
	sess, err := e.queries.GetSession(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errEditorialNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("editorial: get session: %w", err)
	}
	markers, err := e.queries.ListMarkersForSession(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("editorial: list markers: %w", err)
	}
	return editorialSessionToProto(sess, markers), nil
}

// saveSession updates the session title and fully replaces its markers in a
// single transaction. Markers with an empty ID are treated as new and get a
// generated id; sort_order is assigned from slice position so the caller
// controls ordering (any incoming sort_order is ignored).
func (e EditorialAPI) saveSession(ctx context.Context, id, title string, markers []*apiv1.EditorialMarker) error {
	now := time.Now().UnixMilli()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("editorial: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	qtx := e.queries.WithTx(tx)

	n, err := qtx.UpdateSessionTitle(ctx, editorial.UpdateSessionTitleParams{Title: title, UpdatedAt: now, ID: id})
	if err != nil {
		return fmt.Errorf("editorial: update session: %w", err)
	}
	if n == 0 {
		return errEditorialNotFound
	}

	if err := qtx.DeleteMarkersForSession(ctx, id); err != nil {
		return fmt.Errorf("editorial: clear markers: %w", err)
	}

	for i, m := range markers {
		mid := m.GetId()
		if mid == "" {
			mid = uuid.NewString()
		}
		source := m.GetSource()
		if source == "" {
			source = editorial.SourceManual
		}
		if err := qtx.InsertMarker(ctx, editorial.InsertMarkerParams{
			ID:           mid,
			SessionID:    id,
			SortOrder:    int64(i),
			Name:         m.GetName(),
			Contributors: m.GetContributors(),
			Comment:      m.GetComment(),
			BibleVerses:  m.GetBibleVerses(),
			Type:         m.GetType(),
			StartMs:      m.GetStartMs(),
			EndMs:        m.GetEndMs(),
			PublishBmm:   m.GetPublishBmm(),
			PublishBcc:   m.GetPublishBcc(),
			Source:       source,
			CreatedAt:    now,
			UpdatedAt:    now,
		}); err != nil {
			return fmt.Errorf("editorial: insert marker: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("editorial: commit: %w", err)
	}
	return nil
}

// updateMarker runs a single-marker update (which returns the affected row
// count) and bumps the session's updated_at, in one transaction. These are the
// narrow write paths used by the simple view instead of a full save. Returns
// errEditorialNotFound if the marker does not exist in the session.
func (e EditorialAPI) updateMarker(ctx context.Context, sessionID string, update func(qtx *editorial.Queries, now int64) (int64, error)) error {
	now := time.Now().UnixMilli()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	qtx := e.queries.WithTx(tx)

	n, err := update(qtx, now)
	if err != nil {
		return err
	}
	if n == 0 {
		return errEditorialNotFound
	}

	if err := qtx.TouchSession(ctx, editorial.TouchSessionParams{UpdatedAt: now, ID: sessionID}); err != nil {
		return fmt.Errorf("touch session: %w", err)
	}

	return tx.Commit()
}

func (e EditorialAPI) ImportEditorialMarkers(ctx context.Context, req *connect.Request[apiv1.ImportEditorialMarkersRequest]) (*connect.Response[apiv1.ImportEditorialMarkersResponse], error) {
	if _, err := requireEditorial(req, true); err != nil {
		return nil, err
	}
	sess, err := e.queries.GetSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(err)
	}

	markers, err := e.importFromVidispine(sess.Vxid)
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

func editorialSessionToProto(s editorial.Session, markers []editorial.Marker) *apiv1.EditorialSession {
	out := &apiv1.EditorialSession{
		Id:        s.ID,
		VXID:      s.Vxid,
		Title:     s.Title,
		Status:    s.Status,
		CreatedBy: s.CreatedBy,
		CreatedAt: timestamppb.New(time.UnixMilli(s.CreatedAt)),
		UpdatedAt: timestamppb.New(time.UnixMilli(s.UpdatedAt)),
	}
	for _, m := range markers {
		out.Markers = append(out.Markers, editorialMarkerToProto(m))
	}
	return out
}

func editorialMarkerToProto(m editorial.Marker) *apiv1.EditorialMarker {
	return &apiv1.EditorialMarker{
		Id:           m.ID,
		SortOrder:    int32(m.SortOrder),
		Name:         m.Name,
		Contributors: m.Contributors,
		Comment:      m.Comment,
		BibleVerses:  m.BibleVerses,
		Type:         m.Type,
		StartMs:      m.StartMs,
		EndMs:        m.EndMs,
		PublishBmm:   m.PublishBmm,
		PublishBcc:   m.PublishBcc,
		Source:       m.Source,
	}
}
