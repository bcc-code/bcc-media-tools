package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayoutFieldsChangedComparesContributorIDs(t *testing.T) {
	picked := editorial.MarkerRow{Contributors: []editorial.Contributor{{Name: "Ola Nordmann", MemberUID: uid(90001)}}}
	typed := editorial.MarkerRow{Contributors: []editorial.Contributor{{Name: "Ola Nordmann", MemberUID: ""}}}
	assert.True(t, playoutFieldsChanged(typed, picked), "picking a member for the same name is an edit")
}

// markerRows wraps markers that have no contributors.
func markerRows(markers []editorial.Marker) []editorial.MarkerRow {
	rows := make([]editorial.MarkerRow, len(markers))
	for i, m := range markers {
		rows[i] = editorial.MarkerRow{Marker: m}
	}
	return rows
}

func TestEditorialContributorListRoundTrip(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	sess := createTestSession(t, api, "VX-1", "stream")
	saved := saveTestSession(t, api, sess.Id, "t", []*apiv1.EditorialMarker{
		{ContributorList: []*apiv1.EditorialContributor{{Name: "Ola Nordmann", MemberUid: uid(90001)}, {Name: "Ola"}}, StartMs: 0, EndMs: 1},
		{StartMs: 2, EndMs: 3},
	})

	got, err := api.loadSession(ctx, sess.Id, "")
	require.NoError(t, err)
	for _, s := range []*apiv1.EditorialSession{saved, got} {
		assert.Equal(t, []editorial.Contributor{{Name: "Ola Nordmann", MemberUID: uid(90001)}, {Name: "Ola", MemberUID: ""}}, fromProto(s.Markers[0].ContributorList))
		assert.Empty(t, s.Markers[1].ContributorList)
	}

	_, err = api.SetEditorialDetails(ctx, connect.NewRequest(&apiv1.SetEditorialDetailsRequest{
		SessionId: sess.Id, MarkerId: got.Markers[1].Id,
		ContributorList: []*apiv1.EditorialContributor{{Name: "Kari Nordmann", MemberUid: uid(90002)}},
	}))
	require.NoError(t, err)
	got, err = api.loadSession(ctx, sess.Id, "")
	require.NoError(t, err)
	assert.Equal(t, []editorial.Contributor{{Name: "Kari Nordmann", MemberUID: uid(90002)}}, fromProto(got.Markers[1].ContributorList))
}

// joined is a marker's contributor names as the page shows them.
func joined(m *apiv1.EditorialMarker) string {
	return strings.Join(editorial.ContributorNames(fromProto(m.GetContributorList())), ", ")
}

func fromProto(list []*apiv1.EditorialContributor) []editorial.Contributor {
	var out []editorial.Contributor
	for _, c := range list {
		out = append(out, editorial.Contributor{Name: c.GetName(), MemberUID: c.GetMemberUid()})
	}
	return out
}

// uid is a Members uid for the tests, numbered for reading: uid(90001) is
// 00000000-0000-4000-8000-000000090001.
func uid(n int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", n)
}

// typedIn lists contributors typed in by hand.
func typedIn(names ...string) []*apiv1.EditorialContributor {
	out := make([]*apiv1.EditorialContributor, len(names))
	for i, name := range names {
		out[i] = &apiv1.EditorialContributor{Name: name}
	}
	return out
}
