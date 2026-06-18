package usageevent

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/usage"
)

// Repository is the interface the store layer consumes. It is a struct here
// (not a private struct) so the store can hold it by value, matching the
// CPA-Manager-Plus layout. The methods are split across repository.go,
// aggregate.go and analytics.go.
type Repository struct {
	db *sql.DB
}

// New returns a repository bound to the given database handle.
func New(db *sql.DB) Repository {
	return Repository{db: db}
}

// InsertResult records the outcome of a batch insert. Skipped covers events
// rejected by the unique event_hash index (CPA upstream may retry the same
// request, which produces the same EventHash).
type InsertResult struct {
	Inserted            int
	Skipped             int
	InsertedEventHashes []string
}

// InsertBatch upserts events and skips duplicates by event_hash. The original
// CPA-Manager-Plus implementation returns the inserted hashes for downstream
// observability; the plugin does not need them yet but keeps the field so the
// store signature stays stable.
func (r Repository) InsertBatch(ctx context.Context, events []usage.Event) (InsertResult, error) {
	if len(events) == 0 {
		return InsertResult{}, nil
	}
	tx, errBegin := r.db.BeginTx(ctx, nil)
	if errBegin != nil {
		return InsertResult{}, errBegin
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	stmt, errPrepare := tx.PrepareContext(ctx, `insert or ignore into usage_events (
		request_id, event_hash, timestamp_ms, timestamp, provider, executor_type, model, endpoint, method, path,
		auth_type, auth_index, source, source_hash, api_key_hash,
		account_snapshot, auth_label_snapshot, auth_file_snapshot, auth_provider_snapshot, auth_project_id_snapshot, auth_snapshot_at_ms,
		requested_model, resolved_model, reasoning_effort, service_tier,
		input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_tokens, cache_read_tokens, cache_creation_tokens, total_tokens,
		latency_ms, ttft_ms, failed, fail_status_code, fail_summary, fail_body, raw_json, created_at_ms
	) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if errPrepare != nil {
		return InsertResult{}, errPrepare
	}
	defer func() {
		_ = stmt.Close()
	}()

	result := InsertResult{}
	for _, event := range events {
		failed := 0
		if event.Failed {
			failed = 1
		}
		failSummarySource := event.FailSummary
		if failSummarySource == "" {
			failSummarySource = event.FailBody
		}
		failSummary := usage.FailSummaryFromBody(failSummarySource)
		rawJSON := usage.SafeRawJSON(event.RawJSON)
		res, errExec := stmt.ExecContext(
			ctx,
			nullString(event.RequestID),
			event.EventHash,
			event.TimestampMS,
			event.Timestamp,
			nullString(event.Provider),
			nullString(event.ExecutorType),
			event.Model,
			nullString(event.Endpoint),
			nullString(event.Method),
			nullString(event.Path),
			nullString(event.AuthType),
			nullString(event.AuthIndex),
			nullString(event.Source),
			nullString(event.SourceHash),
			nullString(event.APIKeyHash),
			nullString(event.AccountSnapshot),
			nullString(event.AuthLabelSnapshot),
			nullString(event.AuthFileSnapshot),
			nullString(event.AuthProviderSnapshot),
			nullString(event.AuthProjectIDSnapshot),
			nullPositiveInt64(event.AuthSnapshotAtMS),
			nullString(event.RequestedModel),
			nullString(event.ResolvedModel),
			nullString(event.ReasoningEffort),
			nullString(event.ServiceTier),
			event.InputTokens,
			event.OutputTokens,
			event.ReasoningTokens,
			event.CachedTokens,
			event.CacheTokens,
			event.CacheReadTokens,
			event.CacheCreationTokens,
			event.TotalTokens,
			nullInt(event.LatencyMS),
			nullInt(event.TTFTMS),
			failed,
			nullPositiveInt64(int64(event.FailStatusCode)),
			nullString(failSummary),
			nullString(event.FailBody),
			nullString(rawJSON),
			event.CreatedAtMS,
		)
		if errExec != nil {
			return InsertResult{}, errExec
		}
		affected, _ := res.RowsAffected()
		if affected > 0 {
			result.Inserted++
			result.InsertedEventHashes = append(result.InsertedEventHashes, event.EventHash)
		} else {
			result.Skipped++
		}
	}
	if errCommit := tx.Commit(); errCommit != nil {
		return InsertResult{}, errCommit
	}
	committed = true
	return result, nil
}

// ListRecent returns the most recent events newest-first, up to limit rows.
// limit <= 0 means "no cap" (the original CPA-Manager-Plus default of 50000).
func (r Repository) ListRecent(ctx context.Context, limit int) ([]usage.Event, error) {
	if limit <= 0 {
		limit = 50000
	}
	rows, errQuery := r.db.QueryContext(ctx, `select
		request_id, event_hash, timestamp_ms, timestamp, provider, executor_type, model, endpoint, method, path,
		auth_type, auth_index, source, source_hash, api_key_hash,
		account_snapshot, auth_label_snapshot, auth_file_snapshot, auth_provider_snapshot, auth_project_id_snapshot, auth_snapshot_at_ms,
		requested_model, resolved_model, reasoning_effort, service_tier,
		input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_tokens, cache_read_tokens, cache_creation_tokens, total_tokens,
		latency_ms, ttft_ms, failed, fail_status_code, fail_summary, created_at_ms
		from usage_events
		order by timestamp_ms desc, id desc
		limit ?`, limit)
	if errQuery != nil {
		return nil, errQuery
	}
	defer func() {
		_ = rows.Close()
	}()

	events := make([]usage.Event, 0)
	for rows.Next() {
		var event usage.Event
		var requestID, provider, executorType, endpoint, method, path, authType, authIndex, source, sourceHash, apiKeyHash, accountSnapshot, authLabelSnapshot, authFileSnapshot, authProviderSnapshot, authProjectIDSnapshot, requestedModel, resolvedModel, reasoningEffort, serviceTier, failSummary sql.NullString
		var authSnapshotAt sql.NullInt64
		var latency, ttft sql.NullInt64
		var failStatusCode sql.NullInt64
		var failed int
		if errScan := rows.Scan(
			&requestID,
			&event.EventHash,
			&event.TimestampMS,
			&event.Timestamp,
			&provider,
			&executorType,
			&event.Model,
			&endpoint,
			&method,
			&path,
			&authType,
			&authIndex,
			&source,
			&sourceHash,
			&apiKeyHash,
			&accountSnapshot,
			&authLabelSnapshot,
			&authFileSnapshot,
			&authProviderSnapshot,
			&authProjectIDSnapshot,
			&authSnapshotAt,
			&requestedModel,
			&resolvedModel,
			&reasoningEffort,
			&serviceTier,
			&event.InputTokens,
			&event.OutputTokens,
			&event.ReasoningTokens,
			&event.CachedTokens,
			&event.CacheTokens,
			&event.CacheReadTokens,
			&event.CacheCreationTokens,
			&event.TotalTokens,
			&latency,
			&ttft,
			&failed,
			&failStatusCode,
			&failSummary,
			&event.CreatedAtMS,
		); errScan != nil {
			return nil, errScan
		}
		event.RequestID = requestID.String
		event.Provider = provider.String
		event.ExecutorType = executorType.String
		event.Endpoint = endpoint.String
		event.Method = method.String
		event.Path = path.String
		event.AuthType = authType.String
		event.AuthIndex = authIndex.String
		event.Source = source.String
		event.SourceHash = sourceHash.String
		event.APIKeyHash = apiKeyHash.String
		event.AccountSnapshot = accountSnapshot.String
		event.AuthLabelSnapshot = authLabelSnapshot.String
		event.AuthFileSnapshot = authFileSnapshot.String
		event.AuthProviderSnapshot = authProviderSnapshot.String
		event.AuthProjectIDSnapshot = authProjectIDSnapshot.String
		event.RequestedModel = requestedModel.String
		event.ResolvedModel = resolvedModel.String
		event.ReasoningEffort = reasoningEffort.String
		event.ServiceTier = serviceTier.String
		if authSnapshotAt.Valid {
			event.AuthSnapshotAtMS = authSnapshotAt.Int64
		}
		if failStatusCode.Valid {
			event.FailStatusCode = int(failStatusCode.Int64)
		}
		event.FailSummary = failSummary.String
		event.Failed = failed != 0
		if latency.Valid {
			value := latency.Int64
			event.LatencyMS = &value
		}
		if ttft.Valid {
			value := ttft.Int64
			event.TTFTMS = &value
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// Count returns the total number of events stored.
func (r Repository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, `select count(*) from usage_events`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteBefore removes every event older than beforeMS (exclusive) and
// returns the number of rows removed. Used by the retention goroutine.
func (r Repository) DeleteBefore(ctx context.Context, beforeMS int64) (int64, error) {
	result, err := r.db.ExecContext(ctx, `delete from usage_events where timestamp_ms < ?`, beforeMS)
	if err != nil {
		return 0, err
	}
	removed, _ := result.RowsAffected()
	return removed, nil
}

// ExportJSONL streams every stored event as JSONL, oldest first. It drops
// raw_json and the raw fail_body so the export is safe to share; fail_summary
// is the redacted/truncated diagnostic intended for portable JSONL.
func (r Repository) ExportJSONL(ctx context.Context) ([]byte, error) {
	events, err := r.ListRecent(ctx, 0)
	if err != nil {
		return nil, err
	}
	output := make([]byte, 0)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		event.FailBody = ""
		event.RawJSON = ""
		line, errMarshal := json.Marshal(event)
		if errMarshal != nil {
			return nil, errMarshal
		}
		output = append(output, line...)
		output = append(output, '\n')
	}
	return output, nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullInt(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullPositiveInt64(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}