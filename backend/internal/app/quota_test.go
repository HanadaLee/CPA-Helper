package app_test

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	backendApp "cpa-helper/backend/internal/app"
)

type quotaAPIUserResponse struct {
	ID    int `json:"id"`
	Quota struct {
		Unlimited      bool     `json:"unlimited"`
		CanCreateKeys  bool     `json:"can_create_keys"`
		WeeklyQuotaUSD *float64 `json:"weekly_quota_usd"`
		DailyQuotaUSD  *float64 `json:"daily_quota_usd"`
	} `json:"quota"`
}

func TestQuotaCardsAPIPermissionsAndUserIsolation(t *testing.T) {
	t.Setenv("CPA_HELPER_DATA_DIR", t.TempDir())
	a, err := backendApp.New()
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	h := a.Routes()
	admin := requestJSON(t, h, http.MethodPost, "/api/auth/setup", map[string]any{"username": "admin", "password": "admin-password", "nickname": "Admin"}, nil, nil)
	member := quotaAPIUserResponse{}
	requestJSON(t, h, http.MethodPost, "/api/users", map[string]any{"username": "member", "password": "member-password", "nickname": "Member"}, admin, &member)
	other := quotaAPIUserResponse{}
	requestJSON(t, h, http.MethodPost, "/api/users", map[string]any{"username": "other", "password": "other-password", "nickname": "Other"}, admin, &other)
	memberCookies := requestJSON(t, h, http.MethodPost, "/api/auth/login", map[string]any{"username": "member", "password": "member-password"}, nil, nil)
	otherCookies := requestJSON(t, h, http.MethodPost, "/api/auth/login", map[string]any{"username": "other", "password": "other-password"}, nil, nil)
	issue := map[string]any{"user_ids": []int{member.ID}, "kind": "credit", "amount_usd": 10, "name": "API credit"}
	for _, path := range []string{"/api/quota/cards/issue", "/api/quota/cards/revoke", "/api/quota/reset"} {
		requestJSONExpectStatus(t, h, http.MethodPost, path, issue, memberCookies, http.StatusForbidden)
	}
	requestJSONExpectStatus(t, h, http.MethodGet, "/api/quota/cards", nil, memberCookies, http.StatusForbidden)
	requestJSONExpectStatus(t, h, http.MethodPost, "/api/quota/cards/issue", map[string]any{"user_ids": []int{member.ID}, "kind": "credit", "amount_usd": -1}, admin, http.StatusUnprocessableEntity)
	requestJSONExpectStatus(t, h, http.MethodPost, "/api/quota/cards/issue", map[string]any{"user_ids": []int{member.ID}, "kind": "credit", "amount_usd": 1, "expires_at": time.Now().Add(-time.Hour)}, admin, http.StatusUnprocessableEntity)
	requestJSONExpectStatus(t, h, http.MethodPut, "/api/users/"+strconv.Itoa(member.ID)+"/quota", map[string]any{"monthly_quota_usd": 100}, admin, http.StatusUnprocessableEntity)
	requestJSON(t, h, http.MethodPost, "/api/quota/cards/issue", issue, admin, nil)
	status := map[string]any{}
	requestJSON(t, h, http.MethodGet, "/api/account/quota", nil, memberCookies, &status)
	if status["available_usd"] != float64(10) || status["paused"] != false {
		t.Fatalf("issued balance=%v", status)
	}
	if _, ok := status["monthly_quota_usd"]; ok {
		t.Fatal("removed monthly quota exposed")
	}
	if _, ok := status["lifetime_quota_usd"]; ok {
		t.Fatal("removed lifetime quota exposed")
	}
	type cardList struct {
		Items []struct {
			ID     int    `json:"id"`
			UserID int    `json:"user_id"`
			Status string `json:"status"`
		} `json:"items"`
		Total int `json:"total"`
	}
	list := cardList{}
	requestJSON(t, h, http.MethodGet, "/api/account/quota/cards?user_id="+strconv.Itoa(member.ID), nil, otherCookies, &list)
	if list.Total != 0 {
		t.Fatalf("other user could see cards: %+v", list)
	}
	requestJSON(t, h, http.MethodGet, "/api/account/quota/cards", nil, memberCookies, &list)
	if len(list.Items) != 1 || list.Items[0].UserID != member.ID {
		t.Fatalf("own cards=%+v", list)
	}
	creditID := list.Items[0].ID
	requestJSONExpectStatus(t, h, http.MethodPut, "/api/quota/cards/"+strconv.Itoa(creditID), issue, memberCookies, http.StatusForbidden)
	requestJSON(t, h, http.MethodPut, "/api/quota/cards/"+strconv.Itoa(creditID), map[string]any{"name": "Edited", "amount_usd": 12}, admin, nil)
	requestJSON(t, h, http.MethodPost, "/api/quota/cards/issue", map[string]any{"user_ids": []int{member.ID}, "kind": "reset", "count": 1}, admin, nil)
	requestJSON(t, h, http.MethodGet, "/api/account/quota/cards?kind=reset", nil, memberCookies, &list)
	if len(list.Items) != 1 || list.Items[0].Status != "unused" {
		t.Fatalf("reset list=%+v", list)
	}
	path := "/api/account/quota/cards/" + strconv.Itoa(list.Items[0].ID) + "/use"
	requestJSONExpectStatus(t, h, http.MethodPost, path, nil, otherCookies, http.StatusNotFound)
	requestJSON(t, h, http.MethodPost, path, nil, memberCookies, nil)
	requestJSONExpectStatus(t, h, http.MethodPost, path, nil, memberCookies, http.StatusConflict)
	requestJSON(t, h, http.MethodPost, "/api/quota/cards/revoke", map[string]any{"card_ids": []int{creditID}}, admin, nil)
	requestJSON(t, h, http.MethodGet, "/api/account/quota", nil, memberCookies, &status)
	if status["available_usd"] != float64(0) || status["paused"] != true {
		t.Fatalf("revoked balance=%v", status)
	}
	for filter, expected := range map[string]int{"active": 0, "used": 1, "revoked": 1, "expired": 0, "exhausted": 0} {
		list = cardList{}
		requestJSON(t, h, http.MethodGet, "/api/account/quota/cards?status="+filter, nil, memberCookies, &list)
		if list.Total != expected {
			t.Fatalf("status=%s count=%d, want %d", filter, list.Total, expected)
		}
		for _, card := range list.Items {
			if card.Status != filter || card.UserID != member.ID {
				t.Fatalf("filter returned wrong card: %+v", card)
			}
		}
	}
	requestJSONExpectStatus(t, h, http.MethodGet, "/api/account/quota/cards?status=invalid", nil, memberCookies, http.StatusUnprocessableEntity)
}

type quotaAPIStatusResponse struct {
	Unlimited      bool     `json:"unlimited"`
	WeeklyQuotaUSD *float64 `json:"weekly_quota_usd"`
	DailyQuotaUSD  *float64 `json:"daily_quota_usd"`
	CanCreateKeys  bool     `json:"can_create_keys"`
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
	if member.Quota.Unlimited || member.Quota.CanCreateKeys ||
		member.Quota.WeeklyQuotaUSD == nil || *member.Quota.WeeklyQuotaUSD != 0 ||
		member.Quota.DailyQuotaUSD == nil || *member.Quota.DailyQuotaUSD != 0 {
		t.Fatalf("new user quota = %#v, want two zero quotas", member.Quota)
	}

	weekly := 0.3
	daily := 0.1
	updated := quotaAPIStatusResponse{}
	requestJSON(t, handler, http.MethodPut, "/api/users/"+strconv.Itoa(member.ID)+"/quota", map[string]any{
		"weekly_quota_usd": weekly,
		"daily_quota_usd":  daily,
	}, adminCookies, &updated)
	if updated.Unlimited || updated.WeeklyQuotaUSD == nil || *updated.WeeklyQuotaUSD != weekly ||
		updated.DailyQuotaUSD == nil || *updated.DailyQuotaUSD != daily {
		t.Fatalf("updated quota = %#v, want both configured quotas", updated)
	}

	memberCookies := requestJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]any{
		"username": "member",
		"password": "member-password",
	}, nil, nil)

	accountQuota := quotaAPIStatusResponse{}
	requestJSON(t, handler, http.MethodGet, "/api/account/quota", nil, memberCookies, &accountQuota)
	if accountQuota.WeeklyQuotaUSD == nil || *accountQuota.WeeklyQuotaUSD != weekly ||
		accountQuota.DailyQuotaUSD == nil || *accountQuota.DailyQuotaUSD != daily {
		t.Fatalf("account quota = %#v, want both member quotas", accountQuota)
	}

	requestJSONExpectStatus(t, handler, http.MethodPut, "/api/users/"+strconv.Itoa(member.ID)+"/quota", map[string]any{"daily_quota_usd": nil, "weekly_quota_usd": nil}, memberCookies, http.StatusForbidden)
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
	requestJSON(t, handler, http.MethodPut, "/api/users/1/quota", map[string]any{"daily_quota_usd": zero, "weekly_quota_usd": zero}, cookies, nil)

	requestJSONExpectStatus(t, handler, http.MethodPost, "/api/api-keys", map[string]any{
		"description": "VSCode",
	}, cookies, http.StatusConflict)
}
