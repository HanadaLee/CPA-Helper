-- +goose Up
-- Daily/weekly counters now measure the same base-limit spend. Keep the unique
-- charged amount separately so history never sums the two counters twice.
ALTER TABLE user_quota_charges ADD COLUMN limit_deducted_usd REAL NOT NULL DEFAULT 0;
UPDATE user_quota_charges SET limit_deducted_usd = ROUND(daily_deducted_usd + weekly_deducted_usd, 8);

-- +goose Down
ALTER TABLE user_quota_charges DROP COLUMN limit_deducted_usd;
