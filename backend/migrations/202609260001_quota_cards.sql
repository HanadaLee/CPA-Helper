-- +goose Up
CREATE TABLE quota_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id),
    kind TEXT NOT NULL CHECK (kind IN ('credit', 'reset')),
    name TEXT NOT NULL DEFAULT '',
    amount_usd REAL NOT NULL DEFAULT 0 CHECK (amount_usd >= 0),
    used_usd REAL NOT NULL DEFAULT 0 CHECK (used_usd >= 0 AND used_usd <= amount_usd),
    expires_at DATETIME,
    activated_at DATETIME,
    used_at DATETIME,
    revoked_at DATETIME,
    issued_by INTEGER REFERENCES users(id),
    revoked_by INTEGER REFERENCES users(id),
    batch_id TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX ix_quota_cards_user ON quota_cards(user_id, kind, revoked_at, expires_at);
CREATE INDEX ix_quota_cards_batch ON quota_cards(batch_id);

ALTER TABLE user_quota_charges ADD COLUMN cards_deducted_usd REAL NOT NULL DEFAULT 0;
ALTER TABLE user_quota_charges ADD COLUMN uncovered_usd REAL NOT NULL DEFAULT 0;

CREATE TABLE quota_card_deductions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL REFERENCES quota_cards(id),
    charge_id INTEGER NOT NULL REFERENCES user_quota_charges(id),
    amount_usd REAL NOT NULL CHECK (amount_usd > 0),
    created_at DATETIME NOT NULL,
    UNIQUE(card_id, charge_id)
);
CREATE INDEX ix_quota_card_deductions_charge ON quota_card_deductions(charge_id);

CREATE TABLE quota_resets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id),
    actor_id INTEGER NOT NULL REFERENCES users(id),
    card_id INTEGER UNIQUE REFERENCES quota_cards(id),
    daily_used_usd REAL NOT NULL,
    weekly_used_usd REAL NOT NULL,
    quota_day TEXT NOT NULL,
    quota_week TEXT NOT NULL,
    created_at DATETIME NOT NULL
);
CREATE INDEX ix_quota_resets_user ON quota_resets(user_id, id);

-- Preserve the remaining balances at migration time as separately auditable cards.
INSERT INTO quota_cards(user_id, kind, name, amount_usd, expires_at, activated_at, batch_id, created_at, updated_at)
SELECT id, 'credit', '月额度迁移',
       ROUND(MAX(0, quota_monthly_usd - CASE WHEN quota_month = strftime('%Y-%m', 'now', '+8 hours') THEN quota_month_used_usd ELSE 0 END), 8),
       strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+8 hours', '+1 month', 'floor', '-8 hours'),
       strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), 'migration-monthly',
       strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM users WHERE quota_monthly_usd > 0
  AND quota_monthly_usd > CASE WHEN quota_month = strftime('%Y-%m', 'now', '+8 hours') THEN quota_month_used_usd ELSE 0 END;

INSERT INTO quota_cards(user_id, kind, name, amount_usd, activated_at, batch_id, created_at, updated_at)
SELECT id, 'credit', '不限时额度迁移', ROUND(quota_lifetime_usd, 8),
       strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), 'migration-lifetime',
       strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM users WHERE quota_lifetime_usd > 0;

UPDATE users SET quota_daily_usd = COALESCE(quota_daily_usd, 0), quota_weekly_usd = COALESCE(quota_weekly_usd, 0),
    quota_monthly_usd = 0, quota_lifetime_usd = 0
WHERE quota_monthly_usd IS NOT NULL OR quota_lifetime_usd IS NOT NULL OR quota_daily_usd IS NOT NULL OR quota_weekly_usd IS NOT NULL;
UPDATE app_settings SET new_user_quota_monthly_usd = 0, new_user_quota_lifetime_usd = 0;

-- +goose Down
-- Balance conversion is irreversible after packs have been spent or edited.
-- Fail inside the migration transaction; downgrade by restoring a pre-upgrade backup.
CREATE TEMP TABLE quota_card_downgrade_guard (
    restore_pre_upgrade_backup INTEGER CONSTRAINT restore_pre_upgrade_backup_required CHECK (restore_pre_upgrade_backup = 1)
);
INSERT INTO quota_card_downgrade_guard VALUES (0);
