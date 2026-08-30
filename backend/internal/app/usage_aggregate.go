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
	zeroToken int
	cost      float64
	unpriced  int
	ttftTotal float64
	ttftCount int
}

type usageAggregateMetrics struct {
	records     int
	failed      int
	input       int
	output      int
	cached      int
	cacheRead   int
	cacheCreate int
	cacheHit    int
	reasoning   int
	tokens      int
	zeroToken   int
	cost        float64
	unpriced    int
	ttftTotal   float64
	ttftCount   int
	firstSeenAt time.Time
	lastSeenAt  time.Time
}

type usageAggregateEntry struct {
	record  UsageRecord
	metrics usageAggregateMetrics
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
	b.addMetrics(record, usageMetricsForRecord(record, cost, unpriced))
}

func (b *usageAggregateBuilder) addMetrics(record UsageRecord, metrics usageAggregateMetrics) {
	if b.parts&usageAggregateSummary != 0 {
		b.summary.records += metrics.records
		b.summary.failed += metrics.failed
		b.summary.input += metrics.input
		b.summary.output += metrics.output
		b.summary.cached += metrics.cached
		b.summary.cacheHit += metrics.cacheHit
		b.summary.reasoning += metrics.reasoning
		b.summary.tokens += metrics.tokens
		b.summary.zeroToken += metrics.zeroToken
		b.summary.cost = mathRound(b.summary.cost+metrics.cost, 8)
		b.summary.unpriced += metrics.unpriced
		b.summary.ttftTotal += metrics.ttftTotal
		b.summary.ttftCount += metrics.ttftCount
	}
	if b.parts&usageAggregateTrends != 0 {
		bucket := usageTrendBucket(b.filters, record.Timestamp)
		addUsageMetricGroup(b.trends, bucket, bucket, metrics, nil, nil)
	}
	if b.parts&usageAggregateRankings != 0 {
		b.addRankings(record, metrics)
	}
	if b.parts&usageAggregateDistributions != 0 {
		provider := valueOr(record.Provider, "unknown")
		model := valueOr(record.Model, "unknown")
		endpoint := valueOr(record.Endpoint, "unknown")
		addUsageMetricGroup(b.providers, provider, provider, metrics, nil, nil)
		addUsageMetricGroup(b.modelDistribution, model, model, metrics, nil, nil)
		addUsageMetricGroup(b.endpoints, endpoint, endpoint, metrics, nil, nil)
	} else if b.parts&usageAggregateEndpoints != 0 {
		endpoint := valueOr(record.Endpoint, "unknown")
		addUsageMetricGroup(b.endpoints, endpoint, endpoint, metrics, nil, nil)
	}
}

func (b *usageAggregateBuilder) addRankings(record UsageRecord, metrics usageAggregateMetrics) {
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
	addUsageMetricGroup(b.apiKeys, apiKey, apiKeyLabel, metrics, nil, descriptionPtr)

	provider := valueOr(record.Provider, "unknown")
	model := valueOr(record.Model, "unknown")
	modelKey := provider + "::" + model
	addUsageMetricGroup(b.models, modelKey, provider+" / "+model, metrics, nil, nil)

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
	addUsageMetricGroup(b.usersByKey, userKey, userLabel, metrics, userID, nil)
}

func addUsageMetricGroup(groups map[string]*usageMetricGroup, key, label string, metrics usageAggregateMetrics, userID *int, description *string) {
	group := groups[key]
	if group == nil {
		group = &usageMetricGroup{key: key, label: label, userID: userID, description: description}
		groups[key] = group
	}
	group.records += metrics.records
	group.failed += metrics.failed
	group.tokens += metrics.tokens
	group.cost = mathRound(group.cost+metrics.cost, 8)
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
		"zero_token_records": b.summary.zeroToken,
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

type usageAggregateQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (a *App) walkAggregateUsageEntries(ctx context.Context, filters UsageFilters, visit func(usageAggregateEntry)) error {
	fullStart, leftBoundary := usageFullHourStart(filters.Start)
	fullEnd, rightBoundary := usageFullHourEnd(filters.End)
	minimumCutoff := time.Now().In(appTimeLocation).AddDate(0, 0, -minimumUsageRetentionDays)
	potentiallyExpired := (leftBoundary && filters.Start != nil && filters.Start.Before(minimumCutoff)) ||
		(rightBoundary && fullEnd != nil && fullEnd.Before(minimumCutoff))
	if potentiallyExpired {
		cfg, err := a.loadConfig(ctx)
		if err != nil {
			return err
		}
		cutoff := time.Now().In(appTimeLocation).AddDate(0, 0, -cfg.UsageDetailRetentionDays)
		leftExpired := leftBoundary && filters.Start != nil && filters.Start.Before(cutoff)
		rightExpired := rightBoundary && fullEnd != nil && fullEnd.Before(cutoff)
		if leftExpired || rightExpired {
			return validationError("超出明细保留期的历史用量查询必须使用整点时间范围")
		}
	}
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if filters.RequestID != nil {
		if err := walkRawAggregateUsageEntries(ctx, tx, filters, "", nil, visit); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		committed = true
		return nil
	}

	var watermark int
	if err := tx.QueryRowContext(ctx, `SELECT last_rolled_usage_id FROM usage_rollup_state WHERE id = 1`).Scan(&watermark); err != nil {
		return err
	}
	if fullStart != nil && fullEnd != nil && !fullStart.Before(*fullEnd) {
		if err := walkRawAggregateUsageEntries(ctx, tx, filters, "", nil, visit); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		committed = true
		return nil
	}

	fullFilters := filters
	fullFilters.Start = fullStart
	fullFilters.End = fullEnd
	if err := walkHourlyRollupEntries(ctx, tx, fullFilters, visit); err != nil {
		return err
	}
	if err := walkRawAggregateUsageEntries(ctx, tx, fullFilters, "id > ?", []any{watermark}, visit); err != nil {
		return err
	}
	if leftBoundary {
		left := filters
		left.End = fullStart
		if filters.End != nil && left.End != nil && filters.End.Before(*left.End) {
			left.End = filters.End
		}
		if err := walkRawAggregateUsageEntries(ctx, tx, left, "", nil, visit); err != nil {
			return err
		}
	}
	if rightBoundary {
		right := filters
		right.Start = fullEnd
		if filters.Start != nil && right.Start != nil && filters.Start.After(*right.Start) {
			right.Start = filters.Start
		}
		if err := walkRawAggregateUsageEntries(ctx, tx, right, "", nil, visit); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func usageFullHourStart(value *time.Time) (*time.Time, bool) {
	if value == nil {
		return nil, false
	}
	local := value.In(appTimeLocation)
	floor := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, appTimeLocation)
	if local.Equal(floor) {
		return value, false
	}
	ceil := floor.Add(time.Hour)
	return &ceil, true
}

func usageFullHourEnd(value *time.Time) (*time.Time, bool) {
	if value == nil {
		return nil, false
	}
	local := value.In(appTimeLocation)
	floor := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, appTimeLocation)
	if local.Equal(floor) {
		return value, false
	}
	return &floor, true
}

func walkRawAggregateUsageEntries(ctx context.Context, queryer usageAggregateQueryer, filters UsageFilters, extra string, extraArgs []any, visit func(usageAggregateEntry)) error {
	where, args := usageWhere(filters)
	if extra != "" {
		where += " AND " + extra
		args = append(args, extraArgs...)
	}
	rows, err := queryer.QueryContext(ctx, `
		SELECT CAST(timestamp AS TEXT), usage_username, api_key_description, provider, model, endpoint,
		       ttft_ms, failed, input_tokens, output_tokens, cached_tokens, cache_read_tokens,
		       cache_creation_tokens, reasoning_tokens, total_tokens, cost_usd, unpriced
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
			&record.CacheReadTokens, &record.CacheCreationTokens, &record.ReasoningTokens, &record.TotalTokens,
			&record.CostUSD, &record.Unpriced); err != nil {
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
		record.CostStored = true
		cost, unpriced := recordCost(record, nil)
		metrics := usageMetricsForRecord(record, cost, unpriced)
		visit(usageAggregateEntry{record: record, metrics: metrics})
	}
	return rows.Err()
}

func walkHourlyRollupEntries(ctx context.Context, queryer usageAggregateQueryer, filters UsageFilters, visit func(usageAggregateEntry)) error {
	where, args := usageRollupWhere(filters)
	rows, err := queryer.QueryContext(ctx, `
		SELECT CAST(bucket_start AS TEXT), usage_username, api_key_description, provider, model, endpoint, failed,
		       record_count, failed_count, input_tokens, output_tokens, cached_tokens,
		       cache_read_tokens, cache_creation_tokens, cache_hit_tokens,
		       reasoning_tokens, total_tokens, zero_token_records, cost_usd, unpriced_records, ttft_ms_sum, ttft_sample_count,
		       CAST(first_timestamp AS TEXT), CAST(last_timestamp AS TEXT)
		FROM usage_hourly_rollups `+where, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var record UsageRecord
		var bucket, username, description, provider, model, endpoint, firstTimestamp, lastTimestamp string
		var metrics usageAggregateMetrics
		if err := rows.Scan(&bucket, &username, &description, &provider, &model, &endpoint, &record.Failed,
			&metrics.records, &metrics.failed, &metrics.input, &metrics.output, &metrics.cached,
			&metrics.cacheRead, &metrics.cacheCreate, &metrics.cacheHit,
			&metrics.reasoning, &metrics.tokens, &metrics.zeroToken, &metrics.cost, &metrics.unpriced, &metrics.ttftTotal, &metrics.ttftCount,
			&firstTimestamp, &lastTimestamp); err != nil {
			return err
		}
		if parsed, ok := parseDBTime(bucket); ok {
			record.Timestamp = parsed
		}
		if parsed, ok := parseDBTime(firstTimestamp); ok {
			metrics.firstSeenAt = parsed
		}
		if parsed, ok := parseDBTime(lastTimestamp); ok {
			metrics.lastSeenAt = parsed
		}
		record.UsageUsername = nonEmptyStringPtr(username)
		record.APIKeyDescription = nonEmptyStringPtr(description)
		record.Provider = nonEmptyStringPtr(provider)
		record.Model = nonEmptyStringPtr(model)
		record.Endpoint = nonEmptyStringPtr(endpoint)
		visit(usageAggregateEntry{record: record, metrics: metrics})
	}
	return rows.Err()
}

func usageRollupWhere(filters UsageFilters) (string, []any) {
	clauses := []string{"1 = 1"}
	args := []any{}
	appendValue := func(column string, value *string) {
		if value != nil {
			clauses = append(clauses, column+" = ?")
			args = append(args, *value)
		}
	}
	if filters.Start != nil {
		clauses = append(clauses, "bucket_start >= ?")
		args = append(args, dbTime(*filters.Start))
	}
	if filters.End != nil {
		clauses = append(clauses, "bucket_start < ?")
		args = append(args, dbTime(*filters.End))
	}
	appendValue("usage_username", filters.UsageUsername)
	appendValue("api_key_description", filters.APIKeyDescription)
	appendValue("provider", filters.Provider)
	appendValue("model", filters.Model)
	appendValue("source_key", filters.SourceKey)
	appendValue("endpoint", filters.Endpoint)
	if filters.Failed != nil {
		clauses = append(clauses, "failed = ?")
		args = append(args, *filters.Failed)
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func nonEmptyStringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func usageMetricsForRecord(record UsageRecord, cost float64, unpriced bool) usageAggregateMetrics {
	metrics := usageAggregateMetrics{
		records:     1,
		input:       usageAggregateInputTokens(record),
		output:      record.OutputTokens,
		cached:      record.CachedTokens,
		cacheRead:   record.CacheReadTokens,
		cacheCreate: record.CacheCreationTokens,
		cacheHit:    usageCacheHitTokens(record),
		reasoning:   record.ReasoningTokens,
		tokens:      usageAggregateTotalTokens(record),
		cost:        cost,
		firstSeenAt: record.Timestamp,
		lastSeenAt:  record.Timestamp,
	}
	if record.Failed {
		metrics.failed = 1
	}
	if unpriced {
		metrics.unpriced = 1
	}
	if metrics.tokens == 0 {
		metrics.zeroToken = 1
	}
	if record.TTFTMS != nil && *record.TTFTMS > 0 {
		metrics.ttftTotal = *record.TTFTMS
		metrics.ttftCount = 1
	}
	return metrics
}

func (a *App) buildUsageAggregate(ctx context.Context, filters UsageFilters, parts usageAggregateParts, users map[string]userInfo) (*usageAggregateBuilder, error) {
	builder := newUsageAggregateBuilder(filters, parts, users)
	err := a.walkAggregateUsageEntries(ctx, filters, func(entry usageAggregateEntry) {
		builder.addMetrics(entry.record, entry.metrics)
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

func (a *App) usageOverviewAggregates(ctx context.Context, scoped UsageFilters, scope usageAccessScope, users map[string]userInfo) (map[string]any, error) {
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

	addCurrent := func(record UsageRecord, metrics usageAggregateMetrics) {
		if usageRecordInRange(record, scoped) {
			if usageRecordMatchesFailure(record, scoped.Failed) {
				current.addMetrics(record, metrics)
			}
			if record.Failed {
				failedAggregate.addMetrics(record, metrics)
			}
		}
	}
	addToday := func(record UsageRecord, metrics usageAggregateMetrics) {
		if usageRecordInRange(record, todayFilters) && usageRecordMatchesFailure(record, scoped.Failed) {
			today.addMetrics(record, metrics)
		}
		if usageRecordInRange(record, realtimeFilters) && usageRecordMatchesFailure(record, scoped.Failed) {
			realtime.addMetrics(record, metrics)
		}
	}
	addRecord := func(entry usageAggregateEntry, addToCurrent, addToToday bool) {
		if addToCurrent {
			addCurrent(entry.record, entry.metrics)
		}
		if addToToday {
			addToday(entry.record, entry.metrics)
		}
	}
	if usageRangeInside(scoped, todayFilters) {
		if err := a.walkAggregateUsageEntries(ctx, todayBase, func(entry usageAggregateEntry) {
			addRecord(entry, true, true)
		}); err != nil {
			return nil, err
		}
	} else if usageRangeCoversObservedToday(scoped, todayFilters, now) {
		if err := a.walkAggregateUsageEntries(ctx, currentBase, func(entry usageAggregateEntry) {
			addRecord(entry, true, true)
		}); err != nil {
			return nil, err
		}
	} else {
		if err := a.walkAggregateUsageEntries(ctx, currentBase, func(entry usageAggregateEntry) {
			addRecord(entry, true, false)
		}); err != nil {
			return nil, err
		}
		if err := a.walkAggregateUsageEntries(ctx, todayBase, func(entry usageAggregateEntry) {
			addRecord(entry, false, true)
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
