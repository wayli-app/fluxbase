package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
)

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
	SlowQueryThreshold time.Duration `mapstructure:"slow_query_threshold"` // Log queries slower than this (default: 1s)
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
	if dc.SSLMode == "disable" {
		log.Warn().Msg("database.ssl_mode is 'disable' — database connections are unencrypted. Set ssl_mode to 'require' or higher in production.")
	}

	// Validate connection pool settings
	// MaxConnections must be between 1 and 1000 to prevent resource exhaustion
	if dc.MaxConnections < 1 {
		return fmt.Errorf("max_connections must be at least 1, got: %d", dc.MaxConnections)
	}
	if dc.MaxConnections > 1000 {
		return fmt.Errorf("max_connections must be at most 1000, got: %d", dc.MaxConnections)
	}

	// MinConnections must be non-negative and cannot exceed MaxConnections
	if dc.MinConnections < 0 {
		return fmt.Errorf("min_connections must be at least 0, got: %d", dc.MinConnections)
	}

	if dc.MinConnections > dc.MaxConnections {
		return fmt.Errorf("min_connections (%d) cannot exceed max_connections (%d)",
			dc.MinConnections, dc.MaxConnections)
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

// ConnectionString returns the PostgreSQL connection string using the runtime user
//
// Deprecated: Use RuntimeConnectionString() or AdminConnectionString() instead
func (dc *DatabaseConfig) ConnectionString() string {
	return dc.RuntimeConnectionString()
}

// RuntimeConnectionString returns the PostgreSQL connection string for the runtime user
// Uses url.URL for secure credential handling to prevent password injection
func (dc *DatabaseConfig) RuntimeConnectionString() string {
	return dc.buildSecureConnString(dc.User, dc.Password)
}

// AdminConnectionString returns the PostgreSQL connection string for the admin user
// Uses url.URL for secure credential handling to prevent password injection
func (dc *DatabaseConfig) AdminConnectionString() string {
	user := dc.AdminUser
	if user == "" {
		user = dc.User
	}
	password := dc.AdminPassword
	if password == "" {
		password = dc.Password
	}
	return dc.buildSecureConnString(user, password)
}

// buildSecureConnString creates a connection string using url.URL for secure credential handling
// This prevents password injection via special characters in passwords
func (dc *DatabaseConfig) buildSecureConnString(user, password string) string {
	// Use url.URL to properly encode credentials and prevent injection
	u := &url.URL{
		Scheme:   "postgres",
		Host:     fmt.Sprintf("%s:%d", dc.Host, dc.Port),
		Path:     "/" + dc.Database,
		RawQuery: fmt.Sprintf("sslmode=%s", dc.SSLMode),
	}
	u.User = url.UserPassword(user, password)
	return u.String()
}

// RedactConnString returns a connection string with the password redacted for logging
// Example: postgres://user:****@localhost:5432/db?sslmode=disable
func (dc *DatabaseConfig) RedactConnString(connStr string) string {
	// Parse the connection string
	u, err := url.Parse(connStr)
	if err != nil || u.Scheme == "" {
		// If parsing fails or it's not a valid URL, return a fully redacted string
		return "postgres://****@****:****/****?sslmode=****"
	}

	// Redact the password
	if u.User != nil {
		_, passwordSet := u.User.Password()
		if passwordSet {
			u.User = url.UserPassword(u.User.Username(), "****")
		}
	}

	return u.String()
}
