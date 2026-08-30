-- +goose Up
ALTER TABLE app_settings
ADD COLUMN brand_name_zh TEXT NOT NULL DEFAULT 'CPA-Helper';

ALTER TABLE app_settings
ADD COLUMN brand_name_en TEXT NOT NULL DEFAULT 'CPA-Helper';

ALTER TABLE app_settings
ADD COLUMN brand_subtitle_zh TEXT NOT NULL DEFAULT '边缘网关管理平台';

ALTER TABLE app_settings
ADD COLUMN brand_subtitle_en TEXT NOT NULL DEFAULT 'Edge Gateway Management Platform';

-- +goose Down
ALTER TABLE app_settings DROP COLUMN brand_subtitle_en;
ALTER TABLE app_settings DROP COLUMN brand_subtitle_zh;
ALTER TABLE app_settings DROP COLUMN brand_name_en;
ALTER TABLE app_settings DROP COLUMN brand_name_zh;
