-- +goose Up
CREATE TABLE IF NOT EXISTS notes (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    is_generated BOOLEAN NOT NULL DEFAULT FALSE,
    priority INT DEFAULT 0,
    category VARCHAR(100)
);

-- +goose Down
DROP TABLE IF EXISTS notes;