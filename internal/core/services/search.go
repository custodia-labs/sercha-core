package services

import (
	"context"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driving"
	"github.com/custodia-labs/sercha-core/internal/runtime"
)

// Ensure searchService implements SearchService
var _ driving.SearchService = (*searchService)(nil)

// searchService implements the SearchService interface
type searchService struct {
	searchEngine  driven.SearchEngine
	documentStore driven.DocumentStore
	services      *runtime.Services // Dynamic AI services
}

// NewSearchService creates a new SearchService
// AI services (embedding, LLM) are accessed dynamically via runtime.Services
func NewSearchService(
	searchEngine driven.SearchEngine,
	documentStore driven.DocumentStore,
	services *runtime.Services,
) driving.SearchService {
	return &searchService{
		searchEngine:  searchEngine,
		documentStore: documentStore,
		services:      services,
	}
}

// Search performs a search across all sources
func (s *searchService) Search(ctx context.Context, query string, opts domain.SearchOptions) (*domain.SearchResult, error) {
	start := time.Now()

	// Apply defaults
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	// Determine effective search mode based on what's available NOW
	opts.Mode = s.effectiveMode(opts.Mode)

	// Get embedding service dynamically (may have been configured at runtime)
	embeddingService := s.services.EmbeddingService()

	// Generate query embedding for semantic search
	var queryEmbedding []float32
	if opts.Mode.RequiresEmbedding() {
		if embeddingService != nil {
			embedding, err := embeddingService.EmbedQuery(ctx, query)
			if err != nil {
				// Fall back to text-only if embedding fails
				opts.Mode = domain.SearchModeTextOnly
			} else {
				queryEmbedding = embedding
			}
		} else {
			// Embedding required but not available - degrade
			opts.Mode = domain.SearchModeTextOnly
		}
	}

	// Perform search
	rankedChunks, totalCount, err := s.searchEngine.Search(ctx, query, queryEmbedding, opts)
	if err != nil {
		return nil, err
	}

	// Enrich with document data
	for _, rc := range rankedChunks {
		if rc.Document == nil && rc.Chunk != nil {
			doc, _ := s.documentStore.Get(ctx, rc.Chunk.DocumentID)
			rc.Document = doc
		}
	}

	return &domain.SearchResult{
		Query:      query,
		Mode:       opts.Mode,
		Results:    rankedChunks,
		TotalCount: totalCount,
		Took:       time.Since(start),
	}, nil
}

// SearchBySource performs a search within a specific source
func (s *searchService) SearchBySource(ctx context.Context, sourceID string, query string, opts domain.SearchOptions) (*domain.SearchResult, error) {
	// Add source filter
	opts.SourceIDs = []string{sourceID}
	return s.Search(ctx, query, opts)
}

// Suggest provides search suggestions/autocomplete
func (s *searchService) Suggest(_ context.Context, _ string, _ int) ([]domain.SearchSuggestion, error) {
	// TODO: Implement autocomplete/suggestions
	// This could use:
	// 1. Recent search history
	// 2. Popular terms from indexed content
	// 3. Prefix matching on document titles
	return []domain.SearchSuggestion{}, nil
}

// effectiveMode determines the best search mode based on requested mode and available services
func (s *searchService) effectiveMode(requested domain.SearchMode) domain.SearchMode {
	// Default to hybrid if not specified
	if requested == "" {
		requested = s.services.Config().EffectiveSearchMode()
	}

	config := s.services.Config()

	// Validate requested mode is possible with current capabilities
	switch requested {
	case domain.SearchModeSemanticOnly:
		if !config.CanDoSemanticSearch() {
			return domain.SearchModeTextOnly
		}
	case domain.SearchModeHybrid:
		if !config.CanDoSemanticSearch() {
			return domain.SearchModeTextOnly
		}
	}

	return requested
}
