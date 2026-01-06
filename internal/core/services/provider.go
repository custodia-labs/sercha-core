package services

import (
	"context"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driving"
)

// Ensure providerService implements ProviderService
var _ driving.ProviderService = (*providerService)(nil)

// providerService implements the ProviderService interface.
// It manages OAuth app configurations for data source providers.
type providerService struct {
	configStore driven.ProviderConfigStore
}

// NewProviderService creates a new ProviderService.
func NewProviderService(configStore driven.ProviderConfigStore) driving.ProviderService {
	return &providerService{
		configStore: configStore,
	}
}

// List returns all available providers with their configuration status.
func (s *providerService) List(ctx context.Context) ([]*driving.ProviderListItem, error) {
	// Get configuration status for all providers
	configs, err := s.configStore.List(ctx)
	if err != nil {
		return nil, err
	}

	// Create a map for quick lookup
	configMap := make(map[domain.ProviderType]*domain.ProviderConfigSummary)
	for _, cfg := range configs {
		configMap[cfg.ProviderType] = cfg
	}

	// Build list with all core providers
	coreProviders := domain.CoreProviders()
	items := make([]*driving.ProviderListItem, 0, len(coreProviders))

	for _, providerType := range coreProviders {
		info := providerMetadata(providerType)
		item := &driving.ProviderListItem{
			Type:        providerType,
			Name:        info.name,
			Description: info.description,
			AuthMethods: info.authMethods,
			DocsURL:     info.docsURL,
			Configured:  false,
			Enabled:     false,
		}

		// Check if configured
		if cfg, ok := configMap[providerType]; ok {
			item.Configured = cfg.HasSecrets
			item.Enabled = cfg.Enabled
		}

		items = append(items, item)
	}

	return items, nil
}

// GetConfig retrieves provider config by type (no secrets exposed).
func (s *providerService) GetConfig(ctx context.Context, providerType domain.ProviderType) (*driving.ProviderConfigResponse, error) {
	cfg, err := s.configStore.Get(ctx, providerType)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, domain.ErrNotFound
	}

	return &driving.ProviderConfigResponse{
		ProviderType: cfg.ProviderType,
		HasSecrets:   cfg.IsConfigured(),
		Enabled:      cfg.Enabled,
		CreatedAt:    cfg.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    cfg.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// SaveConfig creates or updates provider config.
func (s *providerService) SaveConfig(ctx context.Context, providerType domain.ProviderType, req driving.SaveProviderConfigRequest) (*driving.ProviderConfigResponse, error) {
	// Validate provider type
	if !isValidProvider(providerType) {
		return nil, domain.ErrInvalidInput
	}

	// Build config
	cfg := &domain.ProviderConfig{
		ProviderType: providerType,
		Enabled:      true, // Default to enabled
	}

	// Set enabled from request if provided
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}

	// Set secrets
	cfg.Secrets = &domain.ProviderSecrets{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		APIKey:       req.APIKey,
	}

	// Save
	if err := s.configStore.Save(ctx, cfg); err != nil {
		return nil, err
	}

	// Reload to get timestamps
	saved, err := s.configStore.Get(ctx, providerType)
	if err != nil {
		return nil, err
	}

	return &driving.ProviderConfigResponse{
		ProviderType: saved.ProviderType,
		HasSecrets:   saved.IsConfigured(),
		Enabled:      saved.Enabled,
		CreatedAt:    saved.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    saved.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// DeleteConfig removes provider config.
func (s *providerService) DeleteConfig(ctx context.Context, providerType domain.ProviderType) error {
	return s.configStore.Delete(ctx, providerType)
}

// providerMeta holds static metadata about a provider.
type providerMeta struct {
	name        string
	description string
	authMethods []domain.AuthMethod
	docsURL     string
}

// providerMetadata returns static metadata for a provider type.
func providerMetadata(pt domain.ProviderType) providerMeta {
	switch pt {
	case domain.ProviderTypeGitHub:
		return providerMeta{
			name:        "GitHub",
			description: "Index repositories, issues, pull requests, and wikis",
			authMethods: []domain.AuthMethod{domain.AuthMethodOAuth2, domain.AuthMethodPAT},
			docsURL:     "https://docs.sercha.dev/connectors/github",
		}
	case domain.ProviderTypeGitLab:
		return providerMeta{
			name:        "GitLab",
			description: "Index repositories, issues, merge requests, and wikis",
			authMethods: []domain.AuthMethod{domain.AuthMethodOAuth2, domain.AuthMethodPAT},
			docsURL:     "https://docs.sercha.dev/connectors/gitlab",
		}
	case domain.ProviderTypeSlack:
		return providerMeta{
			name:        "Slack",
			description: "Index channels, threads, and messages",
			authMethods: []domain.AuthMethod{domain.AuthMethodOAuth2},
			docsURL:     "https://docs.sercha.dev/connectors/slack",
		}
	case domain.ProviderTypeNotion:
		return providerMeta{
			name:        "Notion",
			description: "Index pages, databases, and documents",
			authMethods: []domain.AuthMethod{domain.AuthMethodOAuth2},
			docsURL:     "https://docs.sercha.dev/connectors/notion",
		}
	case domain.ProviderTypeConfluence:
		return providerMeta{
			name:        "Confluence",
			description: "Index spaces, pages, and blog posts",
			authMethods: []domain.AuthMethod{domain.AuthMethodOAuth2, domain.AuthMethodAPIKey},
			docsURL:     "https://docs.sercha.dev/connectors/confluence",
		}
	case domain.ProviderTypeJira:
		return providerMeta{
			name:        "Jira",
			description: "Index projects, issues, and comments",
			authMethods: []domain.AuthMethod{domain.AuthMethodOAuth2, domain.AuthMethodAPIKey},
			docsURL:     "https://docs.sercha.dev/connectors/jira",
		}
	case domain.ProviderTypeGoogleDrive:
		return providerMeta{
			name:        "Google Drive",
			description: "Index files, folders, and shared drives",
			authMethods: []domain.AuthMethod{domain.AuthMethodOAuth2, domain.AuthMethodServiceAccount},
			docsURL:     "https://docs.sercha.dev/connectors/google-drive",
		}
	case domain.ProviderTypeGoogleDocs:
		return providerMeta{
			name:        "Google Docs",
			description: "Index Google Docs documents",
			authMethods: []domain.AuthMethod{domain.AuthMethodOAuth2, domain.AuthMethodServiceAccount},
			docsURL:     "https://docs.sercha.dev/connectors/google-docs",
		}
	case domain.ProviderTypeLinear:
		return providerMeta{
			name:        "Linear",
			description: "Index issues, projects, and comments",
			authMethods: []domain.AuthMethod{domain.AuthMethodOAuth2, domain.AuthMethodAPIKey},
			docsURL:     "https://docs.sercha.dev/connectors/linear",
		}
	case domain.ProviderTypeDropbox:
		return providerMeta{
			name:        "Dropbox",
			description: "Index files and folders",
			authMethods: []domain.AuthMethod{domain.AuthMethodOAuth2},
			docsURL:     "https://docs.sercha.dev/connectors/dropbox",
		}
	case domain.ProviderTypeS3:
		return providerMeta{
			name:        "Amazon S3",
			description: "Index objects from S3 buckets",
			authMethods: []domain.AuthMethod{domain.AuthMethodAPIKey},
			docsURL:     "https://docs.sercha.dev/connectors/s3",
		}
	default:
		return providerMeta{
			name:        string(pt),
			description: "Data source connector",
			authMethods: []domain.AuthMethod{domain.AuthMethodOAuth2},
		}
	}
}

// isValidProvider checks if the provider type is valid for Sercha Core.
func isValidProvider(pt domain.ProviderType) bool {
	for _, p := range domain.CoreProviders() {
		if p == pt {
			return true
		}
	}
	return false
}
