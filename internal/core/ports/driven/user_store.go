package driven

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// UserStore handles user persistence (PostgreSQL)
type UserStore interface {
	// Save creates or updates a user
	Save(ctx context.Context, user *domain.User) error

	// Get retrieves a user by ID
	Get(ctx context.Context, id string) (*domain.User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// List retrieves all users for a team
	List(ctx context.Context, teamID string) ([]*domain.User, error)

	// Delete deletes a user
	Delete(ctx context.Context, id string) error

	// UpdateLastLogin updates the last login timestamp
	UpdateLastLogin(ctx context.Context, id string) error
}
