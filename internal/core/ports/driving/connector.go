package driving

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// ConnectorRegistry manages available connectors and their OAuth configurations.
// This is the driving port used by API handlers to list connectors and handle OAuth.
type ConnectorRegistry interface {
	// List returns all registered provider types.
	List() []domain.ProviderType

	// ListInfo returns info about all available connectors.
	ListInfo() []domain.ProviderInfo

	// GetInfo returns info for a specific provider type.
	GetInfo(providerType domain.ProviderType) (*domain.ProviderInfo, error)

	// IsAvailable checks if a connector is registered.
	IsAvailable(providerType domain.ProviderType) bool

	// SupportsOAuth returns true if the provider supports OAuth.
	SupportsOAuth(providerType domain.ProviderType) bool

	// GetOAuthConfig returns OAuth configuration for a provider.
	// Returns nil if provider doesn't support OAuth.
	GetOAuthConfig(providerType domain.ProviderType) *driven.OAuthConfig

	// BuildAuthURL builds an OAuth authorization URL for the provider.
	// The state parameter should be cryptographically random for CSRF protection.
	// The redirectURL is where the OAuth provider will redirect after authorization.
	BuildAuthURL(providerType domain.ProviderType, state, redirectURL string) (string, error)

	// ExchangeCode exchanges an OAuth authorization code for tokens.
	ExchangeCode(ctx context.Context, providerType domain.ProviderType, code, redirectURL string) (*driven.OAuthToken, error)

	// GetUserInfo retrieves user info using an access token.
	GetUserInfo(ctx context.Context, providerType domain.ProviderType, accessToken string) (*driven.OAuthUserInfo, error)

	// ValidateConfig validates source configuration for a provider.
	ValidateConfig(providerType domain.ProviderType, config domain.SourceConfig) error
}

// CredentialsService manages OAuth credentials
type CredentialsService interface {
	// Create stores new credentials
	Create(ctx context.Context, creds *domain.Credentials) error

	// Get retrieves credentials by ID
	Get(ctx context.Context, id string) (*domain.Credentials, error)

	// List retrieves all credentials
	List(ctx context.Context) ([]*domain.CredentialSummary, error)

	// Update updates credentials
	Update(ctx context.Context, creds *domain.Credentials) error

	// Delete deletes credentials
	Delete(ctx context.Context, id string) error

	// Refresh refreshes OAuth tokens if needed
	Refresh(ctx context.Context, id string) (*domain.Credentials, error)
}

// AuthProviderService manages OAuth provider configurations
type AuthProviderService interface {
	// Get retrieves auth provider config
	Get(ctx context.Context, providerType domain.ProviderType) (*domain.AuthProvider, error)

	// GetAuthURL generates an OAuth authorization URL
	GetAuthURL(ctx context.Context, providerType domain.ProviderType, state string) (string, error)

	// ExchangeCode exchanges an OAuth code for tokens
	ExchangeCode(ctx context.Context, providerType domain.ProviderType, code string) (*domain.Credentials, error)
}
