-- +goose Up
CREATE TABLE editor_requests (
 page_id TEXT PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
 id TEXT NOT NULL,
 requester_id TEXT NOT NULL,
 requester_email TEXT NOT NULL,
 requester_avatar_version TEXT NOT NULL DEFAULT '',
 requester_session TEXT NOT NULL,
 requester_client TEXT NOT NULL,
 holder_session TEXT NOT NULL,
 holder_client TEXT NOT NULL,
 holder_epoch INTEGER NOT NULL,
 deadline INTEGER NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('pending','accepted','denied','cancelled'))
);

-- +goose Down
DROP TABLE editor_requests;
