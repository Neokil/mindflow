-- +goose Up
CREATE TABLE IF NOT EXISTS app_settings(id INTEGER PRIMARY KEY CHECK(id=1), initialized INTEGER NOT NULL DEFAULT 0, config TEXT NOT NULL DEFAULT '{}');
INSERT OR IGNORE INTO app_settings(id) VALUES(1);

-- +goose Down
DROP TABLE IF EXISTS app_settings;
