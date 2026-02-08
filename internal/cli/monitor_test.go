package cli

import "testing"

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"plain url", "http://localhost:9090", "http://localhost:9090"},
		{"with credentials", "http://admin:secret@localhost:9090", "http://REDACTED:REDACTED@localhost:9090"},
		{"user only", "http://admin@localhost:9090", "http://REDACTED:REDACTED@localhost:9090"},
		{"invalid url", "://bad", "[invalid URL]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeURL(tt.url)
			if got != tt.want {
				t.Errorf("sanitizeURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
	}{
		{"valid port", "8080", false},
		{"min port", "1", false},
		{"max port", "65535", false},
		{"zero", "0", true},
		{"negative", "-1", true},
		{"above range", "65536", true},
		{"non-numeric", "abc", true},
		{"empty", "", true},
		{"float", "80.5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePort(tt.port, "test-port")
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePort(%q) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePrometheusURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"http url", "http://localhost:9090", false},
		{"https url", "https://prom.example.com", false},
		{"ftp invalid", "ftp://localhost:9090", true},
		{"missing scheme", "localhost:9090", true},
		{"empty", "", true},
		{"missing host", "http://", true},
		{"with path", "http://localhost:9090/api/v1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePrometheusURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePrometheusURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
