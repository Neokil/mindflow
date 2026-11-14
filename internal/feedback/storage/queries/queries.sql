-- name: FeedbackCreate :exec
INSERT INTO feedback_reports(id,author_id,kind,title,body,screenshot,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?);
-- name: FeedbackList :many
SELECT r.id,r.author_id,r.kind,r.title,r.status,r.release_version,r.created_at,r.updated_at,r.closed_at,
 (SELECT count(*) FROM feedback_events e WHERE e.report_id=r.id AND e.actor_id!=sqlc.arg(user_id)
 AND (CAST(sqlc.arg(administrator) AS INTEGER)=1 OR e.kind='comment')
 AND e.id>COALESCE((SELECT through_event FROM feedback_reads WHERE report_id=r.id AND user_id=sqlc.arg(user_id)),0)) AS unread
 FROM feedback_reports r WHERE (r.author_id=sqlc.arg(user_id) OR CAST(sqlc.arg(administrator) AS INTEGER)=1)
 AND (CAST(sqlc.arg(search) AS TEXT)='' OR instr(lower(r.title),lower(CAST(sqlc.arg(search) AS TEXT)))>0
 OR instr(lower(r.body),lower(CAST(sqlc.arg(search) AS TEXT)))>0
 OR EXISTS(SELECT 1 FROM feedback_comments c WHERE c.report_id=r.id AND instr(lower(c.body),lower(CAST(sqlc.arg(search) AS TEXT)))>0))
 ORDER BY r.updated_at DESC,(SELECT MAX(id) FROM feedback_events WHERE report_id=r.id) DESC,r.id;
-- name: FeedbackFind :one
SELECT id,author_id,kind,title,body,status,release_version,created_at,updated_at,closed_at,CAST(screenshot IS NOT NULL AS INTEGER) AS has_screenshot FROM feedback_reports WHERE id=? AND (author_id=? OR CAST(sqlc.arg(administrator) AS INTEGER)=1);
-- name: FeedbackScreenshot :one
SELECT screenshot FROM feedback_reports WHERE id=? AND (author_id=? OR CAST(sqlc.arg(administrator) AS INTEGER)=1);
-- name: FeedbackComments :many
SELECT c.id,c.author_id,c.body,c.created_at FROM feedback_comments c JOIN feedback_events e ON e.comment_id=c.id WHERE c.report_id=? ORDER BY e.id;
-- name: FeedbackComment :exec
INSERT INTO feedback_comments(id,report_id,author_id,body,created_at) VALUES(?,?,?,?,?);
-- name: FeedbackTouch :exec
UPDATE feedback_reports SET updated_at=? WHERE id=?;
-- name: FeedbackReview :exec
UPDATE feedback_reports SET status=?,release_version=?,updated_at=?,closed_at=? WHERE id=?;
-- name: FeedbackDeleteAuthorComments :exec
DELETE FROM feedback_comments WHERE author_id=?;
-- name: FeedbackDeleteAuthorReports :exec
DELETE FROM feedback_reports WHERE author_id=?;

-- name: FeedbackEvent :exec
INSERT INTO feedback_events(report_id,actor_id,kind,comment_id) VALUES(?,?,?,?);
-- name: FeedbackThrough :one
SELECT CAST(COALESCE(MAX(id),0) AS INTEGER) AS through_event FROM feedback_events WHERE report_id=?;
-- name: FeedbackMarkRead :exec
INSERT INTO feedback_reads(report_id,user_id,through_event) VALUES(?,?,?)
ON CONFLICT(report_id,user_id) DO UPDATE SET through_event=MAX(feedback_reads.through_event,excluded.through_event);
-- name: FeedbackUnread :one
SELECT
 (SELECT count(*) FROM feedback_events e JOIN feedback_reports r ON r.id=e.report_id
 WHERE CAST(sqlc.arg(administrator) AS INTEGER)=1 AND e.actor_id!=sqlc.arg(user_id)
 AND e.id>COALESCE((SELECT through_event FROM feedback_reads WHERE report_id=r.id AND user_id=sqlc.arg(user_id)),0)) AS reports,
 (SELECT count(*) FROM feedback_events e JOIN feedback_reports r ON r.id=e.report_id
 WHERE r.author_id=sqlc.arg(user_id) AND e.kind='comment' AND e.actor_id!=sqlc.arg(user_id)
 AND e.id>COALESCE((SELECT through_event FROM feedback_reads WHERE report_id=r.id AND user_id=sqlc.arg(user_id)),0)) AS replies;
-- name: FeedbackDeleteUserReads :exec
DELETE FROM feedback_reads WHERE user_id=?;

-- name: FeedbackDeleteExpired :exec
DELETE FROM feedback_reports WHERE status='closed' AND closed_at>0 AND closed_at<=?;
