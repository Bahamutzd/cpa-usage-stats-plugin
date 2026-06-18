package dashboard

import (
	"net/http"
	"time"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/store"
)

func timeNowMS() int64 { return time.Now().UnixMilli() }

// buildSummary assembles the DashboardSummaryResponse. It always fills the
// required fields (window, today, rolling_30m, top_models_today,
// recent_failures) and best-effort fills the optional chart blocks; any block
// that errors leaves its slice nil so the front-end treats it as omitted.
func buildSummary(r *http.Request, st *store.Store, params summaryParams) summaryResponse {
	nowMS := params.NowMS
	const thirtyMin = 30 * 60 * 1000
	rollingStart := nowMS - thirtyMin
	if rollingStart < params.TodayStartMS {
		rollingStart = params.TodayStartMS
	}

	resp := summaryResponse{
		GeneratedAtMS: timeNowMS(),
		Window: summaryWindow{
			TodayStartMS:      params.TodayStartMS,
			NowMS:             nowMS,
			Rolling30MStartMS: rollingStart,
		},
		TopModelsToday: []topModel{},
		RecentFailures: []recentFailure{},
	}

	if today, err := st.AggregateBetween(r.Context(), params.TodayStartMS, nowMS); err == nil {
		resp.Today = todaySummary{
			TotalCalls:          today.TotalCalls,
			SuccessCalls:        today.SuccessCalls,
			FailureCalls:        today.FailureCalls,
			SuccessRate:         safeRate(today.SuccessCalls, today.TotalCalls),
			InputTokens:         today.InputTokens,
			OutputTokens:        today.OutputTokens,
			CachedTokens:        today.CachedTokens,
			CacheReadTokens:     today.CacheReadTokens,
			CacheCreationTokens: today.CacheCreationTokens,
			ReasoningTokens:     today.ReasoningTokens,
			TotalTokens:         today.TotalTokens,
			AverageLatencyMS:    nullFloat(today.AvgLatencyMS.Float64, today.AvgLatencyMS.Valid),
			ZeroTokenCalls:      today.ZeroTokenCalls,
		}
	}

	if rolling, err := st.AggregateBetween(r.Context(), rollingStart, nowMS); err == nil {
		resp.Rolling30M = rollingSummary{
			RPM:        float64(rolling.TotalCalls) / 30.0,
			TPM:        float64(rolling.TotalTokens) / 30.0,
			TotalCalls: rolling.TotalCalls,
			TotalTokens: rolling.TotalTokens,
		}
	}

	if topModels, err := st.TopModelsBetween(r.Context(), params.TodayStartMS, nowMS, params.TopModels); err == nil {
		resp.TopModelsToday = buildTopModels(topModels)
	}

	if points, err := st.BucketTimelineBetween(r.Context(), params.TodayStartMS, nowMS, 30*60*1000); err == nil {
		resp.TrafficTimeline = buildTraffic(points)
	}

	if hourly, err := st.HourlyTimelineBetween(r.Context(), params.TodayStartMS, nowMS); err == nil {
		resp.HourlyActivity = buildHourlyActivity(hourly, params.TodayStartMS)
	}

	resp.TokenMix = buildTokenMix(resp.Today)

	todayFilter := store.AnalyticsFilter{FromMS: params.TodayStartMS, ToMS: nowMS, IncludeFailed: true}
	if channels, err := st.ChannelModelStatsWithFilter(r.Context(), todayFilter); err == nil {
		resp.ChannelHealth = buildChannelHealth(channels)
	}
	if sources, err := st.FailureSourcesWithFilter(r.Context(), todayFilter); err == nil {
		resp.FailureSources = buildFailureSources(sources)
	}
	if failures, err := st.RecentFailuresWithFilter(r.Context(), todayFilter, params.RecentFailures); err == nil {
		resp.RecentFailures = buildRecentFailures(failures)
	}

	resp.TodayRequestHealthTimeline = buildHealthTimeline(r, st, params)
	return resp
}

func buildTopModels(stats []store.ModelStat) []topModel {
	rows := make([]topModel, 0, len(stats))
	for _, stat := range stats {
		rows = append(rows, topModel{
			Model:       stat.Model,
			Calls:       stat.Calls,
			Tokens:      stat.TotalTokens,
			SuccessRate: safeRate(stat.SuccessCalls, stat.Calls),
		})
	}
	return rows
}

func buildTraffic(points []store.TimelinePoint) []trafficPoint {
	var totalCalls, totalTokens int64
	for _, point := range points {
		totalCalls += point.Calls
		totalTokens += point.Tokens
	}
	rows := make([]trafficPoint, 0, len(points))
	for _, point := range points {
		rows = append(rows, trafficPoint{
			BucketMS:    point.BucketMS,
			Calls:       point.Calls,
			Tokens:      point.Tokens,
			Success:     point.Success,
			Failure:     point.Failure,
			CallsShare:  safeRate(point.Calls, totalCalls),
			TokensShare: safeRate(point.Tokens, totalTokens),
			FailureRate: safeRate(point.Failure, point.Calls),
		})
	}
	return rows
}

func buildHourlyActivity(points []store.TimelinePoint, todayStartMS int64) []hourlyActivityPoint {
	var maxCalls int64
	for _, point := range points {
		if point.Calls > maxCalls {
			maxCalls = point.Calls
		}
	}
	rows := make([]hourlyActivityPoint, 0, len(points))
	for _, point := range points {
		var intensity float64
		if maxCalls > 0 {
			intensity = float64(point.Calls) / float64(maxCalls)
		}
		hourIndex := (point.BucketMS - todayStartMS) / int64(time.Hour)
		rows = append(rows, hourlyActivityPoint{
			HourIndex: hourIndex,
			BucketMS:  point.BucketMS,
			Calls:     point.Calls,
			Tokens:    point.Tokens,
			Intensity: intensity,
		})
	}
	return rows
}

// buildTokenMix splits today's total tokens into the six segments the front-end
// donut chart renders. The ordering matches the front-end DashboardTokenMixSegment key union.
func buildTokenMix(today todaySummary) []tokenMixSegment {
	segments := []struct {
		key    string
		tokens int64
	}{
		{"input", today.InputTokens},
		{"output", today.OutputTokens},
		{"reasoning", today.ReasoningTokens},
		{"cached", today.CachedTokens},
		{"cache_read", today.CacheReadTokens},
		{"cache_creation", today.CacheCreationTokens},
	}
	total := int64(0)
	for _, segment := range segments {
		total += segment.tokens
	}
	out := make([]tokenMixSegment, 0, len(segments))
	for _, segment := range segments {
		out = append(out, tokenMixSegment{
			Key:    segment.key,
			Tokens: segment.tokens,
			Share:  safeRate(segment.tokens, total),
		})
	}
	return out
}

// buildChannelHealth groups channel stats by auth_index. tone is derived from
// the failure rate: good < 5%, warn < 20%, bad otherwise.
func buildChannelHealth(stats []store.ChannelModelStat) []channelHealth {
	type rollup struct {
		calls, failures, success, tokens int64
		avgLatency                        float64
		latencySamples                    int64
		source, account, label, provider  string
	}
	groups := map[string]*rollup{}
	order := make([]string, 0)
	for _, stat := range stats {
		roll, ok := groups[stat.AuthIndex]
		if !ok {
			roll = &rollup{
				source:   stat.Source,
				account:  stat.AccountSnapshot,
				label:    stat.AuthLabelSnapshot,
				provider: stat.AuthProviderSnapshot,
			}
			groups[stat.AuthIndex] = roll
			order = append(order, stat.AuthIndex)
		}
		roll.calls += stat.Calls
		roll.failures += stat.FailureCalls
		roll.success += stat.SuccessCalls
		roll.tokens += stat.TotalTokens
		if stat.AvgLatencyMS.Valid && stat.LatencySamples > 0 {
			roll.avgLatency += stat.AvgLatencyMS.Float64 * float64(stat.LatencySamples)
			roll.latencySamples += stat.LatencySamples
		}
	}
	rows := make([]channelHealth, 0, len(order))
	for _, authIndex := range order {
		roll := groups[authIndex]
		failureRate := safeRate(roll.failures, roll.calls)
		var avgLatency *float64
		if roll.latencySamples > 0 {
			value := roll.avgLatency / float64(roll.latencySamples)
			avgLatency = &value
		}
		rows = append(rows, channelHealth{
			AuthIndex:            authIndex,
			Source:               roll.source,
			Account:              roll.account,
			AuthLabel:            roll.label,
			AccountSnapshot:      roll.account,
			AuthLabelSnapshot:    roll.label,
			AuthProviderSnapshot: roll.provider,
			Calls:                roll.calls,
			Failures:             roll.failures,
			FailureRate:          failureRate,
			SuccessRate:          safeRate(roll.success, roll.calls),
			Tokens:               roll.tokens,
			AverageLatencyMS:     avgLatency,
			Tone:                 healthTone(failureRate),
		})
	}
	return rows
}

func buildFailureSources(stats []store.FailureSourceStat) []failureSource {
	rows := make([]failureSource, 0, len(stats))
	for _, stat := range stats {
		rows = append(rows, failureSource{
			SourceHash:           stat.SourceHash,
			AuthIndex:            stat.AuthIndex,
			Source:               stat.Source,
			AccountSnapshot:      stat.AccountSnapshot,
			AuthLabelSnapshot:    stat.AuthLabelSnapshot,
			AuthProviderSnapshot: stat.AuthProviderSnapshot,
			Calls:                stat.Calls,
			Failures:             stat.FailureCalls,
			FailureRate:          safeRate(stat.FailureCalls, stat.Calls),
			LastSeenMS:           stat.LastSeenMS,
			AverageLatencyMS:     nullFloat(stat.AvgLatencyMS.Float64, stat.AvgLatencyMS.Valid),
		})
	}
	return rows
}

func buildRecentFailures(failures []store.RecentFailure) []recentFailure {
	rows := make([]recentFailure, 0, len(failures))
	for _, f := range failures {
		row := recentFailure{
			TimestampMS:          f.TimestampMS,
			Model:                f.Model,
			APIKeyHash:           f.APIKeyHash,
			Source:               f.Source,
			SourceHash:           f.SourceHash,
			AuthIndex:            f.AuthIndex,
			AccountSnapshot:      f.AccountSnapshot,
			AuthLabelSnapshot:    f.AuthLabelSnapshot,
			AuthProviderSnapshot: f.AuthProviderSnapshot,
			Endpoint:             f.Endpoint,
			LatencyMS:            nullInt(f.LatencyMS.Int64, f.LatencyMS.Valid),
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

// buildHealthTimeline renders the today request-health timeline as 15-minute
// buckets. tone derivation matches the front-end buckets loosely: future
// buckets (after now) are marked, empty buckets are "empty", and the rest
// follows the same good/warn/bad thresholds as channel health.
func buildHealthTimeline(r *http.Request, st *store.Store, params summaryParams) *todayRequestHealthTimeline {
	const bucket = 15 * 60 * 1000
	points, err := st.BucketTimelineBetween(r.Context(), params.TodayStartMS, params.NowMS, bucket)
	if err != nil {
		return nil
	}
	var successCalls, failureCalls, totalCalls int64
	aligned := make([]todayRequestHealthTimelinePoint, 0, len(points))
	for _, point := range points {
		successCalls += point.Success
		failureCalls += point.Failure
		totalCalls += point.Calls
		future := point.BucketMS > params.NowMS
		failureRate := safeRate(point.Failure, point.Calls)
		aligned = append(aligned, todayRequestHealthTimelinePoint{
			BucketMS:    point.BucketMS,
			Calls:       point.Calls,
			Tokens:      point.Tokens,
			Success:     point.Success,
			Failure:     point.Failure,
			SuccessRate: safeRate(point.Success, point.Calls),
			FailureRate: failureRate,
			Tone:        healthTimelineTone(point.Calls, future, failureRate),
			Intensity:   0,
			Future:      future,
		})
	}
	return &todayRequestHealthTimeline{
		FromMS:       params.TodayStartMS,
		ToMS:         params.NowMS,
		BucketMS:     bucket,
		SuccessCalls: successCalls,
		FailureCalls: failureCalls,
		TotalCalls:   totalCalls,
		SuccessRate:  safeRate(successCalls, totalCalls),
		Points:       aligned,
	}
}

func healthTone(failureRate float64) string {
	if failureRate < 0.05 {
		return "good"
	}
	if failureRate < 0.20 {
		return "warn"
	}
	return "bad"
}

func healthTimelineTone(calls int64, future bool, failureRate float64) string {
	if future {
		return "future"
	}
	if calls == 0 {
		return "empty"
	}
	return healthTone(failureRate)
}

func safeRate(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func nullFloat(value float64, valid bool) *float64 {
	if !valid {
		return nil
	}
	return &value
}

func nullInt(value int64, valid bool) *int64 {
	if !valid {
		return nil
	}
	return &value
}