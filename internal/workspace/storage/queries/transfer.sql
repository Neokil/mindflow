-- name: TransferWorkspace :one
SELECT name FROM active_workspaces WHERE id=?;
-- name: TransferPages :many
SELECT id,name,document FROM active_pages WHERE workspace_id=? ORDER BY position,id;
-- name: TransferTemplates :many
SELECT name,elements FROM workspace_templates WHERE workspace_id=? ORDER BY name,id;
-- name: TransferAsset :one
SELECT original,mime FROM assets WHERE id=?;
-- name: TransferPDFPreview :one
SELECT data FROM asset_variants WHERE asset_id=? AND bucket=1024;
