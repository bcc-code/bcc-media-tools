-- name: CreateSession :exec
INSERT INTO sessions (id, vxid, title, status, created_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

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
INSERT INTO markers (id, session_id, sort_order, name, type, start_ms, end_ms, publish, source, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: SetMarkerPublish :execrows
UPDATE markers SET publish = ?, updated_at = ?
WHERE id = ? AND session_id = ?;
