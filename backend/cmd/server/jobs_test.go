package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_buildJobsQuery(t *testing.T) {
	t.Run("no filters restricts to surfaced types, no ORDER BY", func(t *testing.T) {
		q := buildJobsQuery(nil, nil)
		assert.Contains(t, q, "WorkflowType IN (")
		assert.Contains(t, q, "'VXExport'")
		assert.Contains(t, q, "'VBExport'")
		assert.NotContains(t, q, "ExecutionStatus")
		// The reference lives in the Memo and is filtered client-side.
		assert.NotContains(t, q, "CustomStringField")
		// Standard visibility rejects ORDER BY; default ordering is newest-first.
		assert.NotContains(t, q, "ORDER BY")
	})

	t.Run("tool filter narrows the workflow types", func(t *testing.T) {
		q := buildJobsQuery(nil, []string{"shorts"})
		assert.Contains(t, q, "WorkflowType IN ('GenerateShort')")
	})

	t.Run("subtitles maps to both import workflow types", func(t *testing.T) {
		q := buildJobsQuery(nil, []string{"subtitles"})
		assert.Contains(t, q, "'ImportSubtitles'")
		assert.Contains(t, q, "'ImportSubtitlesFromSubtrans'")
	})

	t.Run("status filter uses Temporal status names", func(t *testing.T) {
		q := buildJobsQuery([]string{"running", "failed"}, nil)
		assert.Contains(t, q, "ExecutionStatus IN ('Running', 'Failed')")
	})

	t.Run("unknown status is dropped", func(t *testing.T) {
		q := buildJobsQuery([]string{"bogus"}, nil)
		assert.NotContains(t, q, "ExecutionStatus")
	})

	t.Run("unknown tool falls back to all surfaced types", func(t *testing.T) {
		q := buildJobsQuery(nil, []string{"nope"})
		assert.Contains(t, q, "'VXExport'")
		assert.Contains(t, q, "'GenerateShort'")
	})
}

func Test_toolForType(t *testing.T) {
	assert.Equal(t, "export", toolForType("VXExport"))
	assert.Equal(t, "subtitles", toolForType("ImportSubtitlesFromSubtrans"))
	assert.Equal(t, "other", toolForType("SomeChildWorkflow"))
}

func Test_temporalUIBaseURL(t *testing.T) {
	// Explicit URL always wins.
	assert.Equal(t, "https://cloud.temporal.io",
		temporalUIBaseURL("https://cloud.temporal.io", "ns.acct.tmprl.cloud:7233"))
	// A scheme-less explicit value is normalized to http://.
	assert.Equal(t, "http://localhost:8233", temporalUIBaseURL("localhost:8233", ""))
	// Local start-dev: derive the Web UI port from the gRPC host.
	assert.Equal(t, "http://localhost:8233", temporalUIBaseURL("", "localhost:7233"))
	assert.Equal(t, "http://127.0.0.1:8233", temporalUIBaseURL("", "127.0.0.1:7233"))
	// Remote host with no explicit URL: hidden (UI address is unknown).
	assert.Equal(t, "", temporalUIBaseURL("", "ns.acct.tmprl.cloud:7233"))
	assert.Equal(t, "", temporalUIBaseURL("", ""))
}

func Test_workflowMemo(t *testing.T) {
	assert.Nil(t, workflowMemo("", ""))
	assert.Equal(t, map[string]any{"startedBy": "a@b.no"}, workflowMemo("a@b.no", ""))
	assert.Equal(t, map[string]any{"reference": "VX-1"}, workflowMemo("", "VX-1"))
	assert.Equal(t,
		map[string]any{"startedBy": "a@b.no", "reference": "VX-1"},
		workflowMemo("a@b.no", "VX-1"))
}
