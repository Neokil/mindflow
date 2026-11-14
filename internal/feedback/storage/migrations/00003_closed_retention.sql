-- +goose Up
ALTER TABLE feedback_reports ADD COLUMN closed_at INTEGER NOT NULL DEFAULT 0;
-- Give reports closed before this policy a full retention window after upgrade.
UPDATE feedback_reports SET closed_at=unixepoch() WHERE status='closed';
CREATE INDEX feedback_closed_retention ON feedback_reports(closed_at) WHERE status='closed';
-- +goose Down
DROP INDEX feedback_closed_retention;
ALTER TABLE feedback_reports DROP COLUMN closed_at;
