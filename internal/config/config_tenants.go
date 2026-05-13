package config

import "time"

// TenantsConfig contains tenant configuration settings
type TenantsConfig struct {
	Enabled        bool                       `mapstructure:"enabled"`
	DatabasePrefix string                     `mapstructure:"database_prefix"`
	MaxTenants     int                        `mapstructure:"max_tenants"`
	Pool           TenantPoolConfig           `mapstructure:"pool"`
	Migrations     TenantMigrationsConfig     `mapstructure:"migrations"`
	Declarative    TenantDeclarativeConfig    `mapstructure:"declarative"`
	Default        DefaultTenantConfig        `mapstructure:"default"`
	Configs        map[string]TenantOverrides `mapstructure:"configs"`
	ConfigDir      string                     `mapstructure:"config_dir"`
}

// TenantPoolConfig contains connection pool settings for tenant databases
type TenantPoolConfig struct {
	MaxTotalConnections int32         `mapstructure:"max_total_connections"`
	EvictionAge         time.Duration `mapstructure:"eviction_age"`
}

// TenantMigrationsConfig contains migration settings for tenant databases
type TenantMigrationsConfig struct {
	CheckInterval time.Duration `mapstructure:"check_interval"`
	OnCreate      bool          `mapstructure:"on_create"`
	OnAccess      bool          `mapstructure:"on_access"`
	Background    bool          `mapstructure:"background"`
}

// TenantDeclarativeConfig contains declarative schema settings for tenant databases
// This allows tenants to define their own schemas declaratively using SQL files
type TenantDeclarativeConfig struct {
	// Enabled controls whether tenant-specific declarative schemas are applied
	Enabled bool `mapstructure:"enabled"`
	// SchemaDir is the directory containing tenant schema files
	// Structure: {SchemaDir}/{tenant-slug}/public.sql
	// Example: schemas/acme-corp/public.sql
	SchemaDir string `mapstructure:"schema_dir"`
	// OnCreate applies declarative schemas when a tenant database is created
	OnCreate bool `mapstructure:"on_create"`
	// OnStartup applies declarative schemas on server startup (for existing tenants)
	OnStartup bool `mapstructure:"on_startup"`
	// AllowDestructive allows destructive schema changes (DROP, ALTER)
	AllowDestructive bool `mapstructure:"allow_destructive"`
}

// TenantOverrides holds configuration overrides for a specific tenant
// Only user-facing sections can be overridden; infrastructure sections remain global
type TenantOverrides struct {
	Auth      *AuthConfig      `mapstructure:"auth"`
	Storage   *StorageConfig   `mapstructure:"storage"`
	Email     *EmailConfig     `mapstructure:"email"`
	Functions *FunctionsConfig `mapstructure:"functions"`
	Jobs      *JobsConfig      `mapstructure:"jobs"`
	AI        *AIConfig        `mapstructure:"ai"`
	Realtime  *RealtimeConfig  `mapstructure:"realtime"`
	API       *APIConfig       `mapstructure:"api"`
	GraphQL   *GraphQLConfig   `mapstructure:"graphql"`
	RPC       *RPCConfig       `mapstructure:"rpc"`
}

// DefaultTenantConfig contains default tenant settings
type DefaultTenantConfig struct {
	Name           string `mapstructure:"name"`
	AnonKey        string `mapstructure:"anon_key"`
	ServiceKey     string `mapstructure:"service_key"`
	AnonKeyFile    string `mapstructure:"anon_key_file"`
	ServiceKeyFile string `mapstructure:"service_key_file"`
}
