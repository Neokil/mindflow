-- +goose Up
CREATE TABLE IF NOT EXISTS account_tokens (
 id TEXT NOT NULL PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 purpose TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, expires_at INTEGER NOT NULL
);

-- +goose Down
-- The physical token table is also used by the other credential-link domain.
DELETE FROM account_tokens WHERE purpose='reset';
