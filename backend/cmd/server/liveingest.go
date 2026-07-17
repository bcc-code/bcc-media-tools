package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"
	"path/filepath"

	"connectrpc.com/connect"
	ingestworkflows "github.com/bcc-code/bcc-media-flows/workflows/ingest"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

// liveIngestWorkflowType is the registered workflow type of the live ingest
// (see ingestworkflows.Incremental). We list running workflows of this type
// rather than hardcoding the fixed "LIVE-INGEST" workflow ID.
const liveIngestWorkflowType = "Incremental"

type LiveIngestAPI struct {
	temporalClient client.Client
}

func NewLiveIngestAPI(temporalClient client.Client) *LiveIngestAPI {
	return &LiveIngestAPI{
		temporalClient: temporalClient,
	}
}

// FinishLiveIngest finds the currently-running live ingest workflow(s) and sends
// the "transfer_finished" signal so the ingest stops copying and completes.
func (l LiveIngestAPI) FinishLiveIngest(ctx context.Context, req *connect.Request[apiv1.FinishLiveIngestRequest]) (*connect.Response[apiv1.FinishLiveIngestResponse], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}

	if !PermissionsForEmail(email).CanLiveIngest() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized"))
	}

	list, err := l.temporalClient.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Query: fmt.Sprintf("WorkflowType = '%s' AND ExecutionStatus = 'Running'", liveIngestWorkflowType),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if len(list.GetExecutions()) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no running live ingest found"))
	}

	finished := make([]*apiv1.FinishedIngest, 0, len(list.GetExecutions()))
	for _, exec := range list.GetExecutions() {
		wfID := exec.GetExecution().GetWorkflowId()
		runID := exec.GetExecution().GetRunId()

		filename, err := l.ingestFilename(ctx, wfID, runID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("resolving filename for %s: %w", wfID, err))
		}

		// The workflow waits for this signal to stop copying the growing source
		// file; its payload is the base filename, which the workflow matches
		// against the file it is ingesting.
		if err := l.temporalClient.SignalWorkflow(ctx, wfID, runID, ingestworkflows.FileTransferredSignal, filename); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("signaling %s: %w", wfID, err))
		}

		finished = append(finished, &apiv1.FinishedIngest{
			WorkflowId: wfID,
			Filename:   filename,
		})
	}

	return connect.NewResponse(&apiv1.FinishLiveIngestResponse{
		Finished: finished,
	}), nil
}

// ingestFilename reads the workflow's start event input to recover the base
// filename it is ingesting, which is what the transfer_finished signal must carry.
func (l LiveIngestAPI) ingestFilename(ctx context.Context, wfID, runID string) (string, error) {
	iter := l.temporalClient.GetWorkflowHistory(ctx, wfID, runID, false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return "", err
		}

		started := event.GetWorkflowExecutionStartedEventAttributes()
		if started == nil {
			continue
		}

		var params ingestworkflows.IncrementalParams
		if err := converter.GetDefaultDataConverter().FromPayloads(started.GetInput(), &params); err != nil {
			return "", err
		}

		return filepath.Base(params.Path), nil
	}

	return "", fmt.Errorf("no start event found")
}
