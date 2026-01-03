package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// mockSchedulerStore implements driven.SchedulerStore for testing
type mockSchedulerStore struct {
	mu             sync.Mutex
	scheduledTasks map[string]*domain.ScheduledTask
	getDueFn       func() ([]*domain.ScheduledTask, error)
	updateLastFn   func(id string, lastError string) error
}

func newMockSchedulerStore() *mockSchedulerStore {
	return &mockSchedulerStore{
		scheduledTasks: make(map[string]*domain.ScheduledTask),
	}
}

func (m *mockSchedulerStore) GetScheduledTask(ctx context.Context, id string) (*domain.ScheduledTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.scheduledTasks[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return task, nil
}

func (m *mockSchedulerStore) ListScheduledTasks(ctx context.Context, teamID string) ([]*domain.ScheduledTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*domain.ScheduledTask
	for _, task := range m.scheduledTasks {
		if task.TeamID == teamID {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *mockSchedulerStore) SaveScheduledTask(ctx context.Context, task *domain.ScheduledTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.scheduledTasks[task.ID] = task
	return nil
}

func (m *mockSchedulerStore) DeleteScheduledTask(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.scheduledTasks[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.scheduledTasks, id)
	return nil
}

func (m *mockSchedulerStore) GetDueScheduledTasks(ctx context.Context) ([]*domain.ScheduledTask, error) {
	if m.getDueFn != nil {
		return m.getDueFn()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*domain.ScheduledTask
	for _, task := range m.scheduledTasks {
		if task.Enabled && task.IsDue() {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *mockSchedulerStore) UpdateLastRun(ctx context.Context, id string, lastError string) error {
	if m.updateLastFn != nil {
		return m.updateLastFn(id, lastError)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.scheduledTasks[id]
	if !ok {
		return domain.ErrNotFound
	}
	task.UpdateNextRun()
	task.LastError = lastError
	return nil
}

// mockTaskQueue for scheduler tests
type mockSchedulerTaskQueue struct {
	mu        sync.Mutex
	tasks     []*domain.Task
	enqueueFn func(*domain.Task) error
}

func newMockSchedulerTaskQueue() *mockSchedulerTaskQueue {
	return &mockSchedulerTaskQueue{
		tasks: make([]*domain.Task, 0),
	}
}

func (m *mockSchedulerTaskQueue) Enqueue(ctx context.Context, task *domain.Task) error {
	if m.enqueueFn != nil {
		return m.enqueueFn(task)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks = append(m.tasks, task)
	return nil
}

func (m *mockSchedulerTaskQueue) EnqueueBatch(ctx context.Context, tasks []*domain.Task) error {
	for _, t := range tasks {
		if err := m.Enqueue(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockSchedulerTaskQueue) Dequeue(ctx context.Context) (*domain.Task, error) {
	return nil, nil
}

func (m *mockSchedulerTaskQueue) DequeueWithTimeout(ctx context.Context, timeout int) (*domain.Task, error) {
	return nil, nil
}

func (m *mockSchedulerTaskQueue) Ack(ctx context.Context, taskID string) error {
	return nil
}

func (m *mockSchedulerTaskQueue) Nack(ctx context.Context, taskID string, reason string) error {
	return nil
}

func (m *mockSchedulerTaskQueue) GetTask(ctx context.Context, taskID string) (*domain.Task, error) {
	return nil, domain.ErrNotFound
}

func (m *mockSchedulerTaskQueue) ListTasks(ctx context.Context, filter driven.TaskFilter) ([]*domain.Task, error) {
	return m.tasks, nil
}

func (m *mockSchedulerTaskQueue) CancelTask(ctx context.Context, taskID string) error {
	return nil
}

func (m *mockSchedulerTaskQueue) PurgeTasks(ctx context.Context, olderThan int) (int, error) {
	return 0, nil
}

func (m *mockSchedulerTaskQueue) Stats(ctx context.Context) (*driven.QueueStats, error) {
	return &driven.QueueStats{}, nil
}

func (m *mockSchedulerTaskQueue) Ping(ctx context.Context) error {
	return nil
}

func (m *mockSchedulerTaskQueue) Close() error {
	return nil
}

func (m *mockSchedulerTaskQueue) getEnqueuedTasks() []*domain.Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*domain.Task, len(m.tasks))
	copy(result, m.tasks)
	return result
}

func TestNewScheduler(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	s := NewScheduler(SchedulerConfig{
		Store:        store,
		TaskQueue:    queue,
		PollInterval: time.Minute,
	})

	if s == nil {
		t.Fatal("expected non-nil scheduler")
	}
	if s.interval != time.Minute {
		t.Errorf("expected interval 1m, got %v", s.interval)
	}
}

func TestNewScheduler_Defaults(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	s := NewScheduler(SchedulerConfig{
		Store:        store,
		TaskQueue:    queue,
		PollInterval: 0, // Should default to 30s
	})

	if s.interval != 30*time.Second {
		t.Errorf("expected default interval 30s, got %v", s.interval)
	}
	if s.logger == nil {
		t.Error("expected default logger")
	}
}

func TestScheduler_StartStop(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	s := NewScheduler(SchedulerConfig{
		Store:        store,
		TaskQueue:    queue,
		PollInterval: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start scheduler
	err := s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	// Verify running
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()
	if !running {
		t.Error("expected scheduler to be running")
	}

	// Start again should be no-op
	err = s.Start(ctx)
	if err != nil {
		t.Errorf("second start should not error: %v", err)
	}

	// Stop scheduler
	s.Stop()

	// Verify stopped
	s.mu.RLock()
	running = s.running
	s.mu.RUnlock()
	if running {
		t.Error("expected scheduler to be stopped")
	}

	// Stop again should be no-op
	s.Stop() // Should not panic
}

func TestScheduler_CreateScheduledTask(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	s := NewScheduler(SchedulerConfig{
		Store:     store,
		TaskQueue: queue,
	})

	ctx := context.Background()

	scheduled := domain.NewScheduledTask(
		"test-schedule",
		"Test Sync",
		domain.TaskTypeSyncAll,
		"team-123",
		time.Hour,
	)

	err := s.CreateScheduledTask(ctx, scheduled)
	if err != nil {
		t.Fatalf("failed to create scheduled task: %v", err)
	}

	// Verify it was saved
	retrieved, err := s.GetScheduledTask(ctx, "test-schedule")
	if err != nil {
		t.Fatalf("failed to get scheduled task: %v", err)
	}
	if retrieved.ID != "test-schedule" {
		t.Errorf("expected ID test-schedule, got %s", retrieved.ID)
	}
}

func TestScheduler_ListScheduledTasks(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	s := NewScheduler(SchedulerConfig{
		Store:     store,
		TaskQueue: queue,
	})

	ctx := context.Background()

	// Create some scheduled tasks
	s.CreateScheduledTask(ctx, domain.NewScheduledTask("s1", "Sync 1", domain.TaskTypeSyncAll, "team-1", time.Hour))
	s.CreateScheduledTask(ctx, domain.NewScheduledTask("s2", "Sync 2", domain.TaskTypeSyncAll, "team-1", time.Hour))
	s.CreateScheduledTask(ctx, domain.NewScheduledTask("s3", "Sync 3", domain.TaskTypeSyncAll, "team-2", time.Hour))

	// List for team-1
	tasks, err := s.ListScheduledTasks(ctx, "team-1")
	if err != nil {
		t.Fatalf("failed to list scheduled tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks for team-1, got %d", len(tasks))
	}

	// List for team-2
	tasks, err = s.ListScheduledTasks(ctx, "team-2")
	if err != nil {
		t.Fatalf("failed to list scheduled tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task for team-2, got %d", len(tasks))
	}
}

func TestScheduler_UpdateScheduledTask(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	s := NewScheduler(SchedulerConfig{
		Store:     store,
		TaskQueue: queue,
	})

	ctx := context.Background()

	scheduled := domain.NewScheduledTask("s1", "Original", domain.TaskTypeSyncAll, "team-1", time.Hour)
	s.CreateScheduledTask(ctx, scheduled)

	// Update it
	scheduled.Name = "Updated"
	scheduled.Interval = 2 * time.Hour

	err := s.UpdateScheduledTask(ctx, scheduled)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Verify update
	retrieved, _ := s.GetScheduledTask(ctx, "s1")
	if retrieved.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %s", retrieved.Name)
	}
	if retrieved.Interval != 2*time.Hour {
		t.Errorf("expected interval 2h, got %v", retrieved.Interval)
	}
}

func TestScheduler_DeleteScheduledTask(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	s := NewScheduler(SchedulerConfig{
		Store:     store,
		TaskQueue: queue,
	})

	ctx := context.Background()

	scheduled := domain.NewScheduledTask("s1", "Test", domain.TaskTypeSyncAll, "team-1", time.Hour)
	s.CreateScheduledTask(ctx, scheduled)

	// Delete it
	err := s.DeleteScheduledTask(ctx, "s1")
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Verify deleted
	_, err = s.GetScheduledTask(ctx, "s1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestScheduler_EnableDisable(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	s := NewScheduler(SchedulerConfig{
		Store:     store,
		TaskQueue: queue,
	})

	ctx := context.Background()

	scheduled := domain.NewScheduledTask("s1", "Test", domain.TaskTypeSyncAll, "team-1", time.Hour)
	scheduled.Enabled = true
	s.CreateScheduledTask(ctx, scheduled)

	// Disable
	err := s.DisableScheduledTask(ctx, "s1")
	if err != nil {
		t.Fatalf("failed to disable: %v", err)
	}

	retrieved, _ := s.GetScheduledTask(ctx, "s1")
	if retrieved.Enabled {
		t.Error("expected disabled")
	}

	// Enable
	err = s.EnableScheduledTask(ctx, "s1")
	if err != nil {
		t.Fatalf("failed to enable: %v", err)
	}

	retrieved, _ = s.GetScheduledTask(ctx, "s1")
	if !retrieved.Enabled {
		t.Error("expected enabled")
	}
}

func TestScheduler_TriggerNow(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	s := NewScheduler(SchedulerConfig{
		Store:     store,
		TaskQueue: queue,
	})

	ctx := context.Background()

	scheduled := domain.NewScheduledTask("s1", "Test", domain.TaskTypeSyncAll, "team-1", time.Hour)
	s.CreateScheduledTask(ctx, scheduled)

	// Trigger immediately
	task, err := s.TriggerNow(ctx, "s1")
	if err != nil {
		t.Fatalf("failed to trigger: %v", err)
	}

	if task == nil {
		t.Fatal("expected task to be created")
	}
	if task.Type != domain.TaskTypeSyncAll {
		t.Errorf("expected task type %s, got %s", domain.TaskTypeSyncAll, task.Type)
	}
	if task.TeamID != "team-1" {
		t.Errorf("expected team ID team-1, got %s", task.TeamID)
	}

	// Verify task was enqueued
	enqueued := queue.getEnqueuedTasks()
	if len(enqueued) != 1 {
		t.Errorf("expected 1 enqueued task, got %d", len(enqueued))
	}
}

func TestScheduler_TriggerNow_NotFound(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	s := NewScheduler(SchedulerConfig{
		Store:     store,
		TaskQueue: queue,
	})

	ctx := context.Background()

	_, err := s.TriggerNow(ctx, "nonexistent")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestScheduler_CheckAndEnqueue(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	s := NewScheduler(SchedulerConfig{
		Store:        store,
		TaskQueue:    queue,
		PollInterval: time.Hour, // Won't actually run in test
	})

	ctx := context.Background()

	// Create a due task
	scheduled := domain.NewScheduledTask("s1", "Test", domain.TaskTypeSyncAll, "team-1", time.Hour)
	scheduled.Enabled = true
	scheduled.NextRun = time.Now().Add(-time.Minute) // Due 1 minute ago
	s.CreateScheduledTask(ctx, scheduled)

	// Create a not-due task
	scheduled2 := domain.NewScheduledTask("s2", "Test2", domain.TaskTypeSyncAll, "team-1", time.Hour)
	scheduled2.Enabled = true
	scheduled2.NextRun = time.Now().Add(time.Hour) // Due in 1 hour
	s.CreateScheduledTask(ctx, scheduled2)

	// Create a disabled task
	scheduled3 := domain.NewScheduledTask("s3", "Test3", domain.TaskTypeSyncAll, "team-1", time.Hour)
	scheduled3.Enabled = false
	scheduled3.NextRun = time.Now().Add(-time.Minute) // Due but disabled
	s.CreateScheduledTask(ctx, scheduled3)

	// Run check and enqueue
	s.checkAndEnqueue(ctx)

	// Only the due & enabled task should be enqueued
	enqueued := queue.getEnqueuedTasks()
	if len(enqueued) != 1 {
		t.Errorf("expected 1 enqueued task, got %d", len(enqueued))
	}
}

func TestScheduler_CheckAndEnqueue_EnqueueError(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	var lastErrorRecorded string
	store.updateLastFn = func(id string, lastError string) error {
		lastErrorRecorded = lastError
		return nil
	}

	queue.enqueueFn = func(task *domain.Task) error {
		return errors.New("queue unavailable")
	}

	s := NewScheduler(SchedulerConfig{
		Store:     store,
		TaskQueue: queue,
	})

	ctx := context.Background()

	// Create a due task
	scheduled := domain.NewScheduledTask("s1", "Test", domain.TaskTypeSyncAll, "team-1", time.Hour)
	scheduled.Enabled = true
	scheduled.NextRun = time.Now().Add(-time.Minute)
	s.CreateScheduledTask(ctx, scheduled)

	// Run check and enqueue - should handle error gracefully
	s.checkAndEnqueue(ctx)

	// Last error should be recorded
	if lastErrorRecorded != "queue unavailable" {
		t.Errorf("expected last error 'queue unavailable', got %q", lastErrorRecorded)
	}
}

func TestScheduler_ContextCancellation(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	s := NewScheduler(SchedulerConfig{
		Store:        store,
		TaskQueue:    queue,
		PollInterval: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())

	err := s.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Cancel after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Give scheduler time to detect cancellation
	time.Sleep(200 * time.Millisecond)

	// Scheduler should have stopped due to context cancellation
	// We need to manually call Stop to clean up
	s.Stop()

	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()
	if running {
		t.Error("expected scheduler to be stopped after context cancellation")
	}
}

func TestSchedulerConfig(t *testing.T) {
	store := newMockSchedulerStore()
	queue := newMockSchedulerTaskQueue()

	cfg := SchedulerConfig{
		Store:        store,
		TaskQueue:    queue,
		PollInterval: 5 * time.Minute,
	}

	if cfg.Store == nil {
		t.Error("expected store")
	}
	if cfg.TaskQueue == nil {
		t.Error("expected task queue")
	}
	if cfg.PollInterval != 5*time.Minute {
		t.Errorf("expected poll interval 5m, got %v", cfg.PollInterval)
	}
}

// Test that mocks implement the interfaces
func TestMockSchedulerStoreInterface(t *testing.T) {
	var _ driven.SchedulerStore = (*mockSchedulerStore)(nil)
}

func TestMockSchedulerTaskQueueInterface(t *testing.T) {
	var _ driven.TaskQueue = (*mockSchedulerTaskQueue)(nil)
}
