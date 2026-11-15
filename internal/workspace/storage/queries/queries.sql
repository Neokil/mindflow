-- name: WorkspaceListWorkspaces :many
SELECT w.id,w.name,w.revision,w.updated_at,wa.role,
 EXISTS(SELECT 1 FROM workspace_members m WHERE m.workspace_id=w.id) AS shared,
 (SELECT count(*) FROM active_pages p WHERE p.workspace_id=w.id) AS page_count,
 CAST(COALESCE(first_page.id,'') AS TEXT) AS first_page_id,
 CAST(COALESCE(first_page.revision,0) AS INTEGER) AS preview_revision,
 CAST(COALESCE(preview.content_id,'') AS TEXT) AS preview_id,
 CAST(COALESCE(preview.empty,0) AS INTEGER) AS preview_empty
FROM active_workspaces w JOIN workspace_access wa ON wa.workspace_id=w.id
LEFT JOIN active_pages first_page ON first_page.id=(SELECT p.id FROM active_pages p WHERE p.workspace_id=w.id ORDER BY p.position,p.id LIMIT 1)
LEFT JOIN workspace_previews preview ON preview.workspace_id=w.id AND preview.page_id=first_page.id
WHERE wa.user_id=? ORDER BY w.updated_at DESC,w.id;

-- name: WorkspaceCreateWorkspace :execresult
INSERT INTO workspaces(id,owner_id,name,created_at,updated_at) VALUES(?,?,?,?,?);

-- name: WorkspaceWorkspaceRevision :one
SELECT revision FROM active_workspaces WHERE id=? AND owner_id=?;

-- name: WorkspaceRenameWorkspace :execresult
UPDATE workspaces SET name=?,revision=revision+1,updated_at=? WHERE id=?;

-- name: WorkspaceDeleteWorkspace :execresult
DELETE FROM workspaces WHERE id=? AND owner_id=?;

-- name: WorkspaceWorkspaceName :one
SELECT name FROM active_workspaces WHERE id=? AND owner_id=?;

-- name: WorkspaceIncrementWorkspaceRevision :execresult
UPDATE workspaces SET revision=revision+1 WHERE id=?;

-- name: WorkspaceTouchWorkspace :execresult
UPDATE workspaces SET updated_at=? WHERE id=(SELECT workspace_id FROM pages WHERE pages.id=?);

-- name: PageOwnedWorkspace :one
SELECT id FROM active_workspaces WHERE id=? AND id IN (SELECT workspace_id FROM workspace_access WHERE user_id=sqlc.arg(owner_id));

-- name: PageCreateRecoveryPage :execresult
INSERT INTO pages(id,workspace_id,name,position,document,revision,updated_at) SELECT ?,?,?,COALESCE(MAX(position),-1)+1,?,1,? FROM pages WHERE pages.workspace_id=?;

-- name: PageListPages :many
SELECT id,workspace_id,name,position,revision,metadata_revision FROM active_pages WHERE active_pages.workspace_id=? ORDER BY position,id;

-- name: PageRenamePage :execresult
UPDATE pages SET name=?,metadata_revision=metadata_revision+1 WHERE id=?;

-- name: PageDeletePage :execresult
DELETE FROM pages WHERE id=?;

-- name: PageCountPages :one
SELECT count(*) FROM active_pages WHERE active_pages.workspace_id=?;

-- name: PagePositionPage :execresult
UPDATE pages SET position=? WHERE id=? AND workspace_id=?;

-- name: PageCreatePage :execresult
INSERT INTO pages(id,workspace_id,name,position,document,updated_at) SELECT ?,?,?,COALESCE(MAX(position),-1)+1,?,? FROM pages WHERE pages.workspace_id=?;

-- name: PagePageDocument :one
SELECT document FROM active_pages WHERE id=?;

-- name: PageOwnedPage :one
SELECT p.id,p.workspace_id,p.name,p.position,p.revision,p.metadata_revision FROM active_pages p JOIN active_workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.id IN (SELECT workspace_id FROM workspace_access WHERE user_id=sqlc.arg(owner_id));

-- name: PagePagesToCopy :many
SELECT id,name FROM active_pages WHERE active_pages.workspace_id=? ORDER BY position;

-- name: CanvasGetDocument :one
SELECT p.document,p.revision,p.metadata_revision,p.name,p.workspace_id FROM active_pages p JOIN active_workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.id IN (SELECT workspace_id FROM workspace_access WHERE user_id=sqlc.arg(owner_id));

-- name: CanvasOwnedPageRevision :one
SELECT p.revision FROM active_pages p JOIN active_workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.id IN (SELECT workspace_id FROM workspace_access WHERE user_id=sqlc.arg(owner_id));

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
SELECT p.id,p.workspace_id,p.name,p.position,p.revision,p.metadata_revision FROM active_pages p JOIN active_workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.id IN (SELECT workspace_id FROM workspace_access WHERE user_id=sqlc.arg(owner_id));

-- name: EditorOwnedPageRevision :one
SELECT p.revision FROM active_pages p JOIN active_workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.id IN (SELECT workspace_id FROM workspace_access WHERE user_id=sqlc.arg(owner_id));

-- name: EditorPageLease :one
SELECT session_id,client_id,epoch,expires_at FROM page_leases WHERE page_id=?;

-- name: EditorEnsureLease :execresult
INSERT OR IGNORE INTO page_leases(page_id) VALUES(?);

-- name: EditorUpdateLease :execresult
UPDATE page_leases SET session_id=?,client_id=?,epoch=?,expires_at=?,holder_user_id=? WHERE page_id=?;

-- name: EditorExpireLeases :execresult
UPDATE page_leases SET epoch=epoch+1,expires_at=0,session_id=NULL,client_id='',holder_user_id='';

-- name: ImageCountOwnedAsset :one
SELECT count(*) FROM assets WHERE id=? AND owner_id=?;

-- name: ImageAssetMetadata :one
SELECT id,width,height,mime FROM assets WHERE id=? AND id IN (SELECT asset_id FROM asset_access WHERE user_id=sqlc.arg(owner_id));

-- name: ImageVariantBuckets :many
SELECT bucket FROM asset_variants WHERE asset_id=? ORDER BY bucket;

-- name: ImageAssetDimensions :one
SELECT width,height,owner_id,mime FROM assets WHERE id=? AND id IN (SELECT asset_id FROM asset_access WHERE user_id=sqlc.arg(owner_id));

-- name: ImageCountVariants :one
SELECT count(*) FROM asset_variants WHERE asset_id=? AND bucket=?;

-- name: ImageQueueVariant :execresult
INSERT OR IGNORE INTO image_jobs(id,user_id,asset_id,dedup_key,payload,run_at) VALUES(?,?,?,?,?,?);

-- name: ImageDueVariant :one
SELECT id,user_id,asset_id,payload,attempts FROM image_jobs WHERE attempts<3 AND asset_id IN(SELECT id FROM active_assets) AND run_at<=? ORDER BY run_at LIMIT 1;

-- name: ImageOriginalImage :one
SELECT original,width,height FROM assets WHERE id=? AND id IN (SELECT asset_id FROM asset_access WHERE user_id=sqlc.arg(owner_id));

-- name: ImageDeleteJob :execresult
DELETE FROM image_jobs WHERE id=?;

-- name: ImageCreateVariant :execresult
INSERT OR IGNORE INTO asset_variants(asset_id,bucket,mime,data,width,height) VALUES(?,?,?,?,?,?);

-- name: ImageRetryVariant :execresult
UPDATE image_jobs SET attempts=attempts+1,run_at=?,last_error='Image preview generation failed.' WHERE id=?;

-- name: ImageCreateAsset :execresult
INSERT INTO assets(id,owner_id,original,mime,width,height,digest,created_at) VALUES(?,?,?,?,?,?,?,?);

-- name: ImageImageVariant :one
SELECT v.data,v.mime,v.width,v.height FROM asset_variants v JOIN assets a ON a.id=v.asset_id WHERE a.id=sqlc.arg(id) AND a.id IN (SELECT asset_id FROM asset_access WHERE user_id=sqlc.arg(owner_id)) AND v.bucket=sqlc.arg(bucket);

-- name: ImageGetImage :one
SELECT original,mime,width,height FROM assets WHERE id=? AND id IN (SELECT asset_id FROM asset_access WHERE user_id=sqlc.arg(owner_id));

-- name: ImageDeleteWorkspaceAssets :execresult
DELETE FROM assets WHERE owner_id=sqlc.arg(owner_id) AND (
 assets.workspace_id=sqlc.arg(workspace_id) OR id IN(SELECT pa.asset_id FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE p.workspace_id=sqlc.arg(workspace_id))
 OR id IN(SELECT ta.asset_id FROM template_assets ta JOIN workspace_templates t ON t.id=ta.template_id WHERE t.workspace_id=sqlc.arg(workspace_id))
) AND NOT EXISTS(SELECT 1 FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE pa.asset_id=assets.id AND p.workspace_id<>sqlc.arg(workspace_id))
 AND NOT EXISTS(SELECT 1 FROM template_assets ta JOIN workspace_templates t ON t.id=ta.template_id WHERE ta.asset_id=assets.id AND t.workspace_id<>sqlc.arg(workspace_id));

-- name: ImageDeletePageAssets :execresult
DELETE FROM assets WHERE owner_id=? AND id IN(SELECT asset_id FROM page_assets WHERE page_assets.page_id=?) AND NOT EXISTS(SELECT 1 FROM page_assets pa WHERE pa.asset_id=assets.id AND pa.page_id<>?) AND NOT EXISTS(SELECT 1 FROM template_assets ta WHERE ta.asset_id=assets.id);

-- name: ImageDeleteOrphanVariants :exec
DELETE FROM image_jobs WHERE asset_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM assets WHERE assets.id=image_jobs.asset_id);

-- name: ImageDeleteUnusedAssets :exec
DELETE FROM assets WHERE created_at<unixepoch()-86400 AND NOT EXISTS(SELECT 1 FROM page_assets WHERE page_assets.asset_id=assets.id) AND NOT EXISTS(SELECT 1 FROM template_assets WHERE template_assets.asset_id=assets.id);

-- name: WorkspaceDeleteOwnerWorkspaces :exec
DELETE FROM workspaces WHERE owner_id=?;

-- name: WorkspaceDeleteOwnerImages :exec
DELETE FROM assets WHERE owner_id=?;

-- name: AccessWorkspace :one
SELECT w.id AS workspace_id,w.owner_id,wa.role FROM active_workspaces w JOIN workspace_access wa ON wa.workspace_id=w.id WHERE w.id=? AND wa.user_id=?;

-- name: AccessPage :one
SELECT w.id AS workspace_id,w.owner_id,wa.role FROM active_pages p JOIN active_workspaces w ON w.id=p.workspace_id JOIN workspace_access wa ON wa.workspace_id=w.id WHERE p.id=? AND wa.user_id=?;

-- name: SharingMembers :many
SELECT user_id,email,role FROM workspace_members WHERE workspace_id=? ORDER BY email;

-- name: SharingSetMember :exec
INSERT INTO workspace_members(workspace_id,user_id,email,role) VALUES(?,?,?,?) ON CONFLICT(workspace_id,user_id) DO UPDATE SET role=excluded.role,email=excluded.email;

-- name: SharingRemoveMember :exec
DELETE FROM workspace_members WHERE workspace_id=? AND user_id=?;

-- name: SharingAudience :many
SELECT user_id FROM workspace_access WHERE workspace_id=?;

-- name: SharingInvalidateLeases :exec
UPDATE page_leases SET epoch=epoch+1,expires_at=0,session_id=NULL,client_id='',holder_user_id='' WHERE holder_user_id=? AND page_id IN(SELECT id FROM pages WHERE workspace_id=?);

-- name: SharingScopeAsset :exec
UPDATE assets SET workspace_id=? WHERE id=?;

-- name: SharingAssetForPage :one
SELECT count(*) FROM assets a JOIN active_pages p ON p.id=sqlc.arg(page_id) JOIN active_workspaces w ON w.id=p.workspace_id WHERE a.id=sqlc.arg(id) AND ((a.owner_id=sqlc.arg(owner_id) AND a.owner_id=w.owner_id) OR a.workspace_id=w.id OR EXISTS(SELECT 1 FROM page_assets pa JOIN active_pages linked ON linked.id=pa.page_id WHERE pa.asset_id=a.id AND linked.workspace_id=w.id) OR EXISTS(SELECT 1 FROM template_assets ta JOIN workspace_templates t ON t.id=ta.template_id WHERE ta.asset_id=a.id AND t.workspace_id=w.id));

-- name: SharingPageOwner :one
SELECT w.owner_id FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=?;

-- name: SharingAssetWorkspaces :many
SELECT a.workspace_id FROM assets a WHERE a.id=? AND a.workspace_id IS NOT NULL UNION SELECT p.workspace_id FROM page_assets pa JOIN active_pages p ON p.id=pa.page_id WHERE pa.asset_id=? UNION SELECT t.workspace_id FROM template_assets ta JOIN workspace_templates t ON t.id=ta.template_id WHERE ta.asset_id=sqlc.arg(template_asset_id);

-- name: SharingRemoveUserMemberships :exec
DELETE FROM workspace_members WHERE user_id=?;

-- name: CommentThreads :many
SELECT id,page_id,author_id,author_email,x,y,resolved,created_at FROM comment_threads WHERE page_id=? ORDER BY created_at,id;

-- name: CommentMessages :many
SELECT id,thread_id,author_id,author_email,body,created_at FROM comment_messages WHERE thread_id=? ORDER BY created_at,id;

-- name: CommentThread :one
SELECT id,page_id,author_id,resolved FROM comment_threads WHERE id=?;

-- name: CommentCreateThread :exec
INSERT INTO comment_threads(id,page_id,author_id,author_email,x,y,created_at) VALUES(?,?,?,?,?,?,?);

-- name: CommentCreateMessage :exec
INSERT INTO comment_messages(id,thread_id,author_id,author_email,body,created_at) VALUES(?,?,?,?,?,?);

-- name: CommentResolve :exec
UPDATE comment_threads SET resolved=? WHERE id=?;

-- name: CommentDelete :exec
DELETE FROM comment_threads WHERE id=?;

-- name: CommentDeleteUserThreads :exec
DELETE FROM comment_threads WHERE author_id=?;

-- name: CommentDeleteUserMessages :exec
DELETE FROM comment_messages WHERE author_id=?;

-- name: PageWorkspace :one
SELECT workspace_id FROM active_pages WHERE id=?;

-- name: CommentCountThreads :one
SELECT count(*) FROM comment_threads WHERE page_id=?;

-- name: CommentCountMessages :one
SELECT count(*) FROM comment_messages WHERE thread_id=?;

-- name: SharingInvalidateUserLeases :exec
UPDATE page_leases SET epoch=epoch+1,expires_at=0,session_id=NULL,client_id='',holder_user_id='' WHERE holder_user_id=?;

-- name: CanvasAssetMIME :one
SELECT mime FROM assets WHERE id=?;
