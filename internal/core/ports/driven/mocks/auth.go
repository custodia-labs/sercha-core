package mocks

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Ensure MockAuthAdapter implements AuthAdapter
var _ driven.AuthAdapter = (*MockAuthAdapter)(nil)

// MockAuthAdapter is a mock implementation of AuthAdapter for testing.
// It uses plain text password comparison and base64-encoded JSON for tokens.
// NOT secure - only for testing.
type MockAuthAdapter struct{}

// NewMockAuthAdapter creates a new MockAuthAdapter
func NewMockAuthAdapter() *MockAuthAdapter {
	return &MockAuthAdapter{}
}

// HashPassword returns the password as-is (for testing only)
func (m *MockAuthAdapter) HashPassword(password string) (string, error) {
	return password, nil
}

// VerifyPassword compares password with hash directly (for testing only)
func (m *MockAuthAdapter) VerifyPassword(password, hash string) bool {
	return password == hash
}

// GenerateToken creates a base64-encoded JSON token from claims
func (m *MockAuthAdapter) GenerateToken(claims *domain.TokenClaims) (string, error) {
	data, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// ParseToken decodes a base64-encoded JSON token and returns claims
func (m *MockAuthAdapter) ParseToken(token string) (*domain.TokenClaims, error) {
	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	var claims domain.TokenClaims
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, domain.ErrTokenInvalid
	}

	return &claims, nil
}
