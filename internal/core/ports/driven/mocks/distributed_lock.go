package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockDistributedLock is a mock implementation of DistributedLock for testing.
// It simulates lock behavior with in-memory state and supports custom behavior injection.
type MockDistributedLock struct {
	mu    sync.Mutex
	locks map[string]lockEntry

	// Custom behavior hooks (optional)
	AcquireFn func(name string, ttl time.Duration) (bool, error)
	ReleaseFn func(name string) error
	ExtendFn  func(name string, ttl time.Duration) error
	PingFn    func() error
}

type lockEntry struct {
	owner  string
	expiry time.Time
}

// NewMockDistributedLock creates a new mock distributed lock.
func NewMockDistributedLock() *MockDistributedLock {
	return &MockDistributedLock{
		locks: make(map[string]lockEntry),
	}
}

// Acquire attempts to acquire a named lock.
// If AcquireFn is set, it delegates to that function.
// Otherwise, it uses internal state to track locks with TTL.
func (m *MockDistributedLock) Acquire(ctx context.Context, name string, ttl time.Duration) (bool, error) {
	if m.AcquireFn != nil {
		return m.AcquireFn(name, ttl)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if lock exists and is not expired
	if entry, exists := m.locks[name]; exists && time.Now().Before(entry.expiry) {
		return false, nil // Lock held
	}

	// Acquire lock
	m.locks[name] = lockEntry{
		owner:  "mock-owner",
		expiry: time.Now().Add(ttl),
	}
	return true, nil
}

// Release releases a named lock.
// If ReleaseFn is set, it delegates to that function.
func (m *MockDistributedLock) Release(ctx context.Context, name string) error {
	if m.ReleaseFn != nil {
		return m.ReleaseFn(name)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.locks, name)
	return nil
}

// Extend extends the TTL of a held lock.
// If ExtendFn is set, it delegates to that function.
func (m *MockDistributedLock) Extend(ctx context.Context, name string, ttl time.Duration) error {
	if m.ExtendFn != nil {
		return m.ExtendFn(name, ttl)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.locks[name]
	if !exists || time.Now().After(entry.expiry) {
		return fmt.Errorf("lock %s not held", name)
	}

	m.locks[name] = lockEntry{
		owner:  entry.owner,
		expiry: time.Now().Add(ttl),
	}
	return nil
}

// Ping checks backend health.
// If PingFn is set, it delegates to that function.
func (m *MockDistributedLock) Ping(ctx context.Context) error {
	if m.PingFn != nil {
		return m.PingFn()
	}
	return nil
}

// Reset clears all locks (useful between tests).
func (m *MockDistributedLock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.locks = make(map[string]lockEntry)
}

// IsHeld checks if a lock is currently held (for test assertions).
func (m *MockDistributedLock) IsHeld(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.locks[name]
	return exists && time.Now().Before(entry.expiry)
}

// SetLockHeld forces a lock to be held (for test setup).
func (m *MockDistributedLock) SetLockHeld(name string, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.locks[name] = lockEntry{
		owner:  "external-owner",
		expiry: time.Now().Add(ttl),
	}
}
