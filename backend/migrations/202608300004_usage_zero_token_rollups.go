package migrations

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(upUsageZeroTokenRollups, nil)
}

func upUsageZeroTokenRollups(ctx context.Context, db *sql.DB) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	columns, err := tableColumns(ctx, tx, "usage_hourly_rollups")
	if err != nil {
		return err
	}
	if !columns["zero_token_records"] {
		if _, err = tx.ExecContext(ctx, `
			ALTER TABLE usage_hourly_rollups
			ADD COLUMN zero_token_records INTEGER NOT NULL DEFAULT 0
		`); err != nil {
			return err
		}
	}
	if err = backfillUsageZeroTokenRollups(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func backfillUsageZeroTokenRollups(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT CAST(timestamp AS TEXT), usage_username, api_key_description, provider, model,
		       source_key, source, auth, endpoint, failed
		FROM usage_records
		WHERE (
			lower(trim(COALESCE(provider, ''))) IN ('claude', 'anthropic')
			AND input_tokens <= 0 AND output_tokens <= 0 AND cache_read_tokens <= 0
			AND cache_creation_tokens <= 0 AND reasoning_tokens <= 0
		) OR (
			lower(trim(COALESCE(provider, ''))) NOT IN ('claude', 'anthropic')
			AND total_tokens <= 0
		)
	`)
	if err != nil {
		return err
	}
	type zeroTokenUsage struct {
		bucketStart       string
		usageUsername     string
		apiKeyDescription string
		provider          string
		model             string
		sourceKey         string
		source            string
		auth              string
		endpoint          string
		failed            bool
	}
	items := map[zeroTokenUsage]int{}
	for rows.Next() {
		var timestamp string
		var username, description, provider, model, sourceKey, source, auth, endpoint sql.NullString
		var failed bool
		if err := rows.Scan(&timestamp, &username, &description, &provider, &model, &sourceKey, &source, &auth, &endpoint, &failed); err != nil {
			_ = rows.Close()
			return err
		}
		normalizedTimestamp, ok := normalizeBeijingTimestampForMigration(timestamp)
		if !ok {
			continue
		}
		parsed, err := time.Parse("2006-01-02T15:04:05.999999-07:00", normalizedTimestamp)
		if err != nil {
			continue
		}
		local := parsed.In(beijingTimeLocationForMigration)
		bucket := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, beijingTimeLocationForMigration)
		item := zeroTokenUsage{
			bucketStart:       formatBeijingTimestampForMigration(bucket),
			usageUsername:     normalizedMigrationRollupDimension(username),
			apiKeyDescription: normalizedMigrationRollupDimension(description),
			provider:          normalizedMigrationRollupDimension(provider),
			model:             normalizedMigrationRollupDimension(model),
			sourceKey:         normalizedMigrationRollupDimension(sourceKey),
			source:            normalizedMigrationRollupDimension(source),
			auth:              normalizedMigrationRollupDimension(auth),
			endpoint:          normalizedMigrationRollupDimension(endpoint),
			failed:            failed,
		}
		items[item]++
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	update, err := tx.PrepareContext(ctx, `
		UPDATE usage_hourly_rollups
		SET zero_token_records = zero_token_records + ?
		WHERE unixepoch(bucket_start) = unixepoch(?) AND usage_username = ? AND api_key_description = ?
		  AND provider = ? AND model = ? AND source_key = ? AND source = ?
		  AND auth = ? AND endpoint = ? AND failed = ?
	`)
	if err != nil {
		return err
	}
	defer update.Close()
	for item, count := range items {
		if _, err := update.ExecContext(ctx, count, item.bucketStart, item.usageUsername, item.apiKeyDescription,
			item.provider, item.model, item.sourceKey, item.source, item.auth, item.endpoint, item.failed); err != nil {
			return err
		}
	}
	return nil
}

func normalizedMigrationRollupDimension(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}
