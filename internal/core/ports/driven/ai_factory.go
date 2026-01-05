package driven

import (
	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// AIServiceFactory creates AI services based on configuration
type AIServiceFactory interface {
	// CreateEmbeddingService creates an embedding service from settings
	// Returns nil, nil if settings are not configured
	CreateEmbeddingService(settings *domain.EmbeddingSettings) (EmbeddingService, error)

	// CreateLLMService creates an LLM service from settings
	// Returns nil, nil if settings are not configured
	CreateLLMService(settings *domain.LLMSettings) (LLMService, error)
}
