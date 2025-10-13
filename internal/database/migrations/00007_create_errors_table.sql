-- +goose Up
-- +goose StatementBegin
CREATE TABLE errors (
    id CHAR(8) PRIMARY KEY CHECK (LENGTH(id) = 8),
    message TEXT NOT NULL,
    occurrences INT NOT NULL DEFAULT 1,
    first_seen TIMESTAMP NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMP NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS errors;
-- +goose StatementEnd
