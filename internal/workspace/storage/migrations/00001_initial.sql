-- +goose Up
CREATE TABLE IF NOT EXISTS workspaces (
 id TEXT NOT NULL PRIMARY KEY, owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 name TEXT NOT NULL, revision INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS workspaces_owner ON workspaces(owner_id);

-- +goose Down
DROP TABLE IF EXISTS workspaces;
