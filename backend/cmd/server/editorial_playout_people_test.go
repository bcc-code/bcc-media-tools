package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/playout"
	"context"
	"encoding/json"
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

func TestPlayoutCandidatesReadPersonIDs(t *testing.T) {
	window := recordingWindow{Start: ts(t, "2026-10-17T11:00:00Z"), End: ts(t, "2026-10-17T12:00:00Z")}
	got := playoutCandidates([]playout.ManifestEntry{
		speakerWithData(t, "Kari Nordmann", `{"personId":"90002","uid":"90002","name":"Kari Nordmann"}`, "2026-10-17T11:15:50Z"),
		speakerWithData(t, "Per Hansen", `{"personId":1337,"uid":"1337","name":"Per Hansen"}`, "2026-10-17T11:29:31Z"),
		speakerWithData(t, "Ola Nordmann", `{"personId":90001,"name":"Ola Nordmann"}`, "2026-10-17T11:30:41Z"),
		speakerWithData(t, "Odd", `{"personId":{"id":1},"name":"Odd"}`, "2026-10-17T11:31:00Z"),
		speakerWithData(t, "Label Only", ``, "2026-10-17T11:32:00Z"),
		songEntry(t, "HV 1 - Test song", "2026-10-17T11:35:10Z"),
	}, window)

	require.Len(t, got, 6)
	assert.Equal(t, []contributor{{"Kari Nordmann", "90002"}}, got[0].Contributors)
	assert.Equal(t, []contributor{{"Per Hansen", ""}}, got[1].Contributors, "1337 is a placeholder")
	assert.Equal(t, []contributor{{"Ola Nordmann", "90001"}}, got[2].Contributors, "a numeric personId")
	assert.Equal(t, []contributor{{"Odd", ""}}, got[3].Contributors, "an odd personId keeps the name")
	assert.Equal(t, []contributor{{"Label Only", ""}}, got[4].Contributors, "no data falls back to the label")
	assert.Empty(t, got[5].Contributors)
	assert.Equal(t, "HV 1 - Test song", got[5].Name)
}

func TestImportFromPlayoutCarriesPersonIDs(t *testing.T) {
	api := EditorialAPI{playout: fakePlayout{manifest: &playout.Manifest{Entries: []playout.ManifestEntry{
		speakerWithData(t, "Kari Nordmann", `{"personId":"90002","uid":"90002","name":"Kari Nordmann"}`, "2026-10-17T11:15:50Z"),
	}}}}
	markers, err := api.importFromPlayout(context.Background(), "1154", recordingWindow{Start: ts(t, "2026-10-17T11:00:00Z"), End: ts(t, "2026-10-17T12:00:00Z")})
	require.NoError(t, err)
	require.Len(t, markers, 1)
	assert.Equal(t, "Kari Nordmann", markers[0].Contributors)
	assert.Equal(t, []contributor{{"Kari Nordmann", "90002"}}, fromProto(markers[0].ContributorList))
}

// A synced row follows Playout's corrections, IDs included, until an editor
// links someone by hand: picking a member for the same name is an edit.
func TestEditorialLiveSyncPersonIDs(t *testing.T) {
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

	row := sync(`{"personId":1337,"name":"Ola Nordmann"}`)
	assert.Equal(t, []contributor{{"Ola Nordmann", ""}}, fromProto(row.ContributorList))
	assert.True(t, row.PlayoutSynced)

	row = sync(`{"personId":"90001","name":"Ola Nordmann"}`)
	assert.Equal(t, []contributor{{"Ola Nordmann", "90001"}}, fromProto(row.ContributorList), "a synced row takes Playout's correction")
	assert.True(t, row.PlayoutSynced)

	row.ContributorList = []*apiv1.EditorialContributor{{Name: "Ola Nordmann", PersonId: "90003"}}
	saved := saveTestSession(t, api, sess.Id, "Lørdag", []*apiv1.EditorialMarker{row})
	assert.False(t, saved.Markers[0].PlayoutSynced, "a member picked by hand stops the sync")

	row = sync(`{"personId":"90002","name":"Kari Nordmann"}`)
	assert.Equal(t, []contributor{{"Ola Nordmann", "90003"}}, fromProto(row.ContributorList), "the pick stays")
}
