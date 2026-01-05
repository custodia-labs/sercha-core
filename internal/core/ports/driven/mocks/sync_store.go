package mocks

import (
	"context"
	"sync"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// MockSyncStateStore is a mock implementation of SyncStateStore for testing
type MockSyncStateStore struct {
	mu     sync.RWMutex
	states map[string]*domain.SyncState
}

// NewMockSyncStateStore creates a new MockSyncStateStore
func NewMockSyncStateStore() *MockSyncStateStore {
	return &MockSyncStateStore{
		states: make(map[string]*domain.SyncState),
	}
}

func (m *MockSyncStateStore) Save(ctx context.Context, state *domain.SyncState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[state.SourceID] = state
	return nil
}

func (m *MockSyncStateStore) Get(ctx context.Context, sourceID string) (*domain.SyncState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.states[sourceID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return state, nil
}

func (m *MockSyncStateStore) List(ctx context.Context) ([]*domain.SyncState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.SyncState
	for _, state := range m.states {
		result = append(result, state)
	}
	return result, nil
}

func (m *MockSyncStateStore) Delete(ctx context.Context, sourceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.states, sourceID)
	return nil
}

func (m *MockSyncStateStore) UpdateStatus(ctx context.Context, sourceID string, status domain.SyncStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.states[sourceID]
	if !ok {
		return domain.ErrNotFound
	}
	state.Status = status
	return nil
}

func (m *MockSyncStateStore) UpdateCursor(ctx context.Context, sourceID string, cursor string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.states[sourceID]
	if !ok {
		return domain.ErrNotFound
	}
	state.Cursor = cursor
	return nil
}

// Helper methods for testing

func (m *MockSyncStateStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states = make(map[string]*domain.SyncState)
}

func (m *MockSyncStateStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.states)
}
