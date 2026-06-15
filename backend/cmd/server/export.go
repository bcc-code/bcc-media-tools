package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"connectrpc.com/connect"
	bccmflows "github.com/bcc-code/bcc-media-flows"
	"github.com/bcc-code/bcc-media-flows/services/vidispine"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vsapi"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vscommon"
	"github.com/bcc-code/bcc-media-flows/utils"
	exportworkflows "github.com/bcc-code/bcc-media-flows/workflows/export"
	"github.com/bcc-code/mediabank-bridge/log"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.temporal.io/sdk/client"
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

// allowedDestinations returns the canonical export destinations the given user
// is permitted to export to.
func allowedDestinations(perms *apiv1.Permissions) []string {
	return lo.Filter(exportworkflows.AssetExportDestinations.Values(), func(d string, _ int) bool {
		return perms.CanExportTo(d)
	})
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
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing VXID"))
	}

	meta, err := e.vidispine.GetMetadata(vxID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	clips := meta.SplitByClips()
	title := ""
	if oc, ok := clips[vsapi.OriginalClip]; ok {
		title = oc.Get(vscommon.FieldTitle, "")
	}
	if title == "" {
		title = meta.Get(vscommon.FieldTitle, "")
	}

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
		Overlays: overlayFilenames(),
		Subclips: e.getSubclips(vxID),
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
	if !perms.CanExport() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized to export"))
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
