package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/members"
	"cmp"
	"context"
	"os"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/bcc-code/mediabank-bridge/log"
)

// newMembersSearcher searches the Core API; nil, finding no one, when
// MEMBERS_CLIENT_ID isn't set.
func newMembersSearcher() members.Searcher {
	if os.Getenv("MEMBERS_CLIENT_ID") == "" {
		log.L.Info().Msg("MEMBERS_CLIENT_ID not set: contributor search finds no one")
		return nil
	}
	return members.NewClient(members.Config{
		APIURL:       cmp.Or(os.Getenv("MEMBERS_API_URL"), "https://api.bcc.no"),
		AuthURL:      cmp.Or(os.Getenv("MEMBERS_AUTH0_BASE_URL"), "https://login.bcc.no"),
		Audience:     cmp.Or(os.Getenv("MEMBERS_AUDIENCE"), "api.bcc.no"),
		ClientID:     os.Getenv("MEMBERS_CLIENT_ID"),
		ClientSecret: os.Getenv("MEMBERS_CLIENT_SECRET"),
	})
}

const (
	defaultMemberResults = 10
	maxMemberResults     = 25
)

// SearchMembers suggests people for a marker's contributors: frequent Playout
// speakers first, then BCC Members' best matches. An empty query lists the
// frequent speakers. When Members fails only they are found, so a name can
// still be typed in by hand.
func (e EditorialAPI) SearchMembers(ctx context.Context, req *connect.Request[apiv1.SearchMembersRequest]) (*connect.Response[apiv1.SearchMembersResponse], error) {
	if _, err := requireEditorial(req, apiv1.EditorialEditor); err != nil {
		return nil, err
	}
	query := strings.TrimSpace(req.Msg.GetQuery())
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = defaultMemberResults
	}
	limit = min(limit, maxMemberResults)

	res := &apiv1.SearchMembersResponse{}
	seen := map[string]bool{}
	for _, p := range e.speakers.match(query, limit) {
		seen[p.PersonID] = true
		res.Members = append(res.Members, &apiv1.Member{PersonId: p.PersonID, DisplayName: p.Name})
	}
	if e.members == nil || utf8.RuneCountInString(query) < 2 || len(res.Members) == limit {
		return connect.NewResponse(res), nil
	}
	people, err := e.members.Search(ctx, query, limit)
	if err != nil {
		log.L.Warn().Err(err).Msg("member search failed")
		return connect.NewResponse(res), nil
	}
	for _, p := range people {
		if len(res.Members) == limit {
			break
		}
		if seen[p.PersonID] {
			continue
		}
		seen[p.PersonID] = true
		res.Members = append(res.Members, &apiv1.Member{PersonId: p.PersonID, DisplayName: p.DisplayName})
	}
	return connect.NewResponse(res), nil
}
