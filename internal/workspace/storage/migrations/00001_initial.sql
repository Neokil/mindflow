-- +goose Up
CREATE TABLE IF NOT EXISTS workspaces (
 id TEXT NOT NULL PRIMARY KEY, owner_id TEXT NOT NULL,
 name TEXT NOT NULL, revision INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS workspaces_owner ON workspaces(owner_id);

CREATE TABLE IF NOT EXISTS pages (
 id TEXT NOT NULL PRIMARY KEY, workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 name TEXT NOT NULL, position INTEGER NOT NULL, metadata_revision INTEGER NOT NULL DEFAULT 1,
 revision INTEGER NOT NULL DEFAULT 0, document TEXT NOT NULL, updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS pages_workspace ON pages(workspace_id,position);

CREATE TABLE IF NOT EXISTS assets (
 id TEXT NOT NULL PRIMARY KEY, owner_id TEXT NOT NULL,
 original BLOB NOT NULL, mime TEXT NOT NULL, width INTEGER NOT NULL, height INTEGER NOT NULL,
 digest TEXT NOT NULL, created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS assets_owner ON assets(owner_id);
CREATE TABLE IF NOT EXISTS asset_variants (
 asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE, bucket INTEGER NOT NULL,
 mime TEXT NOT NULL, data BLOB NOT NULL, width INTEGER NOT NULL, height INTEGER NOT NULL,
 PRIMARY KEY(asset_id,bucket)
);

CREATE TABLE IF NOT EXISTS page_assets (
 page_id TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
 asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
 PRIMARY KEY(page_id,asset_id)
);

CREATE TABLE IF NOT EXISTS save_receipts (
 page_id TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
 client_id TEXT NOT NULL, mutation_id TEXT NOT NULL, digest TEXT NOT NULL,
 revision INTEGER NOT NULL, expires_at INTEGER NOT NULL,
 PRIMARY KEY(page_id,client_id,mutation_id)
);

CREATE TABLE IF NOT EXISTS page_leases (
 page_id TEXT NOT NULL PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
 session_id TEXT,
 client_id TEXT NOT NULL DEFAULT '', epoch INTEGER NOT NULL DEFAULT 0, expires_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS image_jobs (
 id TEXT NOT NULL PRIMARY KEY, user_id TEXT,
 asset_id TEXT REFERENCES assets(id) ON DELETE CASCADE, dedup_key TEXT UNIQUE, payload TEXT NOT NULL,
 attempts INTEGER NOT NULL DEFAULT 0, run_at INTEGER NOT NULL, last_error TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS image_jobs_due ON image_jobs(run_at);

-- +goose Down
DROP TABLE IF EXISTS image_jobs;
DROP TABLE IF EXISTS page_leases;
DROP TABLE IF EXISTS save_receipts;
DROP TABLE IF EXISTS page_assets;
DROP TABLE IF EXISTS asset_variants;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS pages;
DROP TABLE IF EXISTS workspaces;
