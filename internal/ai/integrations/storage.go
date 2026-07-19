// Package integrations provides storage and configuration for non-LLM
// tool integrations (web search, URL fetch, etc.) that chatbot
// specialist agents can call.
//
// Storage shape mirrors ai.providers: rows are tenant-scoped, secrets in
// the config JSONB are encrypted at the application layer (see crypto.go),
// and a synthetic FROM_CONFIG row is injected when env/YAML config is set
// so instance-level deployments work the same as multi-tenant dashboard
// deployments.
package integrations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// IntegrationType categorizes what kind of tool the integration provides.
// Stored in `tool_integrations.integration_type` (CHECK constraint).
type IntegrationType string

const (
	IntegrationTypeWebSearch IntegrationType = "web_search"
	IntegrationTypeFetchURL  IntegrationType = "fetch_url"
)

// ProviderName identifies the specific service within an integration type.
// E.g., for web_search: tavily, brave, jina. Stored in
// `tool_integrations.provider` (CHECK constraint).
type ProviderName string

const (
	ProviderTavily      ProviderName = "tavily"
	ProviderBrave       ProviderName = "brave"
	ProviderJina        ProviderName = "jina"
	ProviderSingleFetch ProviderName = "singlefetch"
)

// AllIntegrationTypes is the closed set of valid integration_type values.
// Used by the API layer for validation.
var AllIntegrationTypes = []IntegrationType{
	IntegrationTypeWebSearch,
	IntegrationTypeFetchURL,
}

// AllProviders is the closed set of valid provider values.
var AllProviders = []ProviderName{
	ProviderTavily,
	ProviderBrave,
	ProviderJina,
	ProviderSingleFetch,
}

// IsValidIntegrationType reports whether v is in AllIntegrationTypes.
func IsValidIntegrationType(v string) bool {
	for _, t := range AllIntegrationTypes {
		if string(t) == v {
			return true
		}
	}
	return false
}

// IsValidProvider reports whether v is in AllProviders.
func IsValidProvider(v string) bool {
	for _, p := range AllProviders {
		if string(p) == v {
			return true
		}
	}
	return false
}

// Integration is the in-memory representation of a tool_integrations row.
// Config holds provider-specific fields. Secret fields (api_key, etc.) are
// encrypted at rest and masked in API responses; consumers receive the
// decrypted value via ResolveConfig.
type Integration struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	IntegrationType IntegrationType   `json:"integration_type"`
	Provider        ProviderName      `json:"provider"`
	Config          map[string]string `json:"config"`
	Enabled         bool              `json:"enabled"`
	IsDefault       bool              `json:"is_default"`
	LastTestedAt    *time.Time        `json:"last_tested_at,omitempty"`
	LastTestStatus  string            `json:"last_test_status,omitempty"`
	LastTestError   string            `json:"last_test_error,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	CreatedBy       string            `json:"created_by,omitempty"`

	// FromConfig marks the synthetic row injected when env/YAML config is
	// set. Read-only in the API/UI. Empty (false) for DB-stored rows.
	FromConfig bool   `json:"from_config,omitempty"`
	ReadOnly   bool   `json:"read_only,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
}

// SecretFields returns the config keys that hold sensitive values for
// this integration. These fields are encrypted at write time, decrypted
// on internal reads, and masked on API responses.
//
// Per-integration-type secret lists (currently just api_key for
// everything). Extend here when a new provider has additional secret
// fields (e.g., oauth_refresh_token).
func (i *Integration) SecretFields() []string {
	return []string{"api_key"}
}

// Storage is the data-access layer for tool_integrations. Mirrors the
// shape of ai.provider_storage.go: tenant-aware CRUD + a synthetic
// FROM_CONFIG row when env/YAML is set.
type Storage struct {
	database.TenantAware
	encryptionKey []byte
	config        *EnvConfig // env/YAML config; nil when not set
}

// NewStorage constructs a new Storage. encryptionKey is the master key
// from cfg.EncryptionKeyBytes; may be nil when encryption is not
// configured (secrets stored plaintext in that case — logged at call
// sites, matches existing AI provider behavior).
func NewStorage(db *database.Connection, encryptionKey []byte, envConfig *EnvConfig) *Storage {
	return &Storage{
		TenantAware:   database.TenantAware{DB: db},
		encryptionKey: encryptionKey,
		config:        envConfig,
	}
}

// SetEnvConfig replaces the env/YAML config at runtime. Used by hot-reload
// paths when settings change.
func (s *Storage) SetEnvConfig(cfg *EnvConfig) {
	s.config = cfg
}

// EnvConfig returns the current env/YAML config (may be nil).
func (s *Storage) EnvConfig() *EnvConfig {
	return s.config
}

// EncryptionKey returns the master encryption key (may be nil).
// Exposed so the API handler can pass it to crypto helpers when building
// responses.
func (s *Storage) EncryptionKey() []byte {
	return s.encryptionKey
}

// scanIntegration populates an Integration from a pgx row. Columns must
// be in the order returned by the standard SELECT in the various query
// helpers below.
func scanIntegration(row pgx.Row) (*Integration, error) {
	i := &Integration{Config: map[string]string{}}
	// config is jsonb — scan into a map[string]string directly. pgbouncer-
	// safe because pgx handles the JSONB → map decode.
	var configMap map[string]any
	err := row.Scan(
		&i.ID, &i.Name, &i.IntegrationType, &i.Provider, &configMap,
		&i.Enabled, &i.IsDefault, &i.LastTestedAt, &i.LastTestStatus, &i.LastTestError,
		&i.CreatedAt, &i.UpdatedAt, &i.CreatedBy, &i.TenantID,
	)
	if err != nil {
		return nil, err
	}
	for k, v := range configMap {
		if s, ok := v.(string); ok {
			i.Config[k] = s
		}
	}
	return i, nil
}

// standardColumns is the SELECT column list used by all read queries.
// Keep in sync with scanIntegration.
const standardColumns = `
	id, name, integration_type, provider, config,
	enabled, is_default, last_tested_at, last_test_status, last_test_error,
	created_at, updated_at, created_by, tenant_id
`

// GetIntegration returns a single integration by ID. Decrypts secrets in
// Config. Returns (nil, nil) when not found — callers should nil-check.
func (s *Storage) GetIntegration(ctx context.Context, id string) (*Integration, error) {
	query := `SELECT ` + standardColumns + ` FROM ai.tool_integrations WHERE id = $1`
	var integration *Integration
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, query, id)
		i, err := scanIntegration(row)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		integration = i
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("GetIntegrations: %w", err)
	}
	if integration == nil {
		return nil, nil
	}
	s.decryptConfigInPlace(integration)
	return integration, nil
}

// GetIntegrationByName returns a single integration by name. Decrypts secrets.
func (s *Storage) GetIntegrationByName(ctx context.Context, name string) (*Integration, error) {
	query := `SELECT ` + standardColumns + ` FROM ai.tool_integrations WHERE name = $1`
	var integration *Integration
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, query, name)
		i, err := scanIntegration(row)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		integration = i
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("GetIntegrationsByName: %w", err)
	}
	if integration == nil {
		return nil, nil
	}
	s.decryptConfigInPlace(integration)
	return integration, nil
}

// GetDefaultIntegration returns the default integration for the given
// integration_type, preferring the synthetic FROM_CONFIG row when env/YAML
// is set. Returns (nil, nil) when none configured — callers fall back to
// disabling the corresponding agent.
//
// resolution order:
//  1. Synthetic FROM_CONFIG row (if env/YAML is set and matches the type)
//  2. DB row where is_default=true for this tenant + type
//  3. Any enabled DB row for this tenant + type (last resort)
func (s *Storage) GetDefaultIntegration(ctx context.Context, integrationType IntegrationType) (*Integration, error) {
	// 1. Synthetic FROM_CONFIG
	if s.config != nil && s.config.HasIntegration(integrationType) {
		return s.config.ToIntegration(integrationType), nil
	}

	// 2. DB default for this type
	query := `
		SELECT ` + standardColumns + ` FROM ai.tool_integrations
		WHERE integration_type = $1 AND is_default = true AND enabled = true
	`
	var integration *Integration
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, query, integrationType)
		i, err := scanIntegration(row)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		integration = i
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("GetDefaultIntegration: %w", err)
	}
	if integration != nil {
		s.decryptConfigInPlace(integration)
		return integration, nil
	}

	// 3. Any enabled row for this type
	query = `
		SELECT ` + standardColumns + ` FROM ai.tool_integrations
		WHERE integration_type = $1 AND enabled = true
		ORDER BY created_at ASC LIMIT 1
	`
	err = s.WithTenant(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, query, integrationType)
		i, err := scanIntegration(row)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		integration = i
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("GetDefaultIntegration (any enabled): %w", err)
	}
	if integration != nil {
		s.decryptConfigInPlace(integration)
	}
	return integration, nil
}

// ListIntegrations returns all integrations, optionally filtered by type.
// Includes the synthetic FROM_CONFIG row when env/YAML config is set.
// Secrets in returned rows are decrypted for internal use; the API layer
// is responsible for masking.
func (s *Storage) ListIntegrations(ctx context.Context, integrationType *IntegrationType) ([]*Integration, error) {
	var (
		query string
		args  []any
	)
	if integrationType != nil {
		query = `SELECT ` + standardColumns + ` FROM ai.tool_integrations WHERE integration_type = $1 ORDER BY created_at DESC`
		args = []any{*integrationType}
	} else {
		query = `SELECT ` + standardColumns + ` FROM ai.tool_integrations ORDER BY created_at DESC`
	}

	var out []*Integration
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			i, err := scanIntegration(rows)
			if err != nil {
				return err
			}
			out = append(out, i)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("ListIntegrations: %w", err)
	}

	// Decrypt secrets on read
	for _, i := range out {
		s.decryptConfigInPlace(i)
	}

	// Prepend synthetic FROM_CONFIG row when env/YAML is set
	if s.config != nil {
		if integrationType == nil {
			for _, t := range AllIntegrationTypes {
				if i := s.config.ToIntegration(t); i != nil {
					out = append([]*Integration{i}, out...)
				}
			}
		} else if i := s.config.ToIntegration(*integrationType); i != nil {
			out = append([]*Integration{i}, out...)
		}
	}

	return out, nil
}

// CreateIntegrationRequest is the input shape for CreateIntegration.
// API handlers translate fiber payloads into this struct.
type CreateIntegrationRequest struct {
	Name            string            `json:"name"`
	IntegrationType IntegrationType   `json:"integration_type"`
	Provider        ProviderName      `json:"provider"`
	Config          map[string]string `json:"config"`
	Enabled         bool              `json:"enabled"`
	IsDefault       bool              `json:"is_default"`
	CreatedBy       string            `json:"created_by,omitempty"`
}

// CreateIntegration inserts a new row. Secrets in Config are encrypted
// before persistence. When IsDefault=true, the existing default for the
// same (integration_type, tenant_id) is cleared first (DB partial unique
// index would otherwise reject the insert).
func (s *Storage) CreateIntegration(ctx context.Context, tenantID string, req *CreateIntegrationRequest) (*Integration, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if !IsValidIntegrationType(string(req.IntegrationType)) {
		return nil, fmt.Errorf("invalid integration_type: %q", req.IntegrationType)
	}
	if !IsValidProvider(string(req.Provider)) {
		return nil, fmt.Errorf("invalid provider: %q", req.Provider)
	}

	integration := &Integration{
		ID:              uuid.New().String(),
		Name:            req.Name,
		IntegrationType: req.IntegrationType,
		Provider:        req.Provider,
		Config:          req.Config,
		Enabled:         req.Enabled,
		IsDefault:       req.IsDefault,
		CreatedBy:       req.CreatedBy,
		TenantID:        tenantID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if integration.Config == nil {
		integration.Config = map[string]string{}
	}

	// Encrypt secrets before INSERT
	encryptedConfig, err := s.encryptConfig(integration)
	if err != nil {
		return nil, err
	}

	// ponytail: defensive NULL for created_by — the column is uuid nullable,
	// but empty string trips "invalid input syntax for type uuid" on insert.
	// Handler should populate from auth context, but storage shouldn't trust it.
	var createdBy interface{}
	if integration.CreatedBy != "" {
		createdBy = integration.CreatedBy
	}

	// Clear prior default for this type within the same transaction
	// (the partial unique index enforces one-default-per-type-per-tenant).
	err = database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		if integration.IsDefault {
			if _, err := tx.Exec(
				ctx,
				`UPDATE ai.tool_integrations SET is_default = false WHERE integration_type = $1`,
				integration.IntegrationType,
			); err != nil {
				return fmt.Errorf("failed to clear prior default: %w", err)
			}
		}
		_, err := tx.Exec(
			ctx, `
			INSERT INTO ai.tool_integrations (
				id, name, integration_type, provider, config,
				enabled, is_default, created_at, updated_at, created_by, tenant_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`,
			integration.ID, integration.Name, integration.IntegrationType, integration.Provider, encryptedConfig,
			integration.Enabled, integration.IsDefault,
			integration.CreatedAt, integration.UpdatedAt, createdBy,
			database.TenantOrNil(tenantID),
		)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("CreateIntegration: %w", err)
	}

	log.Info().
		Str("id", integration.ID).
		Str("name", integration.Name).
		Str("integration_type", string(integration.IntegrationType)).
		Str("provider", string(integration.Provider)).
		Msg("Created tool integration")

	return integration, nil
}

// UpdateIntegrationRequest is the input shape for UpdateIntegration.
// All fields optional (pointer); nil means "don't change".
type UpdateIntegrationRequest struct {
	Name      *string           `json:"name"`
	Config    map[string]string `json:"config"`
	Enabled   *bool             `json:"enabled"`
	IsDefault *bool             `json:"is_default"`
}

// UpdateIntegration updates an existing row. When Config is provided,
// masked secrets ("***masked***") preserve the existing encrypted value;
// non-masked values overwrite. This matches the AI provider handler's
// behavior so admins can change non-secret fields without re-entering
// the API key.
func (s *Storage) UpdateIntegration(ctx context.Context, tenantID, id string, req *UpdateIntegrationRequest) (*Integration, error) {
	existing, err := s.GetIntegration(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("UpdateIntegration: load existing: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("UpdateIntegration: integration %q not found", id)
	}
	if existing.ReadOnly {
		return nil, errors.New("UpdateIntegration: integration is read-only (configured via env/YAML)")
	}

	// Merge request into existing
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.IsDefault != nil {
		existing.IsDefault = *req.IsDefault
	}
	if req.Config != nil {
		// Preserve masked secrets from existing
		merged := map[string]string{}
		for k, v := range existing.Config {
			merged[k] = v
		}
		for k, v := range req.Config {
			if IsMasked(v) {
				// Keep the existing decrypted value (will be re-encrypted below)
				continue
			}
			merged[k] = v
		}
		existing.Config = merged
	}
	existing.UpdatedAt = time.Now()

	// Re-encrypt secrets
	encryptedConfig, err := s.encryptConfig(existing)
	if err != nil {
		return nil, err
	}

	err = database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		// Clear prior default if this row is becoming the new default
		if existing.IsDefault {
			if _, err := tx.Exec(
				ctx,
				`UPDATE ai.tool_integrations SET is_default = false WHERE integration_type = $1 AND id <> $2`,
				existing.IntegrationType, existing.ID,
			); err != nil {
				return fmt.Errorf("failed to clear prior default: %w", err)
			}
		}
		_, err := tx.Exec(
			ctx, `
			UPDATE ai.tool_integrations SET
				name = $2, config = $3, enabled = $4, is_default = $5, updated_at = $6
			WHERE id = $1
		`,
			existing.ID, existing.Name, encryptedConfig,
			existing.Enabled, existing.IsDefault, existing.UpdatedAt,
		)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("UpdateIntegration: %w", err)
	}

	return existing, nil
}

// DeleteIntegration removes a row. Refuses to delete read-only (FROM_CONFIG)
// rows — those are derived from env/YAML and can only be removed by
// clearing the env vars.
func (s *Storage) DeleteIntegration(ctx context.Context, tenantID, id string) error {
	existing, err := s.GetIntegration(ctx, id)
	if err != nil {
		return fmt.Errorf("DeleteIntegration: load: %w", err)
	}
	if existing == nil {
		return nil // idempotent
	}
	if existing.ReadOnly {
		return errors.New("DeleteIntegration: integration is read-only (configured via env/YAML)")
	}
	return database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM ai.tool_integrations WHERE id = $1`, id)
		return err
	})
}

// UpdateTestStatus records the result of a test-connection call. Used by
// the /test endpoint to surface health info in the admin UI.
func (s *Storage) UpdateTestStatus(ctx context.Context, tenantID, id, status, errMsg string) error {
	now := time.Now()
	return database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE ai.tool_integrations SET
				last_tested_at = $2, last_test_status = $3, last_test_error = $4
			WHERE id = $1
		`, id, now, status, errMsg)
		return err
	})
}

// encryptConfig returns a JSONB-safe copy of the integration's Config
// with all secret fields encrypted (and tagged with SecretPrefix).
// Non-secret fields pass through unchanged.
func (s *Storage) encryptConfig(i *Integration) (map[string]any, error) {
	if i.Config == nil {
		return map[string]any{}, nil
	}
	secrets := map[string]bool{}
	for _, k := range i.SecretFields() {
		secrets[k] = true
	}
	out := make(map[string]any, len(i.Config))
	for k, v := range i.Config {
		if !secrets[k] {
			out[k] = v
			continue
		}
		// Already encrypted (e.g., admin passed through a masked value we
		// preserved)? Skip re-encryption.
		if IsEncrypted(v) {
			out[k] = v
			continue
		}
		encrypted, err := EncryptSecret(v, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt field %q: %w", k, err)
		}
		out[k] = encrypted
	}
	return out, nil
}

// decryptConfigInPlace decrypts all secret fields in i.Config, in place.
// Values without SecretPrefix are left unchanged (lazy migration path).
func (s *Storage) decryptConfigInPlace(i *Integration) {
	if i.Config == nil || len(s.encryptionKey) == 0 {
		return
	}
	for _, k := range i.SecretFields() {
		v, ok := i.Config[k]
		if !ok || !IsEncrypted(v) {
			continue
		}
		plaintext, err := DecryptIfEncrypted(v, s.encryptionKey)
		if err != nil {
			log.Warn().Err(err).Str("integration_id", i.ID).Str("field", k).
				Msg("Failed to decrypt integration secret; leaving ciphertext in place")
			continue
		}
		i.Config[k] = plaintext
	}
}
