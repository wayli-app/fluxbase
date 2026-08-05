package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

type SettingsHandler struct {
	db *database.Connection
}

func NewSettingsHandler(db *database.Connection) *SettingsHandler {
	return &SettingsHandler{
		db: db,
	}
}

func (h *SettingsHandler) requireService(c fiber.Ctx) error {
	if h.db == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Database not initialized")
	}
	return nil
}

type SettingResponse struct {
	Value interface{} `json:"value"`
}

type BatchSettingsRequest struct {
	Keys   []string `json:"keys"`
	Prefix string   `json:"prefix,omitempty"`
}

type BatchSettingsResponse struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func (h *SettingsHandler) GetSetting(c fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("key")

	if key == "" {
		return SendMissingField(c, "Setting key")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	var value interface{}
	var queryErr error

	middleware.SetTargetSchema(c, "app")

	err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		var valueJSON []byte
		// Prefer a tenant-specific row over the instance default (tenant_id IS
		// NULL) for this key, mirroring the batch endpoint's resolution.
		queryErr = tx.QueryRow(ctx, `
			SELECT value
			FROM app.settings
			WHERE key = $1
			ORDER BY (tenant_id IS NOT NULL) DESC
			LIMIT 1
		`, key).Scan(&valueJSON)

		if queryErr != nil {
			return queryErr
		}

		var valueMap map[string]interface{}
		if err := json.Unmarshal(valueJSON, &valueMap); err != nil {
			return err
		}

		if val, ok := valueMap["value"]; ok {
			value = val
		} else {
			value = valueMap
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SendNotFound(c, "Setting not found or access denied")
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to get setting")
		return SendInternalError(c, "Failed to retrieve setting")
	}

	return c.JSON(SettingResponse{Value: value})
}

func (h *SettingsHandler) GetSettings(c fiber.Ctx) error {
	ctx := context.Background()

	var req BatchSettingsRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// At least one of keys or prefix is required. A prefix fetch returns all
	// keys under a namespace (e.g. "wayli.*") in a single call, so clients don't
	// need to know every key up front — and don't trigger a 404 per missing key.
	if len(req.Keys) == 0 && req.Prefix == "" {
		return SendMissingField(c, "keys or prefix")
	}

	if len(req.Keys) > 100 {
		return SendBadRequest(c, "Maximum 100 keys allowed per request", ErrCodeInvalidInput)
	}

	// A prefix must be a namespace: it must end with '.' (e.g. "wayli."). This
	// prevents targeted key-existence probing (prefix "wayli.secret") and
	// accidental cross-namespace over-matching. Visibility is still enforced by
	// RLS regardless — this guard is about namespace hygiene, not access control.
	if req.Prefix != "" && !strings.HasSuffix(req.Prefix, ".") {
		return SendBadRequest(c, "prefix must end with a '.' (e.g. 'wayli.')", ErrCodeInvalidInput)
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	results := make(map[string]interface{})

	middleware.SetTargetSchema(c, "app")

	err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		var rows pgx.Rows
		var queryErr error

		// Tenant resolution: instance-level (tenant_id IS NULL) settings are the
		// leading baseline everyone inherits; a tenant-specific row overrides the
		// instance default for that key (mirrors the per-user user→system
		// fallback). DISTINCT ON (key) keeps one row per key, preferring the
		// tenant row (tenant_id IS NOT NULL sorts first under DESC), so when both
		// a tenant and an instance-default row exist for a key, the tenant value
		// wins. RLS still gates which rows are visible (the TenantContext
		// middleware sets app.current_tenant_id), so a caller only ever sees rows
		// in their own tenant + the shared instance defaults.
		switch {
		case req.Prefix != "" && len(req.Keys) > 0:
			// Both: keys within the namespace.
			rows, queryErr = tx.Query(ctx, `
				SELECT key, value FROM (
					SELECT DISTINCT ON (key) key, value
					FROM app.settings
					WHERE key LIKE $1 AND key = ANY($2)
					ORDER BY key, (tenant_id IS NOT NULL) DESC
				) d
				ORDER BY key
				LIMIT 200
			`, req.Prefix+"%", req.Keys)
		case req.Prefix != "":
			// Prefix only: all visible keys in the namespace. LIMIT bounds the
			// result; RLS already hides anything the caller can't see, so missing
			// vs hidden are indistinguishable to the caller (no existence leak).
			rows, queryErr = tx.Query(ctx, `
				SELECT key, value FROM (
					SELECT DISTINCT ON (key) key, value
					FROM app.settings
					WHERE key LIKE $1
					ORDER BY key, (tenant_id IS NOT NULL) DESC
				) d
				ORDER BY key
				LIMIT 200
			`, req.Prefix+"%")
		default:
			// Explicit keys only (original behavior) with the same per-key
			// tenant→instance-default resolution.
			rows, queryErr = tx.Query(ctx, `
				SELECT key, value FROM (
					SELECT DISTINCT ON (key) key, value
					FROM app.settings
					WHERE key = ANY($1)
					ORDER BY key, (tenant_id IS NOT NULL) DESC
				) d
			`, req.Keys)
		}
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()

		for rows.Next() {
			var key string
			var valueJSON []byte

			if err := rows.Scan(&key, &valueJSON); err != nil {
				return err
			}

			var valueMap map[string]interface{}
			if err := json.Unmarshal(valueJSON, &valueMap); err != nil {
				return err
			}

			if val, ok := valueMap["value"]; ok {
				results[key] = val
			} else {
				results[key] = valueMap
			}
		}

		return rows.Err()
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to get settings")
		return SendInternalError(c, "Failed to retrieve settings")
	}

	response := make([]BatchSettingsResponse, 0, len(results))
	for key, value := range results {
		response = append(response, BatchSettingsResponse{
			Key:   key,
			Value: value,
		})
	}

	return c.JSON(response)
}
