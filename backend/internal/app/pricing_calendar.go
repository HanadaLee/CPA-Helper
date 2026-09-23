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
	peakRanges       [][2]int // Beijing local minutes, half-open intervals
}

type pricingPeakPeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func peakMinute(value string, isEnd bool) (int, error) {
	if len(value) != 5 || value[2] != ':' {
		return 0, validationError("高峰时段必须使用 HH:MM 格式")
	}
	for _, position := range []int{0, 1, 3, 4} {
		if value[position] < '0' || value[position] > '9' {
			return 0, validationError("高峰时段必须使用 HH:MM 格式")
		}
	}
	hour, hourErr := strconv.Atoi(value[:2])
	minute, minuteErr := strconv.Atoi(value[3:])
	if hourErr != nil || minuteErr != nil || minute < 0 || minute > 59 || hour < 0 || hour > 24 || (hour == 24 && (!isEnd || minute != 0)) {
		return 0, validationError("高峰时段时间无效")
	}
	return hour*60 + minute, nil
}

func validatePricingPeakPeriods(periods []pricingPeakPeriod) error {
	if len(periods) == 0 || len(periods) > 8 {
		return validationError("高峰时段需要设置 1 至 8 段")
	}
	lastEnd := 0
	for index, period := range periods {
		start, err := peakMinute(period.Start, false)
		if err != nil {
			return err
		}
		end, err := peakMinute(period.End, true)
		if err != nil {
			return err
		}
		if start >= end || (index > 0 && start < lastEnd) {
			return validationError("高峰时段必须按时间排序，且不能重叠或跨日")
		}
		lastEnd = end
	}
	return nil
}

// Peak hours follow the configured Beijing-time intervals, excluding
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
	minute := local.Hour()*60 + local.Minute()
	ranges := calendar.peakRanges
	if len(ranges) == 0 {
		ranges = [][2]int{{9 * 60, 12 * 60}, {14 * 60, 18 * 60}}
	}
	for _, period := range ranges {
		if minute >= period[0] && minute < period[1] {
			return true
		}
	}
	return false
}

func (a *App) loadPricingCalendar(ctx context.Context) (*pricingCalendar, error) {
	calendar := &pricingCalendar{years: map[int]map[string]bool{}}
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var periodsJSON string
	if err := tx.QueryRowContext(ctx, `SELECT peak_on_makeup_days, peak_periods_json FROM pricing_calendar_settings WHERE id = 1`).Scan(&calendar.peakOnMakeupDays, &periodsJSON); err != nil {
		return nil, err
	}
	var periods []pricingPeakPeriod
	if err := json.Unmarshal([]byte(periodsJSON), &periods); err != nil {
		return nil, err
	}
	if err := validatePricingPeakPeriods(periods); err != nil {
		return nil, err
	}
	calendar.peakRanges = make([][2]int, 0, len(periods))
	for _, period := range periods {
		start, _ := peakMinute(period.Start, false)
		end, _ := peakMinute(period.End, true)
		calendar.peakRanges = append(calendar.peakRanges, [2]int{start, end})
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
	PeakPeriods      []pricingPeakPeriod `json:"peak_periods"`
}

type pricingHolidayCalendarPayload struct {
	Year int `json:"year"`
}

type pricingCalendarSettingsPayload struct {
	PeakOnMakeupDays *bool                `json:"peak_on_makeup_days"`
	PeakPeriods      *[]pricingPeakPeriod `json:"peak_periods"`
}

func pricingCalendarYear(year int) error {
	if year < 2000 || year > 2100 {
		return validationError("节假日年份必须在 2000 至 2100 之间")
	}
	return nil
}

func (a *App) pricingHolidayCalendarForYear(ctx context.Context, year int) (pricingHolidayCalendarResponse, error) {
	result := pricingHolidayCalendarResponse{Year: year, Dates: []string{}, Days: []pricingHolidayDay{}}
	var periodsJSON string
	if err := a.db.QueryRowContext(ctx, `SELECT peak_on_makeup_days, peak_periods_json FROM pricing_calendar_settings WHERE id = 1`).Scan(&result.PeakOnMakeupDays, &periodsJSON); err != nil {
		return result, err
	}
	if err := json.Unmarshal([]byte(periodsJSON), &result.PeakPeriods); err != nil {
		return result, err
	}
	if err := validatePricingPeakPeriods(result.PeakPeriods); err != nil {
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
	if payload.PeakOnMakeupDays == nil && payload.PeakPeriods == nil {
		return validationError("缺少峰谷计费配置")
	}
	var settings struct {
		PeakOnMakeupDays bool
		PeriodsJSON      string
	}
	if err := a.db.QueryRowContext(r.Context(), `SELECT peak_on_makeup_days, peak_periods_json FROM pricing_calendar_settings WHERE id = 1`).Scan(&settings.PeakOnMakeupDays, &settings.PeriodsJSON); err != nil {
		return err
	}
	if payload.PeakOnMakeupDays != nil {
		settings.PeakOnMakeupDays = *payload.PeakOnMakeupDays
	}
	if payload.PeakPeriods != nil {
		if err := validatePricingPeakPeriods(*payload.PeakPeriods); err != nil {
			return err
		}
		encoded, err := json.Marshal(*payload.PeakPeriods)
		if err != nil {
			return err
		}
		settings.PeriodsJSON = string(encoded)
	}
	if _, err := a.db.ExecContext(r.Context(), `UPDATE pricing_calendar_settings SET peak_on_makeup_days = ?, peak_periods_json = ? WHERE id = 1`, settings.PeakOnMakeupDays, settings.PeriodsJSON); err != nil {
		return err
	}
	var periods []pricingPeakPeriod
	if err := json.Unmarshal([]byte(settings.PeriodsJSON), &periods); err != nil {
		return err
	}
	a.invalidateUsagePrices()
	writeJSON(w, http.StatusOK, map[string]any{"peak_on_makeup_days": settings.PeakOnMakeupDays, "peak_periods": periods})
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
