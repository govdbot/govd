-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS errors (
    id VARCHAR(16) PRIMARY KEY,
    message TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS errors;
-- +goose StatementEnd
