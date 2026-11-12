-- name: OwnedWorkspace :one
SELECT id FROM workspaces WHERE id=? AND owner_id=?;

-- name: CreateRecoveryPage :execresult
INSERT INTO pages(id,workspace_id,name,position,document,revision,updated_at) SELECT ?,?,?,COALESCE(MAX(position),-1)+1,?,1,? FROM pages WHERE pages.workspace_id=?;

-- name: ListPages :many
SELECT id,workspace_id,name,position,revision,metadata_revision FROM pages WHERE pages.workspace_id=? ORDER BY position,id;

-- name: RenamePage :execresult
UPDATE pages SET name=?,metadata_revision=metadata_revision+1 WHERE id=?;

-- name: DeletePage :execresult
DELETE FROM pages WHERE id=?;

-- name: CountPages :one
SELECT count(*) FROM pages WHERE pages.workspace_id=?;

-- name: PositionPage :execresult
UPDATE pages SET position=? WHERE id=? AND workspace_id=?;

-- name: CreatePage :execresult
INSERT INTO pages(id,workspace_id,name,position,document,updated_at) SELECT ?,?,?,COALESCE(MAX(position),-1)+1,?,? FROM pages WHERE pages.workspace_id=?;

-- name: PageDocument :one
SELECT document FROM pages WHERE id=?;

-- name: OwnedPage :one
SELECT p.id,p.workspace_id,p.name,p.position,p.revision,p.metadata_revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: PagesToCopy :many
SELECT id,name FROM pages WHERE pages.workspace_id=? ORDER BY position;
