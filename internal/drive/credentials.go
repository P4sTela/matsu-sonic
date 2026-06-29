package drive

import (
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// DefaultClientID / DefaultClientSecret hold the OAuth client credentials that
// are baked into release binaries via -ldflags. They let a freshly downloaded
// executable authenticate without the user having to obtain and place a
// client_secret_*.json file next to it.
//
// These belong to a Google "Desktop app" (installed) OAuth client. For that
// client type the secret is NOT treated as confidential by Google: it is
// expected to ship inside distributed apps, and PKCE is what actually protects
// the authorization-code exchange. See:
// https://developers.google.com/identity/protocols/oauth2/native-app
//
// Inject at build time, e.g.:
//
//	go build -ldflags "-X github.com/P4sTela/matsu-sonic/internal/drive.DefaultClientID=... \
//	                   -X github.com/P4sTela/matsu-sonic/internal/drive.DefaultClientSecret=..."
var (
	DefaultClientID     string
	DefaultClientSecret string
)

// hasEmbeddedCredentials reports whether build-time credentials are available.
func hasEmbeddedCredentials() bool {
	return strings.TrimSpace(DefaultClientID) != ""
}

// embeddedOAuthConfig builds an oauth2.Config from the embedded credentials.
func embeddedOAuthConfig(scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     strings.TrimSpace(DefaultClientID),
		ClientSecret: strings.TrimSpace(DefaultClientSecret),
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}
}
