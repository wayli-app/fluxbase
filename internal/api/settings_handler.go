package api

import (
	"context"
	"encoding/json"
	"errors"

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
		return SendNotInitialized(c, "Database")
	}
	return nil
}

type SettingResponse struct {
	Value interface{} `json:"value"`
}

type BatchSettingsRequest struct {
	Keys []string `json:"keys"`
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
		queryErr = tx.QueryRow(ctx, `
			SELECT value
			FROM app.settings
			WHERE key = $1
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

	if len(req.Keys) == 0 {
		return SendMissingField(c, "keys")
	}

	if len(req.Keys) > 100 {
		return SendBadRequest(c, "Maximum 100 keys allowed per request", ErrCodeInvalidInput)
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	results := make(map[string]interface{})

	middleware.SetTargetSchema(c, "app")

	err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT key, value
			FROM app.settings
			WHERE key = ANY($1)
		`, req.Keys)
		if err != nil {
			return err
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
