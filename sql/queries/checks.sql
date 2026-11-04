-- name: CreateCheck :one
INSERT INTO checks (url, status_code, response_time_ms)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetChecks :many
SELECT id, url, status_code, response_time_ms, checked_at 
FROM checks
ORDER BY checked_at DESC
LIMIT $1;