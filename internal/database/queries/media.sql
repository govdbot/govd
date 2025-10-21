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
) RETURNING id;

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