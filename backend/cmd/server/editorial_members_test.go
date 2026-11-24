package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/members"
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/bcc-code/mediabank-bridge/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMembers struct {
	// found for any query; Ola Nordmann when nil
	people []members.Person
	limit  int
	err    error
	// GetMany's calls
	lookups [][]string
	// GetMany finds these; when nil, everyone but uid(99999), as "Person <n>"
	byID map[string]members.Person
	mu   sync.Mutex
	gets map[string]int
}

func (f *fakeMembers) getCount(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.gets[id]
}

func (f *fakeMembers) Search(_ context.Context, _ string, limit int) ([]members.Person, error) {
	f.limit = limit
	if f.err != nil {
		return nil, f.err
	}
	if f.people != nil {
		return f.people, nil
	}
	return []members.Person{{UID: uid(90001), DisplayName: "Ola Nordmann"}}, nil
}

func (f *fakeMembers) GetMany(_ context.Context, ids []string) ([]members.Person, error) {
	f.mu.Lock()
	f.lookups = append(f.lookups, ids)
	if f.gets == nil {
		f.gets = map[string]int{}
	}
	for _, id := range ids {
		f.gets[id]++
	}
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	var out []members.Person
	for _, id := range slices.Backward(ids) {
		if p, ok := f.byID[id]; ok {
			out = append(out, p)
		} else if f.byID == nil && id != uid(99999) {
			out = append(out, members.Person{UID: id, DisplayName: "Person " + strings.TrimLeft(id[24:], "0")})
		}
	}
	return out, nil
}

func searchMembers(t *testing.T, api *EditorialAPI, query string, limit int32) ([]*apiv1.Member, error) {
	t.Helper()
	res, err := api.SearchMembers(context.Background(), connect.NewRequest(&apiv1.SearchMembersRequest{Query: query, Limit: limit}))
	if err != nil {
		return nil, err
	}
	return res.Msg.Members, nil
}

func TestSearchMembers(t *testing.T) {
	api := newTestEditorialAPI(t)
	fake := &fakeMembers{}
	api.members = fake

	got, err := searchMembers(t, api, " ola ", 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, uid(90001), got[0].MemberUid)
	assert.Equal(t, "Ola Nordmann", got[0].DisplayName)
	assert.Equal(t, 10, fake.limit, "default limit")

	_, err = searchMembers(t, api, "ola", 500)
	require.NoError(t, err)
	assert.Equal(t, 25, fake.limit, "capped limit")

	fake.limit = 0
	got, err = searchMembers(t, api, "k", 0)
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Zero(t, fake.limit, "a one-letter query doesn't search")

	log.ConfigureGlobalLogger(zerolog.Disabled) // the failure is logged
	fake.err = errors.New("members down")
	got, err = searchMembers(t, api, "ola", 0)
	require.NoError(t, err, "a failing Members finds no one")
	assert.Empty(t, got)

	api.members = nil
	got, err = searchMembers(t, api, "ola", 0)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// Members is personal data: searching needs the editor role and the explicit
// members permission, which no role implies.
func TestSearchMembersNeedsEditorAndMembers(t *testing.T) {
	api := newTestEditorialAPI(t)
	api.members = &fakeMembers{}

	for perms, why := range map[string]string{
		`{"enabled":true,"members":true}`: "a reviewer can't search",
		`{"editor":true}`:                 "an editor needs the members permission",
		`{"admin":true}`:                  "so does an admin",
	} {
		actAs(t, perms)
		_, err := searchMembers(t, api, "ola", 0)
		assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), why)
	}

	actAs(t, `{"editor":true,"members":true}`)
	got, err := searchMembers(t, api, "ola", 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Ola Nordmann", got[0].DisplayName)
}

func TestNewFrequentContributors(t *testing.T) {
	t.Setenv("MEMBERS_FREQUENT_UIDS", " "+uid(90003)+", ,"+uid(90001)+",")
	assert.Equal(t, []string{uid(90003), uid(90001)}, newFrequentContributors().uids)

	t.Setenv("MEMBERS_FREQUENT_UIDS", "")
	assert.Empty(t, newFrequentContributors().uids)
}

// Frequent contributors are configured as uids only; Members names them, once
// a day.
func TestSearchMembersFrequent(t *testing.T) {
	api := newTestEditorialAPI(t)
	fake := &fakeMembers{}
	api.members = fake
	api.frequent = &frequentContributors{uids: []string{uid(90003), uid(90001), uid(99999), uid(90002)}}

	got, err := searchMembers(t, api, "", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"Person 90003", "Person 90001", "Person 90002"}, names(got), "in order, unknown left out")

	got, err = searchMembers(t, api, " ", 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"Person 90003", "Person 90001"}, names(got))
	assert.Len(t, fake.lookups, 1, "looked up once a day")

	log.ConfigureGlobalLogger(zerolog.Disabled) // the failure is logged
	fake.err = errors.New("members down")
	api.frequent.fetched = time.Now().Add(-25 * time.Hour)
	got, err = searchMembers(t, api, "", 0)
	require.NoError(t, err)
	assert.Len(t, fake.lookups, 2, "looked up again the next day")
	assert.Equal(t, []string{"Person 90003", "Person 90001", "Person 90002"}, names(got), "the last names when Members fails")

	api.members = nil
	got, err = searchMembers(t, api, "", 0)
	require.NoError(t, err)
	assert.Empty(t, got)

	api.members = fake
	api.frequent = &frequentContributors{}
	got, err = searchMembers(t, api, "", 0)
	require.NoError(t, err)
	assert.Empty(t, got, "none configured")
	assert.Len(t, fake.lookups, 2, "and Members isn't asked")
}

// People already linked on markers come first, then Members' matches.
func TestSearchMembersLinkedFirst(t *testing.T) {
	api := newTestEditorialAPI(t)
	fake := &fakeMembers{people: []members.Person{
		{UID: uid(90002), DisplayName: "Kari Nordmann"},
		{UID: uid(90004), DisplayName: "Kari Nordvik"},
	}}
	api.members = fake
	sess := createTestSession(t, api, "VX-1", "stream")
	saveTestSession(t, api, sess.Id, "t", []*apiv1.EditorialMarker{
		{ContributorList: []*apiv1.EditorialContributor{{Name: "Kari Nordmann", MemberUid: uid(90002)}, {Name: "Kari Typed"}}},
		{ContributorList: []*apiv1.EditorialContributor{{Name: "Kari Nordmann", MemberUid: uid(90002)}, {Name: "Øystein Kari-Berg", MemberUid: uid(90003)}}},
		{ContributorList: []*apiv1.EditorialContributor{{Name: "K. Nordmann", MemberUid: uid(90002)}, {Name: "Ola Nordmann", MemberUid: uid(90001)}}},
	})

	got, err := searchMembers(t, api, "kari", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"Kari Nordmann", "Øystein Kari-Berg", "Kari Nordvik"}, names(got),
		"most used name, word in a hyphenated name, typed-in left out, no duplicates")

	got, err = searchMembers(t, api, "ØYST", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"Øystein Kari-Berg", "Kari Nordmann", "Kari Nordvik"}, names(got))

	fake.err = errors.New("members down")
	log.ConfigureGlobalLogger(zerolog.Disabled)
	got, err = searchMembers(t, api, "k nord", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"Kari Nordmann"}, names(got), "linked people still found")

	fake.limit = 0
	got, err = searchMembers(t, api, "k", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"Kari Nordmann", "Øystein Kari-Berg"}, names(got), "one letter finds linked people only")
	assert.Zero(t, fake.limit)
}

func names(list []*apiv1.Member) []string {
	out := make([]string, len(list))
	for i, m := range list {
		out[i] = m.DisplayName
	}
	return out
}
