-- name: FindToken :one
SELECT user_id FROM account_tokens WHERE token_hash=? AND purpose='reset' AND expires_at>?;
-- name: DeleteUserTokens :execresult
DELETE FROM account_tokens WHERE user_id=? AND purpose='reset';
-- name: CreateToken :execresult
INSERT INTO account_tokens(id,user_id,purpose,token_hash,expires_at) VALUES(?,?,'reset',?,?);
-- name: RevokeTokens :exec
DELETE FROM account_tokens WHERE purpose='reset';
-- name: CountValidTokens :one
SELECT count(*) FROM account_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=? AND t.purpose='reset' AND t.expires_at>? AND u.status='active';
-- name: DeleteExpiredTokens :exec
DELETE FROM account_tokens WHERE purpose='reset' AND expires_at<=unixepoch();
