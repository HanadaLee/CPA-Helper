package app_test

import (
	"net/http"
	"testing"

	backendApp "cpa-helper/backend/internal/app"
)

func TestSettingsPersistCPAMCURL(t *testing.T) {
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

	var settings struct {
		CPAMCURL string `json:"cpamc_url"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/settings", nil, cookies, &settings)
	if settings.CPAMCURL != "/management.html" {
		t.Fatalf("default cpamc_url = %q, want /management.html", settings.CPAMCURL)
	}

	requestJSON(t, handler, http.MethodPut, "/api/settings", map[string]any{
		"cpamc_url": " https://cpamc.example.test/panel?theme=dark ",
	}, cookies, &settings)
	if settings.CPAMCURL != "https://cpamc.example.test/panel?theme=dark" {
		t.Fatalf("updated cpamc_url = %q", settings.CPAMCURL)
	}

	requestJSON(t, handler, http.MethodGet, "/api/settings", nil, cookies, &settings)
	if settings.CPAMCURL != "https://cpamc.example.test/panel?theme=dark" {
		t.Fatalf("persisted cpamc_url = %q", settings.CPAMCURL)
	}

	requestJSONExpectStatus(t, handler, http.MethodPut, "/api/settings", map[string]any{
		"cpamc_url": "  ",
	}, cookies, http.StatusUnprocessableEntity)
}

func TestSettingsRequireAtLeastThirtyOneRetentionDays(t *testing.T) {
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

	var settings struct {
		UsageDetailRetentionDays int `json:"usage_detail_retention_days"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/settings", nil, cookies, &settings)
	if settings.UsageDetailRetentionDays != 90 {
		t.Fatalf("default usage detail retention = %d, want 90", settings.UsageDetailRetentionDays)
	}
	requestJSONExpectStatus(t, handler, http.MethodPut, "/api/settings", map[string]any{
		"usage_detail_retention_days": 30,
	}, cookies, http.StatusUnprocessableEntity)
	requestJSON(t, handler, http.MethodPut, "/api/settings", map[string]any{
		"usage_detail_retention_days": 31,
	}, cookies, &settings)
	if settings.UsageDetailRetentionDays != 31 {
		t.Fatalf("updated usage detail retention = %d, want 31", settings.UsageDetailRetentionDays)
	}
}
