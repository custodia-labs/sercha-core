package domain

import (
	"testing"
	"time"
)

func TestSource(t *testing.T) {
	now := time.Now()
	source := &Source{
		ID:           "source-123",
		Name:         "Test Source",
		ProviderType: ProviderTypeGitHub,
		Config: SourceConfig{
			Owner:      "test-org",
			Repository: "test-repo",
			Branch:     "main",
		},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "user-456",
	}

	if source.ID != "source-123" {
		t.Errorf("expected ID source-123, got %s", source.ID)
	}
	if source.Name != "Test Source" {
		t.Errorf("expected Name 'Test Source', got %s", source.Name)
	}
	if source.ProviderType != ProviderTypeGitHub {
		t.Errorf("expected ProviderType github, got %s", source.ProviderType)
	}
	if source.Config.Owner != "test-org" {
		t.Errorf("expected Owner test-org, got %s", source.Config.Owner)
	}
	if source.Config.Repository != "test-repo" {
		t.Errorf("expected Repository test-repo, got %s", source.Config.Repository)
	}
	if !source.Enabled {
		t.Error("expected Enabled to be true")
	}
	if source.CreatedBy != "user-456" {
		t.Errorf("expected CreatedBy user-456, got %s", source.CreatedBy)
	}
}

func TestSourceConfig(t *testing.T) {
	// GitHub config
	ghConfig := SourceConfig{
		CredentialID: "cred-123",
		Owner:        "test-org",
		Repository:   "test-repo",
		Branch:       "main",
		Paths:        []string{"docs/", "src/"},
	}

	if ghConfig.CredentialID != "cred-123" {
		t.Errorf("expected CredentialID cred-123, got %s", ghConfig.CredentialID)
	}
	if len(ghConfig.Paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(ghConfig.Paths))
	}

	// Slack config
	slackConfig := SourceConfig{
		CredentialID: "cred-456",
		Channels:     []string{"general", "engineering"},
	}

	if len(slackConfig.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(slackConfig.Channels))
	}

	// Notion config
	notionConfig := SourceConfig{
		CredentialID: "cred-789",
		DatabaseIDs:  []string{"db-1", "db-2"},
		PageIDs:      []string{"page-1"},
	}

	if len(notionConfig.DatabaseIDs) != 2 {
		t.Errorf("expected 2 database IDs, got %d", len(notionConfig.DatabaseIDs))
	}
	if len(notionConfig.PageIDs) != 1 {
		t.Errorf("expected 1 page ID, got %d", len(notionConfig.PageIDs))
	}

	// Jira config
	jiraConfig := SourceConfig{
		CredentialID: "cred-abc",
		ProjectKeys:  []string{"PROJ1", "PROJ2"},
		JQL:          "status = Open",
	}

	if len(jiraConfig.ProjectKeys) != 2 {
		t.Errorf("expected 2 project keys, got %d", len(jiraConfig.ProjectKeys))
	}
	if jiraConfig.JQL != "status = Open" {
		t.Errorf("expected JQL 'status = Open', got %s", jiraConfig.JQL)
	}
}

func TestSourceSummary(t *testing.T) {
	now := time.Now()
	source := &Source{
		ID:           "source-123",
		Name:         "Test Source",
		ProviderType: ProviderTypeGitHub,
		Enabled:      true,
	}

	summary := &SourceSummary{
		Source:        source,
		DocumentCount: 150,
		LastSyncAt:    &now,
		SyncStatus:    "completed",
	}

	if summary.Source.ID != "source-123" {
		t.Errorf("expected Source ID source-123, got %s", summary.Source.ID)
	}
	if summary.DocumentCount != 150 {
		t.Errorf("expected DocumentCount 150, got %d", summary.DocumentCount)
	}
	if summary.LastSyncAt == nil {
		t.Error("expected LastSyncAt to be set")
	}
	if summary.SyncStatus != "completed" {
		t.Errorf("expected SyncStatus completed, got %s", summary.SyncStatus)
	}
}
