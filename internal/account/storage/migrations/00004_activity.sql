-- +goose Up
CREATE TABLE account_activity (
 user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 last_seen INTEGER NOT NULL,
 cycle TEXT NOT NULL DEFAULT '',
 queued_at INTEGER NOT NULL DEFAULT 0,
 deadline INTEGER NOT NULL DEFAULT 0,
 final_queued_at INTEGER NOT NULL DEFAULT 0,
 final_deadline INTEGER NOT NULL DEFAULT 0
);
INSERT INTO account_activity(user_id,last_seen) SELECT id,unixepoch() FROM users;

-- +goose Down
DROP TABLE account_activity;
