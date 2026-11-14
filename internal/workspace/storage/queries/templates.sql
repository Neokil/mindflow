-- name: TemplateList :many
SELECT id,name,element_count FROM workspace_templates WHERE workspace_id=? ORDER BY name COLLATE NOCASE,id;

-- name: TemplateGet :one
SELECT id,name,elements,element_count FROM workspace_templates WHERE id=? AND workspace_id=?;

-- name: TemplateNameExists :one
SELECT count(*) FROM workspace_templates WHERE workspace_id=? AND name=?;

-- name: TemplateCreate :exec
INSERT INTO workspace_templates(id,workspace_id,name,elements,element_count) VALUES(?,?,?,?,?);

-- name: TemplateDelete :exec
DELETE FROM workspace_templates WHERE id=? AND workspace_id=?;

-- name: TemplateAttachAsset :exec
INSERT OR IGNORE INTO template_assets(template_id,asset_id) VALUES(?,?);

-- name: TemplateAssetAllowed :one
SELECT count(*) FROM assets a JOIN active_workspaces w ON w.id=sqlc.arg(workspace_id)
WHERE a.id=sqlc.arg(asset_id) AND (
 (a.owner_id=sqlc.arg(user_id) AND a.owner_id=w.owner_id) OR a.workspace_id=w.id
 OR EXISTS(SELECT 1 FROM page_assets pa JOIN active_pages p ON p.id=pa.page_id WHERE pa.asset_id=a.id AND p.workspace_id=w.id)
 OR EXISTS(SELECT 1 FROM template_assets ta JOIN workspace_templates t ON t.id=ta.template_id WHERE ta.asset_id=a.id AND t.workspace_id=w.id)
);

-- name: TemplateCopies :many
SELECT id,name,elements,element_count FROM workspace_templates WHERE workspace_id=?;

-- name: TemplateCopyAssets :exec
INSERT INTO template_assets(template_id,asset_id) SELECT sqlc.arg(target_id),source_assets.asset_id FROM template_assets source_assets WHERE source_assets.template_id=sqlc.arg(source_id);
