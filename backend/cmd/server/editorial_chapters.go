package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"context"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vscommon"
	"github.com/go-resty/resty/v2"
)

// Chapters in Mediabanken are Cantemo subclips: Vidispine timed metadata in
// the "Subclips" group, one group instance per chapter, all its fields
// sharing one timespan. Neither Cantemo nor bcc-media-flows has an API for
// writing them, so this writes the metadata the way Cantemo itself does.
//
// Timespans are absolute: they include the item's start timecode (Portal's
// default VIDISPINE_TIMESPAN_BASE), so a marker's offset is added to it.

const subclipsGroup = "Subclips"

// Fields of the Subclips group (Vidispine field-group "Subclips").
const (
	fieldSubclipSystem   = "portal_mf417206" // "SubClips System Field"; Cantemo sets "newsubclip"
	fieldExportAsChapter = "portal_mf457300" // checkbox "Export as Chapter"
	exportAsChapterValue = "export_as_chapter"
)

// cantemoSubclipTypes maps the tool's marker types to Cantemo's subclip type
// lookup (portal_mf594493). Types with no Cantemo equivalent (annet, video)
// are exported without a type.
var cantemoSubclipTypes = map[string]string{
	"sang":         "sang",
	"allsang":      "singalong",
	"tale":         "tale",
	"vitnesbyrd":   "vitnesbyrd",
	"bønn":         "bnn",
	"tydning":      "tydning",
	"programleder": "programleder",
	"intervju":     "intervju",
}

// vidispineMetadata reads and writes an item's metadata documents.
type vidispineMetadata interface {
	GetItemMetadata(ctx context.Context, vxID string) ([]byte, error)
	PutItemMetadata(ctx context.Context, vxID string, doc []byte) error
}

type restVidispineMetadata struct{ rest *resty.Client }

// newVidispineMetadata talks to Vidispine directly, like the Vault tool.
func newVidispineMetadata(baseURL, username, password string) vidispineMetadata {
	rest := resty.New().
		SetBaseURL(strings.TrimRight(baseURL, "/")).
		SetBasicAuth(username, password).
		SetTimeout(60 * time.Second).
		SetDisableWarn(true)
	return restVidispineMetadata{rest: rest}
}

func (v restVidispineMetadata) GetItemMetadata(ctx context.Context, vxID string) ([]byte, error) {
	resp, err := v.rest.R().SetContext(ctx).
		SetHeader("Accept", "application/xml").
		SetPathParam("id", vxID).
		Get("/item/{id}/metadata")
	if err != nil {
		return nil, fmt.Errorf("get metadata for %s: %w", vxID, err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("get metadata for %s: %s: %s", vxID, resp.Status(), resp.String())
	}
	return resp.Body(), nil
}

func (v restVidispineMetadata) PutItemMetadata(ctx context.Context, vxID string, doc []byte) error {
	resp, err := v.rest.R().SetContext(ctx).
		SetHeader("Content-Type", "application/xml").
		SetHeader("Accept", "application/xml").
		SetPathParam("id", vxID).
		SetBody(doc).
		Put("/item/{id}/metadata")
	if err != nil {
		return fmt.Errorf("set metadata for %s: %w", vxID, err)
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("set metadata for %s: %s: %s", vxID, resp.Status(), resp.String())
	}
	return nil
}

// ExportEditorialChapters writes the session's markers to its asset as
// subclips, replacing the subclips an earlier export wrote.
func (e EditorialAPI) ExportEditorialChapters(ctx context.Context, req *connect.Request[apiv1.ExportEditorialChaptersRequest]) (*connect.Response[apiv1.ExportEditorialChaptersResponse], error) {
	if _, err := requireEditorial(req, apiv1.EditorialLive); err != nil {
		return nil, err
	}
	if e.vsMetadata == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("vidispine is not configured"))
	}
	sess, err := e.queries.GetSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(err)
	}
	if sess.Vxid == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("attach the recording before saving chapters"))
	}
	markers, err := e.queries.ListMarkersForSession(ctx, sess.ID)
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: list markers: %w", err))
	}

	meta, err := fetchRecordingMetadata(e.vidispine, sess.Vxid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	startTC, err := parseTimecode(meta.Get(vscommon.FieldStartTC, "0@PAL"))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("start timecode of %s: %w", sess.Vxid, err))
	}

	res, err := e.exportChapters(ctx, sess, markers, startTC)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

// exportChapters writes markers as subclips on the session's asset and
// removes the ones the previous export wrote. Errors are connect errors.
func (e EditorialAPI) exportChapters(ctx context.Context, sess editorial.Session, markers []editorial.Marker, startTC timecode) (*apiv1.ExportEditorialChaptersResponse, error) {
	chapters, skipped := chaptersFromMarkers(markers, startTC)
	res := &apiv1.ExportEditorialChaptersResponse{Skipped: int32(skipped)}

	previous, err := e.queries.ListExportedChapters(ctx, editorial.ListExportedChaptersParams{SessionID: sess.ID, Vxid: sess.Vxid})
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: list exported chapters: %w", err))
	}

	// New subclips are told apart from existing ones by comparing the
	// item's subclips before and after the write.
	before, err := e.subclipUUIDs(ctx, sess.Vxid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Write first, remove the old ones after: a failure in between leaves
	// duplicates for the next export to clean up, never a missing chapter.
	if len(chapters) > 0 {
		doc, err := chaptersDocument(chapters)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if err := e.vsMetadata.PutItemMetadata(ctx, sess.Vxid, doc); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("write chapters: %w", err))
		}
	}
	after, err := e.subclipUUIDs(ctx, sess.Vxid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("chapters were written but could not be read back: %w", err))
	}
	written := newSubclips(before, after)

	// Record what was written before removing anything, so the next export
	// knows which subclips are ours even if the removal below fails.
	now := time.Now().UnixMilli()
	for _, s := range written {
		if err := e.queries.InsertExportedChapter(ctx, editorial.InsertExportedChapterParams{
			SessionID: sess.ID, Vxid: sess.Vxid, Uuid: s.UUID, StartTc: s.Start, EndTc: s.End, ExportedAt: now,
		}); err != nil {
			return nil, editorialErr(fmt.Errorf("editorial: record exported chapter: %w", err))
		}
	}
	res.Written = int32(len(written))

	var stale []subclip
	for _, p := range previous {
		if _, still := after[p.Uuid]; still {
			stale = append(stale, subclip{UUID: p.Uuid, Start: p.StartTc, End: p.EndTc})
		}
	}
	if len(stale) > 0 {
		doc, err := removeSubclipsDocument(stale)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if err := e.vsMetadata.PutItemMetadata(ctx, sess.Vxid, doc); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("new chapters were written, but the previous ones could not be removed: %w", err))
		}
	}
	for _, p := range previous {
		if err := e.queries.DeleteExportedChapter(ctx, editorial.DeleteExportedChapterParams{SessionID: sess.ID, Uuid: p.Uuid}); err != nil {
			return nil, editorialErr(fmt.Errorf("editorial: forget exported chapter: %w", err))
		}
	}
	res.Removed = int32(len(stale))
	return res, nil
}

// subclipUUIDs lists the item's Subclips group instances by uuid.
func (e EditorialAPI) subclipUUIDs(ctx context.Context, vxID string) (map[string]subclip, error) {
	body, err := e.vsMetadata.GetItemMetadata(ctx, vxID)
	if err != nil {
		return nil, err
	}
	return parseSubclips(body)
}

// timecode is a Vidispine time position: samples at a timebase ("PAL" for
// 25 fps, otherwise samples per second, e.g. "1000000" or "48000").
type timecode struct {
	Samples  int64
	Timebase string
	Rate     float64
}

func parseTimecode(tc string) (timecode, error) {
	samplesPart, timebase, ok := strings.Cut(strings.TrimSpace(tc), "@")
	if !ok {
		return timecode{}, fmt.Errorf("timecode %q has no timebase", tc)
	}
	samples, err := strconv.ParseInt(samplesPart, 10, 64)
	if err != nil {
		return timecode{}, fmt.Errorf("timecode %q: %w", tc, err)
	}
	rate := 25.0
	if timebase != "PAL" {
		if rate, err = strconv.ParseFloat(timebase, 64); err != nil || rate <= 0 {
			return timecode{}, fmt.Errorf("timecode %q: unsupported timebase %q", tc, timebase)
		}
	}
	return timecode{Samples: samples, Timebase: timebase, Rate: rate}, nil
}

// plus is the position offsetMs after tc, in tc's timebase.
func (tc timecode) plus(offsetMs int64) string {
	samples := tc.Samples + int64(math.Round(float64(offsetMs)*tc.Rate/1000))
	return fmt.Sprintf("%d@%s", samples, tc.Timebase)
}

// chapter is one subclip to write.
type chapter struct {
	Start, End  string
	Title       string
	Type        string
	Persons     []string
	Description string
	Export      bool
}

// chaptersFromMarkers maps markers to subclips. Markers without a length are
// skipped: Cantemo can't hold a zero-length subclip.
func chaptersFromMarkers(markers []editorial.Marker, start timecode) ([]chapter, int) {
	var out []chapter
	skipped := 0
	for _, m := range markers {
		if m.EndMs <= m.StartMs {
			skipped++
			continue
		}
		c := chapter{
			Start:       start.plus(m.StartMs),
			End:         start.plus(m.EndMs),
			Title:       m.Name,
			Type:        cantemoSubclipTypes[m.Type],
			Description: m.BibleVerses,
			Export:      m.PublishBmm || m.PublishBcc,
		}
		for _, p := range strings.Split(m.Contributors, ",") {
			if p = strings.TrimSpace(p); p != "" {
				c.Persons = append(c.Persons, p)
			}
		}
		// A speaker segment often has only a name: that's what the
		// chapter is called in Cantemo, too.
		if c.Title == "" {
			c.Title = strings.Join(c.Persons, ", ")
		}
		if c.Start == c.End {
			skipped++
			continue
		}
		out = append(out, c)
	}
	return out, skipped
}

// MetadataDocument as Vidispine reads and writes it.
type mdDocument struct {
	XMLName   xml.Name     `xml:"http://xml.vidispine.com/schema/vidispine MetadataDocument"`
	Timespans []mdTimespan `xml:"timespan"`
}

type mdTimespan struct {
	Start  string    `xml:"start,attr"`
	End    string    `xml:"end,attr"`
	Groups []mdGroup `xml:"group"`
}

type mdGroup struct {
	Mode   string    `xml:"mode,attr,omitempty"`
	UUID   string    `xml:"uuid,attr,omitempty"`
	Name   string    `xml:"name,omitempty"`
	Fields []mdField `xml:"field"`
}

type mdField struct {
	Name   string   `xml:"name"`
	Values []string `xml:"value"`
}

// chaptersDocument adds one Subclips group instance per chapter.
func chaptersDocument(chapters []chapter) ([]byte, error) {
	doc := mdDocument{}
	for _, c := range chapters {
		fields := []mdField{
			{Name: vscommon.FieldTitle.Value, Values: []string{c.Title}},
			{Name: fieldSubclipSystem, Values: []string{"newsubclip"}},
		}
		if c.Type != "" {
			fields = append(fields, mdField{Name: vscommon.FieldSubclipType.Value, Values: []string{c.Type}})
		}
		if len(c.Persons) > 0 {
			fields = append(fields, mdField{Name: vscommon.FieldPersonsAppearing.Value, Values: c.Persons})
		}
		if c.Description != "" {
			fields = append(fields, mdField{Name: vscommon.FieldDescription.Value, Values: []string{c.Description}})
		}
		if c.Export {
			fields = append(fields, mdField{Name: fieldExportAsChapter, Values: []string{exportAsChapterValue}})
		}
		doc.Timespans = append(doc.Timespans, mdTimespan{
			Start:  c.Start,
			End:    c.End,
			Groups: []mdGroup{{Mode: "add", Name: subclipsGroup, Fields: fields}},
		})
	}
	return marshalDocument(doc)
}

// removeSubclipsDocument removes Subclips group instances by uuid.
func removeSubclipsDocument(subclips []subclip) ([]byte, error) {
	doc := mdDocument{}
	for _, s := range subclips {
		doc.Timespans = append(doc.Timespans, mdTimespan{
			Start:  s.Start,
			End:    s.End,
			Groups: []mdGroup{{Mode: "remove", UUID: s.UUID, Name: subclipsGroup}},
		})
	}
	return marshalDocument(doc)
}

func marshalDocument(doc mdDocument) ([]byte, error) {
	body, err := xml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("build metadata document: %w", err)
	}
	return append([]byte(xml.Header), body...), nil
}

// subclip is one Subclips group instance on an item.
type subclip struct {
	UUID       string
	Start, End string
}

// parseSubclips finds the Subclips group instances in an item metadata
// document (a MetadataListDocument from GET /item/{id}/metadata).
func parseSubclips(body []byte) (map[string]subclip, error) {
	var doc struct {
		Items []struct {
			Metadata []struct {
				Timespans []mdTimespan `xml:"timespan"`
			} `xml:"metadata"`
		} `xml:"item"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse item metadata: %w", err)
	}
	out := map[string]subclip{}
	for _, item := range doc.Items {
		for _, md := range item.Metadata {
			for _, ts := range md.Timespans {
				if ts.Start == "-INF" {
					continue
				}
				for _, g := range ts.Groups {
					if g.Name == subclipsGroup && g.UUID != "" {
						out[g.UUID] = subclip{UUID: g.UUID, Start: ts.Start, End: ts.End}
					}
				}
			}
		}
	}
	return out, nil
}

// newSubclips are the subclips in after that weren't in before.
func newSubclips(before, after map[string]subclip) []subclip {
	var out []subclip
	for uuid, s := range after {
		if _, existed := before[uuid]; !existed {
			out = append(out, s)
		}
	}
	return out
}
