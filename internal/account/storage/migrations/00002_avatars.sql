-- +goose Up
ALTER TABLE users ADD COLUMN avatar_version TEXT NOT NULL DEFAULT '';
CREATE TABLE avatars (
 user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 content BLOB NOT NULL
);

-- +goose Down
DROP TABLE avatars;
ALTER TABLE users DROP COLUMN avatar_version;
