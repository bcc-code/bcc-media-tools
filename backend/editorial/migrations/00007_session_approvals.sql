-- +goose Up
-- Sign-offs on the whole session: redactional (content) and technical. _by and
-- _at are who last approved or cleared it and when.
ALTER TABLE sessions ADD COLUMN redactional_approved INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN redactional_approved_by TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN redactional_approved_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN technical_approved INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN technical_approved_by TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN technical_approved_at INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE sessions DROP COLUMN technical_approved_at;
ALTER TABLE sessions DROP COLUMN technical_approved_by;
ALTER TABLE sessions DROP COLUMN technical_approved;
ALTER TABLE sessions DROP COLUMN redactional_approved_at;
ALTER TABLE sessions DROP COLUMN redactional_approved_by;
ALTER TABLE sessions DROP COLUMN redactional_approved;
