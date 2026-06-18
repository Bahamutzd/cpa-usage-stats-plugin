// Package monitoring answers /monitoring/analytics. It assembles the
// MonitoringAnalyticsResponse the front-end MonitoringCenterPage renders.
// The shape mirrors CPA-Manager-Plus's service/monitoring output; cost fields
// are zero until model-price overlay lands in a later batch.
package monitoring

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/store"
)

// Handler owns endpoints under /monitoring/*.
type Handler struct {
	Store *store.Store
}

// analyticsRequest mirrors MonitoringAnalyticsRequest from the front-end.
type analyticsRequest struct {
	FromMS           int64                `json:"from_ms"`
	ToMS             int64                `json:"to_ms"`
	NowMS            int64                `json:"now_ms,omitempty"`
	SearchQuery      string               `json:"search_query,omitempty"`
	SearchAPIKeyHash string               `json:"search_api_key_hash,omitempty"`
	Filters          *analyticsFilters    `json:"filters,omitempty"`
	Include          *analyticsInclude    `json:"include,omitempty"`
}

type analyticsFilters struct {
	Models            []string `json:"models,omitempty"`
	Providers         []string `json:"providers,omitempty"`
	Accounts          []string `json:"accounts,omitempty"`
	AuthIndices       []string `json:"auth_indices,omitempty"`
	APIKeyHashes      []string `json:"api_key_hashes,omitempty"`
	SourceHashes      []string `json:"source_hashes,omitempty"`
	IncludeFailed     bool     `json:"include_failed,omitempty"`
	FailedOnly        bool     `json:"failed_only,omitempty"`
	ExcludeZeroTokens bool     `json:"exclude_zero_token,omitempty"`
}

type analyticsInclude struct {
	Summary            *bool                         `json:"summary,omitempty"`
	Timeline           *bool                         `json:"timeline,omitempty"`
	HourlyDistribution *bool                         `json:"hourly_distribution,omitempty"`
	ModelShare         *bool                         `json:"model_share,omitempty"`
	ChannelShare       *bool                         `json:"channel_share,omitempty"`
	ModelStats         *bool                         `json:"model_stats,omitempty"`
	FailureSources     *bool                         `json:"failure_sources,omitempty"`
	AccountStats       *bool                         `json:"account_stats,omitempty"`
	APIKeyStats        *bool                         `json:"api_key_stats,omitempty"`
	FilterOptions      *bool                         `json:"filter_options,omitempty"`
	TaskBuckets        *bool                         `json:"task_buckets,omitempty"`
	RecentFailures     *int                          `json:"recent_failures,omitempty"`
	EventsPage         *analyticsEventsPageRequest   `json:"events_page,omitempty"`
	Granularity        string                        `json:"granularity,omitempty"`
}

type analyticsEventsPageRequest struct {
	Limit     int64 `json:"limit,omitempty"`
	BeforeMS  int64 `json:"before_ms,omitempty"`
	BeforeID  int64 `json:"before_id,omitempty"`
}

// Analytics handles POST /monitoring/analytics. The original CPA-Manager-Plus
// endpoint also accepts GET with query params; we support POST for parity with
// the front-end which sends a JSON body.
func (h *Handler) Analytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	req, errParse := parseAnalyticsRequest(r)
	if errParse != nil {
		writeError(w, http.StatusBadRequest, errParse.Error())
		return
	}
	if req.FromMS <= 0 || req.ToMS <= 0 || req.ToMS <= req.FromMS {
		writeError(w, http.StatusBadRequest, "from_ms and to_ms must form a positive window")
		return
	}
	resp := buildAnalyticsResponse(r, h.Store, req)
	writeJSON(w, http.StatusOK, resp)
}

func parseAnalyticsRequest(r *http.Request) (analyticsRequest, error) {
	req := analyticsRequest{}
	if r.Method == http.MethodPost && r.Body != nil {
		if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
			return analyticsRequest{}, errDecode
		}
		return req, nil
	}
	// GET fallback: read everything from query string so curl works too.
	query := r.URL.Query
	req.FromMS = parseInt64Query(query().Get("from_ms"))
	req.ToMS = parseInt64Query(query().Get("to_ms"))
	req.NowMS = parseInt64Query(query().Get("now_ms"))
	req.SearchQuery = strings.TrimSpace(query().Get("search_query"))
	req.SearchAPIKeyHash = strings.TrimSpace(query().Get("search_api_key_hash"))
	return req, nil
}

func parseInt64Query(value string) int64 {
	parsed, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return parsed
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, errMarshal := json.Marshal(payload)
	if errMarshal != nil {
		writeError(w, http.StatusInternalServerError, errMarshal.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	body, _ := json.Marshal(map[string]string{"error": message})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}