package app_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	backendApp "cpa-helper/backend/internal/app"
)

func TestCPAConfigProxiesCompleteYAML(t *testing.T) {
	configYAML := "host: 127.0.0.1\nport: 8317\ndebug: false\n"
	putCount := 0
	cpa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-management-key" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/v0/management/config.yaml" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = io.WriteString(w, configYAML)
		case http.MethodPut:
			if r.Header.Get("Content-Type") != "application/yaml" {
				t.Fatalf("Content-Type = %q", r.Header.Get("Content-Type"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read proxied config: %v", err)
			}
			configYAML = string(body)
			putCount++
			writeTestJSON(t, w, map[string]string{"status": "ok"})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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

	var response struct {
		Content string `json:"content"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/settings/cpa-config", nil, adminCookies, &response)
	if response.Content != configYAML {
		t.Fatalf("GET content = %q, want %q", response.Content, configYAML)
	}

	want := "# keep comments\nhost: 0.0.0.0\nport: 9000\npayload:\n  default: []\n"
	requestJSON(t, handler, http.MethodPut, "/api/settings/cpa-config", map[string]any{
		"content": want,
	}, adminCookies, &response)
	if putCount != 1 {
		t.Fatalf("PUT count = %d, want 1", putCount)
	}
	if response.Content != want || configYAML != want {
		t.Fatalf("saved content = %q, response = %q", configYAML, response.Content)
	}
}

func TestCPAConfigRequiresAdminAndRejectsEmptyContent(t *testing.T) {
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

	requestJSONExpectStatus(t, handler, http.MethodGet, "/api/settings/cpa-config", nil, memberCookies, http.StatusForbidden)
	requestJSONExpectStatus(t, handler, http.MethodPut, "/api/settings/cpa-config", map[string]any{
		"content": " \n ",
	}, adminCookies, http.StatusUnprocessableEntity)
	requestJSONExpectStatus(t, handler, http.MethodPost, "/api/settings/cpa-config", nil, adminCookies, http.StatusMethodNotAllowed)
}
