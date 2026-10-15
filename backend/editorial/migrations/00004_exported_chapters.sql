-- +goose Up
-- Subclips this tool wrote to an asset in Mediabanken, by their Vidispine
-- group uuid. A re-export removes exactly these before writing the current
-- markers, leaving chapters made by hand in Cantemo alone.
CREATE TABLE IF NOT EXISTS exported_chapters (
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    vxid        TEXT NOT NULL,
    uuid        TEXT NOT NULL,
    start_tc    TEXT NOT NULL,
    end_tc      TEXT NOT NULL,
    exported_at INTEGER NOT NULL,
    PRIMARY KEY (session_id, uuid)
);

-- +goose Down
DROP TABLE IF EXISTS exported_chapters;
