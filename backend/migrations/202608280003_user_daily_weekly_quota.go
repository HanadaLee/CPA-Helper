package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upUserDailyWeeklyQuota, nil)
}

func upUserDailyWeeklyQuota(ctx context.Context, db *sql.DB) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	userColumns, err := tableColumns(ctx, tx, "users")
	if err != nil {
		return err
	}
	columns := []struct {
		name string
		sql  string
	}{
		{"quota_daily_usd", `ALTER TABLE users ADD COLUMN quota_daily_usd REAL`},
		{"quota_weekly_usd", `ALTER TABLE users ADD COLUMN quota_weekly_usd REAL`},
		{"quota_day", `ALTER TABLE users ADD COLUMN quota_day VARCHAR(10) NOT NULL DEFAULT ''`},
		{"quota_day_used_usd", `ALTER TABLE users ADD COLUMN quota_day_used_usd REAL NOT NULL DEFAULT 0`},
		{"quota_week", `ALTER TABLE users ADD COLUMN quota_week VARCHAR(8) NOT NULL DEFAULT ''`},
		{"quota_week_used_usd", `ALTER TABLE users ADD COLUMN quota_week_used_usd REAL NOT NULL DEFAULT 0`},
	}
	for _, column := range columns {
		if !userColumns[column.name] {
			if _, err := tx.ExecContext(ctx, column.sql); err != nil {
				return err
			}
		}
	}

	chargeColumns, err := tableColumns(ctx, tx, "user_quota_charges")
	if err != nil {
		return err
	}
	chargeColumnsToAdd := []struct {
		name string
		sql  string
	}{
		{"daily_deducted_usd", `ALTER TABLE user_quota_charges ADD COLUMN daily_deducted_usd REAL NOT NULL DEFAULT 0`},
		{"weekly_deducted_usd", `ALTER TABLE user_quota_charges ADD COLUMN weekly_deducted_usd REAL NOT NULL DEFAULT 0`},
		{"quota_day", `ALTER TABLE user_quota_charges ADD COLUMN quota_day VARCHAR(10) NOT NULL DEFAULT ''`},
		{"quota_week", `ALTER TABLE user_quota_charges ADD COLUMN quota_week VARCHAR(8) NOT NULL DEFAULT ''`},
	}
	for _, column := range chargeColumnsToAdd {
		if !chargeColumns[column.name] {
			if _, err := tx.ExecContext(ctx, column.sql); err != nil {
				return err
			}
		}
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS ix_user_quota_charges_quota_day ON user_quota_charges(quota_day)`,
		`CREATE INDEX IF NOT EXISTS ix_user_quota_charges_quota_week ON user_quota_charges(quota_week)`,
	}
	for _, statement := range indexes {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	return tx.Commit()
}
