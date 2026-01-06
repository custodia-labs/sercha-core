package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Ensure TokenProviderFactory implements the interface.
var _ driven.TokenProviderFactory = (*TokenProviderFactory)(nil)

// TokenRefresherFunc is a function type for token refresh operations.
type TokenRefresherFunc func(ctx context.Context, refreshToken string) (*driven.OAuthToken, error)

// TokenProviderFactory creates TokenProviders from installation credentials.
type TokenProviderFactory struct {
	installationStore driven.InstallationStore
	refreshers        map[domain.ProviderType]TokenRefresherFunc
}

// NewTokenProviderFactory creates a new TokenProviderFactory.
func NewTokenProviderFactory(
	installationStore driven.InstallationStore,
) *TokenProviderFactory {
	return &TokenProviderFactory{
		installationStore: installationStore,
		refreshers:        make(map[domain.ProviderType]TokenRefresherFunc),
	}
}

// RegisterRefresher registers a token refresh function for a provider type.
func (f *TokenProviderFactory) RegisterRefresher(
	providerType domain.ProviderType,
	refresher TokenRefresherFunc,
) {
	f.refreshers[providerType] = refresher
}

// Create creates a TokenProvider for an installation.
// It looks up the installation by ID, decrypts credentials, and creates
// an appropriate TokenProvider based on the auth method.
func (f *TokenProviderFactory) Create(ctx context.Context, installationID string) (driven.TokenProvider, error) {
	inst, err := f.installationStore.Get(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("get installation: %w", err)
	}
	if inst == nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInstallationNotFound, installationID)
	}

	return f.CreateFromInstallation(ctx, inst)
}

// CreateFromInstallation creates a TokenProvider from an installation directly.
// Use this when you already have the installation loaded.
func (f *TokenProviderFactory) CreateFromInstallation(ctx context.Context, inst *domain.Installation) (driven.TokenProvider, error) {
	if inst.Secrets == nil {
		return nil, fmt.Errorf("installation has no secrets: %s", inst.ID)
	}

	switch inst.AuthMethod {
	case domain.AuthMethodOAuth2:
		refresher := f.refreshers[inst.ProviderType]
		return NewOAuthTokenProvider(
			inst.ID,
			inst.Secrets.AccessToken,
			inst.Secrets.RefreshToken,
			inst.OAuthExpiry,
			refresher,
			f.installationStore,
		), nil

	case domain.AuthMethodAPIKey:
		return NewStaticTokenProvider(inst.Secrets.APIKey, domain.AuthMethodAPIKey), nil

	case domain.AuthMethodPAT:
		token := inst.Secrets.APIKey
		if token == "" {
			token = inst.Secrets.AccessToken
		}
		return NewStaticTokenProvider(token, domain.AuthMethodPAT), nil

	case domain.AuthMethodServiceAccount:
		// Service accounts typically use the service account JSON as-is
		return NewStaticTokenProvider(inst.Secrets.ServiceAccountJSON, domain.AuthMethodServiceAccount), nil

	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrUnsupportedAuthMethod, inst.AuthMethod)
	}
}

// StaticTokenProvider implements TokenProvider for non-OAuth credentials.
// Used for API keys, PATs, and service accounts.
type StaticTokenProvider struct {
	token      string
	authMethod domain.AuthMethod
}

// NewStaticTokenProvider creates a token provider for static credentials.
func NewStaticTokenProvider(token string, authMethod domain.AuthMethod) *StaticTokenProvider {
	return &StaticTokenProvider{
		token:      token,
		authMethod: authMethod,
	}
}

// GetAccessToken returns the static token.
func (p *StaticTokenProvider) GetAccessToken(ctx context.Context) (string, error) {
	return p.token, nil
}

// GetCredentials returns nil for static tokens - use GetAccessToken instead.
func (p *StaticTokenProvider) GetCredentials(ctx context.Context) (*domain.Credentials, error) {
	return &domain.Credentials{
		AuthMethod: p.authMethod,
		APIKey:     p.token,
	}, nil
}

// AuthMethod returns the authentication method.
func (p *StaticTokenProvider) AuthMethod() domain.AuthMethod {
	return p.authMethod
}

// IsValid returns true - static credentials don't expire.
func (p *StaticTokenProvider) IsValid(ctx context.Context) bool {
	return true
}

// OAuthTokenProvider implements TokenProvider for OAuth2 credentials.
// It automatically refreshes tokens when they expire.
type OAuthTokenProvider struct {
	installationID    string
	accessToken       string
	refreshToken      string
	expiry            *time.Time
	refresher         TokenRefresherFunc
	installationStore driven.InstallationStore
}

// NewOAuthTokenProvider creates a token provider for OAuth credentials.
func NewOAuthTokenProvider(
	installationID string,
	accessToken string,
	refreshToken string,
	expiry *time.Time,
	refresher TokenRefresherFunc,
	installationStore driven.InstallationStore,
) *OAuthTokenProvider {
	return &OAuthTokenProvider{
		installationID:    installationID,
		accessToken:       accessToken,
		refreshToken:      refreshToken,
		expiry:            expiry,
		refresher:         refresher,
		installationStore: installationStore,
	}
}

// GetAccessToken returns a valid access token, refreshing if needed.
func (p *OAuthTokenProvider) GetAccessToken(ctx context.Context) (string, error) {
	// Check if we need to refresh
	if p.needsRefresh() {
		if err := p.refresh(ctx); err != nil {
			return "", fmt.Errorf("refresh token: %w", err)
		}
	}
	return p.accessToken, nil
}

// GetCredentials returns credentials for OAuth.
func (p *OAuthTokenProvider) GetCredentials(ctx context.Context) (*domain.Credentials, error) {
	if p.needsRefresh() {
		if err := p.refresh(ctx); err != nil {
			return nil, fmt.Errorf("refresh token: %w", err)
		}
	}
	return &domain.Credentials{
		AuthMethod:   domain.AuthMethodOAuth2,
		AccessToken:  p.accessToken,
		RefreshToken: p.refreshToken,
		TokenExpiry:  p.expiry,
	}, nil
}

// AuthMethod returns OAuth2.
func (p *OAuthTokenProvider) AuthMethod() domain.AuthMethod {
	return domain.AuthMethodOAuth2
}

// IsValid checks if credentials are valid (not expired or can be refreshed).
func (p *OAuthTokenProvider) IsValid(ctx context.Context) bool {
	// If we have a refresh token and refresher, we can always refresh
	if p.refreshToken != "" && p.refresher != nil {
		return true
	}
	// Otherwise, check if access token is still valid
	return !p.isExpired()
}

// needsRefresh returns true if the token should be refreshed.
func (p *OAuthTokenProvider) needsRefresh() bool {
	if p.expiry == nil {
		return false
	}
	// Refresh if expiring within 5 minutes
	return time.Until(*p.expiry) < 5*time.Minute
}

// isExpired returns true if the token has expired.
func (p *OAuthTokenProvider) isExpired() bool {
	if p.expiry == nil {
		return false
	}
	return time.Now().After(*p.expiry)
}

// refresh refreshes the access token using the refresh token.
func (p *OAuthTokenProvider) refresh(ctx context.Context) error {
	if p.refresher == nil {
		return fmt.Errorf("no token refresher configured")
	}
	if p.refreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	tokens, err := p.refresher(ctx, p.refreshToken)
	if err != nil {
		return err
	}

	// Update local state
	p.accessToken = tokens.AccessToken
	if tokens.RefreshToken != "" {
		p.refreshToken = tokens.RefreshToken
	}
	if tokens.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
		p.expiry = &expiry
	}

	// Update installation store
	if p.installationStore != nil {
		secrets := &domain.InstallationSecrets{
			AccessToken:  p.accessToken,
			RefreshToken: p.refreshToken,
		}
		// Ignore error - we have the tokens locally, persistence failure is non-fatal
		_ = p.installationStore.UpdateSecrets(ctx, p.installationID, secrets, p.expiry)
	}

	return nil
}
