//go:build embed_frontend

package main

import (
	"embed"

	"github.com/QuantumNous/new-api/router"
)

//go:embed web/dist
var buildFS embed.FS

//go:embed web/dist/index.html
var indexPage []byte

// webAssets returns the embedded dashboard assets.
//
// This file is compiled only with the `embed_frontend` build tag. The default
// build (see web_assets_stub.go) produces a pure API binary that has no
// dependency on a frontend build output, so the backend can be built and
// deployed independently of the frontend.
//
// Build the monolith with:
//	go build -tags embed_frontend
// The embedded assets are still what the master node serves when
// FRONTEND_BASE_URL is unset.
func webAssets() router.WebAssets {
	return router.WebAssets{
		BuildFS:   &buildFS,
		IndexPage: indexPage,
	}
}
