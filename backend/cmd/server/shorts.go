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

	log.L.Info().
		Str("vxid", req.Msg.GetVXID()).
		Float64("in", req.Msg.GetInSeconds()).
		Float64("out", req.Msg.GetOutSeconds()).
		Msg("shorts: submitting for generation")

	// Trigger flow
	queue := getQueue()
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: queue,
	}

	_, err := s.temporalClient.ExecuteWorkflow(ctx, workflowOptions, exportworkflows.GenerateShort, exportworkflows.GenerateShortDataParams{
		VXID:          req.Msg.VXID,
		InSeconds:     req.Msg.InSeconds,
		OutSeconds:    req.Msg.OutSeconds,
		OutputDirPath: "/mnt/isilon/Input/shorts",
		ModelSize:     "n",
		DebugMode:     false,
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&apiv1.Void{}), nil
}
