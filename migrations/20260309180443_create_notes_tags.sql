-- +goose Up
CREATE TABLE IF NOT EXISTS note_tags (
    note_id BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);

-- +goose Down
DROP TABLE IF EXISTS note_tags;
