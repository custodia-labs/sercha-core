package vespa

import (
	"archive/zip"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/custodia-labs/sercha-core/internal/core/domain"
	"github.com/custodia-labs/sercha-core/internal/core/ports/driven"
)

//go:embed schemas/services.xml schemas/chunk_bm25.sd schemas/chunk_hybrid.sd.tmpl
var schemaFS embed.FS

// Verify interface compliance
var _ driven.VespaDeployer = (*Deployer)(nil)

// Deployer implements driven.VespaDeployer
type Deployer struct {
	httpClient *http.Client
}

// NewDeployer creates a new Vespa deployer
func NewDeployer() *Deployer {
	return &Deployer{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Deploy deploys the Vespa application package
// If existingPkg is provided, merges our schema into it instead of using embedded services.xml
func (d *Deployer) Deploy(ctx context.Context, endpoint string, embeddingDim *int, existingPkg *driven.AppPackage) (*domain.VespaDeployResult, error) {
	endpoint = strings.TrimSuffix(endpoint, "/")

	// Determine schema mode
	mode := domain.VespacSchemaModeBM25
	if embeddingDim != nil && *embeddingDim > 0 {
		mode = domain.VespacSchemaModeHybrid
	}

	// Generate our schema content
	schemaContent, err := d.generateSchema(mode, embeddingDim)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}

	var zipData []byte

	if existingPkg != nil {
		// Production mode: merge our schema into existing package
		zipData, err = d.createMergedAppPackage(existingPkg, schemaContent)
		if err != nil {
			return nil, fmt.Errorf("failed to create merged app package: %w", err)
		}
	} else {
		// Dev mode: use embedded services.xml
		servicesContent, err := schemaFS.ReadFile("schemas/services.xml")
		if err != nil {
			return nil, fmt.Errorf("failed to read services.xml: %w", err)
		}
		zipData, err = d.createDevAppPackage(servicesContent, schemaContent)
		if err != nil {
			return nil, fmt.Errorf("failed to create app package: %w", err)
		}
	}

	// Deploy to Vespa
	deployURL := fmt.Sprintf("%s/application/v2/tenant/default/prepareandactivate", endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deployURL, bytes.NewReader(zipData))
	if err != nil {
		return nil, fmt.Errorf("failed to create deploy request: %w", err)
	}
	req.Header.Set("Content-Type", "application/zip")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("deployment request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("deployment failed with status %s: %s", resp.Status, string(body))
	}

	// Generate version string
	version := fmt.Sprintf("v1-%s", mode)
	if embeddingDim != nil {
		version = fmt.Sprintf("v1-%s-dim%d", mode, *embeddingDim)
	}

	return &domain.VespaDeployResult{
		Success:       true,
		SchemaMode:    mode,
		EmbeddingDim:  safeDeref(embeddingDim),
		SchemaVersion: version,
		Upgraded:      false,
		Message:       fmt.Sprintf("Deployed %s schema", mode),
	}, nil
}

// FetchAppPackage retrieves the currently deployed application package from Vespa
func (d *Deployer) FetchAppPackage(ctx context.Context, endpoint string) (*driven.AppPackage, error) {
	endpoint = strings.TrimSuffix(endpoint, "/")
	baseURL := fmt.Sprintf("%s/application/v2/tenant/default/application/default/environment/default/region/default/instance/default/content", endpoint)

	// Check if application exists first
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/", nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch app package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil // No application deployed
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list app content: %s - %s", resp.Status, string(body))
	}

	pkg := &driven.AppPackage{
		Schemas: make(map[string]string),
	}

	// Fetch services.xml
	servicesXML, err := d.fetchContent(ctx, baseURL+"/services.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch services.xml: %w", err)
	}
	pkg.ServicesXML = servicesXML

	// Fetch hosts.xml (optional)
	hostsXML, err := d.fetchContent(ctx, baseURL+"/hosts.xml")
	if err == nil {
		pkg.HostsXML = hostsXML
	}

	// List and fetch schemas
	schemaList, err := d.fetchContent(ctx, baseURL+"/schemas/")
	if err == nil && schemaList != "" {
		// Parse the directory listing (JSON array of URLs)
		var schemaURLs []string
		if err := json.Unmarshal([]byte(schemaList), &schemaURLs); err == nil {
			for _, schemaURL := range schemaURLs {
				// Extract filename from URL
				parts := strings.Split(schemaURL, "/")
				if len(parts) > 0 {
					filename := parts[len(parts)-1]
					if strings.HasSuffix(filename, ".sd") {
						content, err := d.fetchContent(ctx, baseURL+"/schemas/"+filename)
						if err == nil {
							pkg.Schemas[filename] = content
						}
					}
				}
			}
		}
	}

	// Parse cluster info
	pkg.ClusterInfo = d.parseClusterInfo(pkg)

	return pkg, nil
}

// fetchContent fetches content from a Vespa content API URL
func (d *Deployer) fetchContent(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", fmt.Errorf("not found")
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("fetch failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// parseClusterInfo parses services.xml and hosts.xml into structured cluster info
func (d *Deployer) parseClusterInfo(pkg *driven.AppPackage) *domain.VespaClusterInfo {
	info := &domain.VespaClusterInfo{
		ServicesXML: pkg.ServicesXML,
		HostsXML:    pkg.HostsXML,
	}

	// Parse services.xml
	if pkg.ServicesXML != "" {
		d.parseServicesXML(pkg.ServicesXML, info)
	}

	// Parse hosts.xml
	if pkg.HostsXML != "" {
		d.parseHostsXML(pkg.HostsXML, info)
	}

	// Collect schema names
	for filename := range pkg.Schemas {
		schemaName := strings.TrimSuffix(filename, ".sd")
		info.Schemas = append(info.Schemas, schemaName)
	}

	// Check if our schema is deployed
	_, hasChunk := pkg.Schemas["chunk.sd"]
	info.OurSchemaDeployed = hasChunk

	return info
}

// XML structures for parsing services.xml
type servicesXML struct {
	XMLName    xml.Name         `xml:"services"`
	Content    []contentXML     `xml:"content"`
	Containers []containerXML   `xml:"container"`
}

type contentXML struct {
	ID         string       `xml:"id,attr"`
	Redundancy string       `xml:"redundancy"`
	MinRedundancy string    `xml:"min-redundancy"`
	Documents  documentsXML `xml:"documents"`
	Nodes      nodesXML     `xml:"nodes"`
}

type documentsXML struct {
	Document []documentXML `xml:"document"`
}

type documentXML struct {
	Type string `xml:"type,attr"`
	Mode string `xml:"mode,attr"`
}

type containerXML struct {
	ID          string     `xml:"id,attr"`
	DocumentAPI *struct{}  `xml:"document-api"`
	Search      *struct{}  `xml:"search"`
	HTTP        *httpXML   `xml:"http"`
	Nodes       nodesXML   `xml:"nodes"`
}

type httpXML struct {
	Server []serverXML `xml:"server"`
}

type serverXML struct {
	ID   string `xml:"id,attr"`
	Port int    `xml:"port,attr"`
}

type nodesXML struct {
	Node []nodeXML `xml:"node"`
}

type nodeXML struct {
	HostAlias       string `xml:"hostalias,attr"`
	DistributionKey string `xml:"distribution-key,attr"`
}

// XML structures for parsing hosts.xml
type hostsXML struct {
	XMLName xml.Name  `xml:"hosts"`
	Hosts   []hostXML `xml:"host"`
}

type hostXML struct {
	Name  string `xml:"name,attr"`
	Alias string `xml:"alias"`
}

func (d *Deployer) parseServicesXML(content string, info *domain.VespaClusterInfo) {
	var services servicesXML
	if err := xml.Unmarshal([]byte(content), &services); err != nil {
		return
	}

	// Parse content clusters
	for _, c := range services.Content {
		cluster := domain.VespaContentCluster{
			ID: c.ID,
		}

		// Parse redundancy
		if c.Redundancy != "" {
			cluster.Redundancy, _ = strconv.Atoi(c.Redundancy)
		} else if c.MinRedundancy != "" {
			cluster.Redundancy, _ = strconv.Atoi(c.MinRedundancy)
		}

		// Parse nodes
		for _, n := range c.Nodes.Node {
			cluster.Nodes = append(cluster.Nodes, n.HostAlias)
		}

		// Parse document types
		for _, doc := range c.Documents.Document {
			cluster.Documents = append(cluster.Documents, doc.Type)
		}

		info.ContentClusters = append(info.ContentClusters, cluster)
	}

	// Parse container clusters
	for _, c := range services.Containers {
		cluster := domain.VespaContainerCluster{
			ID:       c.ID,
			HasFeed:  c.DocumentAPI != nil,
			HasQuery: c.Search != nil,
		}

		// Parse port
		if c.HTTP != nil && len(c.HTTP.Server) > 0 {
			cluster.Port = c.HTTP.Server[0].Port
		}

		// Parse nodes
		for _, n := range c.Nodes.Node {
			cluster.Nodes = append(cluster.Nodes, n.HostAlias)
		}

		info.ContainerClusters = append(info.ContainerClusters, cluster)
	}
}

func (d *Deployer) parseHostsXML(content string, info *domain.VespaClusterInfo) {
	var hosts hostsXML
	if err := xml.Unmarshal([]byte(content), &hosts); err != nil {
		return
	}

	for _, h := range hosts.Hosts {
		info.Hosts = append(info.Hosts, domain.VespaHost{
			Alias:    h.Alias,
			Hostname: h.Name,
		})
	}
}

// GetSchemaInfo retrieves information about the currently deployed schema
func (d *Deployer) GetSchemaInfo(ctx context.Context, endpoint string) (*driven.SchemaInfo, error) {
	endpoint = strings.TrimSuffix(endpoint, "/")

	// Try to get application status
	statusURL := fmt.Sprintf("%s/application/v2/tenant/default/application/default", endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return &driven.SchemaInfo{Deployed: false}, nil
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get schema info: %s - %s", resp.Status, string(body))
	}

	return &driven.SchemaInfo{
		Deployed: true,
	}, nil
}

// HealthCheck verifies the Vespa cluster is healthy
func (d *Deployer) HealthCheck(ctx context.Context, endpoint string) error {
	endpoint = strings.TrimSuffix(endpoint, "/")
	healthURL := fmt.Sprintf("%s/state/v1/health", endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unhealthy: %s - %s", resp.Status, string(body))
	}

	return nil
}

// generateSchema generates the appropriate schema based on mode
func (d *Deployer) generateSchema(mode domain.VespaSchemaMode, embeddingDim *int) ([]byte, error) {
	switch mode {
	case domain.VespacSchemaModeBM25:
		return schemaFS.ReadFile("schemas/chunk_bm25.sd")

	case domain.VespacSchemaModeHybrid:
		tmplContent, err := schemaFS.ReadFile("schemas/chunk_hybrid.sd.tmpl")
		if err != nil {
			return nil, err
		}

		tmpl, err := template.New("schema").Parse(string(tmplContent))
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		data := struct {
			EmbeddingDim int
		}{
			EmbeddingDim: safeDeref(embeddingDim),
		}

		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, err
		}

		return buf.Bytes(), nil

	default:
		return nil, fmt.Errorf("unknown schema mode: %s", mode)
	}
}

// createDevAppPackage creates a Vespa application package zip for dev mode
func (d *Deployer) createDevAppPackage(services, schema []byte) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Add services.xml
	servicesWriter, err := zipWriter.Create("services.xml")
	if err != nil {
		return nil, err
	}
	if _, err := servicesWriter.Write(services); err != nil {
		return nil, err
	}

	// Add schema file
	schemaWriter, err := zipWriter.Create("schemas/chunk.sd")
	if err != nil {
		return nil, err
	}
	if _, err := schemaWriter.Write(schema); err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// createMergedAppPackage creates a Vespa application package by merging our schema into existing package
func (d *Deployer) createMergedAppPackage(existingPkg *driven.AppPackage, ourSchema []byte) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Modify services.xml to add our document type to all content clusters
	modifiedServices := d.addChunkDocumentType(existingPkg.ServicesXML)

	// Add services.xml
	servicesWriter, err := zipWriter.Create("services.xml")
	if err != nil {
		return nil, err
	}
	if _, err := servicesWriter.Write([]byte(modifiedServices)); err != nil {
		return nil, err
	}

	// Add hosts.xml if exists
	if existingPkg.HostsXML != "" {
		hostsWriter, err := zipWriter.Create("hosts.xml")
		if err != nil {
			return nil, err
		}
		if _, err := hostsWriter.Write([]byte(existingPkg.HostsXML)); err != nil {
			return nil, err
		}
	}

	// Add existing schemas
	for filename, content := range existingPkg.Schemas {
		if filename == "chunk.sd" {
			continue // We'll add our version
		}
		schemaWriter, err := zipWriter.Create("schemas/" + filename)
		if err != nil {
			return nil, err
		}
		if _, err := schemaWriter.Write([]byte(content)); err != nil {
			return nil, err
		}
	}

	// Add our chunk schema
	chunkWriter, err := zipWriter.Create("schemas/chunk.sd")
	if err != nil {
		return nil, err
	}
	if _, err := chunkWriter.Write(ourSchema); err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// addChunkDocumentType adds <document type="chunk" mode="index"/> to all content clusters
func (d *Deployer) addChunkDocumentType(servicesXML string) string {
	// Use regex to find <documents> sections and add chunk if not present
	if strings.Contains(servicesXML, `type="chunk"`) {
		return servicesXML // Already has chunk document type
	}

	// Find all <documents> tags and add chunk document type
	// This is a simple approach - for complex cases might need proper XML manipulation
	re := regexp.MustCompile(`(<documents[^>]*>)`)
	result := re.ReplaceAllString(servicesXML, `$1
            <document type="chunk" mode="index"/>`)

	return result
}

func safeDeref(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}
