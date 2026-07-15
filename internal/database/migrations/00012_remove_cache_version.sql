-- +goose Up
-- +goose StatementBegin
-- Simplify cache handling: remove versioning, use simple UPSERT with caption overwrite.
-- This makes fresh installs zero-setup (no CACHE_VERSION env needed).
-- Safe: column may not exist on fresh DBs that never ran 00011.
ALTER TABLE media DROP COLUMN IF EXISTS cache_version;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE media ADD COLUMN IF NOT EXISTS cache_version TEXT NOT NULL DEFAULT '1';
-- +goose StatementEnd
