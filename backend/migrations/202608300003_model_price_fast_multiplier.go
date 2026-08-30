package migrations

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upModelPriceFastMultiplier, nil)
}

func upModelPriceFastMultiplier(ctx context.Context, db *sql.DB) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	priceColumns, err := tableColumns(ctx, tx, "model_prices")
	if err != nil {
		return err
	}
	if !priceColumns["fast_multiplier"] {
		if _, err = tx.ExecContext(ctx, `
			ALTER TABLE model_prices
			ADD COLUMN fast_multiplier REAL NOT NULL DEFAULT 1 CHECK (fast_multiplier > 0)
		`); err != nil {
			return err
		}
	}

	usageColumns, err := tableColumns(ctx, tx, "usage_records")
	if err != nil {
		return err
	}
	if !usageColumns["request_service_tier"] {
		if _, err = tx.ExecContext(ctx, `ALTER TABLE usage_records ADD COLUMN request_service_tier VARCHAR(64)`); err != nil {
			return err
		}
	}
	if err = backfillUsageRequestServiceTier(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func backfillUsageRequestServiceTier(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, raw_json
		FROM usage_records
		WHERE request_service_tier IS NULL OR request_service_tier = ''
	`)
	if err != nil {
		return err
	}
	type update struct {
		id          int64
		serviceTier string
	}
	updates := []update{}
	for rows.Next() {
		var id int64
		var rawJSON string
		if err := rows.Scan(&id, &rawJSON); err != nil {
			_ = rows.Close()
			return err
		}
		var parsed any
		if json.Unmarshal([]byte(rawJSON), &parsed) != nil {
			continue
		}
		serviceTier := migrationString(migrationFindFirst(parsed, "request_service_tier", "requestServiceTier"))
		if serviceTier != nil {
			updates = append(updates, update{id: id, serviceTier: *serviceTier})
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, item := range updates {
		if _, err := tx.ExecContext(ctx, `UPDATE usage_records SET request_service_tier = ? WHERE id = ?`, item.serviceTier, item.id); err != nil {
			return err
		}
	}
	return nil
}
