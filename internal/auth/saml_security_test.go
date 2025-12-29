package auth

import (
	"strings"
	"testing"
)

func TestValidateRelayState(t *testing.T) {
	tests := []struct {
		name         string
		relayState   string
		allowedHosts []string
		wantURL      string
		wantError    bool
	}{
		{
			name:         "empty relay state",
			relayState:   "",
			allowedHosts: []string{},
			wantURL:      "",
			wantError:    false,
		},
		{
			name:         "relative URL allowed without hosts",
			relayState:   "/dashboard",
			allowedHosts: []string{},
			wantURL:      "/dashboard",
			wantError:    false,
		},
		{
			name:         "relative URL with query params",
			relayState:   "/dashboard?tab=settings",
			allowedHosts: []string{},
			wantURL:      "/dashboard?tab=settings",
			wantError:    false,
		},
		{
			name:         "protocol-relative URL blocked",
			relayState:   "//evil.com/path",
			allowedHosts: []string{},
			wantURL:      "",
			wantError:    true,
		},
		{
			name:         "absolute URL blocked without allowed hosts",
			relayState:   "https://app.example.com/callback",
			allowedHosts: []string{},
			wantURL:      "",
			wantError:    true,
		},
		{
			name:         "absolute URL allowed when host matches",
			relayState:   "https://app.example.com/callback",
			allowedHosts: []string{"app.example.com"},
			wantURL:      "https://app.example.com/callback",
			wantError:    false,
		},
		{
			name:         "subdomain allowed when parent in list",
			relayState:   "https://api.app.example.com/callback",
			allowedHosts: []string{"app.example.com"},
			wantURL:      "https://api.app.example.com/callback",
			wantError:    false,
		},
		{
			name:         "different domain blocked",
			relayState:   "https://evil.com/phishing",
			allowedHosts: []string{"app.example.com"},
			wantURL:      "",
			wantError:    true,
		},
		{
			name:         "similar domain blocked (suffix attack)",
			relayState:   "https://notapp.example.com/callback",
			allowedHosts: []string{"app.example.com"},
			wantURL:      "",
			wantError:    true,
		},
		{
			name:         "multiple allowed hosts",
			relayState:   "https://dashboard.example.com/home",
			allowedHosts: []string{"app.example.com", "dashboard.example.com"},
			wantURL:      "https://dashboard.example.com/home",
			wantError:    false,
		},
		{
			name:         "HTTP URL with allowed host",
			relayState:   "http://app.example.com/callback",
			allowedHosts: []string{"app.example.com"},
			wantURL:      "http://app.example.com/callback",
			wantError:    false,
		},
		{
			name:         "invalid URL format",
			relayState:   "://invalid",
			allowedHosts: []string{},
			wantURL:      "",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := ValidateRelayState(tt.relayState, tt.allowedHosts)

			if (err != nil) != tt.wantError {
				t.Errorf("ValidateRelayState() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if gotURL != tt.wantURL {
				t.Errorf("ValidateRelayState() = %q, want %q", gotURL, tt.wantURL)
			}
		})
	}
}

func TestSanitizeSAMLAttribute(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal name",
			input: "John Doe",
			want:  "John Doe",
		},
		{
			name:  "name with accents",
			input: "José García",
			want:  "José García",
		},
		{
			name:  "null bytes removed",
			input: "John\x00Doe",
			want:  "JohnDoe",
		},
		{
			name:  "control characters removed",
			input: "John\x01\x02\x03Doe",
			want:  "JohnDoe",
		},
		{
			name:  "tabs preserved",
			input: "John\tDoe",
			want:  "John\tDoe",
		},
		{
			name:  "newlines preserved",
			input: "John\nDoe",
			want:  "John\nDoe",
		},
		{
			name:  "whitespace trimmed",
			input: "  John Doe  ",
			want:  "John Doe",
		},
		{
			name:  "long string truncated",
			input: strings.Repeat("a", 2000),
			want:  strings.Repeat("a", 1024),
		},
		{
			name:  "unicode preserved",
			input: "田中太郎",
			want:  "田中太郎",
		},
		{
			name:  "emoji preserved",
			input: "John 😀 Doe",
			want:  "John 😀 Doe",
		},
		{
			name:  "mixed control and valid",
			input: "\x00John\x1FDoe\x7F",
			want:  "JohnDoe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeSAMLAttribute(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeSAMLAttribute() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateMetadataURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		allowInsecure bool
		wantError     bool
	}{
		{
			name:          "HTTPS allowed",
			url:           "https://idp.example.com/metadata",
			allowInsecure: false,
			wantError:     false,
		},
		{
			name:          "HTTP rejected by default",
			url:           "http://idp.example.com/metadata",
			allowInsecure: false,
			wantError:     true,
		},
		{
			name:          "HTTP allowed when explicitly enabled",
			url:           "http://idp.example.com/metadata",
			allowInsecure: true,
			wantError:     false,
		},
		{
			name:          "invalid URL",
			url:           "://invalid",
			allowInsecure: false,
			wantError:     true,
		},
		{
			name:          "FTP scheme rejected",
			url:           "ftp://idp.example.com/metadata",
			allowInsecure: false,
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMetadataURL(tt.url, tt.allowInsecure)

			if (err != nil) != tt.wantError {
				t.Errorf("validateMetadataURL() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
