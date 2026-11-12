-- +goose Up
CREATE TABLE workspace_members (
 workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 user_id TEXT NOT NULL, email TEXT NOT NULL, role TEXT NOT NULL CHECK(role IN ('read','comment','full')),
 PRIMARY KEY(workspace_id,user_id)
);
CREATE INDEX workspace_members_user ON workspace_members(user_id);
ALTER TABLE page_leases ADD COLUMN holder_user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN workspace_id TEXT REFERENCES workspaces(id) ON DELETE SET NULL;
CREATE VIEW workspace_access AS
 SELECT id AS workspace_id,owner_id AS user_id,'owner' AS role FROM workspaces
 UNION ALL SELECT workspace_id,user_id,role FROM workspace_members;
CREATE VIEW asset_access AS
 SELECT id AS asset_id,owner_id AS user_id FROM assets
 UNION SELECT a.id,wa.user_id FROM assets a JOIN workspace_access wa ON wa.workspace_id=a.workspace_id
 UNION SELECT pa.asset_id,wa.user_id FROM page_assets pa JOIN pages p ON p.id=pa.page_id JOIN workspace_access wa ON wa.workspace_id=p.workspace_id;
CREATE TABLE comment_threads (
 id TEXT NOT NULL PRIMARY KEY, page_id TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
 author_id TEXT NOT NULL, author_email TEXT NOT NULL, x REAL NOT NULL, y REAL NOT NULL,
 resolved INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL
);
CREATE INDEX comments_page ON comment_threads(page_id,created_at);
CREATE TABLE comment_messages (
 id TEXT NOT NULL PRIMARY KEY, thread_id TEXT NOT NULL REFERENCES comment_threads(id) ON DELETE CASCADE,
 author_id TEXT NOT NULL, author_email TEXT NOT NULL, body TEXT NOT NULL, created_at INTEGER NOT NULL
);
CREATE INDEX comments_thread ON comment_messages(thread_id,created_at);

-- +goose Down
DROP TABLE comment_messages;
DROP TABLE comment_threads;
DROP VIEW asset_access;
DROP VIEW workspace_access;
ALTER TABLE assets DROP COLUMN workspace_id;
ALTER TABLE page_leases DROP COLUMN holder_user_id;
DROP TABLE workspace_members;
