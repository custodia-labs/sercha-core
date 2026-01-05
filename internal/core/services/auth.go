package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driving"
)

// Ensure authService implements AuthService
var _ driving.AuthService = (*authService)(nil)

// authService implements the AuthService interface
type authService struct {
	userStore    driven.UserStore
	sessionStore driven.SessionStore
	authAdapter  driven.AuthAdapter
	tokenTTL     time.Duration
}

// NewAuthService creates a new AuthService
func NewAuthService(
	userStore driven.UserStore,
	sessionStore driven.SessionStore,
	authAdapter driven.AuthAdapter,
) driving.AuthService {
	return &authService{
		userStore:    userStore,
		sessionStore: sessionStore,
		authAdapter:  authAdapter,
		tokenTTL:     24 * time.Hour,
	}
}

// Authenticate validates credentials and creates a session
func (s *authService) Authenticate(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error) {
	// Validate input
	if req.Email == "" || req.Password == "" {
		return nil, domain.ErrInvalidInput
	}

	// Get user by email
	user, err := s.userStore.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Check if user is active
	if !user.Active {
		return nil, domain.ErrUnauthorized
	}

	// Verify password
	if !s.authAdapter.VerifyPassword(req.Password, user.PasswordHash) {
		return nil, domain.ErrInvalidCredentials
	}

	// Generate session ID and token
	sessionID := generateID()
	claims := &domain.TokenClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		TeamID:    user.TeamID,
		SessionID: sessionID,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(s.tokenTTL).Unix(),
	}

	token, err := s.authAdapter.GenerateToken(claims)
	if err != nil {
		return nil, err
	}

	refreshToken := generateRefreshToken()
	expiresAt := time.Now().Add(s.tokenTTL)

	// Create session
	session := &domain.Session{
		ID:           sessionID,
		UserID:       user.ID,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
	}

	// Save session
	if err := s.sessionStore.Save(ctx, session); err != nil {
		return nil, err
	}

	// Update last login
	_ = s.userStore.UpdateLastLogin(ctx, user.ID)

	return &domain.LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         user.ToSummary(),
	}, nil
}

// ValidateToken validates a JWT token and returns the auth context
func (s *authService) ValidateToken(ctx context.Context, token string) (*domain.AuthContext, error) {
	if token == "" {
		return nil, domain.ErrTokenInvalid
	}

	// Parse and validate JWT
	claims, err := s.authAdapter.ParseToken(token)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	// Check expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, domain.ErrTokenExpired
	}

	// Verify session exists
	session, err := s.sessionStore.Get(ctx, claims.SessionID)
	if err != nil {
		return nil, domain.ErrSessionNotFound
	}

	if session.IsExpired() {
		return nil, domain.ErrTokenExpired
	}

	return &domain.AuthContext{
		UserID:    claims.UserID,
		Email:     claims.Email,
		Role:      claims.Role,
		TeamID:    claims.TeamID,
		SessionID: claims.SessionID,
	}, nil
}

// RefreshToken generates a new token from a valid refresh token
func (s *authService) RefreshToken(ctx context.Context, req domain.RefreshRequest) (*domain.LoginResponse, error) {
	if req.RefreshToken == "" {
		return nil, domain.ErrTokenInvalid
	}

	// Find session by refresh token
	session, err := s.sessionStore.GetByRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	// Check if session is expired
	if session.IsExpired() {
		return nil, domain.ErrTokenExpired
	}

	// Get user for claims
	user, err := s.userStore.Get(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	// Generate new token
	newSessionID := generateID()
	claims := &domain.TokenClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		TeamID:    user.TeamID,
		SessionID: newSessionID,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(s.tokenTTL).Unix(),
	}

	newToken, err := s.authAdapter.GenerateToken(claims)
	if err != nil {
		return nil, err
	}

	newRefreshToken := generateRefreshToken()
	expiresAt := time.Now().Add(s.tokenTTL)

	// Delete old session
	_ = s.sessionStore.Delete(ctx, session.ID)

	// Create new session
	newSession := &domain.Session{
		ID:           newSessionID,
		UserID:       user.ID,
		Token:        newToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
	}

	if err := s.sessionStore.Save(ctx, newSession); err != nil {
		return nil, err
	}

	return &domain.LoginResponse{
		Token:        newToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
		User:         user.ToSummary(),
	}, nil
}

// Logout invalidates a session
func (s *authService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	claims, err := s.authAdapter.ParseToken(token)
	if err != nil {
		return nil // Already invalid, nothing to do
	}

	return s.sessionStore.Delete(ctx, claims.SessionID)
}

// LogoutAll invalidates all sessions for a user
func (s *authService) LogoutAll(ctx context.Context, userID string) error {
	return s.sessionStore.DeleteByUser(ctx, userID)
}

// ChangePassword changes the password for an authenticated user
func (s *authService) ChangePassword(ctx context.Context, userID string, req domain.ChangePasswordRequest) error {
	if req.CurrentPassword == "" || req.NewPassword == "" {
		return domain.ErrInvalidInput
	}

	user, err := s.userStore.Get(ctx, userID)
	if err != nil {
		return err
	}

	// Verify current password
	if !s.authAdapter.VerifyPassword(req.CurrentPassword, user.PasswordHash) {
		return domain.ErrInvalidCredentials
	}

	// Hash new password
	newHash, err := s.authAdapter.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = newHash
	user.UpdatedAt = time.Now()

	if err := s.userStore.Save(ctx, user); err != nil {
		return err
	}

	// Invalidate all sessions (force re-login)
	return s.sessionStore.DeleteByUser(ctx, userID)
}

// Helper functions

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateRefreshToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
