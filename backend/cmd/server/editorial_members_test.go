package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/members"
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/bcc-code/mediabank-bridge/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMembers struct {
	limit int
	err   error
}

func (f *fakeMembers) Search(_ context.Context, _ string, limit int) ([]members.Person, error) {
	f.limit = limit
	return []members.Person{{PersonID: "90001", DisplayName: "Ola Nordmann"}}, f.err
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
	assert.Equal(t, "90001", got[0].PersonId)
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

func TestSearchMembersNeedsEditor(t *testing.T) {
	api := newTestEditorialAPI(t)
	api.members = &fakeMembers{}

	actAs(t, `{"enabled":true}`)
	_, err := searchMembers(t, api, "ola", 0)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "a reviewer can't search")

	actAs(t, `{"editor":true}`)
	got, err := searchMembers(t, api, "ola", 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Ola Nordmann", got[0].DisplayName)
}
