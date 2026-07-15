-- +goose Up
-- +goose StatementBegin
-- version the cached media row so a binary/extractor bug fix can force a fresh extraction.
-- IF NOT EXISTS so re-apply is safe even if column already exists.
ALTER TABLE media ADD COLUMN IF NOT EXISTS cache_version TEXT NOT NULL DEFAULT '1';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE media DROP COLUMN IF EXISTS cache_version;
-- +goose StatementEnd
