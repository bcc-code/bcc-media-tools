package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vsapi"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vscommon"
	"github.com/bcc-code/mediabank-bridge/log"
	"github.com/go-resty/resty/v2"
)

// VaultPageSize is the fixed number of items returned per search page.
const VaultPageSize = 50

// Media-type filter categories we surface in the UI.
const (
	mediaTypeVideo = "video"
	mediaTypeAudio = "audio"
	mediaTypeImage = "image"
	// mediaTypeOther is still used internally by deriveMediaType for items
	// whose mimeType doesn't match the three surfaced categories, but it is
	// not exposed as a UI filter — Vidispine has no stable way to express
	// "everything except video/audio/image" as a search criterion.
	mediaTypeOther = "other"
)

var vaultMediaCategories = []string{mediaTypeVideo, mediaTypeAudio, mediaTypeImage}

// Vidispine terse-metadata field names not present in vscommon's constant set.
var (
	fieldCreated          = vscommon.FieldType{Value: "created"}
	fieldMimeType         = vscommon.FieldType{Value: "mimeType"}
	fieldOriginalFormat   = vscommon.FieldType{Value: "originalFormat"}
	fieldOriginalFilename = vscommon.FieldType{Value: "originalFilename"}
)

type VaultAPI struct {
	// vidispine is the shared library client, used for per-item metadata/shapes.
	vidispine *vsapi.Client
	// rest is a dedicated basic-auth client for the search + thumbnail calls,
	// kept here so the request/response shapes can be tuned against the live
	// Vidispine instance without a library release cycle.
	baseURL string
	rest    *resty.Client
}

func NewVaultAPI(vidispine *vsapi.Client, baseURL, username, password string) *VaultAPI {
	rest := resty.New()
	rest.SetBasicAuth(username, password)
	rest.SetBaseURL(baseURL)
	rest.SetHeader("Accept", "application/json")
	rest.SetDisableWarn(true)
	return &VaultAPI{vidispine: vidispine, baseURL: baseURL, rest: rest}
}

// --- Vidispine item search (PUT /item with an ItemSearchDocument) ---

type vaultSearchResult struct {
	Hits  int                     `json:"hits"`
	Items []*vsapi.MetadataResult `json:"item"`
}

// wildcardText wraps each whitespace-separated word in the query with '*' so
// Vidispine does substring matching (e.g. "safari interview" ->
// "*safari* *interview*"). Empty input is returned unchanged so an empty search
// still matches everything.
func wildcardText(text string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	for i, w := range words {
		words[i] = "*" + w + "*"
	}
	return strings.Join(words, " ")
}

// buildItemSearchXML builds the ItemSearchDocument body: a free-text query
// plus an optional multi-value mediaType filter.
func buildItemSearchXML(text string, mediaTypes []string) ([]byte, error) {
	type field struct {
		Name   string   `xml:"name"`
		Values []string `xml:"value"`
	}
	type doc struct {
		XMLName xml.Name `xml:"ItemSearchDocument"`
		Xmlns   string   `xml:"xmlns,attr"`
		Text    string   `xml:"text,omitempty"`
		Fields  []field  `xml:"field,omitempty"`
	}
	d := doc{Xmlns: "http://xml.vidispine.com/schema/vidispine", Text: text}
	if len(mediaTypes) > 0 {
		d.Fields = append(d.Fields, field{Name: "mediaType", Values: mediaTypes})
	}
	return xml.Marshal(d)
}

// searchVidispine runs a paginated item search returning terse metadata.
func (v VaultAPI) searchVidispine(text string, mediaTypes []string, first, number int) (*vaultSearchResult, error) {
	body, err := buildItemSearchXML(text, mediaTypes)
	if err != nil {
		return nil, err
	}

	result := &vaultSearchResult{}
	resp, err := v.rest.R().
		SetHeader("Content-Type", "application/xml").
		SetResult(result).
		SetQueryParams(map[string]string{
			"content": "metadata",
			"terse":   "true",
			"first":   strconv.Itoa(first),
			"number":  strconv.Itoa(number),
		}).
		SetBody(body).
		Put("/item")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("vidispine search failed (status %d): %s", resp.StatusCode(), string(resp.Body()))
	}
	return result, nil
}

// countItems returns just the hit count for a query (number=0, no metadata).
func (v VaultAPI) countItems(text string, mediaTypes []string) (int32, error) {
	body, err := buildItemSearchXML(text, mediaTypes)
	if err != nil {
		return 0, err
	}
	result := &vaultSearchResult{}
	resp, err := v.rest.R().
		SetHeader("Content-Type", "application/xml").
		SetResult(result).
		SetQueryParams(map[string]string{"number": "0"}).
		SetBody(body).
		Put("/item")
	if err != nil {
		return 0, err
	}
	if resp.IsError() {
		return 0, fmt.Errorf("vidispine count failed (status %d)", resp.StatusCode())
	}
	return int32(result.Hits), nil
}

// mediaTypeCounts produces the filter-sidebar counts. Vidispine faceting did
// not return usable values for this instance, so counts are derived from
// cheap concurrent number=0 count queries — one per media type.
func (v VaultAPI) mediaTypeCounts(text string) []*apiv1.VaultFacet {
	type result struct {
		key string
		n   int32
	}
	ch := make(chan result, len(vaultMediaCategories))

	var wg sync.WaitGroup
	for _, k := range vaultMediaCategories {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			n, err := v.countItems(text, []string{k})
			if err != nil {
				log.L.Debug().Err(err).Str("mediaType", k).Msg("vault: count query failed")
			}
			ch <- result{k, n}
		}(k)
	}
	wg.Wait()
	close(ch)

	counts := map[string]int32{}
	for r := range ch {
		counts[r.key] = r.n
	}

	out := make([]*apiv1.VaultFacet, 0, len(vaultMediaCategories))
	for _, cat := range vaultMediaCategories {
		out = append(out, &apiv1.VaultFacet{MediaType: cat, Count: counts[cat]})
	}
	return out
}

// --- Thumbnails ---

// fetchThumbnail returns raw thumbnail bytes for an item. The thumbnailresource
// URI is either the image itself or a URIListDocument of available frames; this
// handles both. frac (0..1) selects a frame along the asset for trick-play;
// empty/invalid picks a representative middle frame.
func (v VaultAPI) fetchThumbnail(vxID, frac string) ([]byte, string, error) {
	var list struct {
		URI []string `json:"uri"`
	}
	resp, err := v.rest.R().
		SetResult(&list).
		Get("/item/" + url.PathEscape(vxID) + "/thumbnailresource")
	if err != nil {
		return nil, "", err
	}
	if resp.IsError() {
		return nil, "", fmt.Errorf("listing thumbnail resources for %s failed (status %d)", vxID, resp.StatusCode())
	}
	if len(list.URI) == 0 {
		return nil, "", fmt.Errorf("no thumbnails for %s", vxID)
	}

	resource := v.absolutize(list.URI[0])

	// Fetch the resource: it is either the image directly, or a frame list.
	r, err := v.rest.R().Get(resource)
	if err != nil {
		return nil, "", err
	}
	if r.IsError() {
		return nil, "", fmt.Errorf("thumbnail resource fetch failed (status %d) for %s: %s", r.StatusCode(), vxID, resource)
	}

	ct := r.Header().Get("Content-Type")
	if strings.HasPrefix(ct, "image/") {
		return r.Body(), ct, nil
	}

	// Not an image: treat the body as a URIListDocument of frame entries. Each
	// entry is a timecode (or path ending in one) to append to the resource.
	var frames struct {
		URI []string `json:"uri"`
	}
	_ = json.Unmarshal(r.Body(), &frames)
	if len(frames.URI) == 0 {
		raw := r.Body()
		if len(raw) > 600 {
			raw = raw[:600]
		}
		return nil, "", fmt.Errorf("no thumbnail frames for %s (content-type %s): %s", vxID, ct, string(raw))
	}

	idx := len(frames.URI) / 2
	if frac != "" {
		if f, perr := strconv.ParseFloat(frac, 64); perr == nil {
			if f < 0 {
				f = 0
			} else if f > 1 {
				f = 1
			}
			idx = int(f*float64(len(frames.URI)-1) + 0.5)
		}
	}

	return v.fetchImage(frameURL(resource, frames.URI[idx]))
}

// frameURL builds the URL for a single thumbnail frame. A frame entry is either
// an absolute URL, or a timecode/path whose last segment is appended to the
// resource URL (e.g. resource + "/50").
func frameURL(resource, entry string) string {
	if strings.HasPrefix(entry, "http") {
		return entry
	}
	if i := strings.LastIndex(entry, "/"); i >= 0 {
		entry = entry[i+1:]
	}
	return resource + "/" + entry
}

func (v VaultAPI) fetchImage(u string) ([]byte, string, error) {
	resp, err := v.rest.R().SetHeader("Accept", "image/jpeg").Get(u)
	if err != nil {
		return nil, "", err
	}
	if resp.IsError() {
		return nil, "", fmt.Errorf("thumbnail image fetch failed (status %d): %s", resp.StatusCode(), u)
	}
	ct := resp.Header().Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	return resp.Body(), ct, nil
}

// fetchWaveform returns Vidispine's pre-rendered waveform PNG for an audio
// item. See https://apidoc.vidispine.com/4.17/ref/item/analyze.html — the
// endpoint requires that shape analysis has run on the item.
func (v VaultAPI) fetchWaveform(vxID string, width, height int, bgColor, fgColor string) ([]byte, error) {
	req := v.rest.R().SetQueryParams(map[string]string{
		"width":  strconv.Itoa(width),
		"height": strconv.Itoa(height),
	})
	if bgColor != "" {
		req.SetQueryParam("bgcolor", bgColor)
	}
	if fgColor != "" {
		req.SetQueryParam("fgcolor", fgColor)
	}
	resp, err := req.Get("/item/" + url.PathEscape(vxID) + "/waveform/image")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("waveform fetch failed (status %d): %s", resp.StatusCode(), string(resp.Body()))
	}
	return resp.Body(), nil
}

// absolutize resolves an API-relative URI against the Vidispine host. Vidispine
// usually returns absolute URIs already, in which case this is a no-op.
func (v VaultAPI) absolutize(u string) string {
	if strings.HasPrefix(u, "http") {
		return u
	}
	if b, err := url.Parse(v.baseURL); err == nil {
		return b.Scheme + "://" + b.Host + u
	}
	return u
}

// --- RPC handlers ---

// VaultSearch runs a full-text item search against Vidispine, paginated to
// VaultPageSize, optionally filtered by media type, with per-type counts.
func (v VaultAPI) VaultSearch(_ context.Context, req *connect.Request[apiv1.VaultSearchRequest]) (*connect.Response[apiv1.VaultSearchResponse], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	if !PermissionsForEmail(email).CanVault() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized"))
	}

	page := req.Msg.GetPage()
	if page < 1 {
		page = 1
	}
	text := wildcardText(req.Msg.GetQuery())

	res, err := v.searchVidispine(
		text,
		vidispineMediaTypes(req.Msg.GetMediaTypes()),
		int(page-1)*VaultPageSize+1,
		VaultPageSize,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*apiv1.VaultItem, 0, len(res.Items))
	for _, m := range res.Items {
		items = append(items, metadataToVaultItem(m))
	}

	return connect.NewResponse(&apiv1.VaultSearchResponse{
		Items:     items,
		TotalHits: int32(res.Hits),
		Page:      page,
		PageSize:  VaultPageSize,
		Facets:    v.mediaTypeCounts(text),
	}), nil
}

// GetVaultItem returns a single item's metadata (plus file size) for the detail view.
func (v VaultAPI) GetVaultItem(_ context.Context, req *connect.Request[apiv1.GetVaultItemRequest]) (*connect.Response[apiv1.GetVaultItemResponse], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	if !PermissionsForEmail(email).CanVault() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized"))
	}

	vxID := req.Msg.GetVXID()
	if vxID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing VXID"))
	}

	meta, err := v.vidispine.GetMetadata(vxID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	item := metadataToVaultItem(meta)
	item.VXID = vxID
	if size := v.originalShapeSize(vxID); size > 0 {
		item.Size = humanSize(size)
	}

	return connect.NewResponse(&apiv1.GetVaultItemResponse{Item: item}), nil
}

// originalShapeSize returns the byte size of the item's original shape, or 0.
func (v VaultAPI) originalShapeSize(vxID string) int64 {
	shapes, err := v.vidispine.GetShapes(vxID)
	if err != nil || shapes == nil {
		return 0
	}
	shape := shapes.GetShape("original")
	if shape == nil {
		return 0
	}
	for _, f := range shape.ContainerComponent.File {
		if f.Size > 0 {
			return f.Size
		}
	}
	for _, bc := range shape.BinaryComponent {
		for _, f := range bc.File {
			if f.Size > 0 {
				return f.Size
			}
		}
	}
	for _, vc := range shape.VideoComponent {
		for _, f := range vc.File {
			if f.Size > 0 {
				return f.Size
			}
		}
	}
	return 0
}

// --- metadata mapping ---

func metadataToVaultItem(m *vsapi.MetadataResult) *apiv1.VaultItem {
	mediaType := deriveMediaType(m)

	title := m.Get(vscommon.FieldTitle, "")
	if title == "" {
		title = m.Get(fieldOriginalFilename, m.ID)
	}

	duration := 0
	if d := m.Get(vscommon.FieldDurationSeconds, ""); d != "" {
		if f, err := strconv.ParseFloat(d, 64); err == nil {
			duration = int(f)
		}
	}

	return &apiv1.VaultItem{
		VXID:            m.ID,
		Title:           title,
		MediaType:       mediaType,
		Added:           humanDate(m.Get(fieldCreated, "")),
		Format:          itemFormat(m),
		DurationSeconds: int32(duration),
		HasPreview:      mediaType == mediaTypeVideo || mediaType == mediaTypeAudio,
	}
}

// deriveMediaType classifies an item. The Vidispine "mediaType" field is not in
// the terse metadata for this instance, so we derive it from the mimeType
// (e.g. "video/quicktime" -> video).
func deriveMediaType(m *vsapi.MetadataResult) string {
	mime := strings.ToLower(m.Get(fieldMimeType, ""))
	switch {
	case strings.HasPrefix(mime, "video/"):
		return mediaTypeVideo
	case strings.HasPrefix(mime, "audio/"):
		return mediaTypeAudio
	case strings.HasPrefix(mime, "image/"):
		return mediaTypeImage
	}
	return mediaTypeOther
}

// itemFormat derives a short, human format string: the original file extension
// when available (e.g. "mov", "wav"), then the mimeType subtype, then the first
// token of Vidispine's (often comma-joined) originalFormat.
func itemFormat(m *vsapi.MetadataResult) string {
	for _, name := range []string{m.Get(fieldOriginalFilename, ""), m.Get(vscommon.FieldTitle, "")} {
		if i := strings.LastIndex(name, "."); i >= 0 && i < len(name)-1 && len(name)-i <= 6 {
			return strings.ToLower(name[i+1:])
		}
	}
	if mime := m.Get(fieldMimeType, ""); strings.Contains(mime, "/") {
		return strings.ToLower(strings.SplitN(mime, "/", 2)[1])
	}
	format := m.Get(fieldOriginalFormat, "")
	if i := strings.IndexByte(format, ','); i > 0 {
		return format[:i]
	}
	return format
}

// vidispineMediaTypes maps the UI filter categories to the mediaType values
// Vidispine understands. Only the three "standard" categories are surfaced
// in the UI; any other category is silently dropped.
func vidispineMediaTypes(categories []string) []string {
	out := make([]string, 0, len(categories))
	for _, c := range categories {
		switch strings.ToLower(c) {
		case mediaTypeVideo, mediaTypeAudio, mediaTypeImage:
			out = append(out, strings.ToLower(c))
		}
	}
	return out
}

// humanDate formats a Vidispine ISO timestamp into a readable date, falling
// back to the raw value if it cannot be parsed.
func humanDate(s string) string {
	if s == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000-0700"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("Jan 2 2006, 3:04 pm")
		}
	}
	return s
}

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
