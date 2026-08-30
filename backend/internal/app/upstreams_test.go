package app_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	backendApp "cpa-helper/backend/internal/app"
)

func TestUpstreamManagementProxiesProviderConfiguration(t *testing.T) {
	var receivedCodex []map[string]any
	cpa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-management-key" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v0/management/config":
			writeTestJSON(t, w, map[string]any{
				"gemini-api-key": []any{
					map[string]any{"api-key": "gemini-secret", "base-url": "https://gemini.example.test"},
				},
				"codex-api-key": []any{
					map[string]any{"api-key": "codex-secret", "priority": 3},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v0/management/gemini-api-key":
			writeTestJSON(t, w, map[string]any{
				"gemini-api-key": []any{
					map[string]any{"api-key": "gemini-secret", "prefix": "google"},
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/v0/management/codex-api-key":
			if err := json.NewDecoder(r.Body).Decode(&receivedCodex); err != nil {
				t.Fatalf("decode proxied codex config: %v", err)
			}
			writeTestJSON(t, w, map[string]string{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cpa.Close()

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
	requestJSON(t, handler, http.MethodPut, "/api/settings", map[string]any{
		"cliaproxy_url":  cpa.URL,
		"management_key": "test-management-key",
	}, adminCookies, nil)

	var list struct {
		Sections map[string][]map[string]any `json:"sections"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/upstreams", nil, adminCookies, &list)
	if got := list.Sections["gemini-api-key"][0]["api-key"]; got != "gemini-secret" {
		t.Fatalf("gemini api-key = %#v", got)
	}
	if got := len(list.Sections["openai-compatibility"]); got != 0 {
		t.Fatalf("openai-compatibility length = %d, want 0", got)
	}

	var section struct {
		Items []map[string]any `json:"items"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/upstreams/gemini-api-key", nil, adminCookies, &section)
	if got := section.Items[0]["prefix"]; got != "google" {
		t.Fatalf("gemini prefix = %#v", got)
	}

	requestJSON(t, handler, http.MethodPut, "/api/upstreams/codex-api-key", []any{
		map[string]any{"api-key": "new-codex-secret", "priority": 8},
	}, adminCookies, nil)
	if len(receivedCodex) != 1 || receivedCodex[0]["api-key"] != "new-codex-secret" {
		t.Fatalf("proxied codex config = %#v", receivedCodex)
	}
}

func TestUpstreamManagementRequiresAdminAndWhitelistsSections(t *testing.T) {
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
	requestJSON(t, handler, http.MethodPost, "/api/users", map[string]any{
		"username": "member",
		"password": "member-password",
		"nickname": "Member",
		"is_admin": false,
	}, adminCookies, nil)
	memberCookies := requestJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]any{
		"username": "member",
		"password": "member-password",
	}, nil, nil)

	requestJSONExpectStatus(t, handler, http.MethodGet, "/api/upstreams", nil, memberCookies, http.StatusForbidden)
	requestJSONExpectStatus(t, handler, http.MethodPut, "/api/upstreams/config", []any{}, adminCookies, http.StatusNotFound)
	requestJSONExpectStatus(t, handler, http.MethodPut, "/api/upstreams/codex-api-key", []any{"invalid"}, adminCookies, http.StatusUnprocessableEntity)
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
