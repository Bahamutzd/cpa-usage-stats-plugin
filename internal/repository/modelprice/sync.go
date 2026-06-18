package modelprice

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"
)

// LiteLLMSyncURL is the default catalog the sync endpoint pulls from. It is
// exported so callers can override it for tests or air-gapped mirrors.
var LiteLLMSyncURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

// SyncSourceLiteLLM tags prices imported from the LiteLLM catalog.
const SyncSourceLiteLLM = "litellm"

// FetchLiteLLM downloads the LiteLLM catalog and converts per-token costs to
// per-1M-token costs. The client timeout is generous because GitHub raw can
// be slow; this is a one-shot sync, not a proxy upstream connection, so a
// deadline here does not violate the no-timeout-after-connect rule.
func FetchLiteLLM(ctx context.Context, syncURL string, client *http.Client) (map[string]ModelPrice, int, error) {
	if syncURL == "" {
		syncURL = LiteLLMSyncURL
	}
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, syncURL, nil)
	if errReq != nil {
		return nil, 0, errors.New("model price sync failed: " + errReq.Error())
	}
	if client == nil {
		client = defaultSyncHTTPClient()
	}
	res, errDo := client.Do(req)
	if errDo != nil {
		return nil, 0, errors.New("model price sync failed: " + errDo.Error())
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, 0, errors.New("model price sync failed: " + res.Status)
	}
	body, errRead := io.ReadAll(res.Body)
	if errRead != nil {
		return nil, 0, errRead
	}
	var raw map[string]map[string]any
	if errDecode := json.Unmarshal(body, &raw); errDecode != nil {
		return nil, 0, errDecode
	}
	now := time.Now().UnixMilli()
	prices := map[string]ModelPrice{}
	skipped := 0
	for modelID, entry := range raw {
		promptCost, hasPrompt := readFloat(entry, "input_cost_per_token")
		completionCost, hasCompletion := readFloat(entry, "output_cost_per_token")
		cacheReadCost, hasCacheRead := readFirstFloat(entry, "cache_read_input_token_cost", "input_cache_read")
		cacheCreationCost, hasCacheCreation := readFirstFloat(entry, "cache_creation_input_token_cost", "cache_write_input_token_cost", "input_cache_write", "input_cache_creation")
		if !hasPrompt && !hasCompletion && !hasCacheRead && !hasCacheCreation {
			skipped++
			continue
		}
		rawEntry, _ := json.Marshal(entry)
		prices[modelID] = ModelPrice{
			Prompt:         promptCost * 1_000_000,
			Completion:     completionCost * 1_000_000,
			Cache:          cacheReadCost * 1_000_000,
			CacheRead:      cacheReadCost * 1_000_000,
			CacheCreation:  cacheCreationCost * 1_000_000,
			Source:         SyncSourceLiteLLM,
			SourceModelID:  modelID,
			RawJSON:        string(rawEntry),
			UpdatedAtMS:    now,
			SyncedAtMS:     &now,
		}
	}
	return prices, skipped, nil
}

func defaultSyncHTTPClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

func readFloat(entry map[string]any, key string) (float64, bool) {
	value, ok := entry[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}
	return 0, false
}

func readFirstFloat(entry map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if value, ok := readFloat(entry, key); ok {
			return value, true
		}
	}
	return 0, false
}