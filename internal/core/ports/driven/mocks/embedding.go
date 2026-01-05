package mocks

import (
	"context"
	"hash/fnv"
)

// MockEmbeddingService is a mock implementation of EmbeddingService for testing
type MockEmbeddingService struct {
	dimensions int
	model      string
	failNext   bool
}

// NewMockEmbeddingService creates a new MockEmbeddingService
func NewMockEmbeddingService() *MockEmbeddingService {
	return &MockEmbeddingService{
		dimensions: 384,
		model:      "mock-embedding-model",
	}
}

func (m *MockEmbeddingService) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.failNext {
		m.failNext = false
		return nil, context.DeadlineExceeded
	}

	result := make([][]float32, len(texts))
	for i, text := range texts {
		result[i] = m.generateEmbedding(text)
	}
	return result, nil
}

func (m *MockEmbeddingService) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if m.failNext {
		m.failNext = false
		return nil, context.DeadlineExceeded
	}
	return m.generateEmbedding(query), nil
}

func (m *MockEmbeddingService) Dimensions() int {
	return m.dimensions
}

func (m *MockEmbeddingService) Model() string {
	return m.model
}

func (m *MockEmbeddingService) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockEmbeddingService) Close() error {
	return nil
}

// generateEmbedding generates a deterministic embedding based on text hash
func (m *MockEmbeddingService) generateEmbedding(text string) []float32 {
	h := fnv.New32a()
	h.Write([]byte(text))
	seed := h.Sum32()

	embedding := make([]float32, m.dimensions)
	for i := range embedding {
		// Generate deterministic pseudo-random values
		seed = seed*1103515245 + 12345
		embedding[i] = float32(seed%1000) / 1000.0
	}
	return embedding
}

// Helper methods for testing

func (m *MockEmbeddingService) SetFailNext(fail bool) {
	m.failNext = fail
}

func (m *MockEmbeddingService) SetDimensions(dim int) {
	m.dimensions = dim
}
