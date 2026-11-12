-- name: AuthenticateSession :one
SELECT u.id,u.email,u.role,u.status,u.appearance,s.id AS session_id,s.csrf,s.verifier_hash FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.id=? AND s.expires_at>? AND u.status='active';

-- name: SetupStatus :one
SELECT initialized FROM app_settings WHERE id=1;

-- name: CreateAdministrator :execresult
INSERT INTO users(id,email,password_hash,role,status,created_at) VALUES(?,?,?,'admin','active',?);

-- name: CompleteSetup :execresult
UPDATE app_settings SET initialized=1,config=? WHERE id=1;

-- name: UserByEmail :one
SELECT id,email,role,status,appearance,password_hash FROM users WHERE email=?;

-- name: CreateSession :execresult
INSERT INTO sessions(id,user_id,verifier_hash,csrf,expires_at) VALUES(?,?,?,?,?);

-- name: ListUsers :many
SELECT u.id,u.email,u.role,u.status,u.appearance, CAST(COALESCE((SELECT last_error FROM jobs WHERE user_id=u.id AND kind='mail' ORDER BY run_at DESC LIMIT 1),'') AS TEXT) AS mail_error FROM users u ORDER BY created_at;

-- name: CountEmail :one
SELECT count(*) FROM users WHERE email=?;

-- name: CreateInvitedUser :execresult
INSERT INTO users(id,email,role,status,created_at) VALUES(?,?,?,'invited',?);

-- name: ActiveUserByEmail :one
SELECT id FROM users WHERE email=? AND status='active';

-- name: FindAccountToken :one
SELECT t.user_id,u.status FROM account_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=? AND t.purpose=? AND t.expires_at>?;

-- name: ActivateAccount :execresult
UPDATE users SET password_hash=?,status='active' WHERE id=?;

-- name: DeleteAccountTokens :execresult
DELETE FROM account_tokens WHERE user_id=?;

-- name: DeleteUserSessions :execresult
DELETE FROM sessions WHERE user_id=?;

-- name: DeleteUserMail :execresult
DELETE FROM jobs WHERE user_id=? AND kind='mail';

-- name: UserRoleStatus :one
SELECT role,status FROM users WHERE id=?;

-- name: UpdateUser :execresult
UPDATE users SET role=?,status=? WHERE id=?;

-- name: DeleteUser :execresult
DELETE FROM users WHERE id=?;

-- name: UpdateSettings :execresult
UPDATE app_settings SET config=? WHERE id=1;

-- name: Settings :one
SELECT config FROM app_settings WHERE id=1;

-- name: DeletePurposeTokens :execresult
DELETE FROM account_tokens WHERE user_id=? AND purpose=?;

-- name: DeletePurposeMail :execresult
DELETE FROM jobs WHERE user_id=? AND kind='mail' AND dedup_key=?;

-- name: CreateAccountToken :execresult
INSERT INTO account_tokens(id,user_id,purpose,token_hash,expires_at) VALUES(?,?,?,?,?);

-- name: QueueMail :execresult
INSERT INTO jobs(id,user_id,kind,dedup_key,payload,run_at) VALUES(?,?,'mail',?,?,?);

-- name: UserEmailStatus :one
SELECT email,status FROM users WHERE id=?;

-- name: CountAdministrators :one
SELECT count(*) FROM users WHERE role='admin' AND status='active';

-- name: DueMail :one
SELECT id,payload,attempts FROM jobs WHERE kind='mail' AND run_at<=? AND attempts<8 ORDER BY run_at LIMIT 1;

-- name: DeleteJob :execresult
DELETE FROM jobs WHERE id=?;

-- name: CountValidTokens :one
SELECT count(*) FROM account_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=? AND t.expires_at>? AND u.status IN ('active','invited');

-- name: FailMail :execresult
UPDATE jobs SET attempts=?,payload='{}',last_error='Email delivery failed. Check SMTP settings and resend.' WHERE id=?;

-- name: RetryMail :execresult
UPDATE jobs SET attempts=?,run_at=?,last_error='Email delivery failed; retry scheduled.' WHERE id=?;

-- name: DeleteSession :execresult
DELETE FROM sessions WHERE id=?;

-- name: SetAppearance :execresult
UPDATE users SET appearance=? WHERE id=?;

-- name: DeleteOrphanMail :exec
DELETE FROM jobs WHERE kind='mail' AND NOT EXISTS(SELECT 1 FROM account_tokens WHERE account_tokens.user_id=jobs.user_id AND account_tokens.expires_at>unixepoch());

-- name: DeleteExpiredTokens :exec
DELETE FROM account_tokens WHERE expires_at<=unixepoch();

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at<=unixepoch();


-- name: RevokeSessions :exec
DELETE FROM sessions;

-- name: RevokeTokens :exec
DELETE FROM account_tokens;

-- name: RevokeMail :exec
DELETE FROM jobs WHERE kind='mail';

