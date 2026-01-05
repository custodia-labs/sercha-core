package domain

import "time"

// AuthMethod defines how to authenticate with a provider
type AuthMethod string

const (
	AuthMethodOAuth2         AuthMethod = "oauth2"
	AuthMethodAPIKey         AuthMethod = "api_key"
	AuthMethodBasic          AuthMethod = "basic"
	AuthMethodServiceAccount AuthMethod = "service_account"
	AuthMethodPAT            AuthMethod = "pat" // Personal Access Token
)

// Credentials stores authentication credentials for a source connector
type Credentials struct {
	ID           string       `json:"id"`
	ProviderType ProviderType `json:"provider_type"`
	AuthMethod   AuthMethod   `json:"auth_method"`
	Name         string       `json:"name"` // User-friendly name

	// OAuth2 fields
	AccessToken  string     `json:"-"` // Never serialize
	RefreshToken string     `json:"-"` // Never serialize
	TokenExpiry  *time.Time `json:"token_expiry,omitempty"`
	Scopes       []string   `json:"scopes,omitempty"`

	// API Key / PAT
	APIKey string `json:"-"` // Never serialize

	// Basic Auth
	Username string `json:"-"` // Never serialize
	Password string `json:"-"` // Never serialize

	// Service Account (e.g., Google)
	ServiceAccountJSON string `json:"-"` // Never serialize

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by"` // User ID
}

// CredentialSummary provides a safe view without sensitive data
type CredentialSummary struct {
	ID           string       `json:"id"`
	ProviderType ProviderType `json:"provider_type"`
	AuthMethod   AuthMethod   `json:"auth_method"`
	Name         string       `json:"name"`
	HasToken     bool         `json:"has_token"`
	TokenExpiry  *time.Time   `json:"token_expiry,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
}

// ToSummary converts Credentials to CredentialSummary
func (c *Credentials) ToSummary() *CredentialSummary {
	return &CredentialSummary{
		ID:           c.ID,
		ProviderType: c.ProviderType,
		AuthMethod:   c.AuthMethod,
		Name:         c.Name,
		HasToken:     c.AccessToken != "" || c.APIKey != "",
		TokenExpiry:  c.TokenExpiry,
		CreatedAt:    c.CreatedAt,
	}
}

// IsExpired checks if OAuth tokens have expired
func (c *Credentials) IsExpired() bool {
	if c.TokenExpiry == nil {
		return false
	}
	return time.Now().After(*c.TokenExpiry)
}

// NeedsRefresh checks if tokens should be refreshed (within 5 min of expiry)
func (c *Credentials) NeedsRefresh() bool {
	if c.TokenExpiry == nil {
		return false
	}
	return time.Now().Add(5 * time.Minute).After(*c.TokenExpiry)
}
