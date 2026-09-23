package app

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type pricingCalendar struct {
	years map[int]map[string]bool
}

// Peak hours are Beijing weekdays 09:00–12:00 and 14:00–18:00, excluding
// the published Chinese holiday calendar. A year without a calendar uses the
// off-peak rate so an unknown holiday cannot be charged at the peak rate.
func (calendar *pricingCalendar) isPeak(timestamp time.Time) bool {
	if calendar == nil || timestamp.IsZero() {
		return false
	}
	local := timestamp.In(appTimeLocation)
	holidayDates, configured := calendar.years[local.Year()]
	if !configured || holidayDates[local.Format("2006-01-02")] {
		return false
	}
	if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday {
		return false
	}
	hour := local.Hour()
	return (hour >= 9 && hour < 12) || (hour >= 14 && hour < 18)
}

func (a *App) loadPricingCalendar(ctx context.Context) (*pricingCalendar, error) {
	calendar := &pricingCalendar{years: map[int]map[string]bool{}}
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT year FROM pricing_holiday_years`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			rows.Close()
			return nil, err
		}
		calendar.years[year] = map[string]bool{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	rows, err = tx.QueryContext(ctx, `SELECT year, date FROM pricing_holidays`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var year int
		var date string
		if err := rows.Scan(&year, &date); err != nil {
			rows.Close()
			return nil, err
		}
		if dates, ok := calendar.years[year]; ok {
			dates[date] = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return calendar, nil
}

type pricingHolidayCalendarResponse struct {
	Year       int      `json:"year"`
	Configured bool     `json:"configured"`
	Dates      []string `json:"dates"`
	SourceURL  string   `json:"source_url"`
}

type pricingHolidayCalendarPayload struct {
	Year  int      `json:"year"`
	Dates []string `json:"dates"`
}

func pricingCalendarYear(year int) error {
	if year < 2000 || year > 2100 {
		return validationError("节假日年份必须在 2000 至 2100 之间")
	}
	return nil
}

func (a *App) pricingHolidayCalendarForYear(ctx context.Context, year int) (pricingHolidayCalendarResponse, error) {
	result := pricingHolidayCalendarResponse{Year: year, Dates: []string{}}
	var sourceURL string
	err := a.db.QueryRowContext(ctx, `SELECT source_url FROM pricing_holiday_years WHERE year = ?`, year).Scan(&sourceURL)
	if err == sql.ErrNoRows {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.Configured = true
	result.SourceURL = sourceURL
	rows, err := a.db.QueryContext(ctx, `SELECT date FROM pricing_holidays WHERE year = ? ORDER BY date`, year)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			return result, err
		}
		result.Dates = append(result.Dates, date)
	}
	return result, rows.Err()
}

func (a *App) handlePricingHolidayCalendar(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case http.MethodGet:
		year := time.Now().In(appTimeLocation).Year()
		if value := strings.TrimSpace(r.URL.Query().Get("year")); value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return validationError("节假日年份格式不正确")
			}
			year = parsed
		}
		if err := pricingCalendarYear(year); err != nil {
			return err
		}
		result, err := a.pricingHolidayCalendarForYear(r.Context(), year)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, result)
		return nil
	case http.MethodPut:
		var payload pricingHolidayCalendarPayload
		if err := decodeJSON(r, &payload); err != nil {
			return err
		}
		if err := pricingCalendarYear(payload.Year); err != nil {
			return err
		}
		dates := make([]string, 0, len(payload.Dates))
		seen := map[string]bool{}
		for _, raw := range payload.Dates {
			date := strings.TrimSpace(raw)
			parsed, err := time.Parse("2006-01-02", date)
			if err != nil || parsed.Format("2006-01-02") != date || parsed.Year() != payload.Year {
				return validationError("节假日日期必须为所选年份的 YYYY-MM-DD")
			}
			if !seen[date] {
				seen[date] = true
				dates = append(dates, date)
			}
		}
		if len(dates) > 366 {
			return validationError("节假日日期数量过多")
		}
		sort.Strings(dates)
		tx, err := a.db.BeginTx(r.Context(), nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(r.Context(), `
			INSERT INTO pricing_holiday_years (year, source_url, updated_at) VALUES (?, '', ?)
			ON CONFLICT(year) DO UPDATE SET source_url = '', updated_at = excluded.updated_at
		`, payload.Year, dbTime(time.Now())); err != nil {
			return err
		}
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM pricing_holidays WHERE year = ?`, payload.Year); err != nil {
			return err
		}
		for _, date := range dates {
			if _, err := tx.ExecContext(r.Context(), `INSERT INTO pricing_holidays (date, year) VALUES (?, ?)`, date, payload.Year); err != nil {
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		a.invalidateUsagePrices()
		result, err := a.pricingHolidayCalendarForYear(r.Context(), payload.Year)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, result)
		return nil
	default:
		return methodNotAllowed()
	}
}
