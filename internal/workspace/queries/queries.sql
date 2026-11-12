-- name: ListWorkspaces :many
SELECT w.id,w.name,w.revision,w.updated_at,(SELECT count(*) FROM pages p WHERE p.workspace_id=w.id) AS page_count FROM workspaces w WHERE w.owner_id=? ORDER BY w.updated_at DESC,w.id;

-- name: CreateWorkspace :execresult
INSERT INTO workspaces(id,owner_id,name,created_at,updated_at) VALUES(?,?,?,?,?);

-- name: WorkspaceRevision :one
SELECT revision FROM workspaces WHERE id=? AND owner_id=?;

-- name: RenameWorkspace :execresult
UPDATE workspaces SET name=?,revision=revision+1,updated_at=? WHERE id=?;

-- name: DeleteWorkspace :execresult
DELETE FROM workspaces WHERE id=? AND owner_id=?;

-- name: WorkspaceName :one
SELECT name FROM workspaces WHERE id=? AND owner_id=?;

-- name: IncrementWorkspaceRevision :execresult
UPDATE workspaces SET revision=revision+1 WHERE id=?;

-- name: TouchWorkspace :execresult
UPDATE workspaces SET updated_at=? WHERE id=(SELECT workspace_id FROM pages WHERE pages.id=?);
