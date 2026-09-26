package app

import (
	"context"
	"fmt"
	"testing"
)

func TestQuotaLimitsAreJointAndCardsRemainUsable(t *testing.T) {
	for _, pair := range [][2]float64{{1, 5}, {5, 1}, {0, 5}, {5, 0}} {
		t.Run(fmt.Sprintf("daily_%g_weekly_%g", pair[0], pair[1]), func(t *testing.T) {
			a := newQuotaTestApp(t)
			ctx := context.Background()
			id := seedQuotaTestUser(t, a, "joint-user")
			if _, err := a.updateUserQuota(ctx, id, userQuotaPayload{DailyQuotaUSD: &pair[0], WeeklyQuotaUSD: &pair[1]}); err != nil {
				t.Fatal(err)
			}
			seedQuotaTestAPIKey(t, a, id, "sk-joint")
			seedQuotaTestPrice(t, a, "openai", "joint", 1)
			limit := minQuotaAmount(pair[0], pair[1])
			before, err := a.userQuotaStatus(ctx, id)
			if err != nil || before.AvailableUSD != limit || before.LimitsRemainingUSD != limit {
				t.Fatalf("limits summed instead of intersected: %+v %v", before, err)
			}
			if _, _, err := a.saveUsageMessage(ctx, []byte(`{"api_key":"sk-joint","model":"joint","input_tokens":1200000,"request_id":"first"}`)); err != nil {
				t.Fatal(err)
			}
			status, err := a.userQuotaStatus(ctx, id)
			if err != nil || !status.Paused || status.DailyUsedUSD != limit || status.WeeklyUsedUSD != limit || status.CanCreateKeys {
				t.Fatalf("exhausted limit was bypassed: %+v %v", status, err)
			}
			if pair[1] == 1 {
				if _, err := a.db.Exec("UPDATE users SET quota_day = '2000-01-01' WHERE id = ?", id); err != nil {
					t.Fatal(err)
				}
				status, err = a.userQuotaStatus(ctx, id)
				if err != nil || !status.Paused || status.DailyUsedUSD != 0 || status.WeeklyUsedUSD != 1 {
					t.Fatalf("new day bypassed weekly limit: %+v %v", status, err)
				}
			}
			previousDay, previousWeek := status.DailyUsedUSD, status.WeeklyUsedUSD
			seedQuotaCard(t, a, id, "credit", 2, nil)
			// No keys/config in this isolated test: test availability independent of remote restoration.
			if _, err := a.db.Exec("DELETE FROM user_api_keys WHERE user_id = ?", id); err != nil {
				t.Fatal(err)
			}
			status, err = a.userQuotaStatus(ctx, id)
			if err != nil || status.Paused || status.AvailableUSD != 2 || status.CardsTotalUSD != 2 {
				t.Fatalf("card did not bypass exhausted limit: %+v %v", status, err)
			}
			seedQuotaTestAPIKey(t, a, id, "sk-joint")
			if _, _, err := a.saveUsageMessage(ctx, []byte(`{"api_key":"sk-joint","model":"joint","input_tokens":300000,"request_id":"card"}`)); err != nil {
				t.Fatal(err)
			}
			status, err = a.userQuotaStatus(ctx, id)
			if err != nil || status.DailyUsedUSD != previousDay || status.WeeklyUsedUSD != previousWeek || status.CardsRemainingUSD != 1.7 {
				t.Fatalf("card spend consumed limit counters: %+v %v", status, err)
			}
		})
	}
}
