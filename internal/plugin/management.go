package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/http/router"
	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/webasset"
)

// managementBasePath and resourceBasePath must match the constants in
// CLIProxyAPI's pluginhost package so the plugin can strip the right prefix
// from the request path the host forwards.
const (
	managementBasePath = "/v0/management"
	resourceBasePath   = "/v0/resource/plugins/" + pluginID
)

// managementRoute mirrors pluginapi.ManagementRoute (Handler is bound by the
// host on the receiving side). All paths are relative to /v0/management/.
type managementRoute struct {
	Method      string `json:"Method"`
	Path        string `json:"Path"`
	Menu        string `json:"Menu,omitempty"`
	Description string `json:"Description,omitempty"`
}

type resourceRoute struct {
	Path        string `json:"Path"`
	Menu        string `json:"Menu,omitempty"`
	Description string `json:"Description,omitempty"`
}

// managementRegisterResult mirrors pluginhost.rpcManagementRegistrationResponse:
// the outer fields are lower-case (`routes`, `resources`), while inside each
// route the host expects pluginapi.ManagementRoute's PascalCase field names
// (Method, Path, Menu, ...) because that struct has no JSON tags.
type managementRegisterResult struct {
	Routes    []managementRoute `json:"routes,omitempty"`
	Resources []resourceRoute   `json:"resources,omitempty"`
}

// managementRequest mirrors pluginapi.ManagementRequest. The host also adds a
// host_callback_id field which we deliberately ignore.
type managementRequest struct {
	Method  string              `json:"Method"`
	Path    string              `json:"Path"`
	Headers map[string][]string `json:"Headers"`
	Query   map[string][]string `json:"Query"`
	Body    []byte              `json:"Body"`
}

// managementResponse mirrors pluginapi.ManagementResponse.
type managementResponse struct {
	StatusCode int                 `json:"StatusCode,omitempty"`
	Headers    map[string][]string `json:"Headers,omitempty"`
	Body       []byte              `json:"Body,omitempty"`
}

func handleManagementRegister(_ []byte) []byte {
	// CLIProxyAPI matches plugin management routes by exact "METHOD fullPath"
	// (see pluginhost.ServeManagementHTTP + managementRouteKey). Path parameters
	// are not supported — the path cannot contain ":" or "*". So endpoints that
	// CPA-Manager-Plus served with a path param (e.g. DELETE
	// /api-key-aliases/{hash}) are intentionally NOT registered here: the
	// front-end must delete a single alias by editing the list and PUT-ing the
	// whole table back. All routes below are exact, parameter-free paths.
	return OkEnvelope(managementRegisterResult{
		Routes: []managementRoute{
			{Method: "GET", Path: "/usage"},
			{Method: "GET", Path: "/usage/export"},
			{Method: "POST", Path: "/usage/import"},
			{Method: "GET", Path: "/dashboard/summary"},
			{Method: "POST", Path: "/monitoring/analytics"},
			{Method: "GET", Path: "/monitoring/analytics"},
			{Method: "GET", Path: "/api-key-aliases"},
			{Method: "PUT", Path: "/api-key-aliases"},
			{Method: "GET", Path: "/model-prices"},
			{Method: "PUT", Path: "/model-prices"},
			{Method: "POST", Path: "/model-prices/sync"},
		},
		Resources: []resourceRoute{
			{
				// Path cannot be "/" — pluginhost.normalizeResourceRoute does
				// strings.TrimRight(path, "/") which turns "/" into "" and then
				// rejects it, so the resource route never registers and the
				// management UI shows no menu entry. Use "/index" instead; the
				// menu link opens /v0/resource/plugins/cpa-usage-stats/index and
				// the handler still serves the SPA (it matches by resource base
				// prefix, not the exact trailing segment).
				// The SPA is a single self-contained HTML file (inlined JS/CSS)
				// with a hash router so navigation never issues new sub-path
				// requests — exact-matched resource routes can't serve a tree.
				Path:        "/index",
				Menu:        "Usage Stats",
				Description: "CPA-Manager-Plus style usage statistics dashboard.",
			},
		},
	})
}

func handleManagementHandle(payload []byte) []byte {
	var req managementRequest
	if errDecode := json.Unmarshal(payload, &req); errDecode != nil {
		return ErrorEnvelope("decode_failed", errDecode.Error())
	}

	// Resource requests (the SPA at /v0/resource/plugins/cpa-usage-stats/) are
	// not management-authenticated and are served the bundled HTML directly.
	if strings.HasPrefix(req.Path, resourceBasePath) {
		return OkEnvelope(managementResponse{
			StatusCode: http.StatusOK,
			Headers:    map[string][]string{"Content-Type": {"text/html; charset=utf-8"}},
			Body:       webasset.Index(),
		})
	}

	st := currentStore()
	if st == nil {
		return OkEnvelope(managementResponse{
			StatusCode: http.StatusServiceUnavailable,
			Headers:    map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}},
			Body:       []byte("plugin not initialized"),
		})
	}

	// The host forwards the full /v0/management/<path>. Strip the prefix so
	// the embedded mux can match its parameter-free patterns (/usage, ...).
	relativePath := strings.TrimPrefix(req.Path, managementBasePath)
	if relativePath == "" {
		relativePath = "/"
	}
	req.Path = relativePath

	httpReq := buildHTTPRequest(req)
	recorder := newResponseRecorder()
	router.New(st, currentCache()).ServeHTTP(recorder, httpReq)

	return OkEnvelope(managementResponse{
		StatusCode: recorder.status,
		Headers:    recorder.header,
		Body:       recorder.body.Bytes(),
	})
}

// buildHTTPRequest synthesizes an *http.Request from the host-supplied
// ManagementRequest so the embedded mux can serve it without changes. The path
// has already been stripped of the /v0/management prefix by the caller.
func buildHTTPRequest(req managementRequest) *http.Request {
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	requestURL := &url.URL{Path: req.Path}
	if len(req.Query) > 0 {
		requestURL.RawQuery = url.Values(req.Query).Encode()
	}
	body := emptyReader()
	if len(req.Body) > 0 {
		body = bytesReader(req.Body)
	}
	httpReq, errReq := http.NewRequestWithContext(context.Background(), method, requestURL.String(), body)
	if errReq != nil || httpReq == nil {
		// http.NewRequestWithContext only errors on malformed method/URL; we
		// have already normalized both, so a real failure here is unreachable.
		// Fall back to a bare GET so the caller sees a 404 instead of panicking.
		httpReq, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	}
	for name, values := range req.Headers {
		for _, value := range values {
			httpReq.Header.Add(name, value)
		}
	}
	return httpReq
}