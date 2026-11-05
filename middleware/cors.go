package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// corsAllowedMethods is the set of methods the API accepts from a browser.
//
// It covers every verb the dashboard and the relay endpoints use, plus HEAD
// and OPTIONS, so a preflight never has to be answered by the route table.
const corsAllowedMethods = "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS"

// corsDefaultAllowedHeaders is the fallback set of request headers a
// preflight may ask for.
//
// When the browser sends Access-Control-Request-Headers, that list is echoed
// back verbatim instead; this constant only covers a preflight that asks for
// no specific header.
const corsDefaultAllowedHeaders = "Content-Type, Authorization, X-Security-Proof, X-Auth-Session, X-Requested-With"

// corsExposedHeaders are the response headers a browser may read.
//
// The dashboard reads the version header for diagnostics and the download
// disposition for the log export, so both have to be explicitly exposed.
const corsExposedHeaders = "Content-Type, Content-Length, Content-Disposition, X-New-Api-Version"

// corsPreflightMaxAge is how long a browser may cache a preflight result.
//
// Ten minutes keeps the preflight traffic low without making a deployment
// change (a new allowed origin, a new header) invisible for long.
const corsPreflightMaxAge = "600"

// CORS answers cross-origin browser requests.
//
// Two policies coexist, and which one applies is decided by the origin alone:
//
//   - A credentialed request — one the dashboard makes with cookies — needs
//     an exact origin allowlist, because a wildcard origin is invalid
//     alongside credentials: the browser rejects the response outright.
//   - A request without credentials — an SDK calling the relay endpoints —
//     works from any origin, so it keeps the permissive wildcard policy.
//
// The allowlist comes from CORS_ALLOWED_ORIGINS. An empty allowlist means no
// browser origin may use credentials, which is the right default for a
// same-origin deployment where the dashboard and the API share one origin.
//
// This middleware never rejects a request. A disallowed origin simply gets no
// Access-Control-Allow-Origin header, and the browser — which is the only
// party that can enforce this — blocks the response. Server-to-server
// clients, which never send an Origin header, are unaffected either way.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowed := common.IsCORSOriginAllowed(origin)
		if allowed {
			// Echo the exact origin: a wildcard is rejected by the browser
			// whenever the request carries credentials.
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		} else {
			// Unauthenticated relay traffic stays reachable from anywhere.
			// No credentials header here: the two must not be combined.
			c.Header("Access-Control-Allow-Origin", "*")
		}
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Expose-Headers", corsExposedHeaders)

		if c.Request.Method == http.MethodOptions {
			if allowed {
				requested := c.GetHeader("Access-Control-Request-Headers")
				if requested != "" {
					c.Header("Access-Control-Allow-Headers", requested)
				} else {
					c.Header("Access-Control-Allow-Headers", corsDefaultAllowedHeaders)
				}
				c.Header("Access-Control-Allow-Methods", corsAllowedMethods)
				c.Header("Access-Control-Max-Age", corsPreflightMaxAge)
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Version adds the build version to every response.
func Version() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-New-Api-Version", common.Version)
		c.Next()
	}
}
