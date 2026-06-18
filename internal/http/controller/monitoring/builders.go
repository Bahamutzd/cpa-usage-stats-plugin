package monitoring

import (
	"net/http"
	"strings"
	"time"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/store"
)

// buildAnalyticsResponse assembles the MonitoringAnalyticsResponse for one
// request. Each include flag gates one block; blocks the caller did not ask
// for are omitted (the front-end treats them as undefined). When Include is
// nil we default to the front-end's common request (everything except the
// heavy events page) so the dashboard renders out of the box.
func buildAnalyticsResponse(r *http.Request, st *store.Store, req analyticsRequest) analyticsResponse {
	inc := normalizeInclude(req.Include)
	filter := toAnalyticsFilter(req)
	granularity := normalizeGranularity(inc.Granularity)

	resp := analyticsResponse{
		GeneratedAtMS: time.Now().UnixMilli(),
		Granularity:   granularity,
	}
	if inc.Summary != nil && *inc.Summary {
		resp.Summary = buildSummary(r, st, req, filter)
	}
	if inc.Timeline != nil && *inc.Timeline {
		points, err := st.TimelineWithFilter(r.Context(), filter, granularity)
		if err == nil {
			resp.Timeline = buildTimeline(points, granularity)
		}
	}
	if inc.HourlyDistribution != nil && *inc.HourlyDistribution {
		points, err := st.HourlyDistributionWithFilter(r.Context(), filter)
		if err == nil {
			resp.HourlyDistribution = buildHourly(points)
		}
	}
	if inc.ModelShare != nil && *inc.ModelShare {
		stats, err := st.ModelStatsWithFilter(r.Context(), filter, 0)
		if err == nil {
			resp.ModelShare = buildModelShare(stats)
		}
	}
	if inc.ModelStats != nil && *inc.ModelStats {
		stats, err := st.ModelStatsWithFilter(r.Context(), filter, 0)
		if err == nil {
			resp.ModelStats = buildModelStats(stats)
		}
	}
	if inc.ChannelShare != nil && *inc.ChannelShare {
		stats, err := st.ChannelModelStatsWithFilter(r.Context(), filter)
		if err == nil {
			resp.ChannelShare = buildChannelShare(stats)
		}
	}
	if inc.FailureSources != nil && *inc.FailureSources {
		stats, err := st.FailureSourcesWithFilter(r.Context(), filter)
		if err == nil {
			resp.FailureSources = buildFailureSources(stats)
		}
	}
	if inc.AccountStats != nil && *inc.AccountStats {
		stats, err := st.AccountModelStatsWithFilter(r.Context(), filter)
		if err == nil {
			resp.AccountStats = buildAccountStats(stats)
		}
	}
	if inc.APIKeyStats != nil && *inc.APIKeyStats {
		stats, err := st.APIKeyModelStatsWithFilter(r.Context(), filter)
		if err == nil {
			resp.APIKeyStats = buildAPIKeyStats(stats)
		}
	}
	if inc.FilterOptions != nil && *inc.FilterOptions {
		resp.FilterOptions = buildFilterOptions(r, st, filter)
	}
	if inc.TaskBuckets != nil && *inc.TaskBuckets {
		buckets, err := st.TaskBucketsWithFilter(r.Context(), filter)
		if err == nil {
			resp.TaskBuckets = buildTaskBuckets(buckets)
		}
	}
	if inc.RecentFailures != nil && *inc.RecentFailures > 0 {
		failures, err := st.RecentFailuresWithFilter(r.Context(), filter, *inc.RecentFailures)
		if err == nil {
			resp.RecentFailures = buildRecentFailures(failures)
		}
	}
	if inc.EventsPage != nil {
		resp.Events = buildEvents(r, st, filter, *inc.EventsPage)
	}
	return resp
}

// normalizeInclude fills in sensible defaults when the caller omits the
// include block entirely. The front-end normally sends an explicit include
// object, but curl-friendly defaults keep the endpoint usable.
func normalizeInclude(inc *analyticsInclude) analyticsInclude {
	if inc != nil {
		return *inc
	}
	all := true
	return analyticsInclude{
		Summary:            &all,
		Timeline:           &all,
		HourlyDistribution: &all,
		ModelShare:         &all,
		ModelStats:         &all,
		ChannelShare:       &all,
		FailureSources:     &all,
		AccountStats:       &all,
		APIKeyStats:        &all,
		TaskBuckets:        &all,
		RecentFailures:     intPtr(20),
	}
}

func intPtr(value int) *int { return &value }

func normalizeGranularity(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "day" {
		return "day"
	}
	return "hour"
}

// toAnalyticsFilter translates the JSON request into the store's
// AnalyticsFilter. Monitoring is a "see everything" view, so IncludeFailed
// defaults to true and is never narrowed to successes-only — the failed_only
// flag handles the "show only failures" case via the failed=1 condition in
// analyticsWhere. Go's zero value cannot distinguish "include_failed omitted"
// from "include_failed=false", so treating the field as always-true keeps the
// dashboard from silently dropping every failed request when the front-end
// omits the flag.
func toAnalyticsFilter(req analyticsRequest) store.AnalyticsFilter {
	filter := store.AnalyticsFilter{
		FromMS:            req.FromMS,
		ToMS:              req.ToMS,
		SearchQuery:       req.SearchQuery,
		SearchAPIKeyHash:  req.SearchAPIKeyHash,
		IncludeFailed:     true,
		ExcludeZeroTokens: false,
	}
	if req.Filters != nil {
		filter.Models = req.Filters.Models
		filter.Providers = req.Filters.Providers
		filter.Accounts = req.Filters.Accounts
		filter.AuthIndices = req.Filters.AuthIndices
		filter.APIKeyHashes = req.Filters.APIKeyHashes
		filter.SourceHashes = req.Filters.SourceHashes
		filter.FailedOnly = req.Filters.FailedOnly
		filter.ExcludeZeroTokens = req.Filters.ExcludeZeroTokens
	}
	return filter
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