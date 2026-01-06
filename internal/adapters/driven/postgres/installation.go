package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
	"github.com/lib/pq"
)

// Ensure InstallationStore implements the interface.
var _ driven.InstallationStore = (*InstallationStore)(nil)

// InstallationStore implements driven.InstallationStore using PostgreSQL.
type InstallationStore struct {
	db        *sql.DB
	encryptor *SecretEncryptor
}

// NewInstallationStore creates a new PostgreSQL-backed installation store.
func NewInstallationStore(db *sql.DB, encryptor *SecretEncryptor) *InstallationStore {
	return &InstallationStore{
		db:        db,
		encryptor: encryptor,
	}
}

// Save stores a new installation or updates an existing one.
func (s *InstallationStore) Save(ctx context.Context, inst *domain.Installation) error {
	// Encrypt secrets if present
	var secretBlob []byte
	if inst.Secrets != nil {
		var err error
		secretBlob, err = s.encryptor.Encrypt(inst.Secrets)
		if err != nil {
			return fmt.Errorf("encrypt secrets: %w", err)
		}
	}

	query := `
		INSERT INTO connector_installations (
			id, name, provider_type, auth_method, secret_blob,
			oauth_token_type, oauth_expiry, oauth_scopes, account_id,
			created_at, updated_at, last_used_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			provider_type = EXCLUDED.provider_type,
			auth_method = EXCLUDED.auth_method,
			secret_blob = EXCLUDED.secret_blob,
			oauth_token_type = EXCLUDED.oauth_token_type,
			oauth_expiry = EXCLUDED.oauth_expiry,
			oauth_scopes = EXCLUDED.oauth_scopes,
			account_id = EXCLUDED.account_id,
			updated_at = EXCLUDED.updated_at,
			last_used_at = EXCLUDED.last_used_at
	`

	now := time.Now()
	if inst.CreatedAt.IsZero() {
		inst.CreatedAt = now
	}
	inst.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, query,
		inst.ID,
		inst.Name,
		inst.ProviderType,
		inst.AuthMethod,
		secretBlob,
		nullString(inst.OAuthTokenType),
		nullTime(inst.OAuthExpiry),
		pq.Array(inst.OAuthScopes),
		nullString(inst.AccountID),
		inst.CreatedAt,
		inst.UpdatedAt,
		nullTime(inst.LastUsedAt),
	)
	if err != nil {
		return fmt.Errorf("save installation: %w", err)
	}

	return nil
}

// Get retrieves an installation by ID with decrypted secrets.
func (s *InstallationStore) Get(ctx context.Context, id string) (*domain.Installation, error) {
	query := `
		SELECT id, name, provider_type, auth_method, secret_blob,
			   oauth_token_type, oauth_expiry, oauth_scopes, account_id,
			   created_at, updated_at, last_used_at
		FROM connector_installations
		WHERE id = $1
	`

	var inst domain.Installation
	var secretBlob []byte
	var oauthTokenType, accountID sql.NullString
	var oauthExpiry, lastUsedAt sql.NullTime
	var oauthScopes []string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&inst.ID,
		&inst.Name,
		&inst.ProviderType,
		&inst.AuthMethod,
		&secretBlob,
		&oauthTokenType,
		&oauthExpiry,
		pq.Array(&oauthScopes),
		&accountID,
		&inst.CreatedAt,
		&inst.UpdatedAt,
		&lastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get installation: %w", err)
	}

	// Decrypt secrets if present
	if len(secretBlob) > 0 {
		inst.Secrets = &domain.InstallationSecrets{}
		if err := s.encryptor.Decrypt(secretBlob, inst.Secrets); err != nil {
			return nil, fmt.Errorf("decrypt secrets: %w", err)
		}
	}

	inst.OAuthTokenType = oauthTokenType.String
	if oauthExpiry.Valid {
		inst.OAuthExpiry = &oauthExpiry.Time
	}
	inst.OAuthScopes = oauthScopes
	inst.AccountID = accountID.String
	if lastUsedAt.Valid {
		inst.LastUsedAt = &lastUsedAt.Time
	}

	return &inst, nil
}

// List retrieves all installations as summaries (no secrets).
func (s *InstallationStore) List(ctx context.Context) ([]*domain.InstallationSummary, error) {
	query := `
		SELECT id, name, provider_type, auth_method, account_id,
			   oauth_expiry, created_at, last_used_at
		FROM connector_installations
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list installations: %w", err)
	}
	defer rows.Close()

	var summaries []*domain.InstallationSummary
	for rows.Next() {
		var summary domain.InstallationSummary
		var accountID sql.NullString
		var oauthExpiry, lastUsedAt sql.NullTime

		if err := rows.Scan(
			&summary.ID,
			&summary.Name,
			&summary.ProviderType,
			&summary.AuthMethod,
			&accountID,
			&oauthExpiry,
			&summary.CreatedAt,
			&lastUsedAt,
		); err != nil {
			return nil, fmt.Errorf("scan installation: %w", err)
		}

		summary.AccountID = accountID.String
		if oauthExpiry.Valid {
			summary.OAuthExpiry = &oauthExpiry.Time
		}
		if lastUsedAt.Valid {
			summary.LastUsedAt = &lastUsedAt.Time
		}

		summaries = append(summaries, &summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate installations: %w", err)
	}

	return summaries, nil
}

// Delete removes an installation by ID.
func (s *InstallationStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM connector_installations WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete installation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// GetByProvider retrieves installations for a provider type (no secrets).
func (s *InstallationStore) GetByProvider(ctx context.Context, providerType domain.ProviderType) ([]*domain.InstallationSummary, error) {
	query := `
		SELECT id, name, provider_type, auth_method, account_id,
			   oauth_expiry, created_at, last_used_at
		FROM connector_installations
		WHERE provider_type = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, providerType)
	if err != nil {
		return nil, fmt.Errorf("list installations by provider: %w", err)
	}
	defer rows.Close()

	var summaries []*domain.InstallationSummary
	for rows.Next() {
		var summary domain.InstallationSummary
		var accountID sql.NullString
		var oauthExpiry, lastUsedAt sql.NullTime

		if err := rows.Scan(
			&summary.ID,
			&summary.Name,
			&summary.ProviderType,
			&summary.AuthMethod,
			&accountID,
			&oauthExpiry,
			&summary.CreatedAt,
			&lastUsedAt,
		); err != nil {
			return nil, fmt.Errorf("scan installation: %w", err)
		}

		summary.AccountID = accountID.String
		if oauthExpiry.Valid {
			summary.OAuthExpiry = &oauthExpiry.Time
		}
		if lastUsedAt.Valid {
			summary.LastUsedAt = &lastUsedAt.Time
		}

		summaries = append(summaries, &summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate installations: %w", err)
	}

	return summaries, nil
}

// GetByAccountID retrieves an installation by provider type and account ID.
func (s *InstallationStore) GetByAccountID(ctx context.Context, providerType domain.ProviderType, accountID string) (*domain.Installation, error) {
	query := `
		SELECT id, name, provider_type, auth_method, secret_blob,
			   oauth_token_type, oauth_expiry, oauth_scopes, account_id,
			   created_at, updated_at, last_used_at
		FROM connector_installations
		WHERE provider_type = $1 AND account_id = $2
	`

	var inst domain.Installation
	var secretBlob []byte
	var oauthTokenType, accountIDVal sql.NullString
	var oauthExpiry, lastUsedAt sql.NullTime
	var oauthScopes []string

	err := s.db.QueryRowContext(ctx, query, providerType, accountID).Scan(
		&inst.ID,
		&inst.Name,
		&inst.ProviderType,
		&inst.AuthMethod,
		&secretBlob,
		&oauthTokenType,
		&oauthExpiry,
		pq.Array(&oauthScopes),
		&accountIDVal,
		&inst.CreatedAt,
		&inst.UpdatedAt,
		&lastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error for this method
	}
	if err != nil {
		return nil, fmt.Errorf("get installation by account: %w", err)
	}

	// Decrypt secrets if present
	if len(secretBlob) > 0 {
		inst.Secrets = &domain.InstallationSecrets{}
		if err := s.encryptor.Decrypt(secretBlob, inst.Secrets); err != nil {
			return nil, fmt.Errorf("decrypt secrets: %w", err)
		}
	}

	inst.OAuthTokenType = oauthTokenType.String
	if oauthExpiry.Valid {
		inst.OAuthExpiry = &oauthExpiry.Time
	}
	inst.OAuthScopes = oauthScopes
	inst.AccountID = accountIDVal.String
	if lastUsedAt.Valid {
		inst.LastUsedAt = &lastUsedAt.Time
	}

	return &inst, nil
}

// UpdateSecrets updates the encrypted secrets and OAuth metadata.
func (s *InstallationStore) UpdateSecrets(ctx context.Context, id string, secrets *domain.InstallationSecrets, expiry *time.Time) error {
	var secretBlob []byte
	if secrets != nil {
		var err error
		secretBlob, err = s.encryptor.Encrypt(secrets)
		if err != nil {
			return fmt.Errorf("encrypt secrets: %w", err)
		}
	}

	query := `
		UPDATE connector_installations
		SET secret_blob = $1, oauth_expiry = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := s.db.ExecContext(ctx, query, secretBlob, nullTime(expiry), time.Now(), id)
	if err != nil {
		return fmt.Errorf("update secrets: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// UpdateLastUsed updates the last_used_at timestamp.
func (s *InstallationStore) UpdateLastUsed(ctx context.Context, id string) error {
	query := `
		UPDATE connector_installations
		SET last_used_at = $1
		WHERE id = $2
	`

	result, err := s.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("update last used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Helper functions for nullable values

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
