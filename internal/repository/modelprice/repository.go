// Package modelprice stores per-model USD pricing used for cost roll-ups.
// The ModelPrice shape mirrors CPA-Manager-Plus's model.ModelPrice so the
// front-end ModelPricesPage reads the same camelCase fields.
package modelprice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"
)

// ModelPrice is one row in model_prices. Costs are USD per 1M tokens.
type ModelPrice struct {
	Prompt         float64 `json:"prompt"`
	Completion     float64 `json:"completion"`
	Cache          float64 `json:"cache,omitempty"`
	CacheRead      float64 `json:"cacheRead,omitempty"`
	CacheCreation  float64 `json:"cacheCreation,omitempty"`
	Source         string  `json:"source,omitempty"`
	SourceModelID  string  `json:"sourceModelId,omitempty"`
	RawJSON        string  `json:"rawJson,omitempty"`
	UpdatedAtMS    int64   `json:"updatedAtMs,omitempty"`
	SyncedAtMS     *int64  `json:"syncedAtMs,omitempty"`
}

// ModelPriceSyncResult counts how many prices a sync imported vs skipped.
type ModelPriceSyncResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

// Repository owns the model_prices table.
type Repository struct {
	db *sql.DB
}

// New returns a repository bound to the given database handle.
func New(db *sql.DB) Repository {
	return Repository{db: db}
}

// LoadAll returns every price keyed by model id, ordered by model.
func (r Repository) LoadAll(ctx context.Context) (map[string]ModelPrice, error) {
	rows, errQuery := r.db.QueryContext(ctx, `select
		model, prompt_per_1m, completion_per_1m, cache_per_1m, cache_read_per_1m, cache_creation_per_1m, source, source_model_id, raw_json,
		updated_at_ms, synced_at_ms
		from model_prices order by model`)
	if errQuery != nil {
		return nil, errQuery
	}
	defer func() {
		_ = rows.Close()
	}()

	prices := map[string]ModelPrice{}
	for rows.Next() {
		var modelID string
		var price ModelPrice
		var source, sourceModelID, rawJSON sql.NullString
		var syncedAt sql.NullInt64
		if errScan := rows.Scan(
			&modelID,
			&price.Prompt,
			&price.Completion,
			&price.Cache,
			&price.CacheRead,
			&price.CacheCreation,
			&source,
			&sourceModelID,
			&rawJSON,
			&price.UpdatedAtMS,
			&syncedAt,
		); errScan != nil {
			return nil, errScan
		}
		price.Source = source.String
		price.SourceModelID = sourceModelID.String
		price.RawJSON = rawJSON.String
		if syncedAt.Valid {
			value := syncedAt.Int64
			price.SyncedAtMS = &value
		}
		prices[modelID] = price
	}
	return prices, rows.Err()
}

// ReplaceAll deletes every row and reinserts the given prices. Used by the
// manual PUT endpoint so the UI can do full-table edits.
func (r Repository) ReplaceAll(ctx context.Context, prices map[string]ModelPrice) error {
	tx, errBegin := r.db.BeginTx(ctx, nil)
	if errBegin != nil {
		return errBegin
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `delete from model_prices`); err != nil {
		return err
	}
	if len(prices) == 0 {
		if errCommit := tx.Commit(); errCommit != nil {
			return errCommit
		}
		committed = true
		return nil
	}

	stmt, errPrepare := tx.PrepareContext(ctx, `insert into model_prices (
		model, prompt_per_1m, completion_per_1m, cache_per_1m, cache_read_per_1m, cache_creation_per_1m, source, source_model_id,
		raw_json, updated_at_ms, synced_at_ms
	) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if errPrepare != nil {
		return errPrepare
	}
	defer func() {
		_ = stmt.Close()
	}()

	now := time.Now().UnixMilli()
	for modelID, price := range prices {
		if errValidate := validateModelPrice(modelID, price); errValidate != nil {
			return errValidate
		}
		if _, err := stmt.ExecContext(
			ctx,
			modelID,
			price.Prompt,
			price.Completion,
			price.Cache,
			price.CacheRead,
			price.CacheCreation,
			nullString(price.Source),
			nullString(price.SourceModelID),
			nullString(price.RawJSON),
			now,
			nullInt(price.SyncedAtMS),
		); err != nil {
			return err
		}
	}
	if errCommit := tx.Commit(); errCommit != nil {
		return errCommit
	}
	committed = true
	return nil
}

// UpsertSynced upserts prices fetched from an upstream catalog. Rows that fail
// validation are skipped (counted) rather than aborting the whole batch.
func (r Repository) UpsertSynced(ctx context.Context, prices map[string]ModelPrice) (ModelPriceSyncResult, error) {
	if len(prices) == 0 {
		return ModelPriceSyncResult{}, nil
	}
	tx, errBegin := r.db.BeginTx(ctx, nil)
	if errBegin != nil {
		return ModelPriceSyncResult{}, errBegin
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	stmt, errPrepare := tx.PrepareContext(ctx, `insert into model_prices (
		model, prompt_per_1m, completion_per_1m, cache_per_1m, cache_read_per_1m, cache_creation_per_1m, source, source_model_id,
		raw_json, updated_at_ms, synced_at_ms
	) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	on conflict(model) do update set
		prompt_per_1m = excluded.prompt_per_1m,
		completion_per_1m = excluded.completion_per_1m,
		cache_per_1m = excluded.cache_per_1m,
		cache_read_per_1m = excluded.cache_read_per_1m,
		cache_creation_per_1m = excluded.cache_creation_per_1m,
		source = excluded.source,
		source_model_id = excluded.source_model_id,
		raw_json = excluded.raw_json,
		updated_at_ms = excluded.updated_at_ms,
		synced_at_ms = excluded.synced_at_ms`)
	if errPrepare != nil {
		return ModelPriceSyncResult{}, errPrepare
	}
	defer func() {
		_ = stmt.Close()
	}()

	now := time.Now().UnixMilli()
	result := ModelPriceSyncResult{}
	for modelID, price := range prices {
		if errValidate := validateModelPrice(modelID, price); errValidate != nil {
			result.Skipped++
			continue
		}
		if price.Source == "" {
			price.Source = "sync"
		}
		if price.SourceModelID == "" {
			price.SourceModelID = modelID
		}
		price.UpdatedAtMS = now
		price.SyncedAtMS = &now
		if _, err := stmt.ExecContext(
			ctx,
			modelID,
			price.Prompt,
			price.Completion,
			price.Cache,
			price.CacheRead,
			price.CacheCreation,
			nullString(price.Source),
			nullString(price.SourceModelID),
			nullString(price.RawJSON),
			now,
			now,
		); err != nil {
			return ModelPriceSyncResult{}, err
		}
		result.Imported++
	}
	if errCommit := tx.Commit(); errCommit != nil {
		return ModelPriceSyncResult{}, errCommit
	}
	committed = true
	return result, nil
}

func validateModelPrice(modelID string, price ModelPrice) error {
	if modelID == "" {
		return errors.New("model is required")
	}
	if !validPriceValue(price.Prompt) || !validPriceValue(price.Completion) || !validPriceValue(price.Cache) ||
		!validPriceValue(price.CacheRead) || !validPriceValue(price.CacheCreation) {
		return fmt.Errorf("invalid model price for %s", modelID)
	}
	return nil
}

func validPriceValue(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
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