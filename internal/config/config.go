package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Realtime RealtimeConfig `mapstructure:"realtime"`
	Email    EmailConfig    `mapstructure:"email"`
	BaseURL  string         `mapstructure:"base_url"`
	Debug    bool           `mapstructure:"debug"`
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
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxConnections  int32         `mapstructure:"max_connections"`
	MinConnections  int32         `mapstructure:"min_connections"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
	HealthCheck     time.Duration `mapstructure:"health_check_period"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	JWTSecret       string        `mapstructure:"jwt_secret"`
	JWTExpiry       time.Duration `mapstructure:"jwt_expiry"`
	RefreshExpiry   time.Duration `mapstructure:"refresh_expiry"`
	PasswordMinLen  int           `mapstructure:"password_min_length"`
	BcryptCost      int           `mapstructure:"bcrypt_cost"`
	EnableSignup    bool          `mapstructure:"enable_signup"`
	EnableMagicLink bool          `mapstructure:"enable_magic_link"`
	EnableRLS       bool          `mapstructure:"enable_rls"` // Row Level Security enforcement
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
	MagicLinkTemplate     string        `mapstructure:"magic_link_template"`
	VerificationTemplate  string        `mapstructure:"verification_template"`
	PasswordResetTemplate string        `mapstructure:"password_reset_template"`
	MagicLinkExpiry       time.Duration `mapstructure:"magic_link_expiry"`
	PasswordResetExpiry   time.Duration `mapstructure:"password_reset_expiry"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (for local development)
	if err := loadEnvFile(); err != nil {
		log.Debug().Err(err).Msg("No .env file loaded")
	}

	viper.SetConfigName("fluxbase")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/fluxbase")

	// Set defaults
	setDefaults()

	// Enable environment variable support with underscore replacer
	viper.AutomaticEnv()
	viper.SetEnvPrefix("FLUXBASE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file (if it exists)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; use defaults and environment variables
		log.Info().Msg("No config file found, using environment variables and defaults")
	} else {
		log.Info().Str("file", viper.ConfigFileUsed()).Msg("Config file loaded")
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
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.database", "fluxbase")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("database.min_connections", 5)
	viper.SetDefault("database.max_conn_lifetime", "1h")
	viper.SetDefault("database.max_conn_idle_time", "30m")
	viper.SetDefault("database.health_check_period", "1m")

	// Auth defaults
	viper.SetDefault("auth.jwt_secret", "your-secret-key-change-in-production")
	viper.SetDefault("auth.jwt_expiry", "15m")
	viper.SetDefault("auth.refresh_expiry", "168h") // 7 days in hours
	viper.SetDefault("auth.password_min_length", 8)
	viper.SetDefault("auth.bcrypt_cost", 10)
	viper.SetDefault("auth.enable_signup", true)
	viper.SetDefault("auth.enable_magic_link", true)
	viper.SetDefault("auth.enable_rls", true) // Row Level Security enabled by default

	// Storage defaults
	viper.SetDefault("storage.provider", "local")
	viper.SetDefault("storage.local_path", "./storage")
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
	viper.SetDefault("email.magic_link_expiry", "15m")

	// General defaults
	viper.SetDefault("base_url", "http://localhost:8080")
	viper.SetDefault("debug", false)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Auth.JWTSecret == "your-secret-key-change-in-production" {
		return fmt.Errorf("please set a secure JWT secret")
	}

	if c.Database.MaxConnections < c.Database.MinConnections {
		return fmt.Errorf("max_connections must be greater than or equal to min_connections")
	}

	if c.Storage.Provider != "local" && c.Storage.Provider != "s3" {
		return fmt.Errorf("storage provider must be 'local' or 's3'")
	}

	if c.Storage.Provider == "s3" {
		if c.Storage.S3Endpoint == "" || c.Storage.S3AccessKey == "" ||
			c.Storage.S3SecretKey == "" || c.Storage.S3Bucket == "" {
			return fmt.Errorf("S3 configuration is incomplete")
		}
	}

	// Validate email configuration if enabled
	if c.Email.Enabled {
		if err := c.Email.Validate(); err != nil {
			return fmt.Errorf("email configuration error: %w", err)
		}
	}

	return nil
}

// ConnectionString returns the PostgreSQL connection string
func (dc *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dc.User, dc.Password, dc.Host, dc.Port, dc.Database, dc.SSLMode)
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
