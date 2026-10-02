-- +goose Up
-- IF NOT EXISTS so databases created before goose was introduced adopt this
-- migration without error. Those already have this schema (plus an unused
-- legacy markers.publish column).
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
    publish_bmm INTEGER NOT NULL DEFAULT 0,
    publish_bcc INTEGER NOT NULL DEFAULT 0,
    source      TEXT NOT NULL DEFAULT 'manual',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_markers_session ON markers(session_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_sessions_vxid   ON sessions(vxid);

-- +goose Down
DROP INDEX IF EXISTS idx_sessions_vxid;
DROP INDEX IF EXISTS idx_markers_session;
DROP TABLE IF EXISTS markers;
DROP TABLE IF EXISTS sessions;
