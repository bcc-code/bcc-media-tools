package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"

	"connectrpc.com/connect"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

// jobToolByType maps a Temporal workflow type name to the user-facing tool key
// surfaced in the jobs dashboard. Only these top-level types are listed; child
// workflows and activities are filtered out by matching against this set.
var jobToolByType = map[string]string{
	"VXExport":                    "export",
	"VBExport":                    "vb_export",
	"ExportTimedMetadata":         "timed_metadata",
	"GenerateShort":               "shorts",
	"BmmIngestUpload":             "bmm_upload",
	"ImportSubtitles":             "subtitles",
	"ImportSubtitlesFromSubtrans": "subtitles",
	"TranscribeVX":                "transcribe",
}

// temporalStatusName maps our lowercase status key to the name Temporal's
// visibility query language expects for ExecutionStatus.
var temporalStatusName = map[string]string{
	"running":          "Running",
	"completed":        "Completed",
	"failed":           "Failed",
	"canceled":         "Canceled",
	"terminated":       "Terminated",
	"timed_out":        "TimedOut",
	"continued_as_new": "ContinuedAsNew",
}

type JobsAPI struct {
	temporalClient client.Client
	namespace      string
	// uiBaseURL is the Temporal Web UI base (e.g. http://localhost:8233). Empty
	// disables the "see in Temporal" deep link.
	uiBaseURL string
}

func NewJobsAPI(temporalClient client.Client, namespace, uiBaseURL string) *JobsAPI {
	return &JobsAPI{temporalClient: temporalClient, namespace: namespace, uiBaseURL: uiBaseURL}
}

// temporalUIBaseURL resolves the Temporal Web UI base. An explicit uiURL
// (TEMPORAL_UI_URL) always wins. Otherwise, for a local start-dev host
// (localhost/127.0.0.1) the UI is derived by swapping the gRPC port for the Web
// UI port (8233). Remote hosts return "" — their UI address is unrelated to the
// gRPC host:port — so the deep link stays hidden until TEMPORAL_UI_URL is set.
func temporalUIBaseURL(uiURL, hostPort string) string {
	if uiURL != "" {
		// Tolerate a scheme-less value (e.g. "localhost:8233") so the link
		// doesn't resolve as a relative path in the browser.
		if !strings.Contains(uiURL, "://") {
			uiURL = "http://" + uiURL
		}
		return uiURL
	}
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return ""
	}
	if host == "localhost" || host == "127.0.0.1" {
		return "http://" + host + ":8233"
	}
	return ""
}

// temporalURL builds the Temporal Web UI history link for a workflow run, or ""
// when no UI base URL is configured.
func (j JobsAPI) temporalURL(wfID, runID string) string {
	if j.uiBaseURL == "" {
		return ""
	}
	ns := j.namespace
	if ns == "" {
		ns = "default"
	}
	return fmt.Sprintf("%s/namespaces/%s/workflows/%s/%s/history",
		strings.TrimRight(j.uiBaseURL, "/"),
		url.PathEscape(ns),
		url.PathEscape(wfID),
		url.PathEscape(runID),
	)
}

// workflowMemo builds the Memo attached to every workflow this app starts, so
// the jobs dashboard can show who launched it and the asset it relates to.
// Memo needs no Temporal registration (unlike a search attribute), so it works
// on any namespace. Empty fields are omitted; an all-empty memo returns nil.
func workflowMemo(email, reference string) map[string]any {
	memo := map[string]any{}
	if email != "" {
		memo["startedBy"] = email
	}
	if reference != "" {
		memo["reference"] = reference
	}
	if len(memo) == 0 {
		return nil
	}
	return memo
}

func (j JobsAPI) ListJobs(ctx context.Context, req *connect.Request[apiv1.ListJobsRequest]) (*connect.Response[apiv1.ListJobsResponse], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	perms := PermissionsForEmail(email)
	if !perms.CanViewJobs() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized"))
	}

	pageSize := req.Msg.GetPageSize()
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 25
	}

	query := buildJobsQuery(req.Msg.GetStatuses(), req.Msg.GetTools())

	res, err := j.temporalClient.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		PageSize:      pageSize,
		NextPageToken: req.Msg.GetPageToken(),
		Query:         query,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	jobs := make([]*apiv1.Job, 0, len(res.GetExecutions()))
	for _, e := range res.GetExecutions() {
		job := executionToJob(e)
		redactStartedBy(job, email, perms.Admin)
		jobs = append(jobs, job)
	}

	return connect.NewResponse(&apiv1.ListJobsResponse{
		Jobs:          jobs,
		NextPageToken: res.GetNextPageToken(),
	}), nil
}

func (j JobsAPI) GetJob(ctx context.Context, req *connect.Request[apiv1.GetJobRequest]) (*connect.Response[apiv1.GetJobResponse], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}
	perms := PermissionsForEmail(email)
	if !perms.CanViewJobs() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized"))
	}
	wfID := req.Msg.GetWorkflowId()
	if wfID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing workflow_id"))
	}

	desc, err := j.temporalClient.DescribeWorkflowExecution(ctx, wfID, req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	job := executionToJob(desc.GetWorkflowExecutionInfo())
	redactStartedBy(job, email, perms.Admin)

	var errMsg string
	if job.GetStatus() == "failed" {
		// A closed workflow's result is available immediately; for a failed one
		// that resolves to the failure, which carries the message we want.
		run := j.temporalClient.GetWorkflow(ctx, job.GetWorkflowId(), job.GetRunId())
		if runErr := run.Get(ctx, nil); runErr != nil {
			errMsg = runErr.Error()
		}
	}

	return connect.NewResponse(&apiv1.GetJobResponse{
		Job:          job,
		ErrorMessage: errMsg,
		TemporalUrl:  j.temporalURL(job.GetWorkflowId(), job.GetRunId()),
	}), nil
}

// buildJobsQuery assembles a Temporal visibility query from the filters,
// restricting to the surfaced workflow types plus an optional status filter.
// It deliberately uses only always-present system attributes (WorkflowType,
// ExecutionStatus): the asset reference lives in the Memo (not queryable) and
// is filtered client-side, so this query works on any visibility backend.
// No ORDER BY is added — standard (non-Elasticsearch) visibility rejects it and
// already returns results newest-first by default.
func buildJobsQuery(statuses, tools []string) string {
	clauses := []string{inClause("WorkflowType", jobTypesForTools(tools))}

	var statusNames []string
	for _, s := range statuses {
		if name, ok := temporalStatusName[s]; ok {
			statusNames = append(statusNames, name)
		}
	}
	if len(statusNames) > 0 {
		clauses = append(clauses, inClause("ExecutionStatus", statusNames))
	}

	return strings.Join(clauses, " AND ")
}

// jobTypesForTools returns the workflow types to query: the intersection of the
// requested tool keys with the surfaced set, or all surfaced types when no (or
// only unknown) tools are given.
func jobTypesForTools(tools []string) []string {
	all := sortedSurfacedTypes()
	if len(tools) == 0 {
		return all
	}
	want := map[string]bool{}
	for _, t := range tools {
		want[t] = true
	}
	var out []string
	for _, wtype := range all {
		if want[jobToolByType[wtype]] {
			out = append(out, wtype)
		}
	}
	if len(out) == 0 {
		return all
	}
	return out
}

func sortedSurfacedTypes() []string {
	types := make([]string, 0, len(jobToolByType))
	for t := range jobToolByType {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func inClause(field string, values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = "'" + v + "'"
	}
	return fmt.Sprintf("%s IN (%s)", field, strings.Join(quoted, ", "))
}

func executionToJob(e *workflowpb.WorkflowExecutionInfo) *apiv1.Job {
	typeName := e.GetType().GetName()
	// Reference comes from the Memo (set on every namespace); older/prod jobs
	// may only have it as the CustomStringField search attribute, so fall back.
	reference := decodePayloadString(e.GetMemo().GetFields(), "reference")
	if reference == "" {
		reference = decodePayloadString(e.GetSearchAttributes().GetIndexedFields(), "CustomStringField")
	}

	job := &apiv1.Job{
		WorkflowId:   e.GetExecution().GetWorkflowId(),
		RunId:        e.GetExecution().GetRunId(),
		Tool:         toolForType(typeName),
		WorkflowType: typeName,
		Status:       jobStatus(e.GetStatus()),
		Reference:    reference,
		StartedBy:    decodePayloadString(e.GetMemo().GetFields(), "startedBy"),
		StartedAt:    e.GetStartTime(),
	}
	if e.GetCloseTime() != nil {
		job.ClosedAt = e.GetCloseTime()
	}
	return job
}

// redactStartedBy hides who started a job from non-admins, except for their own
// jobs — so colleagues' email addresses aren't exposed to every tool user.
func redactStartedBy(job *apiv1.Job, viewerEmail string, isAdmin bool) {
	if isAdmin || job.GetStartedBy() == viewerEmail {
		return
	}
	job.StartedBy = ""
}

func toolForType(typeName string) string {
	if tool, ok := jobToolByType[typeName]; ok {
		return tool
	}
	return "other"
}

func jobStatus(s enumspb.WorkflowExecutionStatus) string {
	switch s {
	case enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:
		return "running"
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return "completed"
	case enumspb.WORKFLOW_EXECUTION_STATUS_FAILED:
		return "failed"
	case enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED:
		return "canceled"
	case enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		return "terminated"
	case enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return "timed_out"
	case enumspb.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		return "continued_as_new"
	}
	return "unknown"
}

// decodePayloadString reads a single string value from a payload map (used for
// both memo fields and indexed search attributes, which share the encoding).
func decodePayloadString(fields map[string]*commonpb.Payload, key string) string {
	p, ok := fields[key]
	if !ok {
		return ""
	}
	var s string
	if err := converter.GetDefaultDataConverter().FromPayload(p, &s); err != nil {
		return ""
	}
	return s
}
