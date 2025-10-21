-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS media (
    id BIGSERIAL PRIMARY KEY,
    content_id VARCHAR(50) NOT NULL,
    content_url TEXT NOT NULL,
    extractor_id VARCHAR(30) NOT NULL,
    caption TEXT,
    nsfw BOOLEAN DEFAULT FALSE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (content_id, extractor_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS media CASCADE;
DROP INDEX IF EXISTS idx_media_content_id;
DROP INDEX IF EXISTS idx_media_extractor_id;
-- +goose StatementEnd
