package domain

import "time"

// AIProvider identifies the AI/embedding provider
type AIProvider string

const (
	AIProviderOpenAI    AIProvider = "openai"
	AIProviderAnthropic AIProvider = "anthropic"
	AIProviderOllama    AIProvider = "ollama"
	AIProviderCohere    AIProvider = "cohere"
	AIProviderVoyage    AIProvider = "voyage"
)

// Settings holds team-wide configuration
type Settings struct {
	TeamID string `json:"team_id"`

	// AI Configuration
	AIProvider     AIProvider `json:"ai_provider"`
	EmbeddingModel string     `json:"embedding_model"`
	AIEndpoint     string     `json:"ai_endpoint,omitempty"` // For self-hosted (Ollama)

	// Search Defaults
	DefaultSearchMode SearchMode `json:"default_search_mode"`
	ResultsPerPage    int        `json:"results_per_page"`
	MaxResultsPerPage int        `json:"max_results_per_page"`

	// Sync Configuration
	SyncIntervalMinutes int  `json:"sync_interval_minutes"`
	SyncEnabled         bool `json:"sync_enabled"`

	// Feature Flags
	SemanticSearchEnabled bool `json:"semantic_search_enabled"`
	AutoSuggestEnabled    bool `json:"auto_suggest_enabled"`

	// Metadata
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"` // User ID
}

// DefaultSettings returns sensible defaults for a new team
func DefaultSettings(teamID string) *Settings {
	return &Settings{
		TeamID:                teamID,
		AIProvider:            AIProviderOpenAI,
		EmbeddingModel:        "text-embedding-3-small",
		DefaultSearchMode:     SearchModeHybrid,
		ResultsPerPage:        20,
		MaxResultsPerPage:     100,
		SyncIntervalMinutes:   60,
		SyncEnabled:           true,
		SemanticSearchEnabled: true,
		AutoSuggestEnabled:    true,
		UpdatedAt:             time.Now(),
	}
}

// APIKeyConfig holds API key configuration (stored encrypted)
type APIKeyConfig struct {
	Provider AIProvider `json:"provider"`
	APIKey   string     `json:"-"` // Never serialize
	BaseURL  string     `json:"base_url,omitempty"`
}

// EmbeddingConfig holds embedding model configuration
type EmbeddingConfig struct {
	Provider   AIProvider `json:"provider"`
	Model      string     `json:"model"`
	Dimensions int        `json:"dimensions"`
	BatchSize  int        `json:"batch_size"`
}

// DefaultEmbeddingConfig returns default embedding configuration
func DefaultEmbeddingConfig() *EmbeddingConfig {
	return &EmbeddingConfig{
		Provider:   AIProviderOpenAI,
		Model:      "text-embedding-3-small",
		Dimensions: 1536,
		BatchSize:  100,
	}
}

// AISettings holds AI service configuration (embedding and LLM)
// This can be updated at runtime via API
type AISettings struct {
	TeamID    string            `json:"team_id"`
	Embedding EmbeddingSettings `json:"embedding"`
	LLM       LLMSettings       `json:"llm"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// EmbeddingSettings configures the embedding service
type EmbeddingSettings struct {
	Provider AIProvider `json:"provider"`
	Model    string     `json:"model"`
	APIKey   string     `json:"-"` // Never serialize to JSON
	BaseURL  string     `json:"base_url,omitempty"`
}

// IsConfigured returns true if embedding settings are properly configured
func (e *EmbeddingSettings) IsConfigured() bool {
	if e.Provider == "" {
		return false
	}
	if e.Provider.RequiresAPIKey() && e.APIKey == "" {
		return false
	}
	return true
}

// LLMSettings configures the LLM service
type LLMSettings struct {
	Provider AIProvider `json:"provider"`
	Model    string     `json:"model"`
	APIKey   string     `json:"-"` // Never serialize to JSON
	BaseURL  string     `json:"base_url,omitempty"`
}

// IsConfigured returns true if LLM settings are properly configured
func (l *LLMSettings) IsConfigured() bool {
	if l.Provider == "" {
		return false
	}
	if l.Provider.RequiresAPIKey() && l.APIKey == "" {
		return false
	}
	return true
}

// RequiresAPIKey returns true if this provider requires an API key
func (p AIProvider) RequiresAPIKey() bool {
	switch p {
	case AIProviderOllama:
		return false // Self-hosted, no API key needed
	default:
		return true
	}
}

// IsValid returns true if this is a known provider
func (p AIProvider) IsValid() bool {
	switch p {
	case AIProviderOpenAI, AIProviderAnthropic, AIProviderOllama, AIProviderCohere, AIProviderVoyage:
		return true
	default:
		return false
	}
}

// Validate checks if AISettings are valid
func (s *AISettings) Validate() error {
	if s.Embedding.Provider != "" && !s.Embedding.Provider.IsValid() {
		return ErrInvalidProvider
	}
	if s.LLM.Provider != "" && !s.LLM.Provider.IsValid() {
		return ErrInvalidProvider
	}
	return nil
}
