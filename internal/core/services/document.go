package services

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driving"
)

// Ensure documentService implements DocumentService
var _ driving.DocumentService = (*documentService)(nil)

// documentService implements the DocumentService interface
type documentService struct {
	documentStore driven.DocumentStore
	chunkStore    driven.ChunkStore
}

// NewDocumentService creates a new DocumentService
func NewDocumentService(
	documentStore driven.DocumentStore,
	chunkStore driven.ChunkStore,
) driving.DocumentService {
	return &documentService{
		documentStore: documentStore,
		chunkStore:    chunkStore,
	}
}

// Get retrieves a document by ID
func (s *documentService) Get(ctx context.Context, id string) (*domain.Document, error) {
	return s.documentStore.Get(ctx, id)
}

// GetWithChunks retrieves a document with its chunks
func (s *documentService) GetWithChunks(ctx context.Context, id string) (*domain.DocumentWithChunks, error) {
	doc, err := s.documentStore.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	chunks, err := s.chunkStore.GetByDocument(ctx, id)
	if err != nil {
		return nil, err
	}

	return &domain.DocumentWithChunks{
		Document: doc,
		Chunks:   chunks,
	}, nil
}

// GetContent retrieves the full content of a document
func (s *documentService) GetContent(ctx context.Context, id string) (*domain.DocumentContent, error) {
	doc, err := s.documentStore.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get all chunks and reconstruct content
	chunks, err := s.chunkStore.GetByDocument(ctx, id)
	if err != nil {
		return nil, err
	}

	// Reconstruct body from chunks
	var body string
	for _, chunk := range chunks {
		body += chunk.Content
	}

	return &domain.DocumentContent{
		DocumentID: doc.ID,
		Title:      doc.Title,
		Body:       body,
		Metadata:   doc.Metadata,
	}, nil
}

// GetBySource retrieves all documents for a source
func (s *documentService) GetBySource(ctx context.Context, sourceID string, limit, offset int) ([]*domain.Document, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	return s.documentStore.GetBySource(ctx, sourceID, limit, offset)
}

// Count returns the total number of documents
func (s *documentService) Count(ctx context.Context) (int, error) {
	return s.documentStore.Count(ctx)
}

// CountBySource returns the document count for a source
func (s *documentService) CountBySource(ctx context.Context, sourceID string) (int, error) {
	return s.documentStore.CountBySource(ctx, sourceID)
}
