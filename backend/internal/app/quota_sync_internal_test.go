package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestQuotaKeyMaintenanceDoesNotOverwriteConcurrentKeyChanges(t *testing.T) {
	a := newQuotaTestApp(t)
	ctx := context.Background()
	var mu sync.Mutex
	keys := []string{"sk-untouched"}
	for i := 0; i < 10; i++ {
		keys = append(keys, fmt.Sprintf("sk-old-%d", i))
	}
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Method == http.MethodPut {
			if err := json.NewDecoder(r.Body).Decode(&keys); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"api-keys": keys})
	}))
	defer remote.Close()
	cfg, err := a.loadConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Collector.CLIProxyURL = remote.URL
	cfg.Collector.ManagementKey = "test-management"
	if err := a.saveConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			errs <- a.addRemoteAPIKey(ctx, fmt.Sprintf("sk-new-%d", i))
		}(i)
		go func(i int) {
			defer wg.Done()
			errs <- a.removeRemoteAPIKeyHash(ctx, hashAPIKey(fmt.Sprintf("sk-old-%d", i)))
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(keys) != 11 {
		t.Fatalf("lost remote key changes: %v", keys)
	}
	want := map[string]bool{"sk-untouched": true}
	for i := 0; i < 10; i++ {
		want[fmt.Sprintf("sk-new-%d", i)] = true
	}
	for _, key := range keys {
		if !want[key] {
			t.Fatalf("unexpected key %s", key)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing keys: %v", want)
	}
}

func TestQuotaExpiredCardRetriesPauseAndResetRestoresOnlyEnabledKeys(t *testing.T) {
	a := newQuotaTestApp(t)
	ctx := context.Background()
	id := seedQuotaTestUser(t, a, "sync-user")
	daily, weekly := 1.0, 2.0
	if _, err := a.updateUserQuota(ctx, id, userQuotaPayload{DailyQuotaUSD: &daily, WeeklyQuotaUSD: &weekly}); err != nil {
		t.Fatal(err)
	}
	seedQuotaTestAPIKey(t, a, id, "sk-active")
	seedQuotaTestAPIKey(t, a, id, "sk-disabled")
	if _, err := a.db.Exec("UPDATE user_api_keys SET disabled = 1 WHERE api_key = 'sk-disabled'"); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	keys := []string{"sk-active"}
	failRemove := true
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path != "/v0/management/api-keys" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodPut:
			if failRemove {
				http.Error(w, "unavailable", 503)
				return
			}
			if err := json.NewDecoder(r.Body).Decode(&keys); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
		case http.MethodPatch:
			var p struct {
				New string `json:"new"`
			}
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			keys = append(keys, p.New)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"api-keys": keys})
	}))
	defer remote.Close()
	cfg, err := a.loadConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Collector.CLIProxyURL = remote.URL
	cfg.Collector.ManagementKey = "test-management"
	if err := a.saveConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	card := seedQuotaCard(t, a, id, "credit", 10, nil)
	if _, err := a.db.Exec("UPDATE users SET quota_day_used_usd = 1, quota_week_used_usd = 2 WHERE id = ?", id); err != nil {
		t.Fatal(err)
	}
	if _, err := a.db.Exec("UPDATE quota_cards SET expires_at = ? WHERE id = ?", dbTime(time.Now().Add(-time.Minute)), card); err != nil {
		t.Fatal(err)
	}
	if err := a.reconcileQuotaUsers(ctx); err != nil {
		t.Fatal(err)
	}
	user, err := a.getUser(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if user.QuotaPausedAt == nil || user.QuotaSyncError == nil {
		t.Fatalf("failed pause not recorded: %+v", user)
	}
	mu.Lock()
	failRemove = false
	mu.Unlock()
	if err := a.reconcileQuotaUsers(ctx); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	remaining := len(keys)
	mu.Unlock()
	if remaining != 0 {
		t.Fatal("failed pause was not retried")
	}
	// A natural daily rollover must restore the enabled key even without a page visit.
	if _, err := a.db.Exec("UPDATE users SET quota_day = '2000-01-01' WHERE id = ?", id); err != nil {
		t.Fatal(err)
	}
	if err := a.reconcileQuotaUsers(ctx); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(keys) != 1 || keys[0] != "sk-active" {
		t.Fatalf("restored wrong keys: %v", keys)
	}
}
