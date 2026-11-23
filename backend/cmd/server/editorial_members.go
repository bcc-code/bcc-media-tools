package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/members"
	"cmp"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
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

// frequentContributorIDs are the Members uids suggested for an empty field, in
// order: the people who spoke on the most days in Playout's events of the
// 12 months up to November 2026. Refresh by hand when it gets stale.
var frequentContributorIDs = []string{
	"a3f7b724-c05f-46e9-b173-2eff0186eaa7",
	"a10eeb14-5dca-4172-a985-c76a3db75f30",
	"9c0b6bd3-8759-423d-965b-d18947db9d31",
	"20e23a50-3285-4454-b6a3-a6ea1045a1e9",
	"12cfa08e-a47b-4876-af81-a2bb64af17f7",
	"d95046d8-c194-4e8b-8642-d58fcce9265c",
	"696e1045-ae8e-4ce8-9354-334ca4f8352c",
	"926893aa-5426-479a-b2e4-b10a18a55a33",
	"7b5b2e1e-fd0e-4ab3-a044-cb0244d02fa0",
	"5e2b0640-cb7f-4411-bf4a-3fa43d233bec",
	"a743e027-76b6-4fae-991e-75faa7a559b8",
	"d1d1d715-3396-4e10-bd15-7d63ea50ad2b",
	"bd023e36-f8ce-4e8e-8f0f-991070d01250",
	"f273bf7c-2b5d-45aa-a38b-3bd2c80bc4b0",
	"22693469-d111-4321-beb7-027cb9753c5f",
	"f498bf42-d429-4970-a682-fcb848f9e0e3",
	"c6cde9de-8bf0-4fcc-a22e-27a6686aa7fe",
	"3f5ba505-0bab-47e8-8fd1-3b59b3bf7c10",
	"3ef590a3-5000-481d-8dc1-8e3ff4cf6e23",
	"c335c5a3-8a13-4dd5-bd98-806909e73a0b",
}

// frequentContributors names frequentContributorIDs through Members, once a
// day.
type frequentContributors struct {
	mu      sync.Mutex
	people  []members.Person
	fetched time.Time
}

// get returns the named people in frequentContributorIDs' order; the last
// ones when Members fails.
func (f *frequentContributors) get(ctx context.Context, m members.Searcher) []members.Person {
	f.mu.Lock()
	defer f.mu.Unlock()
	if time.Since(f.fetched) < 24*time.Hour {
		return f.people
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	people, err := m.GetMany(ctx, frequentContributorIDs)
	if err != nil {
		log.L.Warn().Err(err).Msg("member lookup failed")
		return f.people
	}
	// Members' order isn't ours
	byID := map[string]members.Person{}
	for _, p := range people {
		byID[p.UID] = p
	}
	f.people = nil
	for _, id := range frequentContributorIDs {
		if p, ok := byID[id]; ok {
			f.people = append(f.people, p)
		}
	}
	f.fetched = time.Now()
	return f.people
}

// SearchMembers suggests contributors: linked ones first, then Members.
func (e EditorialAPI) SearchMembers(ctx context.Context, req *connect.Request[apiv1.SearchMembersRequest]) (*connect.Response[apiv1.SearchMembersResponse], error) {
	email, err := requireEditorial(req, apiv1.EditorialEditor)
	if err != nil {
		return nil, err
	}
	if !PermissionsForEmail(email).CanLookUpMembers() {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized: needs the members permission"))
	}
	query := strings.TrimSpace(req.Msg.GetQuery())
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = defaultMemberResults
	}
	limit = min(limit, maxMemberResults)

	res := &apiv1.SearchMembersResponse{}
	seen := map[string]bool{}
	add := func(id, name string) {
		if len(res.Members) < limit && !seen[id] {
			seen[id] = true
			res.Members = append(res.Members, &apiv1.Member{MemberUid: id, DisplayName: name})
		}
	}

	if query == "" {
		if e.members != nil {
			for _, p := range e.frequent.get(ctx, e.members) {
				add(p.UID, p.DisplayName)
			}
		}
		return connect.NewResponse(res), nil
	}

	linked, err := e.queries.SearchLinkedContributors(ctx, query)
	if err != nil {
		return nil, editorialErr(err)
	}
	for _, c := range linked {
		add(c.MemberUID, c.Name)
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
		add(p.UID, p.DisplayName)
	}
	return connect.NewResponse(res), nil
}
