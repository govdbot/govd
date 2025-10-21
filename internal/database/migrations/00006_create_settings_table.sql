-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS settings (
    chat_id BIGINT NOT NULL UNIQUE REFERENCES chat(chat_id) ON DELETE CASCADE,
    nsfw BOOLEAN NOT NULL,
    media_album_limit INT NOT NULL,
    captions BOOLEAN NOT NULL,
    silent BOOLEAN NOT NULL,
    language CHAR(2) NOT NULL CHECK (LENGTH(language) = 2),
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
