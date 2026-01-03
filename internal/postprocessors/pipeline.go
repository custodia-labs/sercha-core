package postprocessors

import (
	"sort"
	"strings"
	"sync"

	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Verify interface compliance
var _ driven.PostProcessorPipeline = (*Pipeline)(nil)

// Pipeline implements PostProcessorPipeline.
// It chains multiple post-processors in order, starting with a Chunker.
type Pipeline struct {
	mu         sync.RWMutex
	processors []driven.PostProcessor
	sorted     bool
}

// NewPipeline creates a new post-processor pipeline.
func NewPipeline() *Pipeline {
	return &Pipeline{
		processors: make([]driven.PostProcessor, 0),
	}
}

// Add adds a processor to the pipeline.
// Processors are sorted by Order() before processing.
func (p *Pipeline) Add(processor driven.PostProcessor) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.processors = append(p.processors, processor)
	p.sorted = false
}

// Process applies all processors in order.
// Input is the raw document content.
// Output is the processed chunks ready for embedding/indexing.
func (p *Pipeline) Process(content string) []driven.Chunk {
	p.mu.Lock()
	if !p.sorted {
		sort.Slice(p.processors, func(i, j int) bool {
			return p.processors[i].Order() < p.processors[j].Order()
		})
		p.sorted = true
	}
	p.mu.Unlock()

	p.mu.RLock()
	processors := make([]driven.PostProcessor, len(p.processors))
	copy(processors, p.processors)
	p.mu.RUnlock()

	// Start with a single chunk containing all content
	chunks := []driven.Chunk{
		{
			Content:     content,
			Position:    0,
			StartOffset: 0,
			EndOffset:   len(content),
		},
	}

	// Apply each processor in order
	for _, proc := range processors {
		chunks = proc.Process(chunks)
	}

	return chunks
}

// List returns processor names in order.
func (p *Pipeline) List() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	names := make([]string, len(p.processors))
	for i, proc := range p.processors {
		names[i] = proc.Name()
	}
	return names
}

// DefaultPipeline creates a pipeline with the default processors.
func DefaultPipeline() *Pipeline {
	p := NewPipeline()
	p.Add(NewChunker(DefaultChunkConfig()))
	return p
}

// ChunkConfig configures the chunker behavior.
type ChunkConfig struct {
	// MaxChunkSize is the maximum characters per chunk
	MaxChunkSize int

	// Overlap is the character overlap between chunks
	Overlap int

	// PreserveSentences tries to break at sentence boundaries
	PreserveSentences bool

	// PreserveParagraphs tries to break at paragraph boundaries
	PreserveParagraphs bool
}

// DefaultChunkConfig returns sensible defaults.
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		MaxChunkSize:       1000,
		Overlap:            200,
		PreserveSentences:  true,
		PreserveParagraphs: true,
	}
}

// Chunker splits content into overlapping chunks.
// This is typically the first processor in the pipeline (Order = 0).
type Chunker struct {
	config ChunkConfig
}

// Verify interface compliance
var _ driven.PostProcessor = (*Chunker)(nil)

// NewChunker creates a new chunker with the given config.
func NewChunker(config ChunkConfig) *Chunker {
	return &Chunker{config: config}
}

// Process splits content into chunks.
func (c *Chunker) Process(chunks []driven.Chunk) []driven.Chunk {
	var result []driven.Chunk
	position := 0

	for _, chunk := range chunks {
		newChunks := c.splitContent(chunk.Content, chunk.StartOffset, &position)
		result = append(result, newChunks...)
	}

	return result
}

// Name returns the processor name.
func (c *Chunker) Name() string {
	return "chunker"
}

// Order returns 0 - chunker should be first.
func (c *Chunker) Order() int {
	return 0
}

// splitContent splits content into overlapping chunks.
func (c *Chunker) splitContent(content string, baseOffset int, position *int) []driven.Chunk {
	if len(content) <= c.config.MaxChunkSize {
		chunk := driven.Chunk{
			Content:     content,
			Position:    *position,
			StartOffset: baseOffset,
			EndOffset:   baseOffset + len(content),
		}
		*position++
		return []driven.Chunk{chunk}
	}

	var chunks []driven.Chunk
	start := 0

	for start < len(content) {
		end := start + c.config.MaxChunkSize
		if end > len(content) {
			end = len(content)
		}

		// Try to find a good break point
		if end < len(content) && c.config.PreserveSentences {
			breakPoint := c.findBreakPoint(content, start, end)
			if breakPoint > start {
				end = breakPoint
			}
		}

		chunkContent := content[start:end]

		chunk := driven.Chunk{
			Content:     chunkContent,
			Position:    *position,
			StartOffset: baseOffset + start,
			EndOffset:   baseOffset + end,
		}
		chunks = append(chunks, chunk)
		*position++

		// If we've reached the end, stop
		if end >= len(content) {
			break
		}

		// Move start with overlap, ensuring we always advance
		nextStart := end - c.config.Overlap
		if nextStart <= start {
			// Ensure we always make progress
			nextStart = start + 1
		}
		start = nextStart
	}

	return chunks
}

// findBreakPoint finds a good break point for chunking.
func (c *Chunker) findBreakPoint(content string, start, maxEnd int) int {
	searchStart := maxEnd - 100
	if searchStart < start {
		searchStart = start
	}

	searchContent := content[searchStart:maxEnd]

	// Try to break at paragraph boundary (double newline)
	if c.config.PreserveParagraphs {
		if idx := strings.LastIndex(searchContent, "\n\n"); idx != -1 {
			return searchStart + idx + 2 // After the double newline
		}
	}

	// Try to break at sentence boundary
	if c.config.PreserveSentences {
		// Look for sentence endings
		sentenceEnders := []string{". ", "! ", "? ", ".\n", "!\n", "?\n"}
		bestIdx := -1

		for _, ender := range sentenceEnders {
			if idx := strings.LastIndex(searchContent, ender); idx != -1 {
				endPos := idx + len(ender)
				if endPos > bestIdx {
					bestIdx = endPos
				}
			}
		}

		if bestIdx > 0 {
			return searchStart + bestIdx
		}
	}

	// Try to break at word boundary
	if idx := strings.LastIndex(searchContent, " "); idx != -1 {
		return searchStart + idx + 1
	}

	// No good break point found, use maxEnd
	return maxEnd
}

// DeduplicatorConfig configures the deduplicator.
type DeduplicatorConfig struct {
	// MinDuplicateLength is the minimum chunk length to check for duplicates
	MinDuplicateLength int

	// SimilarityThreshold is the minimum similarity (0-1) to consider chunks duplicate
	SimilarityThreshold float64
}

// DefaultDeduplicatorConfig returns sensible defaults.
func DefaultDeduplicatorConfig() DeduplicatorConfig {
	return DeduplicatorConfig{
		MinDuplicateLength:  50,
		SimilarityThreshold: 0.95,
	}
}

// Deduplicator removes duplicate or near-duplicate chunks.
type Deduplicator struct {
	config DeduplicatorConfig
}

// Verify interface compliance
var _ driven.PostProcessor = (*Deduplicator)(nil)

// NewDeduplicator creates a new deduplicator with the given config.
func NewDeduplicator(config DeduplicatorConfig) *Deduplicator {
	return &Deduplicator{config: config}
}

// Process removes duplicate chunks.
func (d *Deduplicator) Process(chunks []driven.Chunk) []driven.Chunk {
	if len(chunks) <= 1 {
		return chunks
	}

	seen := make(map[string]bool)
	var result []driven.Chunk

	for _, chunk := range chunks {
		if len(chunk.Content) < d.config.MinDuplicateLength {
			result = append(result, chunk)
			continue
		}

		// Normalize for comparison
		normalized := strings.TrimSpace(strings.ToLower(chunk.Content))

		if !seen[normalized] {
			seen[normalized] = true
			result = append(result, chunk)
		}
	}

	return result
}

// Name returns the processor name.
func (d *Deduplicator) Name() string {
	return "deduplicator"
}

// Order returns 10 - deduplicator runs after chunker.
func (d *Deduplicator) Order() int {
	return 10
}

// WhitespaceNormalizer normalizes whitespace in chunks.
type WhitespaceNormalizer struct{}

// Verify interface compliance
var _ driven.PostProcessor = (*WhitespaceNormalizer)(nil)

// NewWhitespaceNormalizer creates a new whitespace normalizer.
func NewWhitespaceNormalizer() *WhitespaceNormalizer {
	return &WhitespaceNormalizer{}
}

// Process normalizes whitespace in chunks.
func (w *WhitespaceNormalizer) Process(chunks []driven.Chunk) []driven.Chunk {
	result := make([]driven.Chunk, 0, len(chunks))

	for _, chunk := range chunks {
		content := chunk.Content

		// Normalize line endings
		content = strings.ReplaceAll(content, "\r\n", "\n")
		content = strings.ReplaceAll(content, "\r", "\n")

		// Collapse multiple spaces (but preserve newlines)
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			for strings.Contains(line, "  ") {
				line = strings.ReplaceAll(line, "  ", " ")
			}
			lines[i] = strings.TrimSpace(line)
		}
		content = strings.Join(lines, "\n")

		// Remove excessive blank lines
		for strings.Contains(content, "\n\n\n") {
			content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
		}

		content = strings.TrimSpace(content)

		if len(content) > 0 {
			newChunk := chunk
			newChunk.Content = content
			result = append(result, newChunk)
		}
	}

	return result
}

// Name returns the processor name.
func (w *WhitespaceNormalizer) Name() string {
	return "whitespace-normalizer"
}

// Order returns 5 - runs between chunker and deduplicator.
func (w *WhitespaceNormalizer) Order() int {
	return 5
}
