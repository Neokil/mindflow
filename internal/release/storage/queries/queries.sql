-- name: ReleaseCurrent :one
SELECT version FROM release_state WHERE id=1;
-- name: ReleaseObserve :exec
INSERT INTO release_state(id,version) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET version=excluded.version;
