package editorial

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
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
INSERT INTO markers (id, session_id, sort_order, name, contributors, comment, start_ms, end_ms, publish_bmm, created_at, updated_at) VALUES ('m1', 's1', 0, 'old', ' A ,B,, C', 'kept', 0, 1, 1, 1, 1);
INSERT INTO markers (id, session_id, sort_order, name, contributors, start_ms, end_ms, created_at, updated_at) VALUES ('m0', 's1', 1, 'none', '', 1, 2, 1, 1);`

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
	require.Len(t, markers, 2)
	assert.Equal(t, "old", markers[0].Name)
	assert.Equal(t, "kept", markers[0].Comment)
	assert.True(t, markers[0].PublishBmm)

	// The contributors string became rows, in order and without IDs.
	assert.Equal(t, []string{"A", "B", "C"}, contributorNames(t, db, "m1"))
	assert.Empty(t, contributorNames(t, db, "m0"))
	var left int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM pragma_table_info('markers') WHERE name = 'contributors'`).Scan(&left))
	assert.Zero(t, left, "the contributors column is dropped")

	require.NoError(t, q.InsertMarker(ctx, InsertMarkerParams{
		ID: "m2", SessionID: "s1", SortOrder: 2, Name: "new",
		Comment: "c", BibleVerses: "John 3:16", PublishBmm: true, PublishBcc: true,
		Source: SourceManual, CreatedAt: 2, UpdatedAt: 2,
	}))
	markers, err = q.ListMarkersForSession(ctx, "s1")
	require.NoError(t, err)
	require.Len(t, markers, 3)
	assert.Equal(t, "John 3:16", markers[2].BibleVerses)
	assert.True(t, markers[2].PublishBcc)

	require.NoError(t, q.DeleteMarkersForSession(ctx, "s1"))
	assert.Empty(t, contributorNames(t, db, "m1"), "contributors go with their marker")
}

// Rolling back the contributors table puts the names back into the
// contributors string; IDs are lost.
func TestContributorsMigrationRollsBack(t *testing.T) {
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "e.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	q := New(db)

	_, err = db.Exec(`INSERT INTO sessions (id, vxid, created_by, created_at, updated_at) VALUES ('s1', 'VX-1', 'e@bcc.media', 1, 1)`)
	require.NoError(t, err)
	for _, id := range []string{"m1", "m2"} {
		require.NoError(t, q.InsertMarker(ctx, InsertMarkerParams{ID: id, SessionID: "s1", Source: SourceManual}))
	}
	for i, name := range []string{"B", "A"} {
		require.NoError(t, q.InsertMarkerContributor(ctx, InsertMarkerContributorParams{
			MarkerID: "m1", Position: int64(i), Name: name, MemberUID: sql.NullString{String: "uid-" + name, Valid: true},
		}))
	}

	fsys, err := fs.Sub(migrations, "migrations")
	require.NoError(t, err)
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, fsys)
	require.NoError(t, err)
	_, err = provider.DownTo(ctx, 5) // back to before 00006_marker_contributors
	require.NoError(t, err)
	var got []string
	rows, err := db.Query(`SELECT contributors FROM markers ORDER BY id`)
	require.NoError(t, err)
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		got = append(got, s)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"B, A", ""}, got)

	_, err = provider.Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"B", "A"}, contributorNames(t, db, "m1"))
	var linked int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM marker_contributors WHERE member_uid IS NOT NULL`).Scan(&linked))
	assert.Zero(t, linked)
}

func contributorNames(t *testing.T, db *sql.DB, markerID string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM marker_contributors WHERE marker_id = ? ORDER BY position`, markerID)
	require.NoError(t, err)
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	return names
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
