package domain

import (
	"testing"
	"time"
)

func TestDocument(t *testing.T) {
	now := time.Now()
	doc := &Document{
		ID:         "doc-123",
		SourceID:   "source-456",
		ExternalID: "ext-789",
		Path:       "/path/to/file.md",
		Title:      "Test Document",
		MimeType:   "text/markdown",
		Metadata: map[string]string{
			"author": "test-user",
		},
		CreatedAt: now,
		UpdatedAt: now,
		IndexedAt: now,
	}

	if doc.ID != "doc-123" {
		t.Errorf("expected ID doc-123, got %s", doc.ID)
	}
	if doc.SourceID != "source-456" {
		t.Errorf("expected SourceID source-456, got %s", doc.SourceID)
	}
	if doc.ExternalID != "ext-789" {
		t.Errorf("expected ExternalID ext-789, got %s", doc.ExternalID)
	}
	if doc.Path != "/path/to/file.md" {
		t.Errorf("expected Path /path/to/file.md, got %s", doc.Path)
	}
	if doc.Title != "Test Document" {
		t.Errorf("expected Title 'Test Document', got %s", doc.Title)
	}
	if doc.MimeType != "text/markdown" {
		t.Errorf("expected MimeType text/markdown, got %s", doc.MimeType)
	}
	if doc.Metadata["author"] != "test-user" {
		t.Errorf("expected author test-user, got %s", doc.Metadata["author"])
	}
}

func TestChunk(t *testing.T) {
	now := time.Now()
	embedding := []float32{0.1, 0.2, 0.3}

	chunk := &Chunk{
		ID:         "chunk-123",
		DocumentID: "doc-456",
		SourceID:   "source-789",
		Content:    "This is the chunk content.",
		Embedding:  embedding,
		Position:   0,
		StartChar:  0,
		EndChar:    26,
		CreatedAt:  now,
	}

	if chunk.ID != "chunk-123" {
		t.Errorf("expected ID chunk-123, got %s", chunk.ID)
	}
	if chunk.DocumentID != "doc-456" {
		t.Errorf("expected DocumentID doc-456, got %s", chunk.DocumentID)
	}
	if chunk.SourceID != "source-789" {
		t.Errorf("expected SourceID source-789, got %s", chunk.SourceID)
	}
	if chunk.Content != "This is the chunk content." {
		t.Errorf("expected Content 'This is the chunk content.', got %s", chunk.Content)
	}
	if len(chunk.Embedding) != 3 {
		t.Errorf("expected 3 embedding dimensions, got %d", len(chunk.Embedding))
	}
	if chunk.Position != 0 {
		t.Errorf("expected Position 0, got %d", chunk.Position)
	}
	if chunk.StartChar != 0 {
		t.Errorf("expected StartChar 0, got %d", chunk.StartChar)
	}
	if chunk.EndChar != 26 {
		t.Errorf("expected EndChar 26, got %d", chunk.EndChar)
	}
}

func TestDocumentContent(t *testing.T) {
	content := &DocumentContent{
		DocumentID: "doc-123",
		Title:      "Test Document",
		Body:       "This is the document body with more content.",
		Metadata: map[string]string{
			"source": "test",
		},
	}

	if content.DocumentID != "doc-123" {
		t.Errorf("expected DocumentID doc-123, got %s", content.DocumentID)
	}
	if content.Title != "Test Document" {
		t.Errorf("expected Title 'Test Document', got %s", content.Title)
	}
	if content.Body != "This is the document body with more content." {
		t.Errorf("unexpected body content")
	}
	if content.Metadata["source"] != "test" {
		t.Errorf("expected source test, got %s", content.Metadata["source"])
	}
}

func TestDocumentWithChunks(t *testing.T) {
	doc := &Document{
		ID:    "doc-123",
		Title: "Test Document",
	}
	chunks := []*Chunk{
		{ID: "chunk-1", DocumentID: "doc-123", Content: "First chunk"},
		{ID: "chunk-2", DocumentID: "doc-123", Content: "Second chunk"},
	}

	docWithChunks := &DocumentWithChunks{
		Document: doc,
		Chunks:   chunks,
	}

	if docWithChunks.Document.ID != "doc-123" {
		t.Errorf("expected Document ID doc-123, got %s", docWithChunks.Document.ID)
	}
	if len(docWithChunks.Chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(docWithChunks.Chunks))
	}
	if docWithChunks.Chunks[0].Content != "First chunk" {
		t.Errorf("expected first chunk content 'First chunk', got %s", docWithChunks.Chunks[0].Content)
	}
	if docWithChunks.Chunks[1].Content != "Second chunk" {
		t.Errorf("expected second chunk content 'Second chunk', got %s", docWithChunks.Chunks[1].Content)
	}
}
