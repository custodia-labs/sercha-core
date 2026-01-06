package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// MockInstallationStore is a mock implementation of InstallationStore for testing
type MockInstallationStore struct {
	mu            sync.RWMutex
	installations map[string]*domain.Installation
	byProvider    map[domain.ProviderType]map[string]*domain.Installation
	byAccount     map[string]*domain.Installation // key: providerType:accountID
}

// NewMockInstallationStore creates a new MockInstallationStore
func NewMockInstallationStore() *MockInstallationStore {
	return &MockInstallationStore{
		installations: make(map[string]*domain.Installation),
		byProvider:    make(map[domain.ProviderType]map[string]*domain.Installation),
		byAccount:     make(map[string]*domain.Installation),
	}
}

func (m *MockInstallationStore) Save(ctx context.Context, inst *domain.Installation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.installations[inst.ID] = inst

	if m.byProvider[inst.ProviderType] == nil {
		m.byProvider[inst.ProviderType] = make(map[string]*domain.Installation)
	}
	m.byProvider[inst.ProviderType][inst.ID] = inst

	if inst.AccountID != "" {
		key := string(inst.ProviderType) + ":" + inst.AccountID
		m.byAccount[key] = inst
	}

	return nil
}

func (m *MockInstallationStore) Get(ctx context.Context, id string) (*domain.Installation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, ok := m.installations[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return inst, nil
}

func (m *MockInstallationStore) List(ctx context.Context) ([]*domain.InstallationSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.InstallationSummary
	for _, inst := range m.installations {
		result = append(result, inst.ToSummary())
	}
	return result, nil
}

func (m *MockInstallationStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, ok := m.installations[id]
	if !ok {
		return domain.ErrNotFound
	}

	delete(m.installations, id)
	delete(m.byProvider[inst.ProviderType], id)

	if inst.AccountID != "" {
		key := string(inst.ProviderType) + ":" + inst.AccountID
		delete(m.byAccount, key)
	}

	return nil
}

func (m *MockInstallationStore) GetByProvider(ctx context.Context, providerType domain.ProviderType) ([]*domain.InstallationSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.InstallationSummary
	for _, inst := range m.byProvider[providerType] {
		result = append(result, inst.ToSummary())
	}
	return result, nil
}

func (m *MockInstallationStore) GetByAccountID(ctx context.Context, providerType domain.ProviderType, accountID string) (*domain.Installation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := string(providerType) + ":" + accountID
	inst, ok := m.byAccount[key]
	if !ok {
		return nil, nil
	}
	return inst, nil
}

func (m *MockInstallationStore) UpdateSecrets(ctx context.Context, id string, secrets *domain.InstallationSecrets, expiry *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, ok := m.installations[id]
	if !ok {
		return domain.ErrNotFound
	}

	inst.Secrets = secrets
	inst.OAuthExpiry = expiry
	inst.UpdatedAt = time.Now()

	return nil
}

func (m *MockInstallationStore) UpdateLastUsed(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, ok := m.installations[id]
	if !ok {
		return domain.ErrNotFound
	}

	now := time.Now()
	inst.LastUsedAt = &now

	return nil
}

// Helper methods for testing

func (m *MockInstallationStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installations = make(map[string]*domain.Installation)
	m.byProvider = make(map[domain.ProviderType]map[string]*domain.Installation)
	m.byAccount = make(map[string]*domain.Installation)
}

func (m *MockInstallationStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.installations)
}
