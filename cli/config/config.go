// Package config provides configuration management for the Fluxbase CLI.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Version is the config file format version
const Version = "1"

// Config represents the CLI configuration file
type Config struct {
	// Version of the config file format
	Version string `yaml:"version"`

	// CurrentProfile is the active profile name
	CurrentProfile string `yaml:"current_profile"`

	// Profiles is a map of profile name to profile configuration
	Profiles map[string]*Profile `yaml:"profiles"`

	// Defaults for all commands
	Defaults Defaults `yaml:"defaults,omitempty"`
}

// Profile represents a named configuration profile
type Profile struct {
	// Name is the profile identifier (e.g., "dev", "staging", "prod")
	Name string `yaml:"name"`

	// Server is the Fluxbase server URL
	Server string `yaml:"server"`

	// CredentialStore is "file" or "keychain"
	CredentialStore string `yaml:"credential_store"`

	// Credentials when using file storage
	Credentials *Credentials `yaml:"credentials,omitempty"`

	// User info cached from last login
	User *UserInfo `yaml:"user,omitempty"`

	// DefaultNamespace for functions/jobs/chatbots
	DefaultNamespace string `yaml:"default_namespace,omitempty"`

	// OutputFormat default for this profile
	OutputFormat string `yaml:"output_format,omitempty"`
}

// Credentials stores authentication tokens
type Credentials struct {
	// AccessToken is the JWT access token
	AccessToken string `yaml:"access_token,omitempty"`

	// RefreshToken is used to obtain new access tokens
	RefreshToken string `yaml:"refresh_token,omitempty"`

	// ExpiresAt is the access token expiration time (unix timestamp)
	ExpiresAt int64 `yaml:"expires_at,omitempty"`

	// APIKey is an alternative to JWT (for service accounts)
	APIKey string `yaml:"api_key,omitempty"`
}

// UserInfo caches user information
type UserInfo struct {
	ID            string `yaml:"id"`
	Email         string `yaml:"email"`
	Role          string `yaml:"role"`
	EmailVerified bool   `yaml:"email_verified"`
}

// Defaults contains default settings for commands
type Defaults struct {
	// Output format: table, json, yaml
	Output string `yaml:"output,omitempty"`

	// NoHeaders suppresses table headers
	NoHeaders bool `yaml:"no_headers,omitempty"`

	// Quiet mode for minimal output
	Quiet bool `yaml:"quiet,omitempty"`

	// Namespace default
	Namespace string `yaml:"namespace,omitempty"`
}

// DefaultConfigDir returns the default config directory path
func DefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".fluxbase"
	}
	return filepath.Join(home, ".fluxbase")
}

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "config.yaml")
}

// New creates a new empty configuration
func New() *Config {
	return &Config{
		Version:  Version,
		Profiles: make(map[string]*Profile),
		Defaults: Defaults{
			Output: "table",
		},
	}
}

// Load reads configuration from the specified path
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s - run 'fluxbase auth login' to create one", path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*Profile)
	}

	return &cfg, nil
}

// LoadOrCreate reads configuration or creates a new one if it doesn't exist
func LoadOrCreate(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}

	cfg, err := Load(path)
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, os.ErrNotExist) {
			return New(), nil
		}
		// Check if the error message contains "not found"
		if errors.As(err, new(*os.PathError)) || os.IsNotExist(err) {
			return New(), nil
		}
		// If config doesn't exist, create a new one
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			return New(), nil
		}
		return nil, err
	}

	return cfg, nil
}

// Save writes the configuration to the specified path
func (c *Config) Save(path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}

	// Ensure directory exists with restricted permissions
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with 0600 permissions (owner read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetProfile returns the named profile or the current profile if name is empty
func (c *Config) GetProfile(name string) (*Profile, error) {
	if name == "" {
		name = c.CurrentProfile
	}

	if name == "" {
		return nil, fmt.Errorf("no profile specified and no current profile set")
	}

	profile, ok := c.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}

	return profile, nil
}

// SetProfile adds or updates a profile
func (c *Config) SetProfile(profile *Profile) {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}
	c.Profiles[profile.Name] = profile
}

// DeleteProfile removes a profile
func (c *Config) DeleteProfile(name string) error {
	if _, ok := c.Profiles[name]; !ok {
		return fmt.Errorf("profile '%s' not found", name)
	}

	delete(c.Profiles, name)

	// If we deleted the current profile, clear it
	if c.CurrentProfile == name {
		c.CurrentProfile = ""
		// Set to first available profile if any
		for pName := range c.Profiles {
			c.CurrentProfile = pName
			break
		}
	}

	return nil
}

// ListProfiles returns a list of all profile names
func (c *Config) ListProfiles() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}

// HasCredentials returns true if the profile has valid credentials
func (p *Profile) HasCredentials() bool {
	if p.Credentials == nil {
		return false
	}
	return p.Credentials.AccessToken != "" || p.Credentials.APIKey != ""
}

// IsTokenExpired returns true if the access token has expired
func (p *Profile) IsTokenExpired() bool {
	if p.Credentials == nil || p.Credentials.ExpiresAt == 0 {
		return false
	}
	return p.Credentials.ExpiresAt < currentUnixTime()
}

// currentUnixTime returns the current unix timestamp
func currentUnixTime() int64 {
	return 0 // Will be implemented with time.Now().Unix()
}
