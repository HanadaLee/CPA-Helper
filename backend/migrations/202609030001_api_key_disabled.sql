-- +goose Up
ALTER TABLE user_api_keys
ADD COLUMN disabled BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE user_api_keys DROP COLUMN disabled;
