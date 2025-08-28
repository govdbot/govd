-- name: SetChatLanguage :exec
UPDATE settings
SET language = @language, updated_at = CURRENT_TIMESTAMP
WHERE chat_id = @chat_id;

-- name: ToggleChatCaptions :exec
UPDATE settings
SET captions = NOT captions, updated_at = CURRENT_TIMESTAMP
WHERE chat_id = @chat_id;

-- name: ToggleChatNsfw :exec
UPDATE settings
SET nsfw = NOT nsfw, updated_at = CURRENT_TIMESTAMP
WHERE chat_id = @chat_id;

-- name: ToggleChatSilentMode :exec
UPDATE settings
SET silent = NOT silent, updated_at = CURRENT_TIMESTAMP
WHERE chat_id = @chat_id;

-- name: SetChatMediaAlbumLimit :exec
UPDATE settings
SET media_album_limit = @media_album_limit, updated_at = CURRENT_TIMESTAMP
WHERE chat_id = @chat_id;