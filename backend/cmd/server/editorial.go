package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"bcc-media-tools/playout"
	"context"
	"database/sql"
	"encoding/json"
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
	playout   playoutClient
	// Tells open pages when a session changes.
	hub *editorialHub
}

type playoutClient interface {
	GetManifest(context.Context, string, ...string) (*playout.Manifest, error)
	ListEvents(context.Context) ([]playout.Event, error)
}

func NewEditorialAPI(db *sql.DB, vs vidispine.Client, cantemoClient *cantemo.Client, playoutClient playoutClient) *EditorialAPI {
	return &EditorialAPI{db: db, queries: editorial.New(db), vidispine: vs, cantemo: cantemoClient, playout: playoutClient, hub: newEditorialHub()}
}

// requireEditorial authenticates the caller and checks they have at least the
// given editorial role (see apiv1.EditorialRole). Returns the email.
func requireEditorial[T any](req *connect.Request[T], role apiv1.EditorialRole) (string, error) {
	email := getEmail(req)
	if email == "" {
		return "", connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	if PermissionsForEmail(email).EditorialRole() < role {
		return "", connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized: needs the %s editorial role", editorialRoleNames[role]))
	}
	return email, nil
}

var editorialRoleNames = map[apiv1.EditorialRole]string{
	apiv1.EditorialReviewer: "reviewer",
	apiv1.EditorialEditor:   "editor",
	apiv1.EditorialLive:     "live",
	apiv1.EditorialAdmin:    "admin",
}

// editorialErr maps DB errors to appropriate connect codes.
func editorialErr(err error) error {
	if errors.Is(err, errEditorialNotFound) || errors.Is(err, sql.ErrNoRows) {
		return connect.NewError(connect.CodeNotFound, errEditorialNotFound)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func (e EditorialAPI) ListEditorialSessions(ctx context.Context, req *connect.Request[apiv1.Void]) (*connect.Response[apiv1.ListEditorialSessionsResponse], error) {
	if _, err := requireEditorial(req, apiv1.EditorialReviewer); err != nil {
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
	email, err := requireEditorial(req, apiv1.EditorialLive)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	sess := editorial.Session{
		ID:        uuid.NewString(),
		Vxid:      req.Msg.GetVXID(),
		Title:     req.Msg.GetTitle(),
		Status:    editorial.StatusDraft,
		CreatedBy: email,
		CreatedAt: now,
		UpdatedAt: now,
		Kind:      editorial.KindRecording,
	}
	if req.Msg.GetLive() {
		// Not in Mediabanken yet: the recording is attached when logging is done.
		if req.Msg.GetPlayoutEventId() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing playout_event_id"))
		}
		sess.Vxid = ""
		sess.Kind = editorial.KindLive
		sess.PlayoutEventID = req.Msg.GetPlayoutEventId()
		sess.ClockStartMs = now
		if start := req.Msg.GetClockStart(); start != nil {
			sess.ClockStartMs = start.AsTime().UnixMilli()
		}
	} else if sess.Vxid == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing VXID"))
	}
	if err := e.queries.CreateSession(ctx, editorial.CreateSessionParams{
		ID:             sess.ID,
		Vxid:           sess.Vxid,
		Title:          sess.Title,
		Status:         sess.Status,
		CreatedBy:      sess.CreatedBy,
		CreatedAt:      sess.CreatedAt,
		UpdatedAt:      sess.UpdatedAt,
		Kind:           sess.Kind,
		PlayoutEventID: sess.PlayoutEventID,
		ClockStartMs:   sess.ClockStartMs,
	}); err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: create session: %w", err))
	}
	return connect.NewResponse(editorialSessionToProto(sess, nil)), nil
}

func (e EditorialAPI) GetEditorialSession(ctx context.Context, req *connect.Request[apiv1.GetEditorialSessionRequest]) (*connect.Response[apiv1.EditorialSession], error) {
	if _, err := requireEditorial(req, apiv1.EditorialReviewer); err != nil {
		return nil, err
	}
	out, err := e.loadSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(err)
	}
	// The session (which the caller is authorized to see) is the authorization
	// for its preview — resolve the URL here so the client never asks for a
	// video by VXID. Best-effort: a preview failure shouldn't block the review.
	// A live session has no asset to preview yet.
	if out.VXID != "" {
		if url, err := e.cantemo.GetPreviewUrl(out.VXID); err == nil {
			out.PreviewUrl = url
		}
	}
	return connect.NewResponse(out), nil
}

func (e EditorialAPI) SaveEditorialSession(ctx context.Context, req *connect.Request[apiv1.SaveEditorialSessionRequest]) (*connect.Response[apiv1.EditorialSession], error) {
	if _, err := requireEditorial(req, apiv1.EditorialLive); err != nil {
		return nil, err
	}
	if err := e.saveSession(ctx, req.Msg.GetId(), req.Msg.GetTitle(), req.Msg.GetMarkers()); err != nil {
		return nil, editorialErr(err)
	}
	e.hub.changed(req.Msg.GetId())
	out, err := e.loadSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(err)
	}
	return connect.NewResponse(out), nil
}

func (e EditorialAPI) SetEditorialPublish(ctx context.Context, req *connect.Request[apiv1.SetEditorialPublishRequest]) (*connect.Response[apiv1.Void], error) {
	if _, err := requireEditorial(req, apiv1.EditorialReviewer); err != nil {
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
	e.hub.changed(sessionID)
	return connect.NewResponse(&apiv1.Void{}), nil
}

func (e EditorialAPI) SetEditorialComment(ctx context.Context, req *connect.Request[apiv1.SetEditorialCommentRequest]) (*connect.Response[apiv1.Void], error) {
	if _, err := requireEditorial(req, apiv1.EditorialReviewer); err != nil {
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
	e.hub.changed(sessionID)
	return connect.NewResponse(&apiv1.Void{}), nil
}

func (e EditorialAPI) SetEditorialName(ctx context.Context, req *connect.Request[apiv1.SetEditorialNameRequest]) (*connect.Response[apiv1.Void], error) {
	if _, err := requireEditorial(req, apiv1.EditorialReviewer); err != nil {
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
	e.hub.changed(sessionID)
	return connect.NewResponse(&apiv1.Void{}), nil
}

func (e EditorialAPI) SetEditorialDetails(ctx context.Context, req *connect.Request[apiv1.SetEditorialDetailsRequest]) (*connect.Response[apiv1.Void], error) {
	if _, err := requireEditorial(req, apiv1.EditorialEditor); err != nil {
		return nil, err
	}
	if req.Msg.GetSessionId() == "" || req.Msg.GetMarkerId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing session_id or marker_id"))
	}
	sessionID, markerID := req.Msg.GetSessionId(), req.Msg.GetMarkerId()
	err := e.updateMarker(ctx, sessionID, func(qtx *editorial.Queries, now int64) (int64, error) {
		return qtx.SetMarkerDetails(ctx, editorial.SetMarkerDetailsParams{
			Name:         req.Msg.GetName(),
			Contributors: req.Msg.GetContributors(),
			Type:         req.Msg.GetType(),
			BibleVerses:  req.Msg.GetBibleVerses(),
			UpdatedAt:    now,
			ID:           markerID,
			SessionID:    sessionID,
		})
	})
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: set details: %w", err))
	}
	e.hub.changed(sessionID)
	return connect.NewResponse(&apiv1.Void{}), nil
}

func (e EditorialAPI) DeleteEditorialSession(ctx context.Context, req *connect.Request[apiv1.DeleteEditorialSessionRequest]) (*connect.Response[apiv1.Void], error) {
	if _, err := requireEditorial(req, apiv1.EditorialAdmin); err != nil {
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
	e.hub.changed(req.Msg.GetId())
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
// controls ordering (any incoming sort_order is ignored). Live sessions are
// kept in time order instead.
//
// Playout bookkeeping is server-side: a row stays synced only while the
// fields Playout fills are unchanged, and a Playout row that is left out is
// dismissed so the next sync doesn't bring it back.
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
	sess, err := qtx.GetSession(ctx, id)
	if err != nil {
		return fmt.Errorf("editorial: get session: %w", err)
	}
	stored, err := qtx.ListMarkersForSession(ctx, id)
	if err != nil {
		return fmt.Errorf("editorial: list markers: %w", err)
	}
	storedByID := make(map[string]editorial.Marker, len(stored))
	for _, m := range stored {
		storedByID[m.ID] = m
	}

	rows := make([]editorial.Marker, 0, len(markers))
	kept := map[string]bool{}
	for _, m := range markers {
		row := editorial.Marker{
			ID:           m.GetId(),
			SessionID:    id,
			Name:         m.GetName(),
			Contributors: m.GetContributors(),
			Comment:      m.GetComment(),
			BibleVerses:  m.GetBibleVerses(),
			Type:         m.GetType(),
			StartMs:      m.GetStartMs(),
			EndMs:        m.GetEndMs(),
			PublishBmm:   m.GetPublishBmm(),
			PublishBcc:   m.GetPublishBcc(),
			Source:       m.GetSource(),
			PlayoutKey:   m.GetPlayoutKey(),
		}
		if row.ID == "" {
			row.ID = uuid.NewString()
		}
		if row.Source == "" {
			row.Source = editorial.SourceManual
		}
		if prev, ok := storedByID[row.ID]; ok && prev.PlayoutKey == row.PlayoutKey {
			row.PlayoutSynced = prev.PlayoutSynced && !playoutFieldsChanged(prev, row)
		}
		if row.PlayoutKey != "" {
			kept[row.PlayoutKey] = true
		}
		rows = append(rows, row)
	}
	for _, m := range stored {
		if m.PlayoutKey != "" && !kept[m.PlayoutKey] {
			if err := qtx.DismissPlayoutKey(ctx, editorial.DismissPlayoutKeyParams{SessionID: id, PlayoutKey: m.PlayoutKey}); err != nil {
				return fmt.Errorf("editorial: dismiss playout row: %w", err)
			}
		}
	}
	if sess.Kind == editorial.KindLive {
		normalizeLiveMarkers(rows)
	}

	if err := replaceMarkers(ctx, qtx, id, rows, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("editorial: commit: %w", err)
	}
	return nil
}

// replaceMarkers swaps the session's markers for rows, in slice order.
func replaceMarkers(ctx context.Context, qtx *editorial.Queries, sessionID string, rows []editorial.Marker, now int64) error {
	if err := qtx.DeleteMarkersForSession(ctx, sessionID); err != nil {
		return fmt.Errorf("editorial: clear markers: %w", err)
	}
	for i, m := range rows {
		if err := qtx.InsertMarker(ctx, editorial.InsertMarkerParams{
			ID:            m.ID,
			SessionID:     sessionID,
			SortOrder:     int64(i),
			Name:          m.Name,
			Contributors:  m.Contributors,
			Comment:       m.Comment,
			BibleVerses:   m.BibleVerses,
			Type:          m.Type,
			StartMs:       m.StartMs,
			EndMs:         m.EndMs,
			PublishBmm:    m.PublishBmm,
			PublishBcc:    m.PublishBcc,
			Source:        m.Source,
			CreatedAt:     now,
			UpdatedAt:     now,
			PlayoutKey:    m.PlayoutKey,
			PlayoutSynced: m.PlayoutSynced,
		}); err != nil {
			return fmt.Errorf("editorial: insert marker: %w", err)
		}
	}
	return nil
}

// playoutFieldsChanged reports whether an editor touched a field Playout fills.
func playoutFieldsChanged(a, b editorial.Marker) bool {
	return a.Name != b.Name || a.Contributors != b.Contributors || a.BibleVerses != b.BibleVerses || a.Type != b.Type
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
	if _, err := requireEditorial(req, apiv1.EditorialLive); err != nil {
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

// ImportEditorialMarkersFromPlayout pulls every content-manifest entry for the
// supplied Playout event. Like the Vidispine import, it only returns candidate
// rows; the client decides whether to save them.
func (e EditorialAPI) ImportEditorialMarkersFromPlayout(ctx context.Context, req *connect.Request[apiv1.ImportEditorialMarkersFromPlayoutRequest]) (*connect.Response[apiv1.ImportEditorialMarkersResponse], error) {
	if _, err := requireEditorial(req, apiv1.EditorialLive); err != nil {
		return nil, err
	}
	if req.Msg.GetId() == "" || req.Msg.GetEventId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing id or event_id"))
	}
	sess, err := e.queries.GetSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(err)
	}

	// A Playout event spans a whole conference, not one meeting, so without
	// this recording's wall-clock window we can't tell which manifest
	// entries are its — required, not best-effort.
	window, manual, err := e.resolveWindow(sess, req.Msg.GetRecordingStart(), req.Msg.GetRecordingEnd())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("cannot place %s on the wall clock: %w", sess.Vxid, err))
	}

	markers, err := e.importFromPlayout(ctx, req.Msg.GetEventId(), window)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("import Playout markers: %w", err))
	}
	// Empty is the symptom of a window on the wrong time; saying so beats
	// returning nothing, which reads as "this event has no content".
	if len(markers) == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf(
			"no Playout content between %s and %s",
			window.Start.Format(time.RFC3339), window.End.Format(time.RFC3339)))
	}
	// Only hand-entered windows are worth keeping; derived ones are
	// recomputed each time and would go stale.
	if manual {
		n, err := e.queries.SetPlayoutWindow(ctx, editorial.SetPlayoutWindowParams{
			PlayoutEventID:   req.Msg.GetEventId(),
			RecordingStartMs: window.Start.UnixMilli(),
			RecordingEndMs:   window.End.UnixMilli(),
			UpdatedAt:        time.Now().UnixMilli(),
			ID:               sess.ID,
		})
		if err != nil {
			return nil, editorialErr(fmt.Errorf("editorial: set playout window: %w", err))
		}
		if n == 0 {
			return nil, editorialErr(errEditorialNotFound)
		}
	}
	return connect.NewResponse(&apiv1.ImportEditorialMarkersResponse{Markers: markers}), nil
}

// ListPlayoutEvents lists every event known to the tenant so the client can
// offer a picker instead of requiring a Playout event id to be typed in.
func (e EditorialAPI) ListPlayoutEvents(ctx context.Context, req *connect.Request[apiv1.ListPlayoutEventsRequest]) (*connect.Response[apiv1.ListPlayoutEventsResponse], error) {
	if _, err := requireEditorial(req, apiv1.EditorialLive); err != nil {
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

// resolveWindow picks the span to import against: supplied here, else stored
// on the session, else derived from metadata. The bool means "came from a
// human".
func (e EditorialAPI) resolveWindow(sess editorial.Session, start, end *timestamppb.Timestamp) (recordingWindow, bool, error) {
	if start != nil {
		return e.windowFromManualStart(sess.Vxid, start.AsTime(), end)
	}
	if stored, ok := storedRecordingWindow(sess); ok {
		return stored, false, nil
	}
	window, err := recordingWindowForItem(e.vidispine, sess.Vxid)
	return window, false, err
}

// storedRecordingWindow returns the window remembered from the last manual
// Playout import, if any (0 ms means unset).
func storedRecordingWindow(sess editorial.Session) (recordingWindow, bool) {
	if sess.RecordingStartMs == 0 || sess.RecordingEndMs == 0 {
		return recordingWindow{}, false
	}
	return recordingWindow{
		Start: time.UnixMilli(sess.RecordingStartMs).UTC(),
		End:   time.UnixMilli(sess.RecordingEndMs).UTC(),
	}, true
}

// windowFromManualStart builds a window around an editor-supplied start; the
// end comes from the request, or failing that the asset's duration.
func (e EditorialAPI) windowFromManualStart(vxID string, start time.Time, end *timestamppb.Timestamp) (recordingWindow, bool, error) {
	if end != nil {
		if !end.AsTime().After(start) {
			return recordingWindow{}, true, fmt.Errorf("recording end must be after the start")
		}
		return recordingWindow{Start: start.UTC(), End: end.AsTime().UTC()}, true, nil
	}
	meta, err := fetchRecordingMetadata(e.vidispine, vxID)
	if err != nil {
		return recordingWindow{}, true, err
	}
	seconds, err := durationFromMetadata(meta)
	if err != nil {
		return recordingWindow{}, true, fmt.Errorf("a recording end is required: %w", err)
	}
	return recordingWindow{
		Start: start.UTC(),
		End:   start.UTC().Add(time.Duration(seconds * float64(time.Second))),
	}, true, nil
}

// GetRecordingWindow reports the span the import would use, so the dialog can
// show and correct it first. An underivable window is reported in the response,
// not as an error: the dialog turns it into a prompt.
func (e EditorialAPI) GetRecordingWindow(ctx context.Context, req *connect.Request[apiv1.GetRecordingWindowRequest]) (*connect.Response[apiv1.GetRecordingWindowResponse], error) {
	if _, err := requireEditorial(req, apiv1.EditorialLive); err != nil {
		return nil, err
	}
	sess, err := e.queries.GetSession(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, editorialErr(err)
	}

	// A live session has no asset of its own: the attach dialog names the one
	// it is about to attach, and the session's stored window doesn't apply.
	vxID := sess.Vxid
	if req.Msg.GetVXID() != "" {
		vxID = req.Msg.GetVXID()
	}
	if vxID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing VXID"))
	}

	info, err := recordingInfoForItem(e.vidispine, vxID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &apiv1.GetRecordingWindowResponse{DurationMs: info.DurationMs}
	stored, hasStored := storedRecordingWindow(sess)
	hasStored = hasStored && vxID == sess.Vxid
	switch {
	case hasStored:
		resp.Start = timestamppb.New(stored.Start)
		resp.End = timestamppb.New(stored.End)
		resp.Source = "manual"
	case info.WindowErr == nil:
		resp.Start = timestamppb.New(info.Window.Start)
		resp.End = timestamppb.New(info.Window.End)
		resp.Source = "metadata"
	default:
		resp.Error = info.WindowErr.Error()
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

	candidates := playoutCandidates(manifest.Entries, window)
	out := make([]*apiv1.EditorialMarker, 0, len(candidates))
	for i, c := range candidates {
		marker := &apiv1.EditorialMarker{
			Name:         c.Name,
			Contributors: c.Contributors,
			BibleVerses:  c.BibleVerses,
			Type:         editorialTypeFromPlayout(c.PlayoutType),
			StartMs:      c.At.Sub(window.Start).Milliseconds(),
			Source:       editorial.SourceImport,
		}
		// The next content action marks the end of the previous segment.
		// Nothing follows the final segment to bound it, but the recording
		// itself ends, so the window's end is its end.
		if i+1 < len(candidates) {
			marker.EndMs = candidates[i+1].At.Sub(window.Start).Milliseconds()
		} else {
			marker.EndMs = window.End.Sub(window.Start).Milliseconds()
		}
		out = append(out, marker)
	}
	return out, nil
}

// playoutCandidate is one content action from a Playout manifest, mapped to
// editorial fields but still on the wall clock.
type playoutCandidate struct {
	// identifies the manifest entry across re-reads of the manifest
	Key          string
	PlayoutType  string
	At           time.Time
	Name         string
	Contributors string
	BibleVerses  string
}

// playoutKey identifies a manifest entry: Playout gives entries no id, but a
// content action's type and timestamp don't change between reads.
func playoutKey(entry playout.ManifestEntry) string {
	return entry.Type + "|" + entry.Timestamp.UTC().Format(time.RFC3339Nano)
}

// playoutCandidates maps the manifest entries inside window to candidate rows,
// in manifest order. Scripture doesn't form rows of its own: it is collected on
// the speaker segment it falls in.
func playoutCandidates(entries []playout.ManifestEntry, window recordingWindow) []playoutCandidate {
	out := make([]playoutCandidate, 0, len(entries))
	currentSpeaker := -1
	var currentSpeakerVerses map[string]struct{}
	for _, entry := range entries {
		// Entries from the event's other meetings are not part of this
		// recording; mapping them would place markers past its end.
		if !window.Contains(entry.Timestamp) {
			continue
		}
		if entry.Type == playout.ContentTypeScripture {
			// Scripture actions belong to the current speaker segment rather
			// than forming standalone editorial rows. A scripture before the
			// first speaker has no segment to attach to and is ignored.
			if currentSpeaker >= 0 && entry.Label != "" {
				if _, exists := currentSpeakerVerses[entry.Label]; exists {
					continue
				}
				speaker := &out[currentSpeaker]
				if speaker.BibleVerses != "" {
					speaker.BibleVerses += ", "
				}
				speaker.BibleVerses += entry.Label
				currentSpeakerVerses[entry.Label] = struct{}{}
			}
			continue
		}

		c := playoutCandidate{Key: playoutKey(entry), PlayoutType: entry.Type, At: entry.Timestamp}
		switch entry.Type {
		case playout.ContentTypeSpeaker:
			var data struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(entry.Data, &data) == nil && data.Name != "" {
				c.Contributors = data.Name
			} else {
				c.Contributors = entry.Label
			}
			currentSpeaker = len(out)
			currentSpeakerVerses = map[string]struct{}{}
		default:
			// Non-speaker content, such as a song, uses the Playout label as
			// its editorial title. Speaker names live in Contributors instead.
			c.Name = entry.Label
			// A song or other content segment ends the active speaker segment;
			// subsequent scripture must not be attributed to that speaker.
			currentSpeaker = -1
			currentSpeakerVerses = nil
		}
		out = append(out, c)
	}
	return out
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

func editorialSessionToProto(s editorial.Session, markers []editorial.Marker) *apiv1.EditorialSession {
	out := &apiv1.EditorialSession{
		Id:             s.ID,
		VXID:           s.Vxid,
		Title:          s.Title,
		Status:         s.Status,
		CreatedBy:      s.CreatedBy,
		CreatedAt:      timestamppb.New(time.UnixMilli(s.CreatedAt)),
		UpdatedAt:      timestamppb.New(time.UnixMilli(s.UpdatedAt)),
		Kind:           s.Kind,
		PlayoutEventId: s.PlayoutEventID,
	}
	if s.Kind == editorial.KindLive {
		out.ClockStart = timestamppb.New(time.UnixMilli(s.ClockStartMs))
	}
	for _, m := range markers {
		out.Markers = append(out.Markers, editorialMarkerToProto(m))
	}
	return out
}

func editorialMarkerToProto(m editorial.Marker) *apiv1.EditorialMarker {
	return &apiv1.EditorialMarker{
		Id:            m.ID,
		SortOrder:     int32(m.SortOrder),
		Name:          m.Name,
		Contributors:  m.Contributors,
		Comment:       m.Comment,
		BibleVerses:   m.BibleVerses,
		Type:          m.Type,
		StartMs:       m.StartMs,
		EndMs:         m.EndMs,
		PublishBmm:    m.PublishBmm,
		PublishBcc:    m.PublishBcc,
		Source:        m.Source,
		PlayoutKey:    m.PlayoutKey,
		PlayoutSynced: m.PlayoutSynced,
	}
}
