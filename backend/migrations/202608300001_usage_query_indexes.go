package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upUsageQueryIndexes, nil)
}

func upUsageQueryIndexes(ctx context.Context, db *sql.DB) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	columns, err := tableColumns(ctx, tx, "usage_records")
	if err != nil {
		return err
	}
	if !columns["source_key"] {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE usage_records ADD COLUMN source_key VARCHAR(64)`); err != nil {
			return err
		}
	}
	if err := backfillUsageSourceKeys(ctx, tx); err != nil {
		return err
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS ix_usage_records_source_key_timestamp ON usage_records(source_key, timestamp)`,
		`CREATE INDEX IF NOT EXISTS ix_usage_records_usage_username_timestamp ON usage_records(usage_username, timestamp)`,
		`CREATE INDEX IF NOT EXISTS ix_usage_records_failed_timestamp ON usage_records(failed, timestamp)`,
		`CREATE INDEX IF NOT EXISTS ix_usage_records_api_key_description_timestamp ON usage_records(api_key_description, timestamp)`,
		`CREATE INDEX IF NOT EXISTS ix_usage_records_provider_timestamp ON usage_records(provider, timestamp)`,
		`CREATE INDEX IF NOT EXISTS ix_usage_records_model_timestamp ON usage_records(model, timestamp)`,
		`CREATE INDEX IF NOT EXISTS ix_usage_records_endpoint_timestamp ON usage_records(endpoint, timestamp)`,
	}
	for _, statement := range indexes {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func backfillUsageSourceKeys(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, source
		FROM usage_records
		WHERE source IS NOT NULL AND trim(source) <> ''
	`)
	if err != nil {
		return err
	}
	type sourceKeyUpdate struct {
		id  int64
		key string
	}
	updates := []sourceKeyUpdate{}
	for rows.Next() {
		var id int64
		var source string
		if err := rows.Scan(&id, &source); err != nil {
			_ = rows.Close()
			return err
		}
		normalized := strings.TrimSpace(source)
		if normalized == "" {
			continue
		}
		sum := sha256.Sum256([]byte(normalized))
		updates = append(updates, sourceKeyUpdate{id: id, key: hex.EncodeToString(sum[:])})
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, update := range updates {
		if _, err := tx.ExecContext(ctx, `UPDATE usage_records SET source_key = ? WHERE id = ?`, update.key, update.id); err != nil {
			return err
		}
	}
	return nil
}
