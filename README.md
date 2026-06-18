# cpa-usage-stats

A standalone CLIProxyAPI plugin that captures every request's usage record
via the `usage_plugin` capability and exposes a CPA-Manager-Plus style
dashboard / monitoring API through `management_api`. No separate process to
deploy: install the plugin, the dashboard lives inside CLIProxyAPI's
management UI under **请求监控**.

## Status

Functional usage-statistics plugin. What it does:

- Declares `usage_plugin` + `management_api` capabilities; receives every
  request's `UsageRecord` via `usage.handle` and stores it in an embedded
  SQLite database (`modernc.org/sqlite`, pure Go — no host sqlite required).
- Exposes the CPA-Manager-Plus-style Management API under `/v0/management/`:
  - `GET /usage` — compatible usage payload (events grouped by API × model).
  - `GET /usage/export` · `POST /usage/import` — JSONL export/import (the
    import parser also accepts the two legacy CPA-Manager-Plus formats).
  - `GET /dashboard/summary` — today window, rolling-30m, top models, traffic
    timeline, hourly activity, token mix, channel health, failure sources,
    recent failures, health timeline.
  - `POST /monitoring/analytics` — summary, timeline, hourly distribution,
    model stats/share, channel share, failure sources, account/api-key stats,
    task buckets, recent failures, event pagination, filter options.
  - `GET /api-key-aliases` · `PUT /api-key-aliases` — alias labels.
  - `GET /model-prices` · `PUT /model-prices` · `POST /model-prices/sync` —
    per-model pricing; sync pulls the LiteLLM catalog.
- `retention_days > 0` runs a hourly prune goroutine.
- Serves a single-file dashboard (inline CSS/JS, hash router) at the plugin
  resource route **请求监控**. The exact page URL is
  `/v0/resource/plugins/cpa-usage-stats/index`.
  The shipped page includes its own connection bar, stores the management key
  locally, and can infer the same-origin API prefix from the current resource
  URL.

### Known limitations (host architecture, not bugs)

- CLIProxyAPI matches plugin management/resource routes by **exact path** and
  forbids path parameters. So `DELETE /api-key-aliases/{hash}` is unreachable;
  delete a single alias by editing the list and `PUT`-ing the whole table back.
- The dashboard is one self-contained HTML file — a tree of static assets
  cannot be served through the plugin resource route table.
- Backend `cost` fields are still `0`; once model prices are loaded, the
  dashboard estimates request costs client-side for monitoring/overview cards
  and tables.

## Using the dashboard

1. Configure the plugin (see Install) and (re)start CLIProxyAPI.
2. Open the CPA management UI; the plugin appears as the **请求监控** menu
   item, served at `/v0/resource/plugins/cpa-usage-stats/index`.
3. In the dashboard header, paste the **Management Key** from the host's
   `management-key` config. It is stored in `localStorage` and sent as
   `Authorization: Bearer <key>` on every API call. API base can stay empty when
   the dashboard is served from the same origin as CLIProxyAPI; the page will
   infer the deploy prefix from its own `/v0/resource/plugins/...` URL. Only
   fill API base when the management API is exposed on a different origin.

## Install

In the host CLIProxyAPI's `config.yaml`:

```yaml
plugins:
  store-sources:
    - https://raw.githubusercontent.com/Bahamutzd/cpa-usage-stats-plugin/main/plugins-store/registry.json
  configs:
    cpa-usage-stats:
      enabled: true
      # Optional. Defaults to <plugins_dir>/cpa-usage-stats/usage.db.
      db_path: ./data/cpa-usage.db
      # Optional. Maximum age in days to retain events. 0 disables pruning.
      retention_days: 90
```

Then open the plugin store in the management UI; the registry source above
will appear next to the official source and the plugin can be installed in
one click. After install, the **请求监控** menu item appears in the
management UI.

> **Release contract:** CLIProxyAPI installs third-party plugins from the
> plugin entry's GitHub `latest` release. Each release must contain
> `<pluginID>_<version>_<goos>_<goarch>.zip` archives plus `checksums.txt`,
> with the shared library at the zip root. The workflow in
> `.github/workflows/release.yml` builds that layout from a `vX.Y.Z` tag.

## Build locally

The runtime dashboard is embedded from `internal/webasset/panel.html`; no
separate React build is required for the plugin to serve its shipped UI.
The `web/` directory is optional iterative UI work and is not used by the
runtime loader.

```bash
# linux/macOS shared library
CGO_ENABLED=1 go build -trimpath -buildmode=c-shared \
    -o cpa-usage-stats.so .

# windows .dll (mingw-w64 toolchain required)
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
    go build -trimpath -buildmode=c-shared -o cpa-usage-stats.dll .
```

## License

MIT
