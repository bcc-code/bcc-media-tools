-- +goose Up
-- Live sessions are logged in real time before the recording exists in
-- Mediabanken: no VXID yet, and marker offsets are measured from
-- clock_start_ms (the wall-clock moment the session was created) until the
-- recording is attached.
ALTER TABLE sessions ADD COLUMN kind TEXT NOT NULL DEFAULT 'recording';
ALTER TABLE sessions ADD COLUMN clock_start_ms INTEGER NOT NULL DEFAULT 0;

-- The Playout manifest entry a marker was created or filled from, so a re-sync
-- recognises it. playout_synced stays 1 until an editor changes one of the
-- fields Playout fills; until then each sync refreshes them.
ALTER TABLE markers ADD COLUMN playout_key TEXT NOT NULL DEFAULT '';
ALTER TABLE markers ADD COLUMN playout_synced INTEGER NOT NULL DEFAULT 0;

-- Playout entries whose row an editor removed; a re-sync must not bring them back.
CREATE TABLE IF NOT EXISTS playout_dismissed (
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    playout_key TEXT NOT NULL,
    PRIMARY KEY (session_id, playout_key)
);

-- +goose Down
DROP TABLE IF EXISTS playout_dismissed;
ALTER TABLE markers DROP COLUMN playout_synced;
ALTER TABLE markers DROP COLUMN playout_key;
ALTER TABLE sessions DROP COLUMN clock_start_ms;
ALTER TABLE sessions DROP COLUMN kind;
