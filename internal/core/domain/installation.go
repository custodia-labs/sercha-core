package domain

import "time"

// Installation represents a connector installation with stored credentials.
// An installation is the authenticated connection to a provider (GitHub, Google, etc.)
// Sources reference installations to get their authentication context.
type Installation struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	ProviderType ProviderType `json:"provider_type"`
	AuthMethod   AuthMethod   `json:"auth_method"`

	// Secrets contains decrypted secret values (never persisted as-is)
	// This is populated when fetching from store, nil when listing
	Secrets *InstallationSecrets `json:"-"`

	// OAuth metadata (non-secret, safe to expose)
	OAuthTokenType string     `json:"oauth_token_type,omitempty"`
	OAuthExpiry    *time.Time `json:"oauth_expiry,omitempty"`
	OAuthScopes    []string   `json:"oauth_scopes,omitempty"`

	// AccountID is the provider account identifier (email, username)
	// Used for display and uniqueness constraint
	AccountID string `json:"account_id,omitempty"`

	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// InstallationSecrets contains decrypted secret values.
// These are encrypted before storage and decrypted on retrieval.
type InstallationSecrets struct {
	// OAuth2 tokens
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`

	// API Key (for api_key auth method)
	APIKey string `json:"api_key,omitempty"`

	// Service Account JSON (for service_account auth method, e.g., Google)
	ServiceAccountJSON string `json:"service_account_json,omitempty"`
}

// InstallationSummary is a safe view without secrets for listing.
type InstallationSummary struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	ProviderType ProviderType `json:"provider_type"`
	AuthMethod   AuthMethod   `json:"auth_method"`
	AccountID    string       `json:"account_id,omitempty"`
	OAuthExpiry  *time.Time   `json:"oauth_expiry,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	LastUsedAt   *time.Time   `json:"last_used_at,omitempty"`
}

// ToSummary converts Installation to InstallationSummary.
func (i *Installation) ToSummary() *InstallationSummary {
	return &InstallationSummary{
		ID:           i.ID,
		Name:         i.Name,
		ProviderType: i.ProviderType,
		AuthMethod:   i.AuthMethod,
		AccountID:    i.AccountID,
		OAuthExpiry:  i.OAuthExpiry,
		CreatedAt:    i.CreatedAt,
		LastUsedAt:   i.LastUsedAt,
	}
}

// NeedsRefresh returns true if OAuth tokens should be refreshed.
// Returns true if within 5 minutes of expiry.
func (i *Installation) NeedsRefresh() bool {
	if i.OAuthExpiry == nil {
		return false
	}
	return time.Until(*i.OAuthExpiry) < 5*time.Minute
}

// IsExpired returns true if OAuth tokens have expired.
func (i *Installation) IsExpired() bool {
	if i.OAuthExpiry == nil {
		return false
	}
	return time.Now().After(*i.OAuthExpiry)
}

// HasSecrets returns true if the installation has secrets loaded.
func (i *Installation) HasSecrets() bool {
	return i.Secrets != nil
}

// GetAccessToken returns the access token if available.
// For OAuth2: returns the access token
// For API Key/PAT: returns the API key
func (i *Installation) GetAccessToken() string {
	if i.Secrets == nil {
		return ""
	}
	if i.AuthMethod == AuthMethodOAuth2 {
		return i.Secrets.AccessToken
	}
	if i.AuthMethod == AuthMethodAPIKey || i.AuthMethod == AuthMethodPAT {
		return i.Secrets.APIKey
	}
	return ""
}
