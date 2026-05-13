package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// TracingConfig contains OpenTelemetry tracing settings
type TracingConfig struct {
	Enabled     bool    `mapstructure:"enabled"`      // Enable OpenTelemetry tracing
	Endpoint    string  `mapstructure:"endpoint"`     // OTLP endpoint (e.g., "localhost:4317")
	ServiceName string  `mapstructure:"service_name"` // Service name for traces (default: "fluxbase")
	Environment string  `mapstructure:"environment"`  // Environment name (development, staging, production)
	SampleRate  float64 `mapstructure:"sample_rate"`  // Sample rate 0.0-1.0 (1.0 = 100%)
	Insecure    bool    `mapstructure:"insecure"`     // Use insecure connection (for local dev)
}

// MetricsConfig contains Prometheus metrics settings
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"` // Enable Prometheus metrics endpoint
	Port    int    `mapstructure:"port"`    // Port for metrics server (default: 9090)
	Path    string `mapstructure:"path"`    // Path for metrics endpoint (default: /metrics)
}

// LoggingConfig contains central logging configuration
type LoggingConfig struct {
	// Console output settings
	ConsoleEnabled bool   `mapstructure:"console_enabled"` // Enable console output (default: true)
	ConsoleLevel   string `mapstructure:"console_level"`   // Minimum level for console: trace, debug, info, warn, error
	ConsoleFormat  string `mapstructure:"console_format"`  // Output format: json or console

	// Backend settings
	Backend string `mapstructure:"backend"` // Primary backend: postgres (default), s3, local, timescaledb, loki, elasticsearch, opensearch, clickhouse

	// S3 backend settings (when backend is "s3")
	S3Bucket string `mapstructure:"s3_bucket"` // S3 bucket for logs
	S3Prefix string `mapstructure:"s3_prefix"` // Prefix for log objects (default: "logs")

	// Local backend settings (when backend is "local")
	LocalPath string `mapstructure:"local_path"` // Directory for log files (default: "./logs")

	// TimescaleDB settings (when backend is "timescaledb")
	TimescaleDBEnabled       bool          `mapstructure:"timescaledb_enabled"`
	TimescaleDBCompression   bool          `mapstructure:"timescaledb_compression"`
	TimescaleDBCompressAfter time.Duration `mapstructure:"timescaledb_compress_after"` // Compress after this duration (default: 7d)
	TimescaleDBRetainAfter   time.Duration `mapstructure:"timescaledb_retain_after"`   // Drop chunks older than this (default: 90d)

	// Loki settings (when backend is "loki")
	LokiURL      string   `mapstructure:"loki_url"`       // Loki server URL (required)
	LokiUsername string   `mapstructure:"loki_username"`  // Username for basic auth
	LokiPassword string   `mapstructure:"loki_password"`  // Password for basic auth
	LokiTenantID string   `mapstructure:"loki_tenant_id"` // Tenant ID for multi-tenant Loki
	LokiLabels   []string `mapstructure:"loki_labels"`    // Static labels to add to all logs

	// Elasticsearch settings (when backend is "elasticsearch")
	ElasticsearchURLs     []string `mapstructure:"elasticsearch_urls"`     // Elasticsearch node URLs
	ElasticsearchUsername string   `mapstructure:"elasticsearch_username"` // Username for basic auth
	ElasticsearchPassword string   `mapstructure:"elasticsearch_password"` // Password for basic auth
	ElasticsearchIndex    string   `mapstructure:"elasticsearch_index"`    // Index name pattern (default: "fluxbase-logs")
	ElasticsearchVersion  int      `mapstructure:"elasticsearch_version"`  // Major version: 8 or 9 (default: 8)

	// OpenSearch settings (when backend is "opensearch")
	OpenSearchURLs     []string `mapstructure:"opensearch_urls"`     // OpenSearch node URLs
	OpenSearchUsername string   `mapstructure:"opensearch_username"` // Username for basic auth
	OpenSearchPassword string   `mapstructure:"opensearch_password"` // Password for basic auth
	OpenSearchIndex    string   `mapstructure:"opensearch_index"`    // Index name pattern (default: "fluxbase-logs")
	OpenSearchVersion  int      `mapstructure:"opensearch_version"`  // Major version (default: 2)

	// ClickHouse settings (when backend is "clickhouse")
	ClickHouseAddresses []string `mapstructure:"clickhouse_addresses"` // ClickHouse node addresses (default: ["localhost:9000"])
	ClickHouseUsername  string   `mapstructure:"clickhouse_username"`  // Username (default: "default")
	ClickHousePassword  string   `mapstructure:"clickhouse_password"`  // Password
	ClickHouseDatabase  string   `mapstructure:"clickhouse_database"`  // Database name (default: "fluxbase")
	ClickHouseTable     string   `mapstructure:"clickhouse_table"`     // Table name (default: "logs")
	ClickHouseTTL       int      `mapstructure:"clickhouse_ttl_days"`  // TTL in days (default: 30)

	// Batching settings
	BatchSize     int           `mapstructure:"batch_size"`     // Number of entries per batch (default: 100)
	FlushInterval time.Duration `mapstructure:"flush_interval"` // Max time before flushing (default: 1s)
	BufferSize    int           `mapstructure:"buffer_size"`    // Async buffer size (default: 10000)

	// PubSub notifications (for realtime streaming)
	PubSubEnabled bool `mapstructure:"pubsub_enabled"` // Enable PubSub notifications for execution logs

	// Retention settings (days, 0 = keep forever)
	SystemRetentionDays    int `mapstructure:"system_retention_days"`    // App/system logs (default: 7)
	HTTPRetentionDays      int `mapstructure:"http_retention_days"`      // HTTP access logs (default: 30)
	SecurityRetentionDays  int `mapstructure:"security_retention_days"`  // Security/audit logs (default: 90)
	ExecutionRetentionDays int `mapstructure:"execution_retention_days"` // Function/job/RPC logs (default: 30)
	AIRetentionDays        int `mapstructure:"ai_retention_days"`        // AI query audit logs (default: 30)

	// Retention service settings
	RetentionEnabled       bool          `mapstructure:"retention_enabled"`        // Enable retention cleanup (default: true)
	RetentionCheckInterval time.Duration `mapstructure:"retention_check_interval"` // Interval between cleanup checks (default: 24h)

	// Custom categories
	CustomCategories    []string `mapstructure:"custom_categories"`     // List of allowed custom category names
	CustomRetentionDays int      `mapstructure:"custom_retention_days"` // Retention for custom categories (default: 30)
}

// Validate validates tracing configuration
func (tc *TracingConfig) Validate() error {
	if !tc.Enabled {
		return nil // No validation needed if tracing is disabled
	}

	// Validate endpoint
	if tc.Endpoint == "" {
		return fmt.Errorf("tracing endpoint is required when tracing is enabled")
	}

	// Validate sample rate
	if tc.SampleRate < 0 || tc.SampleRate > 1 {
		return fmt.Errorf("tracing sample_rate must be between 0.0 and 1.0, got: %f", tc.SampleRate)
	}

	// Warn if sample rate is 100% in production
	if tc.Environment == "production" && tc.SampleRate >= 1.0 {
		log.Warn().Msg("Tracing sample_rate is 100% in production - consider reducing to lower overhead")
	}

	return nil
}

// Validate validates metrics configuration
func (mc *MetricsConfig) Validate() error {
	if !mc.Enabled {
		return nil // No validation needed if metrics is disabled
	}

	// Validate port
	if mc.Port < 1 || mc.Port > 65535 {
		return fmt.Errorf("metrics port must be between 1 and 65535, got: %d", mc.Port)
	}

	// Validate path
	if mc.Path == "" {
		return fmt.Errorf("metrics path cannot be empty")
	}
	if !strings.HasPrefix(mc.Path, "/") {
		return fmt.Errorf("metrics path must start with '/', got: %s", mc.Path)
	}

	return nil
}

// Validate validates logging configuration
func (lc *LoggingConfig) Validate() error {
	// Validate console level
	validLevels := []string{"trace", "debug", "info", "warn", "error"}
	levelValid := false
	for _, level := range validLevels {
		if lc.ConsoleLevel == level {
			levelValid = true
			break
		}
	}
	if !levelValid && lc.ConsoleLevel != "" {
		return fmt.Errorf("invalid console_level: %s (must be one of: %v)", lc.ConsoleLevel, validLevels)
	}

	// Validate console format
	if lc.ConsoleFormat != "" && lc.ConsoleFormat != "json" && lc.ConsoleFormat != "console" {
		return fmt.Errorf("invalid console_format: %s (must be 'json' or 'console')", lc.ConsoleFormat)
	}

	// Validate backend
	validBackends := []string{"postgres", "postgres-timescaledb", "timescaledb", "s3", "local", "elasticsearch", "opensearch", "clickhouse", "loki"}
	backendValid := false
	for _, backend := range validBackends {
		if lc.Backend == backend {
			backendValid = true
			break
		}
	}
	if !backendValid && lc.Backend != "" {
		return fmt.Errorf("invalid logging backend: %s (must be one of: %v)", lc.Backend, validBackends)
	}

	// Validate S3 settings when backend is s3
	if lc.Backend == "s3" && lc.S3Bucket == "" {
		return fmt.Errorf("s3_bucket is required when logging backend is 's3'")
	}

	// Validate batching settings
	if lc.BatchSize < 0 {
		return fmt.Errorf("batch_size cannot be negative, got: %d", lc.BatchSize)
	}
	if lc.FlushInterval < 0 {
		return fmt.Errorf("flush_interval cannot be negative, got: %v", lc.FlushInterval)
	}
	if lc.BufferSize < 0 {
		return fmt.Errorf("buffer_size cannot be negative, got: %d", lc.BufferSize)
	}

	// Validate retention settings
	if lc.SystemRetentionDays < 0 {
		return fmt.Errorf("system_retention_days cannot be negative, got: %d", lc.SystemRetentionDays)
	}
	if lc.HTTPRetentionDays < 0 {
		return fmt.Errorf("http_retention_days cannot be negative, got: %d", lc.HTTPRetentionDays)
	}
	if lc.SecurityRetentionDays < 0 {
		return fmt.Errorf("security_retention_days cannot be negative, got: %d", lc.SecurityRetentionDays)
	}
	if lc.ExecutionRetentionDays < 0 {
		return fmt.Errorf("execution_retention_days cannot be negative, got: %d", lc.ExecutionRetentionDays)
	}
	if lc.AIRetentionDays < 0 {
		return fmt.Errorf("ai_retention_days cannot be negative, got: %d", lc.AIRetentionDays)
	}

	// Warn about short retention periods for security logs
	if lc.SecurityRetentionDays > 0 && lc.SecurityRetentionDays < 30 {
		log.Warn().Int("security_retention_days", lc.SecurityRetentionDays).Msg("Security log retention is less than 30 days - consider increasing for compliance")
	}

	return nil
}
