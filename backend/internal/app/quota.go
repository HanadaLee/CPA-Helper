package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

const quotaPauseReasonExhausted = "quota_exhausted"

type userQuotaPayload struct {
	LifetimeQuotaUSD *float64 `json:"lifetime_quota_usd"`
	MonthlyQuotaUSD  *float64 `json:"monthly_quota_usd"`
	WeeklyQuotaUSD   *float64 `json:"weekly_quota_usd"`
	DailyQuotaUSD    *float64 `json:"daily_quota_usd"`
}

type UserQuotaStatusResponse struct {
	Unlimited            bool       `json:"unlimited"`
	LifetimeQuotaUSD     *float64   `json:"lifetime_quota_usd"`
	LifetimeRemainingUSD *float64   `json:"lifetime_remaining_usd"`
	MonthlyQuotaUSD      *float64   `json:"monthly_quota_usd"`
	MonthlyUsedUSD       float64    `json:"monthly_used_usd"`
	MonthlyRemainingUSD  *float64   `json:"monthly_remaining_usd"`
	QuotaMonth           string     `json:"quota_month"`
	WeeklyQuotaUSD       *float64   `json:"weekly_quota_usd"`
	WeeklyUsedUSD        float64    `json:"weekly_used_usd"`
	WeeklyRemainingUSD   *float64   `json:"weekly_remaining_usd"`
	QuotaWeek            string     `json:"quota_week"`
	DailyQuotaUSD        *float64   `json:"daily_quota_usd"`
	DailyUsedUSD         float64    `json:"daily_used_usd"`
	DailyRemainingUSD    *float64   `json:"daily_remaining_usd"`
	QuotaDay             string     `json:"quota_day"`
	Paused               bool       `json:"paused"`
	PausedAt             *time.Time `json:"paused_at"`
	PauseReason          *string    `json:"pause_reason"`
	SyncError            *string    `json:"sync_error"`
	UnpricedRecords      int        `json:"unpriced_records"`
	CanCreateKeys        bool       `json:"can_create_keys"`
	StartedAt            *time.Time `json:"started_at"`
}

func (a *App) handleCurrentUserQuota(w http.ResponseWriter, r *http.Request) error {
	user, err := a.readyUser(r.Context(), r)
	if err != nil {
		return err
	}
	if err := requireMethod(r, http.MethodGet); err != nil {
		return err
	}
	status, err := a.userQuotaStatus(r.Context(), user.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, status)
	return nil
}

func (a *App) updateUserQuota(ctx context.Context, userID int, payload userQuotaPayload) (UserQuotaStatusResponse, error) {
	user, err := a.getUser(ctx, userID)
	if err != nil {
		return UserQuotaStatusResponse{}, err
	}
	lifetime, err := normalizedQuotaAmount(payload.LifetimeQuotaUSD)
	if err != nil {
		return UserQuotaStatusResponse{}, err
	}
	monthly, err := normalizedQuotaAmount(payload.MonthlyQuotaUSD)
	if err != nil {
		return UserQuotaStatusResponse{}, err
	}
	weekly, err := normalizedQuotaAmount(payload.WeeklyQuotaUSD)
	if err != nil {
		return UserQuotaStatusResponse{}, err
	}
	daily, err := normalizedQuotaAmount(payload.DailyQuotaUSD)
	if err != nil {
		return UserQuotaStatusResponse{}, err
	}

	currentTime := time.Now()
	now := dbTime(currentTime)
	currentMonth := quotaMonth(currentTime)
	currentWeek := quotaWeek(currentTime)
	currentDay := quotaDay(currentTime)
	var startedAt any
	if lifetime != nil || monthly != nil || weekly != nil || daily != nil {
		if user.QuotaStartedAt != nil {
			startedAt = dbTime(*user.QuotaStartedAt)
		} else {
			startedAt = now
		}
	}
	monthUsed := 0.0
	monthValue := ""
	if monthly != nil {
		monthValue = currentMonth
		if user.QuotaMonthlyUSD != nil && user.QuotaMonth == currentMonth {
			monthUsed = mathRound(user.QuotaMonthUsedUSD, 8)
		}
	}
	weekUsed := 0.0
	weekValue := ""
	if weekly != nil {
		weekValue = currentWeek
		if user.QuotaWeeklyUSD != nil && user.QuotaWeek == currentWeek {
			weekUsed = mathRound(user.QuotaWeekUsedUSD, 8)
		}
	}
	dayUsed := 0.0
	dayValue := ""
	if daily != nil {
		dayValue = currentDay
		if user.QuotaDailyUSD != nil && user.QuotaDay == currentDay {
			dayUsed = mathRound(user.QuotaDayUsedUSD, 8)
		}
	}
	_, err = a.db.ExecContext(ctx, `
		UPDATE users
		SET quota_lifetime_usd = ?, quota_monthly_usd = ?, quota_weekly_usd = ?, quota_daily_usd = ?,
		    quota_started_at = ?, quota_month = ?, quota_month_used_usd = ?,
		    quota_week = ?, quota_week_used_usd = ?, quota_day = ?, quota_day_used_usd = ?,
		    quota_sync_error = NULL,
		    updated_at = ?
		WHERE id = ?
	`, quotaAmountArg(lifetime), quotaAmountArg(monthly), quotaAmountArg(weekly), quotaAmountArg(daily),
		startedAt, monthValue, monthUsed, weekValue, weekUsed, dayValue, dayUsed, now, userID)
	if err != nil {
		return UserQuotaStatusResponse{}, err
	}
	user, err = a.getUser(ctx, userID)
	if err != nil {
		return UserQuotaStatusResponse{}, err
	}
	if quotaHasAvailable(user) {
		_ = a.restoreQuotaPausedUserIfAvailable(ctx, user.ID)
	} else {
		_ = a.pauseUserKeysForQuota(ctx, user.ID, quotaPauseReasonExhausted)
	}
	return a.userQuotaStatus(ctx, userID)
}

func (a *App) userQuotaStatus(ctx context.Context, userID int) (UserQuotaStatusResponse, error) {
	user, err := a.getUser(ctx, userID)
	if err != nil {
		return UserQuotaStatusResponse{}, err
	}
	user, err = a.ensureQuotaPeriods(ctx, user)
	if err != nil {
		return UserQuotaStatusResponse{}, err
	}
	if user.QuotaPausedAt != nil {
		_ = a.restoreQuotaPausedUserIfAvailable(ctx, user.ID)
		user, err = a.getUser(ctx, userID)
		if err != nil {
			return UserQuotaStatusResponse{}, err
		}
	} else if !quotaIsUnlimited(user) && !quotaHasAvailable(user) {
		_ = a.pauseUserKeysForQuota(ctx, user.ID, quotaPauseReasonExhausted)
		user, err = a.getUser(ctx, userID)
		if err != nil {
			return UserQuotaStatusResponse{}, err
		}
	}
	return quotaStatusFromUser(user), nil
}

func (a *App) ensureUserQuotaReadyForKeys(ctx context.Context, userID int) error {
	status, err := a.userQuotaStatus(ctx, userID)
	if err != nil {
		return err
	}
	if !status.CanCreateKeys {
		return conflictError("用户额度已用尽，API KEY 已暂停，请联系管理员补充额度")
	}
	return nil
}

func (a *App) applyQuotaCharge(ctx context.Context, record UsageRecord) error {
	if record.UsageUsername == nil || strings.TrimSpace(*record.UsageUsername) == "" {
		return nil
	}
	user, err := a.userByUsername(ctx, *record.UsageUsername)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	user, err = a.ensureQuotaPeriods(ctx, user)
	if err != nil {
		return err
	}
	if quotaIsUnlimited(user) {
		if user.QuotaPausedAt != nil {
			_ = a.restoreQuotaPausedUserIfAvailable(ctx, user.ID)
		}
		return nil
	}

	var existing int
	err = a.db.QueryRowContext(ctx, `SELECT id FROM user_quota_charges WHERE usage_record_id = ?`, record.ID).Scan(&existing)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	prices, err := a.priceMap(ctx)
	if err != nil {
		return err
	}
	amount, unpriced := recordCost(record, prices)
	amount = mathRound(amount, 8)
	dailyDeducted, weeklyDeducted, monthlyDeducted, lifetimeDeducted := 0.0, 0.0, 0.0, 0.0
	remaining := amount
	if !unpriced && remaining > 0 {
		if dailyRemaining := quotaDailyRemaining(user); dailyRemaining != nil && *dailyRemaining > 0 {
			dailyDeducted = minQuotaAmount(remaining, *dailyRemaining)
			user.QuotaDayUsedUSD = mathRound(user.QuotaDayUsedUSD+dailyDeducted, 8)
			remaining = mathRound(remaining-dailyDeducted, 8)
		}
		if remaining > 0 {
			if weeklyRemaining := quotaWeeklyRemaining(user); weeklyRemaining != nil && *weeklyRemaining > 0 {
				weeklyDeducted = minQuotaAmount(remaining, *weeklyRemaining)
				user.QuotaWeekUsedUSD = mathRound(user.QuotaWeekUsedUSD+weeklyDeducted, 8)
				remaining = mathRound(remaining-weeklyDeducted, 8)
			}
		}
		if remaining > 0 {
			if monthlyRemaining := quotaMonthlyRemaining(user); monthlyRemaining != nil && *monthlyRemaining > 0 {
				monthlyDeducted = minQuotaAmount(remaining, *monthlyRemaining)
				user.QuotaMonthUsedUSD = mathRound(user.QuotaMonthUsedUSD+monthlyDeducted, 8)
				remaining = mathRound(remaining-monthlyDeducted, 8)
			}
		}
		if remaining > 0 && user.QuotaLifetimeUSD != nil && *user.QuotaLifetimeUSD > 0 {
			lifetimeDeducted = minQuotaAmount(remaining, *user.QuotaLifetimeUSD)
			nextLifetime := mathRound(*user.QuotaLifetimeUSD-lifetimeDeducted, 8)
			user.QuotaLifetimeUSD = &nextLifetime
			remaining = mathRound(remaining-lifetimeDeducted, 8)
		}
	}
	if unpriced {
		user.QuotaUnpricedRecords++
	}

	now := dbTime(time.Now())
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_quota_charges (
			usage_record_id, user_id, usage_username, amount_usd,
			daily_deducted_usd, weekly_deducted_usd, monthly_deducted_usd, lifetime_deducted_usd, unpriced,
			quota_day, quota_week, quota_month, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, record.ID, user.ID, user.Username, amount, dailyDeducted, weeklyDeducted, monthlyDeducted,
		lifetimeDeducted, unpriced, nonBlank(user.QuotaDay, quotaDay(record.Timestamp)),
		nonBlank(user.QuotaWeek, quotaWeek(record.Timestamp)), nonBlank(user.QuotaMonth, quotaMonth(record.Timestamp)), now)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE users
		SET quota_day_used_usd = ?, quota_week_used_usd = ?, quota_month_used_usd = ?, quota_lifetime_usd = ?,
		    quota_unpriced_records = ?, updated_at = ?
		WHERE id = ?
	`, mathRound(user.QuotaDayUsedUSD, 8), mathRound(user.QuotaWeekUsedUSD, 8),
		mathRound(user.QuotaMonthUsedUSD, 8), quotaAmountArg(user.QuotaLifetimeUSD), user.QuotaUnpricedRecords, now, user.ID)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	if remaining > 0 || !quotaHasAvailable(user) {
		_ = a.pauseUserKeysForQuota(ctx, user.ID, quotaPauseReasonExhausted)
	}
	return nil
}

func (a *App) ensureQuotaPeriods(ctx context.Context, user UserRecord) (UserRecord, error) {
	currentTime := time.Now()
	changed := false
	if user.QuotaDailyUSD != nil && user.QuotaDay != quotaDay(currentTime) {
		user.QuotaDay = quotaDay(currentTime)
		user.QuotaDayUsedUSD = 0
		changed = true
	}
	if user.QuotaWeeklyUSD != nil && user.QuotaWeek != quotaWeek(currentTime) {
		user.QuotaWeek = quotaWeek(currentTime)
		user.QuotaWeekUsedUSD = 0
		changed = true
	}
	if user.QuotaMonthlyUSD != nil && user.QuotaMonth != quotaMonth(currentTime) {
		user.QuotaMonth = quotaMonth(currentTime)
		user.QuotaMonthUsedUSD = 0
		changed = true
	}
	if !changed {
		return user, nil
	}
	_, err := a.db.ExecContext(ctx, `
		UPDATE users
		SET quota_day = ?, quota_day_used_usd = ?, quota_week = ?, quota_week_used_usd = ?,
		    quota_month = ?, quota_month_used_usd = ?, quota_sync_error = NULL, updated_at = ?
		WHERE id = ?
	`, user.QuotaDay, user.QuotaDayUsedUSD, user.QuotaWeek, user.QuotaWeekUsedUSD,
		user.QuotaMonth, user.QuotaMonthUsedUSD, dbTime(currentTime), user.ID)
	if err != nil {
		return UserRecord{}, err
	}
	return a.getUser(ctx, user.ID)
}

func (a *App) pauseUserKeysForQuota(ctx context.Context, userID int, reason string) error {
	user, err := a.getUser(ctx, userID)
	if err != nil {
		return err
	}
	if user.QuotaPausedAt != nil && user.QuotaSyncError == nil {
		return nil
	}
	now := dbTime(time.Now())
	_, err = a.db.ExecContext(ctx, `
		UPDATE users
		SET quota_paused_at = COALESCE(quota_paused_at, ?),
		    quota_pause_reason = ?, quota_sync_error = NULL, updated_at = ?
		WHERE id = ?
	`, now, reason, now, userID)
	if err != nil {
		return err
	}
	keys, err := a.userAPIKeys(ctx, userID)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := a.removeRemoteAPIKeyHash(ctx, key.APIKeyHash); err != nil {
			_ = a.setQuotaSyncError(ctx, userID, err)
			return nil
		}
	}
	return nil
}

func (a *App) restoreQuotaPausedUserIfAvailable(ctx context.Context, userID int) error {
	user, err := a.getUser(ctx, userID)
	if err != nil {
		return err
	}
	user, err = a.ensureQuotaPeriods(ctx, user)
	if err != nil {
		return err
	}
	if user.QuotaPausedAt == nil || user.DisabledAt != nil || !quotaHasAvailable(user) {
		return nil
	}
	keys, err := a.userAPIKeys(ctx, userID)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if key.APIKey == nil {
			return a.setQuotaSyncMessage(ctx, userID, "存在无法恢复的 API KEY，请重新绑定后再恢复")
		}
	}
	restored := []string{}
	for _, key := range keys {
		if key.APIKey == nil {
			continue
		}
		if err := a.addRemoteAPIKey(ctx, *key.APIKey); err != nil {
			for _, hash := range restored {
				_ = a.removeRemoteAPIKeyHash(ctx, hash)
			}
			_ = a.setQuotaSyncError(ctx, userID, err)
			return nil
		}
		restored = append(restored, key.APIKeyHash)
	}
	_, err = a.db.ExecContext(ctx, `
		UPDATE users
		SET quota_paused_at = NULL, quota_pause_reason = NULL,
		    quota_sync_error = NULL, updated_at = ?
		WHERE id = ?
	`, dbTime(time.Now()), userID)
	return err
}

func (a *App) setQuotaSyncError(ctx context.Context, userID int, err error) error {
	return a.setQuotaSyncMessage(ctx, userID, err.Error())
}

func (a *App) setQuotaSyncMessage(ctx context.Context, userID int, message string) error {
	if len(message) > 1000 {
		message = message[:1000]
	}
	_, err := a.db.ExecContext(ctx, `UPDATE users SET quota_sync_error = ?, updated_at = ? WHERE id = ?`, message, dbTime(time.Now()), userID)
	return err
}

func quotaStatusFromUser(user UserRecord) UserQuotaStatusResponse {
	monthlyRemaining := quotaMonthlyRemaining(user)
	weeklyRemaining := quotaWeeklyRemaining(user)
	dailyRemaining := quotaDailyRemaining(user)
	var lifetimeRemaining *float64
	if user.QuotaLifetimeUSD != nil {
		value := mathRound(maxQuotaAmount(*user.QuotaLifetimeUSD, 0), 8)
		lifetimeRemaining = &value
	}
	return UserQuotaStatusResponse{
		Unlimited:            quotaIsUnlimited(user),
		LifetimeQuotaUSD:     user.QuotaLifetimeUSD,
		LifetimeRemainingUSD: lifetimeRemaining,
		MonthlyQuotaUSD:      user.QuotaMonthlyUSD,
		MonthlyUsedUSD:       mathRound(user.QuotaMonthUsedUSD, 8),
		MonthlyRemainingUSD:  monthlyRemaining,
		QuotaMonth:           user.QuotaMonth,
		WeeklyQuotaUSD:       user.QuotaWeeklyUSD,
		WeeklyUsedUSD:        mathRound(user.QuotaWeekUsedUSD, 8),
		WeeklyRemainingUSD:   weeklyRemaining,
		QuotaWeek:            user.QuotaWeek,
		DailyQuotaUSD:        user.QuotaDailyUSD,
		DailyUsedUSD:         mathRound(user.QuotaDayUsedUSD, 8),
		DailyRemainingUSD:    dailyRemaining,
		QuotaDay:             user.QuotaDay,
		Paused:               user.QuotaPausedAt != nil,
		PausedAt:             user.QuotaPausedAt,
		PauseReason:          user.QuotaPauseReason,
		SyncError:            user.QuotaSyncError,
		UnpricedRecords:      user.QuotaUnpricedRecords,
		CanCreateKeys:        user.QuotaPausedAt == nil && quotaHasAvailable(user),
		StartedAt:            user.QuotaStartedAt,
	}
}

func quotaIsUnlimited(user UserRecord) bool {
	return user.QuotaLifetimeUSD == nil && user.QuotaMonthlyUSD == nil &&
		user.QuotaWeeklyUSD == nil && user.QuotaDailyUSD == nil
}

func quotaHasAvailable(user UserRecord) bool {
	if quotaIsUnlimited(user) {
		return true
	}
	if dailyRemaining := quotaDailyRemaining(user); dailyRemaining != nil && *dailyRemaining > 0 {
		return true
	}
	if weeklyRemaining := quotaWeeklyRemaining(user); weeklyRemaining != nil && *weeklyRemaining > 0 {
		return true
	}
	if monthlyRemaining := quotaMonthlyRemaining(user); monthlyRemaining != nil && *monthlyRemaining > 0 {
		return true
	}
	return user.QuotaLifetimeUSD != nil && *user.QuotaLifetimeUSD > 0
}

func quotaDailyRemaining(user UserRecord) *float64 {
	if user.QuotaDailyUSD == nil {
		return nil
	}
	remaining := mathRound(*user.QuotaDailyUSD-user.QuotaDayUsedUSD, 8)
	remaining = maxQuotaAmount(remaining, 0)
	return &remaining
}

func quotaWeeklyRemaining(user UserRecord) *float64 {
	if user.QuotaWeeklyUSD == nil {
		return nil
	}
	remaining := mathRound(*user.QuotaWeeklyUSD-user.QuotaWeekUsedUSD, 8)
	remaining = maxQuotaAmount(remaining, 0)
	return &remaining
}

func quotaMonthlyRemaining(user UserRecord) *float64 {
	if user.QuotaMonthlyUSD == nil {
		return nil
	}
	remaining := mathRound(*user.QuotaMonthlyUSD-user.QuotaMonthUsedUSD, 8)
	remaining = maxQuotaAmount(remaining, 0)
	return &remaining
}

func normalizedQuotaAmount(value *float64) (*float64, error) {
	if value == nil {
		return nil, nil
	}
	if math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 {
		return nil, validationError("额度金额不能小于 0")
	}
	normalized := mathRound(*value, 8)
	return &normalized, nil
}

func quotaAmountArg(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func quotaMonth(value time.Time) string {
	return value.In(appTimeLocation).Format("2006-01")
}

func quotaWeek(value time.Time) string {
	year, week := value.In(appTimeLocation).ISOWeek()
	return fmt.Sprintf("%04d-W%02d", year, week)
}

func quotaDay(value time.Time) string {
	return value.In(appTimeLocation).Format("2006-01-02")
}

func minQuotaAmount(left, right float64) float64 {
	if left < right {
		return mathRound(left, 8)
	}
	return mathRound(right, 8)
}

func maxQuotaAmount(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func isUniqueConstraintError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}
