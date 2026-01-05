package mocks

import (
	"context"
	"sync"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// MockSessionStore is a mock implementation of SessionStore for testing
type MockSessionStore struct {
	mu             sync.RWMutex
	sessions       map[string]*domain.Session
	byToken        map[string]*domain.Session
	byRefreshToken map[string]*domain.Session
	byUser         map[string][]*domain.Session
}

// NewMockSessionStore creates a new MockSessionStore
func NewMockSessionStore() *MockSessionStore {
	return &MockSessionStore{
		sessions:       make(map[string]*domain.Session),
		byToken:        make(map[string]*domain.Session),
		byRefreshToken: make(map[string]*domain.Session),
		byUser:         make(map[string][]*domain.Session),
	}
}

func (m *MockSessionStore) Save(ctx context.Context, session *domain.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.ID] = session
	m.byToken[session.Token] = session
	if session.RefreshToken != "" {
		m.byRefreshToken[session.RefreshToken] = session
	}
	m.byUser[session.UserID] = append(m.byUser[session.UserID], session)
	return nil
}

func (m *MockSessionStore) Get(ctx context.Context, id string) (*domain.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}
	return session, nil
}

func (m *MockSessionStore) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.byToken[token]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}
	return session, nil
}

func (m *MockSessionStore) GetByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.byRefreshToken[refreshToken]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}
	return session, nil
}

func (m *MockSessionStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	if !ok {
		return nil
	}
	delete(m.byToken, session.Token)
	delete(m.sessions, id)
	return nil
}

func (m *MockSessionStore) DeleteByToken(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.byToken[token]
	if !ok {
		return nil
	}
	delete(m.sessions, session.ID)
	delete(m.byToken, token)
	return nil
}

func (m *MockSessionStore) DeleteByUser(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	sessions := m.byUser[userID]
	for _, session := range sessions {
		delete(m.sessions, session.ID)
		delete(m.byToken, session.Token)
	}
	delete(m.byUser, userID)
	return nil
}

func (m *MockSessionStore) ListByUser(ctx context.Context, userID string) ([]*domain.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.byUser[userID], nil
}

// Helper methods for testing

func (m *MockSessionStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = make(map[string]*domain.Session)
	m.byToken = make(map[string]*domain.Session)
	m.byRefreshToken = make(map[string]*domain.Session)
	m.byUser = make(map[string][]*domain.Session)
}

func (m *MockSessionStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
