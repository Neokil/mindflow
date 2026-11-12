-- name: AccountCreateAdministrator :execresult
INSERT INTO users(id,email,password_hash,role,status,created_at) VALUES(?,?,?,'admin','active',?);

-- name: AccountUserByEmail :one
SELECT id,email,role,status,appearance,password_hash FROM users WHERE email=?;

-- name: AccountListUsers :many
SELECT u.id,u.email,u.role,u.status,u.appearance FROM users u ORDER BY created_at;

-- name: AccountCountEmail :one
SELECT count(*) FROM users WHERE email=?;

-- name: AccountCreateInvitedUser :execresult
INSERT INTO users(id,email,role,status,created_at) VALUES(?,?,?,'invited',?);

-- name: AccountActiveUserByEmail :one
SELECT id FROM users WHERE email=? AND status='active';

-- name: AccountActivateAccount :execresult
UPDATE users SET password_hash=?,status='active' WHERE id=?;

-- name: AccountUserRoleStatus :one
SELECT role,status FROM users WHERE id=?;

-- name: AccountUpdateUser :execresult
UPDATE users SET role=?,status=? WHERE id=?;

-- name: AccountDeleteUser :execresult
DELETE FROM users WHERE id=?;

-- name: AccountUserEmailStatus :one
SELECT email,status FROM users WHERE id=?;

-- name: AccountCountAdministrators :one
SELECT count(*) FROM users WHERE role='admin' AND status='active';

-- name: AccountSetAppearance :execresult
UPDATE users SET appearance=? WHERE id=?;

-- name: AccountFindUser :one
SELECT id,email,role,status,appearance FROM users WHERE id=?;

-- name: SessionCreateSession :execresult
INSERT INTO sessions(id,user_id,verifier_hash,csrf,expires_at) VALUES(?,?,?,?,?);

-- name: SessionDeleteUserSessions :execresult
DELETE FROM sessions WHERE user_id=?;

-- name: SessionDeleteSession :execresult
DELETE FROM sessions WHERE id=?;

-- name: SessionDeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at<=unixepoch();

-- name: SessionRevokeSessions :exec
DELETE FROM sessions;

-- name: SessionFindSession :one
SELECT id,user_id,csrf,verifier_hash FROM sessions WHERE id=? AND expires_at>?;

-- name: InvitationFindToken :one
SELECT user_id FROM account_tokens WHERE token_hash=? AND purpose='invite' AND expires_at>?;
-- name: InvitationDeleteUserTokens :execresult
DELETE FROM account_tokens WHERE user_id=? AND purpose='invite';
-- name: InvitationCreateToken :execresult
INSERT INTO account_tokens(id,user_id,purpose,token_hash,expires_at) VALUES(?,?,'invite',?,?);
-- name: InvitationRevokeTokens :exec
DELETE FROM account_tokens WHERE purpose='invite';
-- name: InvitationCountValidTokens :one
SELECT count(*) FROM account_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=? AND t.purpose='invite' AND t.expires_at>? AND u.status='invited';
-- name: InvitationDeleteExpiredTokens :exec
DELETE FROM account_tokens WHERE purpose='invite' AND expires_at<=unixepoch();

-- name: PasswordresetFindToken :one
SELECT user_id FROM account_tokens WHERE token_hash=? AND purpose='reset' AND expires_at>?;
-- name: PasswordresetDeleteUserTokens :execresult
DELETE FROM account_tokens WHERE user_id=? AND purpose='reset';
-- name: PasswordresetCreateToken :execresult
INSERT INTO account_tokens(id,user_id,purpose,token_hash,expires_at) VALUES(?,?,'reset',?,?);
-- name: PasswordresetRevokeTokens :exec
DELETE FROM account_tokens WHERE purpose='reset';
-- name: PasswordresetCountValidTokens :one
SELECT count(*) FROM account_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=? AND t.purpose='reset' AND t.expires_at>? AND u.status='active';
-- name: PasswordresetDeleteExpiredTokens :exec
DELETE FROM account_tokens WHERE purpose='reset' AND expires_at<=unixepoch();
