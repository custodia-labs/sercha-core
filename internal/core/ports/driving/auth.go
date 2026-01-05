package driving

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// AuthService handles user authentication
type AuthService interface {
	// Authenticate validates credentials and creates a session
	Authenticate(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error)

	// ValidateToken validates a JWT token and returns the auth context
	ValidateToken(ctx context.Context, token string) (*domain.AuthContext, error)

	// RefreshToken generates a new token from a valid refresh token
	RefreshToken(ctx context.Context, req domain.RefreshRequest) (*domain.LoginResponse, error)

	// Logout invalidates a session
	Logout(ctx context.Context, token string) error

	// LogoutAll invalidates all sessions for a user
	LogoutAll(ctx context.Context, userID string) error

	// ChangePassword changes the password for an authenticated user
	ChangePassword(ctx context.Context, userID string, req domain.ChangePasswordRequest) error
}
