-- +goose Up
CREATE TABLE IF NOT EXISTS pages (
 id TEXT NOT NULL PRIMARY KEY, workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 name TEXT NOT NULL, position INTEGER NOT NULL, metadata_revision INTEGER NOT NULL DEFAULT 1,
 revision INTEGER NOT NULL DEFAULT 0, document TEXT NOT NULL, updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS pages_workspace ON pages(workspace_id,position);

-- +goose Down
DROP TABLE IF EXISTS pages;
