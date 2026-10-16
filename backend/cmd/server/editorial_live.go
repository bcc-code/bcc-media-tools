package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"bcc-media-tools/playout"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

// Live sessions are logged while a meeting is running, before its recording is
// in Mediabanken. Markers are stamped with the editor's clock and stored as
// offsets from the session's clock start; Playout fills in who spoke and what
// was sung; attaching the recording moves the offsets onto its timeline.

// liveSession loads a session and checks it is still live.
func (e EditorialAPI) liveSession(ctx context.Context, qtx *editorial.Queries, id string) (editorial.Session, error) {
	sess, err := qtx.GetSession(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return sess, errEditorialNotFound
	}
	if err != nil {
		return sess, fmt.Errorf("editorial: get session: %w", err)
	}
	if sess.Kind != editorial.KindLive {
		return sess, errNotLive
	}
	return sess, nil
}

var errNotLive = errors.New("editorial: session is not live")

// liveErr maps errNotLive to FailedPrecondition and everything else as
// editorialErr does.
func liveErr(err error) error {
	if errors.Is(err, errNotLive) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return editorialErr(err)
}

func (e EditorialAPI) AddEditorialMarker(ctx context.Context, req *connect.Request[apiv1.AddEditorialMarkerRequest]) (*connect.Response[apiv1.EditorialMarker], error) {
	if _, err := requireEditorial(req, apiv1.EditorialLive); err != nil {
		return nil, err
	}
	if req.Msg.GetSessionId() == "" || req.Msg.GetAt() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing session_id or at"))
	}
	now := time.Now().UnixMilli()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: begin tx: %w", err))
	}
	defer func() { _ = tx.Rollback() }()
	qtx := e.queries.WithTx(tx)

	sess, err := e.liveSession(ctx, qtx, req.Msg.GetSessionId())
	if err != nil {
		return nil, liveErr(err)
	}
	rows, err := qtx.ListMarkersForSession(ctx, sess.ID)
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: list markers: %w", err))
	}
	// The editor's clock, not the server's: that is the moment they saw.
	marker := editorial.Marker{
		ID:        uuid.NewString(),
		SessionID: sess.ID,
		Type:      req.Msg.GetType(),
		StartMs:   req.Msg.GetAt().AsTime().UnixMilli() - sess.ClockStartMs,
		Source:    editorial.SourceManual,
	}
	rows = append(rows, marker)
	normalizeLiveMarkers(rows)
	if err := replaceMarkers(ctx, qtx, sess.ID, rows, now); err != nil {
		return nil, editorialErr(err)
	}
	if err := qtx.TouchSession(ctx, editorial.TouchSessionParams{UpdatedAt: now, ID: sess.ID}); err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: touch session: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: commit: %w", err))
	}
	e.hub.changed(sess.ID)
	for _, m := range rows {
		if m.ID == marker.ID {
			return connect.NewResponse(editorialMarkerToProto(m)), nil
		}
	}
	return nil, editorialErr(errEditorialNotFound)
}

// SyncEditorialSessionFromPlayout re-reads the session's Playout manifest from
// its clock start until now and merges it into the saved markers (mergeLive).
// Safe to repeat: entries already merged are recognised by their key.
func (e EditorialAPI) SyncEditorialSessionFromPlayout(ctx context.Context, req *connect.Request[apiv1.SyncEditorialSessionFromPlayoutRequest]) (*connect.Response[apiv1.EditorialSession], error) {
	if _, err := requireEditorial(req, apiv1.EditorialLive); err != nil {
		return nil, err
	}
	if e.playout == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("playout client is not configured"))
	}
	sess, err := e.liveSession(ctx, e.queries, req.Msg.GetId())
	if err != nil {
		return nil, liveErr(err)
	}
	// One tab syncs for everyone; the others get its result as a change.
	if !e.hub.maySync(sess.ID, req.Msg.GetClientId()) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("another tab runs the Playout sync for this session"))
	}
	// Fetched outside the transaction: it is the slow part.
	manifest, err := e.playout.GetManifest(ctx, sess.PlayoutEventID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get Playout manifest: %w", err))
	}
	clockStart := time.UnixMilli(sess.ClockStartMs).UTC()
	window := recordingWindow{Start: clockStart, End: time.Now().UTC()}
	candidates := playoutCandidates(manifest.Entries, window)
	scriptures := playoutScriptures(manifest.Entries, window)

	now := time.Now().UnixMilli()
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: begin tx: %w", err))
	}
	defer func() { _ = tx.Rollback() }()
	qtx := e.queries.WithTx(tx)

	rows, err := qtx.ListMarkersForSession(ctx, sess.ID)
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: list markers: %w", err))
	}
	dismissedKeys, err := qtx.ListDismissedPlayoutKeys(ctx, sess.ID)
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: list dismissed: %w", err))
	}
	dismissed := make(map[string]bool, len(dismissedKeys))
	for _, k := range dismissedKeys {
		dismissed[k] = true
	}

	merged, changed := mergeLive(rows, candidates, scriptures, dismissed, clockStart, uuid.NewString)
	if changed {
		if err := replaceMarkers(ctx, qtx, sess.ID, merged, now); err != nil {
			return nil, editorialErr(err)
		}
		if err := qtx.TouchSession(ctx, editorial.TouchSessionParams{UpdatedAt: now, ID: sess.ID}); err != nil {
			return nil, editorialErr(fmt.Errorf("editorial: touch session: %w", err))
		}
		if err := tx.Commit(); err != nil {
			return nil, editorialErr(fmt.Errorf("editorial: commit: %w", err))
		}
		e.hub.changed(sess.ID)
	}

	out, err := e.loadSession(ctx, sess.ID)
	if err != nil {
		return nil, editorialErr(err)
	}
	return connect.NewResponse(out), nil
}

// AttachEditorialRecording ends live logging: the ingested recording becomes
// the session's asset and marker offsets move from the session's clock start
// onto the recording's own timeline. Writing chapters to Mediabanken (via
// Cantemo) is a separate step.
func (e EditorialAPI) AttachEditorialRecording(ctx context.Context, req *connect.Request[apiv1.AttachEditorialRecordingRequest]) (*connect.Response[apiv1.EditorialSession], error) {
	if _, err := requireEditorial(req, apiv1.EditorialLive); err != nil {
		return nil, err
	}
	vxID := req.Msg.GetVXID()
	if req.Msg.GetId() == "" || vxID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing id or VXID"))
	}
	sess, err := e.liveSession(ctx, e.queries, req.Msg.GetId())
	if err != nil {
		return nil, liveErr(err)
	}

	var window recordingWindow
	manual := req.Msg.GetRecordingStart() != nil
	if manual {
		window, _, err = e.windowFromManualStart(vxID, req.Msg.GetRecordingStart().AsTime(), req.Msg.GetRecordingEnd())
	} else {
		window, err = recordingWindowForItem(e.vidispine, vxID)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("cannot place %s on the wall clock: %w", vxID, err))
	}

	now := time.Now().UnixMilli()
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: begin tx: %w", err))
	}
	defer func() { _ = tx.Rollback() }()
	qtx := e.queries.WithTx(tx)

	rows, err := qtx.ListMarkersForSession(ctx, sess.ID)
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: list markers: %w", err))
	}
	shiftToRecording(rows, sess.ClockStartMs, window)
	if err := replaceMarkers(ctx, qtx, sess.ID, rows, now); err != nil {
		return nil, editorialErr(err)
	}
	n, err := qtx.AttachRecording(ctx, editorial.AttachRecordingParams{
		Vxid:      vxID,
		Kind:      editorial.KindRecording,
		UpdatedAt: now,
		ID:        sess.ID,
	})
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: attach recording: %w", err))
	}
	if n == 0 {
		return nil, editorialErr(errEditorialNotFound)
	}
	// Like the Playout import, only a hand-entered window is worth keeping.
	if manual {
		if _, err := qtx.SetPlayoutWindow(ctx, editorial.SetPlayoutWindowParams{
			PlayoutEventID:   sess.PlayoutEventID,
			RecordingStartMs: window.Start.UnixMilli(),
			RecordingEndMs:   window.End.UnixMilli(),
			UpdatedAt:        now,
			ID:               sess.ID,
		}); err != nil {
			return nil, editorialErr(fmt.Errorf("editorial: set playout window: %w", err))
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: commit: %w", err))
	}

	e.hub.changed(sess.ID)
	out, err := e.loadSession(ctx, sess.ID)
	if err != nil {
		return nil, editorialErr(err)
	}
	if vxID != "" {
		out.PreviewUrl = editorialPreviewPath(out.Id, out.VXID)
	}
	return connect.NewResponse(out), nil
}

// shiftToRecording re-bases offsets from the live clock start onto the
// recording, whose start may be before or after it. The still-open last row
// ends with the recording.
func shiftToRecording(rows []editorial.Marker, clockStartMs int64, window recordingWindow) {
	shift := clockStartMs - window.Start.UnixMilli()
	for i := range rows {
		rows[i].StartMs += shift
		if rows[i].EndMs != 0 {
			rows[i].EndMs += shift
		}
	}
	if n := len(rows); n > 0 && rows[n-1].EndMs == 0 {
		rows[n-1].EndMs = window.End.Sub(window.Start).Milliseconds()
	}
}

// normalizeLiveMarkers keeps a live session in time order: each row ends where
// the next starts, and the last stays open (0) while the meeting runs.
func normalizeLiveMarkers(rows []editorial.Marker) {
	slices.SortStableFunc(rows, func(a, b editorial.Marker) int {
		switch {
		case a.StartMs < b.StartMs:
			return -1
		case a.StartMs > b.StartMs:
			return 1
		}
		return 0
	})
	for i := range rows {
		rows[i].SortOrder = int64(i)
		if i+1 < len(rows) {
			rows[i].EndMs = rows[i+1].StartMs
		} else {
			rows[i].EndMs = 0
		}
	}
}

// isBlank reports whether an editor marked a moment without saying what it
// was: no title, no contributors, and not yet filled from Playout.
func isBlank(m editorial.Marker) bool {
	return m.Source == editorial.SourceManual && m.PlayoutKey == "" && m.Name == "" && m.Contributors == ""
}

// fillableBy reports whether a blank marker of the editor's type may take a
// Playout entry. Only lower-thirds (speakers) and songs fill markers; a type
// the editor set first must match.
func fillableBy(markerType, playoutType string) bool {
	switch playoutType {
	case playout.ContentTypeSong:
		return markerType == "" || markerType == "sang" || markerType == "allsang"
	case playout.ContentTypeSpeaker:
		return markerType != "sang" && markerType != "allsang"
	}
	return false
}

// applyCandidate copies the fields Playout fills onto a row. keepType leaves an
// editor-chosen type alone.
//
// Bible verses aren't copied: in a live session they go to whichever speech
// marker they fall in (attachScriptures).
func applyCandidate(m *editorial.Marker, c playoutCandidate, keepType bool) {
	switch c.PlayoutType {
	case playout.ContentTypeSpeaker:
		m.Contributors = c.Contributors
	default:
		m.Name = c.Name
	}
	if !keepType || m.Type == "" {
		m.Type = editorialTypeFromPlayout(c.PlayoutType)
	}
}

// mergeVerses adds the verses in incoming that current doesn't list yet,
// after the ones it has: verses typed in by hand stay, in their order.
// Both are ", "-separated lists.
func mergeVerses(current, incoming string) string {
	seen := map[string]bool{}
	var out []string
	for _, v := range strings.Split(current, ",") {
		if v = strings.TrimSpace(v); v != "" && !seen[strings.ToLower(v)] {
			seen[strings.ToLower(v)] = true
			out = append(out, v)
		}
	}
	for _, v := range strings.Split(incoming, ",") {
		if v = strings.TrimSpace(v); v != "" && !seen[strings.ToLower(v)] {
			seen[strings.ToLower(v)] = true
			out = append(out, v)
		}
	}
	return strings.Join(out, ", ")
}

// mergeLive folds Playout candidates (in time order) into a live session's
// markers:
//
//   - a speaker or song fills the latest editor marker before it if that
//     marker is still blank (and its type, if set, matches); the marker keeps
//     the editor's timestamp
//   - every other candidate becomes its own row at Playout's timestamp
//   - rows already merged are refreshed while still synced, i.e. until an
//     editor changes a field Playout fills; after that they still collect
//     new Bible verses
//   - dismissed candidates (rows an editor removed) are skipped
//
// It returns the rows in time order and whether anything changed.
func mergeLive(rows []editorial.Marker, candidates []playoutCandidate, scriptures []scripture, dismissed map[string]bool, clockStart time.Time, newID func() string) ([]editorial.Marker, bool) {
	rows = slices.Clone(rows)
	normalizeLiveMarkers(rows)
	before := slices.Clone(rows)

	byKey := map[string]int{}
	for i, m := range rows {
		if m.PlayoutKey != "" {
			byKey[m.PlayoutKey] = i
		}
	}

	var added []editorial.Marker
	for _, c := range candidates {
		if i, ok := byKey[c.Key]; ok {
			// i < 0: added earlier in this pass (a duplicate manifest entry).
			if i >= 0 {
				// An edited row keeps its edits; its verses still arrive
				// through attachScriptures.
				if rows[i].PlayoutSynced {
					applyCandidate(&rows[i], c, rows[i].Source == editorial.SourceManual)
				}
			}
			continue
		}
		if dismissed[c.Key] {
			continue
		}
		offset := c.At.Sub(clockStart).Milliseconds()

		// The editor's latest own marker at or before this entry.
		target := -1
		for i, m := range rows {
			if m.StartMs > offset {
				break
			}
			if m.Source == editorial.SourceManual {
				target = i
			}
		}
		if target >= 0 && isBlank(rows[target]) && fillableBy(rows[target].Type, c.PlayoutType) {
			m := &rows[target]
			applyCandidate(m, c, true)
			m.PlayoutKey = c.Key
			m.PlayoutSynced = true
			byKey[c.Key] = target
			continue
		}

		m := editorial.Marker{
			ID:            newID(),
			SessionID:     sessionIDOf(rows),
			StartMs:       offset,
			Source:        editorial.SourceImport,
			PlayoutKey:    c.Key,
			PlayoutSynced: true,
		}
		applyCandidate(&m, c, false)
		added = append(added, m)
		// Indexed after the loop: added rows never become fill targets, so
		// their positions don't matter until then.
		byKey[c.Key] = -1
	}

	rows = append(rows, added...)
	normalizeLiveMarkers(rows)
	attachScriptures(rows, scriptures, clockStart)
	return rows, len(added) > 0 || !slices.Equal(rows, before)
}

// scripture is a Bible verse shown on screen.
type scripture struct {
	At    time.Time
	Label string
}

// playoutScriptures lists the verses shown inside window, in manifest order.
func playoutScriptures(entries []playout.ManifestEntry, window recordingWindow) []scripture {
	var out []scripture
	for _, e := range entries {
		if e.Type == playout.ContentTypeScripture && e.Label != "" && window.Contains(e.Timestamp) {
			out = append(out, scripture{At: e.Timestamp, Label: e.Label})
		}
	}
	return out
}

// isSpeech reports whether a marker type is someone speaking.
func isSpeech(markerType string) bool {
	return markerType == "tale" || markerType == "vitnesbyrd"
}

// attachScriptures adds each verse to the marker it falls in (rows in time
// order) when that marker is a speech, whoever created it: a marker typed in
// by hand collects verses as well as one Playout filled. Verses already
// listed aren't added again, and none are removed.
func attachScriptures(rows []editorial.Marker, scriptures []scripture, clockStart time.Time) {
	for _, sc := range scriptures {
		offset := sc.At.Sub(clockStart).Milliseconds()
		i, _ := slices.BinarySearchFunc(rows, offset, func(m editorial.Marker, t int64) int {
			if m.StartMs <= t {
				return -1
			}
			return 1
		})
		// i is the first row starting after the verse; the one before
		// holds it.
		if i == 0 || !isSpeech(rows[i-1].Type) {
			continue
		}
		rows[i-1].BibleVerses = mergeVerses(rows[i-1].BibleVerses, sc.Label)
	}
}

func sessionIDOf(rows []editorial.Marker) string {
	if len(rows) > 0 {
		return rows[0].SessionID
	}
	return ""
}
