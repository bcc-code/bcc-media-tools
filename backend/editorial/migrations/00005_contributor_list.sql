-- +goose Up
-- A marker's contributors as JSON, [{"name": ..., "personId": ...}], so a name
-- picked from BCC Members keeps its personID. contributors stays the names
-- joined with ", ". Rows from before this column read their contributors from
-- that string instead.
ALTER TABLE markers ADD COLUMN contributor_list TEXT NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE markers DROP COLUMN contributor_list;
