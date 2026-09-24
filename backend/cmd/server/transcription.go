package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bcc-code/bcc-media-flows/services/cantemo"
	ingestworkflows "github.com/bcc-code/bcc-media-flows/workflows/ingest"
	"github.com/samber/lo"
	"go.temporal.io/sdk/client"
)

type TranscriptionAPI struct {
	cantemoClient  *cantemo.Client
	temporalClient client.Client
}

func NewTranscriptionAPI(baseURL, token string, temporalClient client.Client) *TranscriptionAPI {
	return &TranscriptionAPI{
		cantemoClient:  cantemo.NewClient(baseURL, token),
		temporalClient: temporalClient,
	}
}

// GetTranscriptionPreview resolves the source-video preview URL for the
// transcription editor. Admins see everything; a volunteer may only preview a
// video while it is shared for transcription editing.
func (t TranscriptionAPI) GetTranscriptionPreview(ctx context.Context, req *connect.Request[apiv1.GetPreviewRequest]) (*connect.Response[apiv1.Preview], error) {
	if err := t.authorize(getEmail(req), req.Msg.VXID); err != nil {
		return nil, err
	}

	url, err := t.cantemoClient.GetPreviewUrl(req.Msg.VXID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&apiv1.Preview{Url: url}), nil
}

// transcriptionAccess decides who may read or write a transcription.
//
// sharedForEditing is a func so the ACL lookup is skipped for admins, who do not
// need it, and so the policy can be exercised without Cantemo.
func transcriptionAccess(email string, perms *apiv1.Permissions, sharedForEditing func() bool) error {
	if email == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}

	tp := perms.GetTranscription()
	if !tp.GetAdmin() && !tp.GetMediabanken() {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not enough permissions for transcription"))
	}

	if perms.GetAdmin() || tp.GetAdmin() || sharedForEditing() {
		return nil
	}

	return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("asset is not shared for transcription editing"))
}

func (t TranscriptionAPI) authorize(email, vxid string) error {
	return transcriptionAccess(email, PermissionsForEmail(email), func() bool {
		return t.inTranscriptionCollection(vxid)
	})
}

// inTranscriptionCollection reports whether the asset is currently shared for
// volunteer transcription editing — i.e. it inherits ACLs from the
// _AccessibleByTools collection (VX-2677). Videos are added there so volunteers
// can fix auto-transcriptions without admin access, and removed on submit so a
// volunteer can no longer reach that item.
func (t TranscriptionAPI) inTranscriptionCollection(vxid string) bool {
	m, err := t.cantemoClient.GetACL(vxid)
	if err != nil || m == nil {
		return false
	}
	for _, acl := range m.ACLs {
		if acl.InheritedFrom != nil && acl.InheritedFrom.ID == "VX-2677" {
			return true
		}
	}
	return false
}

func (t TranscriptionAPI) GetTranscription(ctx context.Context, req *connect.Request[apiv1.GetTranscriptionReqest]) (*connect.Response[apiv1.Transcription], error) {
	if err := t.authorize(getEmail(req), req.Msg.VXID); err != nil {
		return nil, err
	}

	transcription, err := t.cantemoClient.GetTranscriptionJSON(req.Msg.VXID)

	if err != nil {
		return nil, err
	}

	tr := apiv1.Transcription{
		Text:     transcription.Text,
		Segments: make([]*apiv1.Segments, len(transcription.Segments)),
	}

	for i, s := range transcription.Segments {
		tr.Segments[i] = &apiv1.Segments{
			Start:            s.Start,
			End:              s.End,
			Text:             s.Text,
			Id:               float64(s.ID),
			Seek:             int32(s.Seek),
			Tokens:           lo.Map(s.Tokens, func(_ int, t int) int32 { return int32(t) }),
			Temperature:      s.Temperature,
			AvgLogprob:       s.AvgLogprob,
			CompressionRatio: s.CompressionRatio,
			NoSpeechProb:     s.NoSpeechProb,
			Confidence:       s.Confidence,
			Words:            make([]*apiv1.Words, len(s.Words)),
		}

		for j, w := range s.Words {
			tr.Segments[i].Words[j] = &apiv1.Words{
				Start:      w.Start,
				End:        w.End,
				Text:       w.Text,
				Confidence: w.Confidence,
			}
		}
	}

	return connect.NewResponse(&tr), nil
}

func (t TranscriptionAPI) SubmitTranscription(ctx context.Context, req *connect.Request[apiv1.SubmitTranscriptionRequest]) (*connect.Response[apiv1.Void], error) {
	if err := t.authorize(getEmail(req), req.Msg.VXID); err != nil {
		return nil, err
	}

	// Trigger flow
	queue := getQueue()
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: queue,
	}

	_, err := t.temporalClient.ExecuteWorkflow(ctx, workflowOptions, ingestworkflows.ImportSubtitles, ingestworkflows.ImportSubtitlesInput{
		VXID:      req.Msg.VXID,
		Subtitles: mapApiTranscriptionToModel(req.Msg.Transcription),
		Language:  "no", // Hardcoded to norwegian for now
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&apiv1.Void{}), nil
}

// mapApiTranscriptionToModel maps apiv1.Transcription (protobuf) to ingestworkflows.Transcription (model)
func mapApiTranscriptionToModel(api *apiv1.Transcription) ingestworkflows.Transcription {
	if api == nil {
		return ingestworkflows.Transcription{}
	}
	segments := make([]ingestworkflows.Segment, len(api.Segments))
	for i, s := range api.Segments {
		segments[i] = ingestworkflows.Segment{
			Start:            s.Start,
			End:              s.End,
			Text:             s.Text,
			ID:               i,
			Seek:             int(s.Seek),
			Tokens:           toIntSlice(s.Tokens),
			Temperature:      s.Temperature,
			AvgLogprob:       s.AvgLogprob,
			CompressionRatio: s.CompressionRatio,
			NoSpeechProb:     s.NoSpeechProb,
			Confidence:       s.Confidence,
			Words:            mapApiWordsToModel(s.Words),
		}
	}
	return ingestworkflows.Transcription{
		Text:     api.Text,
		Segments: segments,
	}
}

func mapApiWordsToModel(words []*apiv1.Words) []ingestworkflows.Word {
	if words == nil {
		return nil
	}
	result := make([]ingestworkflows.Word, len(words))
	for i, w := range words {
		result[i] = ingestworkflows.Word{
			Start:      w.Start,
			End:        w.End,
			Text:       w.Text,
			Confidence: w.Confidence,
		}
	}
	return result
}

func toIntSlice(tokens []int32) []int {
	if tokens == nil {
		return nil
	}
	result := make([]int, len(tokens))
	for i, t := range tokens {
		result[i] = int(t)
	}
	return result
}
