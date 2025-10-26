-- +goose Up
-- +goose StatementBegin
ALTER TABLE settings ADD COLUMN disabled_extractors TEXT[] DEFAULT '{}' NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE settings DROP COLUMN disabled_extractors;
-- +goose StatementEnd