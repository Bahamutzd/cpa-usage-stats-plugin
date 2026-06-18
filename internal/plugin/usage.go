package plugin

import (
	"context"
	"encoding/json"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/usage"
)

// usageRecord mirrors pluginapi.UsageRecord. We keep a local copy so the
// plugin compiles without depending on the host module — only the JSON shape
// matters at the RPC boundary.
type usageRecord struct {
	Provider        string              `json:"Provider"`
	ExecutorType    string              `json:"ExecutorType"`
	Model           string              `json:"Model"`
	Alias           string              `json:"Alias"`
	APIKey          string              `json:"APIKey"`
	AuthID          string              `json:"AuthID"`
	AuthIndex       string              `json:"AuthIndex"`
	AuthType        string              `json:"AuthType"`
	Source          string              `json:"Source"`
	ReasoningEffort string              `json:"ReasoningEffort"`
	ServiceTier     string              `json:"ServiceTier"`
	RequestedAt     string              `json:"RequestedAt"`
	Latency         int64               `json:"Latency"`
	TTFT            int64               `json:"TTFT"`
	Failed          bool                `json:"Failed"`
	Failure         usageFailure        `json:"Failure"`
	Detail          usageDetail         `json:"Detail"`
	ResponseHeaders map[string][]string `json:"ResponseHeaders"`
}

type usageFailure struct {
	StatusCode int    `json:"StatusCode"`
	Body       string `json:"Body"`
}

type usageDetail struct {
	InputTokens         int64 `json:"InputTokens"`
	OutputTokens        int64 `json:"OutputTokens"`
	ReasoningTokens     int64 `json:"ReasoningTokens"`
	CachedTokens        int64 `json:"CachedTokens"`
	CacheReadTokens     int64 `json:"CacheReadTokens"`
	CacheCreationTokens int64 `json:"CacheCreationTokens"`
	TotalTokens         int64 `json:"TotalTokens"`
}

func handleUsage(payload []byte) []byte {
	st := currentStore()
	if st == nil {
		// Plugin received usage before plugin.register completed. Drop it
		// rather than failing — the host treats usage.handle as fire-and-forget.
		return OkEnvelope(nil)
	}
	if len(payload) == 0 {
		return OkEnvelope(nil)
	}

	var rec usageRecord
	if errDecode := json.Unmarshal(payload, &rec); errDecode != nil {
		return ErrorEnvelope("decode_failed", errDecode.Error())
	}

	event := usage.FromRecord(usage.Record{
		Provider:        rec.Provider,
		ExecutorType:    rec.ExecutorType,
		Model:           rec.Model,
		Alias:           rec.Alias,
		APIKey:          rec.APIKey,
		AuthID:          rec.AuthID,
		AuthIndex:       rec.AuthIndex,
		AuthType:        rec.AuthType,
		Source:          rec.Source,
		ReasoningEffort: rec.ReasoningEffort,
		ServiceTier:     rec.ServiceTier,
		RequestedAt:     rec.RequestedAt,
		LatencyNS:       rec.Latency,
		TTFTNS:          rec.TTFT,
		Failed:          rec.Failed,
		FailStatusCode:  rec.Failure.StatusCode,
		FailBody:        rec.Failure.Body,
		InputTokens:     rec.Detail.InputTokens,
		OutputTokens:    rec.Detail.OutputTokens,
		ReasoningTokens: rec.Detail.ReasoningTokens,
		CachedTokens:    rec.Detail.CachedTokens,
		CacheRead:       rec.Detail.CacheReadTokens,
		CacheCreation:   rec.Detail.CacheCreationTokens,
		TotalTokens:     rec.Detail.TotalTokens,
	})

	if _, errInsert := st.InsertEvents(context.Background(), []usage.Event{event}); errInsert != nil {
		return ErrorEnvelope("insert_failed", errInsert.Error())
	}
	return OkEnvelope(nil)
}
