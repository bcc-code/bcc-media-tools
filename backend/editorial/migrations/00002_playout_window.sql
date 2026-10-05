-- +goose Up
-- Remembered from the last Playout import so a hand-corrected recording
-- window survives a reload and is reused by the next import.
ALTER TABLE sessions ADD COLUMN playout_event_id TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN recording_start_ms INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN recording_end_ms INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE sessions DROP COLUMN recording_end_ms;
ALTER TABLE sessions DROP COLUMN recording_start_ms;
ALTER TABLE sessions DROP COLUMN playout_event_id;
