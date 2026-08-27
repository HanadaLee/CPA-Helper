-- +goose Up
ALTER TABLE app_settings
ADD COLUMN cpamc_url VARCHAR(2000) NOT NULL DEFAULT '/management.html';

-- +goose Down
ALTER TABLE app_settings DROP COLUMN cpamc_url;
