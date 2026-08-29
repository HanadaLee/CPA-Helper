package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestUsageRetentionMigrationPreservesChargedCostAndAuditRows(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "usage-retention.sqlite3")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, statement := range []string{
		`CREATE TABLE app_settings (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE users (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE model_prices (
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			input_usd_per_million REAL NOT NULL,
			output_usd_per_million REAL NOT NULL,
			cache_read_usd_per_million REAL NOT NULL,
			cache_creation_usd_per_million REAL NOT NULL,
			request_usd REAL
		)`,
		`CREATE TABLE usage_records (
			id INTEGER PRIMARY KEY,
			created_at DATETIME NOT NULL,
			timestamp DATETIME NOT NULL,
			provider TEXT,
			model TEXT,
			failed BOOLEAN NOT NULL DEFAULT 0,
			input_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			cached_tokens INTEGER NOT NULL DEFAULT 0,
			cache_read_tokens INTEGER NOT NULL DEFAULT 0,
			cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
			reasoning_tokens INTEGER NOT NULL DEFAULT 0,
			total_tokens INTEGER NOT NULL DEFAULT 0,
			dedupe_key TEXT NOT NULL UNIQUE
		)`,
		`CREATE TABLE user_quota_charges (
			id INTEGER PRIMARY KEY,
			usage_record_id INTEGER NOT NULL UNIQUE,
			user_id INTEGER NOT NULL,
			usage_username TEXT NOT NULL,
			amount_usd REAL NOT NULL DEFAULT 0,
			daily_deducted_usd REAL NOT NULL DEFAULT 0,
			weekly_deducted_usd REAL NOT NULL DEFAULT 0,
			monthly_deducted_usd REAL NOT NULL DEFAULT 0,
			lifetime_deducted_usd REAL NOT NULL DEFAULT 0,
			unpriced BOOLEAN NOT NULL DEFAULT 0,
			quota_day TEXT NOT NULL DEFAULT '',
			quota_week TEXT NOT NULL DEFAULT '',
			quota_month TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY(usage_record_id) REFERENCES usage_records(id),
			FOREIGN KEY(user_id) REFERENCES users(id)
		)`,
		`INSERT INTO app_settings (id) VALUES (1)`,
		`INSERT INTO users (id) VALUES (1)`,
		`INSERT INTO model_prices (
			provider, model, input_usd_per_million, output_usd_per_million,
			cache_read_usd_per_million, cache_creation_usd_per_million
		) VALUES ('openai', 'gpt-migrated', 9, 0, 0, 0)`,
		`INSERT INTO usage_records (
			id, created_at, timestamp, provider, model, input_tokens, total_tokens, dedupe_key
		) VALUES (1, '2026-08-01 00:00:00', '2026-08-01 00:00:00', 'openai', 'gpt-migrated', 1000000, 1000000, 'migration-dedupe')`,
		`INSERT INTO user_quota_charges (
			id, usage_record_id, user_id, usage_username, amount_usd,
			daily_deducted_usd, quota_day, quota_week, quota_month, created_at
		) VALUES (1, 1, 1, 'migration-user', 1.25, 1.25, '2026-08-01', '2026-31', '2026-08', '2026-08-01 00:00:01')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("prepare migration fixture: %v\n%s", err, statement)
		}
	}

	if err := upUsageRetentionRollups(context.Background(), db); err != nil {
		t.Fatalf("upUsageRetentionRollups failed: %v", err)
	}

	var cost float64
	var unpriced bool
	if err := db.QueryRow(`SELECT cost_usd, unpriced FROM usage_records WHERE id = 1`).Scan(&cost, &unpriced); err != nil {
		t.Fatal(err)
	}
	if cost != 1.25 || unpriced {
		t.Fatalf("stored migrated cost = %v, unpriced %v; want charged 1.25/false", cost, unpriced)
	}
	var dedupRecordID, chargeRecordID sql.NullInt64
	var dedupCount, chargeCount int
	if err := db.QueryRow(`SELECT COUNT(*), usage_record_id FROM usage_ingest_dedup WHERE dedupe_key = 'migration-dedupe'`).Scan(&dedupCount, &dedupRecordID); err != nil {
		t.Fatal(err)
	}
	if dedupCount != 1 || !dedupRecordID.Valid || dedupRecordID.Int64 != 1 {
		t.Fatalf("migrated dedup row = count %d, record %#v", dedupCount, dedupRecordID)
	}

	if _, err := db.Exec(`DELETE FROM usage_records WHERE id = 1`); err != nil {
		t.Fatalf("delete migrated usage detail: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*), usage_record_id FROM usage_ingest_dedup WHERE dedupe_key = 'migration-dedupe'`).Scan(&dedupCount, &dedupRecordID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*), usage_record_id FROM user_quota_charges WHERE usage_dedupe_key = 'migration-dedupe'`).Scan(&chargeCount, &chargeRecordID); err != nil {
		t.Fatal(err)
	}
	if dedupCount != 1 || dedupRecordID.Valid || chargeCount != 1 || chargeRecordID.Valid {
		t.Fatalf("retained rows after detail deletion = dedup %d/%#v, charge %d/%#v", dedupCount, dedupRecordID, chargeCount, chargeRecordID)
	}
}
