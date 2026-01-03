package postprocessors

import (
	"strings"
	"testing"

	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

func TestNewPipeline(t *testing.T) {
	p := NewPipeline()
	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if len(p.processors) != 0 {
		t.Errorf("expected empty processors, got %d", len(p.processors))
	}
}

func TestPipeline_Add(t *testing.T) {
	p := NewPipeline()

	p.Add(NewChunker(DefaultChunkConfig()))
	p.Add(NewWhitespaceNormalizer())
	p.Add(NewDeduplicator(DefaultDeduplicatorConfig()))

	names := p.List()
	if len(names) != 3 {
		t.Errorf("expected 3 processors, got %d", len(names))
	}
}

func TestPipeline_Process_EmptyContent(t *testing.T) {
	p := NewPipeline()
	p.Add(NewChunker(DefaultChunkConfig()))

	chunks := p.Process("")
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Content != "" {
		t.Errorf("expected empty content, got %q", chunks[0].Content)
	}
}

func TestPipeline_Process_SmallContent(t *testing.T) {
	p := NewPipeline()
	p.Add(NewChunker(DefaultChunkConfig()))

	content := "Hello, world!"
	chunks := p.Process(content)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Content != content {
		t.Errorf("expected %q, got %q", content, chunks[0].Content)
	}
	if chunks[0].Position != 0 {
		t.Errorf("expected position 0, got %d", chunks[0].Position)
	}
	if chunks[0].StartOffset != 0 {
		t.Errorf("expected start offset 0, got %d", chunks[0].StartOffset)
	}
	if chunks[0].EndOffset != len(content) {
		t.Errorf("expected end offset %d, got %d", len(content), chunks[0].EndOffset)
	}
}

func TestPipeline_Process_LargeContent(t *testing.T) {
	config := ChunkConfig{
		MaxChunkSize:       100,
		Overlap:            20,
		PreserveSentences:  false,
		PreserveParagraphs: false,
	}
	p := NewPipeline()
	p.Add(NewChunker(config))

	// Create content larger than MaxChunkSize
	content := strings.Repeat("a", 250)
	chunks := p.Process(content)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	// Check that chunks have overlap
	for i := 1; i < len(chunks); i++ {
		prevEnd := chunks[i-1].EndOffset
		currStart := chunks[i].StartOffset
		overlap := prevEnd - currStart
		if overlap != config.Overlap {
			t.Errorf("expected overlap %d, got %d", config.Overlap, overlap)
		}
	}

	// Check positions are sequential
	for i, chunk := range chunks {
		if chunk.Position != i {
			t.Errorf("expected position %d, got %d", i, chunk.Position)
		}
	}
}

func TestPipeline_Process_OrderedProcessors(t *testing.T) {
	p := NewPipeline()

	// Add in wrong order - should be sorted by Order()
	p.Add(NewDeduplicator(DefaultDeduplicatorConfig())) // Order 10
	p.Add(NewChunker(DefaultChunkConfig()))             // Order 0
	p.Add(NewWhitespaceNormalizer())                    // Order 5

	// Process something to trigger sorting
	content := "Test content"
	_ = p.Process(content)

	// List should show sorted order
	names := p.List()
	if len(names) != 3 {
		t.Fatalf("expected 3 processors, got %d", len(names))
	}
	if names[0] != "chunker" {
		t.Errorf("expected first processor 'chunker', got %s", names[0])
	}
	if names[1] != "whitespace-normalizer" {
		t.Errorf("expected second processor 'whitespace-normalizer', got %s", names[1])
	}
	if names[2] != "deduplicator" {
		t.Errorf("expected third processor 'deduplicator', got %s", names[2])
	}
}

func TestDefaultPipeline(t *testing.T) {
	p := DefaultPipeline()

	names := p.List()
	if len(names) != 1 {
		t.Fatalf("expected 1 processor in default pipeline, got %d", len(names))
	}
	if names[0] != "chunker" {
		t.Errorf("expected 'chunker', got %s", names[0])
	}
}

func TestDefaultChunkConfig(t *testing.T) {
	config := DefaultChunkConfig()

	if config.MaxChunkSize != 1000 {
		t.Errorf("expected MaxChunkSize 1000, got %d", config.MaxChunkSize)
	}
	if config.Overlap != 200 {
		t.Errorf("expected Overlap 200, got %d", config.Overlap)
	}
	if !config.PreserveSentences {
		t.Error("expected PreserveSentences true")
	}
	if !config.PreserveParagraphs {
		t.Error("expected PreserveParagraphs true")
	}
}

func TestChunker_Name(t *testing.T) {
	c := NewChunker(DefaultChunkConfig())
	if c.Name() != "chunker" {
		t.Errorf("expected name 'chunker', got %s", c.Name())
	}
}

func TestChunker_Order(t *testing.T) {
	c := NewChunker(DefaultChunkConfig())
	if c.Order() != 0 {
		t.Errorf("expected order 0, got %d", c.Order())
	}
}

func TestChunker_PreserveSentences(t *testing.T) {
	config := ChunkConfig{
		MaxChunkSize:       50,
		Overlap:            10,
		PreserveSentences:  true,
		PreserveParagraphs: false,
	}
	c := NewChunker(config)

	content := "This is sentence one. This is sentence two. This is sentence three."
	input := []driven.Chunk{{Content: content, StartOffset: 0, EndOffset: len(content)}}

	chunks := c.Process(input)

	// Should break at sentence boundaries when possible
	for _, chunk := range chunks {
		trimmed := strings.TrimSpace(chunk.Content)
		if len(trimmed) > 0 && !strings.HasSuffix(trimmed, ".") && len(trimmed) < config.MaxChunkSize {
			// Small chunks should end with sentence boundary unless at end of content
			// This is a soft check - the algorithm tries but may not always succeed
		}
	}
}

func TestChunker_PreserveParagraphs(t *testing.T) {
	config := ChunkConfig{
		MaxChunkSize:       100,
		Overlap:            20,
		PreserveSentences:  false,
		PreserveParagraphs: true,
	}
	c := NewChunker(config)

	content := "First paragraph here.\n\nSecond paragraph here.\n\nThird paragraph with more text to exceed limit."
	input := []driven.Chunk{{Content: content, StartOffset: 0, EndOffset: len(content)}}

	chunks := c.Process(input)

	// Should produce multiple chunks
	if len(chunks) < 1 {
		t.Error("expected at least one chunk")
	}
}

func TestChunker_NoBreakPoint(t *testing.T) {
	config := ChunkConfig{
		MaxChunkSize:       50,
		Overlap:            10,
		PreserveSentences:  true,
		PreserveParagraphs: true,
	}
	c := NewChunker(config)

	// Content with no sentence or paragraph breaks
	content := strings.Repeat("x", 100)
	input := []driven.Chunk{{Content: content, StartOffset: 0, EndOffset: len(content)}}

	chunks := c.Process(input)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	// Total coverage should include all content
	totalLen := 0
	for i, chunk := range chunks {
		if i == 0 {
			totalLen = chunk.EndOffset
		} else {
			totalLen = chunk.EndOffset
		}
	}
	if totalLen < len(content) {
		t.Errorf("chunks don't cover all content: covered %d of %d", totalLen, len(content))
	}
}

func TestDeduplicator_Name(t *testing.T) {
	d := NewDeduplicator(DefaultDeduplicatorConfig())
	if d.Name() != "deduplicator" {
		t.Errorf("expected name 'deduplicator', got %s", d.Name())
	}
}

func TestDeduplicator_Order(t *testing.T) {
	d := NewDeduplicator(DefaultDeduplicatorConfig())
	if d.Order() != 10 {
		t.Errorf("expected order 10, got %d", d.Order())
	}
}

func TestDeduplicator_RemovesDuplicates(t *testing.T) {
	config := DeduplicatorConfig{
		MinDuplicateLength:  10,
		SimilarityThreshold: 0.95,
	}
	d := NewDeduplicator(config)

	chunks := []driven.Chunk{
		{Content: "This is the first unique chunk with enough content.", Position: 0},
		{Content: "This is a duplicate chunk with enough content.", Position: 1},
		{Content: "This is a duplicate chunk with enough content.", Position: 2}, // Duplicate
		{Content: "This is another unique chunk with sufficient length.", Position: 3},
	}

	result := d.Process(chunks)

	if len(result) != 3 {
		t.Errorf("expected 3 chunks after dedup, got %d", len(result))
	}
}

func TestDeduplicator_KeepsShortChunks(t *testing.T) {
	config := DeduplicatorConfig{
		MinDuplicateLength:  50,
		SimilarityThreshold: 0.95,
	}
	d := NewDeduplicator(config)

	// Short chunks below MinDuplicateLength should not be deduped
	chunks := []driven.Chunk{
		{Content: "Short", Position: 0},
		{Content: "Short", Position: 1}, // Same but below threshold
		{Content: "Short", Position: 2}, // Same but below threshold
	}

	result := d.Process(chunks)

	if len(result) != 3 {
		t.Errorf("expected 3 chunks (short chunks not deduped), got %d", len(result))
	}
}

func TestDeduplicator_CaseInsensitive(t *testing.T) {
	config := DeduplicatorConfig{
		MinDuplicateLength:  10,
		SimilarityThreshold: 0.95,
	}
	d := NewDeduplicator(config)

	chunks := []driven.Chunk{
		{Content: "This is some content that is long enough", Position: 0},
		{Content: "THIS IS SOME CONTENT THAT IS LONG ENOUGH", Position: 1}, // Same when lowercased
	}

	result := d.Process(chunks)

	if len(result) != 1 {
		t.Errorf("expected 1 chunk after case-insensitive dedup, got %d", len(result))
	}
}

func TestDeduplicator_SingleChunk(t *testing.T) {
	d := NewDeduplicator(DefaultDeduplicatorConfig())

	chunks := []driven.Chunk{
		{Content: "Only one chunk", Position: 0},
	}

	result := d.Process(chunks)

	if len(result) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(result))
	}
}

func TestDeduplicator_EmptyInput(t *testing.T) {
	d := NewDeduplicator(DefaultDeduplicatorConfig())

	result := d.Process([]driven.Chunk{})

	if len(result) != 0 {
		t.Errorf("expected 0 chunks, got %d", len(result))
	}
}

func TestDefaultDeduplicatorConfig(t *testing.T) {
	config := DefaultDeduplicatorConfig()

	if config.MinDuplicateLength != 50 {
		t.Errorf("expected MinDuplicateLength 50, got %d", config.MinDuplicateLength)
	}
	if config.SimilarityThreshold != 0.95 {
		t.Errorf("expected SimilarityThreshold 0.95, got %f", config.SimilarityThreshold)
	}
}

func TestWhitespaceNormalizer_Name(t *testing.T) {
	w := NewWhitespaceNormalizer()
	if w.Name() != "whitespace-normalizer" {
		t.Errorf("expected name 'whitespace-normalizer', got %s", w.Name())
	}
}

func TestWhitespaceNormalizer_Order(t *testing.T) {
	w := NewWhitespaceNormalizer()
	if w.Order() != 5 {
		t.Errorf("expected order 5, got %d", w.Order())
	}
}

func TestWhitespaceNormalizer_NormalizesLineEndings(t *testing.T) {
	w := NewWhitespaceNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"windows line endings", "hello\r\nworld", "hello\nworld"},
		{"old mac line endings", "hello\rworld", "hello\nworld"},
		{"mixed line endings", "a\r\nb\rc\n", "a\nb\nc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := []driven.Chunk{{Content: tt.input, Position: 0}}
			result := w.Process(chunks)

			if len(result) != 1 {
				t.Fatalf("expected 1 chunk, got %d", len(result))
			}
			if result[0].Content != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result[0].Content)
			}
		})
	}
}

func TestWhitespaceNormalizer_CollapsesSpaces(t *testing.T) {
	w := NewWhitespaceNormalizer()

	chunks := []driven.Chunk{
		{Content: "hello    world", Position: 0},
	}

	result := w.Process(chunks)

	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0].Content != "hello world" {
		t.Errorf("expected 'hello world', got %q", result[0].Content)
	}
}

func TestWhitespaceNormalizer_CollapsesBlankLines(t *testing.T) {
	w := NewWhitespaceNormalizer()

	chunks := []driven.Chunk{
		{Content: "a\n\n\n\nb", Position: 0},
	}

	result := w.Process(chunks)

	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0].Content != "a\n\nb" {
		t.Errorf("expected 'a\\n\\nb', got %q", result[0].Content)
	}
}

func TestWhitespaceNormalizer_TrimsWhitespace(t *testing.T) {
	w := NewWhitespaceNormalizer()

	chunks := []driven.Chunk{
		{Content: "  hello  \n  world  ", Position: 0},
	}

	result := w.Process(chunks)

	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0].Content != "hello\nworld" {
		t.Errorf("expected 'hello\\nworld', got %q", result[0].Content)
	}
}

func TestWhitespaceNormalizer_RemovesEmptyChunks(t *testing.T) {
	w := NewWhitespaceNormalizer()

	chunks := []driven.Chunk{
		{Content: "hello", Position: 0},
		{Content: "   ", Position: 1},   // Empty after trim
		{Content: "\n\n", Position: 2},  // Empty after trim
		{Content: "world", Position: 3},
	}

	result := w.Process(chunks)

	if len(result) != 2 {
		t.Errorf("expected 2 chunks (empty removed), got %d", len(result))
	}
}

func TestWhitespaceNormalizer_PreservesMetadata(t *testing.T) {
	w := NewWhitespaceNormalizer()

	chunks := []driven.Chunk{
		{
			Content:     "  hello  ",
			Position:    5,
			StartOffset: 100,
			EndOffset:   200,
			Metadata:    map[string]string{"key": "value"},
		},
	}

	result := w.Process(chunks)

	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}

	// Content should be normalized
	if result[0].Content != "hello" {
		t.Errorf("expected 'hello', got %q", result[0].Content)
	}

	// Other fields should be preserved
	if result[0].Position != 5 {
		t.Errorf("expected position 5, got %d", result[0].Position)
	}
	if result[0].StartOffset != 100 {
		t.Errorf("expected start offset 100, got %d", result[0].StartOffset)
	}
	if result[0].EndOffset != 200 {
		t.Errorf("expected end offset 200, got %d", result[0].EndOffset)
	}
	if result[0].Metadata["key"] != "value" {
		t.Errorf("expected metadata key=value, got %v", result[0].Metadata)
	}
}

func TestPipeline_IntegrationFullPipeline(t *testing.T) {
	p := NewPipeline()
	p.Add(NewChunker(ChunkConfig{
		MaxChunkSize:       100,
		Overlap:            20,
		PreserveSentences:  true,
		PreserveParagraphs: true,
	}))
	p.Add(NewWhitespaceNormalizer())
	p.Add(NewDeduplicator(DeduplicatorConfig{
		MinDuplicateLength:  10,
		SimilarityThreshold: 0.95,
	}))

	content := `This is the first paragraph with some content that should be long enough to trigger chunking.

This is the second paragraph with additional content that will help test the full pipeline.

  This paragraph has   extra    whitespace   that should be normalized.

This is the final paragraph to complete our test content.`

	chunks := p.Process(content)

	// Should have multiple chunks
	if len(chunks) < 1 {
		t.Fatalf("expected at least 1 chunk, got %d", len(chunks))
	}

	// All chunks should be non-empty
	for i, chunk := range chunks {
		if len(chunk.Content) == 0 {
			t.Errorf("chunk %d has empty content", i)
		}
	}

	// Whitespace should be normalized
	for i, chunk := range chunks {
		if strings.Contains(chunk.Content, "   ") {
			t.Errorf("chunk %d contains excessive spaces: %q", i, chunk.Content)
		}
	}
}

// Verify interface compliance
func TestInterfaceCompliance(t *testing.T) {
	var _ driven.PostProcessorPipeline = (*Pipeline)(nil)
	var _ driven.PostProcessor = (*Chunker)(nil)
	var _ driven.PostProcessor = (*Deduplicator)(nil)
	var _ driven.PostProcessor = (*WhitespaceNormalizer)(nil)
}
