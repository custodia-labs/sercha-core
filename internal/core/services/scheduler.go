package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Scheduler manages periodic task scheduling.
// It runs on worker nodes and enqueues tasks based on schedules.
type Scheduler struct {
	store     driven.SchedulerStore
	taskQueue driven.TaskQueue
	logger    *slog.Logger

	// Internal state
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
	doneCh   chan struct{}
	interval time.Duration
}

// SchedulerConfig holds configuration for the scheduler.
type SchedulerConfig struct {
	Store        driven.SchedulerStore
	TaskQueue    driven.TaskQueue
	Logger       *slog.Logger
	PollInterval time.Duration // How often to check for due tasks
}

// NewScheduler creates a new scheduler.
func NewScheduler(cfg SchedulerConfig) *Scheduler {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	interval := cfg.PollInterval
	if interval == 0 {
		interval = 30 * time.Second
	}

	return &Scheduler{
		store:     cfg.Store,
		taskQueue: cfg.TaskQueue,
		logger:    logger,
		interval:  interval,
	}
}

// Start begins the scheduler loop.
// It runs until Stop is called or context is cancelled.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.mu.Unlock()

	s.logger.Info("scheduler starting", "poll_interval", s.interval)

	go s.run(ctx)

	return nil
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	close(s.stopCh)
	s.mu.Unlock()

	// Wait for the scheduler to finish
	<-s.doneCh

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	s.logger.Info("scheduler stopped")
}

// run is the main scheduler loop.
func (s *Scheduler) run(ctx context.Context) {
	defer close(s.doneCh)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	s.checkAndEnqueue(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler context cancelled")
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndEnqueue(ctx)
		}
	}
}

// checkAndEnqueue checks for due scheduled tasks and enqueues them.
func (s *Scheduler) checkAndEnqueue(ctx context.Context) {
	tasks, err := s.store.GetDueScheduledTasks(ctx)
	if err != nil {
		s.logger.Error("failed to get due scheduled tasks", "error", err)
		return
	}

	for _, scheduled := range tasks {
		if !scheduled.Enabled || !scheduled.IsDue() {
			continue
		}

		// Create a task for the queue
		task := s.createTask(scheduled)

		// Enqueue the task
		if err := s.taskQueue.Enqueue(ctx, task); err != nil {
			s.logger.Error("failed to enqueue scheduled task",
				"scheduled_id", scheduled.ID,
				"error", err,
			)
			// Update last error
			_ = s.store.UpdateLastRun(ctx, scheduled.ID, err.Error())
			continue
		}

		s.logger.Info("enqueued scheduled task",
			"scheduled_id", scheduled.ID,
			"task_id", task.ID,
			"task_type", task.Type,
		)

		// Update the scheduled task's next run time
		if err := s.store.UpdateLastRun(ctx, scheduled.ID, ""); err != nil {
			s.logger.Warn("failed to update scheduled task last run",
				"scheduled_id", scheduled.ID,
				"error", err,
			)
		}
	}
}

// createTask creates a queue task from a scheduled task.
func (s *Scheduler) createTask(scheduled *domain.ScheduledTask) *domain.Task {
	task := domain.NewTask(scheduled.Type, scheduled.TeamID, nil)

	// Add any payload from the scheduled task configuration
	// For sync_all, no additional payload needed
	// For sync_source, would need source_id in scheduled task payload

	return task
}

// CreateScheduledTask creates a new scheduled task.
func (s *Scheduler) CreateScheduledTask(ctx context.Context, scheduled *domain.ScheduledTask) error {
	return s.store.SaveScheduledTask(ctx, scheduled)
}

// GetScheduledTask retrieves a scheduled task by ID.
func (s *Scheduler) GetScheduledTask(ctx context.Context, id string) (*domain.ScheduledTask, error) {
	return s.store.GetScheduledTask(ctx, id)
}

// ListScheduledTasks lists all scheduled tasks for a team.
func (s *Scheduler) ListScheduledTasks(ctx context.Context, teamID string) ([]*domain.ScheduledTask, error) {
	return s.store.ListScheduledTasks(ctx, teamID)
}

// UpdateScheduledTask updates a scheduled task.
func (s *Scheduler) UpdateScheduledTask(ctx context.Context, scheduled *domain.ScheduledTask) error {
	return s.store.SaveScheduledTask(ctx, scheduled)
}

// DeleteScheduledTask deletes a scheduled task.
func (s *Scheduler) DeleteScheduledTask(ctx context.Context, id string) error {
	return s.store.DeleteScheduledTask(ctx, id)
}

// EnableScheduledTask enables a scheduled task.
func (s *Scheduler) EnableScheduledTask(ctx context.Context, id string) error {
	scheduled, err := s.store.GetScheduledTask(ctx, id)
	if err != nil {
		return err
	}
	scheduled.Enabled = true
	return s.store.SaveScheduledTask(ctx, scheduled)
}

// DisableScheduledTask disables a scheduled task.
func (s *Scheduler) DisableScheduledTask(ctx context.Context, id string) error {
	scheduled, err := s.store.GetScheduledTask(ctx, id)
	if err != nil {
		return err
	}
	scheduled.Enabled = false
	return s.store.SaveScheduledTask(ctx, scheduled)
}

// TriggerNow immediately enqueues a scheduled task (ignoring schedule).
func (s *Scheduler) TriggerNow(ctx context.Context, id string) (*domain.Task, error) {
	scheduled, err := s.store.GetScheduledTask(ctx, id)
	if err != nil {
		return nil, err
	}

	task := s.createTask(scheduled)

	if err := s.taskQueue.Enqueue(ctx, task); err != nil {
		return nil, err
	}

	s.logger.Info("manually triggered scheduled task",
		"scheduled_id", scheduled.ID,
		"task_id", task.ID,
	)

	return task, nil
}
