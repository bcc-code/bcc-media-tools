//go:build integration

package main

// Writes chapters to a real Vidispine item and reads them back, to check the
// metadata documents Vidispine accepts. Run against a test system only:
//
//	VIDISPINE_TEST_URL=http://c6.lan.bcc.media:8080/API VIDISPINE_TEST_VXID=VX-519687 \
//	VIDISPINE_USERNAME=… VIDISPINE_PASSWORD=… \
//	go test -tags integration ./cmd/server/ -run TestChaptersAgainstVidispine -v
//
// It leaves the item as it found it: the last step removes the chapters.
//
// To look at the chapters in Cantemo, add VIDISPINE_TEST_KEEP=1: the test
// stops after writing them and logs a VIDISPINE_TEST_REMOVE value. Run
// TestRemoveTestChapters with that value to take them off again.

import (
	"context"
	"encoding/xml"
	"os"
	"strings"
	"testing"

	"bcc-media-tools/editorial"

	"github.com/bcc-code/bcc-media-flows/services/vidispine/vscommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// subclipFields reads the fields of one Subclips group instance by uuid.
func subclipFields(t *testing.T, body []byte, uuid string) map[string][]string {
	t.Helper()
	var doc struct {
		Items []struct {
			Metadata []struct {
				Timespans []mdTimespan `xml:"timespan"`
			} `xml:"metadata"`
		} `xml:"item"`
	}
	require.NoError(t, xml.Unmarshal(body, &doc))
	for _, item := range doc.Items {
		for _, md := range item.Metadata {
			for _, ts := range md.Timespans {
				for _, g := range ts.Groups {
					if g.UUID == uuid {
						out := map[string][]string{}
						for _, f := range g.Fields {
							out[f.Name] = f.Values
						}
						return out
					}
				}
			}
		}
	}
	t.Fatalf("subclip %s not found", uuid)
	return nil
}

func TestChaptersAgainstVidispine(t *testing.T) {
	url, vxID := os.Getenv("VIDISPINE_TEST_URL"), os.Getenv("VIDISPINE_TEST_VXID")
	if url == "" || vxID == "" {
		t.Skip("set VIDISPINE_TEST_URL and VIDISPINE_TEST_VXID")
	}
	ctx := context.Background()
	vs := newVidispineMetadata(url, os.Getenv("VIDISPINE_USERNAME"), os.Getenv("VIDISPINE_PASSWORD"))

	api := newTestEditorialAPI(t)
	api.vsMetadata = vs
	created := createTestSession(t, api, vxID, "integration test")
	sess, err := api.queries.GetSession(ctx, created.Id)
	require.NoError(t, err)

	// The item's own start timecode, as the handler reads it.
	body, err := vs.GetItemMetadata(ctx, vxID)
	require.NoError(t, err)
	// startTimeCode is an item-wide field, directly in the -INF/+INF timespan.
	var item struct {
		Items []struct {
			Metadata []struct {
				Timespans []struct {
					Fields []mdField `xml:"field"`
				} `xml:"timespan"`
			} `xml:"metadata"`
		} `xml:"item"`
	}
	require.NoError(t, xml.Unmarshal(body, &item))
	startRaw := ""
	for _, md := range item.Items[0].Metadata {
		for _, ts := range md.Timespans {
			for _, f := range ts.Fields {
				if f.Name == vscommon.FieldStartTC.Value && len(f.Values) > 0 {
					startRaw = f.Values[0]
				}
			}
		}
	}
	if startRaw == "" {
		startRaw = "0@PAL"
	}
	start, err := parseTimecode(startRaw)
	require.NoError(t, err)
	t.Logf("start timecode %s", startRaw)

	before, err := parseSubclips(body)
	require.NoError(t, err)

	// 1. Write three chapters.
	first := []editorial.Marker{
		{Name: "", Contributors: "Kåre J. Smith, Harald Hansen", Type: "tale", BibleVerses: "John 3:16", StartMs: 10_000, EndMs: 70_000, PublishBmm: true},
		{Name: "HV 16 – Se ditt hjem", Type: "allsang", StartMs: 70_000, EndMs: 130_000},
		{Name: "Kunngjøringer", Type: "annet", StartMs: 130_000, EndMs: 140_000},
	}
	res, err := api.exportChapters(ctx, sess, first, start)
	require.NoError(t, err)
	assert.EqualValues(t, 3, res.Written)

	body, err = vs.GetItemMetadata(ctx, vxID)
	require.NoError(t, err)
	after, err := parseSubclips(body)
	require.NoError(t, err)
	ours := newSubclips(before, after)
	require.Len(t, ours, 3, "three new Subclips group instances")

	byStart := map[string]subclip{}
	for _, s := range ours {
		byStart[s.Start] = s
	}
	speech, ok := byStart[start.plus(10_000)]
	require.True(t, ok, "speech chapter at start timecode + 10 s (%s)", start.plus(10_000))
	assert.Equal(t, start.plus(70_000), speech.End)
	f := subclipFields(t, body, speech.UUID)
	t.Logf("speech chapter fields: %v", f)
	assert.Equal(t, []string{"Kåre J. Smith, Harald Hansen"}, f[vscommon.FieldTitle.Value])
	assert.Equal(t, []string{"tale"}, f[vscommon.FieldSubclipType.Value])
	assert.ElementsMatch(t, []string{"Kåre J. Smith", "Harald Hansen"}, f[vscommon.FieldPersonsAppearing.Value])
	assert.Equal(t, []string{"John 3:16"}, f[vscommon.FieldDescription.Value])
	assert.Equal(t, []string{exportAsChapterValue}, f[fieldExportAsChapter])
	assert.Equal(t, []string{"newsubclip"}, f[fieldSubclipSystem])

	song := byStart[start.plus(70_000)]
	f = subclipFields(t, body, song.UUID)
	assert.Equal(t, []string{"singalong"}, f[vscommon.FieldSubclipType.Value])

	if os.Getenv("VIDISPINE_TEST_KEEP") != "" {
		var keep []string
		for _, s := range ours {
			keep = append(keep, s.UUID+"|"+s.Start+"|"+s.End)
		}
		t.Logf("chapters left on %s; remove them with VIDISPINE_TEST_REMOVE=%q", vxID, strings.Join(keep, ","))
		return
	}

	// 2. Export again with two markers: the three are replaced.
	second := []editorial.Marker{
		{Name: "Tale", Type: "tale", StartMs: 10_000, EndMs: 100_000},
		{Name: "Sang", Type: "sang", StartMs: 100_000, EndMs: 160_000},
	}
	res, err = api.exportChapters(ctx, sess, second, start)
	require.NoError(t, err)
	assert.EqualValues(t, 2, res.Written)
	assert.EqualValues(t, 3, res.Removed)

	body, err = vs.GetItemMetadata(ctx, vxID)
	require.NoError(t, err)
	now, err := parseSubclips(body)
	require.NoError(t, err)
	assert.Len(t, newSubclips(before, now), 2, "only the second export's chapters are left")
	for _, s := range ours {
		_, still := now[s.UUID]
		assert.False(t, still, "first export's %s was removed", s.UUID)
	}

	// 3. Clean up: exporting no markers removes ours.
	res, err = api.exportChapters(ctx, sess, nil, start)
	require.NoError(t, err)
	assert.EqualValues(t, 2, res.Removed)
	body, err = vs.GetItemMetadata(ctx, vxID)
	require.NoError(t, err)
	final, err := parseSubclips(body)
	require.NoError(t, err)
	assert.Empty(t, newSubclips(before, final), "the item is back to how it was")
}

// TestRemoveTestChapters removes the chapters a VIDISPINE_TEST_KEEP run left,
// as listed in VIDISPINE_TEST_REMOVE ("uuid|start|end,…").
func TestRemoveTestChapters(t *testing.T) {
	url, vxID, list := os.Getenv("VIDISPINE_TEST_URL"), os.Getenv("VIDISPINE_TEST_VXID"), os.Getenv("VIDISPINE_TEST_REMOVE")
	if url == "" || vxID == "" || list == "" {
		t.Skip("set VIDISPINE_TEST_URL, VIDISPINE_TEST_VXID and VIDISPINE_TEST_REMOVE")
	}
	ctx := context.Background()
	vs := newVidispineMetadata(url, os.Getenv("VIDISPINE_USERNAME"), os.Getenv("VIDISPINE_PASSWORD"))
	var remove []subclip
	for _, entry := range strings.Split(list, ",") {
		parts := strings.Split(entry, "|")
		require.Len(t, parts, 3, "uuid|start|end")
		remove = append(remove, subclip{UUID: parts[0], Start: parts[1], End: parts[2]})
	}
	doc, err := removeSubclipsDocument(remove)
	require.NoError(t, err)
	require.NoError(t, vs.PutItemMetadata(ctx, vxID, doc))

	body, err := vs.GetItemMetadata(ctx, vxID)
	require.NoError(t, err)
	left, err := parseSubclips(body)
	require.NoError(t, err)
	for _, s := range remove {
		assert.NotContains(t, left, s.UUID)
	}
}
