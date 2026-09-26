package migrations

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestQuotaCardsMigrationPreservesBalancesAndExpiry(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "quota.sqlite3")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY, quota_monthly_usd REAL, quota_lifetime_usd REAL,
			quota_daily_usd REAL, quota_weekly_usd REAL, quota_month TEXT, quota_month_used_usd REAL);
		CREATE TABLE app_settings(id INTEGER PRIMARY KEY, new_user_quota_monthly_usd REAL, new_user_quota_lifetime_usd REAL);
		CREATE TABLE user_quota_charges(id INTEGER PRIMARY KEY);
		INSERT INTO users VALUES (1,100,50,1,5,strftime('%Y-%m','now','+8 hours'),25);
		INSERT INTO users VALUES (2,10,0,NULL,NULL,'2000-01',10);
		INSERT INTO users VALUES (3,NULL,NULL,NULL,NULL,'',0);
		INSERT INTO users VALUES (4,1,0,0,0,strftime('%Y-%m','now','+8 hours'),2);
		INSERT INTO app_settings VALUES (1,5,10);
	`)
	if err != nil {
		t.Fatal(err)
	}
	script, err := FS.ReadFile("202609260001_quota_cards.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(strings.Split(string(script), "-- +goose Down")[0]); err != nil {
		t.Fatal(err)
	}
	var monthly, lifetime float64
	var expires, lifetimeExpiry sql.NullString
	if err := db.QueryRow("SELECT amount_usd, expires_at FROM quota_cards WHERE user_id = 1 AND batch_id = 'migration-monthly'").Scan(&monthly, &expires); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT amount_usd, expires_at FROM quota_cards WHERE user_id = 1 AND batch_id = 'migration-lifetime'").Scan(&lifetime, &lifetimeExpiry); err != nil {
		t.Fatal(err)
	}
	if monthly != 75 || lifetime != 50 || !expires.Valid || lifetimeExpiry.Valid {
		t.Fatalf("migration balances=%v/%v expires=%v/%v", monthly, lifetime, expires, lifetimeExpiry)
	}
	end, err := time.Parse(time.RFC3339, expires.String)
	if err != nil {
		t.Fatal(err)
	}
	if end.Before(time.Now().AddDate(0, 0, 27)) || end.After(time.Now().AddDate(0, 0, 32)) {
		t.Fatalf("monthly card expiry=%v", end)
	}
	var nextMonth float64
	if err := db.QueryRow("SELECT amount_usd FROM quota_cards WHERE user_id = 2").Scan(&nextMonth); err != nil || nextMonth != 10 {
		t.Fatalf("stale month=%v %v", nextMonth, err)
	}
	var daily, weekly sql.NullFloat64
	if err := db.QueryRow("SELECT quota_daily_usd, quota_weekly_usd FROM users WHERE id = 2").Scan(&daily, &weekly); err != nil || !daily.Valid || !weekly.Valid || daily.Float64 != 0 || weekly.Float64 != 0 {
		t.Fatalf("limited user became unlimited: %v/%v %v", daily, weekly, err)
	}
	if err := db.QueryRow("SELECT quota_daily_usd, quota_weekly_usd FROM users WHERE id = 3").Scan(&daily, &weekly); err != nil || daily.Valid || weekly.Valid {
		t.Fatalf("unlimited user changed: %v/%v %v", daily, weekly, err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM quota_cards WHERE user_id = 4").Scan(&count); err != nil || count != 0 {
		t.Fatalf("exhausted monthly balance migrated: %d %v", count, err)
	}
	var monthEnd string
	if err := db.QueryRow("SELECT date('2026-01-31', '+1 month', 'floor')").Scan(&monthEnd); err != nil || monthEnd != "2026-02-28" {
		t.Fatalf("calendar month boundary=%s %v", monthEnd, err)
	}
	if err := db.QueryRow("SELECT datetime('2026-01-31 18:00:00', '+8 hours', '+1 month', 'floor', '-8 hours')").Scan(&monthEnd); err != nil || monthEnd != "2026-02-28 18:00:00" {
		t.Fatalf("Beijing calendar month boundary=%s %v", monthEnd, err)
	}
	if _, err := db.Exec(strings.Split(string(script), "-- +goose Down")[1]); err == nil || !strings.Contains(err.Error(), "restore_pre_upgrade_backup_required") {
		t.Fatalf("unsafe downgrade must be rejected: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM quota_cards").Scan(&count); err != nil || count != 3 {
		t.Fatalf("downgrade lost balances: count=%d err=%v", count, err)
	}
}
