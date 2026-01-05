package driving

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// SyncOrchestrator coordinates document synchronization
type SyncOrchestrator interface {
	// SyncSource triggers a sync for a specific source
	SyncSource(ctx context.Context, sourceID string) (*domain.SyncResult, error)

	// SyncAll triggers a sync for all enabled sources
	SyncAll(ctx context.Context) ([]*domain.SyncResult, error)

	// GetSyncState retrieves the sync state for a source
	GetSyncState(ctx context.Context, sourceID string) (*domain.SyncState, error)

	// ListSyncStates retrieves sync states for all sources
	ListSyncStates(ctx context.Context) ([]*domain.SyncState, error)

	// CancelSync cancels an ongoing sync for a source
	CancelSync(ctx context.Context, sourceID string) error
}

// Scheduler manages periodic sync scheduling
type Scheduler interface {
	// Start begins the sync scheduler
	Start(ctx context.Context) error

	// Stop stops the sync scheduler
	Stop(ctx context.Context) error

	// ScheduleSource schedules a source for sync
	ScheduleSource(ctx context.Context, sourceID string) error

	// UnscheduleSource removes a source from scheduling
	UnscheduleSource(ctx context.Context, sourceID string) error
}
