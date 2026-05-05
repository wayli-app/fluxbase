package auth

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
)

type SAMLSessionStore struct {
	db *database.Connection
}

func NewSAMLSessionStore(db *database.Connection) *SAMLSessionStore {
	return &SAMLSessionStore{db: db}
}

func (ss *SAMLSessionStore) CreateSAMLSession(ctx context.Context, session *SAMLSession) error {
	tenantID := database.TenantFromContext(ctx)
	return database.WrapWithServiceRoleAndTenant(ctx, ss.db, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(
			ctx, `
			INSERT INTO auth.saml_sessions (id, user_id, provider_id, provider_name, name_id, name_id_format, session_index, attributes, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`,
			session.ID,
			session.UserID,
			session.ProviderID,
			session.ProviderName,
			session.NameID,
			session.NameIDFormat,
			session.SessionIndex,
			session.Attributes,
			session.ExpiresAt,
		)
		return err
	})
}

func (ss *SAMLSessionStore) DeleteSAMLSession(ctx context.Context, sessionID string) error {
	return database.WrapWithServiceRole(ctx, ss.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM auth.saml_sessions WHERE id = $1`, sessionID)
		return err
	})
}

func (ss *SAMLSessionStore) GetSAMLSessionByUserID(ctx context.Context, userID string) (*SAMLSession, error) {
	var session SAMLSession
	err := database.WrapWithServiceRole(ctx, ss.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, user_id, provider_id, provider_name, name_id, name_id_format, session_index, attributes, expires_at, created_at
			FROM auth.saml_sessions
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT 1
		`, userID).Scan(
			&session.ID,
			&session.UserID,
			&session.ProviderID,
			&session.ProviderName,
			&session.NameID,
			&session.NameIDFormat,
			&session.SessionIndex,
			&session.Attributes,
			&session.ExpiresAt,
			&session.CreatedAt,
		)
	})
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (ss *SAMLSessionStore) GetSAMLSessionByNameID(ctx context.Context, providerName, nameID string) (*SAMLSession, error) {
	var session SAMLSession
	err := database.WrapWithServiceRole(ctx, ss.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, user_id, provider_id, provider_name, name_id, name_id_format, session_index, attributes, expires_at, created_at
			FROM auth.saml_sessions
			WHERE provider_name = $1 AND name_id = $2
			ORDER BY created_at DESC
			LIMIT 1
		`, providerName, nameID).Scan(
			&session.ID,
			&session.UserID,
			&session.ProviderID,
			&session.ProviderName,
			&session.NameID,
			&session.NameIDFormat,
			&session.SessionIndex,
			&session.Attributes,
			&session.ExpiresAt,
			&session.CreatedAt,
		)
	})
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (ss *SAMLSessionStore) GetSAMLSessionBySessionIndex(ctx context.Context, providerName, sessionIndex string) (*SAMLSession, error) {
	var session SAMLSession
	err := database.WrapWithServiceRole(ctx, ss.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, user_id, provider_id, provider_name, name_id, name_id_format, session_index, attributes, expires_at, created_at
			FROM auth.saml_sessions
			WHERE provider_name = $1 AND session_index = $2
			ORDER BY created_at DESC
			LIMIT 1
		`, providerName, sessionIndex).Scan(
			&session.ID,
			&session.UserID,
			&session.ProviderID,
			&session.ProviderName,
			&session.NameID,
			&session.NameIDFormat,
			&session.SessionIndex,
			&session.Attributes,
			&session.ExpiresAt,
			&session.CreatedAt,
		)
	})
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (ss *SAMLSessionStore) DeleteSAMLSessionsByUserID(ctx context.Context, userID string) error {
	return database.WrapWithServiceRole(ctx, ss.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM auth.saml_sessions WHERE user_id = $1`, userID)
		return err
	})
}

func (ss *SAMLSessionStore) DeleteSAMLSessionByNameID(ctx context.Context, providerName, nameID string) error {
	return database.WrapWithServiceRole(ctx, ss.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM auth.saml_sessions WHERE provider_name = $1 AND name_id = $2`, providerName, nameID)
		return err
	})
}

func (ss *SAMLSessionStore) CleanupExpiredAssertions(ctx context.Context) error {
	return database.WrapWithServiceRole(ctx, ss.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM auth.saml_assertion_ids WHERE expires_at < NOW()`)
		return err
	})
}
