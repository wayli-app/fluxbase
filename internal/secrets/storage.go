package secrets

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/crypto"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Secret represents a stored secret
type Secret struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Scope          string     `json:"scope"`               // "global" or "namespace"
	Namespace      *string    `json:"namespace,omitempty"` // NULL for global, set for namespace-scoped
	EncryptedValue string     `json:"-"`                   // Never expose in JSON
	Description    *string    `json:"description,omitempty"`
	Version        int        `json:"version"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy      *uuid.UUID `json:"updated_by,omitempty"`
}

// SecretSummary is a lightweight version for list responses (never includes value)
type SecretSummary struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Scope       string     `json:"scope"`
	Namespace   *string    `json:"namespace,omitempty"`
	Description *string    `json:"description,omitempty"`
	Version     int        `json:"version"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsExpired   bool       `json:"is_expired"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty"`
}

// SecretVersion represents a historical version of a secret
type SecretVersion struct {
	ID        uuid.UUID  `json:"id"`
	SecretID  uuid.UUID  `json:"secret_id"`
	Version   int        `json:"version"`
	CreatedAt time.Time  `json:"created_at"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
}

// Storage manages secret persistence with encryption
type Storage struct {
	db            *database.Connection
	encryptionKey string
}

// NewStorage creates a new secrets storage manager
func NewStorage(db *database.Connection, encryptionKey string) *Storage {
	return &Storage{
		db:            db,
		encryptionKey: encryptionKey,
	}
}

// CreateSecret creates a new secret with encrypted value
func (s *Storage) CreateSecret(ctx context.Context, secret *Secret, plainValue string, userID *uuid.UUID) error {
	// Encrypt the value before storage
	encryptedValue, err := crypto.Encrypt(plainValue, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret value: %w", err)
	}

	query := `
		INSERT INTO functions.secrets (
			name, scope, namespace, encrypted_value, description, expires_at, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, version, created_at, updated_at
	`

	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			secret.Name, secret.Scope, secret.Namespace, encryptedValue,
			secret.Description, secret.ExpiresAt, userID,
		).Scan(&secret.ID, &secret.Version, &secret.CreatedAt, &secret.UpdatedAt)
	})

	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	// Store initial version in history
	if err := s.storeVersion(ctx, secret.ID, 1, encryptedValue, userID); err != nil {
		// Log but don't fail - the secret was created successfully
		fmt.Printf("warning: failed to store initial secret version: %v\n", err)
	}

	return nil
}

// GetSecret retrieves a secret by ID (metadata only, no value)
func (s *Storage) GetSecret(ctx context.Context, id uuid.UUID) (*Secret, error) {
	query := `
		SELECT id, name, scope, namespace, description, version, expires_at,
		       created_at, updated_at, created_by, updated_by
		FROM functions.secrets
		WHERE id = $1
	`

	secret := &Secret{}
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, id).Scan(
			&secret.ID, &secret.Name, &secret.Scope, &secret.Namespace,
			&secret.Description, &secret.Version, &secret.ExpiresAt,
			&secret.CreatedAt, &secret.UpdatedAt, &secret.CreatedBy, &secret.UpdatedBy,
		)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	return secret, nil
}

// GetSecretByName retrieves a secret by name and optional namespace
func (s *Storage) GetSecretByName(ctx context.Context, name string, namespace *string) (*Secret, error) {
	var query string
	var args []interface{}

	if namespace == nil {
		query = `
			SELECT id, name, scope, namespace, description, version, expires_at,
			       created_at, updated_at, created_by, updated_by
			FROM functions.secrets
			WHERE name = $1 AND scope = 'global' AND namespace IS NULL
		`
		args = []interface{}{name}
	} else {
		query = `
			SELECT id, name, scope, namespace, description, version, expires_at,
			       created_at, updated_at, created_by, updated_by
			FROM functions.secrets
			WHERE name = $1 AND namespace = $2
		`
		args = []interface{}{name, *namespace}
	}

	secret := &Secret{}
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(
			&secret.ID, &secret.Name, &secret.Scope, &secret.Namespace,
			&secret.Description, &secret.Version, &secret.ExpiresAt,
			&secret.CreatedAt, &secret.UpdatedAt, &secret.CreatedBy, &secret.UpdatedBy,
		)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get secret by name: %w", err)
	}

	return secret, nil
}

// ListSecrets returns all secrets matching the filter criteria (metadata only)
func (s *Storage) ListSecrets(ctx context.Context, scope *string, namespace *string) ([]SecretSummary, error) {
	query := `
		SELECT id, name, scope, namespace, description, version, expires_at,
		       CASE WHEN expires_at IS NOT NULL AND expires_at < NOW() THEN true ELSE false END as is_expired,
		       created_at, updated_at, created_by, updated_by
		FROM functions.secrets
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if scope != nil {
		query += fmt.Sprintf(" AND scope = $%d", argIdx)
		args = append(args, *scope)
		argIdx++
	}

	if namespace != nil {
		query += fmt.Sprintf(" AND namespace = $%d", argIdx)
		args = append(args, *namespace)
		argIdx++
	}

	query += " ORDER BY scope, namespace NULLS FIRST, name"

	var secrets []SecretSummary
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			secret := SecretSummary{}
			err := rows.Scan(
				&secret.ID, &secret.Name, &secret.Scope, &secret.Namespace,
				&secret.Description, &secret.Version, &secret.ExpiresAt,
				&secret.IsExpired, &secret.CreatedAt, &secret.UpdatedAt,
				&secret.CreatedBy, &secret.UpdatedBy,
			)
			if err != nil {
				return err
			}
			secrets = append(secrets, secret)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	return secrets, nil
}

// UpdateSecret updates a secret's value (increments version and stores history)
func (s *Storage) UpdateSecret(ctx context.Context, id uuid.UUID, plainValue *string, description *string, expiresAt *time.Time, userID *uuid.UUID) error {
	// Start with base updates
	updates := "updated_at = NOW(), updated_by = $2"
	args := []interface{}{id, userID}
	argIdx := 3

	// Handle value update with encryption
	if plainValue != nil {
		encryptedValue, err := crypto.Encrypt(*plainValue, s.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt secret value: %w", err)
		}
		updates += fmt.Sprintf(", encrypted_value = $%d, version = version + 1", argIdx)
		args = append(args, encryptedValue)
		argIdx++
	}

	if description != nil {
		updates += fmt.Sprintf(", description = $%d", argIdx)
		args = append(args, *description)
		argIdx++
	}

	if expiresAt != nil {
		updates += fmt.Sprintf(", expires_at = $%d", argIdx)
		args = append(args, *expiresAt)
		argIdx++
	}

	query := fmt.Sprintf(`
		UPDATE functions.secrets
		SET %s
		WHERE id = $1
		RETURNING version, encrypted_value
	`, updates)

	var newVersion int
	var encryptedValue string
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(&newVersion, &encryptedValue)
	})

	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	// Store new version in history (only if value was updated)
	if plainValue != nil {
		if err := s.storeVersion(ctx, id, newVersion, encryptedValue, userID); err != nil {
			fmt.Printf("warning: failed to store secret version: %v\n", err)
		}
	}

	return nil
}

// DeleteSecret deletes a secret by ID
func (s *Storage) DeleteSecret(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM functions.secrets WHERE id = $1"

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return fmt.Errorf("secret not found")
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}

// GetVersions returns the version history for a secret
func (s *Storage) GetVersions(ctx context.Context, secretID uuid.UUID) ([]SecretVersion, error) {
	query := `
		SELECT id, secret_id, version, created_at, created_by
		FROM functions.secret_versions
		WHERE secret_id = $1
		ORDER BY version DESC
	`

	var versions []SecretVersion
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, secretID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			version := SecretVersion{}
			err := rows.Scan(
				&version.ID, &version.SecretID, &version.Version,
				&version.CreatedAt, &version.CreatedBy,
			)
			if err != nil {
				return err
			}
			versions = append(versions, version)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get secret versions: %w", err)
	}

	return versions, nil
}

// RollbackToVersion restores a secret to a previous version
func (s *Storage) RollbackToVersion(ctx context.Context, secretID uuid.UUID, version int, userID *uuid.UUID) error {
	// Get the encrypted value from the specified version
	getQuery := `
		SELECT encrypted_value
		FROM functions.secret_versions
		WHERE secret_id = $1 AND version = $2
	`

	var encryptedValue string
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, getQuery, secretID, version).Scan(&encryptedValue)
	})

	if err != nil {
		return fmt.Errorf("failed to get version %d: %w", version, err)
	}

	// Update the secret with the old value and increment version
	updateQuery := `
		UPDATE functions.secrets
		SET encrypted_value = $2, version = version + 1, updated_at = NOW(), updated_by = $3
		WHERE id = $1
		RETURNING version
	`

	var newVersion int
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, updateQuery, secretID, encryptedValue, userID).Scan(&newVersion)
	})

	if err != nil {
		return fmt.Errorf("failed to rollback secret: %w", err)
	}

	// Store the rollback as a new version
	if err := s.storeVersion(ctx, secretID, newVersion, encryptedValue, userID); err != nil {
		fmt.Printf("warning: failed to store rollback version: %v\n", err)
	}

	return nil
}

// GetSecretsForNamespace returns decrypted secrets for a specific namespace
// This includes both global secrets and namespace-specific secrets
// Expired secrets are excluded
func (s *Storage) GetSecretsForNamespace(ctx context.Context, namespace string) (map[string]string, error) {
	query := `
		SELECT name, encrypted_value
		FROM functions.secrets
		WHERE (scope = 'global' OR (scope = 'namespace' AND namespace = $1))
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY scope DESC
	`
	// scope DESC ensures namespace-specific secrets override global ones

	secrets := make(map[string]string)
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, namespace)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var name, encryptedValue string
			if err := rows.Scan(&name, &encryptedValue); err != nil {
				return err
			}

			// Decrypt the value
			plainValue, err := crypto.Decrypt(encryptedValue, s.encryptionKey)
			if err != nil {
				// Skip secrets that can't be decrypted (corrupted or wrong key)
				fmt.Printf("warning: failed to decrypt secret %s: %v\n", name, err)
				continue
			}

			secrets[name] = plainValue
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get secrets for namespace: %w", err)
	}

	return secrets, nil
}

// storeVersion stores a version record for audit trail
func (s *Storage) storeVersion(ctx context.Context, secretID uuid.UUID, version int, encryptedValue string, userID *uuid.UUID) error {
	query := `
		INSERT INTO functions.secret_versions (secret_id, version, encrypted_value, created_by)
		VALUES ($1, $2, $3, $4)
	`

	return database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, secretID, version, encryptedValue, userID)
		return err
	})
}

// GetStats returns statistics about secrets
func (s *Storage) GetStats(ctx context.Context) (total int, expiringSoon int, expired int, err error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE expires_at IS NOT NULL AND expires_at > NOW() AND expires_at < NOW() + INTERVAL '7 days') as expiring_soon,
			COUNT(*) FILTER (WHERE expires_at IS NOT NULL AND expires_at < NOW()) as expired
		FROM functions.secrets
	`

	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query).Scan(&total, &expiringSoon, &expired)
	})

	if err != nil {
		err = fmt.Errorf("failed to get secret stats: %w", err)
	}

	return
}
