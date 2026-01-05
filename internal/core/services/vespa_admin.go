package services

import (
	"context"
	"fmt"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driving"
	"github.com/custodia-labs/sercha-core/internal/runtime"
)

// Ensure vespaAdminService implements VespaAdminService
var _ driving.VespaAdminService = (*vespaAdminService)(nil)

// vespaAdminService implements the VespaAdminService interface
type vespaAdminService struct {
	deployer      driven.VespaDeployer
	configStore   driven.VespaConfigStore
	settingsStore driven.SettingsStore
	services      *runtime.Services
	teamID        string
}

// NewVespaAdminService creates a new VespaAdminService
func NewVespaAdminService(
	deployer driven.VespaDeployer,
	configStore driven.VespaConfigStore,
	settingsStore driven.SettingsStore,
	services *runtime.Services,
	teamID string,
) driving.VespaAdminService {
	return &vespaAdminService{
		deployer:      deployer,
		configStore:   configStore,
		settingsStore: settingsStore,
		services:      services,
		teamID:        teamID,
	}
}

// Connect connects to Vespa and deploys the schema
func (s *vespaAdminService) Connect(ctx context.Context, req driving.ConnectVespaRequest) (*driving.VespaStatus, error) {
	// Get current config or create default
	config, err := s.configStore.GetVespaConfig(ctx, s.teamID)
	if err != nil {
		config = domain.DefaultVespaConfig(s.teamID)
	}

	// Resolve endpoint
	endpoint := req.Endpoint
	if endpoint == "" {
		endpoint = config.Endpoint
	}
	if endpoint == "" {
		endpoint = "http://vespa:19071"
	}

	// Health check first
	if err := s.deployer.HealthCheck(ctx, endpoint); err != nil {
		return nil, fmt.Errorf("vespa health check failed: %w", err)
	}

	// Determine target schema mode based on embedding service
	var embeddingDim *int
	var embeddingProvider domain.AIProvider

	embSvc := s.services.EmbeddingService()
	if embSvc != nil {
		dim := embSvc.Dimensions()
		embeddingDim = &dim
		// Get provider from AI settings
		aiSettings, _ := s.settingsStore.GetAISettings(ctx, s.teamID)
		if aiSettings != nil {
			embeddingProvider = aiSettings.Embedding.Provider
		}
	}

	// Check upgrade path: can't downgrade from hybrid to bm25
	if config.SchemaMode == domain.VespacSchemaModeHybrid && embeddingDim == nil {
		return nil, fmt.Errorf("cannot downgrade from hybrid to BM25-only schema; embeddings are already indexed")
	}

	// Check dimension compatibility: can't change embedding dimension
	if config.SchemaMode == domain.VespacSchemaModeHybrid && embeddingDim != nil && *embeddingDim != config.EmbeddingDim {
		return nil, fmt.Errorf("cannot change embedding dimension from %d to %d; would require reindexing all documents", config.EmbeddingDim, *embeddingDim)
	}

	// Fetch existing app package for production mode (dev_mode=false)
	var existingPkg *driven.AppPackage
	var clusterInfo *domain.VespaClusterInfo

	if !req.DevMode {
		// Production mode: fetch existing app package and merge our schema
		existingPkg, err = s.deployer.FetchAppPackage(ctx, endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch existing app package: %w", err)
		}
		if existingPkg == nil {
			return nil, fmt.Errorf("no existing Vespa application found; use dev_mode=true or deploy your application package first")
		}
		clusterInfo = existingPkg.ClusterInfo
	}

	// Deploy schema (with existing package for production mode, nil for dev mode)
	result, err := s.deployer.Deploy(ctx, endpoint, embeddingDim, existingPkg)
	if err != nil {
		return nil, fmt.Errorf("vespa schema deployment failed: %w", err)
	}

	// Update config
	config.Endpoint = endpoint
	config.Connected = result.Success
	config.DevMode = req.DevMode
	config.SchemaMode = result.SchemaMode
	config.SchemaVersion = result.SchemaVersion
	config.ClusterInfo = clusterInfo
	if embeddingDim != nil {
		config.EmbeddingDim = *embeddingDim
		config.EmbeddingProvider = embeddingProvider
	}
	config.ConnectedAt = time.Now()
	config.UpdatedAt = time.Now()

	// Save config
	if err := s.configStore.SaveVespaConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to save vespa config: %w", err)
	}

	// Build status response
	status := &driving.VespaStatus{
		Connected:         config.Connected,
		Endpoint:          config.Endpoint,
		DevMode:           config.DevMode,
		SchemaMode:        config.SchemaMode,
		EmbeddingsEnabled: config.HasEmbeddings(),
		EmbeddingDim:      config.EmbeddingDim,
		EmbeddingProvider: config.EmbeddingProvider,
		SchemaVersion:     config.SchemaVersion,
		CanUpgrade:        config.CanUpgradeToHybrid(),
		ReindexRequired:   result.Upgraded,
		Healthy:           true,
		ClusterInfo:       config.ClusterInfo,
	}

	return status, nil
}

// Status returns the current Vespa connection and schema status
func (s *vespaAdminService) Status(ctx context.Context) (*driving.VespaStatus, error) {
	config, err := s.configStore.GetVespaConfig(ctx, s.teamID)
	if err != nil {
		// Return unconfigured status
		return &driving.VespaStatus{
			Connected: false,
			Endpoint:  "http://vespa:19071",
			Healthy:   false,
		}, nil
	}

	// Check if Vespa is actually healthy
	healthy := false
	if config.Connected && config.Endpoint != "" {
		if err := s.deployer.HealthCheck(ctx, config.Endpoint); err == nil {
			healthy = true
		}
	}

	// Check if current embedding service matches stored config
	canUpgrade := config.CanUpgradeToHybrid()
	embSvc := s.services.EmbeddingService()
	if embSvc != nil && config.SchemaMode == domain.VespacSchemaModeBM25 {
		// Have embedding service but running BM25-only schema
		canUpgrade = true
	}

	return &driving.VespaStatus{
		Connected:         config.Connected,
		Endpoint:          config.Endpoint,
		DevMode:           config.DevMode,
		SchemaMode:        config.SchemaMode,
		EmbeddingsEnabled: config.HasEmbeddings(),
		EmbeddingDim:      config.EmbeddingDim,
		EmbeddingProvider: config.EmbeddingProvider,
		SchemaVersion:     config.SchemaVersion,
		CanUpgrade:        canUpgrade,
		ReindexRequired:   false,
		Healthy:           healthy,
		ClusterInfo:       config.ClusterInfo,
	}, nil
}

// HealthCheck performs a health check on the Vespa cluster
func (s *vespaAdminService) HealthCheck(ctx context.Context) error {
	config, err := s.configStore.GetVespaConfig(ctx, s.teamID)
	if err != nil {
		return fmt.Errorf("vespa not configured")
	}

	if !config.Connected {
		return fmt.Errorf("vespa not connected")
	}

	return s.deployer.HealthCheck(ctx, config.Endpoint)
}
