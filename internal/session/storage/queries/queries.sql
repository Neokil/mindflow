-- name: CreateSession :execresult
INSERT INTO sessions(id,user_id,verifier_hash,csrf,expires_at) VALUES(?,?,?,?,?);

-- name: DeleteUserSessions :execresult
DELETE FROM sessions WHERE user_id=?;

-- name: DeleteSession :execresult
DELETE FROM sessions WHERE id=?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at<=unixepoch();

-- name: RevokeSessions :exec
DELETE FROM sessions;

-- name: FindSession :one
SELECT id,user_id,csrf,verifier_hash FROM sessions WHERE id=? AND expires_at>?;
