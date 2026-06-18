// Package apikeyalias answers /api-key-aliases*. The routes let the dashboard
// list, replace and delete the labels mapped to api key hashes. Authentication
// is handled by the host before the request reaches the plugin.
package apikeyalias

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/repository/apikeyalias"
	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/store"
)

// Handler answers requests under /api-key-aliases*.
type Handler struct {
	Store *store.Store
}

// saveRequest mirrors the front-end PUT body (camelCase keys).
type saveRequest struct {
	Items                  []apikeyalias.APIKeyAlias `json:"items"`
	ActiveAPIKeyHashes     []string                  `json:"activeApiKeyHashes,omitempty"`
	AllowOrphanAliasCleanup bool                     `json:"allowOrphanAliasCleanup,omitempty"`
}

// Handle dispatches GET (list), PUT (replace), DELETE /api-key-aliases/{hash}.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimRight(r.URL.Path, "/")
	const basePath = "/api-key-aliases"
	switch {
	case path == basePath && r.Method == http.MethodGet:
		h.list(w, r)
	case path == basePath && r.Method == http.MethodPut:
		h.save(w, r)
	case strings.HasPrefix(path, basePath+"/") && r.Method == http.MethodDelete:
		apiKeyHash := strings.TrimPrefix(path, basePath+"/")
		h.delete(w, r, apiKeyHash)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	aliases, err := h.Store.LoadAPIKeyAliases(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": aliases})
}

func (h *Handler) save(w http.ResponseWriter, r *http.Request) {
	var req saveRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		writeError(w, http.StatusBadRequest, errDecode.Error())
		return
	}
	if err := h.Store.UpsertAPIKeyAliases(r.Context(), req.Items, req.ActiveAPIKeyHashes, req.AllowOrphanAliasCleanup); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	aliases, err := h.Store.LoadAPIKeyAliases(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": aliases})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, apiKeyHash string) {
	if err := h.Store.DeleteAPIKeyAlias(r.Context(), apiKeyHash); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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