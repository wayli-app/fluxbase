package config

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server        ServerConfig     `mapstructure:"server"`
	Database      DatabaseConfig   `mapstructure:"database"`
	Auth          AuthConfig       `mapstructure:"auth"`
	Security      SecurityConfig   `mapstructure:"security"`
	CORS          CORSConfig       `mapstructure:"cors"`
	Storage       StorageConfig    `mapstructure:"storage"`
	Realtime      RealtimeConfig   `mapstructure:"realtime"`
	Email         EmailConfig      `mapstructure:"email"`
	Functions     FunctionsConfig  `mapstructure:"functions"`
	API           APIConfig        `mapstructure:"api"`
	Migrations    MigrationsConfig `mapstructure:"migrations"`
	Jobs          JobsConfig       `mapstructure:"jobs"`
	Deno          DenoConfig       `mapstructure:"deno"`
	Tracing       TracingConfig    `mapstructure:"tracing"`
	Metrics       MetricsConfig    `mapstructure:"metrics"`
	AI            AIConfig         `mapstructure:"ai"`
	RPC           RPCConfig        `mapstructure:"rpc"`
	GraphQL       GraphQLConfig    `mapstructure:"graphql"`
	MCP           MCPConfig        `mapstructure:"mcp"`
	Branching     BranchingConfig  `mapstructure:"branching"`
	Scaling       ScalingConfig    `mapstructure:"scaling"`
	Logging       LoggingConfig    `mapstructure:"logging"`
	Admin         AdminConfig      `mapstructure:"admin"`
	Tenants       TenantsConfig    `mapstructure:"tenants"`
	BaseURL       string           `mapstructure:"base_url"`        // Internal base URL (for server-to-server communication)
	PublicBaseURL string           `mapstructure:"public_base_url"` // Public base URL (for user-facing links, OAuth callbacks, etc.)
	Debug         bool             `mapstructure:"debug"`

	// EncryptionKey is used to encrypt sensitive data stored in the database (e.g., client keys, credentials)
	// Must be exactly 32 bytes for AES-256. Generate with: openssl rand -base64 32 | head -c 32
	// Only required if you configure providers (Email, AI) through the admin dashboard instead of env vars
	EncryptionKey      string `mapstructure:"encryption_key"`
	EncryptionKeyBytes []byte
}

// AdminConfig contains admin dashboard settings
type AdminConfig struct {
	Enabled bool `mapstructure:"enabled"` // Enable admin dashboard UI (React app). API routes are always available when setup_token is set.
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`   // List of allowed origins (use ["*"] for all)
	AllowedMethods   []string `mapstructure:"allowed_methods"`   // List of allowed HTTP methods
	AllowedHeaders   []string `mapstructure:"allowed_headers"`   // List of allowed headers
	ExposedHeaders   []string `mapstructure:"exposed_headers"`   // List of exposed headers
	AllowCredentials bool     `mapstructure:"allow_credentials"` // Allow credentials (cookies, authorization headers)
	MaxAge           int      `mapstructure:"max_age"`           // Max age for preflight cache in seconds
}

// RealtimeConfig contains realtime/websocket settings
type RealtimeConfig struct {
	Enabled                bool          `mapstructure:"enabled"`
	MaxConnections         int           `mapstructure:"max_connections"`
	MaxConnectionsPerUser  int           `mapstructure:"max_connections_per_user"`  // Max connections per authenticated user (0 = unlimited)
	MaxConnectionsPerIP    int           `mapstructure:"max_connections_per_ip"`    // Max connections per IP for anonymous connections (0 = unlimited)
	RLSCacheSize           int           `mapstructure:"rls_cache_size"`            // Maximum entries in RLS cache (default: 100000)
	RLSCacheTTL            time.Duration `mapstructure:"rls_cache_ttl"`             // TTL for RLS cache entries (default: 30s)
	ListenerPoolSize       int           `mapstructure:"listener_pool_size"`        // Number of LISTEN connections for redundancy (default: 2)
	NotificationWorkers    int           `mapstructure:"notification_workers"`      // Number of workers for parallel notification processing (default: 4)
	ClientMessageQueueSize int           `mapstructure:"client_message_queue_size"` // Size of per-client message queue for async sending (default: 256)
	SlowClientThreshold    int           `mapstructure:"slow_client_threshold"`     // Queue length threshold for slow client detection (default: 100)
	SlowClientTimeout      time.Duration `mapstructure:"slow_client_timeout"`       // Duration before disconnecting slow clients (default: 30s)
}

// MigrationsConfig contains migrations API security settings
type MigrationsConfig struct {
	Enabled         bool     `mapstructure:"enabled"`           // Enable migrations API (enabled by default)
	AllowedIPRanges []string `mapstructure:"allowed_ip_ranges"` // IP CIDR ranges allowed to access migrations API
}

// RPCConfig contains RPC (Remote Procedure Call) configuration
type RPCConfig struct {
	Enabled             bool     `mapstructure:"enabled"`                // Enable RPC functionality
	ProceduresDir       string   `mapstructure:"procedures_dir"`         // Directory for RPC procedure definitions
	AutoLoadOnBoot      bool     `mapstructure:"auto_load_on_boot"`      // Load procedures from filesystem at boot
	DefaultMaxRows      int      `mapstructure:"default_max_rows"`       // Default max rows returned
	SyncAllowedIPRanges []string `mapstructure:"sync_allowed_ip_ranges"` // IP CIDR ranges allowed to sync procedures
}

// DenoConfig contains Deno runtime settings for edge functions and background jobs
type DenoConfig struct {
	NpmRegistry string `mapstructure:"npm_registry"` // Custom npm registry URL (e.g., https://npm.your-company.com/)
	JsrRegistry string `mapstructure:"jsr_registry"` // Custom JSR registry URL (e.g., https://jsr.your-company.com/)
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (for local development)
	if err := loadEnvFile(); err != nil {
		log.Debug().Msg("No .env file found - using environment variables and defaults")
	}

	// Set defaults
	setDefaults()

	// Enable environment variable support with underscore replacer
	viper.AutomaticEnv()
	viper.SetEnvPrefix("FLUXBASE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try to load config file from specific paths (in order of priority)
	// This is more explicit than using SetConfigName which would also match .example files
	configPaths := []string{
		"./fluxbase.yaml",
		"./fluxbase.yml",
		"./config/fluxbase.yaml",
		"./config/fluxbase.yml",
		"/etc/fluxbase/fluxbase.yaml",
		"/etc/fluxbase/fluxbase.yml",
	}

	var configLoaded bool
	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); err == nil {
			viper.SetConfigFile(configPath)
			if err := viper.ReadInConfig(); err != nil {
				log.Warn().Err(err).Str("file", configPath).Msg("Config file found but could not be parsed, using environment variables and defaults")
			} else {
				log.Info().Str("file", configPath).Msg("Config file loaded")
				configLoaded = true
			}
			break
		}
	}

	if !configLoaded {
		log.Info().Msg("No config file found, using environment variables and defaults")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// loadEnvFile loads environment variables from .env file
func loadEnvFile() error {
	// Check multiple locations for .env file
	locations := []string{
		".env",
		".env.local",
		"../.env", // For when running from subdirectories
	}

	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			if err := godotenv.Load(location); err != nil {
				return fmt.Errorf("error loading .env file from %s: %w", location, err)
			}
			log.Info().Str("file", location).Msg(".env file loaded")
			return nil
		}
	}

	return fmt.Errorf("no .env file found")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server configuration
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server configuration error: %w", err)
	}

	// Validate database configuration
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database configuration error: %w", err)
	}

	// Validate auth configuration
	if err := c.Auth.Validate(); err != nil {
		return fmt.Errorf("auth configuration error: %w", err)
	}

	// Validate storage configuration
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage configuration error: %w", err)
	}

	// Validate security configuration
	if err := c.Security.Validate(); err != nil {
		return fmt.Errorf("security configuration error: %w", err)
	}

	// Validate email configuration if enabled
	if c.Email.Enabled {
		if err := c.Email.Validate(); err != nil {
			return fmt.Errorf("email configuration error: %w", err)
		}
	}

	// Validate functions configuration if enabled
	if c.Functions.Enabled {
		if err := c.Functions.Validate(); err != nil {
			return fmt.Errorf("functions configuration error: %w", err)
		}
	}

	// Validate API configuration
	if err := c.API.Validate(); err != nil {
		return fmt.Errorf("api configuration error: %w", err)
	}

	// Validate jobs configuration if enabled
	if c.Jobs.Enabled {
		if err := c.Jobs.Validate(); err != nil {
			return fmt.Errorf("jobs configuration error: %w", err)
		}
	}

	// Validate tracing configuration if enabled
	if c.Tracing.Enabled {
		if err := c.Tracing.Validate(); err != nil {
			return fmt.Errorf("tracing configuration error: %w", err)
		}
	}

	// Validate metrics configuration if enabled
	if c.Metrics.Enabled {
		if err := c.Metrics.Validate(); err != nil {
			return fmt.Errorf("metrics configuration error: %w", err)
		}
	}

	// Validate AI configuration if enabled
	if c.AI.Enabled {
		if err := c.AI.Validate(); err != nil {
			return fmt.Errorf("ai configuration error: %w", err)
		}
	}

	// Validate GraphQL configuration if enabled
	if c.GraphQL.Enabled {
		if err := c.GraphQL.Validate(); err != nil {
			return fmt.Errorf("graphql configuration error: %w", err)
		}
	}

	// Validate MCP configuration if enabled
	if c.MCP.Enabled {
		if err := c.MCP.Validate(); err != nil {
			return fmt.Errorf("mcp configuration error: %w", err)
		}
	}

	// Validate branching configuration if enabled
	if c.Branching.Enabled {
		if err := c.Branching.Validate(); err != nil {
			return fmt.Errorf("branching configuration error: %w", err)
		}
	}

	// Validate scaling configuration
	if err := c.Scaling.Validate(); err != nil {
		return fmt.Errorf("scaling configuration error: %w", err)
	}

	// Validate logging configuration
	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging configuration error: %w", err)
	}

	// Validate encryption key - required for secure secrets storage
	if c.EncryptionKey == "" {
		return fmt.Errorf("encryption_key is required for AES-256 encryption (must be exactly 32 bytes)")
	}
	if len(c.EncryptionKey) != 32 {
		return fmt.Errorf("encryption_key must be exactly 32 bytes for AES-256, got %d bytes", len(c.EncryptionKey))
	}
	c.EncryptionKeyBytes = []byte(c.EncryptionKey)

	// Validate base URL
	if c.BaseURL != "" {
		parsedURL, err := url.Parse(c.BaseURL)
		if err != nil {
			return fmt.Errorf("invalid base_url: %w", err)
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("base_url must use http or https scheme, got: %s", parsedURL.Scheme)
		}
	}

	// Validate public base URL if set
	if c.PublicBaseURL != "" {
		parsedURL, err := url.Parse(c.PublicBaseURL)
		if err != nil {
			return fmt.Errorf("invalid public_base_url: %w", err)
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("public_base_url must use http or https scheme, got: %s", parsedURL.Scheme)
		}
	}

	return nil
}

// GetPublicBaseURL returns the public-facing base URL.
// If PublicBaseURL is set, it returns that; otherwise, it falls back to BaseURL.
// This should be used for all user-facing URLs (magic links, OAuth callbacks, invitation links, etc.)
func (c *Config) GetPublicBaseURL() string {
	if c.PublicBaseURL != "" {
		return c.PublicBaseURL
	}
	return c.BaseURL
}

// calculateEntropy calculates the Shannon entropy of a string in bits.
// Higher entropy indicates more randomness and better security.
// Formula: H = -Σ p(x) * log2(p(x)) where p(x) is the probability of character x
func calculateEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	// Count frequency of each character
	freq := make(map[rune]int)
	for _, char := range s {
		freq[char]++
	}

	// Calculate Shannon entropy
	length := float64(len(s))
	entropy := 0.0

	for _, count := range freq {
		probability := float64(count) / length
		entropy -= probability * math.Log2(probability)
	}

	return entropy
}
