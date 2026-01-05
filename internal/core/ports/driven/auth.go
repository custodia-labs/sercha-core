package driven

import "github.com/custodia-labs/sercha-core/internal/core/domain"

// AuthAdapter handles authentication cryptographic operations.
// This does NOT handle storage - use SessionStore for session persistence.
type AuthAdapter interface {
	// Password operations
	HashPassword(password string) (string, error)
	VerifyPassword(password, hash string) bool

	// Token operations
	GenerateToken(claims *domain.TokenClaims) (string, error)
	ParseToken(token string) (*domain.TokenClaims, error)
}
