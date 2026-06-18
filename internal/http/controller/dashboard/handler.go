package dashboard

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/cache"
	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/store"
)

// Handler owns endpoints under /dashboard/*.
type Handler struct {
	Store *store.Store
	Cache *cache.Store
}

// summaryParams mirrors DashboardSummaryParams.
type summaryParams struct {
	TodayStartMS   int64
	NowMS          int64
	TopModels      int
	RecentFailures int
}

// Summary handles GET /dashboard/summary. Results are cached for 15 seconds
// to avoid re-running 9 SQLite aggregation queries on every 30s auto-refresh.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	params, errParse := parseSummaryParams(r)
	if errParse != nil {
		writeError(w, http.StatusBadRequest, errParse.Error())
		return
	}

	cacheKey := "dash:" + strconv.FormatInt(params.TodayStartMS, 10)
	if h.Cache != nil {
		if cached, ok := h.Cache.Get(cacheKey); ok {
			writeJSON(w, http.StatusOK, cached)
			return
		}
	}

	resp := buildSummary(r, h.Store, params)

	if h.Cache != nil {
		h.Cache.Set(cacheKey, resp, 15*1000)
	}
	writeJSON(w, http.StatusOK, resp)
}

func parseSummaryParams(r *http.Request) (summaryParams, error) {
	query := r.URL.Query()
	todayStartRaw := strings.TrimSpace(query.Get("today_start_ms"))
	if todayStartRaw == "" {
		return summaryParams{}, errSummary("today_start_ms is required")
	}
	todayStartMS, err := strconv.ParseInt(todayStartRaw, 10, 64)
	if err != nil || todayStartMS <= 0 {
		return summaryParams{}, errSummary("today_start_ms must be a positive integer")
	}
	nowMS, err := readOptionalInt64(query.Get("now_ms"))
	if err != nil {
		return summaryParams{}, err
	}
	if nowMS <= 0 {
		nowMS = timeNowMS()
	}
	topModels, err := readOptionalInt(query.Get("top_models"))
	if err != nil {
		return summaryParams{}, err
	}
	if topModels <= 0 {
		topModels = 5
	}
	recentFailures, err := readOptionalInt(query.Get("recent_failures"))
	if err != nil {
		return summaryParams{}, err
	}
	if recentFailures <= 0 {
		recentFailures = 10
	}
	return summaryParams{
		TodayStartMS:   todayStartMS,
		NowMS:          nowMS,
		TopModels:      topModels,
		RecentFailures: recentFailures,
	}, nil
}

type summaryError string

func (e summaryError) Error() string { return string(e) }

func errSummary(message string) error { return summaryError(message) }

func readOptionalInt64(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, errSummary("now_ms must be an integer")
	}
	return parsed, nil
}

func readOptionalInt(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, errSummary("must be an integer")
	}
	return parsed, nil
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