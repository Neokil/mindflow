-- name: DueMail :one
SELECT id,payload,attempts FROM jobs WHERE kind='mail' AND run_at<=? AND attempts<8 ORDER BY run_at LIMIT 1;

-- name: DeleteJob :execresult
DELETE FROM jobs WHERE id=? AND kind='mail';

-- name: FailMail :execresult
UPDATE jobs SET attempts=?,payload='{}',last_error='Email delivery failed. Check SMTP settings and resend.' WHERE id=? AND kind='mail';

-- name: RetryMail :execresult
UPDATE jobs SET attempts=?,run_at=?,last_error='Email delivery failed; retry scheduled.' WHERE id=? AND kind='mail';

-- name: DeleteUserMail :execresult
DELETE FROM jobs WHERE user_id=? AND kind='mail';

-- name: DeletePurposeMail :execresult
DELETE FROM jobs WHERE user_id=? AND kind='mail' AND dedup_key=?;

-- name: QueueMail :execresult
INSERT INTO jobs(id,user_id,kind,dedup_key,payload,run_at) VALUES(?,?,'mail',?,?,?);

-- name: RevokeMail :exec
DELETE FROM jobs WHERE kind='mail';
