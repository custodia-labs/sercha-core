package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

// Verify interface compliance
var _ driven.VespaConfigStore = (*VespaConfigStore)(nil)

// VespaConfigStore implements driven.VespaConfigStore using PostgreSQL
type VespaConfigStore struct {
	db *DB
}

// NewVespaConfigStore creates a new VespaConfigStore
func NewVespaConfigStore(db *DB) *VespaConfigStore {
	return &VespaConfigStore{db: db}
}

// GetVespaConfig retrieves Vespa config for a team
func (s *VespaConfigStore) GetVespaConfig(ctx context.Context, teamID string) (*domain.VespaConfig, error) {
	query := `
		SELECT team_id, endpoint, connected, schema_mode, embedding_dim,
			   embedding_provider, schema_version, connected_at, updated_at
		FROM vespa_config
		WHERE team_id = $1
	`

	var config domain.VespaConfig
	var endpoint, schemaMode, embProvider, schemaVersion sql.NullString
	var embeddingDim sql.NullInt32
	var connectedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, teamID).Scan(
		&config.TeamID,
		&endpoint,
		&config.Connected,
		&schemaMode,
		&embeddingDim,
		&embProvider,
		&schemaVersion,
		&connectedAt,
		&config.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	config.Endpoint = endpoint.String
	config.SchemaMode = domain.VespaSchemaMode(schemaMode.String)
	if embeddingDim.Valid {
		config.EmbeddingDim = int(embeddingDim.Int32)
	}
	if embProvider.Valid {
		config.EmbeddingProvider = domain.AIProvider(embProvider.String)
	}
	config.SchemaVersion = schemaVersion.String
	if connectedAt.Valid {
		config.ConnectedAt = connectedAt.Time
	}

	return &config, nil
}

// SaveVespaConfig persists Vespa config
func (s *VespaConfigStore) SaveVespaConfig(ctx context.Context, config *domain.VespaConfig) error {
	query := `
		INSERT INTO vespa_config (team_id, endpoint, connected, schema_mode, embedding_dim,
								  embedding_provider, schema_version, connected_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (team_id) DO UPDATE SET
			endpoint = EXCLUDED.endpoint,
			connected = EXCLUDED.connected,
			schema_mode = EXCLUDED.schema_mode,
			embedding_dim = EXCLUDED.embedding_dim,
			embedding_provider = EXCLUDED.embedding_provider,
			schema_version = EXCLUDED.schema_version,
			connected_at = EXCLUDED.connected_at,
			updated_at = EXCLUDED.updated_at
	`

	config.UpdatedAt = time.Now()

	var connectedAt *time.Time
	if !config.ConnectedAt.IsZero() {
		connectedAt = &config.ConnectedAt
	}

	var embeddingDim *int
	if config.EmbeddingDim > 0 {
		embeddingDim = &config.EmbeddingDim
	}

	_, err := s.db.ExecContext(ctx, query,
		config.TeamID,
		config.Endpoint,
		config.Connected,
		string(config.SchemaMode),
		embeddingDim,
		string(config.EmbeddingProvider),
		config.SchemaVersion,
		connectedAt,
		config.UpdatedAt,
	)
	return err
}
