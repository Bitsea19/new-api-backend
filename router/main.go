package router

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetRouter(router *gin.Engine, assets WebAssets) {
	SetApiRouter(router)
	SetDashboardRouter(router)
	SetRelayRouter(router)
	SetTaskPluginProtocolRouter(router)
	SetVideoRouter(router)
	SetTaskRouter(router)
	pluginDispatcher := SetPluginRouter(router)

	frontendBaseUrl := resolveFrontendBaseUrl()
	if frontendBaseUrl != "" {
		setFrontendRedirectRouter(router, pluginDispatcher, frontendBaseUrl)
		return
	}
	SetWebRouter(router, assets, pluginDispatcher)
}

// resolveFrontendBaseUrl returns the frontend URL that browser navigation
// requests are redirected to.
//
// Unlike the monolith, a separated deployment must redirect on every node.
// The old code only redirected on workers and let the master serve the
// embedded SPA, which assumed the master had a frontend build. In a
// separated deployment the master has no build at all, so redirecting there
// too is what makes a single FRONTEND_BASE_URL work across the fleet.
func resolveFrontendBaseUrl() string {
	frontendBaseUrl := strings.TrimRight(strings.TrimSpace(os.Getenv("FRONTEND_BASE_URL")), "/")
	if frontendBaseUrl == "" {
		return ""
	}
	if !common.IsMasterNode {
		common.SysLog(common.LogText("FRONTEND_BASE_URL is set: this node redirects browser navigation to %s", frontendBaseUrl))
	}
	return frontendBaseUrl
}

// setFrontendRedirectRouter sends browser navigation to the frontend and
// answers everything else with a JSON 404.
//
// A fetch from the dashboard never reaches here — it carries an application
// content type and asks for JSON — so the redirect is limited to navigations.
// Without that distinction every unknown API path would answer with a
// 302 to an HTML page, and the browser would render the frontend's index
// where the caller expected an error.
func setFrontendRedirectRouter(router *gin.Engine, pluginDispatcher gin.HandlerFunc, frontendBaseUrl string) {
	router.NoRoute(
		pluginDispatcher,
		middleware.RouteTag("web"),
		middleware.AccessTokenAudit(),
		func(c *gin.Context) {
			if !isBrowserNavigationRequest(c) {
				c.Header("Cache-Control", "no-store")
				c.JSON(http.StatusNotFound, gin.H{
					"success": false,
					"message": "This deployment serves the API only. Open the frontend URL configured as FRONTEND_BASE_URL.",
				})
				return
			}
			c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("%s%s", frontendBaseUrl, c.Request.RequestURI))
		},
	)
}
