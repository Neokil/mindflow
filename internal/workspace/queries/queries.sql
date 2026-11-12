-- name: GetDocument :one
SELECT p.document,p.revision,p.metadata_revision,p.name,p.workspace_id FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: OwnedPageRevision :one
SELECT p.revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: SaveReceipt :one
SELECT digest,revision FROM save_receipts WHERE page_id=? AND client_id=? AND mutation_id=? AND expires_at>?;

-- name: PageLease :one
SELECT session_id,client_id,epoch,expires_at FROM page_leases WHERE page_id=?;

-- name: SaveDocument :execresult
UPDATE pages SET document=?,revision=?,updated_at=? WHERE id=?;

-- name: TouchWorkspace :execresult
UPDATE workspaces SET updated_at=? WHERE id=(SELECT workspace_id FROM pages WHERE pages.id=?);

-- name: CreateSaveReceipt :execresult
INSERT INTO save_receipts(page_id,client_id,mutation_id,digest,revision,expires_at) VALUES(?,?,?,?,?,?);

-- name: OwnedWorkspace :one
SELECT id FROM workspaces WHERE id=? AND owner_id=?;

-- name: CreateRecoveryPage :execresult
INSERT INTO pages(id,workspace_id,name,position,document,revision,updated_at) SELECT ?,?,?,COALESCE(MAX(position),-1)+1,?,1,? FROM pages WHERE pages.workspace_id=?;

-- name: EnsureLease :execresult
INSERT OR IGNORE INTO page_leases(page_id) VALUES(?);

-- name: UpdateLease :execresult
UPDATE page_leases SET session_id=?,client_id=?,epoch=?,expires_at=? WHERE page_id=?;

-- name: CountOwnedAsset :one
SELECT count(*) FROM assets WHERE id=? AND owner_id=?;

-- name: AttachAsset :execresult
INSERT OR IGNORE INTO page_assets(page_id,asset_id) VALUES(?,?);

-- name: AssetMetadata :one
SELECT id,width,height,mime FROM assets WHERE id=? AND owner_id=?;

-- name: VariantBuckets :many
SELECT bucket FROM asset_variants WHERE asset_id=? ORDER BY bucket;

-- name: AssetDimensions :one
SELECT width,height FROM assets WHERE id=? AND owner_id=?;

-- name: CountVariants :one
SELECT count(*) FROM asset_variants WHERE asset_id=? AND bucket=?;

-- name: QueueVariant :execresult
INSERT OR IGNORE INTO jobs(id,user_id,asset_id,kind,dedup_key,payload,run_at) VALUES(?,?,?,'variant',?,?,?);

-- name: DueVariant :one
SELECT id,user_id,asset_id,payload,attempts FROM jobs WHERE kind='variant' AND attempts<3 AND run_at<=? ORDER BY run_at LIMIT 1;

-- name: OriginalImage :one
SELECT original,width,height FROM assets WHERE id=? AND owner_id=?;

-- name: DeleteJob :execresult
DELETE FROM jobs WHERE id=?;

-- name: CreateVariant :execresult
INSERT OR IGNORE INTO asset_variants(asset_id,bucket,mime,data,width,height) VALUES(?,?,?,?,?,?);

-- name: RetryVariant :execresult
UPDATE jobs SET attempts=attempts+1,run_at=?,last_error='Image preview generation failed.' WHERE id=?;

-- name: CreateAsset :execresult
INSERT INTO assets(id,owner_id,original,mime,width,height,digest,created_at) VALUES(?,?,?,?,?,?,?,?);

-- name: ImageVariant :one
SELECT v.data,v.mime,v.width,v.height FROM asset_variants v JOIN assets a ON a.id=v.asset_id WHERE a.id=? AND a.owner_id=? AND v.bucket=?;

-- name: GetImage :one
SELECT original,mime,width,height FROM assets WHERE id=? AND owner_id=?;

-- name: ExpireLeases :execresult
UPDATE page_leases SET epoch=epoch+1,expires_at=0,session_id=NULL,client_id='';

-- name: ListWorkspaces :many
SELECT w.id,w.name,w.revision,w.updated_at,(SELECT count(*) FROM pages p WHERE p.workspace_id=w.id) AS page_count FROM workspaces w WHERE w.owner_id=? ORDER BY w.updated_at DESC,w.id;

-- name: CreateWorkspace :execresult
INSERT INTO workspaces(id,owner_id,name,created_at,updated_at) VALUES(?,?,?,?,?);

-- name: WorkspaceRevision :one
SELECT revision FROM workspaces WHERE id=? AND owner_id=?;

-- name: RenameWorkspace :execresult
UPDATE workspaces SET name=?,revision=revision+1,updated_at=? WHERE id=?;

-- name: DeleteWorkspaceAssets :execresult
DELETE FROM assets WHERE owner_id=? AND id IN(SELECT pa.asset_id FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE p.workspace_id=?) AND NOT EXISTS(SELECT 1 FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE pa.asset_id=assets.id AND p.workspace_id<>?);

-- name: DeleteWorkspace :execresult
DELETE FROM workspaces WHERE id=? AND owner_id=?;

-- name: WorkspaceName :one
SELECT name FROM workspaces WHERE id=? AND owner_id=?;

-- name: PagesToCopy :many
SELECT id,name FROM pages WHERE pages.workspace_id=? ORDER BY position;

-- name: ListPages :many
SELECT id,workspace_id,name,position,revision,metadata_revision FROM pages WHERE pages.workspace_id=? ORDER BY position,id;

-- name: RenamePage :execresult
UPDATE pages SET name=?,metadata_revision=metadata_revision+1 WHERE id=?;

-- name: DeletePageAssets :execresult
DELETE FROM assets WHERE owner_id=? AND id IN(SELECT asset_id FROM page_assets WHERE page_assets.page_id=?) AND NOT EXISTS(SELECT 1 FROM page_assets pa WHERE pa.asset_id=assets.id AND pa.page_id<>?);

-- name: DeletePage :execresult
DELETE FROM pages WHERE id=?;

-- name: CountPages :one
SELECT count(*) FROM pages WHERE pages.workspace_id=?;

-- name: PositionPage :execresult
UPDATE pages SET position=? WHERE id=? AND workspace_id=?;

-- name: IncrementWorkspaceRevision :execresult
UPDATE workspaces SET revision=revision+1 WHERE id=?;

-- name: CreatePage :execresult
INSERT INTO pages(id,workspace_id,name,position,document,updated_at) SELECT ?,?,?,COALESCE(MAX(position),-1)+1,?,? FROM pages WHERE pages.workspace_id=?;

-- name: PageDocument :one
SELECT document FROM pages WHERE id=?;

-- name: CopyPageAssets :execresult
INSERT INTO page_assets(page_id,asset_id) SELECT ?,asset_id FROM page_assets WHERE page_assets.page_id=?;

-- name: OwnedPage :one
SELECT p.id,p.workspace_id,p.name,p.position,p.revision,p.metadata_revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: DeleteOrphanVariants :exec
DELETE FROM jobs WHERE asset_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM assets WHERE assets.id=jobs.asset_id);

-- name: DeleteExpiredReceipts :exec
DELETE FROM save_receipts WHERE expires_at<=unixepoch();

-- name: DeleteUnusedAssets :exec
DELETE FROM assets WHERE created_at<unixepoch()-86400 AND NOT EXISTS(SELECT 1 FROM page_assets WHERE page_assets.asset_id=assets.id);

