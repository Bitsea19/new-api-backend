package system_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// FrontendBaseURL returns the public URL of the dashboard a browser visits.
//
// It is the origin that must appear in OAuth redirect_uri values, in
// password-reset links, and in payment return paths: those all end in a
// browser navigation, and in a separated deployment the browser is on the
// frontend's origin, not the API's.
//
// The order is deliberate:
//
//  1. The FRONTEND_BASE_URL environment variable, which is a deployment
//     concern and identical on every node.
//  2. The database-managed ServerAddress, so the monolith — where the
//     dashboard and the API share one origin — keeps its current behaviour
//     with no configuration at all.
//
// An empty result means the deployment has no known frontend URL; callers
// that need one must fall back to the API address or fail loudly.
func FrontendBaseURL() string {
	return common.ResolveWebsiteAddress(ServerAddress)
}

// APIBaseURL returns the public URL of the backend API.
//
// It is the origin that must appear in links the backend itself serves and
// that a browser or an upstream service fetches: the Midjourney image proxy,
// video content URLs, and the payment gateway's notify callback.
//
// The order is deliberate:
//
//  1. The API_BASE_URL environment variable, which is a deployment concern.
//  2. The database-managed ServerAddress, so the monolith keeps its current
//     behaviour with no configuration at all.
func APIBaseURL() string {
	return common.ResolveAPIAddress(ServerAddress)
}

// WebsiteAddress returns the public URL of the website a browser visits.
//
// It is the origin that must appear in browser-facing trust decisions: the
// Passkey relying-party domain and the origins a WebAuthn ceremony is
// allowed to run under.
//
// The browser is what performs those ceremonies, so the address that matters
// is the frontend's, not the API's. In a monolith they are the same.
func WebsiteAddress() string {
	return FrontendBaseURL()
}

// trimTrailingSlash removes a trailing slash from an address so that
// concatenating a path never produces a double slash.
func trimTrailingSlash(address string) string {
	return strings.TrimRight(strings.TrimSpace(address), "/")
}
