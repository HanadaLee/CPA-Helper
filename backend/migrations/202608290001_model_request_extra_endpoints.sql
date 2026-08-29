-- +goose Up
ALTER TABLE app_settings
ADD COLUMN model_request_extra_endpoints TEXT NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE app_settings DROP COLUMN model_request_extra_endpoints;
