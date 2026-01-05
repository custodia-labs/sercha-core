package driven

import (
	"context"
)

// EmbeddingService generates text embeddings
type EmbeddingService interface {
	// Embed generates embeddings for multiple texts
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedQuery generates an embedding for a search query
	// May use different model/parameters optimized for queries
	EmbedQuery(ctx context.Context, query string) ([]float32, error)

	// Dimensions returns the embedding dimension size
	Dimensions() int

	// Model returns the model name being used
	Model() string

	// HealthCheck verifies the embedding service is available
	HealthCheck(ctx context.Context) error

	// Close releases resources held by the embedding service
	Close() error
}
