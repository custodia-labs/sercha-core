package main

// @title           Sercha Core API
// @version         1.0
// @description     Privacy-focused enterprise search API. Sercha Core provides full-text and semantic search across your connected data sources.

// @contact.name   Sercha OSS
// @contact.url    https://github.com/custodia-labs/sercha-core/issues

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8081
// @BasePath  /api/v1
// @schemes   http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Bearer token. Format: "Bearer {token}"

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/custodia-labs/sercha-core/internal/adapters/driven/ai"
	"github.com/custodia-labs/sercha-core/internal/adapters/driven/auth"
	"github.com/custodia-labs/sercha-core/internal/adapters/driven/postgres"
	postgresqueue "github.com/custodia-labs/sercha-core/internal/adapters/driven/queue/postgres"
	redisqueue "github.com/custodia-labs/sercha-core/internal/adapters/driven/queue/redis"
	redisadapter "github.com/custodia-labs/sercha-core/internal/adapters/driven/redis"
	"github.com/custodia-labs/sercha-core/internal/adapters/driven/vespa"
	"github.com/custodia-labs/sercha-core/internal/adapters/driving/http"
	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driving"
	"github.com/custodia-labs/sercha-core/internal/core/services"
	"github.com/custodia-labs/sercha-core/internal/normalisers"
	"github.com/custodia-labs/sercha-core/internal/postprocessors"
	"github.com/custodia-labs/sercha-core/internal/runtime"
	"github.com/custodia-labs/sercha-core/internal/worker"
	"github.com/redis/go-redis/v9"
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
	databaseURL := getEnv("DATABASE_URL", "postgres://sercha:sercha_dev@localhost:5432/sercha?sslmode=disable")
	redisURL := getEnv("REDIS_URL", "")
	vespaURL := getEnv("VESPA_URL", "http://localhost:19071")

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

	// ===== Initialize PostgreSQL =====
	log.Println("Connecting to PostgreSQL...")
	dbConfig := postgres.Config{
		URL:             databaseURL,
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_SEC", 300)) * time.Second,
		ConnMaxIdleTime: time.Duration(getEnvInt("DB_CONN_MAX_IDLE_SEC", 60)) * time.Second,
	}
	db, err := postgres.Connect(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize schema (idempotent)
	if err := db.InitSchema(ctx); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}
	log.Println("PostgreSQL connected and schema initialized")

	// ===== Initialize Redis (optional) =====
	var redisClient *redis.Client
	if redisURL != "" {
		log.Println("Connecting to Redis...")
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Fatalf("Failed to parse Redis URL: %v", err)
		}
		redisClient = redis.NewClient(opts)
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Fatalf("Failed to connect to Redis: %v", err)
		}
		defer redisClient.Close()
		log.Println("Redis connected")
	}

	// ===== Initialize Vespa =====
	log.Println("Connecting to Vespa...")
	searchEngine := vespa.NewSearchEngine(vespa.DefaultConfig(vespaURL))
	if err := searchEngine.HealthCheck(ctx); err != nil {
		log.Printf("Warning: Vespa health check failed: %v (search may not work)", err)
	} else {
		log.Println("Vespa connected")
	}

	// ===== Driven adapters (infrastructure) =====
	authAdapter := auth.NewAdapter(jwtSecret)
	aiFactory := ai.NewFactory()

	// ===== PostgreSQL Stores =====
	userStore := postgres.NewUserStore(db)
	documentStore := postgres.NewDocumentStore(db)
	chunkStore := postgres.NewChunkStore(db)
	sourceStore := postgres.NewSourceStore(db)
	syncStore := postgres.NewSyncStateStore(db)
	settingsStore := postgres.NewSettingsStore(db)
	schedulerStore := postgres.NewSchedulerStore(db)
	vespaConfigStore := postgres.NewVespaConfigStore(db)

	// ===== Vespa Deployer =====
	vespaDeployer := vespa.NewDeployer()

	// ===== Session Store (Redis if available, otherwise PostgreSQL) =====
	var sessionStore driven.SessionStore
	if redisClient != nil {
		sessionStore = redisadapter.NewSessionStore(redisClient)
		log.Println("Using Redis session store")
	} else {
		sessionStore = postgres.NewSessionStore(db)
		log.Println("Using PostgreSQL session store")
	}

	// ===== Task Queue (Redis if available, otherwise PostgreSQL) =====
	var taskQueue driven.TaskQueue
	if redisClient != nil {
		var err error
		taskQueue, err = redisqueue.NewQueue(redisClient, fmt.Sprintf("worker-%d", os.Getpid()))
		if err != nil {
			log.Fatalf("Failed to create task queue: %v", err)
		}
		log.Println("Using Redis task queue")
	} else {
		taskQueue = postgresqueue.NewQueue(db.DB)
		log.Println("Using PostgreSQL task queue")
	}

	// ===== Distributed Lock (Redis if available, otherwise PostgreSQL advisory locks) =====
	var distributedLock driven.DistributedLock
	if redisClient != nil {
		distributedLock = redisadapter.NewLock(redisClient)
		log.Println("Using Redis distributed lock")
	} else {
		distributedLock = postgres.NewAdvisoryLock(db)
		log.Println("Using PostgreSQL advisory lock")
	}

	// Connector factory (TODO: implement with actual connectors)
	var connectorFactory driven.ConnectorFactory

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
	vespaAdminService := services.NewVespaAdminService(vespaDeployer, vespaConfigStore, settingsStore, runtimeServices, teamID)

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

	// Create scheduler for worker mode (if enabled)
	schedulerEnabled := getEnvBool("SCHEDULER_ENABLED", true)
	schedulerLockRequired := getEnvBool("SCHEDULER_LOCK_REQUIRED", true)

	var scheduler *services.Scheduler
	if schedulerEnabled {
		scheduler = services.NewScheduler(services.SchedulerConfig{
			Store:        schedulerStore,
			TaskQueue:    taskQueue,
			Lock:         distributedLock,
			Logger:       slog.Default(),
			LockRequired: schedulerLockRequired,
		})
		log.Printf("Scheduler enabled (lock_required=%t)", schedulerLockRequired)
	} else {
		log.Println("Scheduler disabled via SCHEDULER_ENABLED=false")
	}

	switch mode {
	case "api":
		// API-only mode: HTTP server, no worker
		runAPI(port, authService, userService, searchService, sourceService, documentService, settingsService, vespaAdminService)

	case "worker":
		// Worker-only mode: Task processing, scheduler, no HTTP server
		runWorkerMode(ctx, taskQueue, syncOrchestrator, scheduler)

	case "all":
		// Combined mode: Run both API and Worker
		// Start worker in background
		go runWorkerMode(ctx, taskQueue, syncOrchestrator, scheduler)
		// Run API in foreground (blocks)
		runAPI(port, authService, userService, searchService, sourceService, documentService, settingsService, vespaAdminService)

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
	vespaAdminService driving.VespaAdminService,
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
		vespaAdminService,
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

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}
