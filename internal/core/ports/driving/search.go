package driving

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// SearchService handles document search operations
type SearchService interface {
	// Search performs a search across all sources
	Search(ctx context.Context, query string, opts domain.SearchOptions) (*domain.SearchResult, error)

	// SearchBySource performs a search within a specific source
	SearchBySource(ctx context.Context, sourceID string, query string, opts domain.SearchOptions) (*domain.SearchResult, error)

	// Suggest provides search suggestions/autocomplete
	Suggest(ctx context.Context, prefix string, limit int) ([]domain.SearchSuggestion, error)
}
