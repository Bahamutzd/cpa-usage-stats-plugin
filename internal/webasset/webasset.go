// Package webasset embeds the single-file dashboard served at the plugin
// resource route. CLIProxyAPI matches plugin resource routes by exact path and
// forbids path parameters, so the panel must stay self-contained with inlined
// JS/CSS — there is no way to serve a tree of static assets through the plugin
// resource route table.
package webasset

import "embed"

// Panel holds the embedded dashboard assets. panel.html is the plugin-focused
// runtime entrypoint; index.html is kept as a legacy fallback.
//
//go:embed panel.html index.html
var Panel embed.FS

// Index returns the dashboard HTML bytes.
func Index() []byte {
	for _, name := range []string{"panel.html", "index.html"} {
		data, err := Panel.ReadFile(name)
		if err == nil {
			return data
		}
	}
	return []byte("dashboard asset missing")
}
