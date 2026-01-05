package runtime

import (
	"context"
	"sync"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Services holds references to dynamically configurable services.
// AI services (Embedding, LLM) can be updated at runtime via API.
// Thread-safe for concurrent access.
type Services struct {
	mu sync.RWMutex

	// Config tracks capability flags
	config *domain.RuntimeConfig

	// Dynamic services (can be nil, updated at runtime)
	embeddingService driven.EmbeddingService
	llmService       driven.LLMService
}

// NewServices creates a new Services registry
func NewServices(config *domain.RuntimeConfig) *Services {
	return &Services{
		config: config,
	}
}

// Config returns the runtime configuration
func (s *Services) Config() *domain.RuntimeConfig {
	return s.config
}

// EmbeddingService returns the current embedding service (may be nil)
func (s *Services) EmbeddingService() driven.EmbeddingService {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.embeddingService
}

// LLMService returns the current LLM service (may be nil)
func (s *Services) LLMService() driven.LLMService {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.llmService
}

// SetEmbeddingService updates the embedding service.
// Closes the old service if present. Updates config flags.
func (s *Services) SetEmbeddingService(svc driven.EmbeddingService) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close old service
	if s.embeddingService != nil {
		_ = s.embeddingService.Close()
	}

	s.embeddingService = svc
	s.config.SetEmbeddingAvailable(svc != nil)
}

// SetLLMService updates the LLM service.
// Closes the old service if present. Updates config flags.
func (s *Services) SetLLMService(svc driven.LLMService) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close old service
	if s.llmService != nil {
		_ = s.llmService.Close()
	}

	s.llmService = svc
	s.config.SetLLMAvailable(svc != nil)
}

// Close shuts down all services
func (s *Services) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.embeddingService != nil {
		_ = s.embeddingService.Close()
		s.embeddingService = nil
	}
	if s.llmService != nil {
		_ = s.llmService.Close()
		s.llmService = nil
	}

	s.config.SetEmbeddingAvailable(false)
	s.config.SetLLMAvailable(false)

	return nil
}

// ValidateAndSetEmbedding validates connectivity before setting embedding service
func (s *Services) ValidateAndSetEmbedding(ctx context.Context, svc driven.EmbeddingService) error {
	if svc == nil {
		s.SetEmbeddingService(nil)
		return nil
	}

	// Validate connectivity
	if err := svc.HealthCheck(ctx); err != nil {
		_ = svc.Close()
		return err
	}

	s.SetEmbeddingService(svc)
	return nil
}

// ValidateAndSetLLM validates connectivity before setting LLM service
func (s *Services) ValidateAndSetLLM(ctx context.Context, svc driven.LLMService) error {
	if svc == nil {
		s.SetLLMService(nil)
		return nil
	}

	// Validate connectivity
	if err := svc.Ping(ctx); err != nil {
		_ = svc.Close()
		return err
	}

	s.SetLLMService(svc)
	return nil
}
