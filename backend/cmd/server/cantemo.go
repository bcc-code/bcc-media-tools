package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"

	"connectrpc.com/connect"
	miscworkflows "github.com/bcc-code/bcc-media-flows/workflows/misc"
	"go.temporal.io/sdk/client"
)

type CantemoAPI struct {
	temporalClient client.Client
}

func NewCantemoAPI(temporalClient client.Client) *CantemoAPI {
	return &CantemoAPI{
		temporalClient: temporalClient,
	}
}

// TriggerCantemoAction starts one of the fire-and-forget workflows exposed by the
// Cantemo action panel. Each action is gated by its own permission.
func (c CantemoAPI) TriggerCantemoAction(ctx context.Context, req *connect.Request[apiv1.TriggerCantemoActionRequest]) (*connect.Response[apiv1.Void], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}

	vxID := req.Msg.GetVXID()
	if vxID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing VXID"))
	}

	perms := PermissionsForEmail(email)

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: getQueue(),
	}

	var (
		workflow any
		params   any
	)

	switch req.Msg.GetAction() {
	case apiv1.CantemoAction_CANTEMO_ACTION_PREVIEW:
		if !perms.CanCantemoPreview() {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized"))
		}
		workflow = miscworkflows.TranscodePreviewVX
		params = miscworkflows.TranscodePreviewVXInput{VXID: vxID}
	case apiv1.CantemoAction_CANTEMO_ACTION_TRANSCRIBE:
		if !perms.CanCantemoTranscribe() {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized"))
		}
		workflow = miscworkflows.TranscribeVX
		params = miscworkflows.TranscribeVXInput{VXID: vxID, Language: "no"}
	case apiv1.CantemoAction_CANTEMO_ACTION_SUBTITLE_FROM_SUBTRANS:
		if !perms.CanCantemoSubtitles() {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized"))
		}
		workflow = miscworkflows.ImportSubtitlesFromSubtrans
		params = miscworkflows.ImportSubtitlesFromSubtransInput{VXID: vxID}
	case apiv1.CantemoAction_CANTEMO_ACTION_UPDATE_RELATIONS:
		if !perms.CanCantemoRelations() {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized"))
		}
		workflow = miscworkflows.UpdateAssetRelations
		params = miscworkflows.UpdateAssetRelationsParams{AssetID: vxID}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown action %v", req.Msg.GetAction()))
	}

	_, err := c.temporalClient.ExecuteWorkflow(ctx, workflowOptions, workflow, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&apiv1.Void{}), nil
}
