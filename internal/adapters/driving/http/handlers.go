package http

import (
	"encoding/json"
	"net/http"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driving"
)

// ErrorResponse represents an API error response
// @Description API error response
type ErrorResponse struct {
	Error string `json:"error" example:"invalid request body"`
}

// StatusResponse represents a simple status response
// @Description Simple status response
type StatusResponse struct {
	Status string `json:"status" example:"ok"`
}

// VersionResponse represents the API version response
// @Description API version response
type VersionResponse struct {
	Version string `json:"version" example:"1.0.0"`
}

// Health endpoints

// handleHealth godoc
// @Summary      Health check
// @Description  Returns the health status of the API
// @Tags         Health
// @Produce      json
// @Success      200  {object}  StatusResponse
// @Router       /health [get]
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReady godoc
// @Summary      Readiness check
// @Description  Returns the readiness status of the API (checks database and service connections)
// @Tags         Health
// @Produce      json
// @Success      200  {object}  StatusResponse
// @Router       /ready [get]
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// TODO: Check database and service connections
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// handleVersion godoc
// @Summary      Get API version
// @Description  Returns the current API version
// @Tags         Health
// @Produce      json
// @Success      200  {object}  VersionResponse
// @Router       /version [get]
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": s.version})
}

// Auth endpoints

// handleLogin godoc
// @Summary      User login
// @Description  Authenticate with email and password to receive a JWT token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      domain.LoginRequest  true  "Login credentials"
// @Success      200      {object}  domain.LoginResponse
// @Failure      400      {object}  ErrorResponse  "Invalid request body"
// @Failure      401      {object}  ErrorResponse  "Invalid credentials or account disabled"
// @Failure      500      {object}  ErrorResponse  "Internal server error"
// @Router       /auth/login [post]
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := s.authService.Authenticate(r.Context(), req)
	if err != nil {
		switch err {
		case domain.ErrInvalidCredentials:
			writeError(w, http.StatusUnauthorized, "invalid credentials")
		case domain.ErrUnauthorized:
			writeError(w, http.StatusUnauthorized, "account disabled")
		default:
			writeError(w, http.StatusInternalServerError, "authentication failed")
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleRefresh godoc
// @Summary      Refresh token
// @Description  Exchange a refresh token for a new JWT token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      domain.RefreshRequest  true  "Refresh token"
// @Success      200      {object}  domain.LoginResponse
// @Failure      400      {object}  ErrorResponse  "Invalid request body"
// @Failure      401      {object}  ErrorResponse  "Invalid refresh token"
// @Router       /auth/refresh [post]
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req domain.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := s.authService.RefreshToken(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleLogout godoc
// @Summary      Logout user
// @Description  Invalidate the current session token
// @Tags         Authentication
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  StatusResponse
// @Router       /auth/logout [post]
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	_ = s.authService.Logout(r.Context(), token)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Setup endpoint (no auth required, one-time use)

// handleSetup godoc
// @Summary      Initial setup
// @Description  Create the initial admin user. This endpoint can only be called once when no users exist.
// @Tags         Setup
// @Accept       json
// @Produce      json
// @Param        request  body      driving.SetupRequest  true  "Admin user details"
// @Success      201      {object}  driving.SetupResponse
// @Failure      400      {object}  ErrorResponse  "Invalid input"
// @Failure      403      {object}  ErrorResponse  "Setup already complete"
// @Failure      500      {object}  ErrorResponse  "Setup failed"
// @Router       /setup [post]
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	var req driving.SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := s.userService.Setup(r.Context(), req)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			writeError(w, http.StatusBadRequest, "email, password, and name are required")
		case domain.ErrForbidden:
			writeError(w, http.StatusForbidden, "setup already complete")
		default:
			writeError(w, http.StatusInternalServerError, "setup failed")
		}
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// User endpoints

// handleGetMe godoc
// @Summary      Get current user
// @Description  Get the currently authenticated user's profile
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  domain.UserSummary
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      404  {object}  ErrorResponse  "User not found"
// @Router       /me [get]
func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := s.userService.Get(r.Context(), authCtx.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user.ToSummary())
}

// handleListUsers godoc
// @Summary      List all users
// @Description  Get a list of all users (admin only)
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   domain.UserSummary
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      403  {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /users [get]
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.userService.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	summaries := make([]*domain.UserSummary, len(users))
	for i, u := range users {
		summaries[i] = u.ToSummary()
	}

	writeJSON(w, http.StatusOK, summaries)
}

// handleCreateUser godoc
// @Summary      Create user
// @Description  Create a new user (admin only)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      driving.CreateUserRequest  true  "User details"
// @Success      201      {object}  domain.UserSummary
// @Failure      400      {object}  ErrorResponse  "Invalid input"
// @Failure      401      {object}  ErrorResponse  "Unauthorized"
// @Failure      403      {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      409      {object}  ErrorResponse  "User already exists"
// @Failure      500      {object}  ErrorResponse  "Internal server error"
// @Router       /users [post]
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req driving.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := s.userService.Create(r.Context(), req)
	if err != nil {
		switch err {
		case domain.ErrAlreadyExists:
			writeError(w, http.StatusConflict, "user already exists")
		case domain.ErrInvalidInput:
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "failed to create user")
		}
		return
	}

	writeJSON(w, http.StatusCreated, user.ToSummary())
}

// handleDeleteUser godoc
// @Summary      Delete user
// @Description  Delete a user by ID (admin only)
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  StatusResponse
// @Failure      400  {object}  ErrorResponse  "Missing user ID"
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      403  {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      404  {object}  ErrorResponse  "User not found"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /users/{id} [delete]
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing user id")
		return
	}

	if err := s.userService.Delete(r.Context(), id); err != nil {
		switch err {
		case domain.ErrNotFound:
			writeError(w, http.StatusNotFound, "user not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete user")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Search endpoints

// SearchRequest represents a search query request
// @Description Search query request
type searchRequest struct {
	Query     string            `json:"query" example:"how to configure authentication"`
	Mode      domain.SearchMode `json:"mode,omitempty" example:"hybrid" enums:"hybrid,text,semantic"`
	Limit     int               `json:"limit,omitempty" example:"20"`
	Offset    int               `json:"offset,omitempty" example:"0"`
	SourceIDs []string          `json:"source_ids,omitempty"`
}

// handleSearch godoc
// @Summary      Search documents
// @Description  Execute a search query across all indexed documents. Supports hybrid (BM25 + semantic), text-only, and semantic-only modes.
// @Tags         Search
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      searchRequest  true  "Search query"
// @Success      200      {object}  domain.SearchResult
// @Failure      400      {object}  ErrorResponse  "Invalid request or missing query"
// @Failure      401      {object}  ErrorResponse  "Unauthorized"
// @Failure      500      {object}  ErrorResponse  "Search failed"
// @Router       /search [post]
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	opts := domain.SearchOptions{
		Mode:      req.Mode,
		Limit:     req.Limit,
		Offset:    req.Offset,
		SourceIDs: req.SourceIDs,
	}

	result, err := s.searchService.Search(r.Context(), req.Query, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Document endpoints

// handleGetDocument godoc
// @Summary      Get document
// @Description  Get a document by ID with all its chunks
// @Tags         Documents
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Document ID"
// @Success      200  {object}  domain.DocumentWithChunks
// @Failure      400  {object}  ErrorResponse  "Missing document ID"
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      404  {object}  ErrorResponse  "Document not found"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /documents/{id} [get]
func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing document id")
		return
	}

	doc, err := s.docService.GetWithChunks(r.Context(), id)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			writeError(w, http.StatusNotFound, "document not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to get document")
		}
		return
	}

	writeJSON(w, http.StatusOK, doc)
}

// Source endpoints

// handleListSources godoc
// @Summary      List sources
// @Description  Get a list of all configured data sources with sync status
// @Tags         Sources
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   domain.SourceSummary
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /sources [get]
func (s *Server) handleListSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.sourceService.ListWithSummary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sources")
		return
	}

	writeJSON(w, http.StatusOK, sources)
}

// handleGetSource godoc
// @Summary      Get source
// @Description  Get a data source by ID
// @Tags         Sources
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Source ID"
// @Success      200  {object}  domain.Source
// @Failure      400  {object}  ErrorResponse  "Missing source ID"
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      404  {object}  ErrorResponse  "Source not found"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /sources/{id} [get]
func (s *Server) handleGetSource(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing source id")
		return
	}

	source, err := s.sourceService.Get(r.Context(), id)
	if err != nil {
		switch err {
		case domain.ErrNotFound:
			writeError(w, http.StatusNotFound, "source not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to get source")
		}
		return
	}

	writeJSON(w, http.StatusOK, source)
}

// handleCreateSource godoc
// @Summary      Create source
// @Description  Create a new data source (admin only)
// @Tags         Sources
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      driving.CreateSourceRequest  true  "Source configuration"
// @Success      201      {object}  domain.Source
// @Failure      400      {object}  ErrorResponse  "Invalid input"
// @Failure      401      {object}  ErrorResponse  "Unauthorized"
// @Failure      403      {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      409      {object}  ErrorResponse  "Source already exists"
// @Failure      500      {object}  ErrorResponse  "Internal server error"
// @Router       /sources [post]
func (s *Server) handleCreateSource(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req driving.CreateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	source, err := s.sourceService.Create(r.Context(), authCtx.UserID, req)
	if err != nil {
		switch err {
		case domain.ErrAlreadyExists:
			writeError(w, http.StatusConflict, "source already exists")
		case domain.ErrInvalidInput:
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "failed to create source")
		}
		return
	}

	writeJSON(w, http.StatusCreated, source)
}

// handleDeleteSource godoc
// @Summary      Delete source
// @Description  Delete a data source by ID (admin only). This also removes all indexed documents from this source.
// @Tags         Sources
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Source ID"
// @Success      200  {object}  StatusResponse
// @Failure      400  {object}  ErrorResponse  "Missing source ID"
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      403  {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      404  {object}  ErrorResponse  "Source not found"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /sources/{id} [delete]
func (s *Server) handleDeleteSource(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing source id")
		return
	}

	if err := s.sourceService.Delete(r.Context(), id); err != nil {
		switch err {
		case domain.ErrNotFound:
			writeError(w, http.StatusNotFound, "source not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete source")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// SyncAcceptedResponse represents the response when sync is triggered
// @Description Sync accepted response
type SyncAcceptedResponse struct {
	Status   string `json:"status" example:"accepted"`
	SourceID string `json:"source_id" example:"src_abc123"`
}

// handleTriggerSync godoc
// @Summary      Trigger sync
// @Description  Trigger a sync operation for a specific source (admin only)
// @Tags         Sync
// @Produce      json
// @Security     BearerAuth
// @Param        sourceId  path      string  true  "Source ID"
// @Success      202       {object}  SyncAcceptedResponse
// @Failure      400       {object}  ErrorResponse  "Missing source ID"
// @Failure      401       {object}  ErrorResponse  "Unauthorized"
// @Failure      403       {object}  ErrorResponse  "Forbidden - admin only"
// @Router       /sync/{sourceId} [post]
func (s *Server) handleTriggerSync(w http.ResponseWriter, r *http.Request) {
	sourceID := r.PathValue("sourceId")
	if sourceID == "" {
		writeError(w, http.StatusBadRequest, "missing source id")
		return
	}

	// TODO: Trigger sync via SyncOrchestrator
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":    "accepted",
		"source_id": sourceID,
	})
}

// Settings endpoints

// handleGetSettings godoc
// @Summary      Get settings
// @Description  Get system settings (admin only)
// @Tags         Settings
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  domain.Settings
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      403  {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /settings [get]
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.settingsService.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get settings")
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

// handleUpdateSettings godoc
// @Summary      Update settings
// @Description  Update system settings (admin only)
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      driving.UpdateSettingsRequest  true  "Settings to update"
// @Success      200      {object}  domain.Settings
// @Failure      400      {object}  ErrorResponse  "Invalid request"
// @Failure      401      {object}  ErrorResponse  "Unauthorized"
// @Failure      403      {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      500      {object}  ErrorResponse  "Internal server error"
// @Router       /settings [put]
func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req driving.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	settings, err := s.settingsService.Update(r.Context(), authCtx.UserID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update settings")
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

// AI Settings endpoints

// handleGetAISettings godoc
// @Summary      Get AI settings
// @Description  Get AI provider configuration (admin only). API keys are masked.
// @Tags         AI Settings
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  aiSettingsResponse
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      403  {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /settings/ai [get]
func (s *Server) handleGetAISettings(w http.ResponseWriter, r *http.Request) {
	aiSettings, err := s.settingsService.GetAISettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get AI settings")
		return
	}

	// Mask API keys for security
	resp := aiSettingsResponse{
		Embedding: aiProviderInfo{
			Provider:    aiSettings.Embedding.Provider,
			Model:       aiSettings.Embedding.Model,
			BaseURL:     aiSettings.Embedding.BaseURL,
			HasAPIKey:   aiSettings.Embedding.APIKey != "",
			IsConfigured: aiSettings.Embedding.IsConfigured(),
		},
		LLM: aiProviderInfo{
			Provider:    aiSettings.LLM.Provider,
			Model:       aiSettings.LLM.Model,
			BaseURL:     aiSettings.LLM.BaseURL,
			HasAPIKey:   aiSettings.LLM.APIKey != "",
			IsConfigured: aiSettings.LLM.IsConfigured(),
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

type aiSettingsResponse struct {
	Embedding aiProviderInfo `json:"embedding"`
	LLM       aiProviderInfo `json:"llm"`
}

// aiProviderInfo represents AI provider configuration status
// @Description AI provider configuration status
type aiProviderInfo struct {
	Provider     domain.AIProvider `json:"provider,omitempty" example:"openai"`
	Model        string            `json:"model,omitempty" example:"text-embedding-3-small"`
	BaseURL      string            `json:"base_url,omitempty" example:"https://api.openai.com/v1"`
	HasAPIKey    bool              `json:"has_api_key" example:"true"`
	IsConfigured bool              `json:"is_configured" example:"true"`
}

// handleUpdateAISettings godoc
// @Summary      Update AI settings
// @Description  Update AI provider configuration (admin only). This triggers hot-reload of AI services.
// @Tags         AI Settings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      driving.UpdateAISettingsRequest  true  "AI settings to update"
// @Success      200      {object}  driving.AISettingsStatus
// @Failure      400      {object}  ErrorResponse  "Invalid configuration or unsupported provider"
// @Failure      401      {object}  ErrorResponse  "Unauthorized"
// @Failure      403      {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      500      {object}  ErrorResponse  "Internal server error"
// @Router       /settings/ai [put]
func (s *Server) handleUpdateAISettings(w http.ResponseWriter, r *http.Request) {
	var req driving.UpdateAISettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	status, err := s.settingsService.UpdateAISettings(r.Context(), req)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			writeError(w, http.StatusBadRequest, "invalid AI configuration")
		case domain.ErrInvalidProvider:
			writeError(w, http.StatusBadRequest, "unsupported AI provider")
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// handleGetAIStatus godoc
// @Summary      Get AI status
// @Description  Get the current status of AI services including embedding, LLM, and Vespa connection status
// @Tags         AI Settings
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  driving.AISettingsStatus
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /settings/ai/status [get]
func (s *Server) handleGetAIStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.settingsService.GetAIStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get AI status")
		return
	}

	// Add Vespa status if service is available
	if s.vespaAdminService != nil {
		vespaStatus, err := s.vespaAdminService.Status(r.Context())
		if err == nil && vespaStatus != nil {
			status.Vespa = driving.VespaServiceStatus{
				Connected:         vespaStatus.Connected,
				SchemaMode:        vespaStatus.SchemaMode,
				EmbeddingsEnabled: vespaStatus.EmbeddingsEnabled,
				EmbeddingDim:      vespaStatus.EmbeddingDim,
				CanUpgrade:        vespaStatus.CanUpgrade,
				Healthy:           vespaStatus.Healthy,
			}
			// Include embedding dimension in embedding status if Vespa is configured with embeddings
			if vespaStatus.EmbeddingsEnabled && vespaStatus.EmbeddingDim > 0 {
				status.Embedding.EmbeddingDim = vespaStatus.EmbeddingDim
			}
		}
	}

	writeJSON(w, http.StatusOK, status)
}

// handleTestAIConnection godoc
// @Summary      Test AI connection
// @Description  Test connectivity to configured AI providers (admin only)
// @Tags         AI Settings
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  StatusResponse
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      403  {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      503  {object}  ErrorResponse  "AI service unavailable"
// @Router       /settings/ai/test [post]
func (s *Server) handleTestAIConnection(w http.ResponseWriter, r *http.Request) {
	if err := s.settingsService.TestConnection(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "connected"})
}

// Vespa admin endpoints

// handleVespaConnect godoc
// @Summary      Connect to Vespa
// @Description  Connect to a Vespa cluster and deploy the search schema (admin only). Use dev_mode=true for local development, dev_mode=false for production clusters.
// @Tags         Vespa
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      driving.ConnectVespaRequest  true  "Vespa connection settings"
// @Success      200      {object}  driving.VespaStatus
// @Failure      400      {object}  ErrorResponse  "Invalid request"
// @Failure      401      {object}  ErrorResponse  "Unauthorized"
// @Failure      403      {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      500      {object}  ErrorResponse  "Connection failed"
// @Router       /admin/vespa/connect [post]
func (s *Server) handleVespaConnect(w http.ResponseWriter, r *http.Request) {
	var req driving.ConnectVespaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.ContentLength > 0 {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	status, err := s.vespaAdminService.Connect(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// handleVespaStatus godoc
// @Summary      Get Vespa status
// @Description  Get the current Vespa connection and schema status (admin only)
// @Tags         Vespa
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  driving.VespaStatus
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      403  {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /admin/vespa/status [get]
func (s *Server) handleVespaStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.vespaAdminService.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// handleVespaHealth godoc
// @Summary      Vespa health check
// @Description  Check if the Vespa cluster is healthy (admin only)
// @Tags         Vespa
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  StatusResponse
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      403  {object}  ErrorResponse  "Forbidden - admin only"
// @Failure      503  {object}  ErrorResponse  "Vespa unhealthy"
// @Router       /admin/vespa/health [get]
func (s *Server) handleVespaHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.vespaAdminService.HealthCheck(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
