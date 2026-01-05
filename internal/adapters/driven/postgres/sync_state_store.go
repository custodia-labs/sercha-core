package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Verify interface compliance
var _ driven.SyncStateStore = (*SyncStateStore)(nil)

// SyncStateStore implements driven.SyncStateStore using PostgreSQL
type SyncStateStore struct {
	db *DB
}

// NewSyncStateStore creates a new SyncStateStore
func NewSyncStateStore(db *DB) *SyncStateStore {
	return &SyncStateStore{db: db}
}

// Save creates or updates sync state
func (s *SyncStateStore) Save(ctx context.Context, state *domain.SyncState) error {
	statsJSON, err := json.Marshal(state.Stats)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO sync_states (source_id, status, last_sync_at, next_sync_at, cursor, stats, error, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (source_id) DO UPDATE SET
			status = EXCLUDED.status,
			last_sync_at = EXCLUDED.last_sync_at,
			next_sync_at = EXCLUDED.next_sync_at,
			cursor = EXCLUDED.cursor,
			stats = EXCLUDED.stats,
			error = EXCLUDED.error,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at
	`

	_, err = s.db.ExecContext(ctx, query,
		state.SourceID,
		string(state.Status),
		NullTime(state.LastSyncAt),
		NullTime(state.NextSyncAt),
		state.Cursor,
		statsJSON,
		state.Error,
		NullTime(state.StartedAt),
		NullTime(state.CompletedAt),
	)
	return err
}

// Get retrieves sync state for a source
func (s *SyncStateStore) Get(ctx context.Context, sourceID string) (*domain.SyncState, error) {
	query := `
		SELECT source_id, status, last_sync_at, next_sync_at, cursor, stats, error, started_at, completed_at
		FROM sync_states
		WHERE source_id = $1
	`

	var state domain.SyncState
	var lastSyncAt, nextSyncAt, startedAt, completedAt sql.NullTime
	var cursor, errStr sql.NullString
	var statsJSON []byte

	err := s.db.QueryRowContext(ctx, query, sourceID).Scan(
		&state.SourceID,
		&state.Status,
		&lastSyncAt,
		&nextSyncAt,
		&cursor,
		&statsJSON,
		&errStr,
		&startedAt,
		&completedAt,
	)
	if err == sql.ErrNoRows {
		// Return default state for new source
		return &domain.SyncState{
			SourceID: sourceID,
			Status:   domain.SyncStatusIdle,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	state.LastSyncAt = TimePtr(lastSyncAt)
	state.NextSyncAt = TimePtr(nextSyncAt)
	state.Cursor = cursor.String
	state.Error = errStr.String
	state.StartedAt = TimePtr(startedAt)
	state.CompletedAt = TimePtr(completedAt)

	if len(statsJSON) > 0 {
		if err := json.Unmarshal(statsJSON, &state.Stats); err != nil {
			return nil, err
		}
	}

	return &state, nil
}

// List retrieves sync states for all sources
func (s *SyncStateStore) List(ctx context.Context) ([]*domain.SyncState, error) {
	query := `
		SELECT source_id, status, last_sync_at, next_sync_at, cursor, stats, error, started_at, completed_at
		FROM sync_states
		ORDER BY last_sync_at DESC NULLS LAST
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []*domain.SyncState
	for rows.Next() {
		var state domain.SyncState
		var lastSyncAt, nextSyncAt, startedAt, completedAt sql.NullTime
		var cursor, errStr sql.NullString
		var statsJSON []byte

		err := rows.Scan(
			&state.SourceID,
			&state.Status,
			&lastSyncAt,
			&nextSyncAt,
			&cursor,
			&statsJSON,
			&errStr,
			&startedAt,
			&completedAt,
		)
		if err != nil {
			return nil, err
		}

		state.LastSyncAt = TimePtr(lastSyncAt)
		state.NextSyncAt = TimePtr(nextSyncAt)
		state.Cursor = cursor.String
		state.Error = errStr.String
		state.StartedAt = TimePtr(startedAt)
		state.CompletedAt = TimePtr(completedAt)

		if len(statsJSON) > 0 {
			if err := json.Unmarshal(statsJSON, &state.Stats); err != nil {
				return nil, err
			}
		}

		states = append(states, &state)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return states, nil
}

// Delete deletes sync state for a source
func (s *SyncStateStore) Delete(ctx context.Context, sourceID string) error {
	query := `DELETE FROM sync_states WHERE source_id = $1`
	_, err := s.db.ExecContext(ctx, query, sourceID)
	return err
}

// UpdateStatus updates only the status field
func (s *SyncStateStore) UpdateStatus(ctx context.Context, sourceID string, status domain.SyncStatus) error {
	query := `
		INSERT INTO sync_states (source_id, status)
		VALUES ($1, $2)
		ON CONFLICT (source_id) DO UPDATE SET
			status = EXCLUDED.status
	`
	_, err := s.db.ExecContext(ctx, query, sourceID, string(status))
	return err
}

// UpdateCursor updates the sync cursor
func (s *SyncStateStore) UpdateCursor(ctx context.Context, sourceID string, cursor string) error {
	query := `
		INSERT INTO sync_states (source_id, status, cursor)
		VALUES ($1, $2, $3)
		ON CONFLICT (source_id) DO UPDATE SET
			cursor = EXCLUDED.cursor
	`
	_, err := s.db.ExecContext(ctx, query, sourceID, string(domain.SyncStatusIdle), cursor)
	return err
}
