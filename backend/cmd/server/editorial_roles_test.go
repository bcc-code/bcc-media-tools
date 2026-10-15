package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	apiv1 "bcc-media-tools/api/v1"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// actAs switches the authenticated test user to one with the given editorial
// permission JSON.
func actAs(t *testing.T, editorialJSON string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "permissions.json")
	body := `{"someone@bcc.media":{"bmm":{},"editorial":` + editorialJSON + `}}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0644))
	PermissionsFile = path
	t.Setenv("DEBUG_AUTH_EMAIL", "someone@bcc.media")
}

func TestEditorialRoleOrder(t *testing.T) {
	for _, tt := range []struct {
		perms *apiv1.Permissions
		want  apiv1.EditorialRole
	}{
		{&apiv1.Permissions{}, apiv1.EditorialNone},
		{&apiv1.Permissions{Editorial: &apiv1.EditorialPermission{}}, apiv1.EditorialNone},
		{&apiv1.Permissions{Editorial: &apiv1.EditorialPermission{Enabled: true}}, apiv1.EditorialReviewer},
		{&apiv1.Permissions{Editorial: &apiv1.EditorialPermission{Editor: true}}, apiv1.EditorialEditor},
		{&apiv1.Permissions{Editorial: &apiv1.EditorialPermission{Live: true}}, apiv1.EditorialLive},
		// A higher role counts even when the lower ones aren't ticked.
		{&apiv1.Permissions{Editorial: &apiv1.EditorialPermission{Enabled: true, Live: true}}, apiv1.EditorialLive},
		{&apiv1.Permissions{Editorial: &apiv1.EditorialPermission{Admin: true}}, apiv1.EditorialAdmin},
		{&apiv1.Permissions{Admin: true}, apiv1.EditorialAdmin},
	} {
		assert.Equal(t, tt.want, tt.perms.EditorialRole(), "%v", tt.perms)
	}
}

// Every editorial endpoint against every role: each call either passes the
// permission check or is refused with PermissionDenied.
func TestEditorialRoleAccess(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	// Set up as the admin newTestEditorialAPI acts as.
	rec := createTestSession(t, api, "VX-1", "recording")
	saved := saveTestSession(t, api, rec.Id, "recording", []*apiv1.EditorialMarker{{Name: "a", StartMs: 0, EndMs: 1}})
	markerID := saved.Markers[0].Id
	// A live session has no VXID, so loading it doesn't ask Cantemo (absent in
	// tests) for a preview.
	live, err := api.CreateEditorialSession(ctx, connect.NewRequest(&apiv1.CreateEditorialSessionRequest{Live: true, PlayoutEventId: "1013"}))
	require.NoError(t, err)
	api.playout = fakePlayout{manifest: nil}

	calls := map[string]struct {
		role apiv1.EditorialRole
		call func() error
	}{
		"list": {apiv1.EditorialReviewer, func() error {
			_, err := api.ListEditorialSessions(ctx, connect.NewRequest(&apiv1.Void{}))
			return err
		}},
		"get": {apiv1.EditorialReviewer, func() error {
			_, err := api.GetEditorialSession(ctx, connect.NewRequest(&apiv1.GetEditorialSessionRequest{Id: live.Msg.Id}))
			return err
		}},
		"export": {apiv1.EditorialReviewer, func() error {
			_, err := api.ExportEditorialSession(ctx, connect.NewRequest(&apiv1.ExportEditorialSessionRequest{Id: rec.Id}))
			return err
		}},
		"comment": {apiv1.EditorialReviewer, func() error {
			_, err := api.SetEditorialComment(ctx, connect.NewRequest(&apiv1.SetEditorialCommentRequest{SessionId: rec.Id, MarkerId: markerID, Comment: "c"}))
			return err
		}},
		"publish": {apiv1.EditorialReviewer, func() error {
			_, err := api.SetEditorialPublish(ctx, connect.NewRequest(&apiv1.SetEditorialPublishRequest{SessionId: rec.Id, MarkerId: markerID, PublishBmm: true}))
			return err
		}},
		"title": {apiv1.EditorialReviewer, func() error {
			_, err := api.SetEditorialName(ctx, connect.NewRequest(&apiv1.SetEditorialNameRequest{SessionId: rec.Id, MarkerId: markerID, Name: "t"}))
			return err
		}},
		"details": {apiv1.EditorialEditor, func() error {
			_, err := api.SetEditorialDetails(ctx, connect.NewRequest(&apiv1.SetEditorialDetailsRequest{SessionId: rec.Id, MarkerId: markerID, Name: "t", BibleVerses: "Joh 3:16"}))
			return err
		}},
		"create meeting": {apiv1.EditorialLive, func() error {
			_, err := api.CreateEditorialSession(ctx, connect.NewRequest(&apiv1.CreateEditorialSessionRequest{Live: true, PlayoutEventId: "1013"}))
			return err
		}},
		"save (edit view)": {apiv1.EditorialLive, func() error {
			_, err := api.SaveEditorialSession(ctx, connect.NewRequest(&apiv1.SaveEditorialSessionRequest{Id: rec.Id, Title: "recording", Markers: saved.Markers}))
			return err
		}},
		"mark now": {apiv1.EditorialLive, func() error {
			// Not a live session: past the permission check it fails differently.
			_, err := api.AddEditorialMarker(ctx, connect.NewRequest(&apiv1.AddEditorialMarkerRequest{SessionId: rec.Id, At: timestamppb.Now()}))
			return err
		}},
		"refresh from Playout": {apiv1.EditorialLive, func() error {
			_, err := api.SyncEditorialSessionFromPlayout(ctx, connect.NewRequest(&apiv1.SyncEditorialSessionFromPlayoutRequest{Id: rec.Id}))
			return err
		}},
		"send to Mediabanken": {apiv1.EditorialLive, func() error {
			_, err := api.AttachEditorialRecording(ctx, connect.NewRequest(&apiv1.AttachEditorialRecordingRequest{Id: rec.Id, VXID: "VX-2"}))
			return err
		}},
		"save chapters": {apiv1.EditorialLive, func() error {
			_, err := api.ExportEditorialChapters(ctx, connect.NewRequest(&apiv1.ExportEditorialChaptersRequest{Id: rec.Id}))
			return err
		}},
		"Playout events": {apiv1.EditorialLive, func() error {
			_, err := api.ListPlayoutEvents(ctx, connect.NewRequest(&apiv1.ListPlayoutEventsRequest{}))
			return err
		}},
		"Playout import": {apiv1.EditorialLive, func() error {
			_, err := api.ImportEditorialMarkersFromPlayout(ctx, connect.NewRequest(&apiv1.ImportEditorialMarkersFromPlayoutRequest{}))
			return err
		}},
		"delete session": {apiv1.EditorialAdmin, func() error {
			// Run last for the admin (map order is random): a missing id keeps
			// the session for the other calls.
			_, err := api.DeleteEditorialSession(ctx, connect.NewRequest(&apiv1.DeleteEditorialSessionRequest{Id: "missing"}))
			return err
		}},
	}

	roles := []struct {
		name string
		json string
		role apiv1.EditorialRole
	}{
		{"none", `{}`, apiv1.EditorialNone},
		{"reviewer", `{"enabled":true}`, apiv1.EditorialReviewer},
		{"editor", `{"editor":true}`, apiv1.EditorialEditor},
		{"live", `{"live":true}`, apiv1.EditorialLive},
		{"admin", `{"admin":true}`, apiv1.EditorialAdmin},
	}
	for _, r := range roles {
		t.Run(r.name, func(t *testing.T) {
			actAs(t, r.json)
			for name, c := range calls {
				err := c.call()
				denied := connect.CodeOf(err) == connect.CodePermissionDenied
				if r.role >= c.role {
					assert.False(t, denied, "%s should be allowed for %s: %v", name, r.name, err)
				} else {
					assert.True(t, denied, "%s should be denied for %s: %v", name, r.name, err)
				}
			}
		})
	}
}
