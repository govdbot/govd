-- name: GetOrCreateChat :one
WITH upsert_chat AS (
    INSERT INTO chat (chat_id, type)
    VALUES (@chat_id, @type)
    ON CONFLICT (chat_id) DO NOTHING
    RETURNING chat_id, type
),
upsert_settings AS (
    INSERT INTO settings (chat_id, language, captions, silent, nsfw, media_album_limit)
    SELECT chat_id, @language, @captions, @silent, @nsfw, @media_album_limit FROM upsert_chat
    ON CONFLICT (chat_id) DO NOTHING
    RETURNING chat_id, nsfw, media_album_limit, silent, captions, language
),
chat_un AS (
    SELECT c.chat_id, c.type
    FROM chat c 
    WHERE c.chat_id = @chat_id
    UNION
    SELECT uc.chat_id, uc.type
    FROM upsert_chat uc
),
settings_un AS (
    SELECT s.chat_id, s.nsfw, s.media_album_limit, s.silent, s.captions, s.language
    FROM settings s
    WHERE s.chat_id = @chat_id
    UNION
    SELECT us.chat_id, us.nsfw, us.media_album_limit, us.silent, us.captions, us.language
    FROM upsert_settings us
)
SELECT c.chat_id, c.type, s.nsfw, s.media_album_limit, s.silent, s.captions, s.language
FROM chat_un c JOIN settings_un s ON s.chat_id = c.chat_id;