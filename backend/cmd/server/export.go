package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"
	bccmflows "github.com/bcc-code/bcc-media-flows"
	"github.com/bcc-code/bcc-media-flows/services/vidispine"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vsapi"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vscommon"
	"github.com/bcc-code/bcc-media-flows/utils"
	exportworkflows "github.com/bcc-code/bcc-media-flows/workflows/export"
	vbexportworkflows "github.com/bcc-code/bcc-media-flows/workflows/vb_export"
	"github.com/bcc-code/mediabank-bridge/log"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.temporal.io/sdk/client"
	"golang.org/x/sync/errgroup"
)

type ExportAPI struct {
	temporalClient client.Client
	vidispine      vidispine.Client
}

func NewExportAPI(vs vidispine.Client, temporalClient client.Client) *ExportAPI {
	return &ExportAPI{
		temporalClient: temporalClient,
		vidispine:      vs,
	}
}

// defaultResolutions mirrors trigger_ui: used when Vidispine returns no
// resolutions for the asset.
func defaultResolutions() []vsapi.Resolution {
	return []vsapi.Resolution{
		{Width: 1920, Height: 1080},
		{Width: 1280, Height: 720},
		{Width: 854, Height: 480},
		{Width: 640, Height: 360},
		{Width: 426, Height: 240},
	}
}

// overlayFilenames lists the watermark/overlay files available in OVERLAYS_DIR,
// with "None" prepended as the default.
func overlayFilenames() []string {
	overlays := []string{"None"}
	dir := os.Getenv("OVERLAYS_DIR")
	if dir == "" {
		return overlays
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		log.L.Warn().Err(err).Str("dir", dir).Msg("could not read OVERLAYS_DIR")
		return overlays
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		overlays = append(overlays, f.Name())
	}
	return overlays
}

// exportLanguages builds the selectable language list from the shared language
// config, ordered by language number. The code is the ISO6391 value, which is
// exactly what VXExportParams.Languages expects (matching trigger_ui).
func exportLanguages() []*apiv1.ExportLanguage {
	langs := lo.Values(bccmflows.LanguagesByISO)
	sort.Slice(langs, func(i, j int) bool {
		return langs[i].LanguageNumber < langs[j].LanguageNumber
	})
	return lo.Map(langs, func(l bccmflows.Language, _ int) *apiv1.ExportLanguage {
		return &apiv1.ExportLanguage{
			Code: l.ISO6391,
			Name: l.LanguageName,
			Mu1:  l.MU1ChannelCount > 0,
			Mu2:  l.MU2ChannelCount > 0,
		}
	})
}

var vxUIDestinationOrder = []exportworkflows.AssetExportDestination{
	exportworkflows.AssetExportDestinationVOD,
	exportworkflows.AssetExportDestinationBMM,
	exportworkflows.AssetExportDestinationBMMIntegration,
	exportworkflows.AssetExportDestinationIsilon,
}

func vxUIDestinations() []string {
	return lo.Map(vxUIDestinationOrder, func(d exportworkflows.AssetExportDestination, _ int) string {
		return d.Value
	})
}

// allowedDestinations returns the UI export destinations the given user is
// permitted to export to.
func allowedDestinations(perms *apiv1.Permissions) []string {
	out := []string{}
	for _, d := range vxUIDestinationOrder {
		if perms.CanExportTo(d.Value) {
			out = append(out, d.Value)
		}
	}
	return out
}

// dirFilenames returns the (sorted) file names in dir, or nil if it can't be read.
func dirFilenames(dir string) []string {
	if dir == "" {
		return nil
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		log.L.Warn().Err(err).Str("dir", dir).Msg("could not read directory")
		return nil
	}
	var names []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		names = append(names, f.Name())
	}
	sort.Strings(names)
	return names
}

var vbUIDestinationOrder = []vbexportworkflows.Destination{
	vbexportworkflows.DestinationAbekas,
	vbexportworkflows.DestinationRawAbekas,
	vbexportworkflows.DestinationBStage,
	vbexportworkflows.DestinationHippoV2,
	vbexportworkflows.DestinationHippoHap,
	vbexportworkflows.DestinationDubbing,
	vbexportworkflows.DestinationHyperdeck,
	vbexportworkflows.DestinationCasparCG,
}

func vbUIDestinations() []string {
	return lo.Map(vbUIDestinationOrder, func(d vbexportworkflows.Destination, _ int) string {
		return d.Value
	})
}

func allowedVBDestinations(perms *apiv1.Permissions) []*apiv1.VBDestination {
	out := []*apiv1.VBDestination{}
	for _, d := range vbUIDestinationOrder {
		if perms.CanVBExportTo(d.Value) {
			out = append(out, &apiv1.VBDestination{
				Id:          d.Value,
				Description: d.Description(),
			})
		}
	}
	return out
}

// assetTitle extracts the display title from asset metadata, preferring the
// original clip's title and falling back to the item title.
func assetTitle(meta *vsapi.MetadataResult) string {
	title := ""
	if oc, ok := meta.SplitByClips()[vsapi.OriginalClip]; ok {
		title = oc.Get(vscommon.FieldTitle, "")
	}
	if title == "" {
		title = meta.Get(vscommon.FieldTitle, "")
	}
	return title
}

func (e ExportAPI) GetExportConfig(ctx context.Context, req *connect.Request[apiv1.GetExportConfigRequest]) (*connect.Response[apiv1.GetExportConfigResponse], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	perms := PermissionsForEmail(email)
	if !perms.CanExport() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized to export"))
	}

	vxID := req.Msg.GetVXID()
	if vxID == "" {
		// Bulk export: no asset selected. Return only the asset-independent
		// config (the settings the user applies to every pasted VX-id), gated
		// by the dedicated bulk permission. No Vidispine lookups.
		if !perms.CanBulkExport() {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized for bulk export"))
		}
		return connect.NewResponse(&apiv1.GetExportConfigResponse{
			Destinations: allowedDestinations(perms),
			AudioSources: vidispine.ExportAudioSources.Values(),
			Languages:    exportLanguages(),
			Resolutions: lo.Map(defaultResolutions(), func(r vsapi.Resolution, _ int) *apiv1.ExportResolution {
				return &apiv1.ExportResolution{Width: int32(r.Width), Height: int32(r.Height)}
			}),
			Overlays: overlayFilenames(),
		}), nil
	}

	meta, err := e.vidispine.GetMetadata(vxID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	title := assetTitle(meta)

	resolutions, err := e.vidispine.GetResolutions(vxID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if len(resolutions) == 0 {
		resolutions = defaultResolutions()
	}

	resp := &apiv1.GetExportConfigResponse{
		VXID:                vxID,
		Title:               title,
		Destinations:        allowedDestinations(perms),
		AudioSources:        vidispine.ExportAudioSources.Values(),
		SelectedAudioSource: meta.Get(vscommon.FieldExportAudioSource, ""),
		Languages:           exportLanguages(),
		SelectedLanguages:   meta.GetArray(vscommon.FieldLangsToExport),
		Resolutions: lo.Map(resolutions, func(r vsapi.Resolution, _ int) *apiv1.ExportResolution {
			return &apiv1.ExportResolution{Width: int32(r.Width), Height: int32(r.Height)}
		}),
		Overlays:               overlayFilenames(),
		Subclips:               e.getSubclips(vxID),
		CanExportTimedMetadata: perms.CanExportTimedMetadata(),
	}

	return connect.NewResponse(resp), nil
}

// getSubclips replicates the trigger_ui subclip extraction. Subclips are
// best-effort: failures are logged and result in an empty list (the whole asset
// can still be exported).
func (e ExportAPI) getSubclips(vxID string) []*apiv1.ExportSubclip {
	exportData, err := vidispine.GetDataForExport(e.vidispine, vxID, nil, nil, "", false)
	if err != nil {
		log.L.Warn().Err(err).Str("vxid", vxID).Msg("could not get export data for subclips")
		return nil
	}

	rawChapters, err := vidispine.GetChapterMetaForClips(e.vidispine, exportData.Clips)
	if err != nil {
		log.L.Warn().Err(err).Str("vxid", vxID).Msg("could not get chapter metadata")
		return nil
	}

	type subclip struct {
		title string
		start float64
	}
	var tmp []subclip
	for _, c := range rawChapters {
		if len(c.Meta.Terse["title"]) == 0 {
			continue
		}
		ts, _ := vscommon.TCToSeconds(c.Meta.Terse["title"][0].Start)
		tmp = append(tmp, subclip{
			title: c.Meta.Get(vscommon.FieldTitle, ""),
			start: ts,
		})
	}
	sort.Slice(tmp, func(i, j int) bool { return tmp[i].start < tmp[j].start })

	return lo.Map(tmp, func(s subclip, _ int) *apiv1.ExportSubclip {
		return &apiv1.ExportSubclip{Title: s.title}
	})
}

func (e ExportAPI) StartExport(ctx context.Context, req *connect.Request[apiv1.StartExportRequest]) (*connect.Response[apiv1.StartExportResponse], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	perms := PermissionsForEmail(email)
	if !perms.CanExport() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized to export"))
	}

	msg := req.Msg
	vxID := msg.GetVXID()
	if vxID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing VXID"))
	}
	if len(msg.GetDestinations()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("no destinations selected"))
	}
	for _, d := range msg.GetDestinations() {
		if !perms.CanExportTo(d) {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized to export to %q", d))
		}
	}

	audioSource := msg.GetAudioSource()
	languages := msg.GetLanguages()

	// Persist the selections back to Vidispine so the form remembers them next
	// time (best-effort, mirrors trigger_ui).
	go e.persistSelections(vxID, audioSource, languages)

	var watermarkPath string
	if o := msg.GetOverlay(); o != "" && o != "None" {
		watermarkPath = filepath.Join(os.Getenv("OVERLAYS_DIR"), o)
	}

	resolutions := lo.Map(msg.GetResolutions(), func(r *apiv1.ExportResolutionSelection, _ int) utils.Resolution {
		return utils.Resolution{
			Width:  int(r.GetWidth()),
			Height: int(r.GetHeight()),
			IsFile: r.GetDownloadable(),
		}
	})

	params := exportworkflows.VXExportParams{
		VXID:          vxID,
		WithChapters:  msg.GetWithChapters(),
		IgnoreSilence: msg.GetIgnoreSilence(),
		SubsAllowAI:   msg.GetExportAiSubs(),
		WatermarkPath: watermarkPath,
		AudioSource:   audioSource,
		Destinations:  msg.GetDestinations(),
		Languages:     languages,
		Resolutions:   resolutions,
	}

	queue := getQueue()
	baseOpts := client.StartWorkflowOptions{TaskQueue: queue}
	if os.Getenv("DEBUG") == "" {
		baseOpts.SearchAttributes = map[string]any{
			"CustomStringField": vxID,
		}
	}

	// One workflow per subclip, or a single workflow for the whole asset.
	subclips := msg.GetSubclips()
	if len(subclips) == 0 {
		subclips = []string{""}
	}

	var workflowIDs []string
	for _, subclip := range subclips {
		params.Subclip = subclip
		opts := baseOpts
		opts.ID = uuid.NewString()
		run, err := e.temporalClient.ExecuteWorkflow(ctx, opts, exportworkflows.VXExport, params)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		workflowIDs = append(workflowIDs, run.GetID())
	}

	return connect.NewResponse(&apiv1.StartExportResponse{WorkflowIds: workflowIDs}), nil
}

func (e ExportAPI) persistSelections(vxID, audioSource string, languages []string) {
	if audioSource != "" {
		err := e.vidispine.SetItemMetadataField(vsapi.ItemMetadataFieldParams{
			ItemID: vxID,
			Key:    vscommon.FieldExportAudioSource.Value,
			Value:  audioSource,
		})
		if err != nil {
			log.L.Warn().Err(err).Str("vxid", vxID).Msg("could not persist export audio source")
		}
	}

	for i, lang := range languages {
		params := vsapi.ItemMetadataFieldParams{
			ItemID: vxID,
			Key:    vscommon.FieldLangsToExport.Value,
			Value:  lang,
		}
		var err error
		if i == 0 {
			err = e.vidispine.SetItemMetadataField(params)
		} else {
			err = e.vidispine.AddToItemMetadataField(params)
		}
		if err != nil {
			log.L.Warn().Err(err).Str("vxid", vxID).Msg("could not persist export language")
		}
	}
}

func (e ExportAPI) ExportTimedMetadata(ctx context.Context, req *connect.Request[apiv1.ExportTimedMetadataRequest]) (*connect.Response[apiv1.Void], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	perms := PermissionsForEmail(email)
	if !perms.CanExportTimedMetadata() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized to export timed metadata"))
	}

	vxID := req.Msg.GetVXID()
	if vxID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing VXID"))
	}

	queue := getQueue()
	opts := client.StartWorkflowOptions{
		TaskQueue: queue,
		ID:        uuid.NewString(),
	}
	if os.Getenv("DEBUG") == "" {
		opts.SearchAttributes = map[string]any{
			"CustomStringField": vxID,
		}
	}

	_, err := e.temporalClient.ExecuteWorkflow(ctx, opts, exportworkflows.ExportTimedMetadata, exportworkflows.ExportTimedMetadataParams{
		VXID: vxID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&apiv1.Void{}), nil
}

// ResolveAssets resolves a list of VX-ids to their titles for the bulk-export
// asset list. Unknown / inaccessible ids are returned with found=false rather
// than failing the whole request.
func (e ExportAPI) ResolveAssets(ctx context.Context, req *connect.Request[apiv1.ResolveAssetsRequest]) (*connect.Response[apiv1.ResolveAssetsResponse], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	perms := PermissionsForEmail(email)
	if !perms.CanBulkExport() && !perms.CanBulkVBExport() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized for bulk export"))
	}

	// Resolve metadata concurrently (bounded), keeping input order. Unknown ids
	// are recorded with Found=false rather than failing the whole request.
	vxIDs := req.Msg.GetVXIDs()
	results := make([]*apiv1.ResolvedAsset, len(vxIDs))
	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(8)
	for i, vxID := range vxIDs {
		if vxID == "" {
			continue
		}
		g.Go(func() error {
			meta, err := e.vidispine.GetMetadata(vxID)
			if err != nil {
				log.L.Warn().Err(err).Str("vxid", vxID).Msg("could not resolve asset for bulk export")
				results[i] = &apiv1.ResolvedAsset{VXID: vxID, Found: false}
				return nil
			}
			results[i] = &apiv1.ResolvedAsset{VXID: vxID, Title: assetTitle(meta), Found: true}
			return nil
		})
	}
	_ = g.Wait()

	// Drop the gaps left by empty ids.
	assets := lo.Filter(results, func(a *apiv1.ResolvedAsset, _ int) bool { return a != nil })

	return connect.NewResponse(&apiv1.ResolveAssetsResponse{Assets: assets}), nil
}

func (e ExportAPI) GetVBExportConfig(ctx context.Context, req *connect.Request[apiv1.GetVBExportConfigRequest]) (*connect.Response[apiv1.GetVBExportConfigResponse], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	perms := PermissionsForEmail(email)
	if !perms.CanVBExport() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized to export"))
	}

	vxID := req.Msg.GetVXID()
	if vxID == "" {
		// Bulk export: no asset selected. Return only the asset-independent
		// config, gated by the dedicated bulk permission. Subtitle shapes are
		// per-asset, so only "None" is offered for bulk burn-in.
		if !perms.CanBulkVBExport() {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized for bulk export"))
		}
		return connect.NewResponse(&apiv1.GetVBExportConfigResponse{
			Destinations:   allowedVBDestinations(perms),
			SubtitleShapes: []string{"None"},
			SubtitleStyles: dirFilenames(os.Getenv("SUBTITLE_STYLES_DIR")),
		}), nil
	}

	meta, err := e.vidispine.GetMetadata(vxID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	title := assetTitle(meta)

	// Subtitle shapes available for burn-in: tags like "sub_xxx_srt".
	subtitleShapes := []string{"None"}
	shapes, err := e.vidispine.GetShapes(vxID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for _, shape := range shapes.Shape {
		for _, tag := range shape.Tag {
			if strings.HasPrefix(tag, "sub_") && strings.HasSuffix(tag, "_srt") {
				subtitleShapes = append(subtitleShapes, tag)
			}
		}
	}

	resp := &apiv1.GetVBExportConfigResponse{
		VXID:           vxID,
		Title:          title,
		Destinations:   allowedVBDestinations(perms),
		SubtitleShapes: subtitleShapes,
		SubtitleStyles: dirFilenames(os.Getenv("SUBTITLE_STYLES_DIR")),
	}

	return connect.NewResponse(resp), nil
}

func (e ExportAPI) StartVBExport(ctx context.Context, req *connect.Request[apiv1.StartVBExportRequest]) (*connect.Response[apiv1.StartVBExportResponse], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	perms := PermissionsForEmail(email)
	if !perms.CanVBExport() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized to export"))
	}

	msg := req.Msg
	vxID := msg.GetVXID()
	if vxID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing VXID"))
	}
	if len(msg.GetDestinations()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("no destinations selected"))
	}
	for _, d := range msg.GetDestinations() {
		if !perms.CanVBExportTo(d) {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized to export to %q", d))
		}
	}

	// "None" / empty means no subtitle burn-in.
	subtitleShape := msg.GetSubtitleShape()
	if subtitleShape == "None" {
		subtitleShape = ""
	}

	params := vbexportworkflows.VBExportParams{
		VXID:             vxID,
		Destinations:     msg.GetDestinations(),
		SubtitleShapeTag: subtitleShape,
		SubtitleStyle:    msg.GetSubtitleStyle(),
	}

	opts := client.StartWorkflowOptions{
		TaskQueue: getQueue(),
		ID:        uuid.NewString(),
	}
	if os.Getenv("DEBUG") == "" {
		opts.SearchAttributes = map[string]any{
			"CustomStringField": vxID,
		}
	}

	run, err := e.temporalClient.ExecuteWorkflow(ctx, opts, vbexportworkflows.VBExport, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&apiv1.StartVBExportResponse{WorkflowId: run.GetID()}), nil
}

// GetExportDestinations returns the full set of selectable destinations,
// straight from the bcc-media-flows enums, so the admin UI is always in sync
// with what the backend actually accepts.
func (e ExportAPI) GetExportDestinations(ctx context.Context, req *connect.Request[apiv1.Void]) (*connect.Response[apiv1.ExportDestinationsResponse], error) {
	if !IsAdmin(req) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized"))
	}
	return connect.NewResponse(&apiv1.ExportDestinationsResponse{
		Vx: vxUIDestinations(),
		Vb: vbUIDestinations(),
	}), nil
}
