-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS media_format (
    format_id VARCHAR(100) NOT NULL,
    item_id BIGINT NOT NULL REFERENCES media_item(id) ON DELETE CASCADE,
    file_id VARCHAR(255) NOT NULL,
    type media_type NOT NULL,
    audio_codec media_codec,
    video_codec media_codec,
    duration INT,
    title VARCHAR(255),
    artist VARCHAR(255),
    width INT,
    height INT,
    bitrate BIGINT,
    file_size BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (item_id)
);

CREATE INDEX IF NOT EXISTS idx_format_format_id
    ON media_format (format_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS media_format CASCADE;
DROP INDEX IF EXISTS idx_format_item_id;
DROP INDEX IF EXISTS idx_format_format_id;
-- +goose StatementEnd
