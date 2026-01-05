package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Verify interface compliance
var _ driven.UserStore = (*UserStore)(nil)

// UserStore implements driven.UserStore using PostgreSQL
type UserStore struct {
	db *DB
}

// NewUserStore creates a new UserStore
func NewUserStore(db *DB) *UserStore {
	return &UserStore{db: db}
}

// Save creates or updates a user
func (s *UserStore) Save(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, role, team_id, active, created_at, updated_at, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			email = EXCLUDED.email,
			password_hash = EXCLUDED.password_hash,
			name = EXCLUDED.name,
			role = EXCLUDED.role,
			team_id = EXCLUDED.team_id,
			active = EXCLUDED.active,
			updated_at = EXCLUDED.updated_at,
			last_login_at = EXCLUDED.last_login_at
	`

	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		string(user.Role),
		user.TeamID,
		user.Active,
		user.CreatedAt,
		user.UpdatedAt,
		NullTime(user.LastLoginAt),
	)
	return err
}

// Get retrieves a user by ID
func (s *UserStore) Get(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, team_id, active, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`

	var user domain.User
	var lastLoginAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.TeamID,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	user.LastLoginAt = TimePtr(lastLoginAt)
	return &user, nil
}

// GetByEmail retrieves a user by email
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, team_id, active, created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1
	`

	var user domain.User
	var lastLoginAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.TeamID,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	user.LastLoginAt = TimePtr(lastLoginAt)
	return &user, nil
}

// List retrieves all users for a team
func (s *UserStore) List(ctx context.Context, teamID string) ([]*domain.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, team_id, active, created_at, updated_at, last_login_at
		FROM users
		WHERE team_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		var lastLoginAt sql.NullTime

		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Name,
			&user.Role,
			&user.TeamID,
			&user.Active,
			&user.CreatedAt,
			&user.UpdatedAt,
			&lastLoginAt,
		)
		if err != nil {
			return nil, err
		}

		user.LastLoginAt = TimePtr(lastLoginAt)
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// Delete deletes a user
func (s *UserStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
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

// UpdateLastLogin updates the last login timestamp
func (s *UserStore) UpdateLastLogin(ctx context.Context, id string) error {
	query := `UPDATE users SET last_login_at = $1, updated_at = $1 WHERE id = $2`
	now := time.Now()
	result, err := s.db.ExecContext(ctx, query, now, id)
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
