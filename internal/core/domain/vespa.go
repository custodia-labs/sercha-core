package domain

import "time"

// VespaSchemaMode represents the current schema deployment mode
type VespaSchemaMode string

const (
	// VespacSchemaModeNone indicates no schema is deployed
	VespacSchemaModeNone VespaSchemaMode = ""

	// VespacSchemaModeBM25 indicates BM25-only schema (no embeddings)
	VespacSchemaModeBM25 VespaSchemaMode = "bm25"

	// VespacSchemaModeHybrid indicates hybrid schema (BM25 + embeddings)
	VespacSchemaModeHybrid VespaSchemaMode = "hybrid"
)

// VespaConfig holds Vespa connection and schema state for a team
type VespaConfig struct {
	TeamID string `json:"team_id"`

	// Connection settings
	Endpoint  string `json:"endpoint"`
	Connected bool   `json:"connected"`
	DevMode   bool   `json:"dev_mode"` // true = we deployed services.xml, false = we only added schema

	// Schema state
	SchemaMode        VespaSchemaMode `json:"schema_mode"`
	EmbeddingDim      int             `json:"embedding_dim,omitempty"`
	EmbeddingProvider AIProvider      `json:"embedding_provider,omitempty"`
	SchemaVersion     string          `json:"schema_version"`

	// Cluster info (populated in production mode)
	ClusterInfo *VespaClusterInfo `json:"cluster_info,omitempty"`

	// Timestamps
	ConnectedAt time.Time `json:"connected_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IsConnected returns true if Vespa is connected and schema deployed
func (c *VespaConfig) IsConnected() bool {
	return c.Connected && c.SchemaMode != VespacSchemaModeNone
}

// HasEmbeddings returns true if the schema supports embeddings
func (c *VespaConfig) HasEmbeddings() bool {
	return c.SchemaMode == VespacSchemaModeHybrid && c.EmbeddingDim > 0
}

// CanUpgradeToHybrid returns true if schema can be upgraded from BM25 to hybrid
func (c *VespaConfig) CanUpgradeToHybrid() bool {
	return c.SchemaMode == VespacSchemaModeBM25
}

// DefaultVespaConfig returns a default unconfigured Vespa config
func DefaultVespaConfig(teamID string) *VespaConfig {
	return &VespaConfig{
		TeamID:    teamID,
		Endpoint:  "http://vespa:19071",
		Connected: false,
		UpdatedAt: time.Now(),
	}
}

// VespaDeployResult represents the result of a schema deployment
type VespaDeployResult struct {
	Success       bool            `json:"success"`
	SchemaMode    VespaSchemaMode `json:"schema_mode"`
	EmbeddingDim  int             `json:"embedding_dim,omitempty"`
	SchemaVersion string          `json:"schema_version"`
	Upgraded      bool            `json:"upgraded"`
	Message       string          `json:"message,omitempty"`
}

// VespaClusterInfo represents parsed information about a Vespa cluster
type VespaClusterInfo struct {
	// Raw XML content (for persistence and future reference)
	ServicesXML string `json:"services_xml,omitempty"`
	HostsXML    string `json:"hosts_xml,omitempty"`

	// Parsed cluster information
	ContentClusters   []VespaContentCluster   `json:"content_clusters,omitempty"`
	ContainerClusters []VespaContainerCluster `json:"container_clusters,omitempty"`
	Hosts             []VespaHost             `json:"hosts,omitempty"`
	Schemas           []string                `json:"schemas,omitempty"`

	// Our schema status
	OurSchemaDeployed bool `json:"our_schema_deployed"`
}

// VespaContentCluster represents a Vespa content cluster
type VespaContentCluster struct {
	ID         string   `json:"id"`
	Redundancy int      `json:"redundancy,omitempty"`
	Nodes      []string `json:"nodes,omitempty"`
	Documents  []string `json:"documents,omitempty"`
}

// VespaContainerCluster represents a Vespa container cluster
type VespaContainerCluster struct {
	ID       string   `json:"id"`
	Port     int      `json:"port,omitempty"`
	Nodes    []string `json:"nodes,omitempty"`
	HasFeed  bool     `json:"has_feed"`
	HasQuery bool     `json:"has_query"`
}

// VespaHost represents a host in the Vespa cluster
type VespaHost struct {
	Alias    string `json:"alias"`
	Hostname string `json:"hostname"`
}
