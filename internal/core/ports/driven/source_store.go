package driven

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// SourceStore handles source persistence (PostgreSQL)
type SourceStore interface {
	// Save creates or updates a source
	Save(ctx context.Context, source *domain.Source) error

	// Get retrieves a source by ID
	Get(ctx context.Context, id string) (*domain.Source, error)

	// GetByName retrieves a source by name
	GetByName(ctx context.Context, name string) (*domain.Source, error)

	// List retrieves all sources
	List(ctx context.Context) ([]*domain.Source, error)

	// ListEnabled retrieves all enabled sources
	ListEnabled(ctx context.Context) ([]*domain.Source, error)

	// Delete deletes a source
	Delete(ctx context.Context, id string) error

	// SetEnabled updates the enabled status
	SetEnabled(ctx context.Context, id string, enabled bool) error

	// CountByInstallation returns the number of sources using an installation
	CountByInstallation(ctx context.Context, installationID string) (int, error)

	// ListByInstallation returns sources using an installation
	ListByInstallation(ctx context.Context, installationID string) ([]*domain.Source, error)

	// UpdateSelection updates the selected containers for a source
	UpdateSelection(ctx context.Context, id string, containers []string) error
}
