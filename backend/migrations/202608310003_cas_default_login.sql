-- +goose Up
ALTER TABLE app_settings
ADD COLUMN cas_default_login BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE app_settings DROP COLUMN cas_default_login;
