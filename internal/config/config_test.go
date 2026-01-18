package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: ServerConfig{
				Address:      ":8080",
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  60 * time.Second,
				BodyLimit:    1024 * 1024,
			},
			wantErr: false,
		},
		{
			name: "empty address",
			config: ServerConfig{
				Address:      "",
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  60 * time.Second,
				BodyLimit:    1024 * 1024,
			},
			wantErr: true,
			errMsg:  "server address cannot be empty",
		},
		{
			name: "zero read timeout",
			config: ServerConfig{
				Address:      ":8080",
				ReadTimeout:  0,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  60 * time.Second,
				BodyLimit:    1024 * 1024,
			},
			wantErr: true,
			errMsg:  "read_timeout must be positive",
		},
		{
			name: "negative write timeout",
			config: ServerConfig{
				Address:      ":8080",
				ReadTimeout:  30 * time.Second,
				WriteTimeout: -1 * time.Second,
				IdleTimeout:  60 * time.Second,
				BodyLimit:    1024 * 1024,
			},
			wantErr: true,
			errMsg:  "write_timeout must be positive",
		},
		{
			name: "zero idle timeout",
			config: ServerConfig{
				Address:      ":8080",
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  0,
				BodyLimit:    1024 * 1024,
			},
			wantErr: true,
			errMsg:  "idle_timeout must be positive",
		},
		{
			name: "zero body limit",
			config: ServerConfig{
				Address:      ":8080",
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  60 * time.Second,
				BodyLimit:    0,
			},
			wantErr: true,
			errMsg:  "body_limit must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDatabaseConfig_Validate(t *testing.T) {
	validConfig := func() DatabaseConfig {
		return DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "postgres",
			Password:        "password",
			Database:        "fluxbase",
			SSLMode:         "disable",
			MaxConnections:  50,
			MinConnections:  10,
			MaxConnLifetime: time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
			HealthCheck:     time.Minute,
		}
	}

	tests := []struct {
		name    string
		modify  func(*DatabaseConfig)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *DatabaseConfig) {},
			wantErr: false,
		},
		{
			name:    "empty host",
			modify:  func(c *DatabaseConfig) { c.Host = "" },
			wantErr: true,
			errMsg:  "database host is required",
		},
		{
			name:    "invalid port - zero",
			modify:  func(c *DatabaseConfig) { c.Port = 0 },
			wantErr: true,
			errMsg:  "database port must be between 1 and 65535",
		},
		{
			name:    "invalid port - too high",
			modify:  func(c *DatabaseConfig) { c.Port = 70000 },
			wantErr: true,
			errMsg:  "database port must be between 1 and 65535",
		},
		{
			name:    "empty user",
			modify:  func(c *DatabaseConfig) { c.User = "" },
			wantErr: true,
			errMsg:  "database user is required",
		},
		{
			name:    "empty database name",
			modify:  func(c *DatabaseConfig) { c.Database = "" },
			wantErr: true,
			errMsg:  "database name is required",
		},
		{
			name:    "invalid ssl mode",
			modify:  func(c *DatabaseConfig) { c.SSLMode = "invalid" },
			wantErr: true,
			errMsg:  "invalid ssl_mode",
		},
		{
			name:    "valid ssl mode - require",
			modify:  func(c *DatabaseConfig) { c.SSLMode = "require" },
			wantErr: false,
		},
		{
			name:    "valid ssl mode - verify-full",
			modify:  func(c *DatabaseConfig) { c.SSLMode = "verify-full" },
			wantErr: false,
		},
		{
			name:    "zero max connections",
			modify:  func(c *DatabaseConfig) { c.MaxConnections = 0 },
			wantErr: true,
			errMsg:  "max_connections must be positive",
		},
		{
			name:    "negative min connections",
			modify:  func(c *DatabaseConfig) { c.MinConnections = -1 },
			wantErr: true,
			errMsg:  "min_connections cannot be negative",
		},
		{
			name: "max less than min",
			modify: func(c *DatabaseConfig) {
				c.MaxConnections = 5
				c.MinConnections = 10
			},
			wantErr: true,
			errMsg:  "max_connections",
		},
		{
			name:    "admin user defaults to user",
			modify:  func(c *DatabaseConfig) { c.AdminUser = "" },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validConfig()
			tt.modify(&config)
			err := config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDatabaseConfig_ConnectionStrings(t *testing.T) {
	config := DatabaseConfig{
		Host:          "localhost",
		Port:          5432,
		User:          "app_user",
		Password:      "app_pass",
		AdminUser:     "admin_user",
		AdminPassword: "admin_pass",
		Database:      "testdb",
		SSLMode:       "disable",
	}

	t.Run("RuntimeConnectionString", func(t *testing.T) {
		connStr := config.RuntimeConnectionString()
		assert.Contains(t, connStr, "app_user")
		assert.Contains(t, connStr, "app_pass")
		assert.Contains(t, connStr, "localhost:5432")
		assert.Contains(t, connStr, "testdb")
	})

	t.Run("AdminConnectionString", func(t *testing.T) {
		connStr := config.AdminConnectionString()
		assert.Contains(t, connStr, "admin_user")
		assert.Contains(t, connStr, "admin_pass")
		assert.Contains(t, connStr, "localhost:5432")
	})

	t.Run("AdminConnectionString falls back to User when AdminUser empty", func(t *testing.T) {
		config.AdminUser = ""
		config.AdminPassword = ""
		connStr := config.AdminConnectionString()
		assert.Contains(t, connStr, "app_user")
		assert.Contains(t, connStr, "app_pass")
	})

	t.Run("ConnectionString is deprecated alias for RuntimeConnectionString", func(t *testing.T) {
		config.AdminUser = "admin"
		assert.Equal(t, config.RuntimeConnectionString(), config.ConnectionString())
	})
}

func TestAuthConfig_Validate(t *testing.T) {
	validConfig := func() AuthConfig {
		return AuthConfig{
			JWTSecret:           "this-is-a-very-secure-secret-key-for-testing-purposes",
			JWTExpiry:           15 * time.Minute,
			RefreshExpiry:       7 * 24 * time.Hour,
			MagicLinkExpiry:     15 * time.Minute,
			PasswordResetExpiry: time.Hour,
			PasswordMinLen:      8,
			BcryptCost:          10,
		}
	}

	tests := []struct {
		name    string
		modify  func(*AuthConfig)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *AuthConfig) {},
			wantErr: false,
		},
		{
			name:    "empty jwt secret",
			modify:  func(c *AuthConfig) { c.JWTSecret = "" },
			wantErr: true,
			errMsg:  "jwt_secret is required",
		},
		{
			name:    "insecure default jwt secret",
			modify:  func(c *AuthConfig) { c.JWTSecret = "your-secret-key-change-in-production" },
			wantErr: true,
			errMsg:  "please set a secure JWT secret",
		},
		{
			name:    "zero jwt expiry",
			modify:  func(c *AuthConfig) { c.JWTExpiry = 0 },
			wantErr: true,
			errMsg:  "jwt_expiry must be positive",
		},
		{
			name:    "zero refresh expiry",
			modify:  func(c *AuthConfig) { c.RefreshExpiry = 0 },
			wantErr: true,
			errMsg:  "refresh_expiry must be positive",
		},
		{
			name:    "zero magic link expiry",
			modify:  func(c *AuthConfig) { c.MagicLinkExpiry = 0 },
			wantErr: true,
			errMsg:  "magic_link_expiry must be positive",
		},
		{
			name:    "zero password reset expiry",
			modify:  func(c *AuthConfig) { c.PasswordResetExpiry = 0 },
			wantErr: true,
			errMsg:  "password_reset_expiry must be positive",
		},
		{
			name:    "zero password min length",
			modify:  func(c *AuthConfig) { c.PasswordMinLen = 0 },
			wantErr: true,
			errMsg:  "password_min_length must be at least 1",
		},
		{
			name:    "bcrypt cost too low",
			modify:  func(c *AuthConfig) { c.BcryptCost = 3 },
			wantErr: true,
			errMsg:  "bcrypt_cost must be between 4 and 31",
		},
		{
			name:    "bcrypt cost too high",
			modify:  func(c *AuthConfig) { c.BcryptCost = 32 },
			wantErr: true,
			errMsg:  "bcrypt_cost must be between 4 and 31",
		},
		{
			name:    "bcrypt cost valid minimum",
			modify:  func(c *AuthConfig) { c.BcryptCost = 4 },
			wantErr: false,
		},
		{
			name:    "bcrypt cost valid maximum",
			modify:  func(c *AuthConfig) { c.BcryptCost = 31 },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validConfig()
			tt.modify(&config)
			err := config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStorageConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  StorageConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid local storage",
			config: StorageConfig{
				Provider:      "local",
				LocalPath:     "./storage",
				MaxUploadSize: 1024 * 1024,
			},
			wantErr: false,
		},
		{
			name: "valid s3 storage",
			config: StorageConfig{
				Provider:      "s3",
				S3Endpoint:    "s3.amazonaws.com",
				S3AccessKey:   "access-key",
				S3SecretKey:   "secret-key",
				S3Bucket:      "my-bucket",
				MaxUploadSize: 1024 * 1024,
			},
			wantErr: false,
		},
		{
			name: "invalid provider",
			config: StorageConfig{
				Provider:      "azure",
				MaxUploadSize: 1024 * 1024,
			},
			wantErr: true,
			errMsg:  "storage provider must be 'local' or 's3'",
		},
		{
			name: "local without path",
			config: StorageConfig{
				Provider:      "local",
				LocalPath:     "",
				MaxUploadSize: 1024 * 1024,
			},
			wantErr: true,
			errMsg:  "local_path is required",
		},
		{
			name: "s3 without endpoint",
			config: StorageConfig{
				Provider:      "s3",
				S3Endpoint:    "",
				S3AccessKey:   "key",
				S3SecretKey:   "secret",
				S3Bucket:      "bucket",
				MaxUploadSize: 1024 * 1024,
			},
			wantErr: true,
			errMsg:  "s3_endpoint is required",
		},
		{
			name: "s3 without access key",
			config: StorageConfig{
				Provider:      "s3",
				S3Endpoint:    "endpoint",
				S3AccessKey:   "",
				S3SecretKey:   "secret",
				S3Bucket:      "bucket",
				MaxUploadSize: 1024 * 1024,
			},
			wantErr: true,
			errMsg:  "s3_access_key is required",
		},
		{
			name: "s3 without secret key",
			config: StorageConfig{
				Provider:      "s3",
				S3Endpoint:    "endpoint",
				S3AccessKey:   "key",
				S3SecretKey:   "",
				S3Bucket:      "bucket",
				MaxUploadSize: 1024 * 1024,
			},
			wantErr: true,
			errMsg:  "s3_secret_key is required",
		},
		{
			name: "s3 without bucket",
			config: StorageConfig{
				Provider:      "s3",
				S3Endpoint:    "endpoint",
				S3AccessKey:   "key",
				S3SecretKey:   "secret",
				S3Bucket:      "",
				MaxUploadSize: 1024 * 1024,
			},
			wantErr: true,
			errMsg:  "s3_bucket is required",
		},
		{
			name: "zero upload size",
			config: StorageConfig{
				Provider:      "local",
				LocalPath:     "./storage",
				MaxUploadSize: 0,
			},
			wantErr: true,
			errMsg:  "max_upload_size must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSecurityConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  SecurityConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with secure token",
			config: SecurityConfig{
				SetupToken: "this-is-a-secure-setup-token-for-testing-purposes",
			},
			wantErr: false,
		},
		{
			name: "empty setup token is valid",
			config: SecurityConfig{
				SetupToken: "",
			},
			wantErr: false,
		},
		{
			name: "insecure default token - changeme",
			config: SecurityConfig{
				SetupToken: "changeme",
			},
			wantErr: true,
			errMsg:  "please set a secure setup token",
		},
		{
			name: "insecure default token - test",
			config: SecurityConfig{
				SetupToken: "test",
			},
			wantErr: true,
			errMsg:  "please set a secure setup token",
		},
		{
			name: "insecure default token - your-secret-setup-token-change-in-production",
			config: SecurityConfig{
				SetupToken: "your-secret-setup-token-change-in-production",
			},
			wantErr: true,
			errMsg:  "please set a secure setup token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEmailConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  EmailConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid smtp config",
			config: EmailConfig{
				Enabled:     true,
				Provider:    "smtp",
				FromAddress: "test@example.com",
				SMTPHost:    "smtp.example.com",
				SMTPPort:    587,
			},
			wantErr: false,
		},
		{
			name: "unconfigured smtp is valid",
			config: EmailConfig{
				Enabled:  true,
				Provider: "smtp",
			},
			wantErr: false,
		},
		{
			name: "invalid provider",
			config: EmailConfig{
				Enabled:     true,
				Provider:    "invalid",
				FromAddress: "test@example.com",
			},
			wantErr: true,
			errMsg:  "invalid email provider",
		},
		{
			name: "empty provider is valid",
			config: EmailConfig{
				Enabled:     true,
				FromAddress: "test@example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEmailConfig_IsConfigured(t *testing.T) {
	tests := []struct {
		name       string
		config     EmailConfig
		configured bool
	}{
		{
			name: "fully configured smtp",
			config: EmailConfig{
				Enabled:     true,
				Provider:    "smtp",
				FromAddress: "test@example.com",
				SMTPHost:    "smtp.example.com",
				SMTPPort:    587,
			},
			configured: true,
		},
		{
			name: "smtp missing host",
			config: EmailConfig{
				Enabled:     true,
				Provider:    "smtp",
				FromAddress: "test@example.com",
				SMTPPort:    587,
			},
			configured: false,
		},
		{
			name: "smtp missing port",
			config: EmailConfig{
				Enabled:     true,
				Provider:    "smtp",
				FromAddress: "test@example.com",
				SMTPHost:    "smtp.example.com",
			},
			configured: false,
		},
		{
			name: "email disabled",
			config: EmailConfig{
				Enabled:     false,
				Provider:    "smtp",
				FromAddress: "test@example.com",
				SMTPHost:    "smtp.example.com",
				SMTPPort:    587,
			},
			configured: false,
		},
		{
			name: "missing from_address",
			config: EmailConfig{
				Enabled:  true,
				Provider: "smtp",
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
			},
			configured: false,
		},
		{
			name: "fully configured sendgrid",
			config: EmailConfig{
				Enabled:        true,
				Provider:       "sendgrid",
				FromAddress:    "test@example.com",
				SendGridAPIKey: "api-key",
			},
			configured: true,
		},
		{
			name: "sendgrid missing api key",
			config: EmailConfig{
				Enabled:     true,
				Provider:    "sendgrid",
				FromAddress: "test@example.com",
			},
			configured: false,
		},
		{
			name: "fully configured mailgun",
			config: EmailConfig{
				Enabled:       true,
				Provider:      "mailgun",
				FromAddress:   "test@example.com",
				MailgunAPIKey: "api-key",
				MailgunDomain: "mg.example.com",
			},
			configured: true,
		},
		{
			name: "mailgun missing domain",
			config: EmailConfig{
				Enabled:       true,
				Provider:      "mailgun",
				FromAddress:   "test@example.com",
				MailgunAPIKey: "api-key",
			},
			configured: false,
		},
		{
			name: "fully configured ses",
			config: EmailConfig{
				Enabled:      true,
				Provider:     "ses",
				FromAddress:  "test@example.com",
				SESAccessKey: "access-key",
				SESSecretKey: "secret-key",
				SESRegion:    "us-east-1",
			},
			configured: true,
		},
		{
			name: "ses missing region",
			config: EmailConfig{
				Enabled:      true,
				Provider:     "ses",
				FromAddress:  "test@example.com",
				SESAccessKey: "access-key",
				SESSecretKey: "secret-key",
			},
			configured: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsConfigured()
			assert.Equal(t, tt.configured, result)
		})
	}
}

func TestAPIConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  APIConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: APIConfig{
				MaxPageSize:     1000,
				MaxTotalResults: 10000,
				DefaultPageSize: 100,
			},
			wantErr: false,
		},
		{
			name: "unlimited values (-1) are valid",
			config: APIConfig{
				MaxPageSize:     -1,
				MaxTotalResults: -1,
				DefaultPageSize: -1,
			},
			wantErr: false,
		},
		{
			name: "zero max page size",
			config: APIConfig{
				MaxPageSize:     0,
				MaxTotalResults: 1000,
				DefaultPageSize: 100,
			},
			wantErr: true,
			errMsg:  "max_page_size must be positive or -1",
		},
		{
			name: "zero max total results",
			config: APIConfig{
				MaxPageSize:     1000,
				MaxTotalResults: 0,
				DefaultPageSize: 100,
			},
			wantErr: true,
			errMsg:  "max_total_results must be positive or -1",
		},
		{
			name: "zero default page size",
			config: APIConfig{
				MaxPageSize:     1000,
				MaxTotalResults: 10000,
				DefaultPageSize: 0,
			},
			wantErr: true,
			errMsg:  "default_page_size must be positive or -1",
		},
		{
			name: "default exceeds max",
			config: APIConfig{
				MaxPageSize:     100,
				MaxTotalResults: 10000,
				DefaultPageSize: 200,
			},
			wantErr: true,
			errMsg:  "default_page_size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestScalingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ScalingConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid local backend",
			config: ScalingConfig{
				Backend: "local",
			},
			wantErr: false,
		},
		{
			name: "valid postgres backend",
			config: ScalingConfig{
				Backend: "postgres",
			},
			wantErr: false,
		},
		{
			name: "valid redis backend",
			config: ScalingConfig{
				Backend:  "redis",
				RedisURL: "redis://localhost:6379",
			},
			wantErr: false,
		},
		{
			name: "invalid backend",
			config: ScalingConfig{
				Backend: "memcached",
			},
			wantErr: true,
			errMsg:  "invalid scaling backend",
		},
		{
			name: "redis without url",
			config: ScalingConfig{
				Backend:  "redis",
				RedisURL: "",
			},
			wantErr: true,
			errMsg:  "redis_url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoggingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  LoggingConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: LoggingConfig{
				ConsoleLevel:  "info",
				ConsoleFormat: "console",
				Backend:       "postgres",
				BatchSize:     100,
			},
			wantErr: false,
		},
		{
			name: "invalid console level",
			config: LoggingConfig{
				ConsoleLevel: "verbose",
			},
			wantErr: true,
			errMsg:  "invalid console_level",
		},
		{
			name: "invalid console format",
			config: LoggingConfig{
				ConsoleFormat: "xml",
			},
			wantErr: true,
			errMsg:  "invalid console_format",
		},
		{
			name: "invalid backend",
			config: LoggingConfig{
				Backend: "cloudwatch",
			},
			wantErr: true,
			errMsg:  "invalid logging backend",
		},
		{
			name: "s3 without bucket",
			config: LoggingConfig{
				Backend:  "s3",
				S3Bucket: "",
			},
			wantErr: true,
			errMsg:  "s3_bucket is required",
		},
		{
			name: "negative batch size",
			config: LoggingConfig{
				BatchSize: -1,
			},
			wantErr: true,
			errMsg:  "batch_size cannot be negative",
		},
		{
			name: "negative buffer size",
			config: LoggingConfig{
				BufferSize: -1,
			},
			wantErr: true,
			errMsg:  "buffer_size cannot be negative",
		},
		{
			name: "negative retention days",
			config: LoggingConfig{
				SystemRetentionDays: -1,
			},
			wantErr: true,
			errMsg:  "system_retention_days cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTracingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  TracingConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "disabled tracing doesn't validate",
			config: TracingConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "valid enabled config",
			config: TracingConfig{
				Enabled:    true,
				Endpoint:   "localhost:4317",
				SampleRate: 0.5,
			},
			wantErr: false,
		},
		{
			name: "enabled without endpoint",
			config: TracingConfig{
				Enabled:  true,
				Endpoint: "",
			},
			wantErr: true,
			errMsg:  "tracing endpoint is required",
		},
		{
			name: "sample rate too low",
			config: TracingConfig{
				Enabled:    true,
				Endpoint:   "localhost:4317",
				SampleRate: -0.1,
			},
			wantErr: true,
			errMsg:  "sample_rate must be between 0.0 and 1.0",
		},
		{
			name: "sample rate too high",
			config: TracingConfig{
				Enabled:    true,
				Endpoint:   "localhost:4317",
				SampleRate: 1.5,
			},
			wantErr: true,
			errMsg:  "sample_rate must be between 0.0 and 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFunctionsConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  FunctionsConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: FunctionsConfig{
				FunctionsDir:       "./functions",
				DefaultTimeout:     30,
				MaxTimeout:         300,
				DefaultMemoryLimit: 128,
				MaxMemoryLimit:     1024,
			},
			wantErr: false,
		},
		{
			name: "empty functions dir",
			config: FunctionsConfig{
				FunctionsDir:       "",
				DefaultTimeout:     30,
				MaxTimeout:         300,
				DefaultMemoryLimit: 128,
				MaxMemoryLimit:     1024,
			},
			wantErr: true,
			errMsg:  "functions_dir cannot be empty",
		},
		{
			name: "default timeout exceeds max",
			config: FunctionsConfig{
				FunctionsDir:       "./functions",
				DefaultTimeout:     600,
				MaxTimeout:         300,
				DefaultMemoryLimit: 128,
				MaxMemoryLimit:     1024,
			},
			wantErr: true,
			errMsg:  "default_timeout",
		},
		{
			name: "default memory exceeds max",
			config: FunctionsConfig{
				FunctionsDir:       "./functions",
				DefaultTimeout:     30,
				MaxTimeout:         300,
				DefaultMemoryLimit: 2048,
				MaxMemoryLimit:     1024,
			},
			wantErr: true,
			errMsg:  "default_memory_limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestJobsConfig_Validate(t *testing.T) {
	validConfig := func() JobsConfig {
		return JobsConfig{
			JobsDir:                   "./jobs",
			WorkerMode:                "embedded",
			EmbeddedWorkerCount:       4,
			MaxConcurrentPerWorker:    5,
			MaxConcurrentPerNamespace: 20,
			DefaultMaxDuration:        5 * time.Minute,
			MaxMaxDuration:            time.Hour,
			DefaultProgressTimeout:    5 * time.Minute,
			PollInterval:              time.Second,
			WorkerHeartbeatInterval:   10 * time.Second,
			WorkerTimeout:             30 * time.Second,
		}
	}

	tests := []struct {
		name    string
		modify  func(*JobsConfig)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *JobsConfig) {},
			wantErr: false,
		},
		{
			name:    "empty jobs dir",
			modify:  func(c *JobsConfig) { c.JobsDir = "" },
			wantErr: true,
			errMsg:  "jobs_dir cannot be empty",
		},
		{
			name:    "invalid worker mode",
			modify:  func(c *JobsConfig) { c.WorkerMode = "distributed" },
			wantErr: true,
			errMsg:  "invalid worker_mode",
		},
		{
			name:    "valid standalone mode",
			modify:  func(c *JobsConfig) { c.WorkerMode = "standalone" },
			wantErr: false,
		},
		{
			name:    "valid disabled mode",
			modify:  func(c *JobsConfig) { c.WorkerMode = "disabled" },
			wantErr: false,
		},
		{
			name:    "negative worker count",
			modify:  func(c *JobsConfig) { c.EmbeddedWorkerCount = -1 },
			wantErr: true,
			errMsg:  "embedded_worker_count cannot be negative",
		},
		{
			name:    "default duration exceeds max",
			modify:  func(c *JobsConfig) { c.DefaultMaxDuration = 2 * time.Hour },
			wantErr: true,
			errMsg:  "default_max_duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validConfig()
			tt.modify(&config)
			err := config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAIConfig_Validate(t *testing.T) {
	validConfig := func() AIConfig {
		return AIConfig{
			ChatbotsDir:          "./chatbots",
			DefaultMaxTokens:     4096,
			QueryTimeout:         30 * time.Second,
			MaxRowsPerQuery:      1000,
			ConversationCacheTTL: 30 * time.Minute,
			MaxConversationTurns: 50,
		}
	}

	tests := []struct {
		name    string
		modify  func(*AIConfig)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *AIConfig) {},
			wantErr: false,
		},
		{
			name:    "empty chatbots dir",
			modify:  func(c *AIConfig) { c.ChatbotsDir = "" },
			wantErr: true,
			errMsg:  "chatbots_dir cannot be empty",
		},
		{
			name:    "zero max tokens",
			modify:  func(c *AIConfig) { c.DefaultMaxTokens = 0 },
			wantErr: true,
			errMsg:  "default_max_tokens must be positive",
		},
		{
			name:    "zero query timeout",
			modify:  func(c *AIConfig) { c.QueryTimeout = 0 },
			wantErr: true,
			errMsg:  "query_timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validConfig()
			tt.modify(&config)
			err := config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMCPConfig_Validate(t *testing.T) {
	validConfig := func() MCPConfig {
		return MCPConfig{
			Enabled:         true,
			BasePath:        "/mcp",
			SessionTimeout:  30 * time.Minute,
			MaxMessageSize:  1024 * 1024,
			RateLimitPerMin: 100,
		}
	}

	tests := []struct {
		name    string
		modify  func(*MCPConfig)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *MCPConfig) {},
			wantErr: false,
		},
		{
			name:    "disabled skips validation",
			modify:  func(c *MCPConfig) { c.Enabled = false; c.BasePath = "" },
			wantErr: false,
		},
		{
			name:    "empty base path when enabled",
			modify:  func(c *MCPConfig) { c.BasePath = "" },
			wantErr: true,
			errMsg:  "mcp base_path cannot be empty when enabled",
		},
		{
			name:    "negative session timeout",
			modify:  func(c *MCPConfig) { c.SessionTimeout = -1 * time.Minute },
			wantErr: true,
			errMsg:  "mcp session_timeout cannot be negative",
		},
		{
			name:    "negative max message size",
			modify:  func(c *MCPConfig) { c.MaxMessageSize = -1 },
			wantErr: true,
			errMsg:  "mcp max_message_size cannot be negative",
		},
		{
			name:    "negative rate limit",
			modify:  func(c *MCPConfig) { c.RateLimitPerMin = -1 },
			wantErr: true,
			errMsg:  "mcp rate_limit_per_min cannot be negative",
		},
		{
			name:    "zero session timeout is valid",
			modify:  func(c *MCPConfig) { c.SessionTimeout = 0 },
			wantErr: false,
		},
		{
			name:    "zero max message size is valid",
			modify:  func(c *MCPConfig) { c.MaxMessageSize = 0 },
			wantErr: false,
		},
		{
			name:    "zero rate limit is valid (unlimited)",
			modify:  func(c *MCPConfig) { c.RateLimitPerMin = 0 },
			wantErr: false,
		},
		{
			name:    "with allowed tools",
			modify:  func(c *MCPConfig) { c.AllowedTools = []string{"query", "storage"} },
			wantErr: false,
		},
		{
			name:    "with allowed resources",
			modify:  func(c *MCPConfig) { c.AllowedResources = []string{"schema://", "storage://"} },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validConfig()
			tt.modify(&config)
			err := config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBranchingConfig_Validate(t *testing.T) {
	validConfig := func() BranchingConfig {
		return BranchingConfig{
			Enabled:              true,
			MaxTotalBranches:     50,
			DefaultDataCloneMode: DataCloneModeSchemaOnly,
			AutoDeleteAfter:      24 * time.Hour,
			DatabasePrefix:       "branch_",
			SeedsPath:            "./seeds",
		}
	}

	tests := []struct {
		name    string
		modify  func(*BranchingConfig)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *BranchingConfig) {},
			wantErr: false,
		},
		{
			name:    "disabled skips validation",
			modify:  func(c *BranchingConfig) { c.Enabled = false; c.DatabasePrefix = "" },
			wantErr: false,
		},
		{
			name:    "negative max total branches",
			modify:  func(c *BranchingConfig) { c.MaxTotalBranches = -1 },
			wantErr: true,
			errMsg:  "branching max_total_branches cannot be negative",
		},
		{
			name:    "invalid data clone mode",
			modify:  func(c *BranchingConfig) { c.DefaultDataCloneMode = "invalid_mode" },
			wantErr: true,
			errMsg:  "branching default_data_clone_mode must be one of",
		},
		{
			name:    "valid full_clone mode",
			modify:  func(c *BranchingConfig) { c.DefaultDataCloneMode = DataCloneModeFullClone },
			wantErr: false,
		},
		{
			name:    "valid seed_data mode",
			modify:  func(c *BranchingConfig) { c.DefaultDataCloneMode = DataCloneModeSeedData },
			wantErr: false,
		},
		{
			name:    "empty data clone mode defaults to schema_only",
			modify:  func(c *BranchingConfig) { c.DefaultDataCloneMode = "" },
			wantErr: false,
		},
		{
			name:    "negative auto delete after",
			modify:  func(c *BranchingConfig) { c.AutoDeleteAfter = -1 * time.Hour },
			wantErr: true,
			errMsg:  "branching auto_delete_after cannot be negative",
		},
		{
			name:    "zero auto delete after is valid (never)",
			modify:  func(c *BranchingConfig) { c.AutoDeleteAfter = 0 },
			wantErr: false,
		},
		{
			name:    "empty database prefix when enabled",
			modify:  func(c *BranchingConfig) { c.DatabasePrefix = "" },
			wantErr: true,
			errMsg:  "branching database_prefix cannot be empty when enabled",
		},
		{
			name:    "empty seeds path gets default",
			modify:  func(c *BranchingConfig) { c.SeedsPath = "" },
			wantErr: false,
		},
		{
			name:    "zero max total branches is valid",
			modify:  func(c *BranchingConfig) { c.MaxTotalBranches = 0 },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validConfig()
			tt.modify(&config)
			err := config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBranchingConfig_SeedsPathDefault(t *testing.T) {
	t.Run("sets default seeds path when empty", func(t *testing.T) {
		config := BranchingConfig{
			Enabled:        true,
			DatabasePrefix: "branch_",
			SeedsPath:      "",
		}

		err := config.Validate()
		require.NoError(t, err)
		assert.Equal(t, "./seeds", config.SeedsPath)
	})

	t.Run("preserves custom seeds path", func(t *testing.T) {
		config := BranchingConfig{
			Enabled:        true,
			DatabasePrefix: "branch_",
			SeedsPath:      "/custom/seeds",
		}

		err := config.Validate()
		require.NoError(t, err)
		assert.Equal(t, "/custom/seeds", config.SeedsPath)
	})
}

func TestGraphQLConfig_Validate(t *testing.T) {
	validConfig := func() GraphQLConfig {
		return GraphQLConfig{
			Enabled:       true,
			MaxDepth:      10,
			MaxComplexity: 1000,
			Introspection: true,
		}
	}

	tests := []struct {
		name    string
		modify  func(*GraphQLConfig)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *GraphQLConfig) {},
			wantErr: false,
		},
		{
			name:    "disabled skips validation",
			modify:  func(c *GraphQLConfig) { c.Enabled = false; c.MaxDepth = 0 },
			wantErr: false,
		},
		{
			name:    "zero max depth when enabled",
			modify:  func(c *GraphQLConfig) { c.MaxDepth = 0 },
			wantErr: true,
			errMsg:  "graphql max_depth must be at least 1",
		},
		{
			name:    "negative max depth",
			modify:  func(c *GraphQLConfig) { c.MaxDepth = -1 },
			wantErr: true,
			errMsg:  "graphql max_depth must be at least 1",
		},
		{
			name:    "zero max complexity when enabled",
			modify:  func(c *GraphQLConfig) { c.MaxComplexity = 0 },
			wantErr: true,
			errMsg:  "graphql max_complexity must be at least 1",
		},
		{
			name:    "negative max complexity",
			modify:  func(c *GraphQLConfig) { c.MaxComplexity = -1 },
			wantErr: true,
			errMsg:  "graphql max_complexity must be at least 1",
		},
		{
			name:    "min valid max depth",
			modify:  func(c *GraphQLConfig) { c.MaxDepth = 1 },
			wantErr: false,
		},
		{
			name:    "min valid max complexity",
			modify:  func(c *GraphQLConfig) { c.MaxComplexity = 1 },
			wantErr: false,
		},
		{
			name:    "introspection disabled",
			modify:  func(c *GraphQLConfig) { c.Introspection = false },
			wantErr: false,
		},
		{
			name:    "high values are valid",
			modify:  func(c *GraphQLConfig) { c.MaxDepth = 100; c.MaxComplexity = 100000 },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validConfig()
			tt.modify(&config)
			err := config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDataCloneModeConstants(t *testing.T) {
	t.Run("constants have expected values", func(t *testing.T) {
		assert.Equal(t, "schema_only", DataCloneModeSchemaOnly)
		assert.Equal(t, "full_clone", DataCloneModeFullClone)
		assert.Equal(t, "seed_data", DataCloneModeSeedData)
	})
}
