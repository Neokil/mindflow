-- +goose Up
ALTER TABLE users ADD COLUMN display_name TEXT NOT NULL DEFAULT 'User' CHECK(length(trim(display_name)) BETWEEN 1 AND 80);
UPDATE users SET display_name=substr(substr(email,1,instr(email,'@')-1),1,80) WHERE instr(email,'@')>1;

-- +goose Down
ALTER TABLE users DROP COLUMN display_name;
