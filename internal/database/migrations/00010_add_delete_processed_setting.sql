-- +goose Up
-- +goose StatementBegin
ALTER TABLE settings ADD COLUMN delete_processed BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE settings DROP COLUMN IF EXISTS delete_processed;
-- +goose StatementEnd