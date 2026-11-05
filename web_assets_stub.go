//go:build !embed_frontend

package main

import (
	"github.com/QuantumNous/new-api/router"
)

// webAssets returns empty dashboard assets.
//
// This file is compiled by default, which makes the backend a pure API
// server: `go build` no longer requires `web/dist` to exist, and the binary
// no longer carries the React bundle. Deploy the frontend separately and
// point it at this backend with VITE_API_BASE_URL.
//
// The monolith (backend + embedded SPA in one binary) is still available by
// building with the `embed_frontend` tag; see web_assets_embed.go.
func webAssets() router.WebAssets {
	return router.WebAssets{}
}
