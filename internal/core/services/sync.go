package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-core/internal/runtime"
)

// We need a ChunkStore for saving chunks separately
// The SyncOrchestrator needs both DocumentStore and ChunkStore

// SyncOrchestrator coordinates the document sync pipeline.
// It implements the 7-step sync flow:
//  1. Get source config
//  2. Create connector
//  3. Validate connector
//  4. Get sync state (cursor for incremental sync)
//  5. Fetch documents
//  6. Process each document (normalise → chunk → embed → store → index)
//  7. Update sync cursor
type SyncOrchestrator struct {
	sourceStore      driven.SourceStore
	documentStore    driven.DocumentStore
	chunkStore       driven.ChunkStore
	syncStore        driven.SyncStateStore
	searchEngine     driven.SearchEngine
	connectorFactory driven.ConnectorFactory
	normaliserReg    driven.NormaliserRegistry
	pipeline         driven.PostProcessorPipeline
	services         *runtime.Services
	logger           *slog.Logger
}

// SyncOrchestratorConfig holds dependencies for SyncOrchestrator.
type SyncOrchestratorConfig struct {
	SourceStore      driven.SourceStore
	DocumentStore    driven.DocumentStore
	ChunkStore       driven.ChunkStore
	SyncStore        driven.SyncStateStore
	SearchEngine     driven.SearchEngine
	ConnectorFactory driven.ConnectorFactory
	NormaliserReg    driven.NormaliserRegistry
	Pipeline         driven.PostProcessorPipeline
	Services         *runtime.Services
	Logger           *slog.Logger
}

// NewSyncOrchestrator creates a new sync orchestrator.
func NewSyncOrchestrator(cfg SyncOrchestratorConfig) *SyncOrchestrator {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &SyncOrchestrator{
		sourceStore:      cfg.SourceStore,
		documentStore:    cfg.DocumentStore,
		chunkStore:       cfg.ChunkStore,
		syncStore:        cfg.SyncStore,
		searchEngine:     cfg.SearchEngine,
		connectorFactory: cfg.ConnectorFactory,
		normaliserReg:    cfg.NormaliserReg,
		pipeline:         cfg.Pipeline,
		services:         cfg.Services,
		logger:           logger,
	}
}

// SyncSource synchronizes a single source.
// This is the main entry point for the sync pipeline.
func (o *SyncOrchestrator) SyncSource(ctx context.Context, sourceID string) (*domain.SyncResult, error) {
	startTime := time.Now()

	o.logger.Info("starting sync", "source_id", sourceID)

	// Step 1: Get source config
	source, err := o.sourceStore.Get(ctx, sourceID)
	if err != nil {
		return o.failSync(ctx, sourceID, startTime, fmt.Errorf("failed to get source: %w", err))
	}

	if !source.Enabled {
		return o.failSync(ctx, sourceID, startTime, fmt.Errorf("source is disabled"))
	}

	// Step 2: Create connector
	connector, err := o.connectorFactory.Create(ctx, source)
	if err != nil {
		return o.failSync(ctx, sourceID, startTime, fmt.Errorf("failed to create connector: %w", err))
	}

	// Step 3: Validate connector (test connection)
	if err := connector.TestConnection(ctx, source); err != nil {
		return o.failSync(ctx, sourceID, startTime, fmt.Errorf("connection test failed: %w", err))
	}

	// Step 4: Get sync state
	syncState, err := o.syncStore.Get(ctx, sourceID)
	if err != nil {
		// Create initial sync state
		syncState = &domain.SyncState{
			SourceID: sourceID,
			Status:   domain.SyncStatusIdle,
			Stats:    domain.SyncStats{},
		}
	}

	// Mark as running
	now := time.Now()
	syncState.Status = domain.SyncStatusRunning
	syncState.StartedAt = &now
	syncState.Error = ""
	if err := o.syncStore.Save(ctx, syncState); err != nil {
		o.logger.Warn("failed to update sync state to running", "error", err)
	}

	// Step 5: Fetch documents
	cursor := syncState.Cursor
	stats := domain.SyncStats{}
	var lastCursor string

	for {
		select {
		case <-ctx.Done():
			return o.failSync(ctx, sourceID, startTime, ctx.Err())
		default:
		}

		changes, nextCursor, err := connector.FetchChanges(ctx, source, cursor)
		if err != nil {
			return o.failSync(ctx, sourceID, startTime, fmt.Errorf("failed to fetch changes: %w", err))
		}

		if len(changes) == 0 {
			break
		}

		// Step 6: Process each document
		for _, change := range changes {
			if err := o.processChange(ctx, source, change, &stats); err != nil {
				o.logger.Warn("failed to process change",
					"source_id", sourceID,
					"external_id", change.ExternalID,
					"error", err,
				)
				stats.Errors++
			}
		}

		lastCursor = nextCursor

		// No more pages
		if nextCursor == "" || nextCursor == cursor {
			break
		}
		cursor = nextCursor
	}

	// Step 7: Update sync cursor
	completedAt := time.Now()
	syncState.Status = domain.SyncStatusCompleted
	syncState.LastSyncAt = &completedAt
	syncState.CompletedAt = &completedAt
	syncState.Cursor = lastCursor
	syncState.Stats = stats
	syncState.Error = ""

	if err := o.syncStore.Save(ctx, syncState); err != nil {
		o.logger.Warn("failed to update sync state", "error", err)
	}

	duration := time.Since(startTime).Seconds()

	o.logger.Info("sync completed",
		"source_id", sourceID,
		"duration_seconds", duration,
		"documents_added", stats.DocumentsAdded,
		"documents_updated", stats.DocumentsUpdated,
		"documents_deleted", stats.DocumentsDeleted,
		"chunks_indexed", stats.ChunksIndexed,
		"errors", stats.Errors,
	)

	return &domain.SyncResult{
		SourceID: sourceID,
		Success:  true,
		Stats:    stats,
		Duration: duration,
		Cursor:   lastCursor,
	}, nil
}

// SyncAll synchronizes all enabled sources for a team.
func (o *SyncOrchestrator) SyncAll(ctx context.Context) ([]*domain.SyncResult, error) {
	sources, err := o.sourceStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}

	var results []*domain.SyncResult
	for _, source := range sources {
		if !source.Enabled {
			continue
		}

		result, err := o.SyncSource(ctx, source.ID)
		if err != nil {
			o.logger.Error("sync failed", "source_id", source.ID, "error", err)
			results = append(results, &domain.SyncResult{
				SourceID: source.ID,
				Success:  false,
				Error:    err.Error(),
			})
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// processChange processes a single document change.
func (o *SyncOrchestrator) processChange(
	ctx context.Context,
	source *domain.Source,
	change *domain.Change,
	stats *domain.SyncStats,
) error {
	switch change.Type {
	case domain.ChangeTypeDeleted:
		return o.processDelete(ctx, source.ID, change, stats)
	case domain.ChangeTypeAdded, domain.ChangeTypeModified:
		return o.processAddOrUpdate(ctx, source, change, stats)
	default:
		return fmt.Errorf("unknown change type: %s", change.Type)
	}
}

// processDelete handles document deletion.
func (o *SyncOrchestrator) processDelete(
	ctx context.Context,
	sourceID string,
	change *domain.Change,
	stats *domain.SyncStats,
) error {
	// Find document by external ID
	doc, err := o.documentStore.GetByExternalID(ctx, sourceID, change.ExternalID)
	if err != nil {
		// Document might not exist, which is fine
		return nil
	}

	// Delete from search engine
	if o.searchEngine != nil {
		if err := o.searchEngine.DeleteByDocument(ctx, doc.ID); err != nil {
			o.logger.Warn("failed to delete from search engine", "doc_id", doc.ID, "error", err)
		}
	}

	// Delete from document store
	if err := o.documentStore.Delete(ctx, doc.ID); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	stats.DocumentsDeleted++
	return nil
}

// processAddOrUpdate handles document creation or update.
func (o *SyncOrchestrator) processAddOrUpdate(
	ctx context.Context,
	source *domain.Source,
	change *domain.Change,
	stats *domain.SyncStats,
) error {
	doc := change.Document
	content := change.Content

	if doc == nil {
		return fmt.Errorf("document is nil for change type %s", change.Type)
	}

	// Check if document exists (for update tracking)
	existingDoc, _ := o.documentStore.GetByExternalID(ctx, source.ID, change.ExternalID)
	isUpdate := existingDoc != nil

	// Ensure document has required fields
	if doc.ID == "" {
		doc.ID = generateID()
	}
	doc.SourceID = source.ID
	doc.ExternalID = change.ExternalID
	now := time.Now()
	doc.UpdatedAt = now
	doc.IndexedAt = now
	if !isUpdate {
		doc.CreatedAt = now
	} else {
		doc.ID = existingDoc.ID // Preserve original ID
		doc.CreatedAt = existingDoc.CreatedAt
	}

	// Step 6a: Check exclusion rules (TODO: implement exclusion rules)

	// Step 6b: Normalise content
	normaliser := o.normaliserReg.Get(doc.MimeType)
	if normaliser != nil {
		content = normaliser.Normalise(content, doc.MimeType)
	}

	// Step 6c: PostProcess (chunk)
	chunks := o.pipeline.Process(content)

	// Step 6d: Generate embeddings (if EmbeddingService available)
	var domainChunks []*domain.Chunk
	for i, chunk := range chunks {
		domainChunk := &domain.Chunk{
			ID:         fmt.Sprintf("%s-chunk-%d", doc.ID, i),
			DocumentID: doc.ID,
			SourceID:   source.ID,
			Content:    chunk.Content,
			Position:   chunk.Position,
			StartChar:  chunk.StartOffset,
			EndChar:    chunk.EndOffset,
			CreatedAt:  now,
		}

		// Generate embedding if available
		if o.services != nil {
			embeddingService := o.services.EmbeddingService()
			if embeddingService != nil {
				embeddings, err := embeddingService.Embed(ctx, []string{chunk.Content})
				if err != nil {
					o.logger.Warn("failed to generate embedding", "chunk_id", domainChunk.ID, "error", err)
				} else if len(embeddings) > 0 {
					domainChunk.Embedding = embeddings[0]
				}
			}
		}

		domainChunks = append(domainChunks, domainChunk)
	}

	// Step 6e: Save to DocumentStore
	if err := o.documentStore.Save(ctx, doc); err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	// Save chunks to ChunkStore
	if o.chunkStore != nil {
		for _, chunk := range domainChunks {
			if err := o.chunkStore.Save(ctx, chunk); err != nil {
				o.logger.Warn("failed to save chunk", "chunk_id", chunk.ID, "error", err)
			}
		}
	}

	// Step 6f & 6g: Index chunks in SearchEngine (Vespa)
	if o.searchEngine != nil {
		if err := o.searchEngine.Index(ctx, domainChunks); err != nil {
			o.logger.Warn("failed to index chunks", "doc_id", doc.ID, "error", err)
		}
	}

	// Update stats
	if isUpdate {
		stats.DocumentsUpdated++
	} else {
		stats.DocumentsAdded++
	}
	stats.ChunksIndexed += len(domainChunks)

	return nil
}

// failSync marks a sync as failed and returns the result.
func (o *SyncOrchestrator) failSync(
	ctx context.Context,
	sourceID string,
	startTime time.Time,
	err error,
) (*domain.SyncResult, error) {
	duration := time.Since(startTime).Seconds()

	o.logger.Error("sync failed", "source_id", sourceID, "duration_seconds", duration, "error", err)

	// Update sync state
	syncState, getErr := o.syncStore.Get(ctx, sourceID)
	if getErr == nil {
		now := time.Now()
		syncState.Status = domain.SyncStatusFailed
		syncState.CompletedAt = &now
		syncState.Error = err.Error()
		_ = o.syncStore.Save(ctx, syncState)
	}

	return &domain.SyncResult{
		SourceID: sourceID,
		Success:  false,
		Error:    err.Error(),
		Duration: duration,
	}, err
}
