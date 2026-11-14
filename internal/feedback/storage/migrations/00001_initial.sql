-- +goose Up
CREATE TABLE feedback_reports (
 id TEXT PRIMARY KEY,
 author_id TEXT NOT NULL,
 kind TEXT NOT NULL CHECK(kind IN ('bug','feature')),
 title TEXT NOT NULL,
 body TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'submitted' CHECK(status IN ('submitted','accepted','denied','closed')),
 release_version TEXT NOT NULL DEFAULT '',
 screenshot BLOB,
 created_at INTEGER NOT NULL,
 updated_at INTEGER NOT NULL
);
CREATE INDEX feedback_author ON feedback_reports(author_id,updated_at);
CREATE TABLE feedback_comments (
 id TEXT PRIMARY KEY,
 report_id TEXT NOT NULL REFERENCES feedback_reports(id) ON DELETE CASCADE,
 author_id TEXT NOT NULL,
 body TEXT NOT NULL,
 created_at INTEGER NOT NULL
);
CREATE INDEX feedback_thread ON feedback_comments(report_id,created_at);
-- +goose Down
DROP TABLE feedback_comments;
DROP TABLE feedback_reports;
