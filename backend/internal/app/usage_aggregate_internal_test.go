package app

import (
	"reflect"
	"testing"
	"time"
)

func TestUsageAggregateBuilderMatchesLegacyResponses(t *testing.T) {
	start := time.Date(2026, 5, 16, 0, 0, 0, 0, appTimeLocation)
	end := start.Add(24 * time.Hour)
	openAI, claude := "openai", "claude"
	gpt, sonnet := "gpt-5.5", "claude-sonnet-4"
	chat, messages := "/v1/chat/completions", "/v1/messages"
	admin, member := "admin", "member"
	desktop, automation := "Desktop", "Automation"
	ttft := 500.0
	records := []UsageRecord{
		{
			Timestamp: start.Add(10 * time.Hour), UsageUsername: &admin, APIKeyDescription: &desktop,
			Provider: &openAI, Model: &gpt, Endpoint: &chat, InputTokens: 100, CachedTokens: 20,
			OutputTokens: 25, TotalTokens: 125, TTFTMS: &ttft,
		},
		{
			Timestamp: start.Add(11 * time.Hour), UsageUsername: &member, APIKeyDescription: &automation,
			Provider: &claude, Model: &sonnet, Endpoint: &messages, Failed: true, InputTokens: 40,
			CacheReadTokens: 10, CacheCreationTokens: 5, OutputTokens: 8, ReasoningTokens: 2, TotalTokens: 65,
		},
		{
			Timestamp: start.Add(12 * time.Hour), UsageUsername: &admin, APIKeyDescription: &desktop,
			Provider: &openAI, Model: &gpt, Endpoint: &chat, InputTokens: 4, OutputTokens: 1, TotalTokens: 5,
		},
		{
			Timestamp: start.Add(13 * time.Hour), UsageUsername: &admin, APIKeyDescription: &desktop,
			Provider: &openAI, Model: &gpt, Endpoint: &chat,
		},
	}
	prices := map[[2]string]ModelPrice{
		priceKey(openAI, gpt):    {Provider: openAI, Model: gpt, InputUSDPerMillion: 1, OutputUSDPerMillion: 2, CacheReadUSDPerMillion: 0.5},
		priceKey(claude, sonnet): {Provider: claude, Model: sonnet, InputUSDPerMillion: 3, OutputUSDPerMillion: 4, CacheReadUSDPerMillion: 0.3, CacheCreationUSDPerMillion: 3.75},
	}
	users := map[string]userInfo{
		admin:  {ID: 1, Username: admin, Name: "Admin"},
		member: {ID: 2, Username: member, Name: "Member"},
	}
	filters := UsageFilters{Start: &start, End: &end}
	builder := newUsageAggregateBuilder(filters, usageAggregateSummary|usageAggregateTrends|usageAggregateRankings|usageAggregateDistributions, users)
	for _, record := range records {
		cost, unpriced := recordCost(record, prices)
		builder.add(record, cost, unpriced)
	}

	assertUsageAggregateEqual(t, "summary", builder.summaryResponse(), usageSummaryFromRecords(filters, records, prices))
	assertUsageAggregateEqual(t, "trends", builder.trendResponse(), trendPointsFromRecords(filters, records, prices))
	assertUsageAggregateEqual(t, "api key ranking", usageRankingResponse("api_key_description", builder.apiKeys), rankingFromRecords(records, prices, "api_key_description", users))
	assertUsageAggregateEqual(t, "user ranking", usageRankingResponse("user", builder.usersByKey), rankingFromRecords(records, prices, "user", users))
	assertUsageAggregateEqual(t, "model ranking", usageRankingResponse("model", builder.models), rankingFromRecords(records, prices, "model", users))
	assertUsageAggregateEqual(t, "distributions", builder.distributionsResponse(), distributionsFromRecords(records, prices))
}

func assertUsageAggregateEqual(t *testing.T, name string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s mismatch\ngot:  %#v\nwant: %#v", name, got, want)
	}
}
