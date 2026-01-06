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

// Ensure ProviderConfigStore implements the interface.
var _ driven.ProviderConfigStore = (*ProviderConfigStore)(nil)

// ProviderConfigStore implements driven.ProviderConfigStore using PostgreSQL.
// One config per provider type - multiple installations can use the same config.
type ProviderConfigStore struct {
	db        *sql.DB
	encryptor *SecretEncryptor
}

// NewProviderConfigStore creates a new PostgreSQL-backed provider config store.
func NewProviderConfigStore(db *sql.DB, encryptor *SecretEncryptor) *ProviderConfigStore {
	return &ProviderConfigStore{
		db:        db,
		encryptor: encryptor,
	}
}

// Save stores or updates a provider config (upsert).
func (s *ProviderConfigStore) Save(ctx context.Context, cfg *domain.ProviderConfig) error {
	// Encrypt secrets if present
	var secretBlob []byte
	if cfg.Secrets != nil {
		var err error
		secretBlob, err = s.encryptor.Encrypt(cfg.Secrets)
		if err != nil {
			return fmt.Errorf("encrypt secrets: %w", err)
		}
	}

	query := `
		INSERT INTO provider_configs (
			provider_type, secret_blob, auth_url, token_url, scopes,
			redirect_uri, enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (provider_type) DO UPDATE SET
			secret_blob = EXCLUDED.secret_blob,
			auth_url = EXCLUDED.auth_url,
			token_url = EXCLUDED.token_url,
			scopes = EXCLUDED.scopes,
			redirect_uri = EXCLUDED.redirect_uri,
			enabled = EXCLUDED.enabled,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, query,
		cfg.ProviderType,
		secretBlob,
		nullString(cfg.AuthURL),
		nullString(cfg.TokenURL),
		pq.Array(cfg.Scopes),
		nullString(cfg.RedirectURI),
		cfg.Enabled,
		cfg.CreatedAt,
		cfg.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("save provider config: %w", err)
	}

	return nil
}

// Get retrieves a provider config by type with decrypted secrets.
func (s *ProviderConfigStore) Get(ctx context.Context, providerType domain.ProviderType) (*domain.ProviderConfig, error) {
	query := `
		SELECT provider_type, secret_blob, auth_url, token_url, scopes,
			   redirect_uri, enabled, created_at, updated_at
		FROM provider_configs
		WHERE provider_type = $1
	`

	var cfg domain.ProviderConfig
	var secretBlob []byte
	var authURL, tokenURL, redirectURI sql.NullString
	var scopes []string

	err := s.db.QueryRowContext(ctx, query, providerType).Scan(
		&cfg.ProviderType,
		&secretBlob,
		&authURL,
		&tokenURL,
		pq.Array(&scopes),
		&redirectURI,
		&cfg.Enabled,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // Not found returns nil, not error
	}
	if err != nil {
		return nil, fmt.Errorf("get provider config: %w", err)
	}

	// Decrypt secrets if present
	if len(secretBlob) > 0 {
		cfg.Secrets = &domain.ProviderSecrets{}
		if err := s.encryptor.Decrypt(secretBlob, cfg.Secrets); err != nil {
			return nil, fmt.Errorf("decrypt secrets: %w", err)
		}
	}

	cfg.AuthURL = authURL.String
	cfg.TokenURL = tokenURL.String
	cfg.Scopes = scopes
	cfg.RedirectURI = redirectURI.String

	return &cfg, nil
}

// List retrieves all provider configs as summaries (no secrets).
func (s *ProviderConfigStore) List(ctx context.Context) ([]*domain.ProviderConfigSummary, error) {
	query := `
		SELECT provider_type, enabled, secret_blob IS NOT NULL AND LENGTH(secret_blob) > 0,
			   created_at, updated_at
		FROM provider_configs
		ORDER BY provider_type
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list provider configs: %w", err)
	}
	defer rows.Close()

	var summaries []*domain.ProviderConfigSummary
	for rows.Next() {
		var summary domain.ProviderConfigSummary

		if err := rows.Scan(
			&summary.ProviderType,
			&summary.Enabled,
			&summary.HasSecrets,
			&summary.CreatedAt,
			&summary.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan provider config: %w", err)
		}

		summaries = append(summaries, &summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider configs: %w", err)
	}

	return summaries, nil
}

// Delete removes a provider config by type.
func (s *ProviderConfigStore) Delete(ctx context.Context, providerType domain.ProviderType) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM provider_configs WHERE provider_type = $1", providerType)
	if err != nil {
		return fmt.Errorf("delete provider config: %w", err)
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

// GetEnabled retrieves all enabled provider configs with secrets.
func (s *ProviderConfigStore) GetEnabled(ctx context.Context) ([]*domain.ProviderConfig, error) {
	query := `
		SELECT provider_type, secret_blob, auth_url, token_url, scopes,
			   redirect_uri, enabled, created_at, updated_at
		FROM provider_configs
		WHERE enabled = true
		ORDER BY provider_type
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list enabled provider configs: %w", err)
	}
	defer rows.Close()

	var configs []*domain.ProviderConfig
	for rows.Next() {
		var cfg domain.ProviderConfig
		var secretBlob []byte
		var authURL, tokenURL, redirectURI sql.NullString
		var scopes []string

		if err := rows.Scan(
			&cfg.ProviderType,
			&secretBlob,
			&authURL,
			&tokenURL,
			pq.Array(&scopes),
			&redirectURI,
			&cfg.Enabled,
			&cfg.CreatedAt,
			&cfg.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan provider config: %w", err)
		}

		// Decrypt secrets if present
		if len(secretBlob) > 0 {
			cfg.Secrets = &domain.ProviderSecrets{}
			if err := s.encryptor.Decrypt(secretBlob, cfg.Secrets); err != nil {
				return nil, fmt.Errorf("decrypt secrets: %w", err)
			}
		}

		cfg.AuthURL = authURL.String
		cfg.TokenURL = tokenURL.String
		cfg.Scopes = scopes
		cfg.RedirectURI = redirectURI.String

		configs = append(configs, &cfg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider configs: %w", err)
	}

	return configs, nil
}
