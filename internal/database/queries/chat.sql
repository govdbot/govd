-- name: GetOrCreateChat :one
WITH upsert_chat AS (
    INSERT INTO chat (chat_id, type)
    VALUES (@chat_id, @type)
    ON CONFLICT (chat_id) DO NOTHING
    RETURNING *
),
upsert_settings AS (
    INSERT INTO settings (chat_id, language, captions, silent, nsfw, media_album_limit)
    VALUES (@chat_id, @language, @captions, @silent, @nsfw, @media_album_limit)
    ON CONFLICT (chat_id) DO UPDATE SET
        language = CASE 
            WHEN settings.language = 'XX' THEN EXCLUDED.language 
            ELSE settings.language 
        END,
        captions = EXCLUDED.captions,
        silent = EXCLUDED.silent,
        nsfw = EXCLUDED.nsfw,
        media_album_limit = EXCLUDED.media_album_limit
    RETURNING *
),
final_chat AS (
    SELECT * FROM upsert_chat
    UNION ALL
    SELECT * FROM chat WHERE chat_id = @chat_id AND NOT EXISTS (SELECT 1 FROM upsert_chat)
),
final_settings AS (
    SELECT * FROM upsert_settings
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
FROM final_chat c 
JOIN final_settings s ON s.chat_id = c.chat_id;