package driven

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// ProviderConfigStore persists OAuth app configurations per provider type.
// One config per provider - multiple installations can use the same config.
type ProviderConfigStore interface {
	// Save stores or updates provider config (encrypts secrets)
	Save(ctx context.Context, cfg *domain.ProviderConfig) error

	// Get retrieves provider config by type (decrypts secrets)
	Get(ctx context.Context, providerType domain.ProviderType) (*domain.ProviderConfig, error)

	// List retrieves all provider configs (summaries only, no secrets)
	List(ctx context.Context) ([]*domain.ProviderConfigSummary, error)

	// Delete removes provider config
	Delete(ctx context.Context, providerType domain.ProviderType) error

	// GetEnabled retrieves all enabled provider configs (with secrets for OAuth flows)
	GetEnabled(ctx context.Context) ([]*domain.ProviderConfig, error)
}
