package monitoring

import (
	"net/http"
	"strings"
	"time"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/store"
)

// buildSummary computes the high-level counters plus the rolling-30m derived
// rates. RPM/TPM use the last 30 minutes of the window; avg_daily uses the
// whole window divided by active days. Cost stays 0 until the price overlay.
func buildSummary(r *http.Request, st *store.Store, req analyticsRequest, filter store.AnalyticsFilter) *summaryRow {
	agg, err := st.AggregateWithFilter(r.Context(), filter)
	if err != nil {
		return nil
	}
	zeroTokenModels, _ := st.ZeroTokenModelsWithFilter(r.Context(), filter)
	activeDays, _ := st.ActiveDaysWithFilter(r.Context(), filter)

	nowMS := req.NowMS
	if nowMS <= 0 {
		nowMS = time.Now().UnixMilli()
	}
	windowStart := req.FromMS
	windowMs := req.ToMS - req.FromMS
	rpm, tpm := rollingRates(nowMS, windowStart, agg.TotalCalls, agg.TotalTokens, windowMs)

	avgDailyReq := 0.0
	avgDailyTokens := 0.0
	if activeDays > 0 {
		avgDailyReq = float64(agg.TotalCalls) / float64(activeDays)
		avgDailyTokens = float64(agg.TotalTokens) / float64(activeDays)
	}

	taskBuckets, _ := st.TaskBucketsWithFilter(r.Context(), filter)
	approxTasks := int64(len(taskBuckets))
	approxFailures := int64(0)
	for _, bucket := range taskBuckets {
		approxFailures += bucket.Failure
	}
	approxSuccessRate := safeRate(approxTasks-approxFailures, approxTasks)

	row := &summaryRow{
		TotalCalls:            agg.TotalCalls,
		SuccessCalls:          agg.SuccessCalls,
		FailureCalls:          agg.FailureCalls,
		SuccessRate:           safeRate(agg.SuccessCalls, agg.TotalCalls),
		InputTokens:           agg.InputTokens,
		OutputTokens:          agg.OutputTokens,
		CachedTokens:          agg.CachedTokens,
		CacheReadTokens:       agg.CacheReadTokens,
		CacheCreationTokens:   agg.CacheCreationTokens,
		ReasoningTokens:       agg.ReasoningTokens,
		TotalTokens:           agg.TotalTokens,
		TotalCost:             0,
		AverageLatencyMS:      nullFloat(agg.AvgLatencyMS.Float64, agg.AvgLatencyMS.Valid),
		ZeroTokenCalls:        agg.ZeroTokenCalls,
		RPM30M:                rpm,
		TPM30M:                tpm,
		AvgDailyRequests:      avgDailyReq,
		AvgDailyTokens:        avgDailyTokens,
		ApproxTasks:           approxTasks,
		ApproxTaskFailures:    approxFailures,
		ApproxTaskSuccessRate: approxSuccessRate,
		ZeroTokenModels:       zeroTokenModels,
	}
	if row.ZeroTokenModels == nil {
		row.ZeroTokenModels = []string{}
	}
	return row
}

// rollingRates estimates requests/tokens per minute for the most recent 30
// minutes. When the requested window is shorter than 30m we use the window
// itself to avoid dividing by zero.
func rollingRates(nowMS, windowStart int64, calls, tokens int64, windowMs int64) (float64, float64) {
	const thirtyMin = 30 * 60 * 1000
	span := windowMs
	if nowMS-windowStart < thirtyMin {
		span = nowMS - windowStart
	}
	if span <= 0 {
		return 0, 0
	}
	minutes := float64(span) / float64(60*1000)
	if minutes <= 0 {
		return 0, 0
	}
	return float64(calls) / minutes, float64(tokens) / minutes
}

func buildTimeline(points []store.TimelinePoint, granularity string) []timelineRow {
	rows := make([]timelineRow, 0, len(points))
	for _, point := range points {
		rows = append(rows, timelineRow{
			BucketMS: point.BucketMS,
			Label:    bucketLabel(point.BucketMS, granularity),
			Calls:    point.Calls,
			Tokens:   point.Tokens,
			Success:  point.Success,
			Failure:  point.Failure,
		})
	}
	return rows
}

func bucketLabel(bucketMS int64, granularity string) string {
	t := time.UnixMilli(bucketMS).UTC()
	if granularity == "day" {
		return t.Format("2006-01-02")
	}
	return t.Format("2006-01-02 15:04")
}

func buildHourly(points []store.HourlyPoint) []hourlyRow {
	rows := make([]hourlyRow, 0, len(points))
	for _, point := range points {
		rows = append(rows, hourlyRow{Hour: point.Hour, Calls: point.Calls, Tokens: point.Tokens})
	}
	return rows
}

func buildModelShare(stats []store.ModelStat) []modelShareRow {
	rows := make([]modelShareRow, 0, len(stats))
	for _, stat := range stats {
		rows = append(rows, modelShareRow{
			Model:  stat.Model,
			Calls:  stat.Calls,
			Tokens: stat.TotalTokens,
		})
	}
	return rows
}

func buildModelStats(stats []store.ModelStat) []modelStatRow {
	rows := make([]modelStatRow, 0, len(stats))
	for _, stat := range stats {
		rows = append(rows, modelStatRow{
			Model:               stat.Model,
			Calls:               stat.Calls,
			SuccessCalls:        stat.SuccessCalls,
			FailureCalls:        stat.Calls - stat.SuccessCalls,
			SuccessRate:         safeRate(stat.SuccessCalls, stat.Calls),
			InputTokens:         stat.InputTokens,
			OutputTokens:        stat.OutputTokens,
			CachedTokens:        stat.CachedTokens,
			CacheReadTokens:     stat.CacheReadTokens,
			CacheCreationTokens: stat.CacheCreationTokens,
			TotalTokens:        stat.TotalTokens,
			Cost:                0,
		})
	}
	return rows
}

func buildChannelShare(stats []store.ChannelModelStat) []channelShareRow {
	rows := make([]channelShareRow, 0, len(stats))
	for _, stat := range stats {
		rows = append(rows, channelShareRow{
			AuthIndex:           stat.AuthIndex,
			Source:              stat.Source,
			AccountSnapshot:     stat.AccountSnapshot,
			AuthLabelSnapshot:   stat.AuthLabelSnapshot,
			AuthProviderSnapshot: stat.AuthProviderSnapshot,
			Calls:               stat.Calls,
			Success:             stat.SuccessCalls,
			Failure:             stat.FailureCalls,
			Tokens:              stat.TotalTokens,
			Cost:                0,
			AverageLatencyMS:    nullFloat(stat.AvgLatencyMS.Float64, stat.AvgLatencyMS.Valid && stat.LatencySamples > 0),
		})
	}
	return rows
}

func buildFailureSources(stats []store.FailureSourceStat) []failureSourceRow {
	rows := make([]failureSourceRow, 0, len(stats))
	for _, stat := range stats {
		rows = append(rows, failureSourceRow{
			Source:               stat.Source,
			SourceHash:           stat.SourceHash,
			AuthIndex:            stat.AuthIndex,
			AccountSnapshot:      stat.AccountSnapshot,
			AuthLabelSnapshot:    stat.AuthLabelSnapshot,
			AuthProviderSnapshot: stat.AuthProviderSnapshot,
			Calls:                stat.Calls,
			Failure:              stat.FailureCalls,
			LastSeenMS:           stat.LastSeenMS,
			AverageLatencyMS:     nullFloat(stat.AvgLatencyMS.Float64, stat.AvgLatencyMS.Valid),
		})
	}
	return rows
}

// buildAccountStats groups the per-account-per-model rows returned by the
// repository into the nested account_stat row shape the front-end expects:
// one row per account with a models[] breakdown.
func buildAccountStats(stats []store.AccountModelStat) []accountStatRow {
	groups := map[string]*accountStatRow{}
	order := make([]string, 0)
	for _, stat := range stats {
		id := accountID(stat)
		group, ok := groups[id]
		if !ok {
			group = &accountStatRow{
				ID:                  id,
				AccountSnapshot:     stat.AccountSnapshot,
				AuthLabelSnapshot:   stat.AuthLabelSnapshot,
				AuthProviderSnapshot: stat.AuthProviderSnapshot,
				AuthIndices:         []string{},
				Sources:             []string{},
				SourceHashes:        []string{},
			}
			groups[id] = group
			order = append(order, id)
		}
		if stat.AuthIndex != "" && !contains(group.AuthIndices, stat.AuthIndex) {
			group.AuthIndices = append(group.AuthIndices, stat.AuthIndex)
		}
		if stat.Source != "" && !contains(group.Sources, stat.Source) {
			group.Sources = append(group.Sources, stat.Source)
		}
		if stat.SourceHash != "" && !contains(group.SourceHashes, stat.SourceHash) {
			group.SourceHashes = append(group.SourceHashes, stat.SourceHash)
		}
		group.Calls += stat.Calls
		group.SuccessCalls += stat.SuccessCalls
		group.FailureCalls += stat.FailureCalls
		group.InputTokens += stat.InputTokens
		group.OutputTokens += stat.OutputTokens
		group.CachedTokens += stat.CachedTokens
		group.CacheReadTokens += stat.CacheReadTokens
		group.CacheCreationTokens += stat.CacheCreationTokens
		group.TotalTokens += stat.TotalTokens
		if stat.LastSeenMS > group.LastSeenMS {
			group.LastSeenMS = stat.LastSeenMS
		}
		group.Models = append(group.Models, accountModelStatRow{
			Model:               stat.Model,
			Calls:               stat.Calls,
			SuccessCalls:        stat.SuccessCalls,
			FailureCalls:        stat.FailureCalls,
			SuccessRate:         safeRate(stat.SuccessCalls, stat.Calls),
			InputTokens:         stat.InputTokens,
			OutputTokens:        stat.OutputTokens,
			CachedTokens:        stat.CachedTokens,
			CacheReadTokens:     stat.CacheReadTokens,
			CacheCreationTokens: stat.CacheCreationTokens,
			TotalTokens:        stat.TotalTokens,
			Cost:                0,
			LastSeenMS:         stat.LastSeenMS,
		})
	}
	rows := make([]accountStatRow, 0, len(order))
	for _, id := range order {
		group := groups[id]
		group.SuccessRate = safeRate(group.SuccessCalls, group.Calls)
		rows = append(rows, *group)
	}
	return rows
}

// buildAPIKeyStats is the API-key counterpart of buildAccountStats.
func buildAPIKeyStats(stats []store.APIKeyModelStat) []apiKeyStatRow {
	groups := map[string]*apiKeyStatRow{}
	order := make([]string, 0)
	for _, stat := range stats {
		id := stat.APIKeyHash
		if id == "" {
			id = "(unknown)"
		}
		group, ok := groups[id]
		if !ok {
			group = &apiKeyStatRow{
				ID:                  id,
				APIKeyHash:          id,
				AccountSnapshot:     stat.AccountSnapshot,
				AuthLabelSnapshot:   stat.AuthLabelSnapshot,
				AuthProviderSnapshot: stat.AuthProviderSnapshot,
				AuthIndices:         []string{},
				Sources:             []string{},
				SourceHashes:        []string{},
			}
			groups[id] = group
			order = append(order, id)
		}
		if stat.AuthIndex != "" && !contains(group.AuthIndices, stat.AuthIndex) {
			group.AuthIndices = append(group.AuthIndices, stat.AuthIndex)
		}
		if stat.Source != "" && !contains(group.Sources, stat.Source) {
			group.Sources = append(group.Sources, stat.Source)
		}
		if stat.SourceHash != "" && !contains(group.SourceHashes, stat.SourceHash) {
			group.SourceHashes = append(group.SourceHashes, stat.SourceHash)
		}
		group.Calls += stat.Calls
		group.SuccessCalls += stat.SuccessCalls
		group.FailureCalls += stat.FailureCalls
		group.InputTokens += stat.InputTokens
		group.OutputTokens += stat.OutputTokens
		group.CachedTokens += stat.CachedTokens
		group.CacheReadTokens += stat.CacheReadTokens
		group.CacheCreationTokens += stat.CacheCreationTokens
		group.TotalTokens += stat.TotalTokens
		if stat.LastSeenMS > group.LastSeenMS {
			group.LastSeenMS = stat.LastSeenMS
		}
		group.Models = append(group.Models, accountModelStatRow{
			Model:               stat.Model,
			Calls:               stat.Calls,
			SuccessCalls:        stat.SuccessCalls,
			FailureCalls:        stat.FailureCalls,
			SuccessRate:         safeRate(stat.SuccessCalls, stat.Calls),
			InputTokens:         stat.InputTokens,
			OutputTokens:        stat.OutputTokens,
			CachedTokens:        stat.CachedTokens,
			CacheReadTokens:     stat.CacheReadTokens,
			CacheCreationTokens: stat.CacheCreationTokens,
			TotalTokens:        stat.TotalTokens,
			Cost:                0,
			LastSeenMS:         stat.LastSeenMS,
		})
	}
	rows := make([]apiKeyStatRow, 0, len(order))
	for _, id := range order {
		group := groups[id]
		group.SuccessRate = safeRate(group.SuccessCalls, group.Calls)
		rows = append(rows, *group)
	}
	return rows
}

func accountID(stat store.AccountModelStat) string {
	parts := []string{stat.AccountSnapshot, stat.AuthLabelSnapshot, stat.AuthProviderSnapshot, stat.AuthIndex}
	joined := strings.Join(parts, "|")
	if strings.Trim(joined, "|") == "" {
		return "(unknown)"
	}
	return joined
}

func buildTaskBuckets(buckets []store.TaskBucket) []taskBucketRow {
	rows := make([]taskBucketRow, 0, len(buckets))
	for _, bucket := range buckets {
		rows = append(rows, taskBucketRow{
			BucketKey:           bucket.BucketKey,
			Total:               bucket.Total,
			Success:             bucket.Success,
			Failure:             bucket.Failure,
			FirstMS:             bucket.FirstMS,
			LastMS:              bucket.LastMS,
			Source:              bucket.Source,
			SourceHash:          bucket.SourceHash,
			AuthIndex:           bucket.AuthIndex,
			Models:              splitCSV(bucket.Models),
			Endpoints:           splitCSV(bucket.Endpoints),
			InputTokens:         bucket.InputTokens,
			OutputTokens:        bucket.OutputTokens,
			CachedTokens:        bucket.CachedTokens,
			CacheReadTokens:     bucket.CacheReadTokens,
			CacheCreationTokens: bucket.CacheCreationTokens,
			TotalTokens:         bucket.TotalTokens,
			AverageLatencyMS:    nullFloat(bucket.AvgLatencyMS.Float64, bucket.AvgLatencyMS.Valid),
			MaxLatencyMS:        nullInt(bucket.MaxLatencyMS.Int64, bucket.MaxLatencyMS.Valid),
		})
	}
	return rows
}

func buildRecentFailures(failures []store.RecentFailure) []recentFailureRow {
	rows := make([]recentFailureRow, 0, len(failures))
	for _, f := range failures {
		row := recentFailureRow{
			TimestampMS:          f.TimestampMS,
			Model:                f.Model,
			APIKeyHash:           f.APIKeyHash,
			Source:               f.Source,
			SourceHash:           f.SourceHash,
			AuthIndex:            f.AuthIndex,
			AccountSnapshot:     f.AccountSnapshot,
			AuthLabelSnapshot:    f.AuthLabelSnapshot,
			AuthProviderSnapshot: f.AuthProviderSnapshot,
			AuthProjectIDSnapshot: f.AuthProjectIDSnapshot,
			Endpoint:             f.Endpoint,
			DurationMS:           nullInt(f.LatencyMS.Int64, f.LatencyMS.Valid),
			FailSummary:          f.FailSummary,
		}
		if f.FailStatusCode.Valid {
			code := int(f.FailStatusCode.Int64)
			row.FailStatusCode = &code
		}
		rows = append(rows, row)
	}
	return rows
}

func buildEvents(r *http.Request, st *store.Store, filter store.AnalyticsFilter, page analyticsEventsPageRequest) *eventsResponse {
	limit := page.Limit
	if limit <= 0 {
		limit = 50
	}
	pageResult, err := st.EventsPageWithFilter(r.Context(), filter, page.BeforeMS, page.BeforeID, int(limit))
	if err != nil {
		return nil
	}
	rows := make([]eventRow, 0, len(pageResult.Items))
	for _, item := range pageResult.Items {
		row := eventRow{
			EventHash:             item.EventHash,
			TimestampMS:           item.TimestampMS,
			Model:                 item.Model,
			Endpoint:              item.Endpoint,
			Method:                item.Method,
			Path:                  item.Path,
			AuthIndex:             item.AuthIndex,
			Source:                item.Source,
			SourceHash:            item.SourceHash,
			APIKeyHash:            item.APIKeyHash,
			AccountSnapshot:       item.AccountSnapshot,
			AuthLabelSnapshot:     item.AuthLabelSnapshot,
			AuthProviderSnapshot:  item.AuthProviderSnapshot,
			AuthProjectIDSnapshot: item.AuthProjectIDSnapshot,
			ResolvedModel:         item.ResolvedModel,
			ReasoningEffort:       item.ReasoningEffort,
			ServiceTier:           item.ServiceTier,
			ExecutorType:          item.ExecutorType,
			InputTokens:           item.InputTokens,
			OutputTokens:          item.OutputTokens,
			CachedTokens:          item.CachedTokens,
			CacheReadTokens:       item.CacheReadTokens,
			CacheCreationTokens:   item.CacheCreationTokens,
			ReasoningTokens:       item.ReasoningTokens,
			TotalTokens:           item.TotalTokens,
			LatencyMS:             nullInt(item.LatencyMS.Int64, item.LatencyMS.Valid),
			TTFTMS:                nullInt(item.TTFTMS.Int64, item.TTFTMS.Valid),
			Failed:                item.Failed,
			FailSummary:           item.FailSummary,
		}
		if item.FailStatusCode.Valid {
			code := int(item.FailStatusCode.Int64)
			row.FailStatusCode = &code
		}
		rows = append(rows, row)
	}
	resp := &eventsResponse{
		Items:        rows,
		NextBeforeMS: pageResult.NextBeforeMS,
		HasMore:      pageResult.HasMore,
	}
	if pageResult.NextBeforeID > 0 {
		resp.NextBeforeID = pageResult.NextBeforeID
	}
	if total, err := st.EventsCountWithFilter(r.Context(), filter); err == nil {
		resp.TotalCount = &total
	}
	return resp
}

// buildFilterOptions returns a lightweight version of the analytics blocks the
// filter dropdowns read. It reuses the same builders so the shapes stay
// consistent with the main response.
func buildFilterOptions(r *http.Request, st *store.Store, filter store.AnalyticsFilter) *filterOptions {
	opts := &filterOptions{}
	if accountStats, err := st.AccountModelStatsWithFilter(r.Context(), filter); err == nil {
		opts.AccountStats = buildAccountStats(accountStats)
	}
	if apiKeyStats, err := st.APIKeyModelStatsWithFilter(r.Context(), filter); err == nil {
		opts.APIKeyStats = buildAPIKeyStats(apiKeyStats)
	}
	if channelShare, err := st.ChannelModelStatsWithFilter(r.Context(), filter); err == nil {
		opts.ChannelShare = buildChannelShare(channelShare)
	}
	if modelStats, err := st.ModelStatsWithFilter(r.Context(), filter, 0); err == nil {
		opts.ModelStats = buildModelStats(modelStats)
	}
	return opts
}

func splitCSV(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}