package sqlite

import (
	"database/sql"
	"fmt"
)

// Migrate runs the idempotent schema bootstrap. Only tables this plugin needs
// are created here — CPA-Manager-Plus's wider schema (codex inspections,
// account actions, quota cooldowns, settings) is intentionally dropped because
// those features are out of scope for a usage-statistics-only plugin.
func Migrate(db *sql.DB) error {
	statements := []string{
		`pragma journal_mode = WAL`,
		`pragma synchronous = FULL`,
		`pragma busy_timeout = 5000`,
		`pragma foreign_keys = ON`,
		`create table if not exists usage_events (
			id integer primary key autoincrement,
			request_id text,
			event_hash text not null unique,
			timestamp_ms integer not null,
			timestamp text not null,
			provider text,
			executor_type text,
			model text not null,
			endpoint text,
			method text,
			path text,
			auth_type text,
			auth_index text,
			source text,
			source_hash text,
			api_key_hash text,
			account_snapshot text,
			auth_label_snapshot text,
			auth_file_snapshot text,
			auth_provider_snapshot text,
			auth_project_id_snapshot text,
			auth_snapshot_at_ms integer,
			requested_model text,
			resolved_model text,
			reasoning_effort text,
			service_tier text,
			input_tokens integer not null default 0,
			output_tokens integer not null default 0,
			reasoning_tokens integer not null default 0,
			cached_tokens integer not null default 0,
			cache_tokens integer not null default 0,
			cache_read_tokens integer not null default 0,
			cache_creation_tokens integer not null default 0,
			total_tokens integer not null default 0,
			latency_ms integer,
			ttft_ms integer,
			failed integer not null default 0,
			fail_status_code integer,
			fail_summary text,
			fail_body text,
			raw_json text,
			created_at_ms integer not null
		)`,
		`create index if not exists idx_usage_events_timestamp on usage_events(timestamp_ms)`,
		`create index if not exists idx_usage_events_request_id on usage_events(request_id)`,
		`create index if not exists idx_usage_events_model on usage_events(model)`,
		`create index if not exists idx_usage_events_auth_index on usage_events(auth_index)`,
		`create index if not exists idx_usage_events_endpoint on usage_events(endpoint)`,
		`create index if not exists idx_usage_events_failed on usage_events(failed)`,
		`create index if not exists idx_usage_events_api_key_hash on usage_events(api_key_hash)`,
		`create index if not exists idx_usage_events_source on usage_events(source)`,
		`create index if not exists idx_usage_events_ts_failed on usage_events(timestamp_ms, failed)`, 
		`create index if not exists idx_usage_events_ts_model on usage_events(timestamp_ms, model)`, 
		`create index if not exists idx_usage_events_ts_auth on usage_events(timestamp_ms, auth_index)`, 
		`create index if not exists idx_usage_events_ts_src on usage_events(timestamp_ms, source_hash)`, 
		`analyze`,
		`create table if not exists api_key_aliases (
			api_key_hash text primary key,
			alias text not null,
			updated_at_ms integer not null
		)`,
		`create table if not exists model_prices (
			model text primary key,
			prompt_per_1m real not null,
			completion_per_1m real not null,
			cache_per_1m real not null,
			cache_read_per_1m real not null default 0,
			cache_creation_per_1m real not null default 0,
			source text,
			source_model_id text,
			raw_json text,
			updated_at_ms integer not null,
			synced_at_ms integer
		)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return ensureUsageEventSnapshotColumns(db)
}

// ensureUsageEventSnapshotColumns mirrors CPA-Manager-Plus's additive
// migration so a database created by an older plugin revision keeps working
// when new snapshot columns appear. For a fresh database every column already
// exists and the loop is a no-op.
func ensureUsageEventSnapshotColumns(db *sql.DB) error {
	rows, err := db.Query(`pragma table_info(usage_events)`)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	existing := map[string]struct{}{}
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Reserved for forward-compatible additive migrations. Kept empty today
	// because the schema above already declares every column the analytics
	// queries read.
	columns := []struct {
		name       string
		definition string
	}{}
	for _, column := range columns {
		if _, ok := existing[column.name]; ok {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf(
			`alter table usage_events add column %s %s`,
			column.name,
			column.definition,
		)); err != nil {
			return err
		}
	}
	return nil
}