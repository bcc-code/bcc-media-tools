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

// frequentContributors are the people suggested for an empty field, named
// through Members once a day.
type frequentContributors struct {
	// Members uids, in the order they are suggested
	uids []string

	mu      sync.Mutex
	people  []members.Person
	fetched time.Time
}

// newFrequentContributors reads MEMBERS_FREQUENT_UIDS: Members uids separated
// by commas. They're kept out of the code as they identify people.
func newFrequentContributors() *frequentContributors {
	var uids []string
	for _, id := range strings.Split(os.Getenv("MEMBERS_FREQUENT_UIDS"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			uids = append(uids, id)
		}
	}
	return &frequentContributors{uids: uids}
}

// get returns the named people in uids' order; the last ones when Members
// fails.
func (f *frequentContributors) get(ctx context.Context, m members.Searcher) []members.Person {
	if len(f.uids) == 0 {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if time.Since(f.fetched) < 24*time.Hour {
		return f.people
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	people, err := m.GetMany(ctx, f.uids)
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
	for _, id := range f.uids {
		if p, ok := byID[id]; ok {
			f.people = append(f.people, p)
		}
	}
	f.fetched = time.Now()
	return f.people
}

// SearchMembers suggests contributors
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

	// ages as of the session's recording, when there is one
	date := today()
	if id := req.Msg.GetSessionId(); id != "" {
		if sess, err := e.queries.GetSession(ctx, id); err == nil {
			date = e.recordingDate(sess)
		}
	}

	res := &apiv1.SearchMembersResponse{}
	// people as Members just sent them; the linked ones are looked up
	known := map[string]members.Person{}
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
				known[p.UID] = p
				add(p.UID, p.DisplayName)
			}
		}
		return e.withDetails(ctx, res, known, date), nil
	}

	linked, err := e.queries.SearchLinkedContributors(ctx, query)
	if err != nil {
		return nil, editorialErr(err)
	}
	for _, c := range linked {
		add(c.MemberUID, c.Name)
	}
	if e.members == nil || utf8.RuneCountInString(query) < 2 || len(res.Members) == limit {
		return e.withDetails(ctx, res, known, date), nil
	}
	people, err := e.members.Search(ctx, query, limit)
	if err != nil {
		log.L.Warn().Err(err).Msg("member search failed")
		return e.withDetails(ctx, res, known, date), nil
	}
	for _, p := range people {
		if !seen[p.UID] {
			known[p.UID] = p
		}
		add(p.UID, p.DisplayName)
	}
	return e.withDetails(ctx, res, known, date), nil
}

// withDetails sets age on date and church: from known, or else from the linked
// contributors kept in e.people. Search results aren't kept.
func (e EditorialAPI) withDetails(ctx context.Context, res *apiv1.SearchMembersResponse, known map[string]members.Person, date time.Time) *connect.Response[apiv1.SearchMembersResponse] {
	var ids []string
	for _, m := range res.Members {
		if _, ok := known[m.MemberUid]; !ok {
			ids = append(ids, m.MemberUid)
		}
	}
	for id, p := range e.lookUpMembers(ctx, ids) {
		known[id] = p
	}
	for _, m := range res.Members {
		if p, ok := known[m.MemberUid]; ok {
			m.Age = int32(p.Age(date))
			m.Church = p.Church
		}
	}
	return connect.NewResponse(res)
}
