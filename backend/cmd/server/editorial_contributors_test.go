package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContributorsFromRequest(t *testing.T) {
	got := contributorsFromRequest([]*apiv1.EditorialContributor{
		{Name: " Ola Nordmann ", PersonId: " 90001 "},
		{Name: " ", PersonId: "90002"},
		{Name: "Ola"},
	})
	assert.Equal(t, []contributor{{"Ola Nordmann", "90001"}, {"Ola", ""}}, got)
	assert.Empty(t, contributorsFromRequest(nil))
}

func TestPlayoutFieldsChangedComparesContributorIDs(t *testing.T) {
	picked := markerRow{Contributors: []contributor{{"Ola Nordmann", "90001"}}}
	typed := markerRow{Contributors: []contributor{{"Ola Nordmann", ""}}}
	assert.True(t, playoutFieldsChanged(typed, picked), "picking a member for the same name is an edit")
}

// markerRows wraps markers that have no contributors.
func markerRows(markers []editorial.Marker) []markerRow {
	rows := make([]markerRow, len(markers))
	for i, m := range markers {
		rows[i] = markerRow{Marker: m}
	}
	return rows
}

func TestEditorialContributorListRoundTrip(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	sess := createTestSession(t, api, "VX-1", "stream")
	saved := saveTestSession(t, api, sess.Id, "t", []*apiv1.EditorialMarker{
		{ContributorList: []*apiv1.EditorialContributor{{Name: "Ola Nordmann", PersonId: "90001"}, {Name: "Ola"}}, StartMs: 0, EndMs: 1},
		{StartMs: 2, EndMs: 3},
	})

	got, err := api.loadSession(ctx, sess.Id)
	require.NoError(t, err)
	for _, s := range []*apiv1.EditorialSession{saved, got} {
		assert.Equal(t, []contributor{{"Ola Nordmann", "90001"}, {"Ola", ""}}, fromProto(s.Markers[0].ContributorList))
		assert.Empty(t, s.Markers[1].ContributorList)
	}

	_, err = api.SetEditorialDetails(ctx, connect.NewRequest(&apiv1.SetEditorialDetailsRequest{
		SessionId: sess.Id, MarkerId: got.Markers[1].Id,
		ContributorList: []*apiv1.EditorialContributor{{Name: "Kari Nordmann", PersonId: "90002"}},
	}))
	require.NoError(t, err)
	got, err = api.loadSession(ctx, sess.Id)
	require.NoError(t, err)
	assert.Equal(t, []contributor{{"Kari Nordmann", "90002"}}, fromProto(got.Markers[1].ContributorList))
}

// joined is a marker's contributor names as the page shows them.
func joined(m *apiv1.EditorialMarker) string {
	return strings.Join(contributorNames(fromProto(m.GetContributorList())), ", ")
}

func fromProto(list []*apiv1.EditorialContributor) []contributor {
	var out []contributor
	for _, c := range list {
		out = append(out, contributor{c.GetName(), c.GetPersonId()})
	}
	return out
}

// typedIn lists contributors typed in by hand.
func typedIn(names ...string) []*apiv1.EditorialContributor {
	out := make([]*apiv1.EditorialContributor, len(names))
	for i, name := range names {
		out[i] = &apiv1.EditorialContributor{Name: name}
	}
	return out
}
