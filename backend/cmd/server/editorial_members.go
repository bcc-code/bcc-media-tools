package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/members"
	"cmp"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
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

// frequentContributorIDs are the personIDs suggested for an empty field, in
// order: the people who spoke on the most days in Playout's events of the
// 12 months up to November 2026. Refresh by hand when it gets stale.
var frequentContributorIDs = []string{
	"10425", "10205", "11629", "11461", "10579", "10360", "10680", "12696", "11165", "17465",
	"11201", "14259", "12687", "15996", "17339", "10129", "11446", "19579", "20996", "10648",
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
		byID[p.PersonID] = p
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

	res := &apiv1.SearchMembersResponse{}
	seen := map[string]bool{}
	add := func(personID, name string) {
		if len(res.Members) < limit && !seen[personID] {
			seen[personID] = true
			res.Members = append(res.Members, &apiv1.Member{PersonId: personID, DisplayName: name})
		}
	}

	if query == "" {
		if e.members != nil {
			for _, p := range e.frequent.get(ctx, e.members) {
				add(p.PersonID, p.DisplayName)
			}
		}
		return e.withDetails(ctx, res), nil
	}

	linked, err := e.queries.ListLinkedContributors(ctx)
	if err != nil {
		return nil, editorialErr(err)
	}
	for _, c := range linked {
		if matchesWords(c.Name, query) {
			add(c.PersonID.String, c.Name)
		}
	}
	if e.members == nil || utf8.RuneCountInString(query) < 2 || len(res.Members) == limit {
		return e.withDetails(ctx, res), nil
	}
	people, err := e.members.Search(ctx, query, limit)
	if err != nil {
		log.L.Warn().Err(err).Msg("member search failed")
		return e.withDetails(ctx, res), nil
	}
	// their details came along: no lookups
	e.people.remember(people)
	for _, p := range people {
		add(p.PersonID, p.DisplayName)
	}
	return e.withDetails(ctx, res), nil
}

func (e EditorialAPI) withDetails(ctx context.Context, res *apiv1.SearchMembersResponse) *connect.Response[apiv1.SearchMembersResponse] {
	ids := make([]string, len(res.Members))
	for i, m := range res.Members {
		ids[i] = m.PersonId
	}
	people := e.people.get(ctx, ids)
	for _, m := range res.Members {
		if p, ok := people[m.PersonId]; ok {
			m.Age = int32(age(p.BirthDate, e.people.today()))
			m.Church = p.Church
		}
	}
	return connect.NewResponse(res)
}

// matchesWords reports whether each word of query starts a word of name,
// ignoring case: "ola nor" finds "Ola Nordmann-Berg".
func matchesWords(name, query string) bool {
	words := nameWords(strings.ToLower(name))
	return !slices.ContainsFunc(nameWords(strings.ToLower(query)), func(typed string) bool {
		return !slices.ContainsFunc(words, func(w string) bool { return strings.HasPrefix(w, typed) })
	})
}

// nameWords splits at spaces, hyphens and dots ("kari nordmann-berg").
func nameWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == '-' || r == '.'
	})
}
