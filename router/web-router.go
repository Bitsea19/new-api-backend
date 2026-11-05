package router

import (
	"embed"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// WebAssets holds the embedded dashboard frontend assets.
//
// The fields are empty in the default (API-only) build, which is what the
// separated deployment uses: the backend serves no static files at all and
// every unknown route answers with a JSON 404 instead of an SPA fallback.
//
// BuildFS is a pointer so that "no embedded frontend" is representable. The
// zero embed.FS is a valid empty filesystem, and fs.Sub on it fails, so the
// presence of a build has to be distinguishable from an empty one.
type WebAssets struct {
	BuildFS   *embed.FS
	IndexPage []byte
}

// HasEmbeddedFrontend reports whether this build carries the embedded SPA.
//
// It is false for the default build and for the `embed_frontend` build whose
// web/dist was not produced, so the router can fall back to API-only
// behaviour instead of panicking on a missing embed target.
func (a WebAssets) HasEmbeddedFrontend() bool {
	return a.BuildFS != nil && len(a.IndexPage) > 0
}

func SetWebRouter(router *gin.Engine, assets WebAssets, pluginDispatcher gin.HandlerFunc) {
	if !assets.HasEmbeddedFrontend() {
		SetAPIRouter(router, pluginDispatcher)
		return
	}

	frontendFS := common.EmbedFolder(*assets.BuildFS, "web/dist")
	compress := gzip.Gzip(gzip.DefaultCompression)

	router.NoRoute(
		pluginDispatcher,
		middleware.RouteTag("web"),
		func(c *gin.Context) {
			// ServeFrontendFiles sends build files already compressed.
			if !frontendFS.Exists("/", c.Request.URL.Path) {
				compress(c)
			}
		},
		middleware.AccessTokenAudit(),
		middleware.GlobalWebRateLimit(frontendFS),
		middleware.Cache(),
		middleware.ServeFrontendFiles(frontendFS),
		func(c *gin.Context) {
			if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
				controller.RelayNotFound(c)
				return
			}
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, "text/html; charset=utf-8", assets.IndexPage)
		},
	)
}

// SetAPIRouter registers the catch-all handler used when no frontend is
// embedded and no FRONTEND_BASE_URL is configured.
//
// A separated backend must never answer an unknown browser route with the
// SPA shell, because that shell does not exist here. API-shaped paths keep the
// OpenAI-style 404 body that clients already parse, while every other path
// answers with a JSON 404 that names the backend, so an operator hitting a
// typo'd frontend URL sees the real cause instead of a blank page.
func SetAPIRouter(router *gin.Engine, pluginDispatcher gin.HandlerFunc) {
	router.NoRoute(
		pluginDispatcher,
		middleware.RouteTag("web"),
		middleware.AccessTokenAudit(),
		func(c *gin.Context) {
			path := c.Request.URL.Path
			if strings.HasPrefix(path, "/v1") ||
				strings.HasPrefix(path, "/v1beta") ||
				strings.HasPrefix(path, "/api") ||
				strings.HasPrefix(path, "/mj") ||
				strings.HasPrefix(path, "/pg") ||
				strings.HasPrefix(path, "/assets") {
				controller.RelayNotFound(c)
				return
			}
			c.Header("Cache-Control", "no-store")
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "This deployment serves the API only. Open the frontend URL configured as FRONTEND_BASE_URL.",
			})
		},
	)
}
