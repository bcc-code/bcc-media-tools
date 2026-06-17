package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vsapi"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vscommon"
)

// VaultPageSize is the fixed number of items returned per search page.
const VaultPageSize = 50

// Vidispine mediaType values we surface as filter categories. Anything that is
// not one of the first three is bucketed into "other".
const (
	mediaTypeVideo = "video"
	mediaTypeAudio = "audio"
	mediaTypeImage = "image"
	mediaTypeOther = "other"
)

var vaultMediaCategories = []string{mediaTypeVideo, mediaTypeAudio, mediaTypeImage, mediaTypeOther}

// Vidispine terse-metadata field names that are not in vscommon's constant set.
var (
	fieldMediaType        = vscommon.FieldType{Value: "mediaType"}
	fieldCreated          = vscommon.FieldType{Value: "created"}
	fieldOriginalFormat   = vscommon.FieldType{Value: "originalFormat"}
	fieldOriginalFilename = vscommon.FieldType{Value: "originalFilename"}
)

type VaultAPI struct {
	vidispine *vsapi.Client
}

func NewVaultAPI(vidispine *vsapi.Client) *VaultAPI {
	return &VaultAPI{vidispine: vidispine}
}

// VaultSearch runs a full-text item search against Vidispine, paginated to
// VaultPageSize, optionally filtered by media type, with per-type facet counts.
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

	res, err := v.vidispine.SearchItems(vsapi.ItemSearchParams{
		Text:       strings.TrimSpace(req.Msg.GetQuery()),
		MediaTypes: vidispineMediaTypes(req.Msg.GetMediaTypes()),
		First:      int(page-1)*VaultPageSize + 1,
		Number:     VaultPageSize,
		Facet:      true,
	})
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
		Facets:    mediaTypeFacets(res.Facet),
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

func metadataToVaultItem(m *vsapi.MetadataResult) *apiv1.VaultItem {
	mediaType := normalizeMediaType(m.Get(fieldMediaType, ""))

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

// itemFormat derives a short, human format string: the original file extension
// when available (e.g. "mov", "wav"), falling back to Vidispine's originalFormat.
func itemFormat(m *vsapi.MetadataResult) string {
	if name := m.Get(fieldOriginalFilename, ""); name != "" {
		if i := strings.LastIndex(name, "."); i >= 0 && i < len(name)-1 {
			return strings.ToLower(name[i+1:])
		}
	}
	return m.Get(fieldOriginalFormat, "")
}

func normalizeMediaType(mt string) string {
	switch strings.ToLower(mt) {
	case mediaTypeVideo, mediaTypeAudio, mediaTypeImage:
		return strings.ToLower(mt)
	default:
		return mediaTypeOther
	}
}

// vidispineMediaTypes maps the UI filter categories to the mediaType values
// Vidispine understands. "other" cannot be expressed as a positive mediaType
// criterion, so it is dropped from the server-side filter.
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

// mediaTypeFacets folds Vidispine's mediaType facet counts into our four
// categories (video / audio / image / other), always returning all four.
func mediaTypeFacets(facets []vsapi.SearchFacet) []*apiv1.VaultFacet {
	counts := map[string]int32{}
	for _, f := range facets {
		if f.Field != "mediaType" {
			continue
		}
		for _, c := range f.Count {
			counts[normalizeMediaType(c.FieldValue)] += int32(c.Count)
		}
	}

	out := make([]*apiv1.VaultFacet, 0, len(vaultMediaCategories))
	for _, cat := range vaultMediaCategories {
		out = append(out, &apiv1.VaultFacet{MediaType: cat, Count: counts[cat]})
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
