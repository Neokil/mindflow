-- name: OwnershipAssets :many
SELECT a.id,EXISTS(SELECT 1 FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE pa.asset_id=a.id AND p.workspace_id<>sqlc.arg(workspace_id)) OR EXISTS(SELECT 1 FROM template_assets ta JOIN workspace_templates t ON t.id=ta.template_id WHERE ta.asset_id=a.id AND t.workspace_id<>sqlc.arg(workspace_id)) AS shared_elsewhere
FROM assets a WHERE a.workspace_id=sqlc.arg(workspace_id) OR a.id IN(SELECT pa.asset_id FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE p.workspace_id=sqlc.arg(workspace_id)) OR a.id IN(SELECT ta.asset_id FROM template_assets ta JOIN workspace_templates t ON t.id=ta.template_id WHERE t.workspace_id=sqlc.arg(workspace_id));
-- name: OwnershipReassignAsset :exec
UPDATE assets SET owner_id=?,workspace_id=? WHERE id=?;
-- name: OwnershipCloneAsset :exec
INSERT INTO assets(id,owner_id,original,mime,width,height,digest,created_at,workspace_id) SELECT sqlc.arg(new_id),sqlc.arg(owner_id),source.original,source.mime,source.width,source.height,source.digest,source.created_at,sqlc.arg(workspace_id) FROM assets source WHERE source.id=sqlc.arg(source_id);
-- name: OwnershipCloneVariants :exec
INSERT INTO asset_variants(asset_id,bucket,mime,data,width,height) SELECT sqlc.arg(new_id),source.bucket,source.mime,source.data,source.width,source.height FROM asset_variants source WHERE source.asset_id=sqlc.arg(source_id);
-- name: OwnershipDetachOriginal :exec
UPDATE assets SET workspace_id=NULL WHERE id=? AND workspace_id=?;
-- name: OwnershipMoveJobs :exec
UPDATE image_jobs SET user_id=? WHERE asset_id=?;
-- name: OwnershipPages :many
SELECT id,document FROM pages WHERE workspace_id=?;
-- name: OwnershipPageDocument :exec
UPDATE pages SET document=?,revision=revision+1 WHERE id=?;
-- name: OwnershipTemplates :many
SELECT id,elements FROM workspace_templates WHERE workspace_id=?;
-- name: OwnershipTemplateElements :exec
UPDATE workspace_templates SET elements=? WHERE id=?;
-- name: OwnershipPageAssets :exec
UPDATE page_assets SET asset_id=? WHERE asset_id=? AND page_id IN(SELECT id FROM pages WHERE workspace_id=?);
-- name: OwnershipTemplateAssets :exec
UPDATE template_assets SET asset_id=? WHERE asset_id=? AND template_id IN(SELECT id FROM workspace_templates WHERE workspace_id=?);
-- name: OwnershipWorkspace :execrows
UPDATE workspaces SET owner_id=sqlc.arg(owner_id),revision=revision+1,updated_at=sqlc.arg(updated_at) WHERE id=sqlc.arg(id) AND owner_id=sqlc.arg(previous_owner) AND revision=sqlc.arg(revision) AND deleted_at=0;
-- name: OwnershipRevokeLeases :exec
UPDATE page_leases SET epoch=epoch+1,expires_at=0,session_id=NULL,client_id='',holder_user_id='' WHERE page_id IN(SELECT id FROM pages WHERE workspace_id=?);
-- name: OwnershipCancelRequests :exec
DELETE FROM editor_requests WHERE page_id IN(SELECT id FROM pages WHERE workspace_id=?);
