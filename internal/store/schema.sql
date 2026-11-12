CREATE TABLE IF NOT EXISTS users (
 id TEXT PRIMARY KEY, email TEXT NOT NULL UNIQUE COLLATE NOCASE, password_hash TEXT NOT NULL DEFAULT '',
 role TEXT NOT NULL CHECK(role IN ('admin','user')), status TEXT NOT NULL CHECK(status IN ('invited','active','inactive')),
 appearance TEXT NOT NULL DEFAULT 'system', created_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
 id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 verifier_hash TEXT NOT NULL, csrf TEXT NOT NULL, expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_user ON sessions(user_id);
CREATE TABLE IF NOT EXISTS account_tokens (
 id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 purpose TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, expires_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS jobs (
 id TEXT PRIMARY KEY, user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
 asset_id TEXT, kind TEXT NOT NULL, dedup_key TEXT UNIQUE, payload TEXT NOT NULL,
 attempts INTEGER NOT NULL DEFAULT 0, run_at INTEGER NOT NULL, last_error TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS jobs_due ON jobs(run_at);
INSERT OR IGNORE INTO schema_migrations(version) VALUES(1);
