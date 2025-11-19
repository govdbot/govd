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

-- name: AddDisabledExtractor :exec
UPDATE settings
SET disabled_extractors = array_append(disabled_extractors, @extractor_id), updated_at = CURRENT_TIMESTAMP
WHERE chat_id = @chat_id
AND NOT (@extractor_id = ANY(disabled_extractors));

-- name: RemoveDisabledExtractor :exec
UPDATE settings
SET disabled_extractors = array_remove(disabled_extractors, @extractor_id), updated_at = CURRENT_TIMESTAMP
WHERE chat_id = @chat_id;

-- name: ToggleChatDeleteLinks :exec
UPDATE settings
SET delete_links = NOT delete_links, updated_at = CURRENT_TIMESTAMP
WHERE chat_id = @chat_id;