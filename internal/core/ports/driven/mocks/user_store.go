package mocks

import (
	"context"
	"sync"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// MockUserStore is a mock implementation of UserStore for testing
type MockUserStore struct {
	mu    sync.RWMutex
	users map[string]*domain.User
	byEmail map[string]*domain.User
}

// NewMockUserStore creates a new MockUserStore
func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		users:   make(map[string]*domain.User),
		byEmail: make(map[string]*domain.User),
	}
}

func (m *MockUserStore) Save(ctx context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *MockUserStore) Get(ctx context.Context, id string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	user, ok := m.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

func (m *MockUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	user, ok := m.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

func (m *MockUserStore) List(ctx context.Context, teamID string) ([]*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.User
	for _, user := range m.users {
		if user.TeamID == teamID {
			result = append(result, user)
		}
	}
	return result, nil
}

func (m *MockUserStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	delete(m.byEmail, user.Email)
	delete(m.users, id)
	return nil
}

func (m *MockUserStore) UpdateLastLogin(ctx context.Context, id string) error {
	return nil
}

// Helper methods for testing

func (m *MockUserStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users = make(map[string]*domain.User)
	m.byEmail = make(map[string]*domain.User)
}

func (m *MockUserStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users)
}
