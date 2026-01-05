package domain

import "sync"

// RuntimeConfig tracks which services are available at runtime.
// This is determined at startup and can be updated dynamically for AI services.
// Thread-safe for concurrent access.
type RuntimeConfig struct {
	mu sync.RWMutex

	// Static (set at startup, read-only)
	SessionBackend string // "redis" or "postgres"

	// Dynamic capability flags (updated when AI services change)
	embeddingAvailable bool
	llmAvailable       bool
}

// NewRuntimeConfig creates a new RuntimeConfig with initial values
func NewRuntimeConfig(sessionBackend string) *RuntimeConfig {
	return &RuntimeConfig{
		SessionBackend: sessionBackend,
	}
}

// EmbeddingAvailable returns whether embedding service is available
func (c *RuntimeConfig) EmbeddingAvailable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.embeddingAvailable
}

// LLMAvailable returns whether LLM service is available
func (c *RuntimeConfig) LLMAvailable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.llmAvailable
}

// SetEmbeddingAvailable updates the embedding availability flag
func (c *RuntimeConfig) SetEmbeddingAvailable(available bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.embeddingAvailable = available
}

// SetLLMAvailable updates the LLM availability flag
func (c *RuntimeConfig) SetLLMAvailable(available bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.llmAvailable = available
}

// CanDoSemanticSearch returns true if semantic search is possible
func (c *RuntimeConfig) CanDoSemanticSearch() bool {
	return c.EmbeddingAvailable()
}

// CanDoLLMAssisted returns true if LLM features are available
func (c *RuntimeConfig) CanDoLLMAssisted() bool {
	return c.LLMAvailable()
}

// CanDoHybridSearch returns true if hybrid search is possible
func (c *RuntimeConfig) CanDoHybridSearch() bool {
	return c.EmbeddingAvailable()
}

// EffectiveSearchMode returns the best available search mode
func (c *RuntimeConfig) EffectiveSearchMode() SearchMode {
	if c.EmbeddingAvailable() {
		return SearchModeHybrid
	}
	return SearchModeTextOnly
}

// RequiresEmbedding returns true if the given search mode requires embedding
func (mode SearchMode) RequiresEmbedding() bool {
	return mode == SearchModeHybrid || mode == SearchModeSemanticOnly
}
