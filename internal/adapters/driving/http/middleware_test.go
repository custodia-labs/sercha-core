package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "valid bearer token",
			header:   "Bearer abc123",
			expected: "abc123",
		},
		{
			name:     "bearer with extra spaces",
			header:   "Bearer   token-with-spaces   ",
			expected: "token-with-spaces",
		},
		{
			name:     "lowercase bearer",
			header:   "bearer token123",
			expected: "token123",
		},
		{
			name:     "empty header",
			header:   "",
			expected: "",
		},
		{
			name:     "no bearer prefix",
			header:   "token123",
			expected: "",
		},
		{
			name:     "basic auth",
			header:   "Basic dXNlcjpwYXNz",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			result := extractBearerToken(req)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetAuthContext(t *testing.T) {
	// Test with empty context (context.TODO represents unknown context)
	result := GetAuthContext(context.TODO())
	if result != nil {
		t.Error("expected nil for empty context")
	}

	// Test with context without auth
	ctx := context.Background()
	result = GetAuthContext(ctx)
	if result != nil {
		t.Error("expected nil for context without auth")
	}

	// Test with context with auth
	authCtx := &domain.AuthContext{
		UserID: "user-123",
		Email:  "test@example.com",
		Role:   domain.RoleAdmin,
	}
	ctx = context.WithValue(context.Background(), authContextKey, authCtx)
	result = GetAuthContext(ctx)
	if result == nil {
		t.Fatal("expected auth context to be returned")
	}
	if result.UserID != "user-123" {
		t.Errorf("expected user ID user-123, got %s", result.UserID)
	}
	if result.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", result.Email)
	}
	if result.Role != domain.RoleAdmin {
		t.Errorf("expected role admin, got %s", result.Role)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	middleware := NewLoggingMiddleware()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	middleware.Handler(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	middleware := NewRecoveryMiddleware()

	// Handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Should not panic
	middleware.Handler(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	middleware := NewCORSMiddleware([]string{"https://example.com", "*"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test allowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	middleware.Handler(handler).ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("expected CORS origin header to be set")
	}

	// Test preflight
	req = httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr = httptest.NewRecorder()

	middleware.Handler(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for preflight, got %d", rr.Code)
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	middleware := NewCORSMiddleware([]string{"https://example.com"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()

	middleware.Handler(handler).ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS header for disallowed origin")
	}
}

func TestResponseWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

	// Default status
	if rw.statusCode != http.StatusOK {
		t.Errorf("expected default status 200, got %d", rw.statusCode)
	}

	// Write header
	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rw.statusCode)
	}
}
