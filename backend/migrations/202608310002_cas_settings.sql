-- +goose Up
ALTER TABLE app_settings
ADD COLUMN cas_enabled BOOLEAN NOT NULL DEFAULT 0;

ALTER TABLE app_settings
ADD COLUMN cas_base_url TEXT NOT NULL DEFAULT '';

ALTER TABLE app_settings
ADD COLUMN cas_validation_url TEXT NOT NULL DEFAULT '';

ALTER TABLE app_settings
ADD COLUMN cas_validation_host TEXT NOT NULL DEFAULT '';

ALTER TABLE app_settings
ADD COLUMN cas_public_url TEXT NOT NULL DEFAULT '';

ALTER TABLE app_settings
ADD COLUMN cas_auto_create_users BOOLEAN NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE app_settings DROP COLUMN cas_auto_create_users;
ALTER TABLE app_settings DROP COLUMN cas_public_url;
ALTER TABLE app_settings DROP COLUMN cas_validation_host;
ALTER TABLE app_settings DROP COLUMN cas_validation_url;
ALTER TABLE app_settings DROP COLUMN cas_base_url;
ALTER TABLE app_settings DROP COLUMN cas_enabled;
