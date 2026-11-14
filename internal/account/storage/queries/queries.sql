-- name: AccountCreateAdministrator :execresult
INSERT INTO users(id,email,password_hash,role,status,created_at,display_name) VALUES(?,?,?,'admin','active',?,?);

-- name: AccountUserByEmail :one
SELECT id,email,role,status,appearance,avatar_version,display_name,password_hash FROM users WHERE email=?;

-- name: AccountListUsers :many
SELECT u.id,u.email,u.role,u.status,u.appearance,u.avatar_version,u.display_name FROM users u ORDER BY created_at;

-- name: AccountCountEmail :one
SELECT count(*) FROM users WHERE email=?;

-- name: AccountCreateInvitedUser :execresult
INSERT INTO users(id,email,role,status,created_at,display_name) VALUES(?,?,?,'invited',?,?);

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
SELECT id,email,role,status,appearance,avatar_version,display_name FROM users WHERE id=?;

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

-- name: AvatarContent :one
SELECT content FROM avatars JOIN users ON users.id=avatars.user_id WHERE user_id=? AND users.status='active';
-- name: AvatarSet :exec
INSERT INTO avatars(user_id,content) VALUES(?,?) ON CONFLICT(user_id) DO UPDATE SET content=excluded.content;
-- name: AvatarDelete :exec
DELETE FROM avatars WHERE user_id=?;
-- name: AvatarVersion :exec
UPDATE users SET avatar_version=? WHERE id=?;

-- name: SessionActiveWithin :one
SELECT count(*) FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.id=? AND s.user_id=? AND s.expires_at>unixepoch() AND u.status='active';

-- name: AccountSetDisplayName :exec
UPDATE users SET display_name=? WHERE id=?;

-- name: AccountDirectory :many
SELECT id,display_name,avatar_version FROM users WHERE status='active' ORDER BY display_name COLLATE NOCASE,id;

-- name: ActivityTouch :exec
INSERT INTO account_activity(user_id,last_seen) VALUES(?,?)
ON CONFLICT(user_id) DO UPDATE SET last_seen=excluded.last_seen,cycle='',queued_at=0,deadline=0,final_queued_at=0,final_deadline=0
WHERE account_activity.last_seen < excluded.last_seen - 3600 OR account_activity.cycle != '';

-- name: ActivityCandidates :many
SELECT a.*,u.email FROM account_activity a JOIN users u ON u.id=a.user_id
WHERE u.status='active' AND u.role='user' AND a.last_seen<=?;

-- name: ActivityFind :one
SELECT a.*,u.email,u.role,u.status FROM account_activity a JOIN users u ON u.id=a.user_id WHERE a.user_id=?;

-- name: ActivityWarning :exec
UPDATE account_activity SET cycle=?,queued_at=?,deadline=0,final_queued_at=0,final_deadline=0 WHERE user_id=?;

-- name: ActivityDeadline :exec
UPDATE account_activity SET deadline=? WHERE user_id=?;

-- name: ActivityFinalQueued :exec
UPDATE account_activity SET final_queued_at=? WHERE user_id=?;

-- name: ActivityFinalDeadline :exec
UPDATE account_activity SET final_deadline=? WHERE user_id=?;

-- name: UpdatesPreference :one
SELECT CAST(COALESCE((SELECT enabled FROM account_update_preferences WHERE user_id=?),0) AS INTEGER) AS enabled;

-- name: UpdatesSetPreference :exec
INSERT INTO account_update_preferences(user_id,enabled) VALUES(?,?)
ON CONFLICT(user_id) DO UPDATE SET enabled=excluded.enabled;

-- name: UpdatesSubscribers :many
SELECT u.id,u.email,u.display_name FROM users u JOIN account_update_preferences p ON p.user_id=u.id WHERE u.status='active' AND p.enabled=1;

-- name: UpdatesSetToken :exec
INSERT INTO account_tokens(id,user_id,purpose,token_hash,expires_at) VALUES(sqlc.arg(token_hash),sqlc.arg(user_id),'updates_unsubscribe',sqlc.arg(token_hash),0);

-- name: UpdatesFindToken :one
SELECT t.user_id FROM account_tokens t JOIN account_update_preferences p ON p.user_id=t.user_id WHERE t.token_hash=? AND t.purpose='updates_unsubscribe' AND p.enabled=1;

-- name: UpdatesDeleteTokens :exec
DELETE FROM account_tokens WHERE user_id=? AND purpose='updates_unsubscribe';
