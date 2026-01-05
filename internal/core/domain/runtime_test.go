package domain

import (
	"testing"
)

func TestNewRuntimeConfig(t *testing.T) {
	config := NewRuntimeConfig("postgres")

	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if config.SessionBackend != "postgres" {
		t.Errorf("expected postgres, got %s", config.SessionBackend)
	}
	if config.EmbeddingAvailable() {
		t.Error("expected embedding to be unavailable initially")
	}
	if config.LLMAvailable() {
		t.Error("expected LLM to be unavailable initially")
	}
}

func TestRuntimeConfig_EmbeddingAvailable(t *testing.T) {
	config := NewRuntimeConfig("redis")

	// Initially unavailable
	if config.EmbeddingAvailable() {
		t.Error("expected embedding to be unavailable initially")
	}

	// Set available
	config.SetEmbeddingAvailable(true)
	if !config.EmbeddingAvailable() {
		t.Error("expected embedding to be available after setting")
	}

	// Set unavailable
	config.SetEmbeddingAvailable(false)
	if config.EmbeddingAvailable() {
		t.Error("expected embedding to be unavailable after clearing")
	}
}

func TestRuntimeConfig_LLMAvailable(t *testing.T) {
	config := NewRuntimeConfig("postgres")

	// Initially unavailable
	if config.LLMAvailable() {
		t.Error("expected LLM to be unavailable initially")
	}

	// Set available
	config.SetLLMAvailable(true)
	if !config.LLMAvailable() {
		t.Error("expected LLM to be available after setting")
	}

	// Set unavailable
	config.SetLLMAvailable(false)
	if config.LLMAvailable() {
		t.Error("expected LLM to be unavailable after clearing")
	}
}

func TestRuntimeConfig_CanDoSemanticSearch(t *testing.T) {
	config := NewRuntimeConfig("postgres")

	// Without embedding
	if config.CanDoSemanticSearch() {
		t.Error("expected CanDoSemanticSearch to be false without embedding")
	}

	// With embedding
	config.SetEmbeddingAvailable(true)
	if !config.CanDoSemanticSearch() {
		t.Error("expected CanDoSemanticSearch to be true with embedding")
	}
}

func TestRuntimeConfig_CanDoLLMAssisted(t *testing.T) {
	config := NewRuntimeConfig("postgres")

	// Without LLM
	if config.CanDoLLMAssisted() {
		t.Error("expected CanDoLLMAssisted to be false without LLM")
	}

	// With LLM
	config.SetLLMAvailable(true)
	if !config.CanDoLLMAssisted() {
		t.Error("expected CanDoLLMAssisted to be true with LLM")
	}
}

func TestRuntimeConfig_CanDoHybridSearch(t *testing.T) {
	config := NewRuntimeConfig("postgres")

	// Without embedding
	if config.CanDoHybridSearch() {
		t.Error("expected CanDoHybridSearch to be false without embedding")
	}

	// With embedding
	config.SetEmbeddingAvailable(true)
	if !config.CanDoHybridSearch() {
		t.Error("expected CanDoHybridSearch to be true with embedding")
	}
}

func TestRuntimeConfig_EffectiveSearchMode(t *testing.T) {
	tests := []struct {
		name      string
		embedding bool
		expected  SearchMode
	}{
		{
			name:      "no embedding - text only",
			embedding: false,
			expected:  SearchModeTextOnly,
		},
		{
			name:      "with embedding - hybrid",
			embedding: true,
			expected:  SearchModeHybrid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewRuntimeConfig("postgres")
			config.SetEmbeddingAvailable(tt.embedding)

			result := config.EffectiveSearchMode()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSearchMode_RequiresEmbedding(t *testing.T) {
	tests := []struct {
		mode     SearchMode
		requires bool
	}{
		{SearchModeTextOnly, false},
		{SearchModeSemanticOnly, true},
		{SearchModeHybrid, true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if tt.mode.RequiresEmbedding() != tt.requires {
				t.Errorf("expected %v, got %v", tt.requires, tt.mode.RequiresEmbedding())
			}
		})
	}
}

func TestRuntimeConfig_ThreadSafety(t *testing.T) {
	config := NewRuntimeConfig("postgres")

	// Run concurrent reads and writes
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			config.SetEmbeddingAvailable(true)
			config.SetLLMAvailable(true)
			config.SetEmbeddingAvailable(false)
			config.SetLLMAvailable(false)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = config.EmbeddingAvailable()
			_ = config.LLMAvailable()
			_ = config.CanDoSemanticSearch()
			_ = config.CanDoLLMAssisted()
			_ = config.EffectiveSearchMode()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}
