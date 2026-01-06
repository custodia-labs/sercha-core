package driven

import (
	"context"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// InstallationStore persists connector installations with encrypted secrets.
type InstallationStore interface {
	// Save stores a new installation or updates an existing one.
	// Secrets are encrypted before storage.
	Save(ctx context.Context, inst *domain.Installation) error

	// Get retrieves an installation by ID with decrypted secrets.
	// Returns domain.ErrNotFound if the installation doesn't exist.
	Get(ctx context.Context, id string) (*domain.Installation, error)

	// List retrieves all installations as summaries (no secrets).
	List(ctx context.Context) ([]*domain.InstallationSummary, error)

	// Delete removes an installation by ID.
	// Returns domain.ErrNotFound if the installation doesn't exist.
	Delete(ctx context.Context, id string) error

	// GetByProvider retrieves installations for a provider type (no secrets).
	GetByProvider(ctx context.Context, providerType domain.ProviderType) ([]*domain.InstallationSummary, error)

	// GetByAccountID retrieves an installation by provider type and account ID.
	// Returns nil if not found.
	GetByAccountID(ctx context.Context, providerType domain.ProviderType, accountID string) (*domain.Installation, error)

	// UpdateSecrets updates the encrypted secrets and OAuth metadata.
	// Used after token refresh.
	UpdateSecrets(ctx context.Context, id string, secrets *domain.InstallationSecrets, expiry *time.Time) error

	// UpdateLastUsed updates the last_used_at timestamp.
	UpdateLastUsed(ctx context.Context, id string) error
}
