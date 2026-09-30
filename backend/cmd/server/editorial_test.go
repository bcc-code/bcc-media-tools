package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestEditorialAPI returns an API backed by a fresh migrated DB, with the
// caller authenticated as an editorial admin.
func newTestEditorialAPI(t *testing.T) *EditorialAPI {
	t.Helper()
	dir := t.TempDir()

	db, err := editorial.Open(filepath.Join(dir, "editorial_test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	permsPath := filepath.Join(dir, "permissions.json")
	require.NoError(t, os.WriteFile(permsPath, []byte(`{"editor@bcc.media":{"bmm":{},"editorial":{"admin":true}}}`), 0644))
	oldPerms := PermissionsFile
	PermissionsFile = permsPath
	t.Cleanup(func() { PermissionsFile = oldPerms })
	t.Setenv("DEBUG_AUTH_EMAIL", "editor@bcc.media")

	return NewEditorialAPI(db, nil, nil)
}

func createTestSession(t *testing.T, api *EditorialAPI, vxID, title string) *apiv1.EditorialSession {
	t.Helper()
	res, err := api.CreateEditorialSession(context.Background(), connect.NewRequest(&apiv1.CreateEditorialSessionRequest{VXID: vxID, Title: title}))
	require.NoError(t, err)
	return res.Msg
}

func saveTestSession(t *testing.T, api *EditorialAPI, id, title string, markers []*apiv1.EditorialMarker) *apiv1.EditorialSession {
	t.Helper()
	res, err := api.SaveEditorialSession(context.Background(), connect.NewRequest(&apiv1.SaveEditorialSessionRequest{Id: id, Title: title, Markers: markers}))
	require.NoError(t, err)
	return res.Msg
}

func requireNotFound(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestEditorialCreateAndGetSession(t *testing.T) {
	api := newTestEditorialAPI(t)

	created := createTestSession(t, api, "VX-123", "Sunday stream")
	assert.NotEmpty(t, created.Id)
	assert.Equal(t, editorial.StatusDraft, created.Status)
	assert.False(t, created.CreatedAt.AsTime().IsZero())

	got, err := api.loadSession(context.Background(), created.Id)
	require.NoError(t, err)
	assert.Equal(t, "VX-123", got.VXID)
	assert.Equal(t, "Sunday stream", got.Title)
	assert.Equal(t, "editor@bcc.media", got.CreatedBy)
	assert.Empty(t, got.Markers)
}

func TestEditorialGetSessionNotFound(t *testing.T) {
	api := newTestEditorialAPI(t)
	_, err := api.loadSession(context.Background(), "does-not-exist")
	assert.ErrorIs(t, err, errEditorialNotFound)
}

func TestEditorialListSessions(t *testing.T) {
	api := newTestEditorialAPI(t)

	a := createTestSession(t, api, "VX-1", "first")
	b := createTestSession(t, api, "VX-2", "second")

	res, err := api.ListEditorialSessions(context.Background(), connect.NewRequest(&apiv1.Void{}))
	require.NoError(t, err)
	list := res.Msg.Sessions
	require.Len(t, list, 2)
	// Ordering is by created_at DESC (ties may collapse on fast clocks, so only
	// assert set membership on ids).
	ids := []string{list[0].Id, list[1].Id}
	assert.Contains(t, ids, a.Id)
	assert.Contains(t, ids, b.Id)
	// List must not include markers.
	assert.Nil(t, list[0].Markers)
}

func TestEditorialSaveSessionReplacesMarkers(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")

	// First save: two new markers (empty IDs → generated).
	saved := saveTestSession(t, api, sess.Id, "renamed", []*apiv1.EditorialMarker{
		{Name: "Speaker A", Contributors: "Alice, Bob", Comment: "double-check timing", BibleVerses: "John 3:16; Rom 8:1-4", Type: "tale", StartMs: 1000, EndMs: 5000, PublishBmm: true, Source: editorial.SourceImport},
		{Name: "Song", Type: "sang", StartMs: 6000, EndMs: 9000},
	})
	assert.Equal(t, "renamed", saved.Title)
	require.Len(t, saved.Markers, 2)
	assert.NotEmpty(t, saved.Markers[0].Id)
	assert.Equal(t, int32(0), saved.Markers[0].SortOrder)
	assert.Equal(t, int32(1), saved.Markers[1].SortOrder)
	assert.Equal(t, "Alice, Bob", saved.Markers[0].Contributors)
	assert.Equal(t, "double-check timing", saved.Markers[0].Comment)
	assert.Equal(t, "John 3:16; Rom 8:1-4", saved.Markers[0].BibleVerses)
	assert.True(t, saved.Markers[0].PublishBmm)
	assert.False(t, saved.Markers[0].PublishBcc)
	assert.Equal(t, editorial.SourceImport, saved.Markers[0].Source)
	// Missing source defaults to manual.
	assert.Equal(t, editorial.SourceManual, saved.Markers[1].Source)

	// Second save with a single marker fully replaces the previous set.
	saved2 := saveTestSession(t, api, sess.Id, "renamed", []*apiv1.EditorialMarker{
		{Name: "Only one", Type: "tale", StartMs: 0, EndMs: 100},
	})
	require.Len(t, saved2.Markers, 1)
	assert.Equal(t, "Only one", saved2.Markers[0].Name)
}

func TestEditorialSaveSessionPreservesGivenOrder(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")

	saved := saveTestSession(t, api, sess.Id, "t", []*apiv1.EditorialMarker{
		{Name: "third"},
		{Name: "first"},
		{Name: "second"},
	})
	require.Len(t, saved.Markers, 3)
	assert.Equal(t, "third", saved.Markers[0].Name)
	assert.Equal(t, "first", saved.Markers[1].Name)
	assert.Equal(t, "second", saved.Markers[2].Name)
}

func TestEditorialSaveSessionNotFound(t *testing.T) {
	api := newTestEditorialAPI(t)
	_, err := api.SaveEditorialSession(context.Background(), connect.NewRequest(&apiv1.SaveEditorialSessionRequest{Id: "nope", Title: "t"}))
	requireNotFound(t, err)
}

func TestEditorialSetPublish(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	sess := createTestSession(t, api, "VX-9", "stream")
	saved := saveTestSession(t, api, sess.Id, "t", []*apiv1.EditorialMarker{
		{Name: "m1", StartMs: 0, EndMs: 1},
		{Name: "m2", StartMs: 2, EndMs: 3, PublishBmm: true, PublishBcc: true},
	})

	_, err := api.SetEditorialPublish(ctx, connect.NewRequest(&apiv1.SetEditorialPublishRequest{
		SessionId: sess.Id, MarkerId: saved.Markers[0].Id, PublishBmm: true, PublishBcc: false,
	}))
	require.NoError(t, err)

	got, err := api.loadSession(ctx, sess.Id)
	require.NoError(t, err)
	assert.True(t, got.Markers[0].PublishBmm)
	assert.False(t, got.Markers[0].PublishBcc)
	// The other marker is untouched.
	assert.True(t, got.Markers[1].PublishBmm)
	assert.True(t, got.Markers[1].PublishBcc)
	assert.Equal(t, "m1", got.Markers[0].Name)
}

func TestEditorialSetPublishNotFound(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")
	_, err := api.SetEditorialPublish(context.Background(), connect.NewRequest(&apiv1.SetEditorialPublishRequest{
		SessionId: sess.Id, MarkerId: "no-such-marker", PublishBmm: true, PublishBcc: true,
	}))
	requireNotFound(t, err)
}

func TestEditorialSetComment(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	sess := createTestSession(t, api, "VX-9", "stream")
	saved := saveTestSession(t, api, sess.Id, "t", []*apiv1.EditorialMarker{
		{Name: "m1", StartMs: 0, EndMs: 1, PublishBmm: true},
		{Name: "m2", StartMs: 2, EndMs: 3, Comment: "keep"},
	})

	_, err := api.SetEditorialComment(ctx, connect.NewRequest(&apiv1.SetEditorialCommentRequest{
		SessionId: sess.Id, MarkerId: saved.Markers[0].Id, Comment: "check audio",
	}))
	require.NoError(t, err)

	got, err := api.loadSession(ctx, sess.Id)
	require.NoError(t, err)
	assert.Equal(t, "check audio", got.Markers[0].Comment)
	// Other fields and markers are untouched.
	assert.Equal(t, "m1", got.Markers[0].Name)
	assert.True(t, got.Markers[0].PublishBmm)
	assert.Equal(t, "keep", got.Markers[1].Comment)
}

func TestEditorialSetCommentNotFound(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")
	_, err := api.SetEditorialComment(context.Background(), connect.NewRequest(&apiv1.SetEditorialCommentRequest{
		SessionId: sess.Id, MarkerId: "no-such-marker", Comment: "x",
	}))
	requireNotFound(t, err)
}

func TestEditorialSetName(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	sess := createTestSession(t, api, "VX-9", "stream")
	saved := saveTestSession(t, api, sess.Id, "t", []*apiv1.EditorialMarker{
		{Name: "old", Comment: "c", StartMs: 0, EndMs: 1},
		{Name: "other", StartMs: 2, EndMs: 3},
	})

	_, err := api.SetEditorialName(ctx, connect.NewRequest(&apiv1.SetEditorialNameRequest{
		SessionId: sess.Id, MarkerId: saved.Markers[0].Id, Name: "new",
	}))
	require.NoError(t, err)

	got, err := api.loadSession(ctx, sess.Id)
	require.NoError(t, err)
	assert.Equal(t, "new", got.Markers[0].Name)
	assert.Equal(t, "c", got.Markers[0].Comment)
	assert.Equal(t, "other", got.Markers[1].Name)
}

func TestEditorialSetNameNotFound(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", "stream")
	_, err := api.SetEditorialName(context.Background(), connect.NewRequest(&apiv1.SetEditorialNameRequest{
		SessionId: sess.Id, MarkerId: "no-such-marker", Name: "x",
	}))
	requireNotFound(t, err)
}

func TestEditorialDeleteSessionCascadesMarkers(t *testing.T) {
	api := newTestEditorialAPI(t)
	ctx := context.Background()
	sess := createTestSession(t, api, "VX-9", "stream")
	saveTestSession(t, api, sess.Id, "t", []*apiv1.EditorialMarker{{Name: "m", StartMs: 0, EndMs: 1}})

	_, err := api.DeleteEditorialSession(ctx, connect.NewRequest(&apiv1.DeleteEditorialSessionRequest{Id: sess.Id}))
	require.NoError(t, err)

	_, err = api.loadSession(ctx, sess.Id)
	assert.ErrorIs(t, err, errEditorialNotFound)

	// Marker rows are gone too (ON DELETE CASCADE).
	markers, err := api.queries.ListMarkersForSession(ctx, sess.Id)
	require.NoError(t, err)
	assert.Empty(t, markers)
}

func TestEditorialDeleteSessionNotFound(t *testing.T) {
	api := newTestEditorialAPI(t)
	_, err := api.DeleteEditorialSession(context.Background(), connect.NewRequest(&apiv1.DeleteEditorialSessionRequest{Id: "nope"}))
	requireNotFound(t, err)
}
