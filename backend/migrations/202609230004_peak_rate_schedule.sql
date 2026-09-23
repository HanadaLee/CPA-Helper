-- +goose Up
ALTER TABLE model_prices RENAME COLUMN off_peak_input_usd_per_million TO peak_input_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN off_peak_output_usd_per_million TO peak_output_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN off_peak_cache_read_usd_per_million TO peak_cache_read_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN off_peak_cache_creation_usd_per_million TO peak_cache_creation_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN off_peak_request_usd TO peak_request_usd;
ALTER TABLE model_prices RENAME COLUMN long_context_off_peak_input_usd_per_million TO long_context_peak_input_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN long_context_off_peak_output_usd_per_million TO long_context_peak_output_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN long_context_off_peak_cache_read_usd_per_million TO long_context_peak_cache_read_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN long_context_off_peak_cache_creation_usd_per_million TO long_context_peak_cache_creation_usd_per_million;

-- Before this migration the base columns held peak rates. Swap the two tiers
-- for configured models so base prices become the ordinary off-peak rates.
UPDATE model_prices SET
    input_usd_per_million = CASE WHEN off_peak_enabled THEN peak_input_usd_per_million ELSE input_usd_per_million END,
    peak_input_usd_per_million = CASE WHEN off_peak_enabled THEN input_usd_per_million ELSE peak_input_usd_per_million END,
    output_usd_per_million = CASE WHEN off_peak_enabled THEN peak_output_usd_per_million ELSE output_usd_per_million END,
    peak_output_usd_per_million = CASE WHEN off_peak_enabled THEN output_usd_per_million ELSE peak_output_usd_per_million END,
    cache_read_usd_per_million = CASE WHEN off_peak_enabled THEN peak_cache_read_usd_per_million ELSE cache_read_usd_per_million END,
    peak_cache_read_usd_per_million = CASE WHEN off_peak_enabled THEN cache_read_usd_per_million ELSE peak_cache_read_usd_per_million END,
    cache_creation_usd_per_million = CASE WHEN off_peak_enabled THEN peak_cache_creation_usd_per_million ELSE cache_creation_usd_per_million END,
    peak_cache_creation_usd_per_million = CASE WHEN off_peak_enabled THEN cache_creation_usd_per_million ELSE peak_cache_creation_usd_per_million END,
    request_usd = CASE WHEN off_peak_enabled THEN peak_request_usd ELSE request_usd END,
    peak_request_usd = CASE WHEN off_peak_enabled THEN request_usd ELSE peak_request_usd END,
    long_context_input_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_peak_input_usd_per_million ELSE long_context_input_usd_per_million END,
    long_context_peak_input_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_input_usd_per_million ELSE long_context_peak_input_usd_per_million END,
    long_context_output_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_peak_output_usd_per_million ELSE long_context_output_usd_per_million END,
    long_context_peak_output_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_output_usd_per_million ELSE long_context_peak_output_usd_per_million END,
    long_context_cache_read_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_peak_cache_read_usd_per_million ELSE long_context_cache_read_usd_per_million END,
    long_context_peak_cache_read_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_cache_read_usd_per_million ELSE long_context_peak_cache_read_usd_per_million END,
    long_context_cache_creation_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_peak_cache_creation_usd_per_million ELSE long_context_cache_creation_usd_per_million END,
    long_context_peak_cache_creation_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_cache_creation_usd_per_million ELSE long_context_peak_cache_creation_usd_per_million END;

-- A previously customised valley/base price is not a LiteLLM-sourced base
-- price. Keep its exact value instead of silently replacing it at next sync.
UPDATE model_prices SET source = 'manual', source_model = NULL, auto_synced = 0, last_synced_at = NULL
WHERE off_peak_enabled = 1 AND auto_synced = 1;

ALTER TABLE pricing_calendar_settings ADD COLUMN peak_periods_json TEXT NOT NULL DEFAULT '[{"start":"09:00","end":"12:00"},{"start":"14:00","end":"18:00"}]';

-- +goose Down
ALTER TABLE pricing_calendar_settings DROP COLUMN peak_periods_json;
UPDATE model_prices SET
    input_usd_per_million = CASE WHEN off_peak_enabled THEN peak_input_usd_per_million ELSE input_usd_per_million END,
    peak_input_usd_per_million = CASE WHEN off_peak_enabled THEN input_usd_per_million ELSE peak_input_usd_per_million END,
    output_usd_per_million = CASE WHEN off_peak_enabled THEN peak_output_usd_per_million ELSE output_usd_per_million END,
    peak_output_usd_per_million = CASE WHEN off_peak_enabled THEN output_usd_per_million ELSE peak_output_usd_per_million END,
    cache_read_usd_per_million = CASE WHEN off_peak_enabled THEN peak_cache_read_usd_per_million ELSE cache_read_usd_per_million END,
    peak_cache_read_usd_per_million = CASE WHEN off_peak_enabled THEN cache_read_usd_per_million ELSE peak_cache_read_usd_per_million END,
    cache_creation_usd_per_million = CASE WHEN off_peak_enabled THEN peak_cache_creation_usd_per_million ELSE cache_creation_usd_per_million END,
    peak_cache_creation_usd_per_million = CASE WHEN off_peak_enabled THEN cache_creation_usd_per_million ELSE peak_cache_creation_usd_per_million END,
    request_usd = CASE WHEN off_peak_enabled THEN peak_request_usd ELSE request_usd END,
    peak_request_usd = CASE WHEN off_peak_enabled THEN request_usd ELSE peak_request_usd END,
    long_context_input_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_peak_input_usd_per_million ELSE long_context_input_usd_per_million END,
    long_context_peak_input_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_input_usd_per_million ELSE long_context_peak_input_usd_per_million END,
    long_context_output_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_peak_output_usd_per_million ELSE long_context_output_usd_per_million END,
    long_context_peak_output_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_output_usd_per_million ELSE long_context_peak_output_usd_per_million END,
    long_context_cache_read_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_peak_cache_read_usd_per_million ELSE long_context_cache_read_usd_per_million END,
    long_context_peak_cache_read_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_cache_read_usd_per_million ELSE long_context_peak_cache_read_usd_per_million END,
    long_context_cache_creation_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_peak_cache_creation_usd_per_million ELSE long_context_cache_creation_usd_per_million END,
    long_context_peak_cache_creation_usd_per_million = CASE WHEN off_peak_enabled THEN long_context_cache_creation_usd_per_million ELSE long_context_peak_cache_creation_usd_per_million END;
ALTER TABLE model_prices RENAME COLUMN peak_input_usd_per_million TO off_peak_input_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN peak_output_usd_per_million TO off_peak_output_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN peak_cache_read_usd_per_million TO off_peak_cache_read_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN peak_cache_creation_usd_per_million TO off_peak_cache_creation_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN peak_request_usd TO off_peak_request_usd;
ALTER TABLE model_prices RENAME COLUMN long_context_peak_input_usd_per_million TO long_context_off_peak_input_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN long_context_peak_output_usd_per_million TO long_context_off_peak_output_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN long_context_peak_cache_read_usd_per_million TO long_context_off_peak_cache_read_usd_per_million;
ALTER TABLE model_prices RENAME COLUMN long_context_peak_cache_creation_usd_per_million TO long_context_off_peak_cache_creation_usd_per_million;
