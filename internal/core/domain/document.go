package domain

import "time"

// Document represents an indexed document from a source
type Document struct {
	ID         string            `json:"id"`
	SourceID   string            `json:"source_id"`
	ExternalID string            `json:"external_id"` // ID from the source system
	Path       string            `json:"path"`        // Path or URL in source
	Title      string            `json:"title"`
	MimeType   string            `json:"mime_type"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	IndexedAt  time.Time         `json:"indexed_at"`
}

// Chunk represents a searchable chunk of a document
type Chunk struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	SourceID   string    `json:"source_id"`
	Content    string    `json:"content"`
	Embedding  []float32 `json:"embedding,omitempty"`
	Position   int       `json:"position"` // Chunk position within document
	StartChar  int       `json:"start_char"`
	EndChar    int       `json:"end_char"`
	CreatedAt  time.Time `json:"created_at"`
}

// DocumentContent holds the full content of a document
type DocumentContent struct {
	DocumentID string            `json:"document_id"`
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Metadata   map[string]string `json:"metadata"`
}

// DocumentWithChunks combines a document with its chunks
type DocumentWithChunks struct {
	Document *Document `json:"document"`
	Chunks   []*Chunk  `json:"chunks"`
}
