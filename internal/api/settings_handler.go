package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// SettingsHandler handles public settings operations with RLS support
type SettingsHandler struct {
	db *database.Connection
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(db *database.Connection) *SettingsHandler {
	return &SettingsHandler{
		db: db,
	}
}

// SettingResponse represents a setting value response
type SettingResponse struct {
	Value interface{} `json:"value"`
}

// BatchSettingsRequest represents a batch settings request
type BatchSettingsRequest struct {
	Keys []string `json:"keys"`
}

// BatchSettingsResponse represents a batch settings response
type BatchSettingsResponse struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// GetSetting retrieves a single setting with RLS enforcement
// GET /api/v1/settings/:key
func (h *SettingsHandler) GetSetting(c *fiber.Ctx) error {
	ctx := context.Background()
	key := c.Params("key")

	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setting key is required",
		})
	}

	var value interface{}
	var queryErr error

	// Use WrapWithRLS to properly set database role + JWT claims
	err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		// Query the setting (RLS policies will be applied)
		var valueJSON []byte
		queryErr = tx.QueryRow(ctx, `
			SELECT value
			FROM app.settings
			WHERE key = $1
		`, key).Scan(&valueJSON)

		if queryErr != nil {
			return queryErr
		}

		// Unmarshal the JSON value
		var valueMap map[string]interface{}
		if err := json.Unmarshal(valueJSON, &valueMap); err != nil {
			return err
		}

		// Extract the actual value from the value map
		if val, ok := valueMap["value"]; ok {
			value = val
		} else {
			value = valueMap
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found or access denied",
			})
		}
		log.Error().Err(err).Str("key", key).Msg("Failed to get setting")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve setting",
		})
	}

	return c.JSON(SettingResponse{Value: value})
}

// GetSettings retrieves multiple settings with RLS enforcement
// POST /api/v1/settings/batch
func (h *SettingsHandler) GetSettings(c *fiber.Ctx) error {
	ctx := context.Background()

	var req BatchSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error().Err(err).Msg("Failed to parse batch settings request")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(req.Keys) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one key is required",
		})
	}

	// Limit the number of keys to prevent abuse
	if len(req.Keys) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Maximum 100 keys allowed per request",
		})
	}

	results := make(map[string]interface{})

	// Use WrapWithRLS to properly set database role + JWT claims
	err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		// Query the settings (RLS policies will be applied)
		rows, err := tx.Query(ctx, `
			SELECT key, value
			FROM app.settings
			WHERE key = ANY($1)
		`, req.Keys)
		if err != nil {
			return err
		}
		defer rows.Close()

		// Build results map
		for rows.Next() {
			var key string
			var valueJSON []byte

			if err := rows.Scan(&key, &valueJSON); err != nil {
				return err
			}

			// Unmarshal the JSON value
			var valueMap map[string]interface{}
			if err := json.Unmarshal(valueJSON, &valueMap); err != nil {
				return err
			}

			// Extract the actual value from the value map
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve settings",
		})
	}

	// Convert map to array for response
	response := make([]BatchSettingsResponse, 0, len(results))
	for key, value := range results {
		response = append(response, BatchSettingsResponse{
			Key:   key,
			Value: value,
		})
	}

	return c.JSON(response)
}
