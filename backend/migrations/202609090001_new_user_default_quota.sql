-- +goose Up
ALTER TABLE app_settings
ADD COLUMN new_user_quota_unlimited BOOLEAN NOT NULL DEFAULT 0 CHECK (new_user_quota_unlimited IN (0, 1));
ALTER TABLE app_settings
ADD COLUMN new_user_quota_daily_usd REAL NOT NULL DEFAULT 0 CHECK (new_user_quota_daily_usd >= 0);
ALTER TABLE app_settings
ADD COLUMN new_user_quota_weekly_usd REAL NOT NULL DEFAULT 0 CHECK (new_user_quota_weekly_usd >= 0);
ALTER TABLE app_settings
ADD COLUMN new_user_quota_monthly_usd REAL NOT NULL DEFAULT 0 CHECK (new_user_quota_monthly_usd >= 0);
ALTER TABLE app_settings
ADD COLUMN new_user_quota_lifetime_usd REAL NOT NULL DEFAULT 0 CHECK (new_user_quota_lifetime_usd >= 0);

-- +goose Down
ALTER TABLE app_settings DROP COLUMN new_user_quota_lifetime_usd;
ALTER TABLE app_settings DROP COLUMN new_user_quota_monthly_usd;
ALTER TABLE app_settings DROP COLUMN new_user_quota_weekly_usd;
ALTER TABLE app_settings DROP COLUMN new_user_quota_daily_usd;
ALTER TABLE app_settings DROP COLUMN new_user_quota_unlimited;
