-- name: EmailDueMail :one
SELECT id,payload,attempts FROM mail_jobs WHERE run_at<=? AND attempts<8 ORDER BY run_at LIMIT 1;

-- name: EmailDeleteJob :execresult
DELETE FROM mail_jobs WHERE id=?;

-- name: EmailFailMail :execresult
UPDATE mail_jobs SET attempts=?,payload='{}',last_error='Email delivery failed. Check SMTP settings and resend.' WHERE id=?;

-- name: EmailRetryMail :execresult
UPDATE mail_jobs SET attempts=?,run_at=?,last_error='Email delivery failed; retry scheduled.' WHERE id=?;

-- name: EmailDeleteUserMail :execresult
DELETE FROM mail_jobs WHERE user_id=?;

-- name: EmailDeletePurposeMail :execresult
DELETE FROM mail_jobs WHERE user_id=? AND dedup_key=?;

-- name: EmailQueueMail :execresult
INSERT INTO mail_jobs(id,user_id,dedup_key,payload,run_at) VALUES(?,?,?,?,?);

-- name: EmailRevokeMail :exec
DELETE FROM mail_jobs;

-- name: EmailDeliveryError :one
SELECT CAST(COALESCE((SELECT last_error FROM mail_jobs WHERE user_id=? ORDER BY run_at DESC LIMIT 1),'') AS TEXT);
