-- name: GetOrCreateChat :one
WITH upsert_chat AS (
    INSERT INTO chat (chat_id, type)
    VALUES (@chat_id, @type)
    ON CONFLICT (chat_id) DO NOTHING
    RETURNING *
),
upsert_settings AS (
    INSERT INTO settings (chat_id, language, captions, silent, nsfw, media_album_limit)
    SELECT chat_id, @language, @captions, @silent, @nsfw, @media_album_limit 
    FROM upsert_chat
    ON CONFLICT (chat_id) DO NOTHING
    RETURNING *
),
chat AS (
    SELECT * FROM chat WHERE chat_id = @chat_id
    UNION SELECT * FROM upsert_chat
),
settings AS (
    SELECT * FROM settings WHERE chat_id = @chat_id
    UNION SELECT * FROM upsert_settings
)
SELECT 
    c.chat_id,
    c.type,
    s.nsfw,
    s.media_album_limit,
    s.captions,
    s.silent,
    s.language,
    s.disabled_extractors
FROM chat c 
JOIN settings s ON s.chat_id = c.chat_id;