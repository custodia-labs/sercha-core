package postgres

import (
	"context"
	"database/sql"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Verify interface compliance
var _ driven.SessionStore = (*SessionStore)(nil)

// SessionStore implements driven.SessionStore using PostgreSQL
type SessionStore struct {
	db *DB
}

// NewSessionStore creates a new SessionStore
func NewSessionStore(db *DB) *SessionStore {
	return &SessionStore{db: db}
}

// Save stores a session
func (s *SessionStore) Save(ctx context.Context, session *domain.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, token, refresh_token, expires_at, created_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			token = EXCLUDED.token,
			refresh_token = EXCLUDED.refresh_token,
			expires_at = EXCLUDED.expires_at,
			user_agent = EXCLUDED.user_agent,
			ip_address = EXCLUDED.ip_address
	`

	_, err := s.db.ExecContext(ctx, query,
		session.ID,
		session.UserID,
		session.Token,
		session.RefreshToken,
		session.ExpiresAt,
		session.CreatedAt,
		session.UserAgent,
		session.IPAddress,
	)
	return err
}

// Get retrieves a session by ID
func (s *SessionStore) Get(ctx context.Context, id string) (*domain.Session, error) {
	query := `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, user_agent, ip_address
		FROM sessions
		WHERE id = $1
	`

	var session domain.Session
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.RefreshToken,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UserAgent,
		&session.IPAddress,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &session, nil
}

// GetByToken retrieves a session by token value
func (s *SessionStore) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	query := `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, user_agent, ip_address
		FROM sessions
		WHERE token = $1
	`

	var session domain.Session
	err := s.db.QueryRowContext(ctx, query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.RefreshToken,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UserAgent,
		&session.IPAddress,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &session, nil
}

// GetByRefreshToken retrieves a session by refresh token value
func (s *SessionStore) GetByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	query := `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, user_agent, ip_address
		FROM sessions
		WHERE refresh_token = $1
	`

	var session domain.Session
	err := s.db.QueryRowContext(ctx, query, refreshToken).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.RefreshToken,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UserAgent,
		&session.IPAddress,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &session, nil
}

// Delete deletes a session
func (s *SessionStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// DeleteByToken deletes a session by token
func (s *SessionStore) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE token = $1`
	_, err := s.db.ExecContext(ctx, query, token)
	return err
}

// DeleteByUser deletes all sessions for a user (logout everywhere)
func (s *SessionStore) DeleteByUser(ctx context.Context, userID string) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

// ListByUser lists all active sessions for a user
func (s *SessionStore) ListByUser(ctx context.Context, userID string) ([]*domain.Session, error) {
	query := `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, user_agent, ip_address
		FROM sessions
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		var session domain.Session
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.Token,
			&session.RefreshToken,
			&session.ExpiresAt,
			&session.CreatedAt,
			&session.UserAgent,
			&session.IPAddress,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, &session)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}
