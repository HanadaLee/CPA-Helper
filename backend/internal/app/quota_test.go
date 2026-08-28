package app_test

import (
	"net/http"
	"strconv"
	"testing"

	backendApp "cpa-helper/backend/internal/app"
)

type quotaAPIUserResponse struct {
	ID    int `json:"id"`
	Quota struct {
		Unlimited        bool     `json:"unlimited"`
		CanCreateKeys    bool     `json:"can_create_keys"`
		LifetimeQuotaUSD *float64 `json:"lifetime_quota_usd"`
	} `json:"quota"`
}

type quotaAPIStatusResponse struct {
	Unlimited        bool     `json:"unlimited"`
	LifetimeQuotaUSD *float64 `json:"lifetime_quota_usd"`
	MonthlyQuotaUSD  *float64 `json:"monthly_quota_usd"`
	WeeklyQuotaUSD   *float64 `json:"weekly_quota_usd"`
	DailyQuotaUSD    *float64 `json:"daily_quota_usd"`
	CanCreateKeys    bool     `json:"can_create_keys"`
}

func TestQuotaAPIPermissionsAndAccountStatus(t *testing.T) {
	t.Setenv("CPA_HELPER_DATA_DIR", t.TempDir())

	app, err := backendApp.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer app.Close()

	handler := app.Routes()
	adminCookies := requestJSON(t, handler, http.MethodPost, "/api/auth/setup", map[string]any{
		"username": "admin",
		"password": "test-password",
		"nickname": "Admin",
	}, nil, nil)

	member := quotaAPIUserResponse{}
	requestJSON(t, handler, http.MethodPost, "/api/users", map[string]any{
		"username": "member",
		"password": "member-password",
		"nickname": "Member",
		"is_admin": false,
	}, adminCookies, &member)
	if !member.Quota.Unlimited || !member.Quota.CanCreateKeys {
		t.Fatalf("new user quota = %#v, want unlimited and creatable", member.Quota)
	}

	lifetime := 1.25
	monthly := 0.5
	weekly := 0.3
	daily := 0.1
	updated := quotaAPIStatusResponse{}
	requestJSON(t, handler, http.MethodPut, "/api/users/"+strconv.Itoa(member.ID)+"/quota", map[string]any{
		"lifetime_quota_usd": lifetime,
		"monthly_quota_usd":  monthly,
		"weekly_quota_usd":   weekly,
		"daily_quota_usd":    daily,
	}, adminCookies, &updated)
	if updated.Unlimited || updated.LifetimeQuotaUSD == nil || *updated.LifetimeQuotaUSD != lifetime ||
		updated.MonthlyQuotaUSD == nil || *updated.MonthlyQuotaUSD != monthly ||
		updated.WeeklyQuotaUSD == nil || *updated.WeeklyQuotaUSD != weekly ||
		updated.DailyQuotaUSD == nil || *updated.DailyQuotaUSD != daily {
		t.Fatalf("updated quota = %#v, want all four configured quotas", updated)
	}

	memberCookies := requestJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]any{
		"username": "member",
		"password": "member-password",
	}, nil, nil)

	accountQuota := quotaAPIStatusResponse{}
	requestJSON(t, handler, http.MethodGet, "/api/account/quota", nil, memberCookies, &accountQuota)
	if accountQuota.LifetimeQuotaUSD == nil || *accountQuota.LifetimeQuotaUSD != lifetime ||
		accountQuota.MonthlyQuotaUSD == nil || *accountQuota.MonthlyQuotaUSD != monthly ||
		accountQuota.WeeklyQuotaUSD == nil || *accountQuota.WeeklyQuotaUSD != weekly ||
		accountQuota.DailyQuotaUSD == nil || *accountQuota.DailyQuotaUSD != daily {
		t.Fatalf("account quota = %#v, want all four member quotas", accountQuota)
	}

	requestJSONExpectStatus(t, handler, http.MethodPut, "/api/users/"+strconv.Itoa(member.ID)+"/quota", map[string]any{
		"lifetime_quota_usd": nil,
	}, memberCookies, http.StatusForbidden)
}

func TestQuotaExhaustedAccountCannotCreateAPIKey(t *testing.T) {
	t.Setenv("CPA_HELPER_DATA_DIR", t.TempDir())

	app, err := backendApp.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer app.Close()

	handler := app.Routes()
	cookies := requestJSON(t, handler, http.MethodPost, "/api/auth/setup", map[string]any{
		"username": "admin",
		"password": "test-password",
		"nickname": "Admin",
	}, nil, nil)

	zero := 0
	requestJSON(t, handler, http.MethodPut, "/api/users/1/quota", map[string]any{
		"lifetime_quota_usd": zero,
	}, cookies, nil)

	requestJSONExpectStatus(t, handler, http.MethodPost, "/api/api-keys", map[string]any{
		"description": "VSCode",
	}, cookies, http.StatusConflict)
}
