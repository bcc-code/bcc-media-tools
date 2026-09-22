package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"bcc-media-tools/playout"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bcc-code/bcc-media-flows/services/cantemo"
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
	cantemo   *cantemo.Client
	playout   playoutClient
}

type playoutClient interface {
	GetManifest(context.Context, string, ...string) (*playout.Manifest, error)
	ListEvents(context.Context) ([]playout.Event, error)
}

func NewEditorialAPI(store *editorial.Store, vs vidispine.Client, cantemoClient *cantemo.Client, playoutClient playoutClient) *EditorialAPI {
	return &EditorialAPI{store: store, vidispine: vs, cantemo: cantemoClient, playout: playoutClient}
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
	out := editorialSessionToProto(sess)
	// The session (which the caller is authorized to see) is the authorization
	// for its preview — resolve the URL here so the client never asks for a
	// video by VXID. Best-effort: a preview failure shouldn't block the review.
	if url, err := e.cantemo.GetPreviewUrl(sess.VXID); err == nil {
		out.PreviewUrl = url
	}
	return connect.NewResponse(out), nil
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
	if err := e.store.SetPublish(ctx, req.Msg.GetSessionId(), req.Msg.GetMarkerId(), req.Msg.GetPublishBmm(), req.Msg.GetPublishBcc()); err != nil {
		return nil, editorialErr(err)
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
	if err := e.store.SetComment(ctx, req.Msg.GetSessionId(), req.Msg.GetMarkerId(), req.Msg.GetComment()); err != nil {
		return nil, editorialErr(err)
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
	if err := e.store.SetName(ctx, req.Msg.GetSessionId(), req.Msg.GetMarkerId(), req.Msg.GetName()); err != nil {
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

// ImportEditorialMarkersFromPlayout pulls every content-manifest entry for the
// supplied Playout event. Like the Vidispine import, it only returns candidate
// rows; the client decides whether to save them.
func (e EditorialAPI) ImportEditorialMarkersFromPlayout(ctx context.Context, req *connect.Request[apiv1.ImportEditorialMarkersFromPlayoutRequest]) (*connect.Response[apiv1.ImportEditorialMarkersResponse], error) {
	if _, err := requireEditorial(req, true); err != nil {
		return nil, err
	}
	if req.Msg.GetId() == "" || req.Msg.GetEventId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing id or event_id"))
	}
	sess, err := e.store.GetSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(err)
	}

	// A Playout event spans a whole conference, not one meeting, so without
	// this recording's wall-clock window we can't tell which manifest
	// entries are its — required, not best-effort.
	window, err := recordingWindowForItem(e.vidispine, sess.VXID)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("cannot place %s on the wall clock: %w", sess.VXID, err))
	}

	markers, err := e.importFromPlayout(ctx, req.Msg.GetEventId(), window)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("import Playout markers: %w", err))
	}
	return connect.NewResponse(&apiv1.ImportEditorialMarkersResponse{Markers: markers}), nil
}

// ListPlayoutEvents lists every event known to the tenant so the client can
// offer a picker instead of requiring a Playout event id to be typed in.
func (e EditorialAPI) ListPlayoutEvents(ctx context.Context, req *connect.Request[apiv1.ListPlayoutEventsRequest]) (*connect.Response[apiv1.ListPlayoutEventsResponse], error) {
	if _, err := requireEditorial(req, true); err != nil {
		return nil, err
	}
	if e.playout == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("playout client is not configured"))
	}
	events, err := e.playout.ListEvents(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list Playout events: %w", err))
	}
	resp := &apiv1.ListPlayoutEventsResponse{}
	for _, ev := range events {
		resp.Events = append(resp.Events, &apiv1.PlayoutEvent{
			Id:             ev.ID,
			Name:           ev.Name,
			Date:           ev.Date,
			Status:         ev.Status,
			ProductionUnit: ev.ProductionUnit,
		})
	}
	return connect.NewResponse(resp), nil
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

// importFromPlayout maps content actions inside the recording's wall-clock
// window to editorial candidates, narrowing the conference-wide event down
// to this one meeting; offsets are measured from the window start.
func (e EditorialAPI) importFromPlayout(ctx context.Context, eventID string, window recordingWindow) ([]*apiv1.EditorialMarker, error) {
	if e.playout == nil {
		return nil, fmt.Errorf("playout client is not configured")
	}
	if window.IsZero() {
		return nil, fmt.Errorf("recording window is required")
	}
	// Omitting the type filter asks Playout for every available content type,
	// including types added by the API in the future.
	manifest, err := e.playout.GetManifest(ctx, eventID)
	if err != nil {
		return nil, err
	}

	out := make([]*apiv1.EditorialMarker, 0, len(manifest.Entries))
	var currentSpeaker *apiv1.EditorialMarker
	var currentSpeakerVerses map[string]struct{}
	for _, entry := range manifest.Entries {
		// Entries from the event's other meetings are not part of this
		// recording; mapping them would place markers past its end.
		if !window.Contains(entry.Timestamp) {
			continue
		}
		if entry.Type == playout.ContentTypeScripture {
			// Scripture actions belong to the current speaker segment rather
			// than forming standalone editorial rows. A scripture before the
			// first speaker has no segment to attach to and is ignored.
			if currentSpeaker != nil && entry.Label != "" {
				if _, exists := currentSpeakerVerses[entry.Label]; exists {
					continue
				}
				if currentSpeaker.BibleVerses != "" {
					currentSpeaker.BibleVerses += ", "
				}
				currentSpeaker.BibleVerses += entry.Label
				currentSpeakerVerses[entry.Label] = struct{}{}
			}
			continue
		}

		marker := &apiv1.EditorialMarker{
			Type:    editorialTypeFromPlayout(entry.Type),
			StartMs: entry.Timestamp.Sub(window.Start).Milliseconds(),
			Source:  editorial.SourceImport,
		}
		// The next content action marks the end of the previous segment.
		// Scripture entries are handled above and intentionally do not create
		// a boundary of their own.
		if len(out) > 0 {
			out[len(out)-1].EndMs = marker.StartMs
		}
		switch entry.Type {
		case playout.ContentTypeSpeaker:
			var data struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(entry.Data, &data) == nil && data.Name != "" {
				marker.Contributors = data.Name
			} else {
				marker.Contributors = entry.Label
			}
			currentSpeaker = marker
			currentSpeakerVerses = map[string]struct{}{}
		default:
			// Non-speaker content, such as a song, uses the Playout label as
			// its editorial title. Speaker names live in Contributors instead.
			marker.Name = entry.Label
			// A song or other content segment ends the active speaker segment;
			// subsequent scripture must not be attributed to that speaker.
			currentSpeaker = nil
			currentSpeakerVerses = nil
		}
		out = append(out, marker)
	}
	// Nothing follows the final segment to bound it, but the recording itself ends so the window's end is its end.
	if len(out) > 0 {
		out[len(out)-1].EndMs = window.End.Sub(window.Start).Milliseconds()
	}
	return out, nil
}

// editorialTypeFromPlayout maps Playout's content types to the values used by
// the editorial type selector. Unknown types are preserved so importing new
// Playout content types remains forward-compatible.
func editorialTypeFromPlayout(playoutType string) string {
	switch playoutType {
	case playout.ContentTypeSpeaker:
		return "tale"
	case playout.ContentTypeSong:
		return "sang"
	default:
		return playoutType
	}
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
	for _, m := range s.Markers {
		out.Markers = append(out.Markers, editorialMarkerToProto(m))
	}
	return out
}

func editorialMarkerToProto(m editorial.Marker) *apiv1.EditorialMarker {
	return &apiv1.EditorialMarker{
		Id:           m.ID,
		SortOrder:    m.SortOrder,
		Name:         m.Name,
		Contributors: m.Contributors,
		Comment:      m.Comment,
		BibleVerses:  m.BibleVerses,
		Type:         m.Type,
		StartMs:      m.StartMS,
		EndMs:        m.EndMS,
		PublishBmm:   m.PublishBMM,
		PublishBcc:   m.PublishBCC,
		Source:       m.Source,
	}
}

// protoToEditorialMarker maps an incoming marker for a save. SortOrder is
// assigned by the store from list position, so any incoming value is ignored.
func protoToEditorialMarker(m *apiv1.EditorialMarker) editorial.Marker {
	return editorial.Marker{
		ID:           m.GetId(),
		Name:         m.GetName(),
		Contributors: m.GetContributors(),
		Comment:      m.GetComment(),
		BibleVerses:  m.GetBibleVerses(),
		Type:         m.GetType(),
		StartMS:      m.GetStartMs(),
		EndMS:        m.GetEndMs(),
		PublishBMM:   m.GetPublishBmm(),
		PublishBCC:   m.GetPublishBcc(),
		Source:       m.GetSource(),
	}
}
