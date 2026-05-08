-- +goose Up
-- +goose StatementBegin
ALTER TABLE settings ADD COLUMN high_quality_videos BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE settings DROP COLUMN IF EXISTS high_quality_videos;
-- +goose StatementEnd
