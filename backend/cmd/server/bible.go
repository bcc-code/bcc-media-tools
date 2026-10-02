package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/bible"
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
)

// BibleAPI resolves the free-text bible references on editorial markers to
// verse text, so a reviewer can read a reference without leaving the tool.
type BibleAPI struct {
	client *bible.Client
}

func NewBibleAPI(client *bible.Client) *BibleAPI {
	return &BibleAPI{client: client}
}

// GetBibleVerses parses one reference and fetches its text. Bible text is not
// privileged, so this only requires a signed-in user rather than a specific
// tool permission.
//
// A reference that cannot be parsed, or that resolves to no text, is not an
// error: the response carries whatever could be made of it and the client
// falls back to showing the reference as typed.
func (b BibleAPI) GetBibleVerses(ctx context.Context, req *connect.Request[apiv1.GetBibleVersesRequest]) (*connect.Response[apiv1.GetBibleVersesResponse], error) {
	if getEmail(req) == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing email header"))
	}

	ref, err := bible.Parse(req.Msg.GetReference())
	if err != nil {
		return connect.NewResponse(&apiv1.GetBibleVersesResponse{
			Reference: req.Msg.GetReference(),
		}), nil
	}

	out := &apiv1.GetBibleVersesResponse{
		Reference:   ref.Display(),
		Translation: b.client.Translation(),
	}

	verses, err := b.client.Verses(ctx, ref)
	if errors.Is(err, bible.ErrNotConfigured) || errors.Is(err, bible.ErrWholeChapter) {
		return connect.NewResponse(out), nil
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}

	for _, v := range verses {
		out.Verses = append(out.Verses, &apiv1.BibleVerse{Number: v.Number, Text: v.Text})
	}
	return connect.NewResponse(out), nil
}
