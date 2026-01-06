package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
	"github.com/lib/pq"
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
		INSERT INTO sources (id, name, provider_type, config, enabled, created_at, updated_at, created_by, installation_id, selected_containers)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			provider_type = EXCLUDED.provider_type,
			config = EXCLUDED.config,
			enabled = EXCLUDED.enabled,
			updated_at = EXCLUDED.updated_at,
			installation_id = EXCLUDED.installation_id,
			selected_containers = EXCLUDED.selected_containers
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
		sql.NullString{String: source.InstallationID, Valid: source.InstallationID != ""},
		pq.Array(source.SelectedContainers),
	)
	return err
}

// Get retrieves a source by ID
func (s *SourceStore) Get(ctx context.Context, id string) (*domain.Source, error) {
	query := `
		SELECT id, name, provider_type, config, enabled, created_at, updated_at, created_by,
		       installation_id, selected_containers
		FROM sources
		WHERE id = $1
	`

	var source domain.Source
	var configJSON []byte
	var createdBy, installationID sql.NullString
	var selectedContainers []string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&source.ID,
		&source.Name,
		&source.ProviderType,
		&configJSON,
		&source.Enabled,
		&source.CreatedAt,
		&source.UpdatedAt,
		&createdBy,
		&installationID,
		pqArray(&selectedContainers),
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
	source.InstallationID = installationID.String
	source.SelectedContainers = selectedContainers

	return &source, nil
}

// GetByName retrieves a source by name
func (s *SourceStore) GetByName(ctx context.Context, name string) (*domain.Source, error) {
	query := `
		SELECT id, name, provider_type, config, enabled, created_at, updated_at, created_by,
		       installation_id, selected_containers
		FROM sources
		WHERE name = $1
	`

	var source domain.Source
	var configJSON []byte
	var createdBy, installationID sql.NullString
	var selectedContainers []string

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&source.ID,
		&source.Name,
		&source.ProviderType,
		&configJSON,
		&source.Enabled,
		&source.CreatedAt,
		&source.UpdatedAt,
		&createdBy,
		&installationID,
		pqArray(&selectedContainers),
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
	source.InstallationID = installationID.String
	source.SelectedContainers = selectedContainers

	return &source, nil
}

// List retrieves all sources
func (s *SourceStore) List(ctx context.Context) ([]*domain.Source, error) {
	query := `
		SELECT id, name, provider_type, config, enabled, created_at, updated_at, created_by,
		       installation_id, selected_containers
		FROM sources
		ORDER BY created_at DESC
	`

	return s.querySources(ctx, query)
}

// ListEnabled retrieves all enabled sources
func (s *SourceStore) ListEnabled(ctx context.Context) ([]*domain.Source, error) {
	query := `
		SELECT id, name, provider_type, config, enabled, created_at, updated_at, created_by,
		       installation_id, selected_containers
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
		var createdBy, installationID sql.NullString
		var selectedContainers []string

		err := rows.Scan(
			&source.ID,
			&source.Name,
			&source.ProviderType,
			&configJSON,
			&source.Enabled,
			&source.CreatedAt,
			&source.UpdatedAt,
			&createdBy,
			&installationID,
			pqArray(&selectedContainers),
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(configJSON, &source.Config); err != nil {
			return nil, err
		}
		source.CreatedBy = createdBy.String
		source.InstallationID = installationID.String
		source.SelectedContainers = selectedContainers
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

// CountByInstallation returns the number of sources using an installation
func (s *SourceStore) CountByInstallation(ctx context.Context, installationID string) (int, error) {
	query := `SELECT COUNT(*) FROM sources WHERE installation_id = $1`
	var count int
	err := s.db.QueryRowContext(ctx, query, installationID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ListByInstallation returns sources using an installation
func (s *SourceStore) ListByInstallation(ctx context.Context, installationID string) ([]*domain.Source, error) {
	query := `
		SELECT id, name, provider_type, config, enabled, created_at, updated_at, created_by,
		       installation_id, selected_containers
		FROM sources
		WHERE installation_id = $1
		ORDER BY created_at DESC
	`

	return s.querySourcesWithInstallation(ctx, query, installationID)
}

// UpdateSelection updates the selected containers for a source
func (s *SourceStore) UpdateSelection(ctx context.Context, id string, containers []string) error {
	query := `UPDATE sources SET selected_containers = $1, updated_at = $2 WHERE id = $3`
	result, err := s.db.ExecContext(ctx, query, pq.Array(containers), time.Now(), id)
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

// querySourcesWithInstallation is like querySources but includes installation fields
func (s *SourceStore) querySourcesWithInstallation(ctx context.Context, query string, args ...interface{}) ([]*domain.Source, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*domain.Source
	for rows.Next() {
		var source domain.Source
		var configJSON []byte
		var createdBy, installationID sql.NullString
		var selectedContainers []string

		err := rows.Scan(
			&source.ID,
			&source.Name,
			&source.ProviderType,
			&configJSON,
			&source.Enabled,
			&source.CreatedAt,
			&source.UpdatedAt,
			&createdBy,
			&installationID,
			pqArray(&selectedContainers),
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(configJSON, &source.Config); err != nil {
			return nil, err
		}
		source.CreatedBy = createdBy.String
		source.InstallationID = installationID.String
		source.SelectedContainers = selectedContainers
		sources = append(sources, &source)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sources, nil
}

// pqArray is a helper for scanning PostgreSQL arrays
func pqArray(arr *[]string) interface{} {
	return &pqStringArray{arr}
}

type pqStringArray struct {
	arr *[]string
}

func (a *pqStringArray) Scan(src interface{}) error {
	if src == nil {
		*a.arr = nil
		return nil
	}

	switch v := src.(type) {
	case []byte:
		return a.scanString(string(v))
	case string:
		return a.scanString(v)
	default:
		return nil
	}
}

func (a *pqStringArray) scanString(s string) error {
	// Handle PostgreSQL array format: {elem1,elem2,...}
	if s == "" || s == "{}" {
		*a.arr = nil
		return nil
	}

	// Remove braces
	if len(s) >= 2 && s[0] == '{' && s[len(s)-1] == '}' {
		s = s[1 : len(s)-1]
	}

	if s == "" {
		*a.arr = nil
		return nil
	}

	// Simple split by comma (doesn't handle quoted strings)
	// For proper handling, use lib/pq's pq.Array
	*a.arr = splitPgArray(s)
	return nil
}

func splitPgArray(s string) []string {
	var result []string
	var current string
	inQuote := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQuote = !inQuote
		case c == ',' && !inQuote:
			result = append(result, current)
			current = ""
		default:
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
