package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestBackfillUsageSourceKeysUsesTrimmedSource(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "usage-source-key.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE usage_records (id INTEGER PRIMARY KEY, source TEXT, source_key TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO usage_records (id, source) VALUES (1, '  vscode-source  '), (2, NULL), (3, '')`); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := backfillUsageSourceKeys(context.Background(), tx); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	var key string
	if err := db.QueryRow(`SELECT source_key FROM usage_records WHERE id = 1`).Scan(&key); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("vscode-source"))
	if want := hex.EncodeToString(sum[:]); key != want {
		t.Fatalf("source_key = %q, want %q", key, want)
	}
	var emptyCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM usage_records WHERE id IN (2, 3) AND source_key IS NULL`).Scan(&emptyCount); err != nil {
		t.Fatal(err)
	}
	if emptyCount != 2 {
		t.Fatalf("empty source rows with NULL source_key = %d, want 2", emptyCount)
	}
}
