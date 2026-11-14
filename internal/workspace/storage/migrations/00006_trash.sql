-- +goose Up
ALTER TABLE workspaces ADD COLUMN deleted_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE pages ADD COLUMN deleted_at INTEGER NOT NULL DEFAULT 0;
CREATE INDEX workspaces_trash ON workspaces(deleted_at);
CREATE INDEX pages_trash ON pages(deleted_at);
CREATE VIEW active_workspaces AS SELECT * FROM workspaces WHERE deleted_at=0;
CREATE VIEW active_pages AS SELECT p.* FROM pages p JOIN active_workspaces w ON w.id=p.workspace_id WHERE p.deleted_at=0;
DROP VIEW asset_access;
DROP VIEW workspace_access;
CREATE VIEW workspace_access AS
 SELECT id AS workspace_id,owner_id AS user_id,'owner' AS role FROM active_workspaces
 UNION ALL SELECT m.workspace_id,m.user_id,m.role FROM workspace_members m JOIN active_workspaces w ON w.id=m.workspace_id;
CREATE VIEW active_assets AS SELECT a.* FROM assets a WHERE
 (NOT EXISTS(SELECT 1 FROM page_assets pa WHERE pa.asset_id=a.id) AND NOT EXISTS(SELECT 1 FROM template_assets ta WHERE ta.asset_id=a.id) AND (a.workspace_id IS NULL OR a.workspace_id IN(SELECT id FROM active_workspaces)))
 OR EXISTS(SELECT 1 FROM page_assets pa JOIN active_pages p ON p.id=pa.page_id WHERE pa.asset_id=a.id)
 OR EXISTS(SELECT 1 FROM template_assets ta JOIN workspace_templates t ON t.id=ta.template_id JOIN active_workspaces w ON w.id=t.workspace_id WHERE ta.asset_id=a.id);
CREATE VIEW asset_access AS
 SELECT id AS asset_id,owner_id AS user_id FROM active_assets
 UNION SELECT a.id,wa.user_id FROM active_assets a JOIN workspace_access wa ON wa.workspace_id=a.workspace_id
 UNION SELECT pa.asset_id,wa.user_id FROM page_assets pa JOIN active_pages p ON p.id=pa.page_id JOIN workspace_access wa ON wa.workspace_id=p.workspace_id
 UNION SELECT ta.asset_id,wa.user_id FROM template_assets ta JOIN workspace_templates t ON t.id=ta.template_id JOIN workspace_access wa ON wa.workspace_id=t.workspace_id;

-- +goose Down
DROP VIEW asset_access;
DROP VIEW active_assets;
DROP VIEW workspace_access;
CREATE VIEW workspace_access AS
 SELECT id AS workspace_id,owner_id AS user_id,'owner' AS role FROM workspaces
 UNION ALL SELECT workspace_id,user_id,role FROM workspace_members;
CREATE VIEW asset_access AS
 SELECT id AS asset_id,owner_id AS user_id FROM assets
 UNION SELECT a.id,wa.user_id FROM assets a JOIN workspace_access wa ON wa.workspace_id=a.workspace_id
 UNION SELECT pa.asset_id,wa.user_id FROM page_assets pa JOIN pages p ON p.id=pa.page_id JOIN workspace_access wa ON wa.workspace_id=p.workspace_id
 UNION SELECT ta.asset_id,wa.user_id FROM template_assets ta JOIN workspace_templates t ON t.id=ta.template_id JOIN workspace_access wa ON wa.workspace_id=t.workspace_id;
DROP VIEW active_pages;
DROP VIEW active_workspaces;
DROP INDEX pages_trash;
DROP INDEX workspaces_trash;
ALTER TABLE pages DROP COLUMN deleted_at;
ALTER TABLE workspaces DROP COLUMN deleted_at;
