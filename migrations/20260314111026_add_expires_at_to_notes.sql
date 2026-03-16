-- +goose Up
ALTER TABLE notes ADD COLUMN expires_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE notes DROP COLUMN expires_at;