-- name: CreateSession :exec
INSERT INTO sessions (id, vxid, title, status, created_by, created_at, updated_at, kind, playout_event_id, clock_start_ms)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListSessions :many
SELECT * FROM sessions
ORDER BY created_at DESC;

-- name: GetSession :one
SELECT * FROM sessions
WHERE id = ?;

-- name: UpdateSessionTitle :execrows
UPDATE sessions SET title = ?, updated_at = ?
WHERE id = ?;

-- name: TouchSession :exec
UPDATE sessions SET updated_at = ?
WHERE id = ?;

-- name: DeleteSession :execrows
DELETE FROM sessions
WHERE id = ?;

-- name: ListMarkersForSession :many
SELECT * FROM markers
WHERE session_id = ?
ORDER BY sort_order ASC;

-- name: DeleteMarkersForSession :exec
DELETE FROM markers
WHERE session_id = ?;

-- name: InsertMarker :exec
INSERT INTO markers (id, session_id, sort_order, name, contributors, comment, bible_verses, type, start_ms, end_ms, publish_bmm, publish_bcc, source, created_at, updated_at, playout_key, playout_synced)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: SetMarkerPublish :execrows
UPDATE markers SET publish_bmm = ?, publish_bcc = ?, updated_at = ?
WHERE id = ? AND session_id = ?;

-- name: SetMarkerComment :execrows
UPDATE markers SET comment = ?, updated_at = ?
WHERE id = ? AND session_id = ?;

-- name: SetMarkerName :execrows
-- A hand-edited title stops Playout from refreshing the row.
UPDATE markers SET name = ?, playout_synced = 0, updated_at = ?
WHERE id = ? AND session_id = ?;

-- name: SetPlayoutWindow :execrows
UPDATE sessions SET playout_event_id = ?, recording_start_ms = ?, recording_end_ms = ?, updated_at = ?
WHERE id = ?;

-- name: AttachRecording :execrows
UPDATE sessions SET vxid = ?, kind = ?, updated_at = ?
WHERE id = ?;

-- name: ListDismissedPlayoutKeys :many
SELECT playout_key FROM playout_dismissed
WHERE session_id = ?;

-- name: DismissPlayoutKey :exec
INSERT OR IGNORE INTO playout_dismissed (session_id, playout_key)
VALUES (?, ?);
