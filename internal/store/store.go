// Package store wires the SQLite database, repositories, and the high-level
// methods used by the HTTP controllers. It is the only place HTTP handlers
// touch the database.
package store

import (
	"context"
	"database/sql"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/repository/apikeyalias"
	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/repository/modelprice"
	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/repository/sqlite"
	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/repository/usageevent"
	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/usage"
)

// Re-export the analytics result types so controllers import one package
// instead of reaching into the repository layer directly.
type (
	Aggregate         = usageevent.Aggregate
	ModelStat         = usageevent.ModelStat
	RecentFailure     = usageevent.RecentFailure
	AnalyticsFilter   = usageevent.AnalyticsFilter
	TimelinePoint     = usageevent.TimelinePoint
	HourlyPoint       = usageevent.HourlyPoint
	ChannelModelStat  = usageevent.ChannelModelStat
	FailureSourceStat = usageevent.FailureSourceStat
	AccountModelStat  = usageevent.AccountModelStat
	APIKeyModelStat   = usageevent.APIKeyModelStat
	TaskBucket        = usageevent.TaskBucket
	EventPageItem     = usageevent.EventPageItem
	EventsPage        = usageevent.EventsPage
	InsertResult      = usageevent.InsertResult
	APIKeyAlias       = apikeyalias.APIKeyAlias
	ModelPrice        = modelprice.ModelPrice
	ModelPriceSyncResult = modelprice.ModelPriceSyncResult
)

// Store is the plugin's facade over the database. Controllers should depend
// on this struct, never on the underlying repository.
type Store struct {
	db *sql.DB

	UsageEvents   usageevent.Repository
	APIKeyAliases apikeyalias.Repository
	ModelPrices   modelprice.Repository
}

// Open creates a Store backed by a fresh SQLite database at the given path.
func Open(path string) (*Store, error) {
	db, err := sqlite.Open(path)
	if err != nil {
		return nil, err
	}
	return New(db), nil
}

// New wires a Store around an existing database handle. Used by tests.
func New(db *sql.DB) *Store {
	return &Store{
		db:            db,
		UsageEvents:   usageevent.New(db),
		APIKeyAliases: apikeyalias.New(db),
		ModelPrices:   modelprice.New(db),
	}
}

// Close releases the database connection.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// InsertEvents proxies to the usage events repository.
func (s *Store) InsertEvents(ctx context.Context, events []usage.Event) (InsertResult, error) {
	return s.UsageEvents.InsertBatch(ctx, events)
}

// Count returns total stored events.
func (s *Store) Count(ctx context.Context) (int64, error) {
	return s.UsageEvents.Count(ctx)
}

// RecentEvents returns the most recent events newest-first, capped by limit.
func (s *Store) RecentEvents(ctx context.Context, limit int) ([]usage.Event, error) {
	return s.UsageEvents.ListRecent(ctx, limit)
}

// DeleteEventsBefore removes events older than beforeMS (exclusive). Used by
// the retention goroutine when retention_days > 0.
func (s *Store) DeleteEventsBefore(ctx context.Context, beforeMS int64) (int64, error) {
	return s.UsageEvents.DeleteBefore(ctx, beforeMS)
}

// ExportJSONL returns a JSONL dump of every stored event (redacted).
func (s *Store) ExportJSONL(ctx context.Context) ([]byte, error) {
	return s.UsageEvents.ExportJSONL(ctx)
}

// AggregateBetween computes summary metrics over [fromMs, toMs).
func (s *Store) AggregateBetween(ctx context.Context, fromMs, toMs int64) (Aggregate, error) {
	return s.UsageEvents.AggregateBetween(ctx, fromMs, toMs)
}

// TopModelsBetween returns the most active models ordered by call count.
func (s *Store) TopModelsBetween(ctx context.Context, fromMs, toMs int64, limit int) ([]ModelStat, error) {
	return s.UsageEvents.TopModelsBetween(ctx, fromMs, toMs, limit)
}

// ModelStatsBetween returns per-model totals for all models in a window.
func (s *Store) ModelStatsBetween(ctx context.Context, fromMs, toMs int64) ([]ModelStat, error) {
	return s.UsageEvents.ModelStatsBetween(ctx, fromMs, toMs)
}

// RecentFailuresBetween returns the most recent failed events in window.
func (s *Store) RecentFailuresBetween(ctx context.Context, fromMs, toMs int64, limit int) ([]RecentFailure, error) {
	return s.UsageEvents.RecentFailuresBetween(ctx, fromMs, toMs, limit)
}

// HourlyTimelineBetween returns hourly buckets over the window.
func (s *Store) HourlyTimelineBetween(ctx context.Context, fromMs, toMs int64) ([]TimelinePoint, error) {
	return s.UsageEvents.HourlyTimelineBetween(ctx, fromMs, toMs)
}

// BucketTimelineBetween returns fixed-size buckets over the window.
func (s *Store) BucketTimelineBetween(ctx context.Context, fromMs, toMs int64, bucketMs int64) ([]TimelinePoint, error) {
	return s.UsageEvents.BucketTimelineBetween(ctx, fromMs, toMs, bucketMs)
}

// AggregateWithFilter computes summary metrics for a filter window.
func (s *Store) AggregateWithFilter(ctx context.Context, filter AnalyticsFilter) (Aggregate, error) {
	return s.UsageEvents.AggregateWithFilter(ctx, filter)
}

// ModelStatsWithFilter returns per-model totals for a filter window.
func (s *Store) ModelStatsWithFilter(ctx context.Context, filter AnalyticsFilter, limit int) ([]ModelStat, error) {
	return s.UsageEvents.ModelStatsWithFilter(ctx, filter, limit)
}

// TimelineWithFilter returns timeline buckets for a filter window.
func (s *Store) TimelineWithFilter(ctx context.Context, filter AnalyticsFilter, granularity string) ([]TimelinePoint, error) {
	return s.UsageEvents.TimelineWithFilter(ctx, filter, granularity)
}

// HourlyDistributionWithFilter returns hourly distribution for a filter window.
func (s *Store) HourlyDistributionWithFilter(ctx context.Context, filter AnalyticsFilter) ([]HourlyPoint, error) {
	return s.UsageEvents.HourlyDistributionWithFilter(ctx, filter)
}

// ChannelModelStatsWithFilter returns per-channel per-model totals.
func (s *Store) ChannelModelStatsWithFilter(ctx context.Context, filter AnalyticsFilter) ([]ChannelModelStat, error) {
	return s.UsageEvents.ChannelModelStatsWithFilter(ctx, filter)
}

// FailureSourcesWithFilter returns grouped failure sources.
func (s *Store) FailureSourcesWithFilter(ctx context.Context, filter AnalyticsFilter) ([]FailureSourceStat, error) {
	return s.UsageEvents.FailureSourcesWithFilter(ctx, filter)
}

// AccountModelStatsWithFilter returns per-account per-model totals.
func (s *Store) AccountModelStatsWithFilter(ctx context.Context, filter AnalyticsFilter) ([]AccountModelStat, error) {
	return s.UsageEvents.AccountModelStatsWithFilter(ctx, filter)
}

// APIKeyModelStatsWithFilter returns per-API-key per-model totals.
func (s *Store) APIKeyModelStatsWithFilter(ctx context.Context, filter AnalyticsFilter) ([]APIKeyModelStat, error) {
	return s.UsageEvents.APIKeyModelStatsWithFilter(ctx, filter)
}

// TaskBucketsWithFilter returns task-bucket aggregates.
func (s *Store) TaskBucketsWithFilter(ctx context.Context, filter AnalyticsFilter) ([]TaskBucket, error) {
	return s.UsageEvents.TaskBucketsWithFilter(ctx, filter)
}

// RecentFailuresWithFilter returns the most recent failures matching a filter.
func (s *Store) RecentFailuresWithFilter(ctx context.Context, filter AnalyticsFilter, limit int) ([]RecentFailure, error) {
	return s.UsageEvents.RecentFailuresWithFilter(ctx, filter, limit)
}

// EventsPageWithFilter returns a page of events for keyset pagination.
func (s *Store) EventsPageWithFilter(ctx context.Context, filter AnalyticsFilter, beforeMS int64, beforeID int64, limit int) (EventsPage, error) {
	return s.UsageEvents.EventsPageWithFilter(ctx, filter, beforeMS, beforeID, limit)
}

// EventsCountWithFilter returns the total number of events matching a filter.
func (s *Store) EventsCountWithFilter(ctx context.Context, filter AnalyticsFilter) (int64, error) {
	return s.UsageEvents.EventsCountWithFilter(ctx, filter)
}

// ActiveDaysWithFilter returns the count of distinct active days.
func (s *Store) ActiveDaysWithFilter(ctx context.Context, filter AnalyticsFilter) (int64, error) {
	return s.UsageEvents.ActiveDaysWithFilter(ctx, filter)
}

// ZeroTokenModelsWithFilter returns models that produced zero tokens.
func (s *Store) ZeroTokenModelsWithFilter(ctx context.Context, filter AnalyticsFilter) ([]string, error) {
	return s.UsageEvents.ZeroTokenModelsWithFilter(ctx, filter)
}

// LoadAPIKeyAliases returns every stored api key alias.
func (s *Store) LoadAPIKeyAliases(ctx context.Context) ([]APIKeyAlias, error) {
	return s.APIKeyAliases.LoadAll(ctx)
}

// UpsertAPIKeyAliases inserts or updates the given aliases.
func (s *Store) UpsertAPIKeyAliases(ctx context.Context, aliases []APIKeyAlias, activeHashes []string, allowOrphanCleanup bool) error {
	return s.APIKeyAliases.UpsertMany(ctx, aliases, activeHashes, allowOrphanCleanup)
}

// DeleteAPIKeyAlias removes a single alias by its hash.
func (s *Store) DeleteAPIKeyAlias(ctx context.Context, apiKeyHash string) error {
	return s.APIKeyAliases.Delete(ctx, apiKeyHash)
}

// LoadModelPrices returns every stored model price keyed by model id.
func (s *Store) LoadModelPrices(ctx context.Context) (map[string]ModelPrice, error) {
	return s.ModelPrices.LoadAll(ctx)
}

// ReplaceModelPrices deletes and reinserts the full price table.
func (s *Store) ReplaceModelPrices(ctx context.Context, prices map[string]ModelPrice) error {
	return s.ModelPrices.ReplaceAll(ctx, prices)
}

// UpsertSyncedModelPrices upserts prices fetched from a catalog.
func (s *Store) UpsertSyncedModelPrices(ctx context.Context, prices map[string]ModelPrice) (ModelPriceSyncResult, error) {
	return s.ModelPrices.UpsertSynced(ctx, prices)
}