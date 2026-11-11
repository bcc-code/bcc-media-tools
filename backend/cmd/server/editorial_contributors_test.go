package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContributorsFromRequest(t *testing.T) {
	kare := &apiv1.EditorialContributor{Name: "Kåre J. Smith", PersonId: "10425"}
	storedKare := []contributor{{"Kåre J. Smith", "10425"}}
	for _, tt := range []struct {
		name   string
		list   []*apiv1.EditorialContributor
		legacy string
		stored []contributor
		want   []contributor
	}{
		{"string only keeps the ids the row has", nil, "Kåre J. Smith, Ola", storedKare, []contributor{{"Kåre J. Smith", "10425"}, {"Ola", ""}}},
		{"string only, renamed, drops the stored id", nil, "Kåre Smith", storedKare, []contributor{{"Kåre Smith", ""}}},
		{"a matching list can unlink a stored id", []*apiv1.EditorialContributor{{Name: "Kåre J. Smith"}}, "Kåre J. Smith", storedKare, []contributor{{"Kåre J. Smith", ""}}},
		{"list matching the string keeps its ids", []*apiv1.EditorialContributor{kare}, "Kåre J. Smith", nil, []contributor{{"Kåre J. Smith", "10425"}}},
		{"list without a string", []*apiv1.EditorialContributor{kare}, "", nil, []contributor{{"Kåre J. Smith", "10425"}}},
		{"name added to the string keeps the linked one", []*apiv1.EditorialContributor{kare}, "Kåre J. Smith, Ola", nil, []contributor{{"Kåre J. Smith", "10425"}, {"Ola", ""}}},
		{"renamed in the string drops the link", []*apiv1.EditorialContributor{kare}, "Kåre Smith", nil, []contributor{{"Kåre Smith", ""}}},
		{"string only, as old clients send it", nil, " A ,, B ", nil, []contributor{{"A", ""}, {"B", ""}}},
		{"blank names in the list are dropped", []*apiv1.EditorialContributor{{Name: " "}, kare}, "", nil, []contributor{{"Kåre J. Smith", "10425"}}},
		{"both empty clears", nil, "", storedKare, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, contributorsFromRequest(tt.list, tt.legacy, tt.stored))
		})
	}
}

func TestMarkerContributorsReadsRowsFromBeforeTheList(t *testing.T) {
	assert.Equal(t, []contributor{{"A", ""}, {"B", ""}}, markerContributors(editorial.Marker{Contributors: "A, B", ContributorList: "[]"}))
	assert.Equal(t, []contributor{{"A", "1"}}, markerContributors(editorial.Marker{Contributors: "A", ContributorList: `[{"name":"A","personId":"1"}]`}))
	assert.Empty(t, markerContributors(editorial.Marker{}))
}

func TestPlayoutFieldsChangedComparesContributorIDs(t *testing.T) {
	var picked editorial.Marker
	setContributors(&picked, []contributor{{"Kåre J. Smith", "10425"}})
	var typed editorial.Marker
	setContributors(&typed, []contributor{{"Kåre J. Smith", ""}})
	old := editorial.Marker{Contributors: "Kåre J. Smith", ContributorList: "[]"}

	assert.True(t, playoutFieldsChanged(typed, picked), "picking a member for the same name is an edit")
	assert.False(t, playoutFieldsChanged(old, typed), "a row from before the list is not edited by saving it as it is")
}

func TestEditorialContributorListRoundTrip(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	sess := createTestSession(t, api, "VX-1", "stream")
	saved := saveTestSession(t, api, sess.Id, "t", []*apiv1.EditorialMarker{
		{ContributorList: []*apiv1.EditorialContributor{{Name: "Kåre J. Smith", PersonId: "10425"}, {Name: "Ola"}}, StartMs: 0, EndMs: 1},
		{Contributors: "A, B", StartMs: 2, EndMs: 3},
	})

	got, err := api.loadSession(ctx, sess.Id)
	require.NoError(t, err)
	for _, s := range []*apiv1.EditorialSession{saved, got} {
		assert.Equal(t, "Kåre J. Smith, Ola", s.Markers[0].Contributors)
		assert.Equal(t, []contributor{{"Kåre J. Smith", "10425"}, {"Ola", ""}}, fromProto(s.Markers[0].ContributorList))
		assert.Equal(t, "A, B", s.Markers[1].Contributors)
		assert.Equal(t, []contributor{{"A", ""}, {"B", ""}}, fromProto(s.Markers[1].ContributorList))
	}

	_, err = api.SetEditorialDetails(ctx, connect.NewRequest(&apiv1.SetEditorialDetailsRequest{
		SessionId: sess.Id, MarkerId: got.Markers[1].Id,
		ContributorList: []*apiv1.EditorialContributor{{Name: "Harald Kronstad", PersonId: "10205"}},
	}))
	require.NoError(t, err)
	got, err = api.loadSession(ctx, sess.Id)
	require.NoError(t, err)
	assert.Equal(t, "Harald Kronstad", got.Markers[1].Contributors)
	assert.Equal(t, []contributor{{"Harald Kronstad", "10205"}}, fromProto(got.Markers[1].ContributorList))
}

// The current frontend sends only the contributors string, on every details
// edit and on a full save. IDs already on the row must survive that.
func TestEditorialStringOnlyWritesKeepStoredIDs(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	sess := createTestSession(t, api, "VX-1", "stream")
	saved := saveTestSession(t, api, sess.Id, "t", []*apiv1.EditorialMarker{
		{ContributorList: []*apiv1.EditorialContributor{{Name: "Kåre J. Smith", PersonId: "10425"}}, StartMs: 0, EndMs: 1},
	})
	markerID := saved.Markers[0].Id
	contributorsOf := func() []contributor {
		got, err := api.loadSession(ctx, sess.Id)
		require.NoError(t, err)
		return fromProto(got.Markers[0].ContributorList)
	}

	_, err := api.SetEditorialDetails(ctx, connect.NewRequest(&apiv1.SetEditorialDetailsRequest{
		SessionId: sess.Id, MarkerId: markerID, Contributors: "Kåre J. Smith", Type: "sang",
	}))
	require.NoError(t, err)
	assert.Equal(t, []contributor{{"Kåre J. Smith", "10425"}}, contributorsOf(), "a details edit of another field")

	saveTestSession(t, api, sess.Id, "t", []*apiv1.EditorialMarker{
		{Id: markerID, Contributors: "Kåre J. Smith, Ola", StartMs: 0, EndMs: 1},
	})
	assert.Equal(t, []contributor{{"Kåre J. Smith", "10425"}, {"Ola", ""}}, contributorsOf(), "a full save adding a name")

	saveTestSession(t, api, sess.Id, "t", []*apiv1.EditorialMarker{
		{Id: markerID, Contributors: "Kåre Smith", StartMs: 0, EndMs: 1},
	})
	assert.Equal(t, []contributor{{"Kåre Smith", ""}}, contributorsOf(), "a rename drops the link")
}

func fromProto(list []*apiv1.EditorialContributor) []contributor {
	var out []contributor
	for _, c := range list {
		out = append(out, contributor{c.GetName(), c.GetPersonId()})
	}
	return out
}
