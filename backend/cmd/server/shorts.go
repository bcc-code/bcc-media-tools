package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/bcc-code/bcc-media-flows/services/cantemo"
	exportworkflows "github.com/bcc-code/bcc-media-flows/workflows/export"
	"github.com/bcc-code/mediabank-bridge/log"
	"go.temporal.io/sdk/client"
)

type ShortsAPI struct {
	temporalClient client.Client
	cantemoClient  *cantemo.Client
}

func NewShortsAPI(temporalClient client.Client, cantemoClient *cantemo.Client) *ShortsAPI {
	return &ShortsAPI{
		temporalClient: temporalClient,
		cantemoClient:  cantemoClient,
	}
}

// GetShortsPreview resolves the source-video preview URL for the shorts editor.
// Access is gated by shorts permission alone.
func (s ShortsAPI) GetShortsPreview(ctx context.Context, req *connect.Request[apiv1.GetPreviewRequest]) (*connect.Response[apiv1.Preview], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	if !PermissionsForEmail(email).CanShorts() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not enough permissions to create shorts"))
	}

	url, err := s.cantemoClient.GetPreviewUrl(req.Msg.VXID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&apiv1.Preview{Url: url}), nil
}

// GetShortsTranscript returns the automatic transcript for an asset so the
// editor can offer the text as a way of finding a clip range.
//
// Gated on shorts permission alone, deliberately: the transcript is derived
// from the audio of a video that GetShortsPreview already streams to the same
// user, so it exposes nothing they cannot hear by pressing play. It is
// read-only — editing transcripts stays behind the transcription permission.
func (s ShortsAPI) GetShortsTranscript(ctx context.Context, req *connect.Request[apiv1.GetTranscriptionReqest]) (*connect.Response[apiv1.Transcription], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	if !PermissionsForEmail(email).CanShorts() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not enough permissions to create shorts"))
	}

	// An asset with no transcription_json yields an empty document rather than
	// an error, which the editor renders as "no transcript".
	transcription, err := s.cantemoClient.GetTranscriptionJSON(req.Msg.VXID)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(mapTranscriptionToAPI(transcription)), nil
}

// validateShortRange mirrors the range checks in the GenerateShort workflow so
// an unusable range is refused at the API boundary rather than deep in Temporal.
func validateShortRange(msg *apiv1.SubmitShortRequest) error {
	if strings.TrimSpace(msg.GetVXID()) == "" {
		return fmt.Errorf("VXID is required")
	}
	if msg.GetInSeconds() < 0 {
		return fmt.Errorf("start must be at or after 0, got %v", msg.GetInSeconds())
	}
	if msg.GetOutSeconds() <= msg.GetInSeconds() {
		return fmt.Errorf("end (%v) must be after start (%v)", msg.GetOutSeconds(), msg.GetInSeconds())
	}
	return nil
}

// SubmitShorts starts one GenerateShort workflow per clip.
//
// Clips are independent, so one bad or unstartable clip reports itself and the
// rest still run: losing four good cuts because the fifth was malformed would
// mean redoing the whole session.
func (s ShortsAPI) SubmitShorts(ctx context.Context, req *connect.Request[apiv1.SubmitShortsRequest]) (*connect.Response[apiv1.SubmitShortsResponse], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	if !PermissionsForEmail(email).CanShorts() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not enough permissions to create shorts"))
	}
	if len(req.Msg.GetClips()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("no clips to submit"))
	}

	results := make([]*apiv1.SubmitShortResult, 0, len(req.Msg.GetClips()))
	for _, clip := range req.Msg.GetClips() {
		single := &apiv1.SubmitShortRequest{
			VXID:       req.Msg.GetVXID(),
			InSeconds:  clip.GetInSeconds(),
			OutSeconds: clip.GetOutSeconds(),
		}

		if err := validateShortRange(single); err != nil {
			results = append(results, &apiv1.SubmitShortResult{Error: err.Error()})
			continue
		}

		run, err := s.startShort(ctx, single)
		if err != nil {
			log.L.Error().Err(err).
				Str("vxid", single.GetVXID()).
				Float64("in", single.GetInSeconds()).
				Float64("out", single.GetOutSeconds()).
				Msg("shorts: could not start generation")
			results = append(results, &apiv1.SubmitShortResult{Error: err.Error()})
			continue
		}

		results = append(results, &apiv1.SubmitShortResult{WorkflowId: run.GetID()})
	}

	return connect.NewResponse(&apiv1.SubmitShortsResponse{Results: results}), nil
}

// startShort launches the generation workflow for one already-validated clip.
func (s ShortsAPI) startShort(ctx context.Context, msg *apiv1.SubmitShortRequest) (client.WorkflowRun, error) {
	log.L.Info().
		Str("vxid", msg.GetVXID()).
		Float64("in", msg.GetInSeconds()).
		Float64("out", msg.GetOutSeconds()).
		Msg("shorts: submitting for generation")

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: getQueue(),
	}

	return s.temporalClient.ExecuteWorkflow(ctx, workflowOptions, exportworkflows.GenerateShort, exportworkflows.GenerateShortDataParams{
		VXID:          msg.GetVXID(),
		InSeconds:     msg.GetInSeconds(),
		OutSeconds:    msg.GetOutSeconds(),
		OutputDirPath: "/mnt/isilon/Input/shorts",
		ModelSize:     "n",
		DebugMode:     false,
	})
}

func (s ShortsAPI) SubmitShort(ctx context.Context, req *connect.Request[apiv1.SubmitShortRequest]) (*connect.Response[apiv1.Void], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	if !PermissionsForEmail(email).CanShorts() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not enough permissions to create shorts"))
	}

	if err := validateShortRange(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if _, err := s.startShort(ctx, req.Msg); err != nil {
		return nil, err
	}

	return connect.NewResponse(&apiv1.Void{}), nil
}
