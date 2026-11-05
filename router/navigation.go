package router

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// isBrowserNavigationRequest reports whether a request is a browser
// navigating to a page rather than a fetch.
//
// A navigation announces itself with an Accept header that prefers HTML. It
// is what separates "open this address in a browser" from "call this
// endpoint", and it is what decides whether an unknown path is redirected to
// the frontend or answered with a JSON 404.
//
// Without this distinction every unknown API path would answer with a 302 to
// an HTML page, and the browser would render the frontend's index where the
// caller expected an error.
func isBrowserNavigationRequest(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	if accept == "" {
		return false
	}
	return strings.Contains(accept, "text/html")
}
