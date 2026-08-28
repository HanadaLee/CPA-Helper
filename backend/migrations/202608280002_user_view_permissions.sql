-- +goose Up
ALTER TABLE app_settings
ADD COLUMN allow_user_account_status BOOLEAN NOT NULL DEFAULT 0;

ALTER TABLE app_settings
ADD COLUMN allow_user_usage_history BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE app_settings DROP COLUMN allow_user_usage_history;
ALTER TABLE app_settings DROP COLUMN allow_user_account_status;
