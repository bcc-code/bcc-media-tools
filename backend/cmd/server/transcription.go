package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"connectrpc.com/connect"
	"context"
	"github.com/bcc-code/bcc-media-flows/services/cantemo"
	"github.com/samber/lo"
)

type TranscriptionAPI struct {
	cantemoClient *cantemo.Client
}

func NewTranscriptionAPI(baseURL, token string) *TranscriptionAPI {
	return &TranscriptionAPI{
		cantemoClient: cantemo.NewClient(baseURL, token),
	}
}

func (t TranscriptionAPI) GetTranscription(_ context.Context, req *connect.Request[apiv1.GetTranscriptionReqest]) (*connect.Response[apiv1.Transcription], error) {
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
			Id:               int32(s.ID),
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

func (t TranscriptionAPI) GetPreview(_ context.Context, req *connect.Request[apiv1.GetPreviewRequest]) (*connect.Response[apiv1.Preview], error) {
	url, err := t.cantemoClient.GetPreviewUrl(req.Msg.VXID)
	return connect.NewResponse(&apiv1.Preview{Url: url}), err
}
