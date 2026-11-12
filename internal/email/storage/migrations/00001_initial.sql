-- +goose Up
CREATE TABLE IF NOT EXISTS mail_jobs (
 id TEXT NOT NULL PRIMARY KEY, user_id TEXT,
 dedup_key TEXT UNIQUE, payload TEXT NOT NULL,
 attempts INTEGER NOT NULL DEFAULT 0, run_at INTEGER NOT NULL, last_error TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS mail_jobs_due ON mail_jobs(run_at);

-- +goose Down
DROP TABLE IF EXISTS mail_jobs;
