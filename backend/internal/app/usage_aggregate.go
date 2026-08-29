package app

import (
	"context"
	"database/sql"
	"sort"
	"strconv"
	"strings"
	"time"
)

type usageAggregateParts uint8

const (
	usageAggregateSummary usageAggregateParts = 1 << iota
	usageAggregateTrends
	usageAggregateRankings
	usageAggregateDistributions
	usageAggregateEndpoints
)

type usageMetricTotals struct {
	records   int
	failed    int
	input     int
	output    int
	cached    int
	cacheHit  int
	reasoning int
	tokens    int
	cost      float64
	unpriced  int
	ttftTotal float64
	ttftCount int
}

type usageMetricGroup struct {
	key         string
	label       string
	records     int
	failed      int
	tokens      int
	cost        float64
	userID      *int
	description *string
}

type usageAggregateBuilder struct {
	filters UsageFilters
	parts   usageAggregateParts
	users   map[string]userInfo

	summary           usageMetricTotals
	trends            map[string]*usageMetricGroup
	apiKeys           map[string]*usageMetricGroup
	usersByKey        map[string]*usageMetricGroup
	models            map[string]*usageMetricGroup
	providers         map[string]*usageMetricGroup
	modelDistribution map[string]*usageMetricGroup
	endpoints         map[string]*usageMetricGroup
}

func newUsageAggregateBuilder(filters UsageFilters, parts usageAggregateParts, users map[string]userInfo) *usageAggregateBuilder {
	builder := &usageAggregateBuilder{filters: filters, parts: parts, users: users}
	if parts&usageAggregateTrends != 0 {
		builder.trends = map[string]*usageMetricGroup{}
	}
	if parts&usageAggregateRankings != 0 {
		builder.apiKeys = map[string]*usageMetricGroup{}
		builder.usersByKey = map[string]*usageMetricGroup{}
		builder.models = map[string]*usageMetricGroup{}
	}
	if parts&usageAggregateDistributions != 0 {
		builder.providers = map[string]*usageMetricGroup{}
		builder.modelDistribution = map[string]*usageMetricGroup{}
		builder.endpoints = map[string]*usageMetricGroup{}
	} else if parts&usageAggregateEndpoints != 0 {
		builder.endpoints = map[string]*usageMetricGroup{}
	}
	return builder
}

func (b *usageAggregateBuilder) add(record UsageRecord, cost float64, unpriced bool) {
	tokens := usageAggregateTotalTokens(record)
	if b.parts&usageAggregateSummary != 0 {
		b.summary.records++
		if record.Failed {
			b.summary.failed++
		}
		b.summary.input += usageAggregateInputTokens(record)
		b.summary.output += record.OutputTokens
		b.summary.cached += record.CachedTokens
		b.summary.cacheHit += usageCacheHitTokens(record)
		b.summary.reasoning += record.ReasoningTokens
		b.summary.tokens += tokens
		b.summary.cost = mathRound(b.summary.cost+cost, 8)
		if unpriced {
			b.summary.unpriced++
		}
		if record.TTFTMS != nil && *record.TTFTMS > 0 {
			b.summary.ttftTotal += *record.TTFTMS
			b.summary.ttftCount++
		}
	}
	if b.parts&usageAggregateTrends != 0 {
		bucket := usageTrendBucket(b.filters, record.Timestamp)
		addUsageMetricGroup(b.trends, bucket, bucket, record, tokens, cost, nil, nil)
	}
	if b.parts&usageAggregateRankings != 0 {
		b.addRankings(record, tokens, cost)
	}
	if b.parts&usageAggregateDistributions != 0 {
		provider := valueOr(record.Provider, "unknown")
		model := valueOr(record.Model, "unknown")
		endpoint := valueOr(record.Endpoint, "unknown")
		addUsageMetricGroup(b.providers, provider, provider, record, tokens, cost, nil, nil)
		addUsageMetricGroup(b.modelDistribution, model, model, record, tokens, cost, nil, nil)
		addUsageMetricGroup(b.endpoints, endpoint, endpoint, record, tokens, cost, nil, nil)
	} else if b.parts&usageAggregateEndpoints != 0 {
		endpoint := valueOr(record.Endpoint, "unknown")
		addUsageMetricGroup(b.endpoints, endpoint, endpoint, record, tokens, cost, nil, nil)
	}
}

func (b *usageAggregateBuilder) addRankings(record UsageRecord, tokens int, cost float64) {
	description := ""
	if record.APIKeyDescription != nil {
		description = strings.TrimSpace(*record.APIKeyDescription)
	}
	apiKey := description
	apiKeyLabel := description
	var descriptionPtr *string
	if apiKey == "" {
		apiKey = "unlabeled"
		apiKeyLabel = "未设置 KEY 描述"
	} else {
		value := description
		descriptionPtr = &value
	}
	addUsageMetricGroup(b.apiKeys, apiKey, apiKeyLabel, record, tokens, cost, nil, descriptionPtr)

	provider := valueOr(record.Provider, "unknown")
	model := valueOr(record.Model, "unknown")
	modelKey := provider + "::" + model
	addUsageMetricGroup(b.models, modelKey, provider+" / "+model, record, tokens, cost, nil, nil)

	if record.UsageUsername == nil {
		return
	}
	username := strings.TrimSpace(*record.UsageUsername)
	if username == "" {
		return
	}
	userKey, userLabel := username, username
	var userID *int
	if info, ok := b.users[username]; ok {
		id := info.ID
		userID = &id
		userKey = strconv.Itoa(id)
		userLabel = info.Name
	}
	addUsageMetricGroup(b.usersByKey, userKey, userLabel, record, tokens, cost, userID, nil)
}

func addUsageMetricGroup(groups map[string]*usageMetricGroup, key, label string, record UsageRecord, tokens int, cost float64, userID *int, description *string) {
	group := groups[key]
	if group == nil {
		group = &usageMetricGroup{key: key, label: label, userID: userID, description: description}
		groups[key] = group
	}
	group.records++
	if record.Failed {
		group.failed++
	}
	group.tokens += tokens
	group.cost = mathRound(group.cost+cost, 8)
}

func usageTrendBucket(filters UsageFilters, timestamp time.Time) string {
	duration := 24 * time.Hour
	if filters.Start != nil && filters.End != nil {
		duration = filters.End.Sub(*filters.Start)
	}
	local := timestamp.In(appTimeLocation)
	if duration <= 48*time.Hour {
		return local.Format("2006-01-02 15:00")
	}
	return local.Format("2006-01-02")
}

func (b *usageAggregateBuilder) summaryResponse() map[string]any {
	return map[string]any{
		"start":              usageAPITimePtr(b.filters.Start),
		"end":                usageAPITimePtr(b.filters.End),
		"total_records":      b.summary.records,
		"failed_records":     b.summary.failed,
		"success_records":    b.summary.records - b.summary.failed,
		"input_tokens":       b.summary.input,
		"output_tokens":      b.summary.output,
		"cached_tokens":      b.summary.cached,
		"cache_hit_tokens":   b.summary.cacheHit,
		"reasoning_tokens":   b.summary.reasoning,
		"total_tokens":       b.summary.tokens,
		"estimated_cost_usd": b.summary.cost,
		"unpriced_records":   b.summary.unpriced,
		"average_ttft_ms":    averageTTFTMS(b.summary.ttftTotal, b.summary.ttftCount),
	}
}

func (b *usageAggregateBuilder) trendResponse() []map[string]any {
	keys := make([]string, 0, len(b.trends))
	for key := range b.trends {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		group := b.trends[key]
		items = append(items, map[string]any{
			"bucket":             key,
			"records":            group.records,
			"failed_records":     group.failed,
			"total_tokens":       group.tokens,
			"estimated_cost_usd": group.cost,
		})
	}
	return items
}

func usageRankingResponse(groupBy string, groups map[string]*usageMetricGroup) map[string]any {
	items := usageMetricGroupItems(groups, true)
	return map[string]any{"group_by": groupBy, "items": items}
}

func usageDistributionResponse(groups map[string]*usageMetricGroup) []map[string]any {
	return usageMetricGroupItems(groups, false)
}

func usageMetricGroupItems(groups map[string]*usageMetricGroup, ranking bool) []map[string]any {
	items := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		item := map[string]any{
			"key":                group.key,
			"label":              group.label,
			"records":            group.records,
			"total_tokens":       group.tokens,
			"estimated_cost_usd": group.cost,
		}
		if ranking {
			item["failed_records"] = group.failed
			item["user_id"] = group.userID
			item["api_key_description"] = group.description
		}
		items = append(items, item)
	}
	if ranking {
		sort.Slice(items, func(i, j int) bool {
			leftTokens := items[i]["total_tokens"].(int)
			rightTokens := items[j]["total_tokens"].(int)
			if leftTokens == rightTokens {
				return items[i]["records"].(int) > items[j]["records"].(int)
			}
			return leftTokens > rightTokens
		})
	} else {
		sort.Slice(items, func(i, j int) bool {
			return items[i]["records"].(int) > items[j]["records"].(int)
		})
	}
	if len(items) > 20 {
		items = items[:20]
	}
	return items
}

func (b *usageAggregateBuilder) distributionsResponse() map[string]any {
	return map[string]any{
		"providers": usageDistributionResponse(b.providers),
		"models":    usageDistributionResponse(b.modelDistribution),
		"endpoints": usageDistributionResponse(b.endpoints),
	}
}

func (a *App) walkAggregateUsageRecords(ctx context.Context, filters UsageFilters, visit func(UsageRecord)) error {
	where, args := usageWhere(filters)
	rows, err := a.db.QueryContext(ctx, `
		SELECT CAST(timestamp AS TEXT), usage_username, api_key_description, provider, model, endpoint,
		       ttft_ms, failed, input_tokens, output_tokens, cached_tokens, cache_read_tokens,
		       cache_creation_tokens, reasoning_tokens, total_tokens
		FROM usage_records `+where, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var record UsageRecord
		var timestamp, username, description, provider, model, endpoint sql.NullString
		var ttft sql.NullFloat64
		if err := rows.Scan(&timestamp, &username, &description, &provider, &model, &endpoint, &ttft,
			&record.Failed, &record.InputTokens, &record.OutputTokens, &record.CachedTokens,
			&record.CacheReadTokens, &record.CacheCreationTokens, &record.ReasoningTokens, &record.TotalTokens); err != nil {
			return err
		}
		if parsed, ok := parseDBTime(timestamp.String); ok {
			record.Timestamp = parsed
		}
		record.UsageUsername = nullableString(username)
		record.APIKeyDescription = nullableString(description)
		record.Provider = nullableString(provider)
		record.Model = nullableString(model)
		record.Endpoint = nullableString(endpoint)
		record.TTFTMS = nullableFloat(ttft)
		visit(record)
	}
	return rows.Err()
}

func (a *App) buildUsageAggregate(ctx context.Context, filters UsageFilters, parts usageAggregateParts, users map[string]userInfo, prices map[[2]string]ModelPrice) (*usageAggregateBuilder, error) {
	builder := newUsageAggregateBuilder(filters, parts, users)
	err := a.walkAggregateUsageRecords(ctx, filters, func(record UsageRecord) {
		cost, unpriced := recordCost(record, prices)
		builder.add(record, cost, unpriced)
	})
	if err != nil {
		return nil, err
	}
	return builder, nil
}

func usageRecordInRange(record UsageRecord, filters UsageFilters) bool {
	return (filters.Start == nil || !record.Timestamp.Before(*filters.Start)) &&
		(filters.End == nil || record.Timestamp.Before(*filters.End))
}

func usageRecordMatchesFailure(record UsageRecord, failed *bool) bool {
	return failed == nil || record.Failed == *failed
}

func usageBaseFilters(filters UsageFilters) UsageFilters {
	filters.Failed = nil
	return filters
}

func usageRangeInside(inner, outer UsageFilters) bool {
	return inner.Start != nil && inner.End != nil && outer.Start != nil && outer.End != nil &&
		!inner.Start.Before(*outer.Start) && !inner.End.After(*outer.End)
}

func usageRangeCoversObservedToday(current, today UsageFilters, now time.Time) bool {
	return current.Start != nil && current.End != nil && today.Start != nil &&
		!current.Start.After(*today.Start) && current.End.After(*today.Start) &&
		!current.End.Before(now.Add(-2*time.Minute))
}

func (a *App) usageOverviewAggregates(ctx context.Context, scoped UsageFilters, scope usageAccessScope, prices map[[2]string]ModelPrice, users map[string]userInfo) (map[string]any, error) {
	now := time.Now().In(appTimeLocation)
	todayStart, todayEnd := defaultTodayRange()
	todayFilters := scoped
	todayFilters.Start = &todayStart
	todayFilters.End = &todayEnd
	realtimeStart := now.Add(-30 * time.Minute)
	realtimeFilters := scoped
	realtimeFilters.Start = &realtimeStart
	realtimeFilters.End = &now
	failedFilters := scoped
	failed := true
	failedFilters.Failed = &failed

	currentBase := usageBaseFilters(scoped)
	todayBase := usageBaseFilters(todayFilters)
	current := newUsageAggregateBuilder(scoped, usageAggregateSummary|usageAggregateTrends|usageAggregateRankings|usageAggregateDistributions, users)
	failedAggregate := newUsageAggregateBuilder(failedFilters, usageAggregateSummary|usageAggregateTrends|usageAggregateEndpoints, users)
	today := newUsageAggregateBuilder(todayFilters, usageAggregateTrends, users)
	realtime := newUsageAggregateBuilder(realtimeFilters, usageAggregateSummary, users)

	addCurrent := func(record UsageRecord, cost float64, unpriced bool) {
		if usageRecordInRange(record, scoped) {
			if usageRecordMatchesFailure(record, scoped.Failed) {
				current.add(record, cost, unpriced)
			}
			if record.Failed {
				failedAggregate.add(record, cost, unpriced)
			}
		}
	}
	addToday := func(record UsageRecord, cost float64, unpriced bool) {
		if usageRecordInRange(record, todayFilters) && usageRecordMatchesFailure(record, scoped.Failed) {
			today.add(record, cost, unpriced)
		}
		if usageRecordInRange(record, realtimeFilters) && usageRecordMatchesFailure(record, scoped.Failed) {
			realtime.add(record, cost, unpriced)
		}
	}
	addRecord := func(record UsageRecord, addToCurrent, addToToday bool) {
		cost, unpriced := recordCost(record, prices)
		if addToCurrent {
			addCurrent(record, cost, unpriced)
		}
		if addToToday {
			addToday(record, cost, unpriced)
		}
	}
	if usageRangeInside(scoped, todayFilters) {
		if err := a.walkAggregateUsageRecords(ctx, todayBase, func(record UsageRecord) {
			addRecord(record, true, true)
		}); err != nil {
			return nil, err
		}
	} else if usageRangeCoversObservedToday(scoped, todayFilters, now) {
		if err := a.walkAggregateUsageRecords(ctx, currentBase, func(record UsageRecord) {
			addRecord(record, true, true)
		}); err != nil {
			return nil, err
		}
	} else {
		if err := a.walkAggregateUsageRecords(ctx, currentBase, func(record UsageRecord) {
			addRecord(record, true, false)
		}); err != nil {
			return nil, err
		}
		if err := a.walkAggregateUsageRecords(ctx, todayBase, func(record UsageRecord) {
			addRecord(record, false, true)
		}); err != nil {
			return nil, err
		}
	}

	apiKeyRanking := usageRankingResponse("api_key_description", current.apiKeys)
	userRanking := map[string]any{"group_by": "user", "items": []any{}}
	if scope.IsAdmin {
		userRanking = usageRankingResponse("user", current.usersByKey)
	}
	return map[string]any{
		"summary":                      current.summaryResponse(),
		"trends":                       current.trendResponse(),
		"user_ranking":                 userRanking,
		"api_key_description_ranking":  apiKeyRanking,
		"api_key_ranking":              apiKeyRanking,
		"model_ranking":                usageRankingResponse("model", current.models),
		"distributions":                current.distributionsResponse(),
		"today_trends":                 today.trendResponse(),
		"failed_summary":               failedAggregate.summaryResponse(),
		"failed_trends":                failedAggregate.trendResponse(),
		"failed_endpoint_distribution": usageDistributionResponse(failedAggregate.endpoints),
		"realtime_summary":             realtime.summaryResponse(),
	}, nil
}
