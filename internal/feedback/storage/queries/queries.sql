-- name: FeedbackCreate :exec
INSERT INTO feedback_reports(id,author_id,kind,title,body,screenshot,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?);
-- name: FeedbackList :many
SELECT id,author_id,kind,title,status,release_version,created_at,updated_at FROM feedback_reports WHERE author_id=? OR CAST(sqlc.arg(administrator) AS INTEGER)=1 ORDER BY updated_at DESC ;
-- name: FeedbackFind :one
SELECT id,author_id,kind,title,body,status,release_version,created_at,updated_at,CAST(screenshot IS NOT NULL AS INTEGER) AS has_screenshot FROM feedback_reports WHERE id=? AND (author_id=? OR CAST(sqlc.arg(administrator) AS INTEGER)=1);
-- name: FeedbackScreenshot :one
SELECT screenshot FROM feedback_reports WHERE id=? AND (author_id=? OR CAST(sqlc.arg(administrator) AS INTEGER)=1);
-- name: FeedbackComments :many
SELECT id,author_id,body,created_at FROM feedback_comments WHERE report_id=? ORDER BY created_at,id;
-- name: FeedbackComment :exec
INSERT INTO feedback_comments(id,report_id,author_id,body,created_at) VALUES(?,?,?,?,?);
-- name: FeedbackTouch :exec
UPDATE feedback_reports SET updated_at=? WHERE id=?;
-- name: FeedbackReview :exec
UPDATE feedback_reports SET status=?,release_version=?,updated_at=? WHERE id=?;
-- name: FeedbackDeleteAuthorComments :exec
DELETE FROM feedback_comments WHERE author_id=?;
-- name: FeedbackDeleteAuthorReports :exec
DELETE FROM feedback_reports WHERE author_id=?;
