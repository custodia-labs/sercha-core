package mocks

import (
	"context"
	"strings"
	"sync"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// MockSearchEngine is a mock implementation of SearchEngine for testing
type MockSearchEngine struct {
	mu     sync.RWMutex
	chunks map[string]*domain.Chunk
	byDoc  map[string][]*domain.Chunk
	bySource map[string][]*domain.Chunk
}

// NewMockSearchEngine creates a new MockSearchEngine
func NewMockSearchEngine() *MockSearchEngine {
	return &MockSearchEngine{
		chunks:   make(map[string]*domain.Chunk),
		byDoc:    make(map[string][]*domain.Chunk),
		bySource: make(map[string][]*domain.Chunk),
	}
}

func (m *MockSearchEngine) Index(ctx context.Context, chunks []*domain.Chunk) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, chunk := range chunks {
		m.chunks[chunk.ID] = chunk
		m.byDoc[chunk.DocumentID] = append(m.byDoc[chunk.DocumentID], chunk)
		m.bySource[chunk.SourceID] = append(m.bySource[chunk.SourceID], chunk)
	}
	return nil
}

func (m *MockSearchEngine) Search(ctx context.Context, query string, queryEmbedding []float32, opts domain.SearchOptions) ([]*domain.RankedChunk, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*domain.RankedChunk
	queryLower := strings.ToLower(query)

	for _, chunk := range m.chunks {
		// Filter by source if specified
		if len(opts.SourceIDs) > 0 {
			found := false
			for _, sourceID := range opts.SourceIDs {
				if chunk.SourceID == sourceID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Simple text matching for mock
		if strings.Contains(strings.ToLower(chunk.Content), queryLower) {
			results = append(results, &domain.RankedChunk{
				Chunk: chunk,
				Score: 1.0,
				Highlights: []string{chunk.Content},
			})
		}
	}

	// Apply pagination
	total := len(results)
	if opts.Offset >= len(results) {
		return []*domain.RankedChunk{}, total, nil
	}
	end := opts.Offset + opts.Limit
	if end > len(results) {
		end = len(results)
	}
	if opts.Limit <= 0 {
		end = len(results)
	}

	return results[opts.Offset:end], total, nil
}

func (m *MockSearchEngine) Delete(ctx context.Context, chunkIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range chunkIDs {
		delete(m.chunks, id)
	}
	return nil
}

func (m *MockSearchEngine) DeleteByDocument(ctx context.Context, documentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	chunks := m.byDoc[documentID]
	for _, chunk := range chunks {
		delete(m.chunks, chunk.ID)
	}
	delete(m.byDoc, documentID)
	return nil
}

func (m *MockSearchEngine) DeleteBySource(ctx context.Context, sourceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	chunks := m.bySource[sourceID]
	for _, chunk := range chunks {
		delete(m.chunks, chunk.ID)
		delete(m.byDoc, chunk.DocumentID)
	}
	delete(m.bySource, sourceID)
	return nil
}

func (m *MockSearchEngine) HealthCheck(ctx context.Context) error {
	return nil
}

// Helper methods for testing

func (m *MockSearchEngine) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chunks = make(map[string]*domain.Chunk)
	m.byDoc = make(map[string][]*domain.Chunk)
	m.bySource = make(map[string][]*domain.Chunk)
}

func (m *MockSearchEngine) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.chunks)
}
