package driven

import (
	"context"
)

// LLMService provides large language model capabilities for search enhancement
type LLMService interface {
	// ExpandQuery takes a search query and returns expanded/related terms
	// Useful for improving search recall
	ExpandQuery(ctx context.Context, query string) ([]string, error)

	// Summarise generates a summary of the given content
	// maxLen is a hint for maximum length (model may not respect exactly)
	Summarise(ctx context.Context, content string, maxLen int) (string, error)

	// RewriteQuery rewrites the query for better search results
	// Returns the rewritten query
	RewriteQuery(ctx context.Context, query string) (string, error)

	// Model returns the model name being used
	Model() string

	// Ping verifies the LLM service is available
	Ping(ctx context.Context) error

	// Close releases resources held by the LLM service
	Close() error
}
