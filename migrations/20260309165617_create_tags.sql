-- +goose Up
CREATE TABLE IF NOT EXISTS tags (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    is_generated BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT
);

-- +goose Down
DROP TABLE IF EXISTS tags;