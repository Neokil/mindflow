-- +goose Up
CREATE TABLE account_update_preferences (
 user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 enabled INTEGER NOT NULL DEFAULT 0 CHECK(enabled IN (0,1))
);
-- +goose Down
DROP TABLE account_update_preferences;
