package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// The production service refreshes the published calendar without putting
// GitHub on the request/ingest path. A failed refresh retains the last good
// snapshot. The admin sync endpoint can force an immediate refresh.
func (a *App) startPricingCalendarSync() {
	ctx, cancel := context.WithCancel(context.Background())
	a.pricingCalendarSyncCancel = cancel
	a.pricingCalendarSyncDone = make(chan struct{})
	go func() {
		defer close(a.pricingCalendarSyncDone)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			a.refreshPricingCalendar(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (a *App) refreshPricingCalendar(ctx context.Context) {
	now := time.Now().In(appTimeLocation)
	years := []int{now.Year()}
	if now.Month() >= time.November {
		years = append(years, now.Year()+1)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	for _, year := range years {
		if ctx.Err() != nil {
			return
		}
		cycleCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		_, err := a.syncPricingCalendar(cycleCtx, client, year, fmt.Sprintf(holidayCNURLPattern, year))
		cancel()
		if err != nil {
			log.Printf("pricing calendar refresh skipped for %d: %v", year, err)
		}
	}
}
