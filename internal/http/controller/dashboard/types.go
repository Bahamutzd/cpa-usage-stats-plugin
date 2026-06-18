package dashboard

// Response types for /dashboard/summary. JSON tags mirror the front-end
// DashboardSummary* interfaces. Optional chart blocks (today_request_health_timeline,
// model_cost_rank) are pointers so they can be omitted when not computed.

type summaryResponse struct {
	GeneratedAtMS               int64                          `json:"generated_at_ms"`
	Window                      summaryWindow                  `json:"window"`
	Today                       todaySummary                  `json:"today"`
	Rolling30M                  rollingSummary                 `json:"rolling_30m"`
	TopModelsToday              []topModel                     `json:"top_models_today"`
	ModelCostRank               []modelCostRank                `json:"model_cost_rank,omitempty"`
	TrafficTimeline             []trafficPoint                 `json:"traffic_timeline,omitempty"`
	HourlyActivity              []hourlyActivityPoint          `json:"hourly_activity,omitempty"`
	TodayRequestHealthTimeline  *todayRequestHealthTimeline    `json:"today_request_health_timeline,omitempty"`
	TokenMix                    []tokenMixSegment              `json:"token_mix,omitempty"`
	ChannelHealth               []channelHealth                `json:"channel_health,omitempty"`
	FailureSources              []failureSource                `json:"failure_sources,omitempty"`
	RecentFailures              []recentFailure                `json:"recent_failures"`
}

type summaryWindow struct {
	TodayStartMS       int64 `json:"today_start_ms"`
	NowMS              int64 `json:"now_ms"`
	Rolling30MStartMS  int64 `json:"rolling_30m_start_ms"`
}

type todaySummary struct {
	TotalCalls          int64    `json:"total_calls"`
	SuccessCalls        int64    `json:"success_calls"`
	FailureCalls        int64    `json:"failure_calls"`
	SuccessRate         float64  `json:"success_rate"`
	InputTokens         int64    `json:"input_tokens"`
	OutputTokens        int64    `json:"output_tokens"`
	CachedTokens        int64    `json:"cached_tokens"`
	CacheReadTokens     int64    `json:"cache_read_tokens"`
	CacheCreationTokens int64    `json:"cache_creation_tokens"`
	ReasoningTokens     int64    `json:"reasoning_tokens"`
	TotalTokens         int64    `json:"total_tokens"`
	TotalCost           float64  `json:"total_cost"`
	AverageLatencyMS    *float64 `json:"average_latency_ms"`
	ZeroTokenCalls      int64    `json:"zero_token_calls"`
}

type rollingSummary struct {
	RPM        float64 `json:"rpm"`
	TPM        float64 `json:"tpm"`
	TotalCalls int64   `json:"total_calls"`
	TotalTokens int64  `json:"total_tokens"`
}

type topModel struct {
	Model       string  `json:"model"`
	Calls       int64   `json:"calls"`
	Tokens      int64   `json:"tokens"`
	Cost        float64 `json:"cost"`
	SuccessRate float64 `json:"success_rate"`
}

type modelCostRank struct {
	Model       string  `json:"model"`
	Calls       int64   `json:"calls"`
	Tokens      int64   `json:"tokens"`
	Cost        float64 `json:"cost"`
	SuccessRate float64 `json:"success_rate"`
	CostShare   float64 `json:"cost_share"`
}

type trafficPoint struct {
	BucketMS    int64   `json:"bucket_ms"`
	Calls       int64   `json:"calls"`
	Tokens      int64   `json:"tokens"`
	Success     int64   `json:"success"`
	Failure     int64   `json:"failure"`
	CallsShare  float64 `json:"calls_share"`
	TokensShare float64 `json:"tokens_share"`
	FailureRate float64 `json:"failure_rate"`
}

type hourlyActivityPoint struct {
	HourIndex int64   `json:"hour_index"`
	BucketMS  int64   `json:"bucket_ms"`
	Calls     int64   `json:"calls"`
	Tokens    int64   `json:"tokens"`
	Intensity float64 `json:"intensity"`
}

type todayRequestHealthTimelinePoint struct {
	BucketMS    int64   `json:"bucket_ms"`
	Calls       int64   `json:"calls"`
	Tokens      int64   `json:"tokens"`
	Success     int64   `json:"success"`
	Failure     int64   `json:"failure"`
	SuccessRate float64 `json:"success_rate"`
	FailureRate float64 `json:"failure_rate"`
	Tone        string  `json:"tone"`
	Intensity   float64 `json:"intensity"`
	Future      bool    `json:"future"`
}

type todayRequestHealthTimeline struct {
	FromMS       int64                              `json:"from_ms"`
	ToMS         int64                              `json:"to_ms"`
	BucketMS     int64                              `json:"bucket_ms"`
	SuccessCalls int64                              `json:"success_calls"`
	FailureCalls int64                              `json:"failure_calls"`
	TotalCalls   int64                              `json:"total_calls"`
	SuccessRate  float64                            `json:"success_rate"`
	Points       []todayRequestHealthTimelinePoint  `json:"points"`
}

type tokenMixSegment struct {
	Key    string  `json:"key"`
	Tokens int64   `json:"tokens"`
	Share  float64 `json:"share"`
}

type channelHealth struct {
	AuthIndex           string   `json:"auth_index"`
	AuthLabel           string   `json:"auth_label,omitempty"`
	Account             string   `json:"account,omitempty"`
	Channel             string   `json:"channel,omitempty"`
	Source              string   `json:"source,omitempty"`
	AccountSnapshot     string   `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot   string   `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot string  `json:"auth_provider_snapshot,omitempty"`
	Calls               int64    `json:"calls"`
	Failures            int64    `json:"failures"`
	FailureRate         float64  `json:"failure_rate"`
	SuccessRate         float64  `json:"success_rate"`
	Tokens              int64    `json:"tokens"`
	Cost                float64  `json:"cost"`
	AverageLatencyMS    *float64 `json:"average_latency_ms"`
	Tone                string   `json:"tone"`
}

type failureSource struct {
	SourceHash           string   `json:"source_hash"`
	AuthIndex            string   `json:"auth_index"`
	AuthLabel            string   `json:"auth_label,omitempty"`
	Account              string   `json:"account,omitempty"`
	Channel              string   `json:"channel,omitempty"`
	Source               string   `json:"source,omitempty"`
	AccountSnapshot      string   `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot    string   `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot string   `json:"auth_provider_snapshot,omitempty"`
	Calls                int64    `json:"calls"`
	Failures             int64    `json:"failures"`
	FailureRate          float64  `json:"failure_rate"`
	LastSeenMS           int64    `json:"last_seen_ms"`
	AverageLatencyMS    *float64 `json:"average_latency_ms"`
}

type recentFailure struct {
	TimestampMS           int64   `json:"timestamp_ms"`
	Model                 string  `json:"model"`
	APIKeyHash            string  `json:"api_key_hash"`
	Source                string  `json:"source,omitempty"`
	SourceHash            string  `json:"source_hash"`
	AuthIndex             string  `json:"auth_index"`
	AuthLabel             string  `json:"auth_label,omitempty"`
	Account               string  `json:"account,omitempty"`
	Channel               string  `json:"channel,omitempty"`
	AccountSnapshot       string  `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot     string  `json:"auth_label_snapshot,omitempty"`
	AuthProviderSnapshot  string  `json:"auth_provider_snapshot,omitempty"`
	Endpoint              string  `json:"endpoint"`
	LatencyMS             *int64  `json:"latency_ms"`
	FailStatusCode        *int    `json:"fail_status_code,omitempty"`
	FailSummary           string  `json:"fail_summary,omitempty"`
}