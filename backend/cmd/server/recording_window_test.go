package main

import (
	"testing"
	"time"

	"github.com/bcc-code/bcc-media-flows/services/vidispine/vsapi"
)

func TestRecordingDateFromMetadata(t *testing.T) {
	oslo, err := time.LoadLocation(recordingTZ)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		fields map[string]string
		want   string // empty means "no date"
	}{
		{
			// VX-519286: the live ingest workflow stamps portal_ingested as
			// recording starts, so it is the better of the two.
			name: "prefers the live-ingest stamp",
			fields: map[string]string{
				"portal_ingested": "2026-08-08T12:46:05.676Z",
				"created":         "2026-08-10T09:00:00.000Z",
			},
			want: "2026-08-08",
		},
		{
			// VX-519332: only the feeds ingested live carry portal_ingested.
			name:   "falls back to created",
			fields: map[string]string{"created": "2026-08-08T15:56:06.342Z"},
			want:   "2026-08-08",
		},
		{
			// 00:30 Oslo on the 9th is 22:30Z on the 8th. The window is built
			// on Oslo wall-clock time, so the date has to be read there too.
			name:   "reads the date in Oslo, not UTC",
			fields: map[string]string{"created": "2026-08-08T22:30:00.000Z"},
			want:   "2026-08-09",
		},
		{
			name:   "nothing to read",
			fields: map[string]string{"startTimeCode": "1327778@PAL"},
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			y, m, d, err := recordingDateFromMetadata(metadata(tt.fields), oslo)
			if tt.want == "" {
				if err == nil {
					t.Fatalf("expected an error, got %04d-%02d-%02d", y, m, d)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
			if got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTimecodeToSeconds(t *testing.T) {
	tests := []struct {
		name string
		tc   string
		want float64
	}{
		// Every timebase observed in Mediabanken.
		{"PAL is 25fps", "1327778@PAL", 53111.12}, // VX-519332, 14:45:11
		{"50fps", "1800000@50", 36000},            // 10:00:00
		{"audio sample rate", "172800000@48000", 3600},
		{"zero", "0@PAL", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := timecodeToSeconds(tt.tc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Float division, so compare within a millisecond.
			if diff := got - tt.want; diff > 0.001 || diff < -0.001 {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}

	for _, tc := range []string{"", "1327778", "abc@PAL", "1000@NTSC", "1000@0", "1000@-25"} {
		t.Run("rejects "+tc, func(t *testing.T) {
			if _, err := timecodeToSeconds(tc); err == nil {
				t.Fatalf("expected an error for %q", tc)
			}
		})
	}
}

func metadata(fields map[string]string) *vsapi.MetadataResult {
	terse := map[string][]*vsapi.MetadataField{}
	for k, v := range fields {
		terse[k] = []*vsapi.MetadataField{{Value: v}}
	}
	return &vsapi.MetadataResult{Terse: terse}
}

func TestRecordingWindowFromMetadata(t *testing.T) {
	// The start timecode is a time of day in Europe/Oslo while the manifest is
	// in UTC, so the offset differs between summer and winter. Getting this
	// wrong shifts every marker by a whole hour or two, which looks like a
	// matching failure rather than a timezone bug — hence both cases.
	tests := []struct {
		name      string
		ingested  string
		tc        string
		duration  string
		wantStart string
		wantEnd   string
	}{
		{
			// Real values from VX-519332. 1327778@PAL = 53111.12s = 14:45:11
			// local; CEST is UTC+2, so 12:45:11Z.
			name:      "summer is UTC+2",
			ingested:  "2026-08-08T12:46:05.676Z",
			tc:        "1327778@PAL",
			duration:  "6770.24",
			wantStart: "2026-08-08T12:45:11Z",
			wantEnd:   "2026-08-08T14:38:01Z",
		},
		{
			// Same time of day in winter: CET is UTC+1, so 13:45:11Z.
			name:      "winter is UTC+1",
			ingested:  "2026-12-16T14:00:00.000Z",
			tc:        "1327778@PAL",
			duration:  "3600",
			wantStart: "2026-12-16T13:45:11Z",
			wantEnd:   "2026-12-16T14:45:11Z",
		},
		{
			// Same instant expressed at 50fps: the timebase must not change
			// the result. vscommon.TCToSeconds would reject this outright.
			name:      "50fps timebase",
			ingested:  "2026-08-08T12:46:05.676Z",
			tc:        "2655556@50",
			duration:  "6770.24",
			wantStart: "2026-08-08T12:45:11Z",
			wantEnd:   "2026-08-08T14:38:01Z",
		},
		{
			// The clocks go forward at 02:00 local on 2026-03-29. A recording
			// later that day is on the summer offset.
			name:      "day the clocks change",
			ingested:  "2026-03-29T13:00:00.000Z",
			tc:        "1327778@PAL",
			duration:  "3600",
			wantStart: "2026-03-29T12:45:11Z",
			wantEnd:   "2026-03-29T13:45:11Z",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := recordingWindowFromMetadata(metadata(map[string]string{
				"portal_ingested": tt.ingested,
				"startTimeCode":   tt.tc,
				"durationSeconds": tt.duration,
			}))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := w.Start.Format(time.RFC3339); got != tt.wantStart {
				t.Errorf("start: got %s, want %s", got, tt.wantStart)
			}
			if got := w.End.Format(time.RFC3339); got != tt.wantEnd {
				t.Errorf("end: got %s, want %s", got, tt.wantEnd)
			}
		})
	}
}

func TestRecordingWindowFromMetadataFallsBackToCreated(t *testing.T) {
	// Only the live-ingested feed carries portal_ingested; everything else is
	// placed by the time its file was copied in.
	w, err := recordingWindowFromMetadata(metadata(map[string]string{
		"created":         "2026-08-08T15:56:06.342Z",
		"startTimeCode":   "1327778@PAL",
		"durationSeconds": "6770.24",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := w.Start.Format(time.RFC3339); got != "2026-08-08T12:45:11Z" {
		t.Fatalf("got %s", got)
	}
}

func TestRecordingWindowFromMetadataErrors(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]string
	}{
		{"no ingest timestamp", map[string]string{
			"startTimeCode":   "1327778@PAL",
			"durationSeconds": "6770.24",
		}},
		{"no start timecode", map[string]string{
			"created":         "2026-08-08T15:56:06.342Z",
			"durationSeconds": "6770.24",
		}},
		{"no duration", map[string]string{
			"created":       "2026-08-08T15:56:06.342Z",
			"startTimeCode": "1327778@PAL",
		}},
		// Without these guards every edited master would derive a window from
		// its default timecode on the day it happened to be ingested. VX-519490
		// (Josef_Musikal_ny_klipp_MAS.mov) is the 01:00:00 case.
		{"default timecode 01:00:00", map[string]string{
			"created":         "2026-09-10T12:26:19.151Z",
			"startTimeCode":   "90000@PAL",
			"durationSeconds": "880.52",
		}},
		{"default timecode 01:00:00 at 48kHz", map[string]string{
			"created":         "2026-09-25T08:04:53.416Z",
			"startTimeCode":   "172800000@48000",
			"durationSeconds": "880.52",
		}},
		{"default timecode 10:00:00", map[string]string{
			"created":         "2026-09-26T13:55:45.179Z",
			"startTimeCode":   "1800000@50",
			"durationSeconds": "120",
		}},
		{"zero timecode", map[string]string{
			"created":         "2026-08-18T11:25:12.859Z",
			"startTimeCode":   "0@PAL",
			"durationSeconds": "576.2",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := recordingWindowFromMetadata(metadata(tt.fields)); err == nil {
				t.Fatal("expected an error, got none")
			}
		})
	}
}

func TestRecordingWindowContains(t *testing.T) {
	at := func(s string) time.Time {
		t.Helper()
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	w := recordingWindow{Start: at("2026-07-26T12:00:00Z"), End: at("2026-07-26T13:45:00Z")}

	for _, tt := range []struct {
		ts   string
		want bool
	}{
		{"2026-07-26T11:28:00Z", false}, // the meeting before
		{"2026-07-26T12:00:00Z", true},  // start is inclusive
		{"2026-07-26T12:06:31Z", true},
		{"2026-07-26T13:45:00Z", true}, // end is inclusive
		{"2026-07-26T16:25:00Z", false},
		{"2026-07-29T13:26:00Z", false}, // a different day of the same event
	} {
		if got := w.Contains(at(tt.ts)); got != tt.want {
			t.Errorf("Contains(%s) = %v, want %v", tt.ts, got, tt.want)
		}
	}
}
