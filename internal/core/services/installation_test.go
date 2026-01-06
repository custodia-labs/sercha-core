package services

import (
	"context"
	"testing"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven/mocks"
)

func TestInstallationService_List(t *testing.T) {
	instStore := mocks.NewMockInstallationStore()
	sourceStore := mocks.NewMockSourceStore()

	svc := NewInstallationService(InstallationServiceConfig{
		InstallationStore: instStore,
		SourceStore:       sourceStore,
	})

	// Save some installations
	now := time.Now()
	inst1 := &domain.Installation{
		ID:           "inst-1",
		Name:         "Test GitHub",
		ProviderType: domain.ProviderTypeGitHub,
		AuthMethod:   domain.AuthMethodOAuth2,
		AccountID:    "user1",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	inst2 := &domain.Installation{
		ID:           "inst-2",
		Name:         "Test Google",
		ProviderType: domain.ProviderTypeGoogleDrive,
		AuthMethod:   domain.AuthMethodOAuth2,
		AccountID:    "user2",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	ctx := context.Background()
	_ = instStore.Save(ctx, inst1)
	_ = instStore.Save(ctx, inst2)

	// Test List
	summaries, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(summaries) != 2 {
		t.Errorf("List() got %d installations, want 2", len(summaries))
	}
}

func TestInstallationService_Get(t *testing.T) {
	instStore := mocks.NewMockInstallationStore()
	sourceStore := mocks.NewMockSourceStore()

	svc := NewInstallationService(InstallationServiceConfig{
		InstallationStore: instStore,
		SourceStore:       sourceStore,
	})

	ctx := context.Background()

	// Test not found
	_, err := svc.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("Get() expected error for nonexistent installation")
	}

	// Save an installation
	now := time.Now()
	inst := &domain.Installation{
		ID:           "inst-1",
		Name:         "Test GitHub",
		ProviderType: domain.ProviderTypeGitHub,
		AuthMethod:   domain.AuthMethodOAuth2,
		AccountID:    "user1",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = instStore.Save(ctx, inst)

	// Test found
	summary, err := svc.Get(ctx, "inst-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if summary.ID != "inst-1" {
		t.Errorf("Get() got ID = %s, want inst-1", summary.ID)
	}
	if summary.Name != "Test GitHub" {
		t.Errorf("Get() got Name = %s, want Test GitHub", summary.Name)
	}
}

func TestInstallationService_Delete(t *testing.T) {
	instStore := mocks.NewMockInstallationStore()
	sourceStore := mocks.NewMockSourceStore()

	svc := NewInstallationService(InstallationServiceConfig{
		InstallationStore: instStore,
		SourceStore:       sourceStore,
	})

	ctx := context.Background()

	// Save an installation
	now := time.Now()
	inst := &domain.Installation{
		ID:           "inst-1",
		Name:         "Test GitHub",
		ProviderType: domain.ProviderTypeGitHub,
		AuthMethod:   domain.AuthMethodOAuth2,
		AccountID:    "user1",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = instStore.Save(ctx, inst)

	// Delete should succeed
	err := svc.Delete(ctx, "inst-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	if instStore.Count() != 0 {
		t.Error("Delete() installation still exists")
	}
}

func TestInstallationService_Delete_InUse(t *testing.T) {
	instStore := mocks.NewMockInstallationStore()
	sourceStore := mocks.NewMockSourceStore()

	svc := NewInstallationService(InstallationServiceConfig{
		InstallationStore: instStore,
		SourceStore:       sourceStore,
	})

	ctx := context.Background()

	// Save an installation
	now := time.Now()
	inst := &domain.Installation{
		ID:           "inst-1",
		Name:         "Test GitHub",
		ProviderType: domain.ProviderTypeGitHub,
		AuthMethod:   domain.AuthMethodOAuth2,
		AccountID:    "user1",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = instStore.Save(ctx, inst)

	// Save a source using this installation
	source := &domain.Source{
		ID:             "src-1",
		Name:           "My Repos",
		ProviderType:   domain.ProviderTypeGitHub,
		InstallationID: "inst-1",
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	_ = sourceStore.Save(ctx, source)

	// Delete should fail with ErrInUse
	err := svc.Delete(ctx, "inst-1")
	if err != domain.ErrInUse {
		t.Errorf("Delete() error = %v, want ErrInUse", err)
	}
}

func TestInstallationService_ListByProvider(t *testing.T) {
	instStore := mocks.NewMockInstallationStore()
	sourceStore := mocks.NewMockSourceStore()

	svc := NewInstallationService(InstallationServiceConfig{
		InstallationStore: instStore,
		SourceStore:       sourceStore,
	})

	ctx := context.Background()

	// Save installations for different providers
	now := time.Now()
	inst1 := &domain.Installation{
		ID:           "inst-1",
		Name:         "GitHub 1",
		ProviderType: domain.ProviderTypeGitHub,
		AuthMethod:   domain.AuthMethodOAuth2,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	inst2 := &domain.Installation{
		ID:           "inst-2",
		Name:         "GitHub 2",
		ProviderType: domain.ProviderTypeGitHub,
		AuthMethod:   domain.AuthMethodOAuth2,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	inst3 := &domain.Installation{
		ID:           "inst-3",
		Name:         "Google Drive",
		ProviderType: domain.ProviderTypeGoogleDrive,
		AuthMethod:   domain.AuthMethodOAuth2,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_ = instStore.Save(ctx, inst1)
	_ = instStore.Save(ctx, inst2)
	_ = instStore.Save(ctx, inst3)

	// List by GitHub
	summaries, err := svc.ListByProvider(ctx, domain.ProviderTypeGitHub)
	if err != nil {
		t.Fatalf("ListByProvider() error = %v", err)
	}
	if len(summaries) != 2 {
		t.Errorf("ListByProvider() got %d installations, want 2", len(summaries))
	}

	// List by Google Drive
	summaries, err = svc.ListByProvider(ctx, domain.ProviderTypeGoogleDrive)
	if err != nil {
		t.Fatalf("ListByProvider() error = %v", err)
	}
	if len(summaries) != 1 {
		t.Errorf("ListByProvider() got %d installations, want 1", len(summaries))
	}
}

// MockContainerListerFactory is a mock for testing
type mockContainerListerFactory struct {
	lister              driven.ContainerLister
	supportsContainerFn func(domain.ProviderType) bool
}

func (m *mockContainerListerFactory) Create(ctx context.Context, providerType domain.ProviderType, installationID string) (driven.ContainerLister, error) {
	return m.lister, nil
}

func (m *mockContainerListerFactory) SupportsContainerSelection(providerType domain.ProviderType) bool {
	if m.supportsContainerFn != nil {
		return m.supportsContainerFn(providerType)
	}
	return true
}

// mockContainerLister for testing
type mockContainerLister struct {
	containers []*driven.Container
	nextCursor string
}

func (m *mockContainerLister) ListContainers(ctx context.Context, cursor string) ([]*driven.Container, string, error) {
	return m.containers, m.nextCursor, nil
}

func TestInstallationService_ListContainers(t *testing.T) {
	instStore := mocks.NewMockInstallationStore()
	sourceStore := mocks.NewMockSourceStore()

	// Create mock container lister
	containers := []*driven.Container{
		{ID: "owner/repo1", Name: "repo1", Type: "repository"},
		{ID: "owner/repo2", Name: "repo2", Type: "repository"},
	}
	lister := &mockContainerLister{containers: containers}
	factory := &mockContainerListerFactory{lister: lister}

	svc := NewInstallationService(InstallationServiceConfig{
		InstallationStore:      instStore,
		SourceStore:            sourceStore,
		ContainerListerFactory: factory,
	})

	ctx := context.Background()

	// Save an installation
	now := time.Now()
	inst := &domain.Installation{
		ID:           "inst-1",
		Name:         "Test GitHub",
		ProviderType: domain.ProviderTypeGitHub,
		AuthMethod:   domain.AuthMethodOAuth2,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = instStore.Save(ctx, inst)

	// List containers
	resp, err := svc.ListContainers(ctx, "inst-1", "")
	if err != nil {
		t.Fatalf("ListContainers() error = %v", err)
	}
	if len(resp.Containers) != 2 {
		t.Errorf("ListContainers() got %d containers, want 2", len(resp.Containers))
	}
}

func TestInstallationService_ListContainers_UnsupportedProvider(t *testing.T) {
	instStore := mocks.NewMockInstallationStore()
	sourceStore := mocks.NewMockSourceStore()

	// Create factory that doesn't support container selection for this provider
	factory := &mockContainerListerFactory{
		supportsContainerFn: func(pt domain.ProviderType) bool {
			return false
		},
	}

	svc := NewInstallationService(InstallationServiceConfig{
		InstallationStore:      instStore,
		SourceStore:            sourceStore,
		ContainerListerFactory: factory,
	})

	ctx := context.Background()

	// Save an installation
	now := time.Now()
	inst := &domain.Installation{
		ID:           "inst-1",
		Name:         "Test GitHub",
		ProviderType: domain.ProviderTypeGitHub,
		AuthMethod:   domain.AuthMethodOAuth2,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = instStore.Save(ctx, inst)

	// List containers should return empty list
	resp, err := svc.ListContainers(ctx, "inst-1", "")
	if err != nil {
		t.Fatalf("ListContainers() error = %v", err)
	}
	if len(resp.Containers) != 0 {
		t.Errorf("ListContainers() got %d containers, want 0 for unsupported provider", len(resp.Containers))
	}
}
