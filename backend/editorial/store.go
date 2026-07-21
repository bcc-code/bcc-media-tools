// Package editorial is the SQLite persistence layer for the editorial approval
// tool. It stores review sessions and their markers. Timestamps are kept as
// Unix-millisecond integers to stay independent of the driver's time handling.
package editorial

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Session status values.
const (
	StatusDraft = "draft"
)

// Marker source values.
const (
	SourceManual = "manual"
	SourceImport = "import"
)

// ErrNotFound is returned when a session does not exist.
var ErrNotFound = errors.New("editorial: session not found")

// Session is a review session tied to a single Mediabanken asset.
type Session struct {
	ID        string
	VXID      string
	Title     string
	Status    string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
	// Markers is only populated by Get; List leaves it nil.
	Markers []Marker
}

// Marker is a single reviewable timestamp/chapter within a session.
type Marker struct {
	ID           string
	SortOrder    int32
	Name         string
	Contributors string
	Comment      string
	BibleVerses  string
	Type         string
	StartMS      int64
	EndMS        int64
	Publish      bool
	Source       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Store owns the SQLite connection pool.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at path, applies pragmas and runs
// the schema migration. The caller must Close the returned store.
func Open(path string) (*Store, error) {
	// WAL + busy_timeout for concurrent reads while a write is in flight;
	// foreign_keys(1) so ON DELETE CASCADE actually fires.
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("editorial: open db: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying connection pool.
func (s *Store) Close() error { return s.db.Close() }

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    vxid        TEXT NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'draft',
    created_by  TEXT NOT NULL,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS markers (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    sort_order  INTEGER NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    contributors TEXT NOT NULL DEFAULT '',
    comment     TEXT NOT NULL DEFAULT '',
    bible_verses TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL DEFAULT '',
    start_ms    INTEGER NOT NULL,
    end_ms      INTEGER NOT NULL,
    publish     INTEGER NOT NULL DEFAULT 0,
    source      TEXT NOT NULL DEFAULT 'manual',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_markers_session ON markers(session_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_sessions_vxid   ON sessions(vxid);
`

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("editorial: migrate: %w", err)
	}
	// Additive column migrations for tables that predate them. CREATE TABLE IF
	// NOT EXISTS above won't add columns to an existing markers table, so add
	// them here; ignore the "duplicate column" error when already present.
	if err := s.addColumnIfMissing(ctx, "markers", "contributors", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "markers", "comment", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "markers", "bible_verses", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	return nil
}

// addColumnIfMissing runs ALTER TABLE ADD COLUMN only when the column is absent,
// making the migration idempotent across restarts.
func (s *Store) addColumnIfMissing(ctx context.Context, table, column, def string) error {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("editorial: inspect %s: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("editorial: scan %s columns: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx,
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, def)); err != nil {
		return fmt.Errorf("editorial: add column %s.%s: %w", table, column, err)
	}
	return nil
}

// millis helpers convert between time.Time and the epoch-millis stored in the
// DB. The zero time maps to 0.
func toMillis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func fromMillis(ms int64) time.Time {
	return time.UnixMilli(ms).UTC()
}

// CreateSession inserts a new draft session for the given asset and returns it.
// The id, timestamps and status are assigned by the store.
func (s *Store) CreateSession(ctx context.Context, vxid, title, createdBy string) (*Session, error) {
	now := time.Now().UTC()
	sess := &Session{
		ID:        uuid.NewString(),
		VXID:      vxid,
		Title:     title,
		Status:    StatusDraft,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, vxid, title, status, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.VXID, sess.Title, sess.Status, sess.CreatedBy,
		toMillis(sess.CreatedAt), toMillis(sess.UpdatedAt),
	)
	if err != nil {
		return nil, fmt.Errorf("editorial: create session: %w", err)
	}
	return sess, nil
}

// ListSessions returns all sessions (newest first) without their markers.
func (s *Store) ListSessions(ctx context.Context) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, vxid, title, status, created_by, created_at, updated_at
		 FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("editorial: list sessions: %w", err)
	}
	defer rows.Close()

	var out []Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sess)
	}
	return out, rows.Err()
}

// GetSession returns a session with its markers ordered by sort_order.
// Returns ErrNotFound if the session does not exist.
func (s *Store) GetSession(ctx context.Context, id string) (*Session, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, vxid, title, status, created_by, created_at, updated_at
		 FROM sessions WHERE id = ?`, id)
	sess, err := scanSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("editorial: get session: %w", err)
	}

	markers, err := s.markersForSession(ctx, id)
	if err != nil {
		return nil, err
	}
	sess.Markers = markers
	return sess, nil
}

func (s *Store) markersForSession(ctx context.Context, sessionID string) ([]Marker, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, sort_order, name, contributors, comment, bible_verses, type, start_ms, end_ms, publish, source, created_at, updated_at
		 FROM markers WHERE session_id = ? ORDER BY sort_order ASC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("editorial: list markers: %w", err)
	}
	defer rows.Close()

	var out []Marker
	for rows.Next() {
		var m Marker
		var createdAt, updatedAt int64
		if err := rows.Scan(&m.ID, &m.SortOrder, &m.Name, &m.Contributors, &m.Comment, &m.BibleVerses, &m.Type, &m.StartMS, &m.EndMS,
			&m.Publish, &m.Source, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("editorial: scan marker: %w", err)
		}
		m.CreatedAt = fromMillis(createdAt)
		m.UpdatedAt = fromMillis(updatedAt)
		out = append(out, m)
	}
	return out, rows.Err()
}

// SaveSession updates the session title and fully replaces its markers in a
// single transaction. Markers with an empty ID are treated as new and get a
// generated id; sort_order is assigned from slice position so the caller
// controls ordering. Returns the reloaded session with its markers.
func (s *Store) SaveSession(ctx context.Context, id, title string, markers []Marker) (*Session, error) {
	now := time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("editorial: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`UPDATE sessions SET title = ?, updated_at = ? WHERE id = ?`,
		title, toMillis(now), id)
	if err != nil {
		return nil, fmt.Errorf("editorial: update session: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM markers WHERE session_id = ?`, id); err != nil {
		return nil, fmt.Errorf("editorial: clear markers: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO markers (id, session_id, sort_order, name, contributors, comment, bible_verses, type, start_ms, end_ms, publish, source, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, fmt.Errorf("editorial: prepare marker insert: %w", err)
	}
	defer stmt.Close()

	for i, m := range markers {
		mid := m.ID
		if mid == "" {
			mid = uuid.NewString()
		}
		source := m.Source
		if source == "" {
			source = SourceManual
		}
		if _, err := stmt.ExecContext(ctx,
			mid, id, int32(i), m.Name, m.Contributors, m.Comment, m.BibleVerses, m.Type, m.StartMS, m.EndMS, m.Publish, source,
			toMillis(now), toMillis(now),
		); err != nil {
			return nil, fmt.Errorf("editorial: insert marker: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("editorial: commit: %w", err)
	}
	return s.GetSession(ctx, id)
}

// SetPublish updates a single marker's publish flag without touching anything
// else. This is the write path for reviewers who may accept/reject but not edit
// markers (the simple view). Returns ErrNotFound if the marker does not exist in
// the session.
func (s *Store) SetPublish(ctx context.Context, sessionID, markerID string, publish bool) error {
	now := time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("editorial: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`UPDATE markers SET publish = ?, updated_at = ? WHERE id = ? AND session_id = ?`,
		publish, toMillis(now), markerID, sessionID)
	if err != nil {
		return fmt.Errorf("editorial: set publish: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE sessions SET updated_at = ? WHERE id = ?`, toMillis(now), sessionID); err != nil {
		return fmt.Errorf("editorial: touch session: %w", err)
	}

	return tx.Commit()
}

// DeleteSession removes a session and (via cascade) its markers. Returns
// ErrNotFound if the session did not exist.
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("editorial: delete session: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanSession(sc scanner) (*Session, error) {
	var sess Session
	var createdAt, updatedAt int64
	if err := sc.Scan(&sess.ID, &sess.VXID, &sess.Title, &sess.Status, &sess.CreatedBy,
		&createdAt, &updatedAt); err != nil {
		return nil, err
	}
	sess.CreatedAt = fromMillis(createdAt)
	sess.UpdatedAt = fromMillis(updatedAt)
	return &sess, nil
}
