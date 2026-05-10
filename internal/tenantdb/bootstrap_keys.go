package tenantdb

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/keys"
)

// EnsureDefaultTenantAndKeys ensures the default tenant and service keys exist.
func EnsureDefaultTenantAndKeys(pool *pgxpool.Pool, cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var tenantID uuid.UUID
	var tenantExists bool

	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = pool.QueryRow(
			ctx,
			"SELECT id, true FROM platform.tenants WHERE slug = 'default' AND deleted_at IS NULL",
		).Scan(&tenantID, &tenantExists)
		if err == nil {
			break
		}
		if isNoRowsError(err) {
			err = nil
			break
		}
		if strings.Contains(err.Error(), "no rows in result set") {
			err = nil
			break
		}
		if attempt < 2 {
			log.Warn().Err(err).Msg("Retrying default tenant check due to connection error")
			time.Sleep(500 * time.Millisecond)
		}
	}
	if err != nil && !isNoRowsError(err) {
		return fmt.Errorf("failed to check for default tenant: %w", err)
	}

	if !tenantExists {
		tenantName := cfg.Tenants.Default.Name
		if tenantName == "" {
			tenantName = "Default"
		}

		err := pool.QueryRow(
			ctx,
			"INSERT INTO platform.tenants (slug, name, is_default) VALUES ('default', $1, true) RETURNING id",
			tenantName,
		).Scan(&tenantID)
		if err != nil {
			return fmt.Errorf("failed to create default tenant: %w", err)
		}
		log.Info().Str("id", tenantID.String()).Msg("Created default tenant")
	} else {
		log.Debug().Str("id", tenantID.String()).Msg("Default tenant already exists")
	}

	if err := ensureServiceKey(ctx, pool, cfg, tenantID, keys.KeyTypeTenantService); err != nil {
		return fmt.Errorf("failed to ensure service key: %w", err)
	}

	if err := ensureServiceKey(ctx, pool, cfg, tenantID, keys.KeyTypeAnon); err != nil {
		return fmt.Errorf("failed to ensure anon key: %w", err)
	}

	return nil
}

func ensureServiceKey(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, tenantID uuid.UUID, keyType string) error {
	var configKey string
	var keyName string

	switch keyType {
	case keys.KeyTypeTenantService:
		configKey = cfg.Tenants.Default.ServiceKey
		if cfg.Tenants.Default.ServiceKeyFile != "" {
			if data, err := os.ReadFile(cfg.Tenants.Default.ServiceKeyFile); err == nil {
				configKey = strings.TrimSpace(string(data))
			} else {
				log.Warn().Err(err).Str("file", cfg.Tenants.Default.ServiceKeyFile).Msg("Failed to read service key file")
			}
		}
		keyName = "Default Service Key"
	case keys.KeyTypeAnon:
		configKey = cfg.Tenants.Default.AnonKey
		if cfg.Tenants.Default.AnonKeyFile != "" {
			if data, err := os.ReadFile(cfg.Tenants.Default.AnonKeyFile); err == nil {
				configKey = strings.TrimSpace(string(data))
			} else {
				log.Warn().Err(err).Str("file", cfg.Tenants.Default.AnonKeyFile).Msg("Failed to read anon key file")
			}
		}
		keyName = "Default Anon Key"
	default:
		return fmt.Errorf("unsupported key type: %s", keyType)
	}

	var existingKeyID uuid.UUID
	var existingKeyHash string
	err := pool.QueryRow(
		ctx,
		"SELECT id, key_hash FROM auth.service_keys WHERE key_type = $1 AND enabled = true AND revoked_at IS NULL",
		keyType,
	).Scan(&existingKeyID, &existingKeyHash)
	hasExistingKey := err == nil

	if configKey != "" {
		keyHash, err := keys.HashKey(configKey)
		if err != nil {
			return fmt.Errorf("failed to hash config key: %w", err)
		}

		if hasExistingKey {
			if keys.VerifyKey(configKey, existingKeyHash) {
				log.Debug().Str("type", keyType).Msg("Config-managed key already stored")
				return nil
			}
			_, err := pool.Exec(
				ctx,
				"UPDATE auth.service_keys SET enabled = false WHERE id = $1",
				existingKeyID,
			)
			if err != nil {
				return fmt.Errorf("failed to disable old key: %w", err)
			}
		}

		keyPrefix := keys.ExtractPrefix(configKey)
		_, err = pool.Exec(
			ctx,
			`INSERT INTO auth.service_keys 
			(name, key_hash, key_prefix, key_type, enabled, scopes, rate_limit_per_minute)
			VALUES ($1, $2, $3, $4, true, $5, $6)`,
			keyName, keyHash, keyPrefix, keyType, defaultScopes(keyType), defaultRateLimit(keyType),
		)
		if err != nil {
			return fmt.Errorf("failed to insert config-managed key: %w", err)
		}

		log.Info().Str("type", keyType).Msg("Stored config-managed key")
		return nil
	}

	if hasExistingKey {
		log.Debug().Str("type", keyType).Msg("Service key already exists")
		return nil
	}

	_, keyHash, keyPrefix, err := keys.GenerateKey(keyType)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	_, err = pool.Exec(
		ctx,
		`INSERT INTO auth.service_keys 
		(name, key_hash, key_prefix, key_type, enabled, scopes, rate_limit_per_minute)
		VALUES ($1, $2, $3, $4, true, $5, $6)`,
		keyName, keyHash, keyPrefix, keyType, defaultScopes(keyType), defaultRateLimit(keyType),
	)
	if err != nil {
		return fmt.Errorf("failed to insert generated key: %w", err)
	}

	log.Info().
		Str("type", keyType).
		Str("prefix", keyPrefix).
		Msg("Generated new service key - configure via tenants.default.anon_key or tenants.default.service_key to persist")

	return nil
}

func defaultScopes(keyType string) []string {
	switch keyType {
	case keys.KeyTypeTenantService:
		return []string{"*"}
	case keys.KeyTypeAnon:
		return []string{"read"}
	default:
		return []string{}
	}
}

func defaultRateLimit(keyType string) int {
	switch keyType {
	case keys.KeyTypeTenantService:
		return 10000
	case keys.KeyTypeAnon:
		return 60
	default:
		return 60
	}
}

// GetDefaultTenantID returns the default tenant ID.
func GetDefaultTenantID(pool *pgxpool.Pool) *string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var tenantID string
	err := pool.QueryRow(
		ctx,
		"SELECT id::text FROM platform.tenants WHERE slug = 'default' AND deleted_at IS NULL",
	).Scan(&tenantID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get default tenant ID")
		return nil
	}
	return &tenantID
}

func isNoRowsError(err error) bool {
	return err != nil && (err.Error() == "no rows in result set" || strings.Contains(err.Error(), "no rows"))
}
