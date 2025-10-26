-- +goose Up
-- +goose StatementBegin
ALTER TABLE media ALTER COLUMN content_id TYPE VARCHAR(150);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE media ALTER COLUMN content_id TYPE VARCHAR(50);
-- +goose StatementEnd
