package config

import (
	"fmt"
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
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Security  SecurityConfig  `mapstructure:"security"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Realtime  RealtimeConfig  `mapstructure:"realtime"`
	Email     EmailConfig     `mapstructure:"email"`
	Functions FunctionsConfig `mapstructure:"functions"`
	BaseURL   string          `mapstructure:"base_url"`
	Debug     bool            `mapstructure:"debug"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Address      string        `mapstructure:"address"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	BodyLimit    int           `mapstructure:"body_limit"`
}

// DatabaseConfig contains PostgreSQL connection settings
type DatabaseConfig struct {
	Host               string        `mapstructure:"host"`
	Port               int           `mapstructure:"port"`
	User               string        `mapstructure:"user"`           // Database user for normal operations
	AdminUser          string        `mapstructure:"admin_user"`     // Optional admin user for migrations (defaults to User)
	Password           string        `mapstructure:"password"`       // Password for runtime user
	AdminPassword      string        `mapstructure:"admin_password"` // Optional password for admin user (defaults to Password)
	Database           string        `mapstructure:"database"`
	SSLMode            string        `mapstructure:"ssl_mode"`
	MaxConnections     int32         `mapstructure:"max_connections"`
	MinConnections     int32         `mapstructure:"min_connections"`
	MaxConnLifetime    time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime    time.Duration `mapstructure:"max_conn_idle_time"`
	HealthCheck        time.Duration `mapstructure:"health_check_period"`
	UserMigrationsPath string        `mapstructure:"user_migrations_path"` // Path to user-provided migration files
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	JWTSecret           string        `mapstructure:"jwt_secret"`
	JWTExpiry           time.Duration `mapstructure:"jwt_expiry"`
	RefreshExpiry       time.Duration `mapstructure:"refresh_expiry"`
	MagicLinkExpiry     time.Duration `mapstructure:"magic_link_expiry"`
	PasswordResetExpiry time.Duration `mapstructure:"password_reset_expiry"`
	PasswordMinLen      int           `mapstructure:"password_min_length"`
	BcryptCost          int           `mapstructure:"bcrypt_cost"`
	EnableSignup        bool          `mapstructure:"enable_signup"`
	EnableMagicLink     bool          `mapstructure:"enable_magic_link"`
	EnableRLS           bool          `mapstructure:"enable_rls"` // Row Level Security enforcement
}

// SecurityConfig contains security-related settings
type SecurityConfig struct {
	EnableGlobalRateLimit bool `mapstructure:"enable_global_rate_limit"` // Global API rate limiting (100 req/min per IP)

	// Admin setup security token
	SetupToken string `mapstructure:"setup_token"` // Required token for admin setup. If empty, admin dashboard is disabled.

	// Rate limiting for specific endpoints
	AdminSetupRateLimit  int           `mapstructure:"admin_setup_rate_limit"`  // Max attempts for admin setup
	AdminSetupRateWindow time.Duration `mapstructure:"admin_setup_rate_window"` // Time window for admin setup rate limit
	AuthLoginRateLimit   int           `mapstructure:"auth_login_rate_limit"`   // Max attempts for auth login
	AuthLoginRateWindow  time.Duration `mapstructure:"auth_login_rate_window"`  // Time window for auth login rate limit
	AdminLoginRateLimit  int           `mapstructure:"admin_login_rate_limit"`  // Max attempts for admin login
	AdminLoginRateWindow time.Duration `mapstructure:"admin_login_rate_window"` // Time window for admin login rate limit
}

// StorageConfig contains file storage settings
type StorageConfig struct {
	Provider      string `mapstructure:"provider"` // local or s3
	LocalPath     string `mapstructure:"local_path"`
	S3Endpoint    string `mapstructure:"s3_endpoint"`
	S3AccessKey   string `mapstructure:"s3_access_key"`
	S3SecretKey   string `mapstructure:"s3_secret_key"`
	S3Bucket      string `mapstructure:"s3_bucket"`
	S3Region      string `mapstructure:"s3_region"`
	MaxUploadSize int64  `mapstructure:"max_upload_size"`
}

// RealtimeConfig contains realtime/websocket settings
type RealtimeConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	MaxConnections    int           `mapstructure:"max_connections"`
	PingInterval      time.Duration `mapstructure:"ping_interval"`
	PongTimeout       time.Duration `mapstructure:"pong_timeout"`
	WriteBufferSize   int           `mapstructure:"write_buffer_size"`
	ReadBufferSize    int           `mapstructure:"read_buffer_size"`
	MessageSizeLimit  int64         `mapstructure:"message_size_limit"`
	ChannelBufferSize int           `mapstructure:"channel_buffer_size"`
}

// EmailConfig contains email/SMTP settings
type EmailConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	Provider       string `mapstructure:"provider"` // smtp, sendgrid, mailgun, ses
	FromAddress    string `mapstructure:"from_address"`
	FromName       string `mapstructure:"from_name"`
	ReplyToAddress string `mapstructure:"reply_to_address"`

	// SMTP Settings
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUsername string `mapstructure:"smtp_username"`
	SMTPPassword string `mapstructure:"smtp_password"`
	SMTPTLS      bool   `mapstructure:"smtp_tls"`

	// SendGrid Settings
	SendGridAPIKey string `mapstructure:"sendgrid_api_key"`

	// Mailgun Settings
	MailgunAPIKey string `mapstructure:"mailgun_api_key"`
	MailgunDomain string `mapstructure:"mailgun_domain"`

	// AWS SES Settings
	SESAccessKey string `mapstructure:"ses_access_key"`
	SESSecretKey string `mapstructure:"ses_secret_key"`
	SESRegion    string `mapstructure:"ses_region"`

	// Templates
	MagicLinkTemplate     string `mapstructure:"magic_link_template"`
	VerificationTemplate  string `mapstructure:"verification_template"`
	PasswordResetTemplate string `mapstructure:"password_reset_template"`
}

// FunctionsConfig contains edge functions settings
type FunctionsConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	FunctionsDir       string `mapstructure:"functions_dir"`
	DefaultTimeout     int    `mapstructure:"default_timeout"`      // seconds
	MaxTimeout         int    `mapstructure:"max_timeout"`          // seconds
	DefaultMemoryLimit int    `mapstructure:"default_memory_limit"` // MB
	MaxMemoryLimit     int    `mapstructure:"max_memory_limit"`     // MB
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

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.address", ":8080")
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("server.idle_timeout", "60s")
	viper.SetDefault("server.body_limit", 2*1024*1024*1024) // 2GB

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres") // Default runtime user
	viper.SetDefault("database.admin_user", "")   // Empty means use user
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.database", "fluxbase")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("database.min_connections", 5)
	viper.SetDefault("database.max_conn_lifetime", "1h")
	viper.SetDefault("database.max_conn_idle_time", "30m")
	viper.SetDefault("database.health_check_period", "1m")
	viper.SetDefault("database.user_migrations_path", "/migrations/user")

	// Auth defaults
	viper.SetDefault("auth.jwt_secret", "your-secret-key-change-in-production")
	viper.SetDefault("auth.jwt_expiry", "15m")
	viper.SetDefault("auth.refresh_expiry", "168h") // 7 days in hours
	viper.SetDefault("auth.magic_link_expiry", "15m")
	viper.SetDefault("auth.password_reset_expiry", "1h")
	viper.SetDefault("auth.password_min_length", 8)
	viper.SetDefault("auth.bcrypt_cost", 10)
	viper.SetDefault("auth.enable_signup", true) // Default to enabled to allow user registration
	viper.SetDefault("auth.enable_magic_link", true)
	viper.SetDefault("auth.enable_rls", true) // Row Level Security enabled by default

	// Security defaults
	viper.SetDefault("security.enable_global_rate_limit", false) // Disabled by default, enable in production if needed
	viper.SetDefault("security.setup_token", "")                 // Empty by default - admin dashboard disabled until set
	viper.SetDefault("security.admin_setup_rate_limit", 5)       // 5 attempts
	viper.SetDefault("security.admin_setup_rate_window", "15m")  // per 15 minutes
	viper.SetDefault("security.auth_login_rate_limit", 10)       // 10 attempts
	viper.SetDefault("security.auth_login_rate_window", "1m")    // per minute
	viper.SetDefault("security.admin_login_rate_limit", 10)      // 10 attempts
	viper.SetDefault("security.admin_login_rate_window", "1m")   // per minute

	// Storage defaults
	viper.SetDefault("storage.provider", "local")
	viper.SetDefault("storage.local_path", "./storage")
	viper.SetDefault("storage.s3_endpoint", "")
	viper.SetDefault("storage.s3_access_key", "")
	viper.SetDefault("storage.s3_secret_key", "")
	viper.SetDefault("storage.s3_bucket", "")
	viper.SetDefault("storage.s3_region", "")
	viper.SetDefault("storage.max_upload_size", 2*1024*1024*1024) // 2GB

	// Realtime defaults
	viper.SetDefault("realtime.enabled", true)
	viper.SetDefault("realtime.max_connections", 1000)
	viper.SetDefault("realtime.ping_interval", "30s")
	viper.SetDefault("realtime.pong_timeout", "60s")
	viper.SetDefault("realtime.write_buffer_size", 1024)
	viper.SetDefault("realtime.read_buffer_size", 1024)
	viper.SetDefault("realtime.message_size_limit", 512*1024) // 512KB
	viper.SetDefault("realtime.channel_buffer_size", 100)

	// Email defaults
	viper.SetDefault("email.enabled", false)
	viper.SetDefault("email.provider", "smtp")
	viper.SetDefault("email.from_address", "noreply@localhost")
	viper.SetDefault("email.from_name", "Fluxbase")
	viper.SetDefault("email.smtp_port", 587)
	viper.SetDefault("email.smtp_tls", true)

	// Functions defaults
	viper.SetDefault("functions.enabled", true)
	viper.SetDefault("functions.functions_dir", "./functions")
	viper.SetDefault("functions.default_timeout", 30)       // 30 seconds
	viper.SetDefault("functions.max_timeout", 300)          // 5 minutes
	viper.SetDefault("functions.default_memory_limit", 128) // 128MB
	viper.SetDefault("functions.max_memory_limit", 1024)    // 1GB

	// General defaults
	viper.SetDefault("base_url", "http://localhost:8080")
	viper.SetDefault("debug", false)
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

	return nil
}

// Validate validates server configuration
func (sc *ServerConfig) Validate() error {
	if sc.Address == "" {
		return fmt.Errorf("server address cannot be empty")
	}

	// Validate timeouts are positive
	if sc.ReadTimeout <= 0 {
		return fmt.Errorf("read_timeout must be positive, got: %v", sc.ReadTimeout)
	}
	if sc.WriteTimeout <= 0 {
		return fmt.Errorf("write_timeout must be positive, got: %v", sc.WriteTimeout)
	}
	if sc.IdleTimeout <= 0 {
		return fmt.Errorf("idle_timeout must be positive, got: %v", sc.IdleTimeout)
	}

	// Validate body limit
	if sc.BodyLimit <= 0 {
		return fmt.Errorf("body_limit must be positive, got: %d", sc.BodyLimit)
	}

	return nil
}

// Validate validates database configuration
func (dc *DatabaseConfig) Validate() error {
	if dc.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if dc.Port < 1 || dc.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535, got: %d", dc.Port)
	}

	if dc.User == "" {
		return fmt.Errorf("database user is required")
	}

	// If AdminUser is not set, default it to User
	if dc.AdminUser == "" {
		dc.AdminUser = dc.User
	}

	if dc.Database == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate SSL mode
	validSSLModes := []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}
	sslModeValid := false
	for _, mode := range validSSLModes {
		if dc.SSLMode == mode {
			sslModeValid = true
			break
		}
	}
	if !sslModeValid {
		return fmt.Errorf("invalid ssl_mode: %s (must be one of: %v)", dc.SSLMode, validSSLModes)
	}

	// Validate connection pool settings
	if dc.MaxConnections <= 0 {
		return fmt.Errorf("max_connections must be positive, got: %d", dc.MaxConnections)
	}

	if dc.MinConnections < 0 {
		return fmt.Errorf("min_connections cannot be negative, got: %d", dc.MinConnections)
	}

	if dc.MaxConnections < dc.MinConnections {
		return fmt.Errorf("max_connections (%d) must be greater than or equal to min_connections (%d)",
			dc.MaxConnections, dc.MinConnections)
	}

	// Validate timeouts are positive
	if dc.MaxConnLifetime <= 0 {
		return fmt.Errorf("max_conn_lifetime must be positive, got: %v", dc.MaxConnLifetime)
	}
	if dc.MaxConnIdleTime <= 0 {
		return fmt.Errorf("max_conn_idle_time must be positive, got: %v", dc.MaxConnIdleTime)
	}
	if dc.HealthCheck <= 0 {
		return fmt.Errorf("health_check_period must be positive, got: %v", dc.HealthCheck)
	}

	return nil
}

// Validate validates auth configuration
func (ac *AuthConfig) Validate() error {
	if ac.JWTSecret == "" {
		return fmt.Errorf("jwt_secret is required")
	}

	if ac.JWTSecret == "your-secret-key-change-in-production" {
		return fmt.Errorf("please set a secure JWT secret (current value is the default insecure value)")
	}

	// Validate JWT secret length (should be at least 32 characters for security)
	if len(ac.JWTSecret) < 32 {
		log.Warn().Msg("JWT secret is shorter than 32 characters - consider using a longer secret for better security")
	}

	// Validate expiry durations are positive
	if ac.JWTExpiry <= 0 {
		return fmt.Errorf("jwt_expiry must be positive, got: %v", ac.JWTExpiry)
	}
	if ac.RefreshExpiry <= 0 {
		return fmt.Errorf("refresh_expiry must be positive, got: %v", ac.RefreshExpiry)
	}
	if ac.MagicLinkExpiry <= 0 {
		return fmt.Errorf("magic_link_expiry must be positive, got: %v", ac.MagicLinkExpiry)
	}
	if ac.PasswordResetExpiry <= 0 {
		return fmt.Errorf("password_reset_expiry must be positive, got: %v", ac.PasswordResetExpiry)
	}

	// Validate password settings
	if ac.PasswordMinLen < 1 {
		return fmt.Errorf("password_min_length must be at least 1, got: %d", ac.PasswordMinLen)
	}
	if ac.PasswordMinLen < 8 {
		log.Warn().Int("min_length", ac.PasswordMinLen).Msg("Password minimum length is less than 8 - consider increasing for better security")
	}

	// Validate bcrypt cost (valid range is 4-31, recommended is 10-14)
	if ac.BcryptCost < 4 || ac.BcryptCost > 31 {
		return fmt.Errorf("bcrypt_cost must be between 4 and 31, got: %d", ac.BcryptCost)
	}

	return nil
}

// Validate validates storage configuration
func (sc *StorageConfig) Validate() error {
	if sc.Provider != "local" && sc.Provider != "s3" {
		return fmt.Errorf("storage provider must be 'local' or 's3', got: %s", sc.Provider)
	}

	if sc.Provider == "local" {
		if sc.LocalPath == "" {
			return fmt.Errorf("local_path is required when using local storage provider")
		}
	}

	if sc.Provider == "s3" {
		if sc.S3Endpoint == "" {
			return fmt.Errorf("s3_endpoint is required when using S3 storage provider")
		}
		if sc.S3AccessKey == "" {
			return fmt.Errorf("s3_access_key is required when using S3 storage provider")
		}
		if sc.S3SecretKey == "" {
			return fmt.Errorf("s3_secret_key is required when using S3 storage provider")
		}
		if sc.S3Bucket == "" {
			return fmt.Errorf("s3_bucket is required when using S3 storage provider")
		}
		// S3Region is optional for some S3-compatible services
	}

	// Validate max upload size
	if sc.MaxUploadSize <= 0 {
		return fmt.Errorf("max_upload_size must be positive, got: %d", sc.MaxUploadSize)
	}

	return nil
}

// ConnectionString returns the PostgreSQL connection string using the runtime user
// Deprecated: Use RuntimeConnectionString() or AdminConnectionString() instead
func (dc *DatabaseConfig) ConnectionString() string {
	return dc.RuntimeConnectionString()
}

// RuntimeConnectionString returns the PostgreSQL connection string for the runtime user
func (dc *DatabaseConfig) RuntimeConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dc.User, dc.Password, dc.Host, dc.Port, dc.Database, dc.SSLMode)
}

// AdminConnectionString returns the PostgreSQL connection string for the admin user
func (dc *DatabaseConfig) AdminConnectionString() string {
	user := dc.AdminUser
	if user == "" {
		user = dc.User
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		user, dc.Password, dc.Host, dc.Port, dc.Database, dc.SSLMode)
}

// Validate validates email configuration
func (ec *EmailConfig) Validate() error {
	// Basic validation
	if ec.FromAddress == "" {
		return fmt.Errorf("from_address is required when email is enabled")
	}

	// Validate provider
	validProviders := []string{"smtp", "sendgrid", "mailgun", "ses"}
	providerValid := false
	for _, p := range validProviders {
		if ec.Provider == p {
			providerValid = true
			break
		}
	}
	if !providerValid {
		return fmt.Errorf("invalid email provider: %s (must be one of: %v)", ec.Provider, validProviders)
	}

	// Provider-specific validation
	switch ec.Provider {
	case "smtp":
		if ec.SMTPHost == "" {
			return fmt.Errorf("smtp_host is required when using SMTP provider")
		}
		if ec.SMTPPort == 0 {
			return fmt.Errorf("smtp_port is required when using SMTP provider")
		}
	case "sendgrid":
		if ec.SendGridAPIKey == "" {
			return fmt.Errorf("sendgrid_api_key is required when using SendGrid provider")
		}
	case "mailgun":
		if ec.MailgunAPIKey == "" {
			return fmt.Errorf("mailgun_api_key is required when using Mailgun provider")
		}
		if ec.MailgunDomain == "" {
			return fmt.Errorf("mailgun_domain is required when using Mailgun provider")
		}
	case "ses":
		if ec.SESAccessKey == "" || ec.SESSecretKey == "" {
			return fmt.Errorf("ses_access_key and ses_secret_key are required when using SES provider")
		}
		if ec.SESRegion == "" {
			return fmt.Errorf("ses_region is required when using SES provider")
		}
	}

	return nil
}

// Validate validates functions configuration
func (fc *FunctionsConfig) Validate() error {
	// Validate functions directory
	if fc.FunctionsDir == "" {
		return fmt.Errorf("functions_dir cannot be empty")
	}

	// Validate timeout settings
	if fc.DefaultTimeout <= 0 {
		return fmt.Errorf("default_timeout must be positive, got: %d", fc.DefaultTimeout)
	}
	if fc.MaxTimeout <= 0 {
		return fmt.Errorf("max_timeout must be positive, got: %d", fc.MaxTimeout)
	}
	if fc.DefaultTimeout > fc.MaxTimeout {
		return fmt.Errorf("default_timeout (%d) cannot be greater than max_timeout (%d)", fc.DefaultTimeout, fc.MaxTimeout)
	}

	// Validate memory limit settings
	if fc.DefaultMemoryLimit <= 0 {
		return fmt.Errorf("default_memory_limit must be positive, got: %d", fc.DefaultMemoryLimit)
	}
	if fc.MaxMemoryLimit <= 0 {
		return fmt.Errorf("max_memory_limit must be positive, got: %d", fc.MaxMemoryLimit)
	}
	if fc.DefaultMemoryLimit > fc.MaxMemoryLimit {
		return fmt.Errorf("default_memory_limit (%d) cannot be greater than max_memory_limit (%d)", fc.DefaultMemoryLimit, fc.MaxMemoryLimit)
	}

	// Warn if max_timeout is very high (over 5 minutes)
	if fc.MaxTimeout > 300 {
		log.Warn().Int("max_timeout", fc.MaxTimeout).Msg("max_timeout is over 5 minutes - long-running functions may impact performance")
	}

	// Warn if max_memory_limit is very high (over 1GB)
	if fc.MaxMemoryLimit > 1024 {
		log.Warn().Int("max_memory_limit", fc.MaxMemoryLimit).Msg("max_memory_limit is over 1GB - high memory functions may impact performance")
	}

	return nil
}
