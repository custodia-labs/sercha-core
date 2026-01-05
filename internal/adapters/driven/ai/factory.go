package ai

import (
	"fmt"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Ensure Factory implements AIServiceFactory
var _ driven.AIServiceFactory = (*Factory)(nil)

// Factory creates AI services based on configuration
type Factory struct{}

// NewFactory creates a new AI service factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateEmbeddingService creates an embedding service from settings
func (f *Factory) CreateEmbeddingService(settings *domain.EmbeddingSettings) (driven.EmbeddingService, error) {
	if settings == nil || !settings.IsConfigured() {
		return nil, nil
	}

	switch settings.Provider {
	case domain.AIProviderOpenAI:
		return NewOpenAIEmbedding(settings.APIKey, settings.Model, settings.BaseURL)
	case domain.AIProviderOllama:
		return NewOllamaEmbedding(settings.BaseURL, settings.Model)
	case domain.AIProviderVoyage:
		return NewVoyageEmbedding(settings.APIKey, settings.Model)
	case domain.AIProviderCohere:
		return NewCohereEmbedding(settings.APIKey, settings.Model)
	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidProvider, settings.Provider)
	}
}

// CreateLLMService creates an LLM service from settings
func (f *Factory) CreateLLMService(settings *domain.LLMSettings) (driven.LLMService, error) {
	if settings == nil || !settings.IsConfigured() {
		return nil, nil
	}

	switch settings.Provider {
	case domain.AIProviderOpenAI:
		return NewOpenAILLM(settings.APIKey, settings.Model, settings.BaseURL)
	case domain.AIProviderAnthropic:
		return NewAnthropicLLM(settings.APIKey, settings.Model)
	case domain.AIProviderOllama:
		return NewOllamaLLM(settings.BaseURL, settings.Model)
	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidProvider, settings.Provider)
	}
}

// Placeholder constructors - these will be replaced with actual implementations
// Note: NewOpenAIEmbedding is implemented in openai_embedding.go

func NewOllamaEmbedding(baseURL, model string) (driven.EmbeddingService, error) {
	// TODO: Implement Ollama embedding adapter
	return nil, fmt.Errorf("Ollama embedding adapter not yet implemented")
}

func NewVoyageEmbedding(apiKey, model string) (driven.EmbeddingService, error) {
	// TODO: Implement Voyage embedding adapter
	return nil, fmt.Errorf("Voyage embedding adapter not yet implemented")
}

func NewCohereEmbedding(apiKey, model string) (driven.EmbeddingService, error) {
	// TODO: Implement Cohere embedding adapter
	return nil, fmt.Errorf("Cohere embedding adapter not yet implemented")
}

func NewOpenAILLM(apiKey, model, baseURL string) (driven.LLMService, error) {
	// TODO: Implement OpenAI LLM adapter
	return nil, fmt.Errorf("OpenAI LLM adapter not yet implemented")
}

func NewAnthropicLLM(apiKey, model string) (driven.LLMService, error) {
	// TODO: Implement Anthropic LLM adapter
	return nil, fmt.Errorf("Anthropic LLM adapter not yet implemented")
}

func NewOllamaLLM(baseURL, model string) (driven.LLMService, error) {
	// TODO: Implement Ollama LLM adapter
	return nil, fmt.Errorf("Ollama LLM adapter not yet implemented")
}
