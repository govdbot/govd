-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS media_format (
    id BIGSERIAL PRIMARY KEY,
    item_id BIGINT NOT NULL REFERENCES media_item(id) ON DELETE CASCADE,
    format_id VARCHAR(255) NOT NULL,
    file_id VARCHAR(255) NOT NULL,
    audio_codec media_codec,
    video_codec media_codec,
    width INT,
    height INT,
    bitrate INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (item_id, format_id)
);

CREATE INDEX IF NOT EXISTS idx_format_item_id
    ON media_format (item_id);

CREATE INDEX IF NOT EXISTS idx_format_format_id
    ON media_format (format_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS media_format CASCADE;
DROP INDEX IF EXISTS idx_format_item_id;
DROP INDEX IF EXISTS idx_format_format_id;
-- +goose StatementEnd
