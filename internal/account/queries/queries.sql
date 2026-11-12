-- name: CreateAdministrator :execresult
INSERT INTO users(id,email,password_hash,role,status,created_at) VALUES(?,?,?,'admin','active',?);

-- name: UserByEmail :one
SELECT id,email,role,status,appearance,password_hash FROM users WHERE email=?;

-- name: ListUsers :many
SELECT u.id,u.email,u.role,u.status,u.appearance, CAST(COALESCE((SELECT last_error FROM jobs WHERE user_id=u.id AND kind='mail' ORDER BY run_at DESC LIMIT 1),'') AS TEXT) AS mail_error FROM users u ORDER BY created_at;

-- name: CountEmail :one
SELECT count(*) FROM users WHERE email=?;

-- name: CreateInvitedUser :execresult
INSERT INTO users(id,email,role,status,created_at) VALUES(?,?,?,'invited',?);

-- name: ActiveUserByEmail :one
SELECT id FROM users WHERE email=? AND status='active';

-- name: ActivateAccount :execresult
UPDATE users SET password_hash=?,status='active' WHERE id=?;

-- name: UserRoleStatus :one
SELECT role,status FROM users WHERE id=?;

-- name: UpdateUser :execresult
UPDATE users SET role=?,status=? WHERE id=?;

-- name: DeleteUser :execresult
DELETE FROM users WHERE id=?;

-- name: UserEmailStatus :one
SELECT email,status FROM users WHERE id=?;

-- name: CountAdministrators :one
SELECT count(*) FROM users WHERE role='admin' AND status='active';

-- name: SetAppearance :execresult
UPDATE users SET appearance=? WHERE id=?;

-- name: FindUser :one
SELECT id,email,role,status,appearance FROM users WHERE id=?;
