-- +goose Up
-- A marker's contributors in order. person_id is the BCC Members personID of
-- a picked or Playout-linked person; NULL when the name was typed in.
CREATE TABLE marker_contributors (
    marker_id TEXT NOT NULL REFERENCES markers(id) ON DELETE CASCADE,
    position  INTEGER NOT NULL,
    name      TEXT NOT NULL,
    person_id TEXT,
    PRIMARY KEY (marker_id, position)
);

-- The old ", "-separated names become rows, without IDs: n counts the items,
-- so positions keep their order (with gaps where an item was empty).
INSERT INTO marker_contributors (marker_id, position, name)
WITH RECURSIVE split (marker_id, n, item, rest) AS (
    SELECT id, -1, '', contributors || ',' FROM markers
    UNION ALL
    SELECT marker_id, n + 1, substr(rest, 1, instr(rest, ',') - 1), substr(rest, instr(rest, ',') + 1)
    FROM split
    WHERE rest <> ''
)
SELECT marker_id, n, trim(item, ' ' || char(9, 10, 13)) FROM split
WHERE n >= 0 AND trim(item, ' ' || char(9, 10, 13)) <> '';

ALTER TABLE markers DROP COLUMN contributors;

-- +goose Down
ALTER TABLE markers ADD COLUMN contributors TEXT NOT NULL DEFAULT '';
UPDATE markers SET contributors = coalesce(
    (SELECT group_concat(name, ', ' ORDER BY position) FROM marker_contributors WHERE marker_id = markers.id),
    ''
);
DROP TABLE marker_contributors;
