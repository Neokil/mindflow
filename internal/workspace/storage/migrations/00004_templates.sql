-- +goose Up
CREATE TABLE workspace_templates (
 id TEXT NOT NULL PRIMARY KEY,
 workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 name TEXT NOT NULL, elements TEXT NOT NULL, element_count INTEGER NOT NULL,
 UNIQUE(workspace_id,name)
);
CREATE INDEX templates_workspace ON workspace_templates(workspace_id);
CREATE TABLE template_assets (
 template_id TEXT NOT NULL REFERENCES workspace_templates(id) ON DELETE CASCADE,
 asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
 PRIMARY KEY(template_id,asset_id)
);
CREATE INDEX template_assets_asset ON template_assets(asset_id);
DROP VIEW asset_access;
CREATE VIEW asset_access AS
 SELECT id AS asset_id,owner_id AS user_id FROM assets
 UNION SELECT a.id,wa.user_id FROM assets a JOIN workspace_access wa ON wa.workspace_id=a.workspace_id
 UNION SELECT pa.asset_id,wa.user_id FROM page_assets pa JOIN pages p ON p.id=pa.page_id JOIN workspace_access wa ON wa.workspace_id=p.workspace_id
 UNION SELECT ta.asset_id,wa.user_id FROM template_assets ta JOIN workspace_templates t ON t.id=ta.template_id JOIN workspace_access wa ON wa.workspace_id=t.workspace_id;

-- +goose Down
DROP VIEW asset_access;
CREATE VIEW asset_access AS
 SELECT id AS asset_id,owner_id AS user_id FROM assets
 UNION SELECT a.id,wa.user_id FROM assets a JOIN workspace_access wa ON wa.workspace_id=a.workspace_id
 UNION SELECT pa.asset_id,wa.user_id FROM page_assets pa JOIN pages p ON p.id=pa.page_id JOIN workspace_access wa ON wa.workspace_id=p.workspace_id;
DROP TABLE template_assets;
DROP TABLE workspace_templates;
