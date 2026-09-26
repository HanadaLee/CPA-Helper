package app

import (
	"context"
	"log"
	"time"
)

func (a *App) startQuotaMaintenance(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	a.quotaMaintenanceCancel = cancel
	a.quotaMaintenanceDone = make(chan struct{})
	go func() {
		defer close(a.quotaMaintenanceDone)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			cycle, stop := context.WithTimeout(ctx, 30*time.Second)
			if err := a.reconcileQuotaUsers(cycle); err != nil && ctx.Err() == nil {
				log.Printf("quota maintenance: %v", err)
			}
			stop()
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (a *App) reconcileQuotaUsers(ctx context.Context) error {
	rows, err := a.db.QueryContext(ctx, `SELECT id FROM users WHERE disabled_at IS NULL AND
		(quota_daily_usd IS NOT NULL OR quota_weekly_usd IS NOT NULL OR quota_paused_at IS NOT NULL OR quota_sync_error IS NOT NULL)`)
	if err != nil {
		return err
	}
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := a.userQuotaStatus(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
