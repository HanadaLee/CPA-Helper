-- +goose Up
ALTER TABLE model_prices
ADD COLUMN fast_enabled BOOLEAN NOT NULL DEFAULT 0 CHECK (fast_enabled IN (0, 1));

-- Preserve existing explicit FAST multipliers when upgrading.
UPDATE model_prices SET fast_enabled = 1 WHERE fast_multiplier <> 1;

-- +goose Down
ALTER TABLE model_prices DROP COLUMN fast_enabled;
