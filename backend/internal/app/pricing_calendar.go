package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const holidayCNURLPattern = "https://raw.githubusercontent.com/NateScarlet/holiday-cn/master/%d.json"
const holidayCNMaxResponseBytes = 1 << 20

type pricingCalendar struct {
	years            map[int]map[string]bool // date -> isOffDay
	peakOnMakeupDays bool
}

// Peak hours are Beijing weekdays 09:00–12:00 and 14:00–18:00, excluding
// published off-days. Published makeup workdays are peak only when opted in.
// An unpublished year stays off-peak instead of risking an overcharge.
func (calendar *pricingCalendar) isPeak(timestamp time.Time) bool {
	if calendar == nil || timestamp.IsZero() {
		return false
	}
	local := timestamp.In(appTimeLocation)
	days, configured := calendar.years[local.Year()]
	if !configured {
		return false
	}
	isOffDay, published := days[local.Format("2006-01-02")]
	if published && isOffDay {
		return false
	}
	if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday {
		if !published || !calendar.peakOnMakeupDays {
			return false
		}
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
	if err := tx.QueryRowContext(ctx, `SELECT peak_on_makeup_days FROM pricing_calendar_settings WHERE id = 1`).Scan(&calendar.peakOnMakeupDays); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT year FROM pricing_holiday_years WHERE source_url <> ''`)
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
	rows, err = tx.QueryContext(ctx, `SELECT date, is_off_day FROM pricing_holidays ORDER BY year, date`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var date string
		var isOffDay bool
		if err := rows.Scan(&date, &isOffDay); err != nil {
			rows.Close()
			return nil, err
		}
		// A published notice can contain dates in an adjacent calendar year.
		if len(date) >= 4 {
			if year, err := strconv.Atoi(date[:4]); err == nil {
				if days, ok := calendar.years[year]; ok {
					days[date] = isOffDay
				}
			}
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

type pricingHolidayDay struct {
	Date     string `json:"date"`
	Name     string `json:"name"`
	IsOffDay bool   `json:"is_off_day"`
}

type pricingHolidayCalendarResponse struct {
	Year             int                 `json:"year"`
	Configured       bool                `json:"configured"`
	Dates            []string            `json:"dates"` // Off-days retained for older clients.
	Days             []pricingHolidayDay `json:"days"`
	SourceURL        string              `json:"source_url"`
	SyncedAt         *time.Time          `json:"synced_at"`
	PeakOnMakeupDays bool                `json:"peak_on_makeup_days"`
}

type pricingHolidayCalendarPayload struct {
	Year int `json:"year"`
}

type pricingCalendarSettingsPayload struct {
	PeakOnMakeupDays *bool `json:"peak_on_makeup_days"`
}

func pricingCalendarYear(year int) error {
	if year < 2000 || year > 2100 {
		return validationError("节假日年份必须在 2000 至 2100 之间")
	}
	return nil
}

func (a *App) pricingHolidayCalendarForYear(ctx context.Context, year int) (pricingHolidayCalendarResponse, error) {
	result := pricingHolidayCalendarResponse{Year: year, Dates: []string{}, Days: []pricingHolidayDay{}}
	if err := a.db.QueryRowContext(ctx, `SELECT peak_on_makeup_days FROM pricing_calendar_settings WHERE id = 1`).Scan(&result.PeakOnMakeupDays); err != nil {
		return result, err
	}
	var sourceURL string
	var syncedAt sql.NullString
	err := a.db.QueryRowContext(ctx, `SELECT source_url, CAST(synced_at AS TEXT) FROM pricing_holiday_years WHERE year = ?`, year).Scan(&sourceURL, &syncedAt)
	if err == sql.ErrNoRows {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	if sourceURL == "" {
		return result, nil
	}
	result.Configured = true
	result.SourceURL = sourceURL
	result.SyncedAt = timePtr(syncedAt)
	rows, err := a.db.QueryContext(ctx, `SELECT date, name, is_off_day FROM pricing_holidays WHERE year = ? ORDER BY date`, year)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var day pricingHolidayDay
		if err := rows.Scan(&day.Date, &day.Name, &day.IsOffDay); err != nil {
			return result, err
		}
		result.Days = append(result.Days, day)
		if day.IsOffDay {
			result.Dates = append(result.Dates, day.Date)
		}
	}
	return result, rows.Err()
}

func (a *App) handlePricingHolidayCalendar(w http.ResponseWriter, r *http.Request) error {
	if err := requireMethod(r, http.MethodGet); err != nil {
		return err
	}
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
}

func (a *App) handlePricingCalendarSettings(w http.ResponseWriter, r *http.Request) error {
	if err := requireMethod(r, http.MethodPut); err != nil {
		return err
	}
	var payload pricingCalendarSettingsPayload
	if err := decodeJSON(r, &payload); err != nil {
		return err
	}
	if payload.PeakOnMakeupDays == nil {
		return validationError("缺少调休日计费选项")
	}
	if _, err := a.db.ExecContext(r.Context(), `UPDATE pricing_calendar_settings SET peak_on_makeup_days = ? WHERE id = 1`, *payload.PeakOnMakeupDays); err != nil {
		return err
	}
	a.invalidateUsagePrices()
	writeJSON(w, http.StatusOK, map[string]bool{"peak_on_makeup_days": *payload.PeakOnMakeupDays})
	return nil
}

type holidayCNDocument struct {
	Year   int      `json:"year"`
	Papers []string `json:"papers"`
	Days   []struct {
		Name     string `json:"name"`
		Date     string `json:"date"`
		IsOffDay *bool  `json:"isOffDay"`
	} `json:"days"`
}

func fetchHolidayCNDays(ctx context.Context, client *http.Client, year int, sourceURL string) ([]pricingHolidayDay, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, appError("holiday_source_unavailable", http.StatusBadGateway, "无法获取 holiday-cn 节假日数据")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, appError("holiday_source_unavailable", http.StatusBadGateway, fmt.Sprintf("holiday-cn 返回 HTTP %d", response.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, holidayCNMaxResponseBytes+1))
	if err != nil || len(body) > holidayCNMaxResponseBytes {
		return nil, appError("holiday_source_invalid", http.StatusBadGateway, "holiday-cn 响应过大或无法读取")
	}
	var document holidayCNDocument
	if err := json.Unmarshal(body, &document); err != nil || document.Year != year {
		return nil, appError("holiday_source_invalid", http.StatusBadGateway, "holiday-cn 年份或数据格式不正确")
	}
	if len(document.Papers) == 0 || len(document.Days) == 0 {
		return nil, validationError("该年份的节假日安排尚未公布")
	}
	if len(document.Days) > 400 {
		return nil, appError("holiday_source_invalid", http.StatusBadGateway, "holiday-cn 日期数量异常")
	}
	days := make([]pricingHolidayDay, 0, len(document.Days))
	seen := map[string]bool{}
	hasOffDay := false
	for _, item := range document.Days {
		date, err := time.Parse("2006-01-02", item.Date)
		if err != nil || date.Format("2006-01-02") != item.Date || date.Year() < year-1 || date.Year() > year+1 || item.IsOffDay == nil || seen[item.Date] {
			return nil, appError("holiday_source_invalid", http.StatusBadGateway, "holiday-cn 日期数据不正确")
		}
		seen[item.Date] = true
		hasOffDay = hasOffDay || *item.IsOffDay
		days = append(days, pricingHolidayDay{Date: item.Date, Name: strings.TrimSpace(item.Name), IsOffDay: *item.IsOffDay})
	}
	if !hasOffDay {
		return nil, appError("holiday_source_invalid", http.StatusBadGateway, "holiday-cn 缺少休息日数据")
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })
	return days, nil
}

func (a *App) syncPricingCalendar(ctx context.Context, client *http.Client, year int, sourceURL string) (pricingHolidayCalendarResponse, error) {
	if err := pricingCalendarYear(year); err != nil {
		return pricingHolidayCalendarResponse{}, err
	}
	days, err := fetchHolidayCNDays(ctx, client, year, sourceURL)
	if err != nil {
		return pricingHolidayCalendarResponse{}, err
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return pricingHolidayCalendarResponse{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO pricing_holiday_years (year, source_url, updated_at, synced_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(year) DO UPDATE SET source_url = excluded.source_url,
			updated_at = excluded.updated_at, synced_at = excluded.synced_at
	`, year, sourceURL, dbTime(time.Now()), dbTime(time.Now())); err != nil {
		return pricingHolidayCalendarResponse{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM pricing_holidays WHERE year = ?`, year); err != nil {
		return pricingHolidayCalendarResponse{}, err
	}
	for _, day := range days {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO pricing_holidays (date, year, name, is_off_day) VALUES (?, ?, ?, ?)
			ON CONFLICT(date) DO UPDATE SET year = excluded.year, name = excluded.name,
				is_off_day = excluded.is_off_day
			WHERE excluded.year >= pricing_holidays.year
		`, day.Date, year, day.Name, day.IsOffDay); err != nil {
			return pricingHolidayCalendarResponse{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return pricingHolidayCalendarResponse{}, err
	}
	a.invalidateUsagePrices()
	return a.pricingHolidayCalendarForYear(ctx, year)
}

func (a *App) handlePricingHolidayCalendarSync(w http.ResponseWriter, r *http.Request) error {
	if err := requireMethod(r, http.MethodPost); err != nil {
		return err
	}
	var payload pricingHolidayCalendarPayload
	if err := decodeJSON(r, &payload); err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	result, err := a.syncPricingCalendar(r.Context(), client, payload.Year, fmt.Sprintf(holidayCNURLPattern, payload.Year))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, result)
	return nil
}
