package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestModelPriceFastMultiplierMigrationAddsDefaultsAndBackfillsServiceTier(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "fast-multiplier.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE model_prices (id INTEGER PRIMARY KEY);
		CREATE TABLE usage_records (id INTEGER PRIMARY KEY, raw_json TEXT NOT NULL);
		INSERT INTO model_prices (id) VALUES (1);
		INSERT INTO usage_records (id, raw_json) VALUES
			(1, '{"request_service_tier":"priority"}'),
			(2, '{"requestServiceTier":"standard"}'),
			(3, '{}');
	`); err != nil {
		t.Fatal(err)
	}

	if err := upModelPriceFastMultiplier(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	var multiplier float64
	if err := db.QueryRow(`SELECT fast_multiplier FROM model_prices WHERE id = 1`).Scan(&multiplier); err != nil {
		t.Fatal(err)
	}
	if multiplier != 1 {
		t.Fatalf("fast_multiplier = %v, want 1", multiplier)
	}
	for id, want := range map[int]string{1: "priority", 2: "standard"} {
		var serviceTier string
		if err := db.QueryRow(`SELECT request_service_tier FROM usage_records WHERE id = ?`, id).Scan(&serviceTier); err != nil {
			t.Fatal(err)
		}
		if serviceTier != want {
			t.Fatalf("row %d request_service_tier = %q, want %q", id, serviceTier, want)
		}
	}
	var empty sql.NullString
	if err := db.QueryRow(`SELECT request_service_tier FROM usage_records WHERE id = 3`).Scan(&empty); err != nil {
		t.Fatal(err)
	}
	if empty.Valid {
		t.Fatalf("empty request_service_tier = %q, want NULL", empty.String)
	}
}
