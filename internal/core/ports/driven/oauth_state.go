package driven

import (
	"context"
	"time"
)

// OAuthState represents a pending OAuth authorization flow state.
// Used for CSRF protection and PKCE code verifier storage.
type OAuthState struct {
	// State is a cryptographically random string used for CSRF protection.
	State string

	// ProviderType is the OAuth provider (github, google_drive, etc.)
	ProviderType string

	// CodeVerifier is the PKCE code verifier (plain text, not hashed).
	// This is used to generate code_challenge for the auth request
	// and sent as code_verifier during token exchange.
	CodeVerifier string

	// RedirectURI is the callback URL where the provider will redirect.
	RedirectURI string

	// CreatedAt is when the state was created.
	CreatedAt time.Time

	// ExpiresAt is when the state expires (typically 10 minutes).
	ExpiresAt time.Time
}

// OAuthStateStore manages OAuth flow state for CSRF protection.
// States are single-use and expire after a short period.
type OAuthStateStore interface {
	// Save stores a new OAuth state.
	// The state typically expires in 10 minutes.
	Save(ctx context.Context, state *OAuthState) error

	// GetAndDelete atomically retrieves and deletes the state.
	// This ensures single-use semantics.
	// Returns nil, nil if the state doesn't exist or has expired.
	GetAndDelete(ctx context.Context, state string) (*OAuthState, error)

	// Cleanup removes expired states.
	// Should be called periodically (e.g., every hour) to clean up orphaned states.
	Cleanup(ctx context.Context) error
}
