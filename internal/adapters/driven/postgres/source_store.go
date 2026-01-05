package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Verify interface compliance
var _ driven.SourceStore = (*SourceStore)(nil)

// SourceStore implements driven.SourceStore using PostgreSQL
type SourceStore struct {
	db *DB
}

// NewSourceStore creates a new SourceStore
func NewSourceStore(db *DB) *SourceStore {
	return &SourceStore{db: db}
}

// Save creates or updates a source
func (s *SourceStore) Save(ctx context.Context, source *domain.Source) error {
	configJSON, err := json.Marshal(source.Config)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO sources (id, name, provider_type, config, enabled, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			provider_type = EXCLUDED.provider_type,
			config = EXCLUDED.config,
			enabled = EXCLUDED.enabled,
			updated_at = EXCLUDED.updated_at
	`

	_, err = s.db.ExecContext(ctx, query,
		source.ID,
		source.Name,
		string(source.ProviderType),
		configJSON,
		source.Enabled,
		source.CreatedAt,
		source.UpdatedAt,
		source.CreatedBy,
	)
	return err
}

// Get retrieves a source by ID
func (s *SourceStore) Get(ctx context.Context, id string) (*domain.Source, error) {
	query := `
		SELECT id, name, provider_type, config, enabled, created_at, updated_at, created_by
		FROM sources
		WHERE id = $1
	`

	var source domain.Source
	var configJSON []byte
	var createdBy sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&source.ID,
		&source.Name,
		&source.ProviderType,
		&configJSON,
		&source.Enabled,
		&source.CreatedAt,
		&source.UpdatedAt,
		&createdBy,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(configJSON, &source.Config); err != nil {
		return nil, err
	}
	source.CreatedBy = createdBy.String

	return &source, nil
}

// GetByName retrieves a source by name
func (s *SourceStore) GetByName(ctx context.Context, name string) (*domain.Source, error) {
	query := `
		SELECT id, name, provider_type, config, enabled, created_at, updated_at, created_by
		FROM sources
		WHERE name = $1
	`

	var source domain.Source
	var configJSON []byte
	var createdBy sql.NullString

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&source.ID,
		&source.Name,
		&source.ProviderType,
		&configJSON,
		&source.Enabled,
		&source.CreatedAt,
		&source.UpdatedAt,
		&createdBy,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(configJSON, &source.Config); err != nil {
		return nil, err
	}
	source.CreatedBy = createdBy.String

	return &source, nil
}

// List retrieves all sources
func (s *SourceStore) List(ctx context.Context) ([]*domain.Source, error) {
	query := `
		SELECT id, name, provider_type, config, enabled, created_at, updated_at, created_by
		FROM sources
		ORDER BY created_at DESC
	`

	return s.querySources(ctx, query)
}

// ListEnabled retrieves all enabled sources
func (s *SourceStore) ListEnabled(ctx context.Context) ([]*domain.Source, error) {
	query := `
		SELECT id, name, provider_type, config, enabled, created_at, updated_at, created_by
		FROM sources
		WHERE enabled = true
		ORDER BY created_at DESC
	`

	return s.querySources(ctx, query)
}

func (s *SourceStore) querySources(ctx context.Context, query string, args ...interface{}) ([]*domain.Source, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*domain.Source
	for rows.Next() {
		var source domain.Source
		var configJSON []byte
		var createdBy sql.NullString

		err := rows.Scan(
			&source.ID,
			&source.Name,
			&source.ProviderType,
			&configJSON,
			&source.Enabled,
			&source.CreatedAt,
			&source.UpdatedAt,
			&createdBy,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(configJSON, &source.Config); err != nil {
			return nil, err
		}
		source.CreatedBy = createdBy.String
		sources = append(sources, &source)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sources, nil
}

// Delete deletes a source
func (s *SourceStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sources WHERE id = $1`
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

// SetEnabled updates the enabled status
func (s *SourceStore) SetEnabled(ctx context.Context, id string, enabled bool) error {
	query := `UPDATE sources SET enabled = $1, updated_at = $2 WHERE id = $3`
	result, err := s.db.ExecContext(ctx, query, enabled, time.Now(), id)
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
