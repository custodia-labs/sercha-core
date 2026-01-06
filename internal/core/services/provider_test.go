package services

import (
	"context"
	"testing"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driving"
)

// mockProviderConfigStore implements driven.ProviderConfigStore for testing
type mockProviderConfigStore struct {
	configs map[domain.ProviderType]*domain.ProviderConfig
}

func newMockProviderConfigStore() *mockProviderConfigStore {
	return &mockProviderConfigStore{
		configs: make(map[domain.ProviderType]*domain.ProviderConfig),
	}
}

func (m *mockProviderConfigStore) Save(ctx context.Context, cfg *domain.ProviderConfig) error {
	now := time.Now()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now
	m.configs[cfg.ProviderType] = cfg
	return nil
}

func (m *mockProviderConfigStore) Get(ctx context.Context, providerType domain.ProviderType) (*domain.ProviderConfig, error) {
	cfg, ok := m.configs[providerType]
	if !ok {
		return nil, nil
	}
	return cfg, nil
}

func (m *mockProviderConfigStore) List(ctx context.Context) ([]*domain.ProviderConfigSummary, error) {
	summaries := make([]*domain.ProviderConfigSummary, 0, len(m.configs))
	for _, cfg := range m.configs {
		summaries = append(summaries, &domain.ProviderConfigSummary{
			ProviderType: cfg.ProviderType,
			Enabled:      cfg.Enabled,
			HasSecrets:   cfg.IsConfigured(),
			CreatedAt:    cfg.CreatedAt,
			UpdatedAt:    cfg.UpdatedAt,
		})
	}
	return summaries, nil
}

func (m *mockProviderConfigStore) Delete(ctx context.Context, providerType domain.ProviderType) error {
	if _, ok := m.configs[providerType]; !ok {
		return domain.ErrNotFound
	}
	delete(m.configs, providerType)
	return nil
}

func (m *mockProviderConfigStore) GetEnabled(ctx context.Context) ([]*domain.ProviderConfig, error) {
	var configs []*domain.ProviderConfig
	for _, cfg := range m.configs {
		if cfg.Enabled {
			configs = append(configs, cfg)
		}
	}
	return configs, nil
}

func TestProviderService_List(t *testing.T) {
	store := newMockProviderConfigStore()
	svc := NewProviderService(store)

	// Save a GitHub config
	_ = store.Save(context.Background(), &domain.ProviderConfig{
		ProviderType: domain.ProviderTypeGitHub,
		Secrets: &domain.ProviderSecrets{
			ClientID:     "test-client-id",
			ClientSecret: "test-secret",
		},
		Enabled: true,
	})

	// List providers
	providers, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// Should return all core providers
	if len(providers) != len(domain.CoreProviders()) {
		t.Errorf("List() returned %d providers, want %d", len(providers), len(domain.CoreProviders()))
	}

	// Find GitHub in the list
	var github *driving.ProviderListItem
	for _, p := range providers {
		if p.Type == domain.ProviderTypeGitHub {
			github = p
			break
		}
	}

	if github == nil {
		t.Fatal("GitHub not found in provider list")
	}

	if !github.Configured {
		t.Error("GitHub should be marked as configured")
	}
	if !github.Enabled {
		t.Error("GitHub should be marked as enabled")
	}
	if github.Name != "GitHub" {
		t.Errorf("GitHub name = %s, want GitHub", github.Name)
	}
}

func TestProviderService_SaveConfig(t *testing.T) {
	store := newMockProviderConfigStore()
	svc := NewProviderService(store)

	// Save config
	enabled := true
	req := driving.SaveProviderConfigRequest{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Enabled:      &enabled,
	}

	cfg, err := svc.SaveConfig(context.Background(), domain.ProviderTypeGitHub, req)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if cfg.ProviderType != domain.ProviderTypeGitHub {
		t.Errorf("ProviderType = %s, want github", cfg.ProviderType)
	}
	if !cfg.HasSecrets {
		t.Error("HasSecrets should be true")
	}
	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestProviderService_SaveConfig_InvalidProvider(t *testing.T) {
	store := newMockProviderConfigStore()
	svc := NewProviderService(store)

	req := driving.SaveProviderConfigRequest{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	_, err := svc.SaveConfig(context.Background(), domain.ProviderType("invalid"), req)
	if err != domain.ErrInvalidInput {
		t.Errorf("SaveConfig() error = %v, want ErrInvalidInput", err)
	}
}

func TestProviderService_GetConfig(t *testing.T) {
	store := newMockProviderConfigStore()
	svc := NewProviderService(store)

	// Save a config first
	_ = store.Save(context.Background(), &domain.ProviderConfig{
		ProviderType: domain.ProviderTypeGitHub,
		Secrets: &domain.ProviderSecrets{
			ClientID:     "test-client-id",
			ClientSecret: "test-secret",
		},
		Enabled: true,
	})

	// Get config
	cfg, err := svc.GetConfig(context.Background(), domain.ProviderTypeGitHub)
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if cfg.ProviderType != domain.ProviderTypeGitHub {
		t.Errorf("ProviderType = %s, want github", cfg.ProviderType)
	}
	if !cfg.HasSecrets {
		t.Error("HasSecrets should be true")
	}
	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestProviderService_GetConfig_NotFound(t *testing.T) {
	store := newMockProviderConfigStore()
	svc := NewProviderService(store)

	_, err := svc.GetConfig(context.Background(), domain.ProviderTypeGitHub)
	if err != domain.ErrNotFound {
		t.Errorf("GetConfig() error = %v, want ErrNotFound", err)
	}
}

func TestProviderService_DeleteConfig(t *testing.T) {
	store := newMockProviderConfigStore()
	svc := NewProviderService(store)

	// Save a config first
	_ = store.Save(context.Background(), &domain.ProviderConfig{
		ProviderType: domain.ProviderTypeGitHub,
		Secrets: &domain.ProviderSecrets{
			ClientID: "test-client-id",
		},
		Enabled: true,
	})

	// Delete it
	err := svc.DeleteConfig(context.Background(), domain.ProviderTypeGitHub)
	if err != nil {
		t.Fatalf("DeleteConfig() error = %v", err)
	}

	// Verify it's gone
	cfg, _ := store.Get(context.Background(), domain.ProviderTypeGitHub)
	if cfg != nil {
		t.Error("Config should be deleted")
	}
}

func TestProviderService_DeleteConfig_NotFound(t *testing.T) {
	store := newMockProviderConfigStore()
	svc := NewProviderService(store)

	err := svc.DeleteConfig(context.Background(), domain.ProviderTypeGitHub)
	if err != domain.ErrNotFound {
		t.Errorf("DeleteConfig() error = %v, want ErrNotFound", err)
	}
}

func TestProviderMetadata(t *testing.T) {
	tests := []struct {
		providerType domain.ProviderType
		expectedName string
	}{
		{domain.ProviderTypeGitHub, "GitHub"},
		{domain.ProviderTypeGitLab, "GitLab"},
		{domain.ProviderTypeSlack, "Slack"},
		{domain.ProviderTypeNotion, "Notion"},
		{domain.ProviderTypeConfluence, "Confluence"},
		{domain.ProviderTypeJira, "Jira"},
		{domain.ProviderTypeGoogleDrive, "Google Drive"},
		{domain.ProviderTypeGoogleDocs, "Google Docs"},
		{domain.ProviderTypeLinear, "Linear"},
		{domain.ProviderTypeDropbox, "Dropbox"},
		{domain.ProviderTypeS3, "Amazon S3"},
	}

	for _, tt := range tests {
		t.Run(string(tt.providerType), func(t *testing.T) {
			meta := providerMetadata(tt.providerType)
			if meta.name != tt.expectedName {
				t.Errorf("providerMetadata(%s).name = %s, want %s", tt.providerType, meta.name, tt.expectedName)
			}
			if len(meta.authMethods) == 0 {
				t.Errorf("providerMetadata(%s).authMethods should not be empty", tt.providerType)
			}
		})
	}
}

func TestIsValidProvider(t *testing.T) {
	tests := []struct {
		providerType domain.ProviderType
		want         bool
	}{
		{domain.ProviderTypeGitHub, true},
		{domain.ProviderTypeGitLab, true},
		{domain.ProviderTypeSlack, true},
		{domain.ProviderType("invalid"), false},
		{domain.ProviderType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.providerType), func(t *testing.T) {
			if got := isValidProvider(tt.providerType); got != tt.want {
				t.Errorf("isValidProvider(%s) = %v, want %v", tt.providerType, got, tt.want)
			}
		})
	}
}
