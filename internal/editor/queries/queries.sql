-- name: OwnedPage :one
SELECT p.id,p.workspace_id,p.name,p.position,p.revision,p.metadata_revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: OwnedPageRevision :one
SELECT p.revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: PageLease :one
SELECT session_id,client_id,epoch,expires_at FROM page_leases WHERE page_id=?;

-- name: EnsureLease :execresult
INSERT OR IGNORE INTO page_leases(page_id) VALUES(?);

-- name: UpdateLease :execresult
UPDATE page_leases SET session_id=?,client_id=?,epoch=?,expires_at=? WHERE page_id=?;

-- name: ExpireLeases :execresult
UPDATE page_leases SET epoch=epoch+1,expires_at=0,session_id=NULL,client_id='';
