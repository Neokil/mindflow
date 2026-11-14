-- +goose Up
CREATE TABLE release_state (id INTEGER PRIMARY KEY CHECK(id=1),version TEXT NOT NULL);
-- +goose Down
DROP TABLE release_state;
