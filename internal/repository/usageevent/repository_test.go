package usageevent

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/repository/sqlite"
	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/usage"
)

func TestInsertBatchStoresUsageEvent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "usage.db")
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	repo := New(db)
	event := usage.FromRecord(usage.Record{
		Provider:        "codex",
		ExecutorType:    "CodexExecutor",
		Model:           "gpt-5.4",
		Alias:           "gpt-5.4",
		APIKey:          "sk-test",
		AuthIndex:       "auth-1",
		AuthType:        "oauth",
		Source:          "tester@example.com",
		ReasoningEffort: "high",
		ServiceTier:     "default",
		RequestedAt:     "2026-06-18T10:00:00Z",
		LatencyNS:       2_000_000_000,
		TTFTNS:          500_000_000,
		InputTokens:     128,
		OutputTokens:    32,
		TotalTokens:     160,
	})

	result, err := repo.InsertBatch(context.Background(), []usage.Event{event})
	if err != nil {
		t.Fatalf("insert batch: %v", err)
	}
	if result.Inserted != 1 {
		t.Fatalf("inserted = %d, want 1", result.Inserted)
	}
	if result.Skipped != 0 {
		t.Fatalf("skipped = %d, want 0", result.Skipped)
	}

	count, err := repo.Count(context.Background())
	if err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}
