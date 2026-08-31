package app_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	backendApp "cpa-helper/backend/internal/app"
)

func TestCASLoginCallbackAutoCreatesUserAndLogsOut(t *testing.T) {
	var mu sync.Mutex
	var validationService string
	var validationHost string
	casServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cas/app/serviceValidate":
			mu.Lock()
			validationService = r.URL.Query().Get("service")
			validationHost = r.Host
			mu.Unlock()
			w.Header().Set("Content-Type", "application/xml")
			if r.URL.Query().Get("ticket") == "ST-expired" {
				fmt.Fprint(w, `<cas:serviceResponse xmlns:cas="http://www.yale.edu/tp/cas"><cas:authenticationFailure code="INVALID_TICKET">expired</cas:authenticationFailure></cas:serviceResponse>`)
				return
			}
			if r.URL.Query().Get("ticket") == "ST-admin" {
				fmt.Fprint(w, `<cas:serviceResponse xmlns:cas="http://www.yale.edu/tp/cas"><cas:authenticationSuccess><cas:user>admin</cas:user><cas:attributes><cas:displayName>CAS Admin</cas:displayName></cas:attributes></cas:authenticationSuccess></cas:serviceResponse>`)
				return
			}
			fmt.Fprint(w, `
<cas:serviceResponse xmlns:cas="http://www.yale.edu/tp/cas">
  <cas:authenticationSuccess>
    <cas:user>cas-member</cas:user>
    <cas:attributes>
      <cas:email>cas-member@example.test</cas:email>
      <cas:displayName>CAS Member</cas:displayName>
      <cas:avatar>https://cdn.example.test/cas-member.png</cas:avatar>
    </cas:attributes>
  </cas:authenticationSuccess>
</cas:serviceResponse>`)
		case "/cas/app/login", "/cas/app/logout":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer casServer.Close()

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

	var settings struct {
		CASEnabled         bool `json:"cas_enabled"`
		CASDefaultLogin    bool `json:"cas_default_login"`
		CASAutoCreateUsers bool `json:"cas_auto_create_users"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/settings", nil, adminCookies, &settings)
	if settings.CASEnabled || settings.CASDefaultLogin || !settings.CASAutoCreateUsers {
		t.Fatalf("unexpected CAS defaults: %+v", settings)
	}

	requestJSON(t, handler, http.MethodPut, "/api/settings", map[string]any{
		"cas_enabled":           true,
		"cas_default_login":     true,
		"cas_base_url":          casServer.URL + "/cas/app/",
		"cas_validation_url":    casServer.URL + "/cas/app/",
		"cas_validation_host":   "cas.internal.test",
		"cas_public_url":        "https://helper.example.test/",
		"cas_auto_create_users": true,
	}, adminCookies, &settings)
	if !settings.CASEnabled || !settings.CASDefaultLogin {
		t.Fatal("CAS and default CAS login should be enabled after saving settings")
	}
	var setupState struct {
		CASEnabled      bool `json:"cas_enabled"`
		CASDefaultLogin bool `json:"cas_default_login"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/auth/setup", nil, nil, &setupState)
	if !setupState.CASEnabled || !setupState.CASDefaultLogin {
		t.Fatalf("unexpected public CAS setup state: %+v", setupState)
	}

	loginRequest := httptest.NewRequest(http.MethodGet, "/cas/login?returnTo=%2Faccount%2Fusage", nil)
	loginRecorder := httptest.NewRecorder()
	handler.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusFound {
		t.Fatalf("CAS login returned %d: %s", loginRecorder.Code, loginRecorder.Body.String())
	}
	loginURL, err := url.Parse(loginRecorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse CAS login redirect: %v", err)
	}
	if loginURL.Path != "/cas/app/login" {
		t.Fatalf("CAS login path = %q", loginURL.Path)
	}
	wantService := "https://helper.example.test/cas/callback?returnTo=%2Faccount%2Fusage"
	if got := loginURL.Query().Get("service"); got != wantService {
		t.Fatalf("CAS login service = %q, want %q", got, wantService)
	}

	callbackRequest := httptest.NewRequest(http.MethodGet, "/cas/callback?ticket=ST-valid&returnTo=%2Faccount%2Fusage", nil)
	callbackRecorder := httptest.NewRecorder()
	handler.ServeHTTP(callbackRecorder, callbackRequest)
	if callbackRecorder.Code != http.StatusFound {
		t.Fatalf("CAS callback returned %d: %s", callbackRecorder.Code, callbackRecorder.Body.String())
	}
	if got := callbackRecorder.Header().Get("Location"); got != "/account/usage" {
		t.Fatalf("CAS callback redirect = %q", got)
	}
	casCookies := callbackRecorder.Result().Cookies()
	if len(casCookies) == 0 || casCookies[0].Name != "cpa_helper_session" {
		t.Fatalf("CAS callback cookies = %+v", casCookies)
	}

	mu.Lock()
	gotService, gotHost := validationService, validationHost
	mu.Unlock()
	if gotService != wantService {
		t.Fatalf("CAS validation service = %q, want %q", gotService, wantService)
	}
	if gotHost != "cas.internal.test" {
		t.Fatalf("CAS validation Host = %q", gotHost)
	}

	var me struct {
		Username          string `json:"username"`
		Nickname          string `json:"nickname"`
		Email             string `json:"email"`
		Avatar            string `json:"avatar"`
		IsAdmin           bool   `json:"is_admin"`
		CASBound          bool   `json:"cas_bound"`
		CanChangePassword bool   `json:"can_change_password"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/auth/me", nil, casCookies, &me)
	if me.Username != "cas-member" || me.Nickname != "CAS Member" || me.IsAdmin {
		t.Fatalf("CAS user = %+v", me)
	}
	if me.Email != "cas-member@example.test" || me.Avatar != "https://cdn.example.test/cas-member.png" || !me.CASBound || me.CanChangePassword {
		t.Fatalf("CAS profile was not synchronized: %+v", me)
	}
	requestJSONExpectStatus(t, handler, http.MethodPost, "/api/auth/change-credentials", map[string]any{
		"current_password": "test-password",
		"password":         "updated-password",
	}, casCookies, http.StatusForbidden)
	adminCallbackRequest := httptest.NewRequest(http.MethodGet, "/cas/callback?ticket=ST-admin&returnTo=%2F", nil)
	adminCallbackRecorder := httptest.NewRecorder()
	handler.ServeHTTP(adminCallbackRecorder, adminCallbackRequest)
	if adminCallbackRecorder.Code != http.StatusFound {
		t.Fatalf("existing admin CAS callback returned %d: %s", adminCallbackRecorder.Code, adminCallbackRecorder.Body.String())
	}
	var casAdmin struct {
		Username string `json:"username"`
		IsAdmin  bool   `json:"is_admin"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/auth/me", nil, adminCallbackRecorder.Result().Cookies(), &casAdmin)
	if casAdmin.Username != "admin" || !casAdmin.IsAdmin {
		t.Fatalf("existing CAS admin role was not preserved: %+v", casAdmin)
	}

	var users []struct {
		Username string `json:"username"`
	}
	requestJSON(t, handler, http.MethodGet, "/api/users", nil, adminCookies, &users)
	count := 0
	for _, user := range users {
		if user.Username == "cas-member" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("CAS user count = %d, users = %+v", count, users)
	}

	expiredRequest := httptest.NewRequest(http.MethodGet, "/cas/callback?ticket=ST-expired&returnTo=%2Faccount%2Fusage", nil)
	expiredRecorder := httptest.NewRecorder()
	handler.ServeHTTP(expiredRecorder, expiredRequest)
	if expiredRecorder.Code != http.StatusFound || expiredRecorder.Header().Get("Location") != "/cas/login?returnTo=%2Faccount%2Fusage" {
		t.Fatalf("expired CAS callback redirect = status %d, location %q", expiredRecorder.Code, expiredRecorder.Header().Get("Location"))
	}

	var logoutResponse struct {
		OK           bool   `json:"ok"`
		CASLogoutURL string `json:"cas_logout_url"`
	}
	requestJSON(t, handler, http.MethodPost, "/api/auth/logout", nil, casCookies, &logoutResponse)
	if !logoutResponse.OK || logoutResponse.CASLogoutURL != "/cas/logout" {
		t.Fatalf("logout response = %+v", logoutResponse)
	}

	logoutRequest := httptest.NewRequest(http.MethodGet, "/cas/logout", nil)
	logoutRecorder := httptest.NewRecorder()
	handler.ServeHTTP(logoutRecorder, logoutRequest)
	if logoutRecorder.Code != http.StatusFound {
		t.Fatalf("CAS logout returned %d", logoutRecorder.Code)
	}
	logoutURL, err := url.Parse(logoutRecorder.Header().Get("Location"))
	if err != nil || logoutURL.Path != "/cas/app/logout" || logoutURL.Query().Get("service") != "https://helper.example.test" {
		t.Fatalf("CAS logout redirect = %q, error = %v", logoutRecorder.Header().Get("Location"), err)
	}
}

func TestCASRejectsUnsafeReturnTargetAndRequiresConfiguration(t *testing.T) {
	t.Setenv("CPA_HELPER_DATA_DIR", t.TempDir())
	app, err := backendApp.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer app.Close()
	handler := app.Routes()

	requestJSONExpectStatus(t, handler, http.MethodGet, "/cas/login", nil, nil, http.StatusServiceUnavailable)
	adminCookies := requestJSON(t, handler, http.MethodPost, "/api/auth/setup", map[string]any{
		"username": "admin",
		"password": "test-password",
		"nickname": "Admin",
	}, nil, nil)
	requestJSONExpectStatus(t, handler, http.MethodPut, "/api/settings", map[string]any{
		"cas_enabled": true,
	}, adminCookies, http.StatusUnprocessableEntity)

	requestJSON(t, handler, http.MethodPut, "/api/settings", map[string]any{
		"cas_enabled":           true,
		"cas_base_url":          "https://cas.example.test/cas/app",
		"cas_public_url":        "https://helper.example.test",
		"cas_auto_create_users": false,
	}, adminCookies, nil)
	loginRequest := httptest.NewRequest(http.MethodGet, "/cas/login?returnTo="+url.QueryEscape("//evil.example/steal"), nil)
	loginRecorder := httptest.NewRecorder()
	handler.ServeHTTP(loginRecorder, loginRequest)
	location := loginRecorder.Header().Get("Location")
	redirectURL, parseErr := url.Parse(location)
	serviceURL, serviceParseErr := url.Parse(redirectURL.Query().Get("service"))
	if loginRecorder.Code != http.StatusFound || parseErr != nil || serviceParseErr != nil || serviceURL.Query().Get("returnTo") != "/" {
		t.Fatalf("unsafe return target was not replaced: status=%d location=%q", loginRecorder.Code, location)
	}
}
