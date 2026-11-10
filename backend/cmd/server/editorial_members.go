package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"errors"

	"connectrpc.com/connect"
)

// SearchMembers suggests BCC Members people for a marker's contributors.
func (e EditorialAPI) SearchMembers(ctx context.Context, req *connect.Request[apiv1.SearchMembersRequest]) (*connect.Response[apiv1.SearchMembersResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("member search is not available yet"))
}
