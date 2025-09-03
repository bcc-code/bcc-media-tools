package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"

	"connectrpc.com/connect"
	exportworkflows "github.com/bcc-code/bcc-media-flows/workflows/export"
	"go.temporal.io/sdk/client"
)

type ShortsAPI struct {
	temporalClient client.Client
}

func NewShortsAPI(temporalClient client.Client) *ShortsAPI {
	return &ShortsAPI{
		temporalClient: temporalClient,
	}
}

func (s ShortsAPI) SubmitShort(ctx context.Context, req *connect.Request[apiv1.SubmitShortRequest]) (*connect.Response[apiv1.Void], error) {
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
