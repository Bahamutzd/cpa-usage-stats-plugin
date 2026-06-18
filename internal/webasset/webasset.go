// Package webasset embeds the single-file SPA served at the plugin resource
// route. CLIProxyAPI matches plugin resource routes by exact path and forbids
// path parameters, so the dashboard must be one self-contained HTML file with
// inlined JS/CSS and a hash router — there is no way to serve a tree of
// static assets through the plugin resource route table.
package webasset

import "embed"

// Panel holds the bundled single-file dashboard. It is replaced by the real
// built SPA in the front-end build step.
//
//go:embed index.html
var Panel embed.FS

// Index returns the dashboard HTML bytes.
func Index() []byte {
	data, err := Panel.ReadFile("index.html")
	if err != nil {
		return []byte("dashboard asset missing")
	}
	return data
}