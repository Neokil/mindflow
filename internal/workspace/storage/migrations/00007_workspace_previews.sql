-- +goose Up
CREATE TABLE workspace_previews (
 workspace_id TEXT PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
 page_id TEXT NOT NULL,
 revision INTEGER NOT NULL,
 content_id TEXT NOT NULL,
 data BLOB NOT NULL,
 empty INTEGER NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE workspace_previews;
