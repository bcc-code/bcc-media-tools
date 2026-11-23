package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"bcc-media-tools/playout"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// speakerWithData is a SPEAKER entry with data as Playout sends it.
func speakerWithData(t *testing.T, label, data, at string) playout.ManifestEntry {
	return playout.ManifestEntry{Type: playout.ContentTypeSpeaker, Label: label, Timestamp: ts(t, at), Data: json.RawMessage(data)}
}

func TestPlayoutCandidatesReadMemberUIDs(t *testing.T) {
	window := recordingWindow{Start: ts(t, "2026-10-17T11:00:00Z"), End: ts(t, "2026-10-17T12:00:00Z")}
	got := playoutCandidates([]playout.ManifestEntry{
		speakerWithData(t, "Kari Nordmann", `{"personId":90002,"uid":"`+uid(90002)+`","name":"Kari Nordmann"}`, "2026-10-17T11:15:50Z"),
		speakerWithData(t, "Per Hansen", `{"personId":1337,"uid":"1337","name":"Per Hansen"}`, "2026-10-17T11:29:31Z"),
		speakerWithData(t, "Ola Nordmann", `{"personId":"90001","uid":"90001","name":"Ola Nordmann"}`, "2026-10-17T11:30:41Z"),
		speakerWithData(t, "Unknown", `{"personId":null,"uid":null,"name":"Unknown"}`, "2026-10-17T11:30:50Z"),
		speakerWithData(t, "Odd", `{"uid":{"id":1},"name":"Odd"}`, "2026-10-17T11:31:00Z"),
		speakerWithData(t, "Label Only", ``, "2026-10-17T11:32:00Z"),
		songEntry(t, "HV 1 - Test song", "2026-10-17T11:35:10Z"),
	}, window)

	require.Len(t, got, 7)
	assert.Equal(t, []editorial.Contributor{{Name: "Kari Nordmann", MemberUID: uid(90002)}}, got[0].Contributors)
	assert.Equal(t, []editorial.Contributor{{Name: "Per Hansen", MemberUID: ""}}, got[1].Contributors, "the 1337 placeholder")
	assert.Equal(t, []editorial.Contributor{{Name: "Ola Nordmann", MemberUID: ""}}, got[2].Contributors, "an older event's uid is the personId")
	assert.Equal(t, []editorial.Contributor{{Name: "Unknown", MemberUID: ""}}, got[3].Contributors, "a person Playout doesn't know")
	assert.Equal(t, []editorial.Contributor{{Name: "Odd", MemberUID: ""}}, got[4].Contributors, "an odd uid keeps the name")
	assert.Equal(t, []editorial.Contributor{{Name: "Label Only", MemberUID: ""}}, got[5].Contributors, "no data falls back to the label")
	assert.Empty(t, got[6].Contributors)
	assert.Equal(t, "HV 1 - Test song", got[6].Name)
}

func TestImportFromPlayoutCarriesMemberUIDs(t *testing.T) {
	api := EditorialAPI{playout: fakePlayout{manifest: &playout.Manifest{Entries: []playout.ManifestEntry{
		speakerWithData(t, "Kari Nordmann", `{"personId":90002,"uid":"`+strings.ToUpper(uid(90002))+`","name":"Kari Nordmann"}`, "2026-10-17T11:15:50Z"),
	}}}}
	markers, err := api.importFromPlayout(context.Background(), "1154", recordingWindow{Start: ts(t, "2026-10-17T11:00:00Z"), End: ts(t, "2026-10-17T12:00:00Z")})
	require.NoError(t, err)
	require.Len(t, markers, 1)
	assert.Equal(t, []editorial.Contributor{{Name: "Kari Nordmann", MemberUID: uid(90002)}}, fromProto(markers[0].ContributorList), "stored lowercase")
}

// A synced row follows Playout's corrections, IDs included, until an editor
// links someone by hand: picking a member for the same name is an edit.
func TestEditorialLiveSyncMemberUIDs(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	created, err := api.CreateEditorialSession(ctx, connect.NewRequest(&apiv1.CreateEditorialSessionRequest{Live: true, Title: "Lørdag", PlayoutEventId: "1154"}))
	require.NoError(t, err)
	sess := created.Msg
	clockStart := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
	_, err = api.db.ExecContext(ctx, "UPDATE sessions SET clock_start_ms = ? WHERE id = ?", clockStart.UnixMilli(), sess.Id)
	require.NoError(t, err)

	at := clockStart.Add(10 * time.Second).UTC().Format(time.RFC3339Nano)
	sync := func(data string) *apiv1.EditorialMarker {
		t.Helper()
		api.playout = fakePlayout{manifest: &playout.Manifest{Entries: []playout.ManifestEntry{
			speakerWithData(t, "label", data, at),
		}}}
		res, err := api.SyncEditorialSessionFromPlayout(ctx, connect.NewRequest(&apiv1.SyncEditorialSessionFromPlayoutRequest{Id: sess.Id}))
		require.NoError(t, err)
		require.Len(t, res.Msg.Markers, 1)
		return res.Msg.Markers[0]
	}

	row := sync(`{"personId":1337,"uid":"1337","name":"Ola Nordmann"}`)
	assert.Equal(t, []editorial.Contributor{{Name: "Ola Nordmann", MemberUID: ""}}, fromProto(row.ContributorList))
	assert.True(t, row.PlayoutSynced)

	row = sync(`{"personId":90001,"uid":"` + uid(90001) + `","name":"Ola Nordmann"}`)
	assert.Equal(t, []editorial.Contributor{{Name: "Ola Nordmann", MemberUID: uid(90001)}}, fromProto(row.ContributorList), "a synced row takes Playout's correction")
	assert.True(t, row.PlayoutSynced)

	row.ContributorList = []*apiv1.EditorialContributor{{Name: "Ola Nordmann", MemberUid: uid(90003)}}
	saved := saveTestSession(t, api, sess.Id, "Lørdag", []*apiv1.EditorialMarker{row})
	assert.False(t, saved.Markers[0].PlayoutSynced, "a member picked by hand stops the sync")

	row = sync(`{"personId":90002,"uid":"` + uid(90002) + `","name":"Kari Nordmann"}`)
	assert.Equal(t, []editorial.Contributor{{Name: "Ola Nordmann", MemberUID: uid(90003)}}, fromProto(row.ContributorList), "the pick stays")
}
