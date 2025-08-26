-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS chat (
    chat_id BIGINT NOT NULL UNIQUE,
    type chat_type NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_chat_chat_id
    ON chat (chat_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS chat CASCADE;
DROP INDEX IF EXISTS idx_chat_chat_id;
-- +goose StatementEnd
