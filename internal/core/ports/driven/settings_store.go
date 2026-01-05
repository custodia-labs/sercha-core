package driven

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// SettingsStore persists team and AI settings
type SettingsStore interface {
	// GetSettings retrieves settings for a team
	GetSettings(ctx context.Context, teamID string) (*domain.Settings, error)

	// SaveSettings persists team settings
	SaveSettings(ctx context.Context, settings *domain.Settings) error

	// GetAISettings retrieves AI-specific settings for a team
	GetAISettings(ctx context.Context, teamID string) (*domain.AISettings, error)

	// SaveAISettings persists AI-specific settings
	SaveAISettings(ctx context.Context, teamID string, settings *domain.AISettings) error
}
