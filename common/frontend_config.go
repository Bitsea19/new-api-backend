package common

import (
	"fmt"
	"os"
	"strings"
)

// FrontendBaseURL is the public URL of the separately deployed frontend,
// taken from the FRONTEND_BASE_URL environment variable.
//
// It is the origin a browser uses to reach the dashboard, so it is what
// belongs in OAuth redirect_uri values, password-reset links, and payment
// return paths.
//
// It is read from the environment and never from the database: the frontend's
// URL is a deployment concern, and a value that only one node knows would
// make OAuth handshakes fail on the others. An empty value means the
// deployment has no separate frontend and the database-managed ServerAddress
// is used instead.
var FrontendBaseURL = ""

// APIBaseURL is the public URL of the backend API, taken from the
// API_BASE_URL environment variable.
//
// It is used to build links that the backend itself serves and that a
// browser or an upstream service must be able to fetch: the Midjourney image
// proxy, video content URLs, and the payment gateway's notify callback.
//
// An empty value means the deployment has no separate API address and the
// database-managed ServerAddress is used instead.
var APIBaseURL = ""

// CORSAllowedOrigins lists the browser origins allowed to call the API with
// credentials.
//
// An empty list means "no cross-origin browser access", which is the correct
// default for a same-origin monolith and for an API-only deployment whose
// frontend is reached through a reverse proxy on the same origin.
var CORSAllowedOrigins []string

// InitFrontendConfig reads the deployment-level frontend/backend URLs.
//
// The frontend URL may also be provided through the FRONTEND_BASE_URL
// environment variable, which takes precedence over the API_BASE_URL
// environment variable and the database-managed ServerAddress.
func InitFrontendConfig() error {
	FrontendBaseURL = strings.TrimRight(strings.TrimSpace(os.Getenv("FRONTEND_BASE_URL")), "/")
	APIBaseURL = strings.TrimRight(strings.TrimSpace(os.Getenv("API_BASE_URL")), "/")

	origins, err := parseCORSAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if err != nil {
		return err
	}
	CORSAllowedOrigins = origins
	return nil
}

// ResolveWebsiteAddress returns the website address a browser uses.
//
// The deployment-level FRONTEND_BASE_URL wins over any database-managed
// default, because the frontend's URL is a property of the deployment and
// has to be identical on every node. The fallback is the caller-supplied
// address, which is the database-managed ServerAddress in the monolith.
func ResolveWebsiteAddress(fallback string) string {
	if configured := strings.TrimRight(strings.TrimSpace(FrontendBaseURL), "/"); configured != "" {
		return configured
	}
	return strings.TrimRight(strings.TrimSpace(fallback), "/")
}

// ResolveAPIAddress returns the address the backend itself is reached at.
//
// The deployment-level API_BASE_URL wins over any database-managed default,
// for the same reason as ResolveWebsiteAddress.
func ResolveAPIAddress(fallback string) string {
	if configured := strings.TrimRight(strings.TrimSpace(APIBaseURL), "/"); configured != "" {
		return configured
	}
	return strings.TrimRight(strings.TrimSpace(fallback), "/")
}

// parseCORSAllowedOrigins parses the CORS_ALLOWED_ORIGINS environment
// variable into a list of exact origins.
//
// Every entry must be an exact scheme/host/port origin. Wildcards are
// rejected on purpose: the API sends credentials, and a wildcard origin is
// incompatible with credentialed requests — the browser rejects the
// response outright, so a wildcard here would look like a working
// configuration while breaking every sign-in.
func parseCORSAllowedOrigins(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var origins []string
	seen := make(map[string]struct{})
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS contains an empty URL")
		}
		if strings.Contains(entry, "*") {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS must not contain wildcards: %s", entry)
		}
		origin, err := NormalizeOrigin(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid CORS_ALLOWED_ORIGINS entry %q: %w", entry, err)
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	return origins, nil
}

// IsCORSOriginAllowed reports whether a browser origin may call the API.
//
// The request's own origin is always allowed: a same-origin request carries
// an Origin header in some cases (a POST from a form, a fetch with a
// non-simple content type), and refusing it would break the monolith.
func IsCORSOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	if len(CORSAllowedOrigins) == 0 {
		return false
	}
	normalized, err := NormalizeOrigin(origin)
	if err != nil {
		return false
	}
	for _, allowed := range CORSAllowedOrigins {
		if allowed == normalized {
			return true
		}
	}
	return false
}
