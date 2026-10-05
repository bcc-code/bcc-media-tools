package editorial

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// prodSchema is the schema of databases created before goose was introduced:
// the original tables with contributors/comment/bible_verses/publish_bmm/
// publish_bcc added ad hoc, and the unused legacy publish column.
const prodSchema = `
CREATE TABLE sessions (id TEXT PRIMARY KEY, vxid TEXT NOT NULL, title TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'draft', created_by TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL);
CREATE TABLE markers (id TEXT PRIMARY KEY, session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE, sort_order INTEGER NOT NULL, name TEXT NOT NULL DEFAULT '', type TEXT NOT NULL DEFAULT '', start_ms INTEGER NOT NULL, end_ms INTEGER NOT NULL, publish INTEGER NOT NULL DEFAULT 0, source TEXT NOT NULL DEFAULT 'manual', created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, contributors TEXT NOT NULL DEFAULT '', comment TEXT NOT NULL DEFAULT '', bible_verses TEXT NOT NULL DEFAULT '', publish_bmm INTEGER NOT NULL DEFAULT 0, publish_bcc INTEGER NOT NULL DEFAULT 0);
CREATE INDEX idx_markers_session ON markers(session_id, sort_order);
CREATE INDEX idx_sessions_vxid   ON sessions(vxid);
INSERT INTO sessions VALUES ('s1', 'VX-1', 't', 'draft', 'e@bcc.media', 1, 1);
INSERT INTO markers (id, session_id, sort_order, name, comment, start_ms, end_ms, publish_bmm, created_at, updated_at) VALUES ('m1', 's1', 0, 'old', 'kept', 0, 1, 1, 1, 1);`

func TestOpenAdoptsPreGooseDatabase(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "prod.db")

	raw, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = raw.Exec(prodSchema)
	require.NoError(t, err)
	require.NoError(t, raw.Close())

	db, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	q := New(db)

	markers, err := q.ListMarkersForSession(ctx, "s1")
	require.NoError(t, err)
	require.Len(t, markers, 1)
	assert.Equal(t, "old", markers[0].Name)
	assert.Equal(t, "kept", markers[0].Comment)
	assert.True(t, markers[0].PublishBmm)

	require.NoError(t, q.InsertMarker(ctx, InsertMarkerParams{
		ID: "m2", SessionID: "s1", SortOrder: 1, Name: "new", Contributors: "A",
		Comment: "c", BibleVerses: "John 3:16", PublishBmm: true, PublishBcc: true,
		Source: SourceManual, CreatedAt: 2, UpdatedAt: 2,
	}))
	markers, err = q.ListMarkersForSession(ctx, "s1")
	require.NoError(t, err)
	require.Len(t, markers, 2)
	assert.Equal(t, "John 3:16", markers[1].BibleVerses)
	assert.True(t, markers[1].PublishBcc)
}

func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "e.db")
	db, err := Open(path)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	db, err = Open(path)
	require.NoError(t, err)
	require.NoError(t, db.Close())
}
