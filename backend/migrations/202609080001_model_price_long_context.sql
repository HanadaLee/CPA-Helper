-- +goose Up
ALTER TABLE model_prices
ADD COLUMN long_context_enabled BOOLEAN NOT NULL DEFAULT 0 CHECK (long_context_enabled IN (0, 1));

ALTER TABLE model_prices
ADD COLUMN long_context_threshold_tokens INTEGER NOT NULL DEFAULT 0 CHECK (long_context_threshold_tokens >= 0);

ALTER TABLE model_prices
ADD COLUMN long_context_input_usd_per_million REAL NOT NULL DEFAULT 0 CHECK (long_context_input_usd_per_million >= 0);

ALTER TABLE model_prices
ADD COLUMN long_context_output_usd_per_million REAL NOT NULL DEFAULT 0 CHECK (long_context_output_usd_per_million >= 0);

ALTER TABLE model_prices
ADD COLUMN long_context_cache_read_usd_per_million REAL NOT NULL DEFAULT 0 CHECK (long_context_cache_read_usd_per_million >= 0);

ALTER TABLE model_prices
ADD COLUMN long_context_cache_creation_usd_per_million REAL NOT NULL DEFAULT 0 CHECK (long_context_cache_creation_usd_per_million >= 0);

-- +goose Down
ALTER TABLE model_prices DROP COLUMN long_context_cache_creation_usd_per_million;
ALTER TABLE model_prices DROP COLUMN long_context_cache_read_usd_per_million;
ALTER TABLE model_prices DROP COLUMN long_context_output_usd_per_million;
ALTER TABLE model_prices DROP COLUMN long_context_input_usd_per_million;
ALTER TABLE model_prices DROP COLUMN long_context_threshold_tokens;
ALTER TABLE model_prices DROP COLUMN long_context_enabled;
