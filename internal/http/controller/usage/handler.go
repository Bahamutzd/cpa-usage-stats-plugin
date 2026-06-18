// Package usage hosts the /usage* management API. It mirrors the
// CPA-Manager-Plus endpoints the dashboard's "recent usage" panel and the
// JSONL export/import flows rely on. Authentication is handled by the host
// (management routes are management-authenticated before the plugin sees the
// request), so no extra auth middleware is needed here.
package usage

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/store"
	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/usage"
)

const maxUsageImportBytes int64 = 64 * 1024 * 1024

// Handler answers requests under /usage*.
type Handler struct {
	Store *store.Store
}

// Handle dispatches GET /usage, GET /usage/export, POST /usage/import. The
// import parser (usage.ParseImportPayload) supports the JSONL format this
// plugin's own export writes, plus the two legacy CPA-Manager-Plus formats.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if strings.HasSuffix(r.URL.Path, "/export") {
			h.export(w, r)
			return
		}
		h.recent(w, r)
	case http.MethodPost:
		if strings.HasSuffix(r.URL.Path, "/import") {
			h.importJSONL(w, r)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
	}
}

// recent returns the CPA-Manager-Plus compatible usage payload. The front-end
// reads total_requests / success_count / failure_count / total_tokens / apis.
func (h *Handler) recent(w http.ResponseWriter, r *http.Request) {
	limit := 50000
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 50000 {
			limit = parsed
		}
	}
	events, errList := h.Store.RecentEvents(r.Context(), limit)
	if errList != nil {
		writeError(w, http.StatusInternalServerError, errList.Error())
		return
	}
	writeJSON(w, http.StatusOK, usage.BuildPayload(events))
}

// export streams the redacted JSONL dump. Content-Disposition matches the
// original CPA-Manager-Plus handler so browsers reuse the same filename.
func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	data, errExport := h.Store.ExportJSONL(r.Context())
	if errExport != nil {
		writeError(w, http.StatusInternalServerError, errExport.Error())
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Content-Disposition", `attachment; filename="usage-events.jsonl"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// importResult mirrors CPA-Manager-Plus's service/usage.ImportResult. The
// front-end UsageImportResponse reads these exact field names.
type importResult struct {
	Format      string   `json:"format"`
	Added       int      `json:"added"`
	Skipped     int      `json:"skipped"`
	Total       int      `json:"total"`
	Failed      int      `json:"failed"`
	Unsupported int      `json:"unsupported"`
	Warnings    []string `json:"warnings"`
}

// importJSONL accepts the JSONL export (or legacy formats) and inserts the
// events. Parse failures that still produced zero events surface as a 400
// with diagnostic fields so the front-end can show what went wrong; a parse
// error after partial events is a 500.
func (h *Handler) importJSONL(w http.ResponseWriter, r *http.Request) {
	body := http.MaxBytesReader(w, r.Body, maxUsageImportBytes)
	data, errRead := io.ReadAll(body)
	if errRead != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(errRead, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, errRead.Error())
			return
		}
		writeError(w, http.StatusBadRequest, errRead.Error())
		return
	}
	parsed, errParse := usage.ParseImportPayload(data)
	if errParse != nil && len(parsed.Events) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":       errParse.Error(),
			"format":      parsed.Format,
			"failed":      parsed.Failed,
			"unsupported": parsed.Unsupported,
			"warnings":    parsed.Warnings,
		})
		return
	}
	result, errInsert := h.Store.InsertEvents(r.Context(), parsed.Events)
	if errInsert != nil {
		writeError(w, http.StatusInternalServerError, errInsert.Error())
		return
	}
	writeJSON(w, http.StatusOK, importResult{
		Format:      parsed.Format,
		Added:       result.Inserted,
		Skipped:     result.Skipped,
		Total:       len(parsed.Events),
		Failed:      parsed.Failed,
		Unsupported: parsed.Unsupported,
		Warnings:    parsed.Warnings,
	})
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