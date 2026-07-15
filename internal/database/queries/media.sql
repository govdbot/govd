-- name: CreateMedia :one
INSERT INTO media (
    content_id,
    content_url,
    extractor_id,
    caption,
    nsfw
) VALUES (
    @content_id,
    @content_url,
    @extractor_id,
    @caption,
    @nsfw
) ON CONFLICT (content_id, extractor_id) DO UPDATE SET
    caption = EXCLUDED.caption,
    content_url = EXCLUDED.content_url,
    nsfw = EXCLUDED.nsfw,
    updated_at = CURRENT_TIMESTAMP
RETURNING id;

-- name: DeleteMediaByContentID :exec
DELETE FROM media WHERE content_id = @content_id AND extractor_id = @extractor_id;

-- name: DeleteMediaItemsByMediaID :exec
DELETE FROM media_item WHERE media_id = @media_id;

-- name: CreateMediaItem :one
INSERT INTO media_item (
    media_id
) VALUES (
    @media_id
) RETURNING id;

-- name: CreateMediaFormat :exec
INSERT INTO media_format (
    format_id,
    item_id,
    file_id,
    type,
    audio_codec,
    video_codec,
    duration,
    file_size,
    title,
    artist,
    width,
    height,
    bitrate
) VALUES (
    @format_id,
    @item_id,
    @file_id,
    @type,
    @audio_codec,
    @video_codec,
    @duration,
    @file_size,
    @title,
    @artist,
    @width,
    @height,
    @bitrate
);

-- name: GetMediaByContentID :one
SELECT 
    id,
    content_id,
    content_url,
    extractor_id,
    caption,
    nsfw
FROM media WHERE content_id = @content_id
AND extractor_id = @extractor_id;

-- name: GetMedia :one
SELECT 
    id,
    content_id,
    content_url,
    extractor_id,
    caption,
    nsfw
FROM media WHERE id = @id;

-- name: GetMediaItems :many
SELECT 
    id,
    media_id
FROM media_item WHERE media_id = @media_id;

-- name: GetMediaItemsWithFormats :many
SELECT
    mi.id as item_id,
    mf.format_id,
    mf.item_id,
    mf.file_id,
    mf.type,
    mf.audio_codec,
    mf.video_codec,
    mf.duration,
    mf.file_size,
    mf.title,
    mf.artist,
    mf.width,
    mf.height,
    mf.bitrate
FROM media_item mi
JOIN media_format mf ON mf.item_id = mi.id
WHERE mi.media_id = @media_id
ORDER BY mi.id;

-- name: GetMediaFormat :one
SELECT 
    format_id,
    item_id,
    file_id,
    type,
    audio_codec,
    video_codec,
    duration,
    file_size,
    title,
    artist,
    width,
    height,
    bitrate
FROM media_format WHERE item_id = @item_id;