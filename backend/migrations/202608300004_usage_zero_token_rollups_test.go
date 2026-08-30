package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestUsageZeroTokenRollupsMigrationBackfillsRetainedDetails(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "zero-token-rollups.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE usage_records (
			id INTEGER PRIMARY KEY, timestamp TEXT NOT NULL, usage_username TEXT,
			api_key_description TEXT, provider TEXT, model TEXT, source_key TEXT,
			source TEXT, auth TEXT, endpoint TEXT, failed BOOLEAN NOT NULL DEFAULT 0,
			input_tokens INTEGER NOT NULL DEFAULT 0, output_tokens INTEGER NOT NULL DEFAULT 0,
			cache_read_tokens INTEGER NOT NULL DEFAULT 0, cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
			reasoning_tokens INTEGER NOT NULL DEFAULT 0, total_tokens INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE usage_hourly_rollups (
			id INTEGER PRIMARY KEY, bucket_start TEXT NOT NULL, usage_username TEXT NOT NULL DEFAULT '',
			api_key_description TEXT NOT NULL DEFAULT '', provider TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '', source_key TEXT NOT NULL DEFAULT '', source TEXT NOT NULL DEFAULT '',
			auth TEXT NOT NULL DEFAULT '', endpoint TEXT NOT NULL DEFAULT '', failed BOOLEAN NOT NULL DEFAULT 0
		);
		INSERT INTO usage_hourly_rollups (
			id, bucket_start, usage_username, api_key_description, provider, model,
			source_key, source, auth, endpoint, failed
		) VALUES (1, '2026-08-30T10:00:00.000000+08:00', 'admin', 'VSCode', 'openai', 'gpt-test', '', '', 'bearer', '/v1/responses', 0);
		INSERT INTO usage_records (
			id, timestamp, usage_username, api_key_description, provider, model,
			source_key, source, auth, endpoint, failed, total_tokens
		) VALUES
			(1, '2026-08-30T10:15:00.000000+08:00', 'admin', 'VSCode', 'openai', 'gpt-test', NULL, NULL, 'bearer', '/v1/responses', 0, 0),
			(2, '2026-08-30T10:20:00.000000+08:00', 'admin', 'VSCode', 'openai', 'gpt-test', NULL, NULL, 'bearer', '/v1/responses', 0, 100);
	`); err != nil {
		t.Fatal(err)
	}
	if err := upUsageZeroTokenRollups(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT zero_token_records FROM usage_hourly_rollups WHERE id = 1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("zero_token_records = %d, want 1", count)
	}
}
