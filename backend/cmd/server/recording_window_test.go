package main

import (
	"testing"
	"time"

	"github.com/bcc-code/bcc-media-flows/services/vidispine/vsapi"
)

func TestRecordingDateFromFileName(t *testing.T) {
	tests := []struct {
		name string
		file string
		want string // empty means "no date"
	}{
		// The live-ingest convention this relies on.
		{"live ingest", "SS26_20260808_1500_CLN_NOR.mxf", "2026-08-08"},
		{"other event code", "BS26_20261016_0930_CLN_NOR.mxf", "2026-10-16"},
		{"no extension", "NC26_20260101_1200_CLN", "2026-01-01"},
		// Material that does not follow it: better to report no date than to
		// invent one from a digit run that means something else.
		{"underscored date", "raw_2026_09_04_MAGA_S05_E04_INT.mov", ""},
		{"edited master", "Josef_Musikal_ny_klipp_MAS_NOR_ENG.wav", ""},
		{"bmm audio", "BMM-118282-nld.wav", ""},
		{"still image", "BS26_MOTE_2_LED_Hoved1.png", ""},
		{"impossible month", "XX26_20261308_1500_CLN.mxf", ""},
		{"digits run on", "XX26_202608081_1500.mxf", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			y, m, d, ok := recordingDateFromFileName(tt.file)
			if tt.want == "" {
				if ok {
					t.Fatalf("expected no date, got %04d-%02d-%02d", y, m, d)
				}
				return
			}
			if !ok {
				t.Fatalf("expected %s, got no date", tt.want)
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
		file      string
		tc        string
		duration  string
		wantStart string
		wantEnd   string
	}{
		{
			// Real values from VX-519332. 1327778@PAL = 53111.12s = 14:45:11
			// local; CEST is UTC+2, so 12:45:11Z.
			name:      "summer is UTC+2",
			file:      "SS26_20260808_1500_CLN_NOR.mxf",
			tc:        "1327778@PAL",
			duration:  "6770.24",
			wantStart: "2026-08-08T12:45:11Z",
			wantEnd:   "2026-08-08T14:38:01Z",
		},
		{
			// Same time of day in winter: CET is UTC+1, so 13:45:11Z.
			name:      "winter is UTC+1",
			file:      "BS26_20261216_1500_CLN_NOR.mxf",
			tc:        "1327778@PAL",
			duration:  "3600",
			wantStart: "2026-12-16T13:45:11Z",
			wantEnd:   "2026-12-16T14:45:11Z",
		},
		{
			// Same instant expressed at 50fps: the timebase must not change
			// the result. vscommon.TCToSeconds would reject this outright.
			name:      "50fps timebase",
			file:      "SS26_20260808_1500_CLN_NOR.mxf",
			tc:        "2655556@50",
			duration:  "6770.24",
			wantStart: "2026-08-08T12:45:11Z",
			wantEnd:   "2026-08-08T14:38:01Z",
		},
		{
			// The clocks go forward at 02:00 local on 2026-03-29. A recording
			// later that day is on the summer offset.
			name:      "day the clocks change",
			file:      "EC26_20260329_1500_CLN_NOR.mxf",
			tc:        "1327778@PAL",
			duration:  "3600",
			wantStart: "2026-03-29T12:45:11Z",
			wantEnd:   "2026-03-29T13:45:11Z",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := recordingWindowFromMetadata(metadata(map[string]string{
				"originalFilename": tt.file,
				"startTimeCode":    tt.tc,
				"durationSeconds":  tt.duration,
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

func TestRecordingWindowFromMetadataFallsBackToTitle(t *testing.T) {
	// Live ingests populate originalFilename, but title carries the same name.
	w, err := recordingWindowFromMetadata(metadata(map[string]string{
		"title":           "SS26_20260808_1500_CLN_NOR.mxf",
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
		{"no date in name", map[string]string{
			"originalFilename": "Josef_Musikal_ny_klipp_MAS.mov",
			"startTimeCode":    "90000@PAL",
			"durationSeconds":  "880.52",
		}},
		{"no start timecode", map[string]string{
			"originalFilename": "SS26_20260808_1500_CLN_NOR.mxf",
			"durationSeconds":  "6770.24",
		}},
		{"no duration", map[string]string{
			"originalFilename": "SS26_20260808_1500_CLN_NOR.mxf",
			"startTimeCode":    "1327778@PAL",
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
