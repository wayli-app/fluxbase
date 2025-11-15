package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/database"
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

	// Determine database role based on authentication
	role := h.getDatabaseRole(c)

	// Execute query with appropriate role
	value, err := h.getSettingWithRole(ctx, key, role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Setting not found or access denied",
			})
		}
		log.Error().Err(err).Str("key", key).Str("role", role).Msg("Failed to get setting")
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

	// Determine database role based on authentication
	role := h.getDatabaseRole(c)

	// Execute query with appropriate role
	results, err := h.getSettingsWithRole(ctx, req.Keys, role)
	if err != nil {
		log.Error().Err(err).Str("role", role).Msg("Failed to get settings")
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

// getDatabaseRole determines the PostgreSQL role based on authentication context
func (h *SettingsHandler) getDatabaseRole(c *fiber.Ctx) string {
	// Check if user is authenticated by looking for user_id in context
	userID := c.Locals("user_id")

	if userID != nil {
		// User is authenticated
		return "authenticated"
	}

	// User is not authenticated (anonymous)
	return "anon"
}

// getSettingWithRole retrieves a single setting with the specified database role
func (h *SettingsHandler) getSettingWithRole(ctx context.Context, key string, role string) (interface{}, error) {
	conn, err := h.db.Pool().Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Start a transaction to ensure role is scoped
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Set the database role for RLS
	_, err = tx.Exec(ctx, "SET LOCAL ROLE "+role)
	if err != nil {
		return nil, err
	}

	// Query the setting (RLS policies will be applied)
	var valueJSON []byte
	err = tx.QueryRow(ctx, `
		SELECT value
		FROM app.settings
		WHERE key = $1
	`, key).Scan(&valueJSON)

	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON value
	var valueMap map[string]interface{}
	if err := json.Unmarshal(valueJSON, &valueMap); err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Extract the actual value from the value map
	if val, ok := valueMap["value"]; ok {
		return val, nil
	}

	return valueMap, nil
}

// getSettingsWithRole retrieves multiple settings with the specified database role
func (h *SettingsHandler) getSettingsWithRole(ctx context.Context, keys []string, role string) (map[string]interface{}, error) {
	conn, err := h.db.Pool().Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Start a transaction to ensure role is scoped
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Set the database role for RLS
	_, err = tx.Exec(ctx, "SET LOCAL ROLE "+role)
	if err != nil {
		return nil, err
	}

	// Query the settings (RLS policies will be applied)
	rows, err := tx.Query(ctx, `
		SELECT key, value
		FROM app.settings
		WHERE key = ANY($1)
	`, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build results map
	results := make(map[string]interface{})
	for rows.Next() {
		var key string
		var valueJSON []byte

		if err := rows.Scan(&key, &valueJSON); err != nil {
			return nil, err
		}

		// Unmarshal the JSON value
		var valueMap map[string]interface{}
		if err := json.Unmarshal(valueJSON, &valueMap); err != nil {
			return nil, err
		}

		// Extract the actual value from the value map
		if val, ok := valueMap["value"]; ok {
			results[key] = val
		} else {
			results[key] = valueMap
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return results, nil
}
