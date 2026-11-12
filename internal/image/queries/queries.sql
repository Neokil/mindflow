-- name: CountOwnedAsset :one
SELECT count(*) FROM assets WHERE id=? AND owner_id=?;

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
DELETE FROM jobs WHERE id=? AND kind='variant';

-- name: CreateVariant :execresult
INSERT OR IGNORE INTO asset_variants(asset_id,bucket,mime,data,width,height) VALUES(?,?,?,?,?,?);

-- name: RetryVariant :execresult
UPDATE jobs SET attempts=attempts+1,run_at=?,last_error='Image preview generation failed.' WHERE id=? AND kind='variant';

-- name: CreateAsset :execresult
INSERT INTO assets(id,owner_id,original,mime,width,height,digest,created_at) VALUES(?,?,?,?,?,?,?,?);

-- name: ImageVariant :one
SELECT v.data,v.mime,v.width,v.height FROM asset_variants v JOIN assets a ON a.id=v.asset_id WHERE a.id=? AND a.owner_id=? AND v.bucket=?;

-- name: GetImage :one
SELECT original,mime,width,height FROM assets WHERE id=? AND owner_id=?;

-- name: DeleteWorkspaceAssets :execresult
DELETE FROM assets WHERE owner_id=? AND id IN(SELECT pa.asset_id FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE p.workspace_id=?) AND NOT EXISTS(SELECT 1 FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE pa.asset_id=assets.id AND p.workspace_id<>?);

-- name: DeletePageAssets :execresult
DELETE FROM assets WHERE owner_id=? AND id IN(SELECT asset_id FROM page_assets WHERE page_assets.page_id=?) AND NOT EXISTS(SELECT 1 FROM page_assets pa WHERE pa.asset_id=assets.id AND pa.page_id<>?);

-- name: DeleteOrphanVariants :exec
DELETE FROM jobs WHERE kind='variant' AND asset_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM assets WHERE assets.id=jobs.asset_id);

-- name: DeleteUnusedAssets :exec
DELETE FROM assets WHERE created_at<unixepoch()-86400 AND NOT EXISTS(SELECT 1 FROM page_assets WHERE page_assets.asset_id=assets.id);
