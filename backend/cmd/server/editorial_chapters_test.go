package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"

	"bcc-media-tools/editorial"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeVidispine keeps an item's subclips and applies the MetadataDocuments
// the export writes: mode="add" groups get a new uuid, mode="remove" groups
// are dropped by uuid.
type fakeVidispine struct {
	subclips map[string]fakeSubclip // by uuid
	puts     [][]byte
	nextID   int
	failPut  int // fail the nth PutItemMetadata (1-based); 0 never
}

type fakeSubclip struct {
	start, end string
	fields     []mdField
}

func newFakeVidispine() *fakeVidispine {
	return &fakeVidispine{subclips: map[string]fakeSubclip{}}
}

func (f *fakeVidispine) GetItemMetadata(context.Context, string) ([]byte, error) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><MetadataListDocument xmlns="http://xml.vidispine.com/schema/vidispine"><item id="VX-1"><metadata>`)
	b.WriteString(`<timespan start="-INF" end="+INF"><field><name>title</name><value>Item</value></field></timespan>`)
	for uuid, s := range f.subclips {
		fmt.Fprintf(&b, `<timespan start="%s" end="%s"><group uuid="%s"><name>Subclips</name></group></timespan>`, s.start, s.end, uuid)
	}
	b.WriteString(`</metadata></item></MetadataListDocument>`)
	return []byte(b.String()), nil
}

func (f *fakeVidispine) PutItemMetadata(_ context.Context, _ string, doc []byte) error {
	f.puts = append(f.puts, doc)
	if f.failPut == len(f.puts) {
		return fmt.Errorf("vidispine is down")
	}
	var parsed mdDocument
	if err := xml.Unmarshal(doc, &parsed); err != nil {
		return err
	}
	for _, ts := range parsed.Timespans {
		for _, g := range ts.Groups {
			switch g.Mode {
			case "add":
				f.nextID++
				f.subclips[fmt.Sprintf("uuid-%d", f.nextID)] = fakeSubclip{start: ts.Start, end: ts.End, fields: g.Fields}
			case "remove":
				delete(f.subclips, g.UUID)
			}
		}
	}
	return nil
}

func (f *fakeVidispine) field(uuid, name string) []string {
	for _, fl := range f.subclips[uuid].fields {
		if fl.Name == name {
			return fl.Values
		}
	}
	return nil
}

func TestTimecodePlus(t *testing.T) {
	for _, tt := range []struct {
		start    string
		offsetMs int64
		want     string
	}{
		// The sample item in bcc-media-flows: start 1699048@PAL, a chapter
		// 1172.8 s in sits at 1728368@PAL.
		{"1699048@PAL", 1_172_800, "1728368@PAL"},
		{"0@PAL", 0, "0@PAL"},
		{"0@PAL", 41, "1@PAL"}, // rounded to the nearest frame
		{"6315480000@1000000", 1_000, "6316480000@1000000"},
		{"480000@48000", 500, "504000@48000"},
	} {
		tc, err := parseTimecode(tt.start)
		require.NoError(t, err)
		assert.Equal(t, tt.want, tc.plus(tt.offsetMs), "%s + %dms", tt.start, tt.offsetMs)
	}
	for _, bad := range []string{"", "1699048", "x@PAL", "10@NTSC", "10@0"} {
		_, err := parseTimecode(bad)
		assert.Error(t, err, bad)
	}
}

func TestChaptersFromMarkers(t *testing.T) {
	start, _ := parseTimecode("1000@PAL")
	got, skipped := chaptersFromMarkers([]editorial.Marker{
		{Name: "", Contributors: "Kåre J. Smith, Harald Hansen", Type: "tale", BibleVerses: "John 3:16", StartMs: 0, EndMs: 60_000, PublishBmm: true},
		{Name: "HV 16", Type: "allsang", StartMs: 60_000, EndMs: 120_000},
		{Name: "Bønn", Type: "bønn", StartMs: 120_000, EndMs: 130_000, PublishBcc: true},
		{Name: "Andre ting", Type: "annet", StartMs: 130_000, EndMs: 140_000},
		{Name: "Running", StartMs: 140_000, EndMs: 0},
		{Name: "Sub-frame", StartMs: 150_000, EndMs: 150_010},
	}, start)

	assert.Equal(t, 2, skipped, "no length and sub-frame length")
	require.Len(t, got, 4)
	assert.Equal(t, chapter{
		Start: "1000@PAL", End: "2500@PAL",
		Title: "Kåre J. Smith, Harald Hansen", Type: "tale",
		Persons: []string{"Kåre J. Smith", "Harald Hansen"}, Description: "John 3:16", Export: true,
	}, got[0])
	assert.Equal(t, "singalong", got[1].Type)
	assert.False(t, got[1].Export)
	assert.Equal(t, "bnn", got[2].Type)
	assert.True(t, got[2].Export)
	assert.Empty(t, got[3].Type, "annet has no Cantemo type")
}

func TestChaptersDocument(t *testing.T) {
	doc, err := chaptersDocument([]chapter{{
		Start: "1000@PAL", End: "2500@PAL", Title: "Kåre & co", Type: "tale",
		Persons: []string{"A", "B"}, Description: "John 3:16", Export: true,
	}})
	require.NoError(t, err)
	s := string(doc)
	assert.True(t, strings.HasPrefix(s, `<?xml version="1.0" encoding="UTF-8"?>`))
	for _, want := range []string{
		`<MetadataDocument xmlns="http://xml.vidispine.com/schema/vidispine">`,
		`<timespan start="1000@PAL" end="2500@PAL"><group mode="add"><name>Subclips</name>`,
		`<field><name>title</name><value>Kåre &amp; co</value></field>`,
		`<field><name>portal_mf417206</name><value>newsubclip</value></field>`,
		`<field><name>portal_mf594493</name><value>tale</value></field>`,
		`<field><name>portal_mf50574</name><value>A</value><value>B</value></field>`,
		`<field><name>portal_mf982016</name><value>John 3:16</value></field>`,
		`<field><name>portal_mf457300</name><value>export_as_chapter</value></field>`,
	} {
		assert.Contains(t, s, want)
	}

	// Optional fields are left out rather than written empty.
	doc, err = chaptersDocument([]chapter{{Start: "0@PAL", End: "25@PAL", Title: "x"}})
	require.NoError(t, err)
	for _, absent := range []string{"portal_mf594493", "portal_mf50574", "portal_mf982016", "portal_mf457300"} {
		assert.NotContains(t, string(doc), absent)
	}
}

func TestParseSubclips(t *testing.T) {
	body := []byte(`<?xml version="1.0"?><MetadataListDocument xmlns="http://xml.vidispine.com/schema/vidispine"><item id="VX-1"><metadata>
		<timespan start="-INF" end="+INF"><group uuid="item-group"><name>Subclips</name></group></timespan>
		<timespan start="10@PAL" end="20@PAL"><group uuid="a"><name>Subclips</name><field><name>title</name><value>x</value></field></group><group uuid="m"><name>AvMarker</name></group></timespan>
	</metadata></item></MetadataListDocument>`)
	got, err := parseSubclips(body)
	require.NoError(t, err)
	assert.Equal(t, map[string]subclip{"a": {UUID: "a", Start: "10@PAL", End: "20@PAL"}}, got)
}

func newChapterTestAPI(t *testing.T) (*EditorialAPI, *fakeVidispine, editorial.Session) {
	api := newTestEditorialAPI(t)
	vs := newFakeVidispine()
	api.vsMetadata = vs
	created := createTestSession(t, api, "VX-1", "Søndag")
	sess, err := api.queries.GetSession(context.Background(), created.Id)
	require.NoError(t, err)
	return api, vs, sess
}

// Exporting twice replaces the first export's chapters and leaves a chapter
// made by hand in Cantemo alone.
func TestExportChaptersReplacesOnlyOurs(t *testing.T) {
	api, vs, sess := newChapterTestAPI(t)
	ctx := context.Background()
	start, _ := parseTimecode("1000@PAL")
	vs.subclips["by-hand"] = fakeSubclip{start: "5000@PAL", end: "6000@PAL"}

	first := []editorial.Marker{
		{Name: "A", StartMs: 0, EndMs: 10_000},
		{Name: "B", StartMs: 10_000, EndMs: 20_000},
	}
	res, err := api.exportChapters(ctx, sess, first, start)
	require.NoError(t, err)
	assert.EqualValues(t, 2, res.Written)
	assert.EqualValues(t, 0, res.Removed)
	assert.Len(t, vs.subclips, 3)

	second := []editorial.Marker{{Name: "A renamed", StartMs: 0, EndMs: 20_000}}
	res, err = api.exportChapters(ctx, sess, second, start)
	require.NoError(t, err)
	assert.EqualValues(t, 1, res.Written)
	assert.EqualValues(t, 2, res.Removed)

	require.Len(t, vs.subclips, 2)
	assert.Contains(t, vs.subclips, "by-hand")
	var ours string
	for uuid := range vs.subclips {
		if uuid != "by-hand" {
			ours = uuid
		}
	}
	assert.Equal(t, []string{"A renamed"}, vs.field(ours, "title"))
	assert.Equal(t, "1000@PAL", vs.subclips[ours].start)
	assert.Equal(t, "1500@PAL", vs.subclips[ours].end)

	recorded, err := api.queries.ListExportedChapters(ctx, editorial.ListExportedChaptersParams{SessionID: sess.ID, Vxid: "VX-1"})
	require.NoError(t, err)
	require.Len(t, recorded, 1)
	assert.Equal(t, ours, recorded[0].Uuid)
}

// If removing the old chapters fails, the new ones stay recorded and the old
// ones stay pending, so the next export removes them.
func TestExportChaptersRetriesFailedRemoval(t *testing.T) {
	api, vs, sess := newChapterTestAPI(t)
	ctx := context.Background()
	start, _ := parseTimecode("0@PAL")
	markers := []editorial.Marker{{Name: "A", StartMs: 0, EndMs: 10_000}}

	_, err := api.exportChapters(ctx, sess, markers, start)
	require.NoError(t, err)

	vs.failPut = len(vs.puts) + 2 // the second export's removal
	_, err = api.exportChapters(ctx, sess, markers, start)
	require.Error(t, err)
	assert.Len(t, vs.subclips, 2, "new written, old not yet removed")

	vs.failPut = 0
	_, err = api.exportChapters(ctx, sess, markers, start)
	require.NoError(t, err)
	assert.Len(t, vs.subclips, 1, "a single current chapter after the retry")
}

// A chapter someone deleted in Cantemo since the last export isn't removed
// again (Vidispine would reject an unknown uuid).
func TestExportChaptersSkipsAlreadyDeleted(t *testing.T) {
	api, vs, sess := newChapterTestAPI(t)
	ctx := context.Background()
	start, _ := parseTimecode("0@PAL")
	markers := []editorial.Marker{{Name: "A", StartMs: 0, EndMs: 10_000}}

	_, err := api.exportChapters(ctx, sess, markers, start)
	require.NoError(t, err)
	for uuid := range vs.subclips {
		delete(vs.subclips, uuid)
	}

	puts := len(vs.puts)
	res, err := api.exportChapters(ctx, sess, markers, start)
	require.NoError(t, err)
	assert.EqualValues(t, 0, res.Removed)
	assert.Len(t, vs.puts, puts+1, "only the write, no removal")
}
