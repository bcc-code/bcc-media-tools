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

func (t TranscriptionAPI) GetTranscription(ctx context.Context, req *connect.Request[apiv1.GetTranscriptionReqest]) (*connect.Response[apiv1.Transcription], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(401, fmt.Errorf("missing email header"))
	}

	perms := PermissionsForEmail(email)
	if perms.Transcription == nil || (!perms.Transcription.Admin && !perms.Transcription.Mediabanken) {
		return nil, connect.NewError(403, fmt.Errorf("Not enough permissions for transcription."))
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

func (t TranscriptionAPI) GetPreview(ctx context.Context, req *connect.Request[apiv1.GetPreviewRequest]) (*connect.Response[apiv1.Preview], error) {
	email := getEmail(req)
	if email == "" {
		return nil, connect.NewError(401, fmt.Errorf("missing email header"))
	}

	perms := PermissionsForEmail(email)
	if perms.Transcription == nil || (!perms.Transcription.Admin && !perms.Transcription.Mediabanken) {
		return nil, connect.NewError(403, fmt.Errorf("Not enough permissions for preview."))
	}

	// Check if any ACL entry is inherited from the requested VXID
	accessAllowed := perms.Transcription.Admin

	if perms.Transcription.Mediabanken {
		m, err := t.cantemoClient.GetACL(req.Msg.VXID)
		if err != nil {
			return nil, err
		}

		if m != nil {
			for _, acl := range m.ACLs {
				if acl.InheritedFrom != nil && acl.InheritedFrom.ID == "VX-2677" {
					// VX-2677 == "collection_name": "_AccessibleByTools"
					accessAllowed = true
					break
				}
			}
		}
	}

	if !accessAllowed {
		return nil, connect.NewError(403, fmt.Errorf("Not enough permissions for preview."))
	}

	preview, err := t.cantemoClient.GetPreviewUrl(req.Msg.VXID)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&apiv1.Preview{Url: preview}), nil
}

func (t TranscriptionAPI) SubmitTranscription(ctx context.Context, req *connect.Request[apiv1.SubmitTranscriptionRequest]) (*connect.Response[apiv1.Void], error) {
	fmt.Printf("Received transcription for VXID %s: %+v\n", req.Msg.GetVXID(), req.Msg.GetTranscription())

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
