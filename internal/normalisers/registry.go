package normalisers

import (
	"sort"
	"strings"
	"sync"

	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Verify interface compliance
var _ driven.NormaliserRegistry = (*Registry)(nil)

// Registry implements NormaliserRegistry with priority-based selection.
// When multiple normalisers match a MIME type, the highest priority one is used.
type Registry struct {
	mu          sync.RWMutex
	normalisers []driven.Normaliser
}

// NewRegistry creates a new normaliser registry.
func NewRegistry() *Registry {
	return &Registry{
		normalisers: make([]driven.Normaliser, 0),
	}
}

// Register registers a normaliser.
// Normalisers are stored and later selected by priority.
func (r *Registry) Register(normaliser driven.Normaliser) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.normalisers = append(r.normalisers, normaliser)
}

// Get retrieves the best-matching normaliser for a MIME type.
// Returns nil if no normaliser is registered for the type.
// When multiple match, the highest priority normaliser is returned.
func (r *Registry) Get(mimeType string) driven.Normaliser {
	matches := r.GetAll(mimeType)
	if len(matches) == 0 {
		return nil
	}
	return matches[0] // Already sorted by priority (highest first)
}

// GetAll retrieves all normalisers that match a MIME type, sorted by priority (highest first).
func (r *Registry) GetAll(mimeType string) []driven.Normaliser {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []driven.Normaliser

	for _, n := range r.normalisers {
		if matchesMIMEType(n.SupportedTypes(), mimeType) {
			matches = append(matches, n)
		}
	}

	// Sort by priority (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Priority() > matches[j].Priority()
	})

	return matches
}

// List returns all registered MIME types.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	typeSet := make(map[string]struct{})
	for _, n := range r.normalisers {
		for _, t := range n.SupportedTypes() {
			typeSet[t] = struct{}{}
		}
	}

	types := make([]string, 0, len(typeSet))
	for t := range typeSet {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// matchesMIMEType checks if any of the supported types match the given MIME type.
// Supports wildcard matching (e.g., "text/*" matches "text/plain").
func matchesMIMEType(supportedTypes []string, mimeType string) bool {
	// Normalize the input
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))

	// Strip charset and other parameters
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	for _, supported := range supportedTypes {
		supported = strings.ToLower(strings.TrimSpace(supported))

		// Exact match
		if supported == mimeType {
			return true
		}

		// Wildcard match (e.g., "text/*" matches "text/plain")
		if strings.HasSuffix(supported, "/*") {
			prefix := supported[:len(supported)-1] // "text/"
			if strings.HasPrefix(mimeType, prefix) {
				return true
			}
		}

		// Universal wildcard
		if supported == "*/*" {
			return true
		}
	}

	return false
}

// DefaultRegistry creates a registry with common normalisers pre-registered.
func DefaultRegistry() *Registry {
	r := NewRegistry()

	// Register built-in normalisers in priority order
	r.Register(&PlaintextNormaliser{})
	r.Register(&MarkdownNormaliser{})
	r.Register(&HTMLNormaliser{})

	// Register connector-specific normalisers (high priority)
	r.Register(&GitHubIssueNormaliser{})
	r.Register(&GitHubPRNormaliser{})

	return r
}

// PlaintextNormaliser handles plain text content.
type PlaintextNormaliser struct{}

func (n *PlaintextNormaliser) Normalise(content string, mimeType string) string {
	// Basic cleanup: normalize line endings, trim whitespace
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return strings.TrimSpace(content)
}

func (n *PlaintextNormaliser) SupportedTypes() []string {
	return []string{"text/plain", "*/*"} // Fallback for any type
}

func (n *PlaintextNormaliser) Priority() int {
	return 1 // Lowest priority - fallback
}

// MarkdownNormaliser handles Markdown content.
type MarkdownNormaliser struct{}

func (n *MarkdownNormaliser) Normalise(content string, mimeType string) string {
	// Basic Markdown cleanup
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// Remove excessive blank lines (more than 2 consecutive)
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(content)
}

func (n *MarkdownNormaliser) SupportedTypes() []string {
	return []string{"text/markdown", "text/x-markdown"}
}

func (n *MarkdownNormaliser) Priority() int {
	return 50 // Medium priority - format-specific
}

// HTMLNormaliser handles HTML content.
type HTMLNormaliser struct{}

func (n *HTMLNormaliser) Normalise(content string, mimeType string) string {
	// Basic HTML text extraction
	// This is a simple implementation - production would use a proper HTML parser

	// Remove script and style blocks
	content = removeHTMLBlocks(content, "script")
	content = removeHTMLBlocks(content, "style")

	// Remove HTML tags (simple approach)
	content = stripHTMLTags(content)

	// Decode common HTML entities
	content = decodeHTMLEntities(content)

	// Clean up whitespace
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// Collapse multiple spaces
	for strings.Contains(content, "  ") {
		content = strings.ReplaceAll(content, "  ", " ")
	}

	// Remove excessive blank lines
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(content)
}

func (n *HTMLNormaliser) SupportedTypes() []string {
	return []string{"text/html", "application/xhtml+xml"}
}

func (n *HTMLNormaliser) Priority() int {
	return 50 // Medium priority - format-specific
}

// Helper functions for HTML processing

func removeHTMLBlocks(content, tagName string) string {
	result := content

	for {
		startTag := "<" + strings.ToLower(tagName)
		endTag := "</" + strings.ToLower(tagName) + ">"

		startIdx := strings.Index(strings.ToLower(result), startTag)
		if startIdx == -1 {
			break
		}

		endIdx := strings.Index(strings.ToLower(result[startIdx:]), endTag)
		if endIdx == -1 {
			break
		}

		result = result[:startIdx] + result[startIdx+endIdx+len(endTag):]
	}

	return result
}

func stripHTMLTags(content string) string {
	var result strings.Builder
	inTag := false

	for _, r := range content {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			result.WriteRune(' ') // Replace tag with space
		case !inTag:
			result.WriteRune(r)
		}
	}

	return result.String()
}

func decodeHTMLEntities(content string) string {
	// Common HTML entities
	replacements := map[string]string{
		"&nbsp;":   " ",
		"&amp;":    "&",
		"&lt;":     "<",
		"&gt;":     ">",
		"&quot;":   "\"",
		"&apos;":   "'",
		"&#39;":    "'",
		"&mdash;":  "—",
		"&ndash;":  "–",
		"&hellip;": "...",
		"&copy;":   "©",
		"&reg;":    "®",
		"&trade;":  "™",
	}

	for entity, replacement := range replacements {
		content = strings.ReplaceAll(content, entity, replacement)
	}

	return content
}

// GitHubIssueNormaliser handles GitHub issue content.
// It preserves Markdown formatting while cleaning up issue-specific artifacts.
type GitHubIssueNormaliser struct{}

func (n *GitHubIssueNormaliser) Normalise(content string, mimeType string) string {
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// Remove GitHub-specific artifacts

	// Remove "<!-- ... -->" HTML comments (task lists, etc.)
	content = removeHTMLComments(content)

	// Clean up checkbox markers for cleaner text
	content = strings.ReplaceAll(content, "- [ ]", "- [ ]")
	content = strings.ReplaceAll(content, "- [x]", "- [x]")
	content = strings.ReplaceAll(content, "- [X]", "- [x]")

	// Remove excessive blank lines
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(content)
}

func (n *GitHubIssueNormaliser) SupportedTypes() []string {
	return []string{"application/x-github-issue"}
}

func (n *GitHubIssueNormaliser) Priority() int {
	return 90 // High priority - connector-specific
}

// GitHubPRNormaliser handles GitHub pull request content.
// It preserves Markdown formatting while cleaning up PR-specific artifacts.
type GitHubPRNormaliser struct{}

func (n *GitHubPRNormaliser) Normalise(content string, mimeType string) string {
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// Remove HTML comments
	content = removeHTMLComments(content)

	// Remove excessive blank lines
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(content)
}

func (n *GitHubPRNormaliser) SupportedTypes() []string {
	return []string{"application/x-github-pr"}
}

func (n *GitHubPRNormaliser) Priority() int {
	return 90 // High priority - connector-specific
}

// removeHTMLComments removes HTML comments from content.
func removeHTMLComments(content string) string {
	result := content

	for {
		startIdx := strings.Index(result, "<!--")
		if startIdx == -1 {
			break
		}

		endIdx := strings.Index(result[startIdx:], "-->")
		if endIdx == -1 {
			break
		}

		result = result[:startIdx] + result[startIdx+endIdx+3:]
	}

	return result
}
