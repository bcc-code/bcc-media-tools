package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	// Embeds tz data: the production image (distroless/static) ships no
	// /usr/share/zoneinfo, so time.LoadLocation would fail without this.
	_ "time/tzdata"

	"github.com/bcc-code/bcc-media-flows/services/vidispine"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vsapi"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vscommon"
)

// recordingTZ is the zone Vidispine's start timecode (a time of day, not an
// instant) is expressed in. Playout timestamps are UTC, so this conversion is
// needed — a fixed offset would be wrong for half the year.
const recordingTZ = "Europe/Oslo"

// File names use fieldOriginalFilename (vault.go), not
// vscommon.FieldOriginalFileName — that Portal field is empty on live ingests.

// recordingWindow is the wall-clock span a Vidispine item covers, in UTC.
type recordingWindow struct {
	Start time.Time
	End   time.Time
}

func (w recordingWindow) IsZero() bool { return w.Start.IsZero() }

// Contains reports whether t is in the window, bounds included.
func (w recordingWindow) Contains(t time.Time) bool {
	return !t.Before(w.Start) && !t.After(w.End)
}

// recordingDateRE matches the live-ingest naming convention, e.g.
// "SS26_20260808_1500_CLN_NOR.mxf"; separators on both sides keep it from
// matching digits inside some other identifier.
var recordingDateRE = regexp.MustCompile(`(?:^|[_-])(\d{4})(\d{2})(\d{2})(?:[_.-]|$)`)

// recordingDateFromFileName pulls the date out of the file name: Vidispine
// only stores a time-of-day start, and the ingest timestamp lags past
// midnight, so the name is what's left. Non-conforming material (edited
// masters, BMM audio) has no date to find; callers must report that, not guess.
func recordingDateFromFileName(name string) (year int, month time.Month, day int, ok bool) {
	m := recordingDateRE.FindStringSubmatch(name)
	if m == nil {
		return 0, 0, 0, false
	}
	y, _ := strconv.Atoi(m[1])
	mo, _ := strconv.Atoi(m[2])
	d, _ := strconv.Atoi(m[3])
	if mo < 1 || mo > 12 || d < 1 || d > 31 {
		return 0, 0, 0, false
	}
	return y, time.Month(mo), d, true
}

// recordingWindowFromMetadata places a recording on the wall clock: date from
// the file name, start from the start timecode, length from the duration.
func recordingWindowFromMetadata(meta *vsapi.MetadataResult) (recordingWindow, error) {
	name := meta.Get(fieldOriginalFilename, "")
	if name == "" {
		name = meta.Get(vscommon.FieldTitle, "")
	}
	year, month, day, ok := recordingDateFromFileName(name)
	if !ok {
		return recordingWindow{}, fmt.Errorf("no recording date in file name %q", name)
	}

	tc := meta.Get(vscommon.FieldStartTC, "")
	if tc == "" {
		return recordingWindow{}, fmt.Errorf("item has no start timecode")
	}
	startSeconds, err := vscommon.TCToSeconds(tc)
	if err != nil {
		return recordingWindow{}, fmt.Errorf("parse start timecode %q: %w", tc, err)
	}

	rawDuration := meta.Get(vscommon.FieldDurationSeconds, "")
	durationSeconds, err := strconv.ParseFloat(rawDuration, 64)
	if err != nil {
		return recordingWindow{}, fmt.Errorf("parse duration %q: %w", rawDuration, err)
	}

	loc, err := time.LoadLocation(recordingTZ)
	if err != nil {
		return recordingWindow{}, fmt.Errorf("load %s: %w", recordingTZ, err)
	}

	// time.Date normalizes the offset as wall-clock time before resolving the
	// zone, keeping this correct on the two DST-transition days.
	sec := int(startSeconds)
	nsec := int((startSeconds - float64(sec)) * float64(time.Second))
	start := time.Date(year, month, day, 0, 0, sec, nsec, loc)

	return recordingWindow{
		Start: start.UTC(),
		End:   start.Add(time.Duration(durationSeconds * float64(time.Second))).UTC(),
	}, nil
}

// recordingWindowForItem fetches just the metadata needed to place an item on
// the wall clock.
//
// Fields are joined into one comma-separated element on purpose:
// GetMetadataFields turns each slice entry into its own &field= param, and
// Vidispine honours only the first — fetching full metadata instead would
// pull well over a megabyte per item.
func recordingWindowForItem(vs vidispine.Client, vxID string) (recordingWindow, error) {
	meta, err := vs.GetMetadataFields(vxID, []string{strings.Join([]string{
		fieldOriginalFilename.Value,
		vscommon.FieldTitle.Value,
		vscommon.FieldStartTC.Value,
		vscommon.FieldDurationSeconds.Value,
	}, ",")})
	if err != nil {
		return recordingWindow{}, fmt.Errorf("get metadata for %s: %w", vxID, err)
	}
	return recordingWindowFromMetadata(meta)
}
