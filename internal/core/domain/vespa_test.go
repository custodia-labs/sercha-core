package domain

import (
	"testing"
	"time"
)

func TestVespaConfig_IsConnected(t *testing.T) {
	tests := []struct {
		name     string
		config   *VespaConfig
		expected bool
	}{
		{
			name: "connected with BM25",
			config: &VespaConfig{
				Connected:  true,
				SchemaMode: VespacSchemaModeBM25,
			},
			expected: true,
		},
		{
			name: "connected with hybrid",
			config: &VespaConfig{
				Connected:  true,
				SchemaMode: VespacSchemaModeHybrid,
			},
			expected: true,
		},
		{
			name: "not connected",
			config: &VespaConfig{
				Connected:  false,
				SchemaMode: VespacSchemaModeBM25,
			},
			expected: false,
		},
		{
			name: "connected but no schema",
			config: &VespaConfig{
				Connected:  true,
				SchemaMode: VespacSchemaModeNone,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsConnected(); got != tt.expected {
				t.Errorf("IsConnected() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestVespaConfig_HasEmbeddings(t *testing.T) {
	tests := []struct {
		name     string
		config   *VespaConfig
		expected bool
	}{
		{
			name: "hybrid with embedding dim",
			config: &VespaConfig{
				SchemaMode:   VespacSchemaModeHybrid,
				EmbeddingDim: 1536,
			},
			expected: true,
		},
		{
			name: "hybrid without embedding dim",
			config: &VespaConfig{
				SchemaMode:   VespacSchemaModeHybrid,
				EmbeddingDim: 0,
			},
			expected: false,
		},
		{
			name: "BM25 only",
			config: &VespaConfig{
				SchemaMode:   VespacSchemaModeBM25,
				EmbeddingDim: 0,
			},
			expected: false,
		},
		{
			name: "BM25 with dimension (invalid state)",
			config: &VespaConfig{
				SchemaMode:   VespacSchemaModeBM25,
				EmbeddingDim: 1536,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.HasEmbeddings(); got != tt.expected {
				t.Errorf("HasEmbeddings() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestVespaConfig_CanUpgradeToHybrid(t *testing.T) {
	tests := []struct {
		name     string
		config   *VespaConfig
		expected bool
	}{
		{
			name: "BM25 can upgrade",
			config: &VespaConfig{
				SchemaMode: VespacSchemaModeBM25,
			},
			expected: true,
		},
		{
			name: "hybrid cannot upgrade",
			config: &VespaConfig{
				SchemaMode: VespacSchemaModeHybrid,
			},
			expected: false,
		},
		{
			name: "none cannot upgrade",
			config: &VespaConfig{
				SchemaMode: VespacSchemaModeNone,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.CanUpgradeToHybrid(); got != tt.expected {
				t.Errorf("CanUpgradeToHybrid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDefaultVespaConfig(t *testing.T) {
	teamID := "test-team"
	config := DefaultVespaConfig(teamID)

	if config.TeamID != teamID {
		t.Errorf("TeamID = %v, want %v", config.TeamID, teamID)
	}
	if config.Endpoint != "http://vespa:19071" {
		t.Errorf("Endpoint = %v, want http://vespa:19071", config.Endpoint)
	}
	if config.Connected {
		t.Error("Connected should be false by default")
	}
	if config.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
	if time.Since(config.UpdatedAt) > time.Second {
		t.Error("UpdatedAt should be recent")
	}
}

func TestVespaSchemaMode_Constants(t *testing.T) {
	// Verify constants have expected values
	if VespacSchemaModeNone != "" {
		t.Errorf("VespacSchemaModeNone = %q, want empty string", VespacSchemaModeNone)
	}
	if VespacSchemaModeBM25 != "bm25" {
		t.Errorf("VespacSchemaModeBM25 = %q, want 'bm25'", VespacSchemaModeBM25)
	}
	if VespacSchemaModeHybrid != "hybrid" {
		t.Errorf("VespacSchemaModeHybrid = %q, want 'hybrid'", VespacSchemaModeHybrid)
	}
}

func TestVespaDeployResult(t *testing.T) {
	result := VespaDeployResult{
		Success:       true,
		SchemaMode:    VespacSchemaModeHybrid,
		EmbeddingDim:  1536,
		SchemaVersion: "v1-hybrid-dim1536",
		Upgraded:      true,
		Message:       "Deployed hybrid schema",
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.SchemaMode != VespacSchemaModeHybrid {
		t.Errorf("SchemaMode = %v, want %v", result.SchemaMode, VespacSchemaModeHybrid)
	}
	if result.EmbeddingDim != 1536 {
		t.Errorf("EmbeddingDim = %v, want 1536", result.EmbeddingDim)
	}
	if result.SchemaVersion != "v1-hybrid-dim1536" {
		t.Errorf("SchemaVersion = %v, want v1-hybrid-dim1536", result.SchemaVersion)
	}
	if !result.Upgraded {
		t.Error("Upgraded should be true")
	}
	if result.Message != "Deployed hybrid schema" {
		t.Errorf("Message = %v, want 'Deployed hybrid schema'", result.Message)
	}
}
