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

