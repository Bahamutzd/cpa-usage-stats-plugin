// Package router builds the embedded http.Handler that the plugin uses to
// answer management.handle requests. Paths are relative to /v0/management
// because CLIProxyAPI strips that prefix before handing the request to the
// plugin's ManagementHandler.
package router

import (
	"net/http"

	apikeyaliasctl "github.com/Bahamutzd/cpa-usage-stats-plugin/internal/http/controller/apikeyalias"
	dashboardctl "github.com/Bahamutzd/cpa-usage-stats-plugin/internal/http/controller/dashboard"
	modelpricectl "github.com/Bahamutzd/cpa-usage-stats-plugin/internal/http/controller/modelprice"
	monitoringctl "github.com/Bahamutzd/cpa-usage-stats-plugin/internal/http/controller/monitoring"
	usagectl "github.com/Bahamutzd/cpa-usage-stats-plugin/internal/http/controller/usage"
	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/store"
)

// New returns the management API mux. The host has already authenticated the
// caller and stripped /v0/management, so handlers see /usage, /dashboard/summary,
// /monitoring/analytics, /api-key-aliases, /model-prices directly.
func New(st *store.Store) http.Handler {
	usageHandler := &usagectl.Handler{Store: st}
	dashboardHandler := &dashboardctl.Handler{Store: st}
	monitoringHandler := &monitoringctl.Handler{Store: st}
	apiKeyAliasHandler := &apikeyaliasctl.Handler{Store: st}
	modelPriceHandler := &modelpricectl.Handler{Store: st}

	mux := http.NewServeMux()
	mux.HandleFunc("/usage", usageHandler.Handle)
	mux.HandleFunc("/usage/", usageHandler.Handle)
	mux.HandleFunc("/dashboard/summary", dashboardHandler.Summary)
	mux.HandleFunc("/monitoring/analytics", monitoringHandler.Analytics)
	mux.HandleFunc("/api-key-aliases", apiKeyAliasHandler.Handle)
	mux.HandleFunc("/api-key-aliases/", apiKeyAliasHandler.Handle)
	mux.HandleFunc("/model-prices", modelPriceHandler.Handle)
	mux.HandleFunc("/model-prices/", modelPriceHandler.Handle)

	return notFoundFallback(mux)
}

// notFoundFallback turns the default ServeMux 404 (plaintext body) into a JSON
// 404 so the management UI surfaces it consistently.
func notFoundFallback(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)
		if pattern == "" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not_found"}`))
			return
		}
		mux.ServeHTTP(w, r)
	})
}