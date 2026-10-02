// Package editorial is the SQLite persistence layer for the editorial approval
// tool. It stores review sessions and their markers. Timestamps are kept as
// Unix-millisecond integers to stay independent of the driver's time handling.
//
// The schema lives in goose migrations under migrations/, applied by Open.
// Queries live in query.sql; run `make sqlc` to regenerate the Go code after
// changing either.
package editorial

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Session status values.
const (
	StatusDraft = "draft"
)

// Marker source values.
const (
	SourceManual = "manual"
	SourceImport = "import"
)

// Open opens (or creates) the SQLite database at path, applies pragmas and runs
// the goose migrations. The caller must Close the returned db.
func Open(path string) (*sql.DB, error) {
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

	if err := migrate(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("editorial: migrations fs: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, fsys)
	if err != nil {
		return fmt.Errorf("editorial: migrate: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("editorial: migrate: %w", err)
	}
	return nil
}
