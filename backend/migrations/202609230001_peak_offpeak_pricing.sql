-- +goose Up
ALTER TABLE model_prices ADD COLUMN off_peak_enabled BOOLEAN NOT NULL DEFAULT 0 CHECK (off_peak_enabled IN (0, 1));
ALTER TABLE model_prices ADD COLUMN off_peak_input_usd_per_million REAL NOT NULL DEFAULT 0 CHECK (off_peak_input_usd_per_million >= 0);
ALTER TABLE model_prices ADD COLUMN off_peak_output_usd_per_million REAL NOT NULL DEFAULT 0 CHECK (off_peak_output_usd_per_million >= 0);
ALTER TABLE model_prices ADD COLUMN off_peak_cache_read_usd_per_million REAL NOT NULL DEFAULT 0 CHECK (off_peak_cache_read_usd_per_million >= 0);
ALTER TABLE model_prices ADD COLUMN off_peak_cache_creation_usd_per_million REAL NOT NULL DEFAULT 0 CHECK (off_peak_cache_creation_usd_per_million >= 0);
ALTER TABLE model_prices ADD COLUMN off_peak_request_usd REAL CHECK (off_peak_request_usd IS NULL OR off_peak_request_usd >= 0);
ALTER TABLE model_prices ADD COLUMN long_context_off_peak_input_usd_per_million REAL NOT NULL DEFAULT 0 CHECK (long_context_off_peak_input_usd_per_million >= 0);
ALTER TABLE model_prices ADD COLUMN long_context_off_peak_output_usd_per_million REAL NOT NULL DEFAULT 0 CHECK (long_context_off_peak_output_usd_per_million >= 0);
ALTER TABLE model_prices ADD COLUMN long_context_off_peak_cache_read_usd_per_million REAL NOT NULL DEFAULT 0 CHECK (long_context_off_peak_cache_read_usd_per_million >= 0);
ALTER TABLE model_prices ADD COLUMN long_context_off_peak_cache_creation_usd_per_million REAL NOT NULL DEFAULT 0 CHECK (long_context_off_peak_cache_creation_usd_per_million >= 0);

CREATE TABLE pricing_holiday_years (
    year INTEGER PRIMARY KEY,
    source_url TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL
);
CREATE TABLE pricing_holidays (
    date TEXT PRIMARY KEY,
    year INTEGER NOT NULL REFERENCES pricing_holiday_years(year) ON DELETE CASCADE
);
CREATE INDEX idx_pricing_holidays_year ON pricing_holidays(year);

-- State Council notice 国办发明电〔2025〕7号: all announced vacation days.
INSERT INTO pricing_holiday_years (year, source_url, updated_at) VALUES
    (2026, 'https://www.gov.cn/zhengce/zhengceku/202511/content_7047091.htm', '2026-09-23 00:00:00');
INSERT INTO pricing_holidays (date, year) VALUES
    ('2026-01-01', 2026), ('2026-01-02', 2026), ('2026-01-03', 2026),
    ('2026-02-15', 2026), ('2026-02-16', 2026), ('2026-02-17', 2026),
    ('2026-02-18', 2026), ('2026-02-19', 2026), ('2026-02-20', 2026),
    ('2026-02-21', 2026), ('2026-02-22', 2026), ('2026-02-23', 2026),
    ('2026-04-04', 2026), ('2026-04-05', 2026), ('2026-04-06', 2026),
    ('2026-05-01', 2026), ('2026-05-02', 2026), ('2026-05-03', 2026),
    ('2026-05-04', 2026), ('2026-05-05', 2026),
    ('2026-06-19', 2026), ('2026-06-20', 2026), ('2026-06-21', 2026),
    ('2026-09-25', 2026), ('2026-09-26', 2026), ('2026-09-27', 2026),
    ('2026-10-01', 2026), ('2026-10-02', 2026), ('2026-10-03', 2026),
    ('2026-10-04', 2026), ('2026-10-05', 2026), ('2026-10-06', 2026),
    ('2026-10-07', 2026);

-- +goose Down
DROP TABLE pricing_holidays;
DROP TABLE pricing_holiday_years;
ALTER TABLE model_prices DROP COLUMN long_context_off_peak_cache_creation_usd_per_million;
ALTER TABLE model_prices DROP COLUMN long_context_off_peak_cache_read_usd_per_million;
ALTER TABLE model_prices DROP COLUMN long_context_off_peak_output_usd_per_million;
ALTER TABLE model_prices DROP COLUMN long_context_off_peak_input_usd_per_million;
ALTER TABLE model_prices DROP COLUMN off_peak_request_usd;
ALTER TABLE model_prices DROP COLUMN off_peak_cache_creation_usd_per_million;
ALTER TABLE model_prices DROP COLUMN off_peak_cache_read_usd_per_million;
ALTER TABLE model_prices DROP COLUMN off_peak_output_usd_per_million;
ALTER TABLE model_prices DROP COLUMN off_peak_input_usd_per_million;
ALTER TABLE model_prices DROP COLUMN off_peak_enabled;
