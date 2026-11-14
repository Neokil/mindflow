-- +goose Up
ALTER TABLE feedback_reports ADD COLUMN state TEXT NOT NULL DEFAULT 'created' CHECK(state IN ('created','accepted','closed'));
UPDATE feedback_reports SET state=CASE status WHEN 'submitted' THEN 'created' WHEN 'denied' THEN 'closed' ELSE status END;
ALTER TABLE feedback_reports DROP COLUMN status;
ALTER TABLE feedback_reports RENAME COLUMN state TO status;

CREATE TABLE feedback_events (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 report_id TEXT NOT NULL REFERENCES feedback_reports(id) ON DELETE CASCADE,
 actor_id TEXT NOT NULL,
 kind TEXT NOT NULL CHECK(kind IN ('report','comment')),
 comment_id TEXT REFERENCES feedback_comments(id) ON DELETE CASCADE
);
CREATE INDEX feedback_event_report ON feedback_events(report_id,id);
CREATE UNIQUE INDEX feedback_event_comment ON feedback_events(comment_id) WHERE comment_id IS NOT NULL;
CREATE TABLE feedback_reads (
 report_id TEXT NOT NULL REFERENCES feedback_reports(id) ON DELETE CASCADE,
 user_id TEXT NOT NULL,
 through_event INTEGER NOT NULL DEFAULT 0,
 PRIMARY KEY(report_id,user_id)
);
INSERT INTO feedback_events(report_id,actor_id,kind)
 SELECT id,author_id,'report' FROM feedback_reports ORDER BY created_at,id;
INSERT INTO feedback_events(report_id,actor_id,kind,comment_id)
 SELECT report_id,author_id,'comment',id FROM feedback_comments ORDER BY created_at,id;

-- +goose Down
DROP TABLE feedback_reads;
DROP TABLE feedback_events;
ALTER TABLE feedback_reports ADD COLUMN old_status TEXT NOT NULL DEFAULT 'submitted' CHECK(old_status IN ('submitted','accepted','denied','closed'));
UPDATE feedback_reports SET old_status=CASE status WHEN 'created' THEN 'submitted' ELSE status END;
ALTER TABLE feedback_reports DROP COLUMN status;
ALTER TABLE feedback_reports RENAME COLUMN old_status TO status;
