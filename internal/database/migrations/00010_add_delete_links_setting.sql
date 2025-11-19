-- +goose Up
-- +goose StatementBegin
ALTER TABLE settings ADD COLUMN delete_links BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE settings DROP COLUMN IF EXISTS delete_links;
-- +goose StatementEnd