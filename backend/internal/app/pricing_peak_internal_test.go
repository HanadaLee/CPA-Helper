package app

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestPricingCalendarBeijingPeakBoundariesAndHolidays(t *testing.T) {
	calendar := &pricingCalendar{years: map[int]map[string]bool{
		2026: {"2026-09-25": true},
	}}
	cases := []struct {
		name string
		at   time.Time
		peak bool
	}{
		{"before morning", time.Date(2026, 9, 24, 8, 59, 59, 0, appTimeLocation), false},
		{"morning starts", time.Date(2026, 9, 24, 9, 0, 0, 0, appTimeLocation), true},
		{"noon starts", time.Date(2026, 9, 24, 12, 0, 0, 0, appTimeLocation), false},
		{"afternoon starts", time.Date(2026, 9, 24, 14, 0, 0, 0, appTimeLocation), true},
		{"evening starts", time.Date(2026, 9, 24, 18, 0, 0, 0, appTimeLocation), false},
		{"official holiday", time.Date(2026, 9, 25, 10, 0, 0, 0, appTimeLocation), false},
		{"weekend", time.Date(2026, 9, 26, 10, 0, 0, 0, appTimeLocation), false},
		{"unknown year", time.Date(2027, 9, 24, 10, 0, 0, 0, appTimeLocation), false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := calendar.isPeak(test.at.UTC()); got != test.peak {
				t.Fatalf("isPeak(%s) = %v, want %v", test.at, got, test.peak)
			}
		})
	}
}

func TestRecordCostPeakOffPeakLongContextAndFast(t *testing.T) {
	model := "gpt-peak-test"
	provider := "openai"
	fast := "fast"
	price := ModelPrice{
		Provider: provider, Model: model,
		InputUSDPerMillion: 2, LongContextEnabled: true, LongContextThresholdTokens: 100,
		LongContextInputUSDPerMillion: 6, OffPeakEnabled: true,
		OffPeakInputUSDPerMillion: 1, LongContextOffPeakInputUSDPerMillion: 3,
		FastMultiplier: 2,
		calendar:       &pricingCalendar{years: map[int]map[string]bool{2026: {"2026-09-25": true}}},
	}
	prices := pricesByKey([]ModelPrice{price})
	cases := []struct {
		name  string
		at    time.Time
		input int
		want  float64
	}{
		{"short peak", time.Date(2026, 9, 24, 10, 0, 0, 0, appTimeLocation), 100, .0004},
		{"long peak", time.Date(2026, 9, 24, 10, 0, 0, 0, appTimeLocation), 101, .001212},
		{"short offpeak", time.Date(2026, 9, 24, 12, 0, 0, 0, appTimeLocation), 100, .0002},
		{"long offpeak", time.Date(2026, 9, 25, 10, 0, 0, 0, appTimeLocation), 101, .000606},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			amount, unpriced := calculateRecordCost(UsageRecord{Provider: &provider, Model: &model, Timestamp: test.at, InputTokens: test.input, RequestServiceTier: &fast}, prices)
			if unpriced || amount != test.want {
				t.Fatalf("cost = %v unpriced=%v, want %v false", amount, unpriced, test.want)
			}
		})
	}
}

func TestRequestPriceUsesOffPeakAndHistoricalCostStaysFixed(t *testing.T) {
	model := "gpt-image-peak-test"
	provider := "openai"
	peakUSD := 2.0
	offPeakUSD := 0.5
	price := ModelPrice{
		Provider: provider, Model: model, RequestUSD: &peakUSD,
		OffPeakEnabled: true, OffPeakRequestUSD: &offPeakUSD,
		calendar: &pricingCalendar{years: map[int]map[string]bool{2026: {}}},
	}
	prices := pricesByKey([]ModelPrice{price})
	record := UsageRecord{Model: &model, Timestamp: time.Date(2026, 9, 24, 12, 0, 0, 0, appTimeLocation)}
	amount, unpriced := calculateRecordCost(record, prices)
	if unpriced || amount != .5 {
		t.Fatalf("off-peak request price = %v unpriced=%v", amount, unpriced)
	}
	record.CostStored = true
	record.CostUSD = amount
	offPeakUSD = 9
	amount, unpriced = recordCost(record, prices)
	if unpriced || amount != .5 {
		t.Fatalf("stored historical price = %v unpriced=%v", amount, unpriced)
	}
}

func TestModelPriceOffPeakRoundTripAndHolidayCalendarAPI(t *testing.T) {
	t.Setenv("CPA_HELPER_DATA_DIR", t.TempDir())
	app, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	handler := app.Routes()
	cookies := requestJSONForPricingTest(t, handler, http.MethodPost, "/api/auth/setup", map[string]any{"username": "admin", "password": "test-password", "nickname": "Admin"}, nil, nil)
	var created ModelPrice
	requestJSONForPricingTest(t, handler, http.MethodPost, "/api/model-prices", map[string]any{
		"provider": "openai", "model": "gpt-peak-api", "input_usd_per_million": 2,
		"off_peak_enabled": true, "off_peak_input_usd_per_million": 1,
		"long_context_enabled": true, "long_context_threshold_tokens": 100,
		"long_context_input_usd_per_million": 6, "long_context_off_peak_input_usd_per_million": 3,
	}, cookies, &created)
	if !created.OffPeakEnabled || created.OffPeakInputUSDPerMillion != 1 || created.LongContextOffPeakInputUSDPerMillion != 3 {
		t.Fatalf("created off-peak price = %#v", created)
	}
	prices, err := app.loadPriceMap(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if amount, unpriced := calculateRecordCost(UsageRecord{Model: &created.Model, Timestamp: time.Date(2026, 9, 25, 10, 0, 0, 0, appTimeLocation), InputTokens: 101}, prices); unpriced || amount != .000303 {
		t.Fatalf("holiday long-context cost = %v unpriced=%v", amount, unpriced)
	}
	var calendar pricingHolidayCalendarResponse
	requestJSONForPricingTest(t, handler, http.MethodGet, "/api/model-prices/holiday-calendar?year=2026", nil, cookies, &calendar)
	if !calendar.Configured || len(calendar.Dates) != 33 || calendar.SourceURL == "" {
		t.Fatalf("seeded 2026 calendar = %#v", calendar)
	}
	requestJSONForPricingTest(t, handler, http.MethodPut, "/api/model-prices/holiday-calendar", map[string]any{"year": 2027, "dates": []string{"2027-01-01"}}, cookies, &calendar)
	if !calendar.Configured || len(calendar.Dates) != 1 || calendar.Dates[0] != "2027-01-01" {
		t.Fatalf("saved 2027 calendar = %#v", calendar)
	}
	prices, err = app.loadPriceMap(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if amount, unpriced := calculateRecordCost(UsageRecord{Model: &created.Model, Timestamp: time.Date(2027, 1, 4, 10, 0, 0, 0, appTimeLocation), InputTokens: 100}, prices); unpriced || amount != .0002 {
		t.Fatalf("configured 2027 weekday peak cost = %v unpriced=%v", amount, unpriced)
	}
	requestJSONForPricingTest(t, handler, http.MethodPut, fmt.Sprintf("/api/model-prices/%d", created.ID), map[string]any{
		"provider": "openai", "model": "gpt-peak-api", "input_usd_per_million": 2,
		"off_peak_enabled": true, "off_peak_input_usd_per_million": .5,
		"long_context_enabled": true, "long_context_threshold_tokens": 100,
		"long_context_input_usd_per_million": 6, "long_context_off_peak_input_usd_per_million": 3,
	}, cookies, &created)
	if created.OffPeakInputUSDPerMillion != .5 {
		t.Fatalf("updated off-peak input = %v", created.OffPeakInputUSDPerMillion)
	}
}
