package driving

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// ProviderService manages provider configurations (OAuth app credentials).
// This is an admin-only service for configuring data source connectors.
type ProviderService interface {
	// List returns all available providers with their configuration status.
	List(ctx context.Context) ([]*ProviderListItem, error)

	// GetConfig retrieves provider config by type (no secrets exposed).
	GetConfig(ctx context.Context, providerType domain.ProviderType) (*ProviderConfigResponse, error)

	// SaveConfig creates or updates provider config.
	SaveConfig(ctx context.Context, providerType domain.ProviderType, req SaveProviderConfigRequest) (*ProviderConfigResponse, error)

	// DeleteConfig removes provider config.
	DeleteConfig(ctx context.Context, providerType domain.ProviderType) error
}

// ProviderListItem represents a provider in the list response.
type ProviderListItem struct {
	Type         domain.ProviderType   `json:"type"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	AuthMethods  []domain.AuthMethod   `json:"auth_methods"`
	Configured   bool                  `json:"configured"`
	Enabled      bool                  `json:"enabled"`
	DocsURL      string                `json:"docs_url,omitempty"`
}

// SaveProviderConfigRequest represents a request to create/update provider config.
// Only client_id and client_secret are needed for OAuth providers.
// redirect_uri is derived from BASE_URL automatically.
type SaveProviderConfigRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	APIKey       string `json:"api_key,omitempty"` // For non-OAuth providers (e.g., S3)
	Enabled      *bool  `json:"enabled,omitempty"` // Defaults to true
}

// ProviderConfigResponse represents the response for provider config (no secrets).
type ProviderConfigResponse struct {
	ProviderType domain.ProviderType `json:"provider_type"`
	HasSecrets   bool                `json:"has_secrets"`
	Enabled      bool                `json:"enabled"`
	CreatedAt    string              `json:"created_at"`
	UpdatedAt    string              `json:"updated_at"`
}
