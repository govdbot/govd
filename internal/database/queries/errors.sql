-- name: LogError :exec
INSERT INTO errors (id, message)
VALUES (@id, @message)
ON CONFLICT (id) DO UPDATE
SET occurrences = errors.occurrences + 1,
    last_seen = NOW();

-- name: GetErrorByID :one
SELECT message
FROM errors
WHERE id = @id;