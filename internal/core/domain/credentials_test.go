package domain

import (
	"testing"
	"time"
)

func TestAuthMethodConstants(t *testing.T) {
	if AuthMethodOAuth2 != "oauth2" {
		t.Errorf("expected AuthMethodOAuth2 = 'oauth2', got %s", AuthMethodOAuth2)
	}
	if AuthMethodAPIKey != "api_key" {
		t.Errorf("expected AuthMethodAPIKey = 'api_key', got %s", AuthMethodAPIKey)
	}
	if AuthMethodBasic != "basic" {
		t.Errorf("expected AuthMethodBasic = 'basic', got %s", AuthMethodBasic)
	}
	if AuthMethodServiceAccount != "service_account" {
		t.Errorf("expected AuthMethodServiceAccount = 'service_account', got %s", AuthMethodServiceAccount)
	}
	if AuthMethodPAT != "pat" {
		t.Errorf("expected AuthMethodPAT = 'pat', got %s", AuthMethodPAT)
	}
}

func TestCredentials(t *testing.T) {
	now := time.Now()
	expiry := now.Add(1 * time.Hour)

	creds := &Credentials{
		ID:           "cred-123",
		ProviderType: ProviderTypeGitHub,
		AuthMethod:   AuthMethodOAuth2,
		Name:         "GitHub OAuth",
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		TokenExpiry:  &expiry,
		Scopes:       []string{"repo", "user"},
		CreatedAt:    now,
		UpdatedAt:    now,
		CreatedBy:    "user-789",
	}

	if creds.ID != "cred-123" {
		t.Errorf("expected ID cred-123, got %s", creds.ID)
	}
	if creds.ProviderType != ProviderTypeGitHub {
		t.Errorf("expected ProviderType github, got %s", creds.ProviderType)
	}
	if creds.AuthMethod != AuthMethodOAuth2 {
		t.Errorf("expected AuthMethod oauth2, got %s", creds.AuthMethod)
	}
	if len(creds.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(creds.Scopes))
	}
}

func TestCredentialsToSummary(t *testing.T) {
	now := time.Now()
	expiry := now.Add(1 * time.Hour)

	creds := &Credentials{
		ID:           "cred-123",
		ProviderType: ProviderTypeGitHub,
		AuthMethod:   AuthMethodOAuth2,
		Name:         "GitHub OAuth",
		AccessToken:  "secret-token",
		TokenExpiry:  &expiry,
		CreatedAt:    now,
	}

	summary := creds.ToSummary()

	if summary.ID != creds.ID {
		t.Errorf("expected ID %s, got %s", creds.ID, summary.ID)
	}
	if summary.ProviderType != creds.ProviderType {
		t.Errorf("expected ProviderType %s, got %s", creds.ProviderType, summary.ProviderType)
	}
	if summary.AuthMethod != creds.AuthMethod {
		t.Errorf("expected AuthMethod %s, got %s", creds.AuthMethod, summary.AuthMethod)
	}
	if summary.Name != creds.Name {
		t.Errorf("expected Name %s, got %s", creds.Name, summary.Name)
	}
	if !summary.HasToken {
		t.Error("expected HasToken to be true")
	}
	if summary.TokenExpiry == nil {
		t.Error("expected TokenExpiry to be set")
	}
}

func TestCredentialsIsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		expiry   *time.Time
		expected bool
	}{
		{
			name:     "nil expiry",
			expiry:   nil,
			expected: false,
		},
		{
			name: "expired",
			expiry: func() *time.Time {
				t := now.Add(-1 * time.Hour)
				return &t
			}(),
			expected: true,
		},
		{
			name: "not expired",
			expiry: func() *time.Time {
				t := now.Add(1 * time.Hour)
				return &t
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := &Credentials{TokenExpiry: tt.expiry}
			if creds.IsExpired() != tt.expected {
				t.Errorf("expected IsExpired() = %v", tt.expected)
			}
		})
	}
}

func TestCredentialsNeedsRefresh(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		expiry   *time.Time
		expected bool
	}{
		{
			name:     "nil expiry",
			expiry:   nil,
			expected: false,
		},
		{
			name: "needs refresh (within 5 min)",
			expiry: func() *time.Time {
				t := now.Add(3 * time.Minute)
				return &t
			}(),
			expected: true,
		},
		{
			name: "does not need refresh",
			expiry: func() *time.Time {
				t := now.Add(1 * time.Hour)
				return &t
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := &Credentials{TokenExpiry: tt.expiry}
			if creds.NeedsRefresh() != tt.expected {
				t.Errorf("expected NeedsRefresh() = %v", tt.expected)
			}
		})
	}
}

func TestCredentialSummaryHasToken(t *testing.T) {
	// With access token
	credsWithToken := &Credentials{AccessToken: "token"}
	summary := credsWithToken.ToSummary()
	if !summary.HasToken {
		t.Error("expected HasToken true when AccessToken is set")
	}

	// With API key
	credsWithAPIKey := &Credentials{APIKey: "api-key"}
	summary = credsWithAPIKey.ToSummary()
	if !summary.HasToken {
		t.Error("expected HasToken true when APIKey is set")
	}

	// Without token
	credsNoToken := &Credentials{}
	summary = credsNoToken.ToSummary()
	if summary.HasToken {
		t.Error("expected HasToken false when no token is set")
	}
}
