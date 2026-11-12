-- +goose Up
CREATE TABLE IF NOT EXISTS users (
 id TEXT NOT NULL PRIMARY KEY, email TEXT NOT NULL UNIQUE COLLATE NOCASE, password_hash TEXT NOT NULL DEFAULT '',
 role TEXT NOT NULL CHECK(role IN ('admin','user')), status TEXT NOT NULL CHECK(status IN ('invited','active','inactive')),
 appearance TEXT NOT NULL DEFAULT 'system', created_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS users;
