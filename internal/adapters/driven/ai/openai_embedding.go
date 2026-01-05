package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Ensure OpenAIEmbedding implements EmbeddingService
var _ driven.EmbeddingService = (*OpenAIEmbedding)(nil)

// OpenAIEmbedding implements EmbeddingService using OpenAI's embedding API
type OpenAIEmbedding struct {
	apiKey     string
	model      string
	baseURL    string
	dimensions int
	client     *http.Client
}

// Model dimensions for OpenAI embedding models
var openAIModelDimensions = map[string]int{
	"text-embedding-3-small": 1536,
	"text-embedding-3-large": 3072,
	"text-embedding-ada-002": 1536,
}

// NewOpenAIEmbedding creates a new OpenAI embedding service
func NewOpenAIEmbedding(apiKey, model, baseURL string) (driven.EmbeddingService, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	if model == "" {
		model = "text-embedding-3-small"
	}

	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	dimensions, ok := openAIModelDimensions[model]
	if !ok {
		// Default to 1536 for unknown models
		dimensions = 1536
	}

	return &OpenAIEmbedding{
		apiKey:     apiKey,
		model:      model,
		baseURL:    baseURL,
		dimensions: dimensions,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// embeddingRequest is the request body for OpenAI embedding API
type embeddingRequest struct {
	Input          interface{} `json:"input"` // string or []string
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
}

// embeddingResponse is the response from OpenAI embedding API
type embeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Embed generates embeddings for multiple texts
func (e *OpenAIEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := embeddingRequest{
		Input:          texts,
		Model:          e.model,
		EncodingFormat: "float",
	}

	resp, err := e.doRequest(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	// Sort by index to ensure order matches input
	embeddings := make([][]float32, len(texts))
	for _, d := range resp.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}

	return embeddings, nil
}

// EmbedQuery generates an embedding for a search query
func (e *OpenAIEmbedding) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	embeddings, err := e.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}
	return embeddings[0], nil
}

// Dimensions returns the embedding dimension size
func (e *OpenAIEmbedding) Dimensions() int {
	return e.dimensions
}

// Model returns the model name being used
func (e *OpenAIEmbedding) Model() string {
	return e.model
}

// HealthCheck verifies the embedding service is available
func (e *OpenAIEmbedding) HealthCheck(ctx context.Context) error {
	// Make a small embedding request to verify connectivity
	_, err := e.EmbedQuery(ctx, "health check")
	return err
}

// Close releases resources held by the embedding service
func (e *OpenAIEmbedding) Close() error {
	e.client.CloseIdleConnections()
	return nil
}

// doRequest makes a request to the OpenAI embedding API
func (e *OpenAIEmbedding) doRequest(ctx context.Context, reqBody embeddingRequest) (*embeddingResponse, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var embResp embeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if embResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
			embResp.Error.Message, embResp.Error.Type, embResp.Error.Code)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	return &embResp, nil
}
