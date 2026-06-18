package plugin

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/cache"
	"github.com/Bahamutzd/cpa-usage-stats-plugin/internal/store"
)

// pluginID matches the registry id and the dynamic library filename.
const pluginID = "cpa-usage-stats"

// metadata is the plugin manifest reported on plugin.register and
// plugin.reconfigure. The values mirror what CLIProxyAPI's plugin store and
// management UI display.
var metadata = pluginMetadata{
	Name:             "请求监控",
	Version:          "0.2.4",
	Author:           "router-for-me",
	GitHubRepository: "https://github.com/Bahamutzd/cpa-usage-stats-plugin",
	Logo:             "",
	ConfigFields: []configField{
		{
			Name:        "db_path",
			Type:        "string",
			Description: "SQLite 数据库路径。留空时默认使用 ./cpa-usage-stats.db。",
		},
		{
			Name:        "retention_days",
			Type:        "integer",
			Description: "事件保留天数。设为 0 表示不清理历史事件。",
		},
	},
}

// capabilities advertises what the plugin can do. usage_plugin enables
// usage.handle delivery; management_api enables management.register so the
// plugin can expose its dashboard API and HTML resource.
var capabilities = pluginCapabilities{
	UsagePlugin:   true,
	ManagementAPI: true,
}

// pluginMetadata mirrors pluginapi.Metadata. That struct has no JSON tags,
// so the host serialises it with PascalCase field names; we keep the same
// shape on the plugin side.
type pluginMetadata struct {
	Name             string        `json:"Name"`
	Version          string        `json:"Version"`
	Author           string        `json:"Author"`
	GitHubRepository string        `json:"GitHubRepository"`
	Logo             string        `json:"Logo,omitempty"`
	ConfigFields     []configField `json:"ConfigFields"`
}

// configField mirrors pluginapi.ConfigField (no JSON tags upstream).
type configField struct {
	Name        string `json:"Name"`
	Type        string `json:"Type"`
	Description string `json:"Description"`
}

// pluginCapabilities mirrors pluginhost.rpcCapabilities. Only the bits this
// plugin actually implements are exposed; the host treats unspecified keys
// as false.
type pluginCapabilities struct {
	UsagePlugin   bool `json:"usage_plugin"`
	ManagementAPI bool `json:"management_api"`
}

type registerResult struct {
	SchemaVersion int                `json:"schema_version"`
	Metadata      pluginMetadata     `json:"metadata"`
	Capabilities  pluginCapabilities `json:"capabilities"`
}

// lifecycleRequest mirrors pluginhost.rpcLifecycleRequest. ConfigYAML carries
// the raw YAML bytes of the plugins.configs.cpa-usage-stats subtree (base64
// encoded by encoding/json because []byte uses base64 in JSON).
type lifecycleRequest struct {
	ConfigYAML    []byte `json:"config_yaml"`
	SchemaVersion uint32 `json:"schema_version"`
}

type runtimeConfig struct {
	DBPath        string
	RetentionDays int
}

func handleRegister(payload []byte) []byte {
	cfg := decodeLifecycleRequest(payload)
	if errBoot := bootstrap(cfg); errBoot != nil {
		return ErrorEnvelope("init_failed", errBoot.Error())
	}
	return OkEnvelope(registerResult{
		SchemaVersion: 1,
		Metadata:      metadata,
		Capabilities:  capabilities,
	})
}

func handleReconfigure(payload []byte) []byte {
	cfg := decodeLifecycleRequest(payload)
	if errBoot := bootstrap(cfg); errBoot != nil {
		return ErrorEnvelope("reconfigure_failed", errBoot.Error())
	}
	return OkEnvelope(registerResult{
		SchemaVersion: 1,
		Metadata:      metadata,
		Capabilities:  capabilities,
	})
}

func handleShutdown() []byte {
	Shutdown()
	return OkEnvelope(nil)
}

// decodeLifecycleRequest parses the host-supplied register/reconfigure payload
// and pulls plugin-specific options out of the raw YAML config. Missing or
// malformed fields fall back to defaults so the plugin still boots — the
// host only sees a hard failure if the local SQLite database can't be opened.
func decodeLifecycleRequest(payload []byte) runtimeConfig {
	cfg := runtimeConfig{RetentionDays: 90}
	if len(payload) == 0 {
		return cfg
	}
	var req lifecycleRequest
	if errDecode := json.Unmarshal(payload, &req); errDecode != nil {
		return cfg
	}
	if len(req.ConfigYAML) == 0 {
		return cfg
	}
	configMap := map[string]any{}
	if errYAML := yaml.Unmarshal(req.ConfigYAML, &configMap); errYAML != nil {
		return cfg
	}
	if path, ok := stringField(configMap, "db_path"); ok && path != "" {
		cfg.DBPath = path
	}
	if days, ok := intField(configMap, "retention_days"); ok {
		cfg.RetentionDays = days
	}
	return cfg
}

func stringField(m map[string]any, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	value, ok := m[key]
	if !ok {
		return "", false
	}
	if s, isString := value.(string); isString {
		return strings.TrimSpace(s), true
	}
	return "", false
}

func intField(m map[string]any, key string) (int, bool) {
	if m == nil {
		return 0, false
	}
	value, ok := m[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	}
	return 0, false
}

// runtimeState holds the long-lived plugin objects: DB, store, and the
// retention goroutine's cancel func. It is recreated on every register/
// reconfigure call so config changes take effect.
type runtimeState struct {
	cfg    runtimeConfig
	store  *store.Store
	cache  *cache.Store
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func (s *runtimeState) Close() {
	if s == nil {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	if s.store != nil {
		_ = s.store.Close()
	}
}

func bootstrap(cfg runtimeConfig) error {
	stateMu.Lock()
	defer stateMu.Unlock()
	if state != nil {
		state.Close()
		state = nil
	}
	dbPath := cfg.DBPath
	if dbPath == "" {
		dbPath = filepath.Clean("cpa-usage-stats.db")
	}
	st, errOpen := store.Open(dbPath)
	if errOpen != nil {
		return errOpen
	}
	ctx, cancel := context.WithCancel(context.Background())
	next := &runtimeState{cfg: cfg, store: st, cache: cache.New(), cancel: cancel}
	if cfg.RetentionDays > 0 {
		next.wg.Add(1)
		go runRetention(ctx, &next.wg, st, cfg.RetentionDays)
	}
	state = next
	return nil
}

func currentStore() *store.Store {
	stateMu.Lock()
	defer stateMu.Unlock()
	if state == nil {
		return nil
	}
	return state.store
}

func currentCache() *cache.Store {
	stateMu.Lock()
	defer stateMu.Unlock()
	if state == nil {
		return nil
	}
	return state.cache
}

// runRetention periodically deletes events older than retentionDays. The
// ticker is one hour; the first sweep runs immediately on startup so a
// freshly started plugin cleans a stale database without waiting an hour.
func runRetention(ctx context.Context, wg *sync.WaitGroup, st *store.Store, retentionDays int) {
	defer wg.Done()
	const interval = time.Hour
	sweep := func() {
		cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).UnixMilli()
		if _, err := st.DeleteEventsBefore(ctx, cutoff); err != nil {
			// Swallow the error: retention is best-effort and must not crash
			// the plugin. The next tick retries.
			_ = err
		}
	}
	sweep()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep()
		}
	}
}
