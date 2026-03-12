-- +goose Up
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(100) PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    ip INET,
    browser TEXT,
    is_generated BOOLEAN NOT NULL DEFAULT FALSE,
    device_type VARCHAR(50)
);

-- +goose Down
DROP TABLE IF EXISTS sessions;