-- +goose Up
ALTER TABLE model_prices
ADD COLUMN long_context_fast_unsupported BOOLEAN NOT NULL DEFAULT 0 CHECK (long_context_fast_unsupported IN (0, 1));

-- +goose Down
ALTER TABLE model_prices DROP COLUMN long_context_fast_unsupported;
