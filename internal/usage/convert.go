package usage

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// Record is the language-internal projection of pluginapi.UsageRecord. The
// host-facing JSON payload is decoded into this struct first so the
// conversion to Event lives in one place. Only fields the usage_plugin
// capability actually supplies are modeled here; the rest of Event stays
// empty (endpoint/method/path/account snapshots come from the collector in
// CPA-Manager-Plus and are not available to a usage.handle-only plugin).
type Record struct {
	Provider        string
	ExecutorType    string
	Model           string
	Alias           string
	APIKey          string
	AuthID          string
	AuthIndex       string
	AuthType        string
	Source          string
	ReasoningEffort string
	ServiceTier     string
	RequestedAt     string
	LatencyNS       int64
	TTFTNS          int64
	Failed          bool
	FailStatusCode  int
	FailBody        string

	InputTokens     int64
	OutputTokens    int64
	ReasoningTokens int64
	CachedTokens    int64
	CacheRead       int64
	CacheCreation   int64
	TotalTokens     int64
}

// FromRecord converts a host-supplied usage record into the canonical Event
// stored locally. It reuses the same buildEventHash / FailSummaryFromBody
// helpers NormalizeRaw uses so dedup and redaction stay consistent with
// CPA-Manager-Plus.
func FromRecord(r Record) Event {
	requestedAt := parseRecordTimestamp(r.RequestedAt)
	if requestedAt.IsZero() {
		requestedAt = time.Now().UTC()
	}
	timestampMS := requestedAt.UnixMilli()
	timestamp := requestedAt.UTC().Format(time.RFC3339Nano)

	requestedModel := strings.TrimSpace(r.Alias)
	if requestedModel == "" {
		requestedModel = strings.TrimSpace(r.Model)
	}
	resolvedModel := strings.TrimSpace(r.Model)
	model := strings.TrimSpace(r.Model)
	if model == "" {
		model = "-"
	}

	var latencyMS *int64
	if r.LatencyNS > 0 {
		ms := r.LatencyNS / int64(time.Millisecond)
		latencyMS = &ms
	}
	var ttftMS *int64
	if r.TTFTNS > 0 {
		ms := r.TTFTNS / int64(time.Millisecond)
		ttftMS = &ms
	}

	event := Event{
		TimestampMS:         timestampMS,
		Timestamp:           timestamp,
		Provider:            strings.TrimSpace(r.Provider),
		ExecutorType:        strings.TrimSpace(r.ExecutorType),
		Model:               model,
		RequestedModel:      requestedModel,
		ResolvedModel:       resolvedModel,
		AuthType:            strings.TrimSpace(r.AuthType),
		AuthIndex:           strings.TrimSpace(r.AuthIndex),
		Source:              strings.TrimSpace(r.Source),
		APIKeyHash:          hashAPIKey(r.APIKey),
		ReasoningEffort:     strings.TrimSpace(r.ReasoningEffort),
		ServiceTier:         strings.TrimSpace(r.ServiceTier),
		InputTokens:         r.InputTokens,
		OutputTokens:        r.OutputTokens,
		ReasoningTokens:     r.ReasoningTokens,
		CachedTokens:        r.CachedTokens,
		CacheReadTokens:     r.CacheRead,
		CacheCreationTokens: r.CacheCreation,
		TotalTokens:         r.TotalTokens,
		LatencyMS:           latencyMS,
		TTFTMS:              ttftMS,
		Failed:              r.Failed,
		FailStatusCode:      r.FailStatusCode,
		FailSummary:         FailSummaryFromBody(r.FailBody),
		FailBody:            r.FailBody,
		CreatedAtMS:         time.Now().UnixMilli(),
	}
	event.EventHash = buildEventHash(event)
	return event
}

// parseRecordTimestamp accepts RFC3339 / RFC3339Nano, the formats Go's
// time.Time marshals to in JSON. Returns the zero time on anything else so
// callers fall back to "now".
func parseRecordTimestamp(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

// hashAPIKey mirrors CPA-Manager-Plus's apiKey hashing: SHA-256 of the raw
// client API key so the dashboard can group by caller without storing the
// secret.
func hashAPIKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}