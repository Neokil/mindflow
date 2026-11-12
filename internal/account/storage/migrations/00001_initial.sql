-- +goose Up
CREATE TABLE IF NOT EXISTS users (
 id TEXT NOT NULL PRIMARY KEY, email TEXT NOT NULL UNIQUE COLLATE NOCASE, password_hash TEXT NOT NULL DEFAULT '',
 role TEXT NOT NULL CHECK(role IN ('admin','user')), status TEXT NOT NULL CHECK(status IN ('invited','active','inactive')),
 appearance TEXT NOT NULL DEFAULT 'system', created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
 id TEXT NOT NULL PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 verifier_hash TEXT NOT NULL, csrf TEXT NOT NULL, expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_user ON sessions(user_id);

CREATE TABLE IF NOT EXISTS account_tokens (
 id TEXT NOT NULL PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 purpose TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, expires_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS account_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
