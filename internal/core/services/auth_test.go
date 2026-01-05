package services

import (
	"context"
	"testing"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven/mocks"
)

func newTestAuthService() (*mocks.MockUserStore, *mocks.MockSessionStore, *mocks.MockAuthAdapter, *authService) {
	userStore := mocks.NewMockUserStore()
	sessionStore := mocks.NewMockSessionStore()
	authAdapter := mocks.NewMockAuthAdapter()
	svc := NewAuthService(userStore, sessionStore, authAdapter).(*authService)
	return userStore, sessionStore, authAdapter, svc
}

func TestAuthService_Authenticate(t *testing.T) {
	userStore, _, _, svc := newTestAuthService()

	// Create a user with known password
	user := &domain.User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: "password123", // Mock hasher uses plain text comparison
		Name:         "Test User",
		Role:         domain.RoleMember,
		TeamID:       "team-123",
		Active:       true,
		CreatedAt:    time.Now(),
	}
	_ = userStore.Save(context.Background(), user)

	tests := []struct {
		name    string
		req     domain.LoginRequest
		wantErr error
	}{
		{
			name: "valid credentials",
			req: domain.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: nil,
		},
		{
			name: "empty email",
			req: domain.LoginRequest{
				Email:    "",
				Password: "password123",
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "empty password",
			req: domain.LoginRequest{
				Email:    "test@example.com",
				Password: "",
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "wrong password",
			req: domain.LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name: "unknown user",
			req: domain.LoginRequest{
				Email:    "unknown@example.com",
				Password: "password123",
			},
			wantErr: domain.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.Authenticate(context.Background(), tt.req)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp == nil {
				t.Fatal("expected response to be returned")
			}
			if resp.Token == "" {
				t.Error("expected token to be generated")
			}
			if resp.RefreshToken == "" {
				t.Error("expected refresh token to be generated")
			}
			if resp.User.Email != tt.req.Email {
				t.Errorf("expected user email %s, got %s", tt.req.Email, resp.User.Email)
			}
		})
	}
}

func TestAuthService_Authenticate_InactiveUser(t *testing.T) {
	userStore, _, _, svc := newTestAuthService()

	// Create an inactive user
	user := &domain.User{
		ID:           "user-123",
		Email:        "inactive@example.com",
		PasswordHash: "password123",
		Name:         "Inactive User",
		Role:         domain.RoleMember,
		TeamID:       "team-123",
		Active:       false, // User is inactive
		CreatedAt:    time.Now(),
	}
	_ = userStore.Save(context.Background(), user)

	_, err := svc.Authenticate(context.Background(), domain.LoginRequest{
		Email:    "inactive@example.com",
		Password: "password123",
	})

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized for inactive user, got %v", err)
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	_, _, _, svc := newTestAuthService()

	// Empty token
	_, err := svc.ValidateToken(context.Background(), "")
	if err != domain.ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid for empty token, got %v", err)
	}

	// Invalid token format
	_, err = svc.ValidateToken(context.Background(), "invalid-token")
	if err != domain.ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid for invalid token, got %v", err)
	}
}

func TestAuthService_Logout(t *testing.T) {
	_, _, _, svc := newTestAuthService()

	// Logout with empty token should not error
	err := svc.Logout(context.Background(), "")
	if err != nil {
		t.Errorf("expected no error for empty token, got %v", err)
	}

	// Logout with invalid token should not error (already invalid)
	err = svc.Logout(context.Background(), "invalid-token")
	if err != nil {
		t.Errorf("expected no error for invalid token, got %v", err)
	}
}

func TestAuthService_LogoutAll(t *testing.T) {
	userStore, sessionStore, _, svc := newTestAuthService()

	// Create a user and session
	user := &domain.User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: "password123",
		Name:         "Test User",
		Role:         domain.RoleMember,
		TeamID:       "team-123",
		Active:       true,
	}
	_ = userStore.Save(context.Background(), user)

	session := &domain.Session{
		ID:        "session-123",
		UserID:    "user-123",
		Token:     "token-123",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	_ = sessionStore.Save(context.Background(), session)

	// Logout all sessions
	err := svc.LogoutAll(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify session is deleted
	_, err = sessionStore.Get(context.Background(), "session-123")
	if err != domain.ErrSessionNotFound {
		t.Error("expected session to be deleted")
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	userStore, sessionStore, _, svc := newTestAuthService()

	// Empty refresh token
	_, err := svc.RefreshToken(context.Background(), domain.RefreshRequest{
		RefreshToken: "",
	})
	if err != domain.ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid for empty refresh token, got %v", err)
	}

	// Non-existent refresh token
	_, err = svc.RefreshToken(context.Background(), domain.RefreshRequest{
		RefreshToken: "non-existent-refresh-token",
	})
	if err != domain.ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid for non-existent refresh token, got %v", err)
	}

	// Create user and session for valid refresh
	user := &domain.User{
		ID:           "user-refresh",
		Email:        "refresh@example.com",
		PasswordHash: "password123",
		Name:         "Refresh User",
		Role:         domain.RoleMember,
		TeamID:       "team-123",
		Active:       true,
	}
	_ = userStore.Save(context.Background(), user)

	session := &domain.Session{
		ID:           "session-refresh",
		UserID:       "user-refresh",
		Token:        "token-refresh",
		RefreshToken: "valid-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	_ = sessionStore.Save(context.Background(), session)

	// Valid refresh token
	resp, err := svc.RefreshToken(context.Background(), domain.RefreshRequest{
		RefreshToken: "valid-refresh-token",
	})
	if err != nil {
		t.Fatalf("expected no error for valid refresh token, got %v", err)
	}
	if resp.Token == "" {
		t.Error("expected new token to be generated")
	}
	if resp.RefreshToken == "" {
		t.Error("expected new refresh token to be generated")
	}
}

func TestAuthService_ChangePassword(t *testing.T) {
	userStore, _, _, svc := newTestAuthService()

	// Create a user
	user := &domain.User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: "oldpassword",
		Name:         "Test User",
		Role:         domain.RoleMember,
		TeamID:       "team-123",
		Active:       true,
	}
	_ = userStore.Save(context.Background(), user)

	tests := []struct {
		name    string
		userID  string
		req     domain.ChangePasswordRequest
		wantErr error
	}{
		{
			name:   "empty current password",
			userID: "user-123",
			req: domain.ChangePasswordRequest{
				CurrentPassword: "",
				NewPassword:     "newpassword",
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:   "empty new password",
			userID: "user-123",
			req: domain.ChangePasswordRequest{
				CurrentPassword: "oldpassword",
				NewPassword:     "",
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:   "wrong current password",
			userID: "user-123",
			req: domain.ChangePasswordRequest{
				CurrentPassword: "wrongpassword",
				NewPassword:     "newpassword",
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:   "non-existent user",
			userID: "unknown-user",
			req: domain.ChangePasswordRequest{
				CurrentPassword: "oldpassword",
				NewPassword:     "newpassword",
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name:   "valid password change",
			userID: "user-123",
			req: domain.ChangePasswordRequest{
				CurrentPassword: "oldpassword",
				NewPassword:     "newpassword",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ChangePassword(context.Background(), tt.userID, tt.req)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuthService_ChangePassword_InvalidatesSessions(t *testing.T) {
	userStore, sessionStore, _, svc := newTestAuthService()

	// Create a user
	user := &domain.User{
		ID:           "user-456",
		Email:        "test2@example.com",
		PasswordHash: "oldpassword",
		Name:         "Test User 2",
		Role:         domain.RoleMember,
		TeamID:       "team-123",
		Active:       true,
	}
	_ = userStore.Save(context.Background(), user)

	// Create a session
	session := &domain.Session{
		ID:        "session-456",
		UserID:    "user-456",
		Token:     "token-456",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	_ = sessionStore.Save(context.Background(), session)

	// Change password
	err := svc.ChangePassword(context.Background(), "user-456", domain.ChangePasswordRequest{
		CurrentPassword: "oldpassword",
		NewPassword:     "newpassword",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify session is deleted
	_, err = sessionStore.Get(context.Background(), "session-456")
	if err != domain.ErrSessionNotFound {
		t.Error("expected session to be invalidated after password change")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("expected non-empty ID")
	}
	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	token1 := generateRefreshToken()
	token2 := generateRefreshToken()

	if token1 == "" {
		t.Error("expected non-empty refresh token")
	}
	if token1 == token2 {
		t.Error("expected unique refresh tokens")
	}
	// Refresh tokens should be longer than regular IDs
	if len(token1) < 30 {
		t.Error("expected longer refresh token")
	}
}
