package domain

import (
	"testing"
)

func TestProviderTypeConstants(t *testing.T) {
	tests := []struct {
		provider ProviderType
		expected string
	}{
		{ProviderTypeGitHub, "github"},
		{ProviderTypeGitLab, "gitlab"},
		{ProviderTypeBitbucket, "bitbucket"},
		{ProviderTypeSlack, "slack"},
		{ProviderTypeDiscord, "discord"},
		{ProviderTypeMSTeams, "msteams"},
		{ProviderTypeNotion, "notion"},
		{ProviderTypeConfluence, "confluence"},
		{ProviderTypeGoogleDocs, "google_docs"},
		{ProviderTypeJira, "jira"},
		{ProviderTypeLinear, "linear"},
		{ProviderTypeGoogleDrive, "google_drive"},
		{ProviderTypeDropbox, "dropbox"},
		{ProviderTypeOneDrive, "onedrive"},
		{ProviderTypeS3, "s3"},
		{ProviderTypeZendesk, "zendesk"},
		{ProviderTypeIntercom, "intercom"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			if string(tt.provider) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.provider))
			}
		})
	}
}

func TestCoreProviders(t *testing.T) {
	providers := CoreProviders()

	if len(providers) != 11 {
		t.Errorf("expected 11 core providers, got %d", len(providers))
	}

	expectedProviders := map[ProviderType]bool{
		ProviderTypeGitHub:      true,
		ProviderTypeGitLab:      true,
		ProviderTypeSlack:       true,
		ProviderTypeNotion:      true,
		ProviderTypeConfluence:  true,
		ProviderTypeJira:        true,
		ProviderTypeGoogleDrive: true,
		ProviderTypeGoogleDocs:  true,
		ProviderTypeLinear:      true,
		ProviderTypeDropbox:     true,
		ProviderTypeS3:          true,
	}

	for _, provider := range providers {
		if !expectedProviders[provider] {
			t.Errorf("unexpected provider in CoreProviders: %s", provider)
		}
	}
}

func TestAuthProvider(t *testing.T) {
	provider := AuthProvider{
		Type:         ProviderTypeGitHub,
		Name:         "GitHub",
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		Scopes:       []string{"repo", "user"},
		ClientID:     "client-id-123",
		ClientSecret: "client-secret-456",
		RedirectURL:  "https://app.example.com/callback",
	}

	if provider.Type != ProviderTypeGitHub {
		t.Errorf("expected Type github, got %s", provider.Type)
	}
	if provider.Name != "GitHub" {
		t.Errorf("expected Name GitHub, got %s", provider.Name)
	}
	if provider.AuthURL != "https://github.com/login/oauth/authorize" {
		t.Errorf("unexpected AuthURL: %s", provider.AuthURL)
	}
	if provider.TokenURL != "https://github.com/login/oauth/access_token" {
		t.Errorf("unexpected TokenURL: %s", provider.TokenURL)
	}
	if len(provider.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(provider.Scopes))
	}
	if provider.ClientID != "client-id-123" {
		t.Errorf("expected ClientID client-id-123, got %s", provider.ClientID)
	}
	if provider.ClientSecret != "client-secret-456" {
		t.Errorf("expected ClientSecret client-secret-456, got %s", provider.ClientSecret)
	}
}

func TestProviderInfo(t *testing.T) {
	info := ProviderInfo{
		Type:        ProviderTypeGitHub,
		Name:        "GitHub",
		Description: "Connect to GitHub repositories",
		IconURL:     "https://github.com/favicon.ico",
		AuthMethods: []AuthMethod{AuthMethodOAuth2, AuthMethodPAT},
		DocsURL:     "https://docs.example.com/github",
		Available:   true,
	}

	if info.Type != ProviderTypeGitHub {
		t.Errorf("expected Type github, got %s", info.Type)
	}
	if info.Name != "GitHub" {
		t.Errorf("expected Name GitHub, got %s", info.Name)
	}
	if info.Description != "Connect to GitHub repositories" {
		t.Errorf("unexpected Description: %s", info.Description)
	}
	if len(info.AuthMethods) != 2 {
		t.Errorf("expected 2 auth methods, got %d", len(info.AuthMethods))
	}
	if !info.Available {
		t.Error("expected Available to be true")
	}
}

func TestProviderConfig_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		config   *ProviderConfig
		expected bool
	}{
		{
			name: "nil secrets",
			config: &ProviderConfig{
				ProviderType: ProviderTypeGitHub,
				Secrets:      nil,
			},
			expected: false,
		},
		{
			name: "empty secrets",
			config: &ProviderConfig{
				ProviderType: ProviderTypeGitHub,
				Secrets:      &ProviderSecrets{},
			},
			expected: false,
		},
		{
			name: "with client_id",
			config: &ProviderConfig{
				ProviderType: ProviderTypeGitHub,
				Secrets: &ProviderSecrets{
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
				},
			},
			expected: true,
		},
		{
			name: "with api_key only",
			config: &ProviderConfig{
				ProviderType: ProviderTypeS3,
				Secrets: &ProviderSecrets{
					APIKey: "test-api-key",
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsConfigured(); got != tt.expected {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProviderSecrets(t *testing.T) {
	secrets := ProviderSecrets{
		ClientID:     "client-id-123",
		ClientSecret: "client-secret-456",
		APIKey:       "api-key-789",
	}

	if secrets.ClientID != "client-id-123" {
		t.Errorf("expected ClientID client-id-123, got %s", secrets.ClientID)
	}
	if secrets.ClientSecret != "client-secret-456" {
		t.Errorf("expected ClientSecret client-secret-456, got %s", secrets.ClientSecret)
	}
	if secrets.APIKey != "api-key-789" {
		t.Errorf("expected APIKey api-key-789, got %s", secrets.APIKey)
	}
}

func TestProviderConfigSummary(t *testing.T) {
	summary := ProviderConfigSummary{
		ProviderType: ProviderTypeGitHub,
		Enabled:      true,
		HasSecrets:   true,
	}

	if summary.ProviderType != ProviderTypeGitHub {
		t.Errorf("expected ProviderType github, got %s", summary.ProviderType)
	}
	if !summary.Enabled {
		t.Error("expected Enabled to be true")
	}
	if !summary.HasSecrets {
		t.Error("expected HasSecrets to be true")
	}
}
