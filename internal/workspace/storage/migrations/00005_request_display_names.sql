-- +goose Up
ALTER TABLE editor_requests ADD COLUMN requester_display_name TEXT NOT NULL DEFAULT 'User';
UPDATE editor_requests SET requester_display_name=substr(substr(requester_email,1,instr(requester_email,'@')-1),1,80) WHERE instr(requester_email,'@')>1;

-- +goose Down
ALTER TABLE editor_requests DROP COLUMN requester_display_name;
