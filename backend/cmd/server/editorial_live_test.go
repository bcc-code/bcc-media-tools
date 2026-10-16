package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"bcc-media-tools/playout"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func seqIDs() func() string {
	n := 0
	return func() string { n++; return fmt.Sprintf("new-%d", n) }
}

func speakerEntry(t *testing.T, name, at string) playout.ManifestEntry {
	data, _ := json.Marshal(map[string]string{"name": name})
	return playout.ManifestEntry{Type: playout.ContentTypeSpeaker, Label: name, Timestamp: ts(t, at), Data: data}
}

func songEntry(t *testing.T, label, at string) playout.ManifestEntry {
	return playout.ManifestEntry{Type: playout.ContentTypeSong, Label: label, Timestamp: ts(t, at)}
}

func scriptureEntry(t *testing.T, label, at string) playout.ManifestEntry {
	return playout.ManifestEntry{Type: playout.ContentTypeScripture, Label: label, Timestamp: ts(t, at)}
}

func manualMarker(id string, startMs int64, name, typ string) editorial.Marker {
	return editorial.Marker{ID: id, SessionID: "s1", StartMs: startMs, Name: name, Type: typ, Source: editorial.SourceManual}
}

// The meeting from the design: two blank marks, one named mark, and a Playout
// manifest with a speaker, scripture, two songs and a second speaker.
func TestMergeLiveWorkedExample(t *testing.T) {
	clockStart := ts(t, "2026-07-26T11:00:00Z")
	rows := []editorial.Marker{
		manualMarker("m1", 5_000, "", ""),                       // 11:00:05
		manualMarker("m2", 760_000, "", "sang"),                 // 11:12:40
		manualMarker("m3", 1_860_000, "Kunngjøringer", "annet"), // 11:31:00
	}
	candidates, candidatesVerses := fromManifest([]playout.ManifestEntry{
		speakerEntry(t, "Kåre J. Smith", "2026-07-26T11:00:41Z"),
		scriptureEntry(t, "John 3:16", "2026-07-26T11:03:10Z"),
		songEntry(t, "HV 16 - Se ditt hjem", "2026-07-26T11:13:02Z"),
		songEntry(t, "FMB 520", "2026-07-26T11:20:00Z"),
		speakerEntry(t, "Harald Hansen", "2026-07-26T11:31:30Z"),
	}, recordingWindow{Start: clockStart, End: ts(t, "2026-07-26T12:00:00Z")})

	got, changed := mergeLive(rows, candidates, candidatesVerses, nil, clockStart, seqIDs())
	require.True(t, changed)
	require.Len(t, got, 5)

	// Blank mark filled by the speaker: keeps its own time, gets the verse.
	assert.Equal(t, "m1", got[0].ID)
	assert.Equal(t, int64(5_000), got[0].StartMs)
	assert.Equal(t, "Kåre J. Smith", got[0].Contributors)
	assert.Equal(t, "John 3:16", got[0].BibleVerses)
	assert.Equal(t, "tale", got[0].Type)
	assert.True(t, got[0].PlayoutSynced)

	// Blank "sang" mark filled by the first song.
	assert.Equal(t, "m2", got[1].ID)
	assert.Equal(t, int64(760_000), got[1].StartMs)
	assert.Equal(t, "HV 16 - Se ditt hjem", got[1].Name)
	assert.Equal(t, "sang", got[1].Type)

	// The second song had no blank mark left: its own row at Playout's time.
	assert.Equal(t, int64(1_200_000), got[2].StartMs)
	assert.Equal(t, "FMB 520", got[2].Name)
	assert.Equal(t, editorial.SourceImport, got[2].Source)

	// The named mark is left alone and the speaker after it gets a row.
	assert.Equal(t, "m3", got[3].ID)
	assert.Equal(t, "Kunngjøringer", got[3].Name)
	assert.Empty(t, got[3].PlayoutKey)
	assert.Equal(t, "Harald Hansen", got[4].Contributors)
	assert.Equal(t, int64(1_890_000), got[4].StartMs)

	// Each row ends where the next starts; the last stays open.
	for i := 0; i < len(got)-1; i++ {
		assert.Equal(t, got[i+1].StartMs, got[i].EndMs, "row %d end", i)
	}
	assert.Zero(t, got[4].EndMs)

	// Syncing again with the same manifest changes nothing.
	again, changed := mergeLive(got, candidates, candidatesVerses, nil, clockStart, seqIDs())
	assert.False(t, changed)
	assert.Equal(t, got, again)
}

// A blank mark typed "sang" waits for a song; the speaker before it gets its
// own row.
func TestMergeLiveTypedMarkWaitsForMatch(t *testing.T) {
	clockStart := ts(t, "2026-07-26T11:00:00Z")
	rows := []editorial.Marker{manualMarker("m1", 0, "", "sang")}
	candidates, candidatesVerses := fromManifest([]playout.ManifestEntry{
		speakerEntry(t, "Programleder", "2026-07-26T11:00:10Z"),
		songEntry(t, "HV 16", "2026-07-26T11:00:40Z"),
	}, recordingWindow{Start: clockStart, End: ts(t, "2026-07-26T12:00:00Z")})

	got, _ := mergeLive(rows, candidates, candidatesVerses, nil, clockStart, seqIDs())
	require.Len(t, got, 2)
	assert.Equal(t, "m1", got[0].ID)
	assert.Equal(t, "HV 16", got[0].Name)
	assert.Empty(t, got[0].Contributors)
	assert.Equal(t, "Programleder", got[1].Contributors)
	assert.Equal(t, editorial.SourceImport, got[1].Source)
}

// Rows keep following Playout corrections until an editor changes them.
func TestMergeLiveRefreshesOnlySyncedRows(t *testing.T) {
	clockStart := ts(t, "2026-07-26T11:00:00Z")
	window := recordingWindow{Start: clockStart, End: ts(t, "2026-07-26T12:00:00Z")}
	first, firstVerses := fromManifest([]playout.ManifestEntry{
		speakerEntry(t, "Kare Smith", "2026-07-26T11:00:10Z"),
		songEntry(t, "HV 16", "2026-07-26T11:10:00Z"),
	}, window)
	rows, _ := mergeLive([]editorial.Marker{manualMarker("m1", 0, "", "")}, first, firstVerses, nil, clockStart, seqIDs())
	require.Len(t, rows, 2)

	// The editor fixes the song row by hand.
	rows[1].Name = "HV 16 - Se ditt hjem"
	rows[1].PlayoutSynced = false

	corrected, correctedVerses := fromManifest([]playout.ManifestEntry{
		speakerEntry(t, "Kåre J. Smith", "2026-07-26T11:00:10Z"),
		songEntry(t, "HV 16 (corrected)", "2026-07-26T11:10:00Z"),
	}, window)
	got, changed := mergeLive(rows, corrected, correctedVerses, nil, clockStart, seqIDs())
	assert.True(t, changed)
	assert.Equal(t, "Kåre J. Smith", got[0].Contributors, "synced row follows Playout")
	assert.Equal(t, "HV 16 - Se ditt hjem", got[1].Name, "edited row is left alone")
}

func TestMergeLiveSkipsDismissed(t *testing.T) {
	clockStart := ts(t, "2026-07-26T11:00:00Z")
	entry := songEntry(t, "HV 16", "2026-07-26T11:10:00Z")
	candidates, candidatesVerses := fromManifest([]playout.ManifestEntry{entry}, recordingWindow{Start: clockStart, End: ts(t, "2026-07-26T12:00:00Z")})

	got, changed := mergeLive(nil, candidates, candidatesVerses, map[string]bool{playoutKey(entry): true}, clockStart, seqIDs())
	assert.False(t, changed)
	assert.Empty(t, got)
}

func TestShiftToRecording(t *testing.T) {
	clockStart := ts(t, "2026-07-26T11:00:00Z")
	rows := []editorial.Marker{
		{StartMs: 5_000, EndMs: 60_000},
		{StartMs: 60_000, EndMs: 0},
	}
	// Recording started 100 s before the session was created and ran an hour.
	window := recordingWindow{Start: clockStart.Add(-100 * time.Second), End: clockStart.Add(3500 * time.Second)}
	shiftToRecording(rows, clockStart.UnixMilli(), window)

	assert.Equal(t, int64(105_000), rows[0].StartMs)
	assert.Equal(t, int64(160_000), rows[0].EndMs)
	assert.Equal(t, int64(160_000), rows[1].StartMs)
	assert.Equal(t, int64(3_600_000), rows[1].EndMs, "open row ends with the recording")
}

// End to end through the handlers: create a live session, mark, sync, then
// edit and delete rows and sync again.
func TestEditorialLiveSessionFlow(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()

	_, err := api.CreateEditorialSession(ctx, connect.NewRequest(&apiv1.CreateEditorialSessionRequest{Live: true, Title: "Søndag"}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "live needs a Playout event")

	created, err := api.CreateEditorialSession(ctx, connect.NewRequest(&apiv1.CreateEditorialSessionRequest{Live: true, Title: "Søndag", PlayoutEventId: "1149"}))
	require.NoError(t, err)
	sess := created.Msg
	assert.Equal(t, editorial.KindLive, sess.Kind)
	assert.Empty(t, sess.VXID)
	require.NotNil(t, sess.ClockStart)
	// Sync reads Playout up to now, so start the meeting an hour ago.
	clockStart := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
	_, err = api.db.ExecContext(ctx, "UPDATE sessions SET clock_start_ms = ? WHERE id = ?", clockStart.UnixMilli(), sess.Id)
	require.NoError(t, err)

	mark, err := api.AddEditorialMarker(ctx, connect.NewRequest(&apiv1.AddEditorialMarkerRequest{
		SessionId: sess.Id,
		At:        timestamppb.New(clockStart.Add(2 * time.Second)),
	}))
	require.NoError(t, err)
	assert.Equal(t, int64(2_000), mark.Msg.StartMs)

	at := func(d time.Duration) string { return clockStart.Add(d).UTC().Format(time.RFC3339Nano) }
	api.playout = fakePlayout{manifest: &playout.Manifest{Entries: []playout.ManifestEntry{
		speakerEntry(t, "Kåre J. Smith", at(10*time.Second)),
		songEntry(t, "HV 16", at(40*time.Second)),
	}}}

	synced, err := api.SyncEditorialSessionFromPlayout(ctx, connect.NewRequest(&apiv1.SyncEditorialSessionFromPlayoutRequest{Id: sess.Id}))
	require.NoError(t, err)
	markers := synced.Msg.Markers
	require.Len(t, markers, 2)
	assert.Equal(t, mark.Msg.Id, markers[0].Id)
	assert.Equal(t, "Kåre J. Smith", markers[0].Contributors)
	assert.Equal(t, "HV 16", markers[1].Name)

	// Edit the filled mark and remove the song row, then sync again: the edit
	// stays and the song doesn't come back.
	markers[0].Contributors = "Kåre Smith"
	saveTestSession(t, api, sess.Id, "Søndag", markers[:1])
	again, err := api.SyncEditorialSessionFromPlayout(ctx, connect.NewRequest(&apiv1.SyncEditorialSessionFromPlayoutRequest{Id: sess.Id}))
	require.NoError(t, err)
	require.Len(t, again.Msg.Markers, 1)
	assert.Equal(t, "Kåre Smith", again.Msg.Markers[0].Contributors)
	assert.False(t, again.Msg.Markers[0].PlayoutSynced)

	// A recording session can't be marked live.
	rec := createTestSession(t, api, "VX-1", "rec")
	_, err = api.AddEditorialMarker(ctx, connect.NewRequest(&apiv1.AddEditorialMarkerRequest{SessionId: rec.Id, At: timestamppb.Now()}))
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

// A session created after the meeting began still syncs from its real start.
func TestEditorialLiveSessionGivenStart(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()

	start := time.Now().Add(-30 * time.Minute).Truncate(time.Millisecond)
	created, err := api.CreateEditorialSession(ctx, connect.NewRequest(&apiv1.CreateEditorialSessionRequest{
		Live: true, Title: "Søndag", PlayoutEventId: "1149", ClockStart: timestamppb.New(start),
	}))
	require.NoError(t, err)
	assert.True(t, created.Msg.ClockStart.AsTime().Equal(start))

	api.playout = fakePlayout{manifest: &playout.Manifest{Entries: []playout.ManifestEntry{
		songEntry(t, "HV 16", start.Add(5*time.Minute).UTC().Format(time.RFC3339Nano)),
	}}}
	synced, err := api.SyncEditorialSessionFromPlayout(ctx, connect.NewRequest(&apiv1.SyncEditorialSessionFromPlayoutRequest{Id: created.Msg.Id}))
	require.NoError(t, err)
	require.Len(t, synced.Msg.Markers, 1)
	assert.Equal(t, int64(5*60_000), synced.Msg.Markers[0].StartMs)
}

// Editing a speaker's row stops Playout from changing its fields, but verses
// shown later in the speech still arrive; hand-typed verses stay.
func TestMergeLiveEditedRowStillCollectsVerses(t *testing.T) {
	clockStart := ts(t, "2026-07-26T11:00:00Z")
	window := recordingWindow{Start: clockStart, End: ts(t, "2026-07-26T12:00:00Z")}
	first, firstVerses := fromManifest([]playout.ManifestEntry{
		speakerEntry(t, "Kare Smith", "2026-07-26T11:00:10Z"),
		scriptureEntry(t, "John 3:16", "2026-07-26T11:01:00Z"),
	}, window)
	rows, _ := mergeLive([]editorial.Marker{manualMarker("m1", 0, "", "")}, first, firstVerses, nil, clockStart, seqIDs())
	require.Len(t, rows, 1)

	// The editor fixes the name and adds a verse of their own.
	rows[0].Contributors = "Kåre J. Smith"
	rows[0].BibleVerses = "John 3:16, Ps 23"
	rows[0].PlayoutSynced = false

	later, laterVerses := fromManifest([]playout.ManifestEntry{
		speakerEntry(t, "Kare Smith", "2026-07-26T11:00:10Z"),
		scriptureEntry(t, "John 3:16", "2026-07-26T11:01:00Z"),
		scriptureEntry(t, "Rom 8:28", "2026-07-26T11:05:00Z"),
	}, window)
	got, changed := mergeLive(rows, later, laterVerses, nil, clockStart, seqIDs())
	assert.True(t, changed)
	assert.Equal(t, "Kåre J. Smith", got[0].Contributors, "the edit stays")
	assert.Equal(t, "John 3:16, Ps 23, Rom 8:28", got[0].BibleVerses)
	assert.False(t, got[0].PlayoutSynced)

	again, changed := mergeLive(got, later, laterVerses, nil, clockStart, seqIDs())
	assert.False(t, changed, "nothing new on the next sync")
	assert.Equal(t, got, again)
}

func TestMergeVerses(t *testing.T) {
	assert.Equal(t, "John 3:16", mergeVerses("", "John 3:16"))
	assert.Equal(t, "A, B, C", mergeVerses("A, B", "B, C"))
	assert.Equal(t, "Rom 8:28", mergeVerses(" , Rom 8:28 ,", ""))
	assert.Equal(t, "john 3:16", mergeVerses("john 3:16", "John 3:16"), "case doesn't make a new verse")
}

// fromManifest splits a manifest the way a live sync does.
func fromManifest(entries []playout.ManifestEntry, window recordingWindow) ([]playoutCandidate, []scripture) {
	return playoutCandidates(entries, window), playoutScriptures(entries, window)
}

// Verses go to the speech marker they fall in, even one created and named by
// hand that Playout never filled; a verse during a song goes nowhere.
func TestMergeLiveVersesReachHandMadeSpeech(t *testing.T) {
	clockStart := ts(t, "2026-07-26T11:00:00Z")
	window := recordingWindow{Start: clockStart, End: ts(t, "2026-07-26T12:00:00Z")}
	speech := manualMarker("m1", 60_000, "", "tale")
	speech.Contributors = "Kåre J. Smith" // named by hand: not blank, never filled
	testimony := manualMarker("m2", 600_000, "", "vitnesbyrd")
	song := manualMarker("m3", 900_000, "HV 16", "sang")
	rows := []editorial.Marker{speech, testimony, song}

	cands, verses := fromManifest([]playout.ManifestEntry{
		scriptureEntry(t, "Ps 1:1", "2026-07-26T11:00:30Z"), // before any marker
		scriptureEntry(t, "John 3:16", "2026-07-26T11:02:00Z"),
		scriptureEntry(t, "Rom 8:28", "2026-07-26T11:05:00Z"),
		scriptureEntry(t, "Fil 4:6", "2026-07-26T11:11:00Z"),
		scriptureEntry(t, "Joh 1:1", "2026-07-26T11:16:00Z"), // during the song
	}, window)
	got, changed := mergeLive(rows, cands, verses, nil, clockStart, seqIDs())
	assert.True(t, changed)
	require.Len(t, got, 3)
	assert.Equal(t, "John 3:16, Rom 8:28", got[0].BibleVerses)
	assert.Equal(t, "Kåre J. Smith", got[0].Contributors)
	assert.Empty(t, got[0].PlayoutKey, "still a hand-made marker")
	assert.Equal(t, "Fil 4:6", got[1].BibleVerses)
	assert.Empty(t, got[2].BibleVerses, "not a speech")
}
