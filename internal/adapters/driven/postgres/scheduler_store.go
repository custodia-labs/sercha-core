package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Verify interface compliance
var _ driven.SchedulerStore = (*SchedulerStore)(nil)

// SchedulerStore implements driven.SchedulerStore using PostgreSQL
type SchedulerStore struct {
	db *DB
}

// NewSchedulerStore creates a new SchedulerStore
func NewSchedulerStore(db *DB) *SchedulerStore {
	return &SchedulerStore{db: db}
}

// GetScheduledTask retrieves a scheduled task by ID
func (s *SchedulerStore) GetScheduledTask(ctx context.Context, id string) (*domain.ScheduledTask, error) {
	query := `
		SELECT id, name, type, team_id, interval_ns, enabled, next_run, last_run, last_error
		FROM scheduled_tasks
		WHERE id = $1
	`

	var task domain.ScheduledTask
	var lastRun sql.NullTime
	var lastError sql.NullString
	var intervalNs int64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.Name,
		&task.Type,
		&task.TeamID,
		&intervalNs,
		&task.Enabled,
		&task.NextRun,
		&lastRun,
		&lastError,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	task.Interval = time.Duration(intervalNs)
	task.LastRun = TimePtr(lastRun)
	task.LastError = lastError.String

	return &task, nil
}

// ListScheduledTasks retrieves all scheduled tasks for a team
func (s *SchedulerStore) ListScheduledTasks(ctx context.Context, teamID string) ([]*domain.ScheduledTask, error) {
	query := `
		SELECT id, name, type, team_id, interval_ns, enabled, next_run, last_run, last_error
		FROM scheduled_tasks
		WHERE team_id = $1
		ORDER BY next_run ASC
	`

	rows, err := s.db.QueryContext(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanScheduledTasks(rows)
}

// SaveScheduledTask creates or updates a scheduled task
func (s *SchedulerStore) SaveScheduledTask(ctx context.Context, task *domain.ScheduledTask) error {
	query := `
		INSERT INTO scheduled_tasks (id, name, type, team_id, interval_ns, enabled, next_run, last_run, last_error, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, '{}')
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			team_id = EXCLUDED.team_id,
			interval_ns = EXCLUDED.interval_ns,
			enabled = EXCLUDED.enabled,
			next_run = EXCLUDED.next_run,
			last_run = EXCLUDED.last_run,
			last_error = EXCLUDED.last_error
	`

	_, err := s.db.ExecContext(ctx, query,
		task.ID,
		task.Name,
		string(task.Type),
		task.TeamID,
		int64(task.Interval),
		task.Enabled,
		task.NextRun,
		NullTime(task.LastRun),
		task.LastError,
	)
	return err
}

// DeleteScheduledTask removes a scheduled task
func (s *SchedulerStore) DeleteScheduledTask(ctx context.Context, id string) error {
	query := `DELETE FROM scheduled_tasks WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// GetDueScheduledTasks retrieves scheduled tasks that are due to run
func (s *SchedulerStore) GetDueScheduledTasks(ctx context.Context) ([]*domain.ScheduledTask, error) {
	query := `
		SELECT id, name, type, team_id, interval_ns, enabled, next_run, last_run, last_error
		FROM scheduled_tasks
		WHERE enabled = true AND next_run <= $1
		ORDER BY next_run ASC
	`

	rows, err := s.db.QueryContext(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanScheduledTasks(rows)
}

// UpdateLastRun updates the last run time and next run time
func (s *SchedulerStore) UpdateLastRun(ctx context.Context, id string, lastError string) error {
	now := time.Now()

	// First get the task to calculate next run
	task, err := s.GetScheduledTask(ctx, id)
	if err != nil {
		return err
	}

	nextRun := now.Add(task.Interval)

	query := `
		UPDATE scheduled_tasks
		SET last_run = $1, next_run = $2, last_error = $3
		WHERE id = $4
	`

	result, err := s.db.ExecContext(ctx, query, now, nextRun, lastError, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (s *SchedulerStore) scanScheduledTasks(rows *sql.Rows) ([]*domain.ScheduledTask, error) {
	var tasks []*domain.ScheduledTask
	for rows.Next() {
		var task domain.ScheduledTask
		var lastRun sql.NullTime
		var lastError sql.NullString
		var intervalNs int64

		err := rows.Scan(
			&task.ID,
			&task.Name,
			&task.Type,
			&task.TeamID,
			&intervalNs,
			&task.Enabled,
			&task.NextRun,
			&lastRun,
			&lastError,
		)
		if err != nil {
			return nil, err
		}

		task.Interval = time.Duration(intervalNs)
		task.LastRun = TimePtr(lastRun)
		task.LastError = lastError.String

		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}
