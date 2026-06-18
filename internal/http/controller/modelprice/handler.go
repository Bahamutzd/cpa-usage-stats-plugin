// Package modelprice answers /model-prices*. GET lists, PUT replaces the
// whole table, POST /model-prices/sync pulls from the LiteLLM catalog and
// upserts. Authentication is handled by the host.
package modelprice

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/repository/modelprice"
	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/store"
)

// Handler answers requests under /model-prices*.
type Handler struct {
	Store *store.Store
}

// Handle dispatches by method and path.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimRight(r.URL.Path, "/")
	switch {
	case path == "/model-prices" && r.Method == http.MethodGet:
		h.list(w, r)
	case path == "/model-prices" && r.Method == http.MethodPut:
		h.replace(w, r)
	case path == "/model-prices/sync" && r.Method == http.MethodPost:
		h.sync(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	prices, err := h.Store.LoadModelPrices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"prices": prices})
}

func (h *Handler) replace(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prices map[string]modelprice.ModelPrice `json:"prices"`
	}
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		writeError(w, http.StatusBadRequest, errDecode.Error())
		return
	}
	if err := h.Store.ReplaceModelPrices(r.Context(), req.Prices); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	prices, err := h.Store.LoadModelPrices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"prices": prices})
}

func (h *Handler) sync(w http.ResponseWriter, r *http.Request) {
	// The front-end sends an empty body or a JSON object; we currently only
	// support the LiteLLM source, so the body is drained and ignored.
	if r.Body != nil {
		if _, errDrain := io.Copy(io.Discard, r.Body); errDrain != nil && !errors.Is(errDrain, io.EOF) {
			writeError(w, http.StatusBadRequest, errDrain.Error())
			return
		}
	}
	prices, skipped, errFetch := modelprice.FetchLiteLLM(r.Context(), "", nil)
	if errFetch != nil {
		writeError(w, http.StatusBadGateway, errFetch.Error())
		return
	}
	result, errUpsert := h.Store.UpsertSyncedModelPrices(r.Context(), prices)
	if errUpsert != nil {
		writeError(w, http.StatusInternalServerError, errUpsert.Error())
		return
	}
	latest, _ := h.Store.LoadModelPrices(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"imported": result.Imported,
		"skipped":  result.Skipped + skipped,
		"prices":   latest,
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