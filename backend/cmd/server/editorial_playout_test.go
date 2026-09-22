package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"bcc-media-tools/playout"
)

type fakePlayout struct {
	manifest *playout.Manifest
}

func (f fakePlayout) GetManifest(context.Context, string, ...string) (*playout.Manifest, error) {
	return f.manifest, nil
}

func (f fakePlayout) ListEvents(context.Context) ([]playout.Event, error) { return nil, nil }

func ts(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// A Playout event covers a whole conference: the manifest below mirrors the
// shape of a real one, with three meetings on two days. Only the middle
// meeting was recorded as the session's asset.
func TestImportFromPlayoutWindowsToTheRecording(t *testing.T) {
	speaker, _ := json.Marshal(map[string]string{"name": "Kåre J. Smith"})
	api := EditorialAPI{playout: fakePlayout{manifest: &playout.Manifest{
		Entries: []playout.ManifestEntry{
			// Earlier meeting, same day.
			{Type: playout.ContentTypeSpeaker, Label: "Someone Else", Timestamp: ts(t, "2026-07-26T11:28:00Z"), Data: speaker},
			// The recorded meeting.
			{Type: playout.ContentTypeSpeaker, Label: "Kåre J. Smith", Timestamp: ts(t, "2026-07-26T12:06:31Z"), Data: speaker},
			{Type: playout.ContentTypeScripture, Label: "John 3:16", Timestamp: ts(t, "2026-07-26T12:10:00Z")},
			{Type: playout.ContentTypeSong, Label: "HV 16 - Se ditt hjem", Timestamp: ts(t, "2026-07-26T12:31:00Z")},
			{Type: playout.ContentTypeSong, Label: "FMB 520 - Å, hvilket herlig gledesbud", Timestamp: ts(t, "2026-07-26T13:36:00Z")},
			// Later meeting, same day, and a meeting three days later.
			{Type: playout.ContentTypeSong, Label: "Not ours", Timestamp: ts(t, "2026-07-26T16:25:00Z")},
			{Type: playout.ContentTypeSpeaker, Label: "Nor ours", Timestamp: ts(t, "2026-07-29T13:26:00Z"), Data: speaker},
		},
	}}}

	window := recordingWindow{
		Start: ts(t, "2026-07-26T12:00:00Z"),
		End:   ts(t, "2026-07-26T13:45:00Z"),
	}
	markers, err := api.importFromPlayout(context.Background(), "1149", window)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(markers) != 3 {
		t.Fatalf("got %d markers, want 3 (the other meetings must be dropped)", len(markers))
	}

	// Offsets are measured from the recording start, not the event start.
	want := []struct {
		start int64
		end   int64
	}{
		{391000, 1860000},  // speaker at 12:06:31, ends when the song starts
		{1860000, 5760000}, // song at 12:31
		{5760000, 6300000}, // song at 13:36, ends with the recording
	}
	for i, w := range want {
		if markers[i].GetStartMs() != w.start {
			t.Errorf("marker %d start: got %d, want %d", i, markers[i].GetStartMs(), w.start)
		}
		if markers[i].GetEndMs() != w.end {
			t.Errorf("marker %d end: got %d, want %d", i, markers[i].GetEndMs(), w.end)
		}
	}

	if got := markers[0].GetBibleVerses(); got != "John 3:16" {
		t.Errorf("scripture should attach to the speaker in the window, got %q", got)
	}
	if got := markers[0].GetContributors(); got != "Kåre J. Smith" {
		t.Errorf("contributors: got %q", got)
	}
}

func TestImportFromPlayoutRequiresAWindow(t *testing.T) {
	api := EditorialAPI{playout: fakePlayout{manifest: &playout.Manifest{}}}
	if _, err := api.importFromPlayout(context.Background(), "1149", recordingWindow{}); err == nil {
		t.Fatal("expected an error without a recording window, got none")
	}
}

// A recording that overlaps none of the event's meetings yields nothing rather
// than everything.
func TestImportFromPlayoutOutsideEveryMeeting(t *testing.T) {
	api := EditorialAPI{playout: fakePlayout{manifest: &playout.Manifest{
		Entries: []playout.ManifestEntry{
			{Type: playout.ContentTypeSong, Label: "Elsewhere", Timestamp: ts(t, "2026-07-26T16:25:00Z")},
		},
	}}}
	markers, err := api.importFromPlayout(context.Background(), "1149", recordingWindow{
		Start: ts(t, "2026-07-26T12:00:00Z"),
		End:   ts(t, "2026-07-26T13:45:00Z"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(markers) != 0 {
		t.Fatalf("got %d markers, want 0", len(markers))
	}
}
