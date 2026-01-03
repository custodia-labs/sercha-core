package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/custodia-labs/sercha-core/internal/adapters/driven/ai"
	"github.com/custodia-labs/sercha-core/internal/adapters/driven/auth"
	"github.com/custodia-labs/sercha-core/internal/adapters/driving/http"
	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driving"
	"github.com/custodia-labs/sercha-core/internal/core/services"
	"github.com/custodia-labs/sercha-core/internal/normalisers"
	"github.com/custodia-labs/sercha-core/internal/postprocessors"
	"github.com/custodia-labs/sercha-core/internal/runtime"
	"github.com/custodia-labs/sercha-core/internal/worker"
)

var version = "dev"

func main() {
	// Get run mode from environment (RUN_MODE) or command line arg
	mode := getEnv("RUN_MODE", "all")
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	log.Printf("sercha-core %s starting in %s mode", version, mode)

	// Configuration from environment
	jwtSecret := getEnv("JWT_SECRET", "development-secret-change-in-production")
	teamID := getEnv("TEAM_ID", "default-team")
	port := getEnvInt("PORT", 8080)

	// Setup context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutdown signal received, stopping...")
		cancel()
	}()

	// Driven adapters (infrastructure)
	authAdapter := auth.NewAdapter(jwtSecret)
	aiFactory := ai.NewFactory()

	// Store adapters (TODO: implement with actual database connections)
	var userStore driven.UserStore         // = postgres.NewUserStore(db)
	var sessionStore driven.SessionStore   // = redis.NewSessionStore(client)
	var documentStore driven.DocumentStore // = postgres.NewDocumentStore(db)
	var chunkStore driven.ChunkStore       // = postgres.NewChunkStore(db)
	var sourceStore driven.SourceStore     // = postgres.NewSourceStore(db)
	var syncStore driven.SyncStateStore    // = postgres.NewSyncStateStore(db)
	var searchEngine driven.SearchEngine   // = vespa.NewSearchEngine(vespaClient)
	var settingsStore driven.SettingsStore // = postgres.NewSettingsStore(db)

	// Task queue (Redis preferred, Postgres fallback)
	var taskQueue driven.TaskQueue       // = initTaskQueue()
	var schedulerStore driven.SchedulerStore // = postgres.NewSchedulerStore(db)
	var connectorFactory driven.ConnectorFactory // = connectors.NewFactory(...)

	// Runtime configuration
	sessionBackend := "postgres"
	if getEnv("REDIS_URL", "") != "" {
		sessionBackend = "redis"
	}
	runtimeConfig := domain.NewRuntimeConfig(sessionBackend)
	runtimeServices := runtime.NewServices(runtimeConfig)

	// Initialize registries (shared across all modes)
	normaliserRegistry := normalisers.DefaultRegistry()
	postProcessorPipeline := postprocessors.DefaultPipeline()

	// Services (core business logic)
	authService := services.NewAuthService(userStore, sessionStore, authAdapter)
	userService := services.NewUserService(userStore, sessionStore, authAdapter, teamID)
	sourceService := services.NewSourceService(sourceStore, documentStore, syncStore, searchEngine)
	documentService := services.NewDocumentService(documentStore, chunkStore)
	searchService := services.NewSearchService(searchEngine, documentStore, runtimeServices)
	settingsService := services.NewSettingsService(settingsStore, aiFactory, runtimeServices, teamID)

	// Log startup configuration
	log.Printf("Runtime config: session_backend=%s, embedding=%t, llm=%t, search_mode=%s",
		runtimeConfig.SessionBackend,
		runtimeConfig.EmbeddingAvailable(),
		runtimeConfig.LLMAvailable(),
		runtimeConfig.EffectiveSearchMode())

	// Create sync orchestrator for worker mode
	syncOrchestrator := services.NewSyncOrchestrator(services.SyncOrchestratorConfig{
		SourceStore:      sourceStore,
		DocumentStore:    documentStore,
		ChunkStore:       chunkStore,
		SyncStore:        syncStore,
		SearchEngine:     searchEngine,
		ConnectorFactory: connectorFactory,
		NormaliserReg:    normaliserRegistry,
		Pipeline:         postProcessorPipeline,
		Services:         runtimeServices,
		Logger:           slog.Default(),
	})

	// Create scheduler for worker mode
	scheduler := services.NewScheduler(services.SchedulerConfig{
		Store:        schedulerStore,
		TaskQueue:    taskQueue,
		Logger:       slog.Default(),
	})

	switch mode {
	case "api":
		// API-only mode: HTTP server, no worker
		runAPI(port, authService, userService, searchService, sourceService, documentService, settingsService)

	case "worker":
		// Worker-only mode: Task processing, scheduler, no HTTP server
		runWorkerMode(ctx, taskQueue, syncOrchestrator, scheduler)

	case "all":
		// Combined mode: Run both API and Worker
		// Start worker in background
		go runWorkerMode(ctx, taskQueue, syncOrchestrator, scheduler)
		// Run API in foreground (blocks)
		runAPI(port, authService, userService, searchService, sourceService, documentService, settingsService)

	default:
		log.Fatalf("Unknown mode: %s (use: api, worker, or all)", mode)
	}
}

func runAPI(
	port int,
	authService driving.AuthService,
	userService driving.UserService,
	searchService driving.SearchService,
	sourceService driving.SourceService,
	documentService driving.DocumentService,
	settingsService driving.SettingsService,
) {
	cfg := http.Config{
		Host:    "0.0.0.0",
		Port:    port,
		Version: version,
	}

	server := http.NewServer(
		cfg,
		authService,
		userService,
		searchService,
		sourceService,
		documentService,
		settingsService,
	)

	log.Printf("API server starting on :%d", port)
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// runWorkerMode starts the worker and scheduler.
// It processes tasks from the queue and runs scheduled syncs.
func runWorkerMode(
	ctx context.Context,
	taskQueue driven.TaskQueue,
	orchestrator *services.SyncOrchestrator,
	scheduler *services.Scheduler,
) {
	log.Println("Starting worker mode...")

	// Create worker
	w := worker.NewWorker(worker.WorkerConfig{
		TaskQueue:      taskQueue,
		Orchestrator:   orchestrator,
		Scheduler:      scheduler,
		Logger:         slog.Default(),
		Concurrency:    getEnvInt("WORKER_CONCURRENCY", 2),
		DequeueTimeout: getEnvInt("WORKER_DEQUEUE_TIMEOUT", 5),
	})

	// Start worker
	if err := w.Start(ctx); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	log.Println("Worker started, processing tasks...")
	log.Println("Worker handles:")
	log.Println("  - sync_source: Sync a specific source")
	log.Println("  - sync_all: Sync all enabled sources")

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	log.Println("Stopping worker...")
	w.Stop()
	log.Println("Worker stopped")
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
