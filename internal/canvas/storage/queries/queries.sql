-- name: GetDocument :one
SELECT p.document,p.revision,p.metadata_revision,p.name,p.workspace_id FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: OwnedPageRevision :one
SELECT p.revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?;

-- name: SaveReceipt :one
SELECT digest,revision FROM save_receipts WHERE page_id=? AND client_id=? AND mutation_id=? AND expires_at>?;

-- name: SaveDocument :execresult
UPDATE pages SET document=?,revision=?,updated_at=? WHERE id=?;

-- name: CreateSaveReceipt :execresult
INSERT INTO save_receipts(page_id,client_id,mutation_id,digest,revision,expires_at) VALUES(?,?,?,?,?,?);

-- name: AttachAsset :execresult
INSERT OR IGNORE INTO page_assets(page_id,asset_id) VALUES(?,?);

-- name: CopyPageAssets :execresult
INSERT INTO page_assets(page_id,asset_id) SELECT ?,asset_id FROM page_assets WHERE page_assets.page_id=?;

-- name: DeleteExpiredReceipts :exec
DELETE FROM save_receipts WHERE expires_at<=unixepoch();
