-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS settings (
    chat_id BIGINT NOT NULL UNIQUE REFERENCES chat(chat_id) ON DELETE CASCADE,
    nsfw BOOLEAN DEFAULT FALSE NOT NULL,
    media_album_limit INT DEFAULT 10 NOT NULL,
    silent BOOLEAN DEFAULT FALSE NOT NULL,
    language VARCHAR(2) DEFAULT 'en' NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_settings_chat_id
    ON settings (chat_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS settings CASCADE;
DROP INDEX IF EXISTS idx_settings_chat_id;
-- +goose StatementEnd
