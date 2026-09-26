package migrations

import (
	"database/sql"
	"strings"
	"testing"
)

func TestJointQuotaMigrationPreservesLegacyCharges(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE user_quota_charges(id INTEGER PRIMARY KEY, daily_deducted_usd REAL, weekly_deducted_usd REAL, amount_usd REAL);
        INSERT INTO user_quota_charges VALUES(1, 1, 2, 4);`); err != nil {
		t.Fatal(err)
	}
	script, err := FS.ReadFile("202609260002_joint_quota_limits.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(strings.Split(string(script), "-- +goose Down")[0]); err != nil {
		t.Fatal(err)
	}
	var daily, weekly, limit, amount float64
	if err := db.QueryRow("SELECT daily_deducted_usd, weekly_deducted_usd, limit_deducted_usd, amount_usd FROM user_quota_charges").Scan(&daily, &weekly, &limit, &amount); err != nil {
		t.Fatal(err)
	}
	if daily != 1 || weekly != 2 || limit != 3 || amount != 4 {
		t.Fatalf("historical charge altered: %v/%v/%v/%v", daily, weekly, limit, amount)
	}
}
