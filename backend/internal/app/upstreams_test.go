package app_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	backendApp "cpa-helper/backend/internal/app"
)

type upstreamAPICallRequest struct {
	AuthIndex string            `json:"auth_index"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Header    map[string]string `json:"header"`
}

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
	requestJSONExpectStatus(t, handler, http.MethodPost, "/api/upstreams/models", map[string]any{
		"section":  "openai-compatibility",
		"base_url": "https://models.example.test/v1",
	}, memberCookies, http.StatusForbidden)
	requestJSONExpectStatus(t, handler, http.MethodPost, "/api/upstreams/models", map[string]any{
		"section": "vertex-api-key",
	}, adminCookies, http.StatusUnprocessableEntity)
	requestJSONExpectStatus(t, handler, http.MethodPut, "/api/upstreams/config", []any{}, adminCookies, http.StatusNotFound)
	requestJSONExpectStatus(t, handler, http.MethodPut, "/api/upstreams/codex-api-key", []any{"invalid"}, adminCookies, http.StatusUnprocessableEntity)
}

func TestUpstreamManagementDiscoversAndNormalizesModels(t *testing.T) {
	openAICalls := 0
	geminiCalls := 0
	cpa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-management-key" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Method != http.MethodPost || r.URL.Path != "/v0/management/api-call" {
			http.NotFound(w, r)
			return
		}
		var payload upstreamAPICallRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode api-call payload: %v", err)
		}
		if payload.Method != http.MethodGet {
			t.Fatalf("api-call method = %q", payload.Method)
		}

		switch {
		case strings.HasPrefix(payload.URL, "https://models.example.test/"):
			openAICalls++
			if payload.URL != "https://models.example.test/v1/models" {
				t.Fatalf("OpenAI models URL = %q", payload.URL)
			}
			if openAICalls == 1 {
				if payload.Header["Authorization"] != "Bearer upstream-secret" || payload.Header["X-Provider"] != "test" {
					t.Fatalf("OpenAI headers = %#v", payload.Header)
				}
				writeTestJSON(t, w, map[string]any{
					"status_code": http.StatusUnauthorized,
					"body":        map[string]any{"error": map[string]any{"message": "key rejected"}},
				})
				return
			}
			if len(payload.Header) != 0 || payload.AuthIndex != "" {
				t.Fatalf("OpenAI fallback credentials = headers %#v, auth_index %q", payload.Header, payload.AuthIndex)
			}
			writeTestJSON(t, w, map[string]any{
				"status_code": http.StatusOK,
				"body": map[string]any{"data": []any{
					map[string]any{"id": "gpt-5"},
					map[string]any{"id": "gpt-4.1", "display_name": "GPT 4.1"},
					map[string]any{"id": "gpt-5"},
				}},
			})
		case strings.HasPrefix(payload.URL, "https://generativelanguage.googleapis.com/"):
			geminiCalls++
			if payload.Header["x-goog-api-key"] != "gemini-secret" {
				t.Fatalf("Gemini headers = %#v", payload.Header)
			}
			if geminiCalls == 1 {
				if payload.URL != "https://generativelanguage.googleapis.com/v1beta/models" {
					t.Fatalf("first Gemini models URL = %q", payload.URL)
				}
				writeTestJSON(t, w, map[string]any{
					"status_code": http.StatusOK,
					"body": map[string]any{
						"models":        []any{map[string]any{"name": "models/gemini-2.5-pro", "displayName": "Gemini 2.5 Pro"}},
						"nextPageToken": "page-2",
					},
				})
				return
			}
			if !strings.Contains(payload.URL, "pageToken=page-2") {
				t.Fatalf("second Gemini models URL = %q", payload.URL)
			}
			writeTestJSON(t, w, map[string]any{
				"status_code": http.StatusOK,
				"body": map[string]any{
					"models": []any{map[string]any{"name": "models/gemini-2.5-flash"}},
				},
			})
		default:
			t.Fatalf("unexpected api-call target %q", payload.URL)
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

	var openAIResult struct {
		Models []struct {
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
		} `json:"models"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/upstreams/models", map[string]any{
		"section":  "openai-compatibility",
		"base_url": "https://models.example.test/v1",
		"api_key":  "upstream-secret",
		"headers":  map[string]string{"X-Provider": "test"},
	}, adminCookies, &openAIResult)
	if openAICalls != 2 {
		t.Fatalf("OpenAI api-call count = %d, want 2", openAICalls)
	}
	if len(openAIResult.Models) != 2 || openAIResult.Models[0].Name != "gpt-4.1" || openAIResult.Models[0].DisplayName != "GPT 4.1" || openAIResult.Models[1].Name != "gpt-5" {
		t.Fatalf("OpenAI models = %#v", openAIResult.Models)
	}

	var geminiResult struct {
		Models []struct {
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
		} `json:"models"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/upstreams/models", map[string]any{
		"section": "gemini-api-key",
		"api_key": "gemini-secret",
	}, adminCookies, &geminiResult)
	if geminiCalls != 2 {
		t.Fatalf("Gemini api-call count = %d, want 2", geminiCalls)
	}
	if len(geminiResult.Models) != 2 || geminiResult.Models[0].Name != "gemini-2.5-flash" || geminiResult.Models[1].Name != "gemini-2.5-pro" || geminiResult.Models[1].DisplayName != "Gemini 2.5 Pro" {
		t.Fatalf("Gemini models = %#v", geminiResult.Models)
	}
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
