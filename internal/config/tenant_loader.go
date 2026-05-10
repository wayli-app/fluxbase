package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// TenantConfigLoader loads and merges tenant-specific configuration overrides
type TenantConfigLoader struct {
	baseConfig *Config
	cache      map[string]*Config // slug -> merged config
}

// NewTenantConfigLoader creates a new tenant configuration loader
func NewTenantConfigLoader(baseConfig *Config) (*TenantConfigLoader, error) {
	loader := &TenantConfigLoader{
		baseConfig: baseConfig,
		cache:      make(map[string]*Config),
	}

	// Load tenant configs from YAML files and inline configs
	if err := loader.loadTenantConfigs(); err != nil {
		return nil, fmt.Errorf("failed to load tenant configs: %w", err)
	}

	return loader, nil
}

// loadTenantConfigs loads tenant configs from the base config and YAML files
func (l *TenantConfigLoader) loadTenantConfigs() error {
	// 1. Load inline tenant configs from base config
	if l.baseConfig.Tenants.Configs != nil {
		for slug, overrides := range l.baseConfig.Tenants.Configs {
			merged, err := l.mergeConfig(overrides, slug)
			if err != nil {
				return fmt.Errorf("failed to merge config for tenant %s: %w", slug, err)
			}
			l.cache[slug] = merged
			log.Debug().Str("slug", slug).Msg("Loaded inline tenant config")
		}
	}

	// 2. Load tenant configs from YAML files
	if l.baseConfig.Tenants.ConfigDir != "" {
		if err := l.loadTenantConfigFiles(); err != nil {
			return fmt.Errorf("failed to load tenant config files: %w", err)
		}
	}

	// 3. Parse tenant-specific environment variables
	// Pattern: FLUXBASE_TENANTS__<SLUG>__<SECTION>__<KEY>
	// Example: FLUXBASE_TENANTS__ACME_CORP__AUTH__JWT_SECRET
	l.parseTenantEnvVars()

	return nil
}

// parseTenantEnvVars scans environment variables for tenant-specific overrides
// Pattern: FLUXBASE_TENANTS__<SLUG>__<SECTION>__<KEY>=value
// Example: FLUXBASE_TENANTS__ACME_CORP__AUTH__JWT_SECRET=secret123
func (l *TenantConfigLoader) parseTenantEnvVars() {
	const prefix = "FLUXBASE_TENANTS__"

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, prefix) {
			continue
		}

		// Parse: FLUXBASE_TENANTS__SLUG__SECTION__KEY=value
		remaining := env[len(prefix):]
		eqIdx := strings.Index(remaining, "=")
		if eqIdx == -1 {
			continue
		}

		pathPart := remaining[:eqIdx]
		value := remaining[eqIdx+1:]

		// Split path: SLUG__SECTION__KEY
		path := strings.Split(pathPart, "__")
		if len(path) < 2 {
			continue
		}

		slug := normalizeSlugFromEnv(path[0])
		if slug == "" {
			continue
		}

		// Get existing merged config or create from base
		merged := l.getOrCreateMergedConfig(slug)

		// Apply the override
		sectionPath := path[1:]
		if err := l.applyEnvOverride(merged, sectionPath, value); err != nil {
			log.Debug().Err(err).Str("slug", slug).Strs("path", sectionPath).Msg("Failed to apply tenant env override")
		} else {
			log.Debug().Str("slug", slug).Strs("path", sectionPath).Msg("Applied tenant env override")
		}
	}
}

// normalizeSlugFromEnv converts ACME_CORP to acme-corp
func normalizeSlugFromEnv(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "_", "-"))
}

// getOrCreateMergedConfig returns existing merged config or creates a new one from base
func (l *TenantConfigLoader) getOrCreateMergedConfig(slug string) *Config {
	if merged, ok := l.cache[slug]; ok {
		return merged
	}

	// Create new merged config from base
	merged := l.deepCopyConfig(l.baseConfig)
	l.cache[slug] = merged
	return merged
}

// applyEnvOverride applies an environment variable override to the merged config
func (l *TenantConfigLoader) applyEnvOverride(merged *Config, path []string, value string) error {
	if len(path) < 1 {
		return fmt.Errorf("invalid path: empty")
	}

	section := strings.ToLower(path[0])
	keyPath := path[1:]

	switch section {
	case "auth":
		return l.applyAuthEnvOverride(&merged.Auth, keyPath, value)
	case "storage":
		return l.applyStorageEnvOverride(&merged.Storage, keyPath, value)
	case "email":
		return l.applyEmailEnvOverride(&merged.Email, keyPath, value)
	case "functions":
		return l.applyFunctionsEnvOverride(&merged.Functions, keyPath, value)
	case "jobs":
		return l.applyJobsEnvOverride(&merged.Jobs, keyPath, value)
	case "ai":
		return l.applyAIEnvOverride(&merged.AI, keyPath, value)
	case "realtime":
		return l.applyRealtimeEnvOverride(&merged.Realtime, keyPath, value)
	case "api":
		return l.applyAPIEnvOverride(&merged.API, keyPath, value)
	case "graphql":
		return l.applyGraphQLEnvOverride(&merged.GraphQL, keyPath, value)
	case "rpc":
		return l.applyRPCEnvOverride(&merged.RPC, keyPath, value)
	default:
		return fmt.Errorf("unknown section: %s", section)
	}
}

// applyAuthEnvOverride applies an override to AuthConfig
func (l *TenantConfigLoader) applyAuthEnvOverride(cfg *AuthConfig, path []string, value string) error {
	if len(path) < 1 {
		return fmt.Errorf("empty auth path")
	}

	key := strings.ToLower(path[0])
	switch key {
	case "jwt_secret", "jwtsecret":
		cfg.JWTSecret = value
	case "jwt_expiry", "jwtexpiry":
		if d, err := parseDuration(value); err == nil {
			cfg.JWTExpiry = d
		}
	case "refresh_expiry", "refreshexpiry":
		if d, err := parseDuration(value); err == nil {
			cfg.RefreshExpiry = d
		}
	case "service_role_ttl", "servicerolettl":
		if d, err := parseDuration(value); err == nil {
			cfg.ServiceRoleTTL = d
		}
	case "anon_ttl", "anonttl":
		if d, err := parseDuration(value); err == nil {
			cfg.AnonTTL = d
		}
	case "magic_link_expiry", "magiclinkexpiry":
		if d, err := parseDuration(value); err == nil {
			cfg.MagicLinkExpiry = d
		}
	case "password_reset_expiry", "passwordresetexpiry":
		if d, err := parseDuration(value); err == nil {
			cfg.PasswordResetExpiry = d
		}
	case "password_min_length", "passwordminlength":
		if i, err := parseInt(value); err == nil {
			cfg.PasswordMinLen = i
		}
	case "bcrypt_cost", "bcryptcost":
		if i, err := parseInt(value); err == nil {
			cfg.BcryptCost = i
		}
	case "signup_enabled", "signupenabled":
		cfg.SignupEnabled = parseBool(value)
	case "magic_link_enabled", "magiclinkenabled":
		cfg.MagicLinkEnabled = parseBool(value)
	default:
		return fmt.Errorf("unknown auth key: %s", key)
	}
	return nil
}

// applyStorageEnvOverride applies an override to StorageConfig
func (l *TenantConfigLoader) applyStorageEnvOverride(cfg *StorageConfig, path []string, value string) error {
	if len(path) < 1 {
		return fmt.Errorf("empty storage path")
	}

	key := strings.ToLower(path[0])
	switch key {
	case "provider":
		cfg.Provider = value
	case "local_path", "localpath":
		cfg.LocalPath = value
	case "s3_bucket", "s3bucket":
		cfg.S3Bucket = value
	case "s3_region", "s3region":
		cfg.S3Region = value
	case "s3_endpoint", "s3endpoint":
		cfg.S3Endpoint = value
	case "s3_access_key", "s3accesskey":
		cfg.S3AccessKey = value
	case "s3_secret_key", "s3secretkey":
		cfg.S3SecretKey = value
	case "max_upload_size", "maxuploadsize":
		if i, err := parseInt64(value); err == nil {
			cfg.MaxUploadSize = i
		}
	default:
		return fmt.Errorf("unknown storage key: %s", key)
	}
	return nil
}

// applyEmailEnvOverride applies an override to EmailConfig
func (l *TenantConfigLoader) applyEmailEnvOverride(cfg *EmailConfig, path []string, value string) error {
	if len(path) < 1 {
		return fmt.Errorf("empty email path")
	}

	key := strings.ToLower(path[0])
	switch key {
	case "enabled":
		cfg.Enabled = parseBool(value)
	case "provider":
		cfg.Provider = value
	case "from_address", "fromaddress":
		cfg.FromAddress = value
	case "from_name", "fromname":
		cfg.FromName = value
	case "reply_to_address", "replytoaddress":
		cfg.ReplyToAddress = value
	case "smtp_host", "smtphost":
		cfg.SMTPHost = value
	case "smtp_port", "smtpport":
		if i, err := parseInt(value); err == nil {
			cfg.SMTPPort = i
		}
	case "smtp_username", "smtpusername":
		cfg.SMTPUsername = value
	case "smtp_password", "smtppassword":
		cfg.SMTPPassword = value
	case "sendgrid_api_key", "sendgridapikey":
		cfg.SendGridAPIKey = value
	case "mailgun_api_key", "mailgunapikey":
		cfg.MailgunAPIKey = value
	case "mailgun_domain", "mailgundomain":
		cfg.MailgunDomain = value
	case "ses_access_key", "sesaccesskey":
		cfg.SESAccessKey = value
	case "ses_secret_key", "sessecretkey":
		cfg.SESSecretKey = value
	case "ses_region", "sesregion":
		cfg.SESRegion = value
	default:
		return fmt.Errorf("unknown email key: %s", key)
	}
	return nil
}

// applyFunctionsEnvOverride applies an override to FunctionsConfig
func (l *TenantConfigLoader) applyFunctionsEnvOverride(cfg *FunctionsConfig, path []string, value string) error {
	if len(path) < 1 {
		return fmt.Errorf("empty functions path")
	}

	key := strings.ToLower(path[0])
	switch key {
	case "enabled":
		cfg.Enabled = parseBool(value)
	case "functions_dir", "functionsdir":
		cfg.FunctionsDir = value
	case "default_timeout", "defaulttimeout", "timeout":
		if i, err := parseInt(value); err == nil {
			cfg.DefaultTimeout = i
		}
	case "max_timeout", "maxtimeout":
		if i, err := parseInt(value); err == nil {
			cfg.MaxTimeout = i
		}
	case "default_memory_limit", "defaultmemorylimit":
		if i, err := parseInt(value); err == nil {
			cfg.DefaultMemoryLimit = i
		}
	case "max_memory_limit", "maxmemorylimit":
		if i, err := parseInt(value); err == nil {
			cfg.MaxMemoryLimit = i
		}
	default:
		return fmt.Errorf("unknown functions key: %s", key)
	}
	return nil
}

// applyJobsEnvOverride applies an override to JobsConfig
func (l *TenantConfigLoader) applyJobsEnvOverride(cfg *JobsConfig, path []string, value string) error {
	if len(path) < 1 {
		return fmt.Errorf("empty jobs path")
	}

	key := strings.ToLower(path[0])
	switch key {
	case "enabled":
		cfg.Enabled = parseBool(value)
	case "jobs_dir", "jobsdir":
		cfg.JobsDir = value
	case "embedded_worker_count", "embeddedworkercount", "worker_count", "workercount":
		if i, err := parseInt(value); err == nil {
			cfg.EmbeddedWorkerCount = i
		}
	case "max_concurrent_per_worker", "maxconcurrentperworker":
		if i, err := parseInt(value); err == nil {
			cfg.MaxConcurrentPerWorker = i
		}
	case "default_max_duration", "defaultmaxduration":
		if d, err := parseDuration(value); err == nil {
			cfg.DefaultMaxDuration = d
		}
	case "poll_interval", "pollinterval":
		if d, err := parseDuration(value); err == nil {
			cfg.PollInterval = d
		}
	default:
		return fmt.Errorf("unknown jobs key: %s", key)
	}
	return nil
}

// applyAIEnvOverride applies an override to AIConfig
func (l *TenantConfigLoader) applyAIEnvOverride(cfg *AIConfig, path []string, value string) error {
	if len(path) < 1 {
		return fmt.Errorf("empty ai path")
	}

	key := strings.ToLower(path[0])
	switch key {
	case "enabled":
		cfg.Enabled = parseBool(value)
	case "chatbots_dir", "chatbotsdir":
		cfg.ChatbotsDir = value
	case "default_max_tokens", "defaultmaxtokens":
		if i, err := parseInt(value); err == nil {
			cfg.DefaultMaxTokens = i
		}
	case "default_model", "defaultmodel":
		cfg.DefaultModel = value
	case "provider_type", "providertype":
		cfg.ProviderType = value
	case "provider_model", "providermodel":
		cfg.ProviderModel = value
	case "embedding_provider", "embeddingprovider":
		cfg.EmbeddingProvider = value
	default:
		return fmt.Errorf("unknown ai key: %s", key)
	}
	return nil
}

// applyRealtimeEnvOverride applies an override to RealtimeConfig
func (l *TenantConfigLoader) applyRealtimeEnvOverride(cfg *RealtimeConfig, path []string, value string) error {
	if len(path) < 1 {
		return fmt.Errorf("empty realtime path")
	}

	key := strings.ToLower(path[0])
	switch key {
	case "enabled":
		cfg.Enabled = parseBool(value)
	case "max_connections", "maxconnections", "max_conns", "maxconns":
		if i, err := parseInt(value); err == nil {
			cfg.MaxConnections = i
		}
	case "max_connections_per_user", "maxconnectionsperuser":
		if i, err := parseInt(value); err == nil {
			cfg.MaxConnectionsPerUser = i
		}
	default:
		return fmt.Errorf("unknown realtime key: %s", key)
	}
	return nil
}

// applyAPIEnvOverride applies an override to APIConfig
func (l *TenantConfigLoader) applyAPIEnvOverride(cfg *APIConfig, path []string, value string) error {
	if len(path) < 1 {
		return fmt.Errorf("empty api path")
	}

	key := strings.ToLower(path[0])
	switch key {
	case "max_page_size", "maxpagesize":
		if i, err := parseInt(value); err == nil {
			cfg.MaxPageSize = i
		}
	case "max_total_results", "maxtotalresults":
		if i, err := parseInt(value); err == nil {
			cfg.MaxTotalResults = i
		}
	case "default_page_size", "defaultpagesize":
		if i, err := parseInt(value); err == nil {
			cfg.DefaultPageSize = i
		}
	case "max_batch_size", "maxbatchsize":
		if i, err := parseInt(value); err == nil {
			cfg.MaxBatchSize = i
		}
	default:
		return fmt.Errorf("unknown api key: %s", key)
	}
	return nil
}

// applyGraphQLEnvOverride applies an override to GraphQLConfig
func (l *TenantConfigLoader) applyGraphQLEnvOverride(cfg *GraphQLConfig, path []string, value string) error {
	if len(path) < 1 {
		return fmt.Errorf("empty graphql path")
	}

	key := strings.ToLower(path[0])
	switch key {
	case "enabled":
		cfg.Enabled = parseBool(value)
	case "max_depth", "maxdepth":
		if i, err := parseInt(value); err == nil {
			cfg.MaxDepth = i
		}
	case "max_complexity", "maxcomplexity":
		if i, err := parseInt(value); err == nil {
			cfg.MaxComplexity = i
		}
	default:
		return fmt.Errorf("unknown graphql key: %s", key)
	}
	return nil
}

// applyRPCEnvOverride applies an override to RPCConfig
func (l *TenantConfigLoader) applyRPCEnvOverride(cfg *RPCConfig, path []string, value string) error {
	if len(path) < 1 {
		return fmt.Errorf("empty rpc path")
	}

	key := strings.ToLower(path[0])
	switch key {
	case "enabled":
		cfg.Enabled = parseBool(value)
	case "procedures_dir", "proceduresdir":
		cfg.ProceduresDir = value
	case "default_max_rows", "defaultmaxrows", "max_rows", "maxrows":
		if i, err := parseInt(value); err == nil {
			cfg.DefaultMaxRows = i
		}
	default:
		return fmt.Errorf("unknown rpc key: %s", key)
	}
	return nil
}

// loadTenantConfigFiles loads tenant configs from the config directory
func (l *TenantConfigLoader) loadTenantConfigFiles() error {
	configDir := l.baseConfig.Tenants.ConfigDir
	if configDir == "" {
		return nil
	}

	// Check if directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		log.Debug().Str("dir", configDir).Msg("Tenant config directory does not exist")
		return nil
	}

	// Read all YAML files from the directory
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("failed to read tenant config directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		filePath := filepath.Join(configDir, entry.Name())
		if err := l.loadTenantConfigFile(filePath); err != nil {
			log.Warn().Err(err).Str("file", filePath).Msg("Failed to load tenant config file")
			continue
		}
	}

	return nil
}

// tenantFileConfig represents the structure of a tenant YAML file
type tenantFileConfig struct {
	Slug     string         `yaml:"slug"`
	Name     string         `yaml:"name"`
	Metadata map[string]any `yaml:"metadata"`
	Config   map[string]any `yaml:"config"`
}

// loadTenantConfigFile loads a single tenant config file
func (l *TenantConfigLoader) loadTenantConfigFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Expand environment variables in the file content
	expanded := l.expandEnvVars(string(data))

	var fileConfig tenantFileConfig
	if err := yaml.Unmarshal([]byte(expanded), &fileConfig); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	if fileConfig.Slug == "" {
		return fmt.Errorf("tenant config file %s missing slug", filePath)
	}

	// Decode raw config map into TenantOverrides using mapstructure
	// (which respects mapstructure tags, matching the same keys used by Viper/env var loading)
	var overrides TenantOverrides
	if len(fileConfig.Config) > 0 {
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName:     "mapstructure",
			Result:      &overrides,
			ErrorUnused: false,
			DecodeHook:  mapstructure.StringToTimeDurationHookFunc(),
		})
		if err != nil {
			return fmt.Errorf("failed to create decoder: %w", err)
		}
		if err := decoder.Decode(fileConfig.Config); err != nil {
			return fmt.Errorf("failed to decode tenant config: %w", err)
		}
	}

	// Merge with base config
	merged, err := l.mergeConfig(overrides, fileConfig.Slug)
	if err != nil {
		return fmt.Errorf("failed to merge config: %w", err)
	}

	l.cache[fileConfig.Slug] = merged
	log.Info().Str("slug", fileConfig.Slug).Str("file", filePath).Msg("Loaded tenant config file")

	return nil
}

// expandEnvVars expands ${VAR_NAME} patterns in the content
func (l *TenantConfigLoader) expandEnvVars(content string) string {
	result := content
	for {
		start := strings.Index(result, "${")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		end += start

		varName := result[start+2 : end]
		varValue := os.Getenv(varName)
		if varValue == "" {
			// Keep the original ${VAR} if not set
			log.Debug().Str("var", varName).Msg("Environment variable not set")
			break
		}

		result = result[:start] + varValue + result[end+1:]
	}
	return result
}

// GetConfigForSlug returns the effective configuration for a tenant by slug.
// For the default tenant (isDefaultTenant=true), falls back to base config (YAML+env).
// For non-default tenants, returns nil if no slug-specific overrides exist,
// ensuring YAML/env config does not leak to non-default tenants.
func (l *TenantConfigLoader) GetConfigForSlug(slug string, isDefaultTenant bool) *Config {
	if merged, ok := l.cache[slug]; ok {
		return merged
	}

	// Only fall back to base config for the default/instance-level tenant
	if isDefaultTenant {
		return l.baseConfig
	}

	// Non-default tenant with no cached overrides: no YAML/env config layer
	return nil
}

// GetBaseConfig returns the base configuration
func (l *TenantConfigLoader) GetBaseConfig() *Config {
	return l.baseConfig
}

// GetLoadedSlugs returns the slugs of all tenants with loaded configurations
func (l *TenantConfigLoader) GetLoadedSlugs() []string {
	slugs := make([]string, 0, len(l.cache))
	for slug := range l.cache {
		slugs = append(slugs, slug)
	}
	return slugs
}

// mergeConfig deep merges tenant overrides with base config
func (l *TenantConfigLoader) mergeConfig(overrides TenantOverrides, slug string) (*Config, error) {
	// Start with a deep copy of base config
	merged := l.deepCopyConfig(l.baseConfig)

	// Apply overrides for each section
	if overrides.Auth != nil {
		merged.Auth = mergeAuthConfig(merged.Auth, *overrides.Auth)
	}
	if overrides.Storage != nil {
		merged.Storage = mergeStorageConfig(merged.Storage, *overrides.Storage)
	}
	if overrides.Email != nil {
		merged.Email = mergeEmailConfig(merged.Email, *overrides.Email)
	}
	if overrides.Functions != nil {
		merged.Functions = mergeFunctionsConfig(merged.Functions, *overrides.Functions)
	}
	if overrides.Jobs != nil {
		merged.Jobs = mergeJobsConfig(merged.Jobs, *overrides.Jobs)
	}
	if overrides.AI != nil {
		merged.AI = mergeAIConfig(merged.AI, *overrides.AI)
	}
	if overrides.Realtime != nil {
		merged.Realtime = mergeRealtimeConfig(merged.Realtime, *overrides.Realtime)
	}
	if overrides.API != nil {
		merged.API = mergeAPIConfig(merged.API, *overrides.API)
	}
	if overrides.GraphQL != nil {
		merged.GraphQL = mergeGraphQLConfig(merged.GraphQL, *overrides.GraphQL)
	}
	if overrides.RPC != nil {
		merged.RPC = mergeRPCConfig(merged.RPC, *overrides.RPC)
	}

	return merged, nil
}

// deepCopyConfig creates a deep copy of the base config
func (l *TenantConfigLoader) deepCopyConfig(base *Config) *Config {
	cpy := new(Config)
	cpy.Server = base.Server
	cpy.Database = base.Database
	cpy.Security = base.Security
	cpy.CORS = base.CORS
	cpy.Migrations = base.Migrations
	cpy.Deno = base.Deno
	cpy.Tracing = base.Tracing
	cpy.Metrics = base.Metrics
	cpy.Branching = base.Branching
	cpy.Scaling = base.Scaling
	cpy.Logging = base.Logging
	cpy.Admin = base.Admin
	cpy.BaseURL = base.BaseURL
	cpy.PublicBaseURL = base.PublicBaseURL
	cpy.Debug = base.Debug
	cpy.EncryptionKey = base.EncryptionKey

	// Deep copy overridable sections
	cpy.Auth = *DeepCopyAuthConfig(&base.Auth)
	cpy.Storage = *DeepCopyStorageConfig(&base.Storage)
	cpy.Email = *DeepCopyEmailConfig(&base.Email)
	cpy.Functions = *DeepCopyFunctionsConfig(&base.Functions)
	cpy.Jobs = *DeepCopyJobsConfig(&base.Jobs)
	cpy.AI = *DeepCopyAIConfig(&base.AI)
	cpy.Realtime = *DeepCopyRealtimeConfig(&base.Realtime)
	cpy.API = *DeepCopyAPIConfig(&base.API)
	cpy.GraphQL = *DeepCopyGraphQLConfig(&base.GraphQL)
	cpy.RPC = *DeepCopyRPCConfig(&base.RPC)
	cpy.Tenants = base.Tenants

	return cpy
}

// Helper functions for parsing env var values

// parseBool parses a boolean string, with a default fallback
func parseBool(s string) bool {
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	return lower == "true" || lower == "1" || lower == "yes" || lower == "on"
}

// parseInt parses an integer string
func parseInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(strings.TrimSpace(s))
}

// parseInt64 parses a 64-bit integer string
func parseInt64(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

// parseDuration parses a duration string (e.g., "15m", "1h", "30s")
func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	return time.ParseDuration(strings.TrimSpace(s))
}
