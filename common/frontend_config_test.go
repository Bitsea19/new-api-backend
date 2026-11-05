package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetFrontendConfigAfterTest(t *testing.T) {
	t.Helper()
	previousFrontend, previousAPI := FrontendBaseURL, APIBaseURL
	previousOrigins := CORSAllowedOrigins
	t.Cleanup(func() {
		FrontendBaseURL, APIBaseURL = previousFrontend, previousAPI
		CORSAllowedOrigins = previousOrigins
	})
}

func TestInitFrontendConfigReadsDeploymentURLs(t *testing.T) {
	resetFrontendConfigAfterTest(t)
	t.Setenv("FRONTEND_BASE_URL", "https://app.example.com/")
	t.Setenv("API_BASE_URL", "https://api.example.com/")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")

	require.NoError(t, InitFrontendConfig())

	assert.Equal(t, "https://app.example.com", FrontendBaseURL)
	assert.Equal(t, "https://api.example.com", APIBaseURL)
	assert.Equal(t, []string{"https://app.example.com"}, CORSAllowedOrigins)
}

func TestInitFrontendConfigLeavesURLsEmptyWhenUnset(t *testing.T) {
	resetFrontendConfigAfterTest(t)
	t.Setenv("FRONTEND_BASE_URL", "")
	t.Setenv("API_BASE_URL", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	require.NoError(t, InitFrontendConfig())

	assert.Empty(t, FrontendBaseURL)
	assert.Empty(t, APIBaseURL)
	assert.Empty(t, CORSAllowedOrigins)
}

func TestResolveAddressesPreferDeploymentOverDatabase(t *testing.T) {
	resetFrontendConfigAfterTest(t)
	t.Setenv("FRONTEND_BASE_URL", "https://app.example.com")
	t.Setenv("API_BASE_URL", "https://api.example.com")
	require.NoError(t, InitFrontendConfig())

	assert.Equal(t, "https://app.example.com", ResolveWebsiteAddress("https://legacy.example.com"))
	assert.Equal(t, "https://api.example.com", ResolveAPIAddress("https://legacy.example.com"))
}

func TestResolveAddressesFallBackToDatabase(t *testing.T) {
	resetFrontendConfigAfterTest(t)
	t.Setenv("FRONTEND_BASE_URL", "")
	t.Setenv("API_BASE_URL", "")
	require.NoError(t, InitFrontendConfig())

	assert.Equal(t, "https://legacy.example.com", ResolveWebsiteAddress("https://legacy.example.com/"))
	assert.Equal(t, "https://legacy.example.com", ResolveAPIAddress("https://legacy.example.com/"))
}

func TestParseCORSAllowedOriginsRejectsWildcards(t *testing.T) {
	// A wildcard is invalid alongside credentials: the browser rejects the
	// response outright, so allowing one here would look configured while
	// breaking every sign-in.
	_, err := parseCORSAllowedOrigins("https://*.example.com")
	require.Error(t, err)
}

func TestParseCORSAllowedOriginsRejectsInvalidEntries(t *testing.T) {
	for _, raw := range []string{
		"https://example.com,",
		"https://example.com/admin",
		"not-a-url",
	} {
		t.Run(raw, func(t *testing.T) {
			_, err := parseCORSAllowedOrigins(raw)
			require.Error(t, err)
		})
	}
}

func TestIsCORSOriginAllowed(t *testing.T) {
	resetFrontendConfigAfterTest(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com:8443")
	require.NoError(t, InitFrontendConfig())

	assert.True(t, IsCORSOriginAllowed("https://app.example.com"))
	assert.True(t, IsCORSOriginAllowed("https://admin.example.com:8443"))
	assert.False(t, IsCORSOriginAllowed("https://evil.example.com"))
	assert.False(t, IsCORSOriginAllowed(""))
}

func TestIsCORSOriginAllowedDeniesEverythingWithoutAnAllowlist(t *testing.T) {
	resetFrontendConfigAfterTest(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	require.NoError(t, InitFrontendConfig())

	assert.False(t, IsCORSOriginAllowed("https://app.example.com"))
}
