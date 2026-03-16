-- +goose Up
ALTER TABLE notes ADD COLUMN user_id INT NOT NULL DEFAULT 1;
ALTER TABLE notes ADD CONSTRAINT fk_notes_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE notes DROP CONSTRAINT fk_notes_user;
ALTER TABLE notes DROP COLUMN user_id;