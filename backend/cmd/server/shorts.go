package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bcc-code/bcc-media-flows/services/cantemo"
	exportworkflows "github.com/bcc-code/bcc-media-flows/workflows/export"
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

func (s ShortsAPI) SubmitShort(ctx context.Context, req *connect.Request[apiv1.SubmitShortRequest]) (*connect.Response[apiv1.Void], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	if !PermissionsForEmail(email).CanShorts() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not enough permissions to create shorts"))
	}

	fmt.Printf("Submitted short with VXID %s for generation", req.Msg.GetVXID())

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
