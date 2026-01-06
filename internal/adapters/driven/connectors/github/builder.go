package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Ensure Builder implements the interface.
var _ driven.ConnectorBuilder = (*Builder)(nil)

// Builder creates GitHub connectors.
type Builder struct {
	config *Config
}

// NewBuilder creates a new GitHub connector builder.
func NewBuilder() *Builder {
	return &Builder{
		config: DefaultConfig(),
	}
}

// NewBuilderWithConfig creates a builder with custom configuration.
func NewBuilderWithConfig(config *Config) *Builder {
	return &Builder{
		config: config,
	}
}

// Type returns the provider type.
func (b *Builder) Type() domain.ProviderType {
	return domain.ProviderTypeGitHub
}

// Build creates a GitHub connector scoped to a specific repository.
// containerID format: "owner/repo"
func (b *Builder) Build(ctx context.Context, tokenProvider driven.TokenProvider, containerID string) (driven.Connector, error) {
	if containerID == "" {
		return nil, fmt.Errorf("containerID is required for GitHub connector (format: owner/repo)")
	}

	owner, repo, err := ParseContainerID(containerID)
	if err != nil {
		return nil, err
	}

	return NewConnector(tokenProvider, owner, repo, b.config), nil
}

// SupportsOAuth returns true - GitHub supports OAuth2.
func (b *Builder) SupportsOAuth() bool {
	return true
}

// OAuthConfig returns OAuth configuration for GitHub.
func (b *Builder) OAuthConfig() *driven.OAuthConfig {
	return &driven.OAuthConfig{
		AuthURL:     "https://github.com/login/oauth/authorize",
		TokenURL:    "https://github.com/login/oauth/access_token",
		Scopes:      []string{"repo", "read:user", "user:email"},
		UserInfoURL: "https://api.github.com/user",
	}
}

// SupportsContainerSelection returns true - GitHub supports repository selection.
func (b *Builder) SupportsContainerSelection() bool {
	return true
}

// ParseContainerID parses a container ID into owner and repo.
// Format: "owner/repo"
func ParseContainerID(containerID string) (owner, repo string, err error) {
	parts := strings.SplitN(containerID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid container ID format: %q (expected: owner/repo)", containerID)
	}
	return parts[0], parts[1], nil
}

// FormatContainerID formats owner and repo into a container ID.
func FormatContainerID(owner, repo string) string {
	return fmt.Sprintf("%s/%s", owner, repo)
}
