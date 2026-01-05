package driven

import (
	"context"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
)

// SessionStore handles session persistence (Redis)
type SessionStore interface {
	// Save stores a session with TTL based on ExpiresAt
	Save(ctx context.Context, session *domain.Session) error

	// Get retrieves a session by ID
	Get(ctx context.Context, id string) (*domain.Session, error)

	// GetByToken retrieves a session by token value
	GetByToken(ctx context.Context, token string) (*domain.Session, error)

	// GetByRefreshToken retrieves a session by refresh token value
	GetByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error)

	// Delete deletes a session
	Delete(ctx context.Context, id string) error

	// DeleteByToken deletes a session by token
	DeleteByToken(ctx context.Context, token string) error

	// DeleteByUser deletes all sessions for a user (logout everywhere)
	DeleteByUser(ctx context.Context, userID string) error

	// ListByUser lists all active sessions for a user
	ListByUser(ctx context.Context, userID string) ([]*domain.Session, error)
}
