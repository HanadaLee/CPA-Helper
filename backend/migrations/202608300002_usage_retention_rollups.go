package migrations

import (
	"context"
	"database/sql"
	"math"
	"strings"

	"github.com/pressly/goose/v3"
)

type storedUsagePrice struct {
	inputPerMillion         float64
	outputPerMillion        float64
	cacheReadPerMillion     float64
	cacheCreationPerMillion float64
	request                 *float64
}

type storedUsageCostRecord struct {
	id                  int64
	provider            sql.NullString
	model               sql.NullString
	failed              bool
	inputTokens         int
	outputTokens        int
	cachedTokens        int
	cacheReadTokens     int
	cacheCreationTokens int
	reasoningTokens     int
	totalTokens         int
}

const storedUsageCostBackfillBatchSize = 2000

func init() {
	goose.AddMigrationNoTxContext(upUsageRetentionRollups, nil)
}

func upUsageRetentionRollups(ctx context.Context, db *sql.DB) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	settingsColumns, err := tableColumns(ctx, tx, "app_settings")
	if err != nil {
		return err
	}
	if !settingsColumns["usage_detail_retention_days"] {
		if _, err = tx.ExecContext(ctx, `ALTER TABLE app_settings ADD COLUMN usage_detail_retention_days INTEGER NOT NULL DEFAULT 90`); err != nil {
			return err
		}
	}

	usageColumns, err := tableColumns(ctx, tx, "usage_records")
	if err != nil {
		return err
	}
	if !usageColumns["cost_usd"] {
		if _, err = tx.ExecContext(ctx, `ALTER TABLE usage_records ADD COLUMN cost_usd REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if !usageColumns["unpriced"] {
		if _, err = tx.ExecContext(ctx, `ALTER TABLE usage_records ADD COLUMN unpriced BOOLEAN NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if err = backfillStoredUsageCosts(ctx, tx); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE usage_records
		SET cost_usd = (
				SELECT charges.amount_usd FROM user_quota_charges AS charges
				WHERE charges.usage_record_id = usage_records.id
			),
			unpriced = (
				SELECT charges.unpriced FROM user_quota_charges AS charges
				WHERE charges.usage_record_id = usage_records.id
			)
		WHERE EXISTS (
			SELECT 1 FROM user_quota_charges AS charges
			WHERE charges.usage_record_id = usage_records.id
		)
	`); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS usage_ingest_dedup (
			dedupe_key VARCHAR(80) PRIMARY KEY,
			usage_timestamp DATETIME NOT NULL,
			first_seen_at DATETIME NOT NULL,
			usage_record_id INTEGER UNIQUE,
			FOREIGN KEY(usage_record_id) REFERENCES usage_records(id) ON DELETE SET NULL
		)
	`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO usage_ingest_dedup (dedupe_key, usage_timestamp, first_seen_at, usage_record_id)
		SELECT dedupe_key, timestamp, created_at, id FROM usage_records
	`); err != nil {
		return err
	}

	if err = rebuildQuotaChargesForRetention(ctx, tx); err != nil {
		return err
	}

	statements := []string{
		`CREATE INDEX IF NOT EXISTS ix_usage_ingest_dedup_usage_timestamp ON usage_ingest_dedup(usage_timestamp)`,
		`CREATE TABLE IF NOT EXISTS usage_hourly_rollups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bucket_start DATETIME NOT NULL,
			usage_username VARCHAR(120) NOT NULL DEFAULT '',
			api_key_description VARCHAR(240) NOT NULL DEFAULT '',
			provider VARCHAR(120) NOT NULL DEFAULT '',
			model VARCHAR(180) NOT NULL DEFAULT '',
			source_key VARCHAR(64) NOT NULL DEFAULT '',
			source VARCHAR(240) NOT NULL DEFAULT '',
			auth VARCHAR(80) NOT NULL DEFAULT '',
			endpoint VARCHAR(240) NOT NULL DEFAULT '',
			failed BOOLEAN NOT NULL DEFAULT 0,
			record_count INTEGER NOT NULL DEFAULT 0,
			failed_count INTEGER NOT NULL DEFAULT 0,
			input_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			cached_tokens INTEGER NOT NULL DEFAULT 0,
			cache_read_tokens INTEGER NOT NULL DEFAULT 0,
			cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
			cache_hit_tokens INTEGER NOT NULL DEFAULT 0,
			reasoning_tokens INTEGER NOT NULL DEFAULT 0,
			total_tokens INTEGER NOT NULL DEFAULT 0,
			cost_usd REAL NOT NULL DEFAULT 0,
			unpriced_records INTEGER NOT NULL DEFAULT 0,
			ttft_ms_sum REAL NOT NULL DEFAULT 0,
			ttft_sample_count INTEGER NOT NULL DEFAULT 0,
			first_timestamp DATETIME NOT NULL,
			last_timestamp DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			CONSTRAINT uq_usage_hourly_rollups_grain UNIQUE (
				bucket_start, usage_username, api_key_description, provider,
				model, source_key, source, auth, endpoint, failed
			)
		)`,
		`CREATE INDEX IF NOT EXISTS ix_usage_hourly_rollups_bucket ON usage_hourly_rollups(bucket_start)`,
		`CREATE INDEX IF NOT EXISTS ix_usage_hourly_rollups_user_bucket ON usage_hourly_rollups(usage_username, bucket_start)`,
		`CREATE INDEX IF NOT EXISTS ix_usage_hourly_rollups_provider_model_bucket ON usage_hourly_rollups(provider, model, bucket_start)`,
		`CREATE INDEX IF NOT EXISTS ix_usage_hourly_rollups_description_bucket ON usage_hourly_rollups(api_key_description, bucket_start)`,
		`CREATE INDEX IF NOT EXISTS ix_usage_hourly_rollups_endpoint_bucket ON usage_hourly_rollups(endpoint, bucket_start)`,
		`CREATE INDEX IF NOT EXISTS ix_usage_hourly_rollups_source_bucket ON usage_hourly_rollups(source_key, bucket_start)`,
		`CREATE TABLE IF NOT EXISTS usage_rollup_state (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			last_rolled_usage_id INTEGER NOT NULL DEFAULT 0,
			phase VARCHAR(20) NOT NULL DEFAULT 'backfilling',
			last_success_at DATETIME,
			last_cleanup_at DATETIME,
			last_cleanup_count INTEGER NOT NULL DEFAULT 0,
			last_error TEXT,
			updated_at DATETIME NOT NULL
		)`,
		`INSERT OR IGNORE INTO usage_rollup_state (id, last_cleanup_at, updated_at) VALUES (1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
	}
	for _, statement := range statements {
		if _, err = tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func backfillStoredUsageCosts(ctx context.Context, tx *sql.Tx) error {
	prices, err := loadStoredUsagePrices(ctx, tx)
	if err != nil {
		return err
	}
	updateCost, err := tx.PrepareContext(ctx, `UPDATE usage_records SET cost_usd = ?, unpriced = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer updateCost.Close()
	lastID := int64(0)
	for {
		rows, err := tx.QueryContext(ctx, `
			SELECT id, provider, model, failed, input_tokens, output_tokens, cached_tokens,
			       cache_read_tokens, cache_creation_tokens, reasoning_tokens, total_tokens
			FROM usage_records
			WHERE id > ?
			ORDER BY id
			LIMIT ?
		`, lastID, storedUsageCostBackfillBatchSize)
		if err != nil {
			return err
		}
		records := make([]storedUsageCostRecord, 0, storedUsageCostBackfillBatchSize)
		for rows.Next() {
			var record storedUsageCostRecord
			if err := rows.Scan(&record.id, &record.provider, &record.model, &record.failed,
				&record.inputTokens, &record.outputTokens, &record.cachedTokens,
				&record.cacheReadTokens, &record.cacheCreationTokens, &record.reasoningTokens,
				&record.totalTokens); err != nil {
				_ = rows.Close()
				return err
			}
			records = append(records, record)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, record := range records {
			cost, unpriced := storedUsageRecordCost(record, prices)
			if _, err := updateCost.ExecContext(ctx, cost, unpriced, record.id); err != nil {
				return err
			}
			lastID = record.id
		}
		if len(records) < storedUsageCostBackfillBatchSize {
			return nil
		}
	}
}

func loadStoredUsagePrices(ctx context.Context, tx *sql.Tx) (map[[2]string]storedUsagePrice, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT provider, model, input_usd_per_million, output_usd_per_million,
		       cache_read_usd_per_million, cache_creation_usd_per_million, request_usd
		FROM model_prices
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	prices := map[[2]string]storedUsagePrice{}
	for rows.Next() {
		var provider, model string
		var price storedUsagePrice
		var request sql.NullFloat64
		if err := rows.Scan(&provider, &model, &price.inputPerMillion, &price.outputPerMillion,
			&price.cacheReadPerMillion, &price.cacheCreationPerMillion, &request); err != nil {
			return nil, err
		}
		if request.Valid {
			value := request.Float64
			price.request = &value
		}
		prices[[2]string{strings.ToLower(strings.TrimSpace(provider)), strings.ToLower(strings.TrimSpace(model))}] = price
	}
	return prices, rows.Err()
}

func storedUsageRecordCost(record storedUsageCostRecord, prices map[[2]string]storedUsagePrice) (float64, bool) {
	model := strings.ToLower(strings.TrimSpace(record.model.String))
	if strings.Contains(model, "image") {
		if record.failed {
			return 0, false
		}
		price, ok := matchingStoredUsagePrice(record.provider.String, model, prices)
		if !ok || price.request == nil {
			return 0, true
		}
		return roundStoredUsageCost(*price.request), false
	}
	price, ok := matchingStoredUsagePrice(record.provider.String, model, prices)
	if !ok {
		return 0, storedUsageTotalTokens(record) > 0
	}
	input := nonNegativeMigrationTokens(record.inputTokens)
	output := nonNegativeMigrationTokens(record.outputTokens)
	amount := 0.0
	if isStoredClaudeProvider(record.provider.String) {
		amount = migrationMillionTokenCost(input, price.inputPerMillion) +
			migrationMillionTokenCost(nonNegativeMigrationTokens(record.cacheReadTokens), price.cacheReadPerMillion) +
			migrationMillionTokenCost(nonNegativeMigrationTokens(record.cacheCreationTokens), price.cacheCreationPerMillion) +
			migrationMillionTokenCost(output, price.outputPerMillion)
	} else {
		cacheRead := nonNegativeMigrationTokens(record.cachedTokens)
		if cacheRead > input {
			cacheRead = input
		}
		amount = migrationMillionTokenCost(input-cacheRead, price.inputPerMillion) +
			migrationMillionTokenCost(cacheRead, price.cacheReadPerMillion) +
			migrationMillionTokenCost(output, price.outputPerMillion)
	}
	return roundStoredUsageCost(amount), false
}

func matchingStoredUsagePrice(provider, model string, prices map[[2]string]storedUsagePrice) (storedUsagePrice, bool) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	model = strings.ToLower(strings.TrimSpace(model))
	candidates := []string{provider}
	if provider == "codex" {
		candidates = append(candidates, "openai")
	} else if provider == "claude" {
		candidates = append(candidates, "anthropic")
	}
	for _, candidate := range candidates {
		if price, ok := prices[[2]string{candidate, model}]; ok {
			return price, true
		}
	}
	return storedUsagePrice{}, false
}

func storedUsageTotalTokens(record storedUsageCostRecord) int {
	if isStoredClaudeProvider(record.provider.String) {
		return nonNegativeMigrationTokens(record.inputTokens) +
			nonNegativeMigrationTokens(record.cacheReadTokens) +
			nonNegativeMigrationTokens(record.cacheCreationTokens) +
			nonNegativeMigrationTokens(record.outputTokens) +
			nonNegativeMigrationTokens(record.reasoningTokens)
	}
	return nonNegativeMigrationTokens(record.totalTokens)
}

func isStoredClaudeProvider(provider string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	return provider == "claude" || provider == "anthropic"
}

func migrationMillionTokenCost(tokens int, rate float64) float64 {
	return float64(tokens) * rate / 1_000_000
}

func nonNegativeMigrationTokens(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func roundStoredUsageCost(value float64) float64 {
	return math.Round(value*100_000_000) / 100_000_000
}

func rebuildQuotaChargesForRetention(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE user_quota_charges_retention_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			usage_record_id INTEGER UNIQUE,
			usage_dedupe_key VARCHAR(80) NOT NULL UNIQUE,
			usage_timestamp DATETIME NOT NULL,
			user_id INTEGER NOT NULL,
			usage_username VARCHAR(120) NOT NULL,
			amount_usd REAL NOT NULL DEFAULT 0,
			daily_deducted_usd REAL NOT NULL DEFAULT 0,
			weekly_deducted_usd REAL NOT NULL DEFAULT 0,
			monthly_deducted_usd REAL NOT NULL DEFAULT 0,
			lifetime_deducted_usd REAL NOT NULL DEFAULT 0,
			unpriced BOOLEAN NOT NULL DEFAULT 0,
			quota_day VARCHAR(10) NOT NULL DEFAULT '',
			quota_week VARCHAR(8) NOT NULL DEFAULT '',
			quota_month VARCHAR(7) NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY(usage_record_id) REFERENCES usage_records(id) ON DELETE SET NULL,
			FOREIGN KEY(user_id) REFERENCES users(id)
		)
	`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_quota_charges_retention_new (
			id, usage_record_id, usage_dedupe_key, usage_timestamp, user_id, usage_username,
			amount_usd, daily_deducted_usd, weekly_deducted_usd, monthly_deducted_usd,
			lifetime_deducted_usd, unpriced, quota_day, quota_week, quota_month, created_at
		)
		SELECT charges.id, charges.usage_record_id, records.dedupe_key, records.timestamp,
		       charges.user_id, charges.usage_username, charges.amount_usd,
		       charges.daily_deducted_usd, charges.weekly_deducted_usd,
		       charges.monthly_deducted_usd, charges.lifetime_deducted_usd,
		       charges.unpriced, charges.quota_day, charges.quota_week,
		       charges.quota_month, charges.created_at
		FROM user_quota_charges AS charges
		JOIN usage_records AS records ON records.id = charges.usage_record_id
	`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE user_quota_charges`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE user_quota_charges_retention_new RENAME TO user_quota_charges`); err != nil {
		return err
	}
	for _, statement := range []string{
		`CREATE INDEX IF NOT EXISTS ix_user_quota_charges_user_id ON user_quota_charges(user_id)`,
		`CREATE INDEX IF NOT EXISTS ix_user_quota_charges_created_at ON user_quota_charges(created_at)`,
		`CREATE INDEX IF NOT EXISTS ix_user_quota_charges_quota_day ON user_quota_charges(quota_day)`,
		`CREATE INDEX IF NOT EXISTS ix_user_quota_charges_quota_week ON user_quota_charges(quota_week)`,
		`CREATE INDEX IF NOT EXISTS ix_user_quota_charges_quota_month ON user_quota_charges(quota_month)`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
