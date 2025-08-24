-- name: GetMedias :many
SELECT * FROM media ORDER BY id;

-- name: AddMedia :one
INSERT INTO media (content_id, content_url, extractor_id, caption, nsfw)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;