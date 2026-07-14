package editorial

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "editorial_test.db")
	s, err := Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestCreateAndGetSession(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	created, err := s.CreateSession(ctx, "VX-123", "Sunday stream", "editor@bcc.media")
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, StatusDraft, created.Status)
	assert.False(t, created.CreatedAt.IsZero())

	got, err := s.GetSession(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "VX-123", got.VXID)
	assert.Equal(t, "Sunday stream", got.Title)
	assert.Equal(t, "editor@bcc.media", got.CreatedBy)
	assert.Empty(t, got.Markers)
}

func TestGetSessionNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetSession(context.Background(), "does-not-exist")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListSessionsNewestFirst(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	a, err := s.CreateSession(ctx, "VX-1", "first", "e@bcc.media")
	require.NoError(t, err)
	b, err := s.CreateSession(ctx, "VX-2", "second", "e@bcc.media")
	require.NoError(t, err)

	list, err := s.ListSessions(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	// Both exist; ordering is by created_at DESC (ties may collapse on fast
	// clocks, so only assert set membership on ids).
	ids := []string{list[0].ID, list[1].ID}
	assert.Contains(t, ids, a.ID)
	assert.Contains(t, ids, b.ID)
	// List must not populate markers.
	assert.Nil(t, list[0].Markers)
}

func TestSaveSessionReplacesMarkers(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	sess, err := s.CreateSession(ctx, "VX-9", "stream", "e@bcc.media")
	require.NoError(t, err)

	// First save: two new markers (empty IDs → generated).
	saved, err := s.SaveSession(ctx, sess.ID, "renamed", []Marker{
		{Name: "Speaker A", Type: "appell", StartMS: 1000, EndMS: 5000, Publish: true, Source: SourceImport},
		{Name: "Song", Type: "sang", StartMS: 6000, EndMS: 9000},
	})
	require.NoError(t, err)
	assert.Equal(t, "renamed", saved.Title)
	require.Len(t, saved.Markers, 2)
	assert.NotEmpty(t, saved.Markers[0].ID)
	assert.Equal(t, int32(0), saved.Markers[0].SortOrder)
	assert.Equal(t, int32(1), saved.Markers[1].SortOrder)
	assert.True(t, saved.Markers[0].Publish)
	assert.Equal(t, SourceImport, saved.Markers[0].Source)
	// Missing source defaults to manual.
	assert.Equal(t, SourceManual, saved.Markers[1].Source)

	// Second save with a single marker fully replaces the previous set.
	saved2, err := s.SaveSession(ctx, sess.ID, "renamed", []Marker{
		{Name: "Only one", Type: "tale", StartMS: 0, EndMS: 100},
	})
	require.NoError(t, err)
	require.Len(t, saved2.Markers, 1)
	assert.Equal(t, "Only one", saved2.Markers[0].Name)
}

func TestSaveSessionPreservesGivenOrder(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	sess, err := s.CreateSession(ctx, "VX-9", "stream", "e@bcc.media")
	require.NoError(t, err)

	saved, err := s.SaveSession(ctx, sess.ID, "t", []Marker{
		{Name: "third"},
		{Name: "first"},
		{Name: "second"},
	})
	require.NoError(t, err)
	require.Len(t, saved.Markers, 3)
	assert.Equal(t, "third", saved.Markers[0].Name)
	assert.Equal(t, "first", saved.Markers[1].Name)
	assert.Equal(t, "second", saved.Markers[2].Name)
}

func TestSaveSessionNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.SaveSession(context.Background(), "nope", "t", nil)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSetPublish(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	sess, err := s.CreateSession(ctx, "VX-9", "stream", "e@bcc.media")
	require.NoError(t, err)
	saved, err := s.SaveSession(ctx, sess.ID, "t", []Marker{
		{Name: "m1", StartMS: 0, EndMS: 1},
		{Name: "m2", StartMS: 2, EndMS: 3, Publish: true},
	})
	require.NoError(t, err)
	m1ID := saved.Markers[0].ID

	require.NoError(t, s.SetPublish(ctx, sess.ID, m1ID, true))

	got, err := s.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	assert.True(t, got.Markers[0].Publish)
	// The other marker is untouched.
	assert.True(t, got.Markers[1].Publish)
	assert.Equal(t, "m1", got.Markers[0].Name)
}

func TestSetPublishNotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	sess, err := s.CreateSession(ctx, "VX-9", "stream", "e@bcc.media")
	require.NoError(t, err)
	assert.ErrorIs(t, s.SetPublish(ctx, sess.ID, "no-such-marker", true), ErrNotFound)
}

func TestDeleteSessionCascadesMarkers(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	sess, err := s.CreateSession(ctx, "VX-9", "stream", "e@bcc.media")
	require.NoError(t, err)
	_, err = s.SaveSession(ctx, sess.ID, "t", []Marker{{Name: "m", StartMS: 0, EndMS: 1}})
	require.NoError(t, err)

	require.NoError(t, s.DeleteSession(ctx, sess.ID))

	_, err = s.GetSession(ctx, sess.ID)
	assert.ErrorIs(t, err, ErrNotFound)

	// Marker rows are gone too (ON DELETE CASCADE).
	var count int
	require.NoError(t, s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM markers WHERE session_id = ?`, sess.ID).Scan(&count))
	assert.Equal(t, 0, count)
}

func TestDeleteSessionNotFound(t *testing.T) {
	s := newTestStore(t)
	assert.ErrorIs(t, s.DeleteSession(context.Background(), "nope"), ErrNotFound)
}
