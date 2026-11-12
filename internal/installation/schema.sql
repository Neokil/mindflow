CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY);

CREATE TABLE IF NOT EXISTS app_settings(id INTEGER PRIMARY KEY, initialized INTEGER NOT NULL DEFAULT 0, config TEXT NOT NULL DEFAULT '{}');

INSERT OR IGNORE INTO schema_migrations(version) VALUES(1);
INSERT OR IGNORE INTO schema_migrations(version) VALUES(2);

