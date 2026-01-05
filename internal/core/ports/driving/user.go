package driving

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Email    string      `json:"email"`
	Password string      `json:"password"`
	Name     string      `json:"name"`
	Role     domain.Role `json:"role"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	Name   *string      `json:"name,omitempty"`
	Role   *domain.Role `json:"role,omitempty"`
	Active *bool        `json:"active,omitempty"`
}

// SetupRequest represents a request to create the initial admin user
type SetupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// SetupResponse represents the response from the setup endpoint
type SetupResponse struct {
	User    *domain.User `json:"user"`
	Message string       `json:"message"`
}

// UserService manages user accounts (admin operations)
type UserService interface {
	// Setup creates the initial admin user (only works if no users exist)
	Setup(ctx context.Context, req SetupRequest) (*SetupResponse, error)

	// Create creates a new user (admin only)
	Create(ctx context.Context, req CreateUserRequest) (*domain.User, error)

	// Get retrieves a user by ID
	Get(ctx context.Context, id string) (*domain.User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// List retrieves all users in the team
	List(ctx context.Context) ([]*domain.User, error)

	// Update updates a user (admin only)
	Update(ctx context.Context, id string, req UpdateUserRequest) (*domain.User, error)

	// Delete deletes a user (admin only)
	Delete(ctx context.Context, id string) error

	// SetPassword sets a new password for a user (admin only)
	SetPassword(ctx context.Context, id string, password string) error
}
