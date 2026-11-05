package common

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
)

// Session cookie SameSite policy.
//
// The default is Strict, which is the strongest choice: the Refresh Cookie
// is only ever sent on a request the site itself initiated, so a link from
// another site cannot trigger a request that carries it.
//
// A separated deployment usually does not need to relax this. Two
// subdomains of one registrable domain — app.example.com and
// api.example.com — are the same site, so Strict still works. Only a
// deployment that spans two separate registrable domains has to relax it,
// and that requires SameSite=None together with Secure.
var SessionCookieSameSite = http.SameSiteStrictMode

// NormalizeOrigin validates and canonicalizes a browser origin. Only an exact
// scheme/host/effective-port match is meaningful; paths and wildcards are not
// accepted for authentication cookie endpoints.
func NormalizeOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" || strings.ContainsAny(raw, "\r\n") {
		return "", fmt.Errorf("origin is empty or invalid")
	}
	parsedURL, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid origin: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("origin scheme must be http or https")
	}
	if parsedURL.Host == "" || parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" || (parsedURL.Path != "" && parsedURL.Path != "/") {
		return "", fmt.Errorf("origin must contain only scheme and host")
	}
	hostname := strings.ToLower(parsedURL.Hostname())
	if hostname == "" || strings.Contains(hostname, "*") {
		return "", fmt.Errorf("origin host is empty")
	}
	port := parsedURL.Port()
	normalizedHost := hostname
	if strings.Contains(hostname, ":") {
		normalizedHost = "[" + hostname + "]"
	}
	if port == "" || (parsedURL.Scheme == "http" && port == "80") || (parsedURL.Scheme == "https" && port == "443") {
		return parsedURL.Scheme + "://" + normalizedHost, nil
	}
	return parsedURL.Scheme + "://" + net.JoinHostPort(hostname, port), nil
}

func InitSessionCookieSettings() error {
	secureRaw := strings.TrimSpace(os.Getenv("SESSION_COOKIE_SECURE"))
	trustedURLsRaw := strings.TrimSpace(os.Getenv("SESSION_COOKIE_TRUSTED_URL"))

	SessionCookieSecure = false
	SessionCookieTrustedURLs = nil

	sameSite, err := parseSessionCookieSameSite(os.Getenv("SESSION_COOKIE_SAME_SITE"))
	if err != nil {
		return err
	}
	SessionCookieSameSite = sameSite

	if secureRaw == "" || strings.EqualFold(secureRaw, "false") {
		if trustedURLsRaw != "" {
			return fmt.Errorf("SESSION_COOKIE_TRUSTED_URL requires SESSION_COOKIE_SECURE=true")
		}
		if sameSite == http.SameSiteNoneMode {
			return fmt.Errorf("SESSION_COOKIE_SAME_SITE=none requires SESSION_COOKIE_SECURE=true")
		}
		return nil
	}

	if !strings.EqualFold(secureRaw, "true") {
		return fmt.Errorf("SESSION_COOKIE_SECURE must be true or false")
	}

	if trustedURLsRaw == "" && strings.TrimSpace(FrontendBaseURL) == "" {
		return fmt.Errorf("SESSION_COOKIE_SECURE=true requires SESSION_COOKIE_TRUSTED_URL or FRONTEND_BASE_URL")
	}

	trustedURLs := strings.SplitSeq(trustedURLsRaw, ",")
	for trustedURL := range trustedURLs {
		trustedURL = strings.TrimSpace(trustedURL)
		if trustedURL == "" {
			return fmt.Errorf("SESSION_COOKIE_TRUSTED_URL contains an empty URL")
		}
		normalizedOrigin, err := NormalizeOrigin(trustedURL)
		if err != nil {
			return fmt.Errorf("invalid SESSION_COOKIE_TRUSTED_URL: %w", err)
		}
		if !strings.HasPrefix(normalizedOrigin, "https://") {
			return fmt.Errorf("SESSION_COOKIE_TRUSTED_URL must contain only https URLs with hosts")
		}
		SessionCookieTrustedURLs = append(SessionCookieTrustedURLs, normalizedOrigin)
	}

	// The frontend's own origin always has to be trusted: it is the browser
	// that presents the Refresh Cookie. In a separated deployment it is not
	// the request's host, so without this the refresh and logout endpoints
	// would reject the very requests that carry a valid session.
	if frontendOrigin := strings.TrimSpace(FrontendBaseURL); frontendOrigin != "" {
		if normalizedOrigin, err := NormalizeOrigin(frontendOrigin); err == nil {
			if !slices.Contains(SessionCookieTrustedURLs, normalizedOrigin) {
				SessionCookieTrustedURLs = append(SessionCookieTrustedURLs, normalizedOrigin)
			}
		}
	}

	SessionCookieSecure = true
	return nil
}

// parseSessionCookieSameSite maps SESSION_COOKIE_SAME_SITE onto a
// net/http SameSite value.
//
// An empty value keeps Strict. "none" is accepted only alongside a Secure
// cookie, because a SameSite=None cookie without Secure is rejected by
// every current browser outright — the setting would look applied while
// silently dropping the session on every response.
func parseSessionCookieSameSite(raw string) (http.SameSite, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "", "strict":
		return http.SameSiteStrictMode, nil
	case "lax":
		return http.SameSiteLaxMode, nil
	case "none":
		return http.SameSiteNoneMode, nil
	default:
		return http.SameSiteStrictMode, fmt.Errorf("SESSION_COOKIE_SAME_SITE must be strict, lax or none")
	}
}
