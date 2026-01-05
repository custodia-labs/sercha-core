package domain

import (
	"testing"
	"time"
)

func TestSessionIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "expired session",
			expiresAt: time.Now().Add(-1 * time.Hour),
			expected:  true,
		},
		{
			name:      "valid session",
			expiresAt: time.Now().Add(1 * time.Hour),
			expected:  false,
		},
		{
			name:      "just expired",
			expiresAt: time.Now().Add(-1 * time.Second),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{ExpiresAt: tt.expiresAt}
			if session.IsExpired() != tt.expected {
				t.Errorf("expected IsExpired() = %v", tt.expected)
			}
		})
	}
}

func TestAuthContextIsAdmin(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleAdmin, true},
		{RoleMember, false},
		{RoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			ctx := &AuthContext{Role: tt.role}
			if ctx.IsAdmin() != tt.expected {
				t.Errorf("expected IsAdmin() = %v for role %s", tt.expected, tt.role)
			}
		})
	}
}

func TestLoginRequest(t *testing.T) {
	req := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	if req.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", req.Email)
	}
	if req.Password != "password123" {
		t.Errorf("expected password password123, got %s", req.Password)
	}
}

func TestLoginResponse(t *testing.T) {
	expiresAt := time.Now().Add(24 * time.Hour)
	userSummary := &UserSummary{
		ID:    "user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  RoleMember,
	}

	resp := &LoginResponse{
		Token:        "jwt-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    expiresAt,
		User:         userSummary,
	}

	if resp.Token != "jwt-token" {
		t.Errorf("expected token jwt-token, got %s", resp.Token)
	}
	if resp.RefreshToken != "refresh-token" {
		t.Errorf("expected refresh token refresh-token, got %s", resp.RefreshToken)
	}
	if resp.User.ID != "user-123" {
		t.Errorf("expected user ID user-123, got %s", resp.User.ID)
	}
}

func TestTokenClaims(t *testing.T) {
	now := time.Now()
	claims := &TokenClaims{
		UserID:    "user-123",
		Email:     "test@example.com",
		Role:      RoleAdmin,
		TeamID:    "team-123",
		SessionID: "session-123",
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(24 * time.Hour).Unix(),
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected UserID user-123, got %s", claims.UserID)
	}
	if claims.Role != RoleAdmin {
		t.Errorf("expected Role admin, got %s", claims.Role)
	}
	if claims.ExpiresAt <= claims.IssuedAt {
		t.Error("ExpiresAt should be after IssuedAt")
	}
}

func TestChangePasswordRequest(t *testing.T) {
	req := ChangePasswordRequest{
		CurrentPassword: "old-password",
		NewPassword:     "new-password",
	}

	if req.CurrentPassword != "old-password" {
		t.Errorf("expected current password old-password, got %s", req.CurrentPassword)
	}
	if req.NewPassword != "new-password" {
		t.Errorf("expected new password new-password, got %s", req.NewPassword)
	}
}
