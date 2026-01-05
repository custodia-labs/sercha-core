package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	server := &Server{version: "test"}

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got %s", response["status"])
	}
}

func TestReadyHandler(t *testing.T) {
	server := &Server{version: "test"}

	req := httptest.NewRequest("GET", "/ready", nil)
	rr := httptest.NewRecorder()

	server.handleReady(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["status"] != "ready" {
		t.Errorf("expected status 'ready', got %s", response["status"])
	}
}

func TestVersionHandler(t *testing.T) {
	server := &Server{version: "1.2.3"}

	req := httptest.NewRequest("GET", "/version", nil)
	rr := httptest.NewRecorder()

	server.handleVersion(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["version"] != "1.2.3" {
		t.Errorf("expected version '1.2.3', got %s", response["version"])
	}
}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()

	data := map[string]string{"foo": "bar"}
	writeJSON(rr, http.StatusCreated, data)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", rr.Header().Get("Content-Type"))
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["foo"] != "bar" {
		t.Errorf("expected foo 'bar', got %s", response["foo"])
	}
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()

	writeError(rr, http.StatusBadRequest, "invalid input")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["error"] != "invalid input" {
		t.Errorf("expected error 'invalid input', got %s", response["error"])
	}
}

func TestSearchRequest(t *testing.T) {
	reqBody := searchRequest{
		Query:     "test query",
		Mode:      "hybrid",
		Limit:     20,
		Offset:    0,
		SourceIDs: []string{"source-1", "source-2"},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	var decoded searchRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if decoded.Query != "test query" {
		t.Errorf("expected query 'test query', got %s", decoded.Query)
	}
	if decoded.Limit != 20 {
		t.Errorf("expected limit 20, got %d", decoded.Limit)
	}
	if len(decoded.SourceIDs) != 2 {
		t.Errorf("expected 2 source IDs, got %d", len(decoded.SourceIDs))
	}
}

func TestHandleLogin_InvalidJSON(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	server.handleLogin(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleRefresh_InvalidJSON(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	server.handleRefresh(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleLogout_NoToken(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()

	server.handleLogout(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestHandleSearch_InvalidJSON(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest("POST", "/api/v1/search", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	server.handleSearch(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleSearch_EmptyQuery(t *testing.T) {
	server := &Server{}

	body, _ := json.Marshal(searchRequest{Query: ""})
	req := httptest.NewRequest("POST", "/api/v1/search", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	server.handleSearch(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["error"] != "query is required" {
		t.Errorf("expected error 'query is required', got %s", response["error"])
	}
}

func TestHandleCreateUser_InvalidJSON(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	server.handleCreateUser(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleCreateSource_InvalidJSON(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest("POST", "/api/v1/sources", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	server.handleCreateSource(rr, req)

	// Should return unauthorized since there's no auth context
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}
