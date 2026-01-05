package vespa

import (
	"testing"
)

func TestValidateEndpoint(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		want      string
		wantError bool
	}{
		{
			name:      "valid http endpoint",
			endpoint:  "http://localhost:19071",
			want:      "http://localhost:19071",
			wantError: false,
		},
		{
			name:      "valid https endpoint",
			endpoint:  "https://vespa.example.com:19071",
			want:      "https://vespa.example.com:19071",
			wantError: false,
		},
		{
			name:      "strips trailing slash",
			endpoint:  "http://localhost:19071/",
			want:      "http://localhost:19071",
			wantError: false,
		},
		{
			name:      "rejects empty string",
			endpoint:  "",
			wantError: true,
		},
		{
			name:      "rejects file scheme",
			endpoint:  "file:///etc/passwd",
			wantError: true,
		},
		{
			name:      "rejects ftp scheme",
			endpoint:  "ftp://example.com",
			wantError: true,
		},
		{
			name:      "rejects no scheme",
			endpoint:  "localhost:19071",
			wantError: true,
		},
		{
			name:      "rejects javascript scheme",
			endpoint:  "javascript:alert(1)",
			wantError: true,
		},
		{
			name:      "rejects data scheme",
			endpoint:  "data:text/html,<script>alert(1)</script>",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateEndpoint(tt.endpoint)
			if tt.wantError {
				if err == nil {
					t.Errorf("validateEndpoint(%q) expected error, got nil", tt.endpoint)
				}
				return
			}
			if err != nil {
				t.Errorf("validateEndpoint(%q) unexpected error: %v", tt.endpoint, err)
				return
			}
			if got != tt.want {
				t.Errorf("validateEndpoint(%q) = %q, want %q", tt.endpoint, got, tt.want)
			}
		})
	}
}
