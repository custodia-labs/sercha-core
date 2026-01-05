package driven

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// SyncStateStore handles sync state persistence (PostgreSQL)
type SyncStateStore interface {
	// Save creates or updates sync state
	Save(ctx context.Context, state *domain.SyncState) error

	// Get retrieves sync state for a source
	Get(ctx context.Context, sourceID string) (*domain.SyncState, error)

	// List retrieves sync states for all sources
	List(ctx context.Context) ([]*domain.SyncState, error)

	// Delete deletes sync state for a source
	Delete(ctx context.Context, sourceID string) error

	// UpdateStatus updates only the status field
	UpdateStatus(ctx context.Context, sourceID string, status domain.SyncStatus) error

	// UpdateCursor updates the sync cursor
	UpdateCursor(ctx context.Context, sourceID string, cursor string) error
}
