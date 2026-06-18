package monitoring

// Response types for /monitoring/analytics. JSON tags mirror the front-end
// MonitoringAnalytics* interfaces verbatim so the React client can consume
// the payload without translation. Cost fields are zero in this build because
// the model-price overlay ships in a later batch.

type analyticsResponse struct {
	GeneratedAtMS       int64                `json:"generated_at_ms"`
	Granularity         string               `json:"granularity"`
	Summary             *summaryRow          `json:"summary,omitempty"`
	Timeline            []timelineRow        `json:"timeline,omitempty"`
	HourlyDistribution  []hourlyRow          `json:"hourly_distribution,omitempty"`
	ModelShare          []modelShareRow      `json:"model_share,omitempty"`
	ModelStats          []modelStatRow       `json:"model_stats,omitempty"`
	ChannelShare        []channelShareRow    `json:"channel_share,omitempty"`
	FailureSources      []failureSourceRow   `json:"failure_sources,omitempty"`
	AccountStats        []accountStatRow     `json:"account_stats,omitempty"`
	APIKeyStats         []apiKeyStatRow      `json:"api_key_stats,omitempty"`
	FilterOptions       *filterOptions       `json:"filter_options,omitempty"`
	TaskBuckets         []taskBucketRow      `json:"task_buckets,omitempty"`
	RecentFailures      []recentFailureRow   `json:"recent_failures,omitempty"`
	Events              *eventsResponse      `json:"events,omitempty"`
}

type summaryRow struct {
	TotalCalls            int64    `json:"total_calls"`
	SuccessCalls          int64    `json:"success_calls"`
	FailureCalls          int64    `json:"failure_calls"`
	SuccessRate           float64  `json:"success_rate"`
	InputTokens           int64    `json:"input_tokens"`
	OutputTokens          int64    `json:"output_tokens"`
	CachedTokens          int64    `json:"cached_tokens"`
	CacheReadTokens       int64    `json:"cache_read_tokens"`
	CacheCreationTokens   int64    `json:"cache_creation_tokens"`
	ReasoningTokens       int64    `json:"reasoning_tokens"`
	TotalTokens           int64    `json:"total_tokens"`
	TotalCost             float64  `json:"total_cost"`
	AverageLatencyMS      *float64 `json:"average_latency_ms"`
	ZeroTokenCalls        int64    `json:"zero_token_calls"`
	RPM30M                float64  `json:"rpm_30m"`
	TPM30M                float64  `json:"tpm_30m"`
	AvgDailyRequests      float64  `json:"avg_daily_requests"`
	AvgDailyTokens        float64  `json:"avg_daily_tokens"`
	ApproxTasks           int64    `json:"approx_tasks"`
	ApproxTaskFailures    int64    `json:"approx_task_failures"`
	ApproxTaskSuccessRate float64  `json:"approx_task_success_rate"`
	ZeroTokenModels       []string `json:"zero_token_models"`
}

type timelineRow struct {
	BucketMS int64   `json:"bucket_ms"`
	Label    string  `json:"label"`
	Calls    int64   `json:"calls"`
	Tokens   int64   `json:"tokens"`
	Success  int64   `json:"success"`
	Failure  int64   `json:"failure"`
}

type hourlyRow struct {
	Hour   int   `json:"hour"`
	Calls  int64 `json:"calls"`
	Tokens int64 `json:"tokens"`
}

type modelShareRow struct {
	Model  string  `json:"model"`
	Calls  int64   `json:"calls"`
	Tokens int64   `json:"tokens"`
	Cost   float64 `json:"cost"`
}

type modelStatRow struct {
	Model               string  `json:"model"`
	Calls               int64   `json:"calls"`
	SuccessCalls        int64   `json:"success_calls"`
	FailureCalls        int64   `json:"failure_calls"`
	SuccessRate         float64 `json:"success_rate"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CachedTokens        int64   `json:"cached_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	TotalTokens         int64   `json:"total_tokens"`
	Cost                float64 `json:"cost"`
}

type channelShareRow struct {
	AuthIndex           string   `json:"auth_index"`
	Source              string   `json:"source,omitempty"`
	AccountSnapshot     string   `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot   string   `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot string  `json:"auth_provider_snapshot,omitempty"`
	Calls               int64    `json:"calls"`
	Success             int64    `json:"success"`
	Failure             int64    `json:"failure"`
	Tokens              int64    `json:"tokens"`
	Cost                float64  `json:"cost"`
	AverageLatencyMS    *float64 `json:"average_latency_ms"`
}

type failureSourceRow struct {
	Source               string   `json:"source,omitempty"`
	SourceHash           string   `json:"source_hash"`
	AuthIndex            string   `json:"auth_index"`
	AccountSnapshot      string   `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot    string   `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot string   `json:"auth_provider_snapshot,omitempty"`
	Calls                int64    `json:"calls"`
	Failure              int64    `json:"failure"`
	LastSeenMS           int64    `json:"last_seen_ms"`
	AverageLatencyMS     *float64 `json:"average_latency_ms"`
}

type accountModelStatRow struct {
	Model               string  `json:"model"`
	Calls               int64   `json:"calls"`
	SuccessCalls         int64   `json:"success_calls"`
	FailureCalls         int64   `json:"failure_calls"`
	SuccessRate          float64 `json:"success_rate"`
	InputTokens          int64   `json:"input_tokens"`
	OutputTokens         int64   `json:"output_tokens"`
	CachedTokens         int64   `json:"cached_tokens"`
	CacheReadTokens      int64   `json:"cache_read_tokens"`
	CacheCreationTokens  int64   `json:"cache_creation_tokens"`
	TotalTokens          int64   `json:"total_tokens"`
	Cost                 float64 `json:"cost"`
	LastSeenMS           int64   `json:"last_seen_ms"`
}

type accountStatRow struct {
	ID                  string                 `json:"id"`
	AccountSnapshot     string                 `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot   string                 `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot string                 `json:"auth_provider_snapshot,omitempty"`
	AuthIndices         []string               `json:"auth_indices,omitempty"`
	Sources             []string               `json:"sources,omitempty"`
	SourceHashes        []string               `json:"source_hashes,omitempty"`
	Calls               int64                  `json:"calls"`
	SuccessCalls        int64                  `json:"success_calls"`
	FailureCalls        int64                  `json:"failure_calls"`
	SuccessRate         float64                `json:"success_rate"`
	InputTokens         int64                  `json:"input_tokens"`
	OutputTokens         int64                  `json:"output_tokens"`
	CachedTokens        int64                  `json:"cached_tokens"`
	CacheReadTokens     int64                  `json:"cache_read_tokens"`
	CacheCreationTokens int64                  `json:"cache_creation_tokens"`
	TotalTokens         int64                  `json:"total_tokens"`
	Cost                float64                `json:"cost"`
	AverageLatencyMS    *float64               `json:"average_latency_ms"`
	LastSeenMS          int64                  `json:"last_seen_ms"`
	Models              []accountModelStatRow  `json:"models,omitempty"`
}

type apiKeyStatRow struct {
	ID                  string                `json:"id"`
	APIKeyHash          string                `json:"api_key_hash"`
	AccountSnapshot     string                `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot   string                `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot string                `json:"auth_provider_snapshot,omitempty"`
	AuthIndices         []string              `json:"auth_indices,omitempty"`
	Sources             []string              `json:"sources,omitempty"`
	SourceHashes        []string              `json:"source_hashes,omitempty"`
	Calls               int64                 `json:"calls"`
	SuccessCalls        int64                 `json:"success_calls"`
	FailureCalls        int64                 `json:"failure_calls"`
	SuccessRate         float64               `json:"success_rate"`
	InputTokens         int64                 `json:"input_tokens"`
	OutputTokens        int64                 `json:"output_tokens"`
	CachedTokens        int64                 `json:"cached_tokens"`
	CacheReadTokens     int64                 `json:"cache_read_tokens"`
	CacheCreationTokens int64                 `json:"cache_creation_tokens"`
	TotalTokens         int64                 `json:"total_tokens"`
	Cost                float64               `json:"cost"`
	AverageLatencyMS   *float64              `json:"average_latency_ms"`
	LastSeenMS          int64                 `json:"last_seen_ms"`
	Models              []accountModelStatRow  `json:"models,omitempty"`
}

type filterOptions struct {
	AccountStats []accountStatRow  `json:"account_stats,omitempty"`
	APIKeyStats  []apiKeyStatRow    `json:"api_key_stats,omitempty"`
	ChannelShare []channelShareRow  `json:"channel_share,omitempty"`
	ModelStats   []modelStatRow     `json:"model_stats,omitempty"`
}

type taskBucketRow struct {
	BucketKey           string   `json:"bucket_key"`
	Total               int64    `json:"total"`
	Success             int64    `json:"success"`
	Failure             int64    `json:"failure"`
	FirstMS             int64    `json:"first_ms"`
	LastMS              int64    `json:"last_ms"`
	Source              string   `json:"source"`
	SourceHash          string   `json:"source_hash"`
	AuthIndex           string   `json:"auth_index"`
	Models              []string `json:"models"`
	Endpoints           []string `json:"endpoints"`
	InputTokens         int64    `json:"input_tokens"`
	OutputTokens        int64    `json:"output_tokens"`
	CachedTokens        int64    `json:"cached_tokens"`
	CacheReadTokens     int64    `json:"cache_read_tokens"`
	CacheCreationTokens int64    `json:"cache_creation_tokens"`
	TotalTokens         int64    `json:"total_tokens"`
	AverageLatencyMS    *float64 `json:"average_latency_ms"`
	MaxLatencyMS        *int64   `json:"max_latency_ms"`
}

type recentFailureRow struct {
	TimestampMS           int64   `json:"timestamp_ms"`
	Model                 string  `json:"model"`
	APIKeyHash            string  `json:"api_key_hash"`
	Source                string  `json:"source,omitempty"`
	SourceHash            string  `json:"source_hash"`
	AuthIndex             string  `json:"auth_index"`
	AccountSnapshot       string  `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot     string  `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot  string  `json:"auth_provider_snapshot,omitempty"`
	AuthProjectIDSnapshot string  `json:"auth_project_id_snapshot,omitempty"`
	Endpoint              string  `json:"endpoint"`
	DurationMS            *int64  `json:"duration_ms"`
	FailStatusCode        *int    `json:"fail_status_code"`
	FailSummary            string  `json:"fail_summary,omitempty"`
}

type eventRow struct {
	EventHash             string `json:"event_hash"`
	TimestampMS           int64  `json:"timestamp_ms"`
	Model                 string `json:"model"`
	Endpoint              string `json:"endpoint"`
	Method                string `json:"method"`
	Path                  string `json:"path"`
	AuthIndex             string `json:"auth_index"`
	Source                string `json:"source"`
	SourceHash            string `json:"source_hash"`
	APIKeyHash            string `json:"api_key_hash"`
	AccountSnapshot       string `json:"account_snapshot"`
	AuthLabelSnapshot     string `json:"auth_label_snapshot"`
	AuthProviderSnapshot  string `json:"auth_provider_snapshot"`
	AuthProjectIDSnapshot string `json:"auth_project_id_snapshot,omitempty"`
	ResolvedModel         string `json:"resolved_model,omitempty"`
	ReasoningEffort       string `json:"reasoning_effort,omitempty"`
	ServiceTier           string `json:"service_tier,omitempty"`
	ExecutorType          string `json:"executor_type,omitempty"`
	InputTokens           int64  `json:"input_tokens"`
	OutputTokens          int64  `json:"output_tokens"`
	CachedTokens          int64  `json:"cached_tokens"`
	CacheReadTokens       int64  `json:"cache_read_tokens"`
	CacheCreationTokens   int64  `json:"cache_creation_tokens"`
	ReasoningTokens       int64  `json:"reasoning_tokens"`
	TotalTokens           int64  `json:"total_tokens"`
	LatencyMS             *int64 `json:"latency_ms"`
	TTFTMS                *int64 `json:"ttft_ms,omitempty"`
	Failed                bool   `json:"failed"`
	FailStatusCode        *int   `json:"fail_status_code,omitempty"`
	FailSummary           string `json:"fail_summary,omitempty"`
}

type eventsResponse struct {
	Items        []eventRow `json:"items"`
	NextBeforeMS int64      `json:"next_before_ms"`
	NextBeforeID int64      `json:"next_before_id,omitempty"`
	HasMore      bool       `json:"has_more"`
	TotalCount   *int64     `json:"total_count,omitempty"`
}