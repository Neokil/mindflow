-- name: WorkspaceListWorkspaces :many
SELECT w.id,w.name,w.revision,w.updated_at,(SELECT count(*) FROM pages p WHERE p.workspace_id=w.id) AS page_count FROM workspaces w WHERE w.owner_id=? ORDER BY w.updated_at DESC,w.id;

-- name: WorkspaceCreateWorkspace :execresult
INSERT INTO workspaces(id,owner_id,name,created_at,updated_at) VALUES(?,?,?,?,?);

-- name: WorkspaceWorkspaceRevision :one
SELECT revision FROM workspaces WHERE id=? AND owner_id=?;

-- name: WorkspaceRenameWorkspace :execresult
UPDATE workspaces SET name=?,revision=revision+1,updated_at=? WHERE id=?;

-- name: WorkspaceDeleteWorkspace :execresult
DELETE FROM workspaces WHERE id=? AND owner_id=?;

-- name: WorkspaceWorkspaceName :one
SELECT name FROM workspaces WHERE id=? AND owner_id=?;

-- name: WorkspaceIncrementWorkspaceRevision :execresult
UPDATE workspaces SET revision=revision+1 WHERE id=?;

-- name: WorkspaceTouchWorkspace :execresult
UPDATE workspaces SET updated_at=? WHERE id=(SELECT workspace_id FROM pages WHERE pages.id=?);

-- name: PageOwnedWorkspace :one
SELECT id FROM workspaces WHERE id=? AND owner_id=?;

-- name: PageCreateRecoveryPage :execresult
INSERT INTO pages(id,workspace_id,name,position,document,revision,updated_at) SELECT ?,?,?,COALESCE(MAX(position),-1)+1,?,1,? FROM pages WHERE pages.workspace_id=?;

-- name: PageListPages :many
SELECT id,workspace_id,name,position,revision,metadata_revision FROM pages WHERE pages.workspace_id=? ORDER BY position,id;

-- name: PageRenamePage :execresult
UPDATE pages SET name=?,metadata_revision=metadata_revision+1 WHERE id=?;

-- name: PageDeletePage :execresult
DELETE FROM pages WHERE id=?;

-- name: PageCountPages :one
SELECT count(*) FROM pages WHERE pages.workspace_id=?;

-- name: PagePositionPage :execresult
UPDATE pages SET position=? WHERE id=? AND workspace_id=?;

-- name: PageCreatePage :execresult
INSERT INTO pages(id,workspace_id,name,position,document,updated_at) SELECT ?,?,?,COALESCE(MAX(position),-1)+1,?,? FROM pages WHERE pages.workspace_id=?;

-- name: PagePageDocument :one
SELECT document FROM pages WHERE id=?;

-- name: PageOwnedPage :one
SELECT p.id,p.workspace_id,p.name,p.position,p.revision,p.metadata_revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: PagePagesToCopy :many
SELECT id,name FROM pages WHERE pages.workspace_id=? ORDER BY position;

-- name: CanvasGetDocument :one
SELECT p.document,p.revision,p.metadata_revision,p.name,p.workspace_id FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: CanvasOwnedPageRevision :one
SELECT p.revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: CanvasSaveReceipt :one
SELECT digest,revision FROM save_receipts WHERE page_id=? AND client_id=? AND mutation_id=? AND expires_at>?;

-- name: CanvasSaveDocument :execresult
UPDATE pages SET document=?,revision=?,updated_at=? WHERE id=?;

-- name: CanvasCreateSaveReceipt :execresult
INSERT INTO save_receipts(page_id,client_id,mutation_id,digest,revision,expires_at) VALUES(?,?,?,?,?,?);

-- name: CanvasAttachAsset :execresult
INSERT OR IGNORE INTO page_assets(page_id,asset_id) VALUES(?,?);

-- name: CanvasCopyPageAssets :execresult
INSERT INTO page_assets(page_id,asset_id) SELECT ?,asset_id FROM page_assets WHERE page_assets.page_id=?;

-- name: CanvasDeleteExpiredReceipts :exec
DELETE FROM save_receipts WHERE expires_at<=unixepoch();

-- name: EditorOwnedPage :one
SELECT p.id,p.workspace_id,p.name,p.position,p.revision,p.metadata_revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: EditorOwnedPageRevision :one
SELECT p.revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: EditorPageLease :one
SELECT session_id,client_id,epoch,expires_at FROM page_leases WHERE page_id=?;

-- name: EditorEnsureLease :execresult
INSERT OR IGNORE INTO page_leases(page_id) VALUES(?);

-- name: EditorUpdateLease :execresult
UPDATE page_leases SET session_id=?,client_id=?,epoch=?,expires_at=? WHERE page_id=?;

-- name: EditorExpireLeases :execresult
UPDATE page_leases SET epoch=epoch+1,expires_at=0,session_id=NULL,client_id='';

-- name: ImageCountOwnedAsset :one
SELECT count(*) FROM assets WHERE id=? AND owner_id=?;

-- name: ImageAssetMetadata :one
SELECT id,width,height,mime FROM assets WHERE id=? AND owner_id=?;

-- name: ImageVariantBuckets :many
SELECT bucket FROM asset_variants WHERE asset_id=? ORDER BY bucket;

-- name: ImageAssetDimensions :one
SELECT width,height FROM assets WHERE id=? AND owner_id=?;

-- name: ImageCountVariants :one
SELECT count(*) FROM asset_variants WHERE asset_id=? AND bucket=?;

-- name: ImageQueueVariant :execresult
INSERT OR IGNORE INTO image_jobs(id,user_id,asset_id,dedup_key,payload,run_at) VALUES(?,?,?,?,?,?);

-- name: ImageDueVariant :one
SELECT id,user_id,asset_id,payload,attempts FROM image_jobs WHERE attempts<3 AND run_at<=? ORDER BY run_at LIMIT 1;

-- name: ImageOriginalImage :one
SELECT original,width,height FROM assets WHERE id=? AND owner_id=?;

-- name: ImageDeleteJob :execresult
DELETE FROM image_jobs WHERE id=?;

-- name: ImageCreateVariant :execresult
INSERT OR IGNORE INTO asset_variants(asset_id,bucket,mime,data,width,height) VALUES(?,?,?,?,?,?);

-- name: ImageRetryVariant :execresult
UPDATE image_jobs SET attempts=attempts+1,run_at=?,last_error='Image preview generation failed.' WHERE id=?;

-- name: ImageCreateAsset :execresult
INSERT INTO assets(id,owner_id,original,mime,width,height,digest,created_at) VALUES(?,?,?,?,?,?,?,?);

-- name: ImageImageVariant :one
SELECT v.data,v.mime,v.width,v.height FROM asset_variants v JOIN assets a ON a.id=v.asset_id WHERE a.id=? AND a.owner_id=? AND v.bucket=?;

-- name: ImageGetImage :one
SELECT original,mime,width,height FROM assets WHERE id=? AND owner_id=?;

-- name: ImageDeleteWorkspaceAssets :execresult
DELETE FROM assets WHERE owner_id=? AND id IN(SELECT pa.asset_id FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE p.workspace_id=?) AND NOT EXISTS(SELECT 1 FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE pa.asset_id=assets.id AND p.workspace_id<>?);

-- name: ImageDeletePageAssets :execresult
DELETE FROM assets WHERE owner_id=? AND id IN(SELECT asset_id FROM page_assets WHERE page_assets.page_id=?) AND NOT EXISTS(SELECT 1 FROM page_assets pa WHERE pa.asset_id=assets.id AND pa.page_id<>?);

-- name: ImageDeleteOrphanVariants :exec
DELETE FROM image_jobs WHERE asset_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM assets WHERE assets.id=image_jobs.asset_id);

-- name: ImageDeleteUnusedAssets :exec
DELETE FROM assets WHERE created_at<unixepoch()-86400 AND NOT EXISTS(SELECT 1 FROM page_assets WHERE page_assets.asset_id=assets.id);

-- name: WorkspaceDeleteOwnerWorkspaces :exec
DELETE FROM workspaces WHERE owner_id=?;

-- name: WorkspaceDeleteOwnerImages :exec
DELETE FROM assets WHERE owner_id=?;
