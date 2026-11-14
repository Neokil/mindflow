-- name: TrashWorkspaces :many
SELECT id,name,deleted_at FROM workspaces WHERE owner_id=? AND deleted_at>sqlc.arg(cutoff) ORDER BY deleted_at DESC,id;
-- name: TrashPages :many
SELECT p.id,p.name,p.workspace_id,w.name AS workspace_name,p.deleted_at FROM pages p JOIN active_workspaces w ON w.id=p.workspace_id JOIN workspace_access wa ON wa.workspace_id=w.id WHERE wa.user_id=? AND wa.role IN('owner','full') AND p.deleted_at>sqlc.arg(cutoff) ORDER BY p.deleted_at DESC,p.id;
-- name: TrashWorkspace :execrows
UPDATE workspaces SET deleted_at=?,revision=revision+1 WHERE id=? AND owner_id=? AND deleted_at=0;
-- name: TrashPage :execrows
UPDATE pages SET deleted_at=?,metadata_revision=metadata_revision+1 WHERE id=? AND deleted_at=0;
-- name: TrashRestoreWorkspace :execrows
UPDATE workspaces SET deleted_at=0,revision=revision+1,updated_at=? WHERE id=? AND owner_id=? AND deleted_at>sqlc.arg(cutoff);
-- name: TrashPageParent :one
SELECT workspace_id FROM pages WHERE id=? AND deleted_at>sqlc.arg(cutoff);
-- name: TrashRestorePage :execrows
UPDATE pages SET deleted_at=0,metadata_revision=metadata_revision+1,position=(SELECT COALESCE(MAX(position),-1)+1 FROM pages p WHERE p.workspace_id=pages.workspace_id) WHERE pages.id=? AND pages.deleted_at>sqlc.arg(cutoff);
-- name: TrashRevokeWorkspace :exec
UPDATE page_leases SET epoch=epoch+1,expires_at=0,session_id=NULL,client_id='',holder_user_id='' WHERE page_id IN(SELECT id FROM pages WHERE workspace_id=?);
-- name: TrashRevokePage :exec
UPDATE page_leases SET epoch=epoch+1,expires_at=0,session_id=NULL,client_id='',holder_user_id='' WHERE page_id=?;
-- name: TrashCancelWorkspaceRequests :exec
DELETE FROM editor_requests WHERE page_id IN(SELECT id FROM pages WHERE workspace_id=?);
-- name: TrashCancelPageRequests :exec
DELETE FROM editor_requests WHERE page_id=?;
-- name: TrashExpiredWorkspaces :many
SELECT id,owner_id FROM workspaces WHERE deleted_at>0 AND deleted_at<=? ORDER BY deleted_at LIMIT 100;
-- name: TrashExpiredPages :many
SELECT id FROM pages WHERE deleted_at>0 AND deleted_at<=? ORDER BY deleted_at LIMIT 100;
