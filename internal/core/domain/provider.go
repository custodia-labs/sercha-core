package domain

// ProviderType identifies a data source provider
type ProviderType string

const (
	// Code repositories
	ProviderTypeGitHub    ProviderType = "github"
	ProviderTypeGitLab    ProviderType = "gitlab"
	ProviderTypeBitbucket ProviderType = "bitbucket"

	// Communication
	ProviderTypeSlack   ProviderType = "slack"
	ProviderTypeDiscord ProviderType = "discord"
	ProviderTypeMSTeams ProviderType = "msteams"

	// Documentation
	ProviderTypeNotion     ProviderType = "notion"
	ProviderTypeConfluence ProviderType = "confluence"
	ProviderTypeGoogleDocs ProviderType = "google_docs"

	// Project management
	ProviderTypeJira   ProviderType = "jira"
	ProviderTypeLinear ProviderType = "linear"

	// File storage
	ProviderTypeGoogleDrive ProviderType = "google_drive"
	ProviderTypeDropbox     ProviderType = "dropbox"
	ProviderTypeOneDrive    ProviderType = "onedrive"
	ProviderTypeS3          ProviderType = "s3"

	// Other
	ProviderTypeZendesk  ProviderType = "zendesk"
	ProviderTypeIntercom ProviderType = "intercom"
)

// AuthProvider holds OAuth configuration for a provider
type AuthProvider struct {
	Type         ProviderType `json:"type"`
	Name         string       `json:"name"`         // Display name
	AuthURL      string       `json:"auth_url"`     // OAuth authorization URL
	TokenURL     string       `json:"token_url"`    // OAuth token URL
	Scopes       []string     `json:"scopes"`       // Required OAuth scopes
	ClientID     string       `json:"client_id"`    // OAuth client ID (public)
	ClientSecret string       `json:"-"`            // OAuth client secret (never serialize)
	RedirectURL  string       `json:"redirect_url"` // OAuth callback URL
}

// ProviderInfo provides metadata about a provider
type ProviderInfo struct {
	Type        ProviderType `json:"type"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IconURL     string       `json:"icon_url,omitempty"`
	AuthMethods []AuthMethod `json:"auth_methods"`
	DocsURL     string       `json:"docs_url,omitempty"`
	Available   bool         `json:"available"` // Whether connector is implemented
}

// CoreProviders returns the 11 providers for Sercha Core
func CoreProviders() []ProviderType {
	return []ProviderType{
		ProviderTypeGitHub,
		ProviderTypeGitLab,
		ProviderTypeSlack,
		ProviderTypeNotion,
		ProviderTypeConfluence,
		ProviderTypeJira,
		ProviderTypeGoogleDrive,
		ProviderTypeGoogleDocs,
		ProviderTypeLinear,
		ProviderTypeDropbox,
		ProviderTypeS3,
	}
}
