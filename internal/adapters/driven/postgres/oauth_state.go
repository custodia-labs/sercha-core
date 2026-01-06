package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Ensure OAuthStateStore implements the interface.
var _ driven.OAuthStateStore = (*OAuthStateStore)(nil)

// DefaultOAuthStateTTL is the default time-to-live for OAuth states.
const DefaultOAuthStateTTL = 10 * time.Minute

// OAuthStateStore implements driven.OAuthStateStore using PostgreSQL.
type OAuthStateStore struct {
	db  *sql.DB
	ttl time.Duration
}

// NewOAuthStateStore creates a new PostgreSQL-backed OAuth state store.
func NewOAuthStateStore(db *sql.DB) *OAuthStateStore {
	return &OAuthStateStore{
		db:  db,
		ttl: DefaultOAuthStateTTL,
	}
}

// NewOAuthStateStoreWithTTL creates an OAuth state store with custom TTL.
func NewOAuthStateStoreWithTTL(db *sql.DB, ttl time.Duration) *OAuthStateStore {
	return &OAuthStateStore{
		db:  db,
		ttl: ttl,
	}
}

// Save stores a new OAuth state.
func (s *OAuthStateStore) Save(ctx context.Context, state *driven.OAuthState) error {
	now := time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	if state.ExpiresAt.IsZero() {
		state.ExpiresAt = now.Add(s.ttl)
	}

	query := `
		INSERT INTO oauth_states (state, provider_type, code_verifier, redirect_uri, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.db.ExecContext(ctx, query,
		state.State,
		state.ProviderType,
		state.CodeVerifier,
		state.RedirectURI,
		state.CreatedAt,
		state.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("save oauth state: %w", err)
	}

	return nil
}

// GetAndDelete atomically retrieves and deletes the state.
// Uses DELETE ... RETURNING for atomic single-use semantics.
func (s *OAuthStateStore) GetAndDelete(ctx context.Context, state string) (*driven.OAuthState, error) {
	query := `
		DELETE FROM oauth_states
		WHERE state = $1 AND expires_at > NOW()
		RETURNING state, provider_type, code_verifier, redirect_uri, created_at, expires_at
	`

	var oauthState driven.OAuthState
	err := s.db.QueryRowContext(ctx, query, state).Scan(
		&oauthState.State,
		&oauthState.ProviderType,
		&oauthState.CodeVerifier,
		&oauthState.RedirectURI,
		&oauthState.CreatedAt,
		&oauthState.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // State not found or expired
	}
	if err != nil {
		return nil, fmt.Errorf("get and delete oauth state: %w", err)
	}

	return &oauthState, nil
}

// Cleanup removes expired states.
func (s *OAuthStateStore) Cleanup(ctx context.Context) error {
	query := `DELETE FROM oauth_states WHERE expires_at < NOW()`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("cleanup oauth states: %w", err)
	}

	return nil
}
