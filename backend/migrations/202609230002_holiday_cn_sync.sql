-- +goose Up
ALTER TABLE pricing_holidays ADD COLUMN name TEXT NOT NULL DEFAULT '';
ALTER TABLE pricing_holidays ADD COLUMN is_off_day BOOLEAN NOT NULL DEFAULT 1 CHECK (is_off_day IN (0, 1));
ALTER TABLE pricing_holiday_years ADD COLUMN synced_at TEXT;

CREATE TABLE pricing_calendar_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    peak_on_makeup_days BOOLEAN NOT NULL DEFAULT 0 CHECK (peak_on_makeup_days IN (0, 1))
);
INSERT INTO pricing_calendar_settings (id, peak_on_makeup_days) VALUES (1, 0);

-- The existing 2026 off-days remain cached. Seed the published makeup days
-- until the first successful holiday-cn refresh, so enabling the option works
-- immediately after migration even when GitHub is temporarily unavailable.
INSERT OR IGNORE INTO pricing_holidays (date, year, name, is_off_day) VALUES
    ('2026-01-04', 2026, '元旦', 0),
    ('2026-02-14', 2026, '春节', 0),
    ('2026-02-28', 2026, '春节', 0),
    ('2026-05-09', 2026, '劳动节', 0),
    ('2026-09-20', 2026, '国庆节', 0),
    ('2026-10-10', 2026, '国庆节', 0);

-- +goose Down
DROP TABLE pricing_calendar_settings;
ALTER TABLE pricing_holiday_years DROP COLUMN synced_at;
ALTER TABLE pricing_holidays DROP COLUMN is_off_day;
ALTER TABLE pricing_holidays DROP COLUMN name;
