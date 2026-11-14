-- +goose Up
CREATE TABLE mail_deliveries (
 dedup_key TEXT PRIMARY KEY,
 user_id TEXT NOT NULL,
 delivered_at INTEGER NOT NULL
);
-- +goose Down
DROP TABLE mail_deliveries;
