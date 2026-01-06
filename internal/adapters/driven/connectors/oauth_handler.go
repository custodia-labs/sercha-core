package connectors

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// OAuthHandler provides OAuth operations for a specific provider.
// Each provider (GitHub, Google Drive, etc.) has its own implementation.
type OAuthHandler interface {
	// BuildAuthURL constructs the OAuth authorization URL.
	// Parameters:
	//   - clientID: OAuth application client ID
	//   - redirectURI: Where to redirect after authorization
	//   - state: CSRF protection token
	//   - codeChallenge: PKCE code challenge (S256 hash of code verifier)
	//   - scopes: Requested OAuth scopes
	// Returns the full authorization URL to redirect the user to.
	BuildAuthURL(clientID, redirectURI, state, codeChallenge string, scopes []string) string

	// ExchangeCode exchanges an authorization code for tokens.
	// Parameters:
	//   - ctx: Context for cancellation
	//   - clientID: OAuth application client ID
	//   - clientSecret: OAuth application client secret
	//   - code: Authorization code from callback
	//   - redirectURI: Must match the URI used in authorization
	//   - codeVerifier: PKCE code verifier (plain text)
	// Returns the OAuth tokens (access token, refresh token, etc.)
	ExchangeCode(ctx context.Context, clientID, clientSecret, code, redirectURI, codeVerifier string) (*driven.OAuthToken, error)

	// RefreshToken refreshes an expired access token.
	// Parameters:
	//   - ctx: Context for cancellation
	//   - clientID: OAuth application client ID
	//   - clientSecret: OAuth application client secret
	//   - refreshToken: The refresh token from previous authorization
	// Returns new OAuth tokens.
	RefreshToken(ctx context.Context, clientID, clientSecret, refreshToken string) (*driven.OAuthToken, error)

	// GetUserInfo fetches the account identifier (email or username).
	// This is used to display which account is connected and to prevent duplicates.
	// Parameters:
	//   - ctx: Context for cancellation
	//   - accessToken: Valid access token
	// Returns the account identifier (typically email or username).
	GetUserInfo(ctx context.Context, accessToken string) (*driven.OAuthUserInfo, error)

	// DefaultConfig returns the provider's default OAuth configuration.
	// This includes auth URL, token URL, and default scopes.
	DefaultConfig() OAuthDefaults
}

// OAuthDefaults contains a provider's default OAuth configuration.
type OAuthDefaults struct {
	// AuthURL is the OAuth authorization endpoint.
	AuthURL string

	// TokenURL is the OAuth token exchange endpoint.
	TokenURL string

	// Scopes are the default OAuth scopes to request.
	Scopes []string

	// UserInfoURL is the endpoint to fetch user information (optional).
	UserInfoURL string

	// SupportsPKCE indicates if the provider supports PKCE.
	SupportsPKCE bool
}
