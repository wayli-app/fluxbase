package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// RPCHandler handles RPC (Remote Procedure Call) endpoints for PostgreSQL functions
type RPCHandler struct {
	db *database.Connection
}

// NewRPCHandler creates a new RPC handler
func NewRPCHandler(db *database.Connection) *RPCHandler {
	return &RPCHandler{db: db}
}

// RegisterRoutes registers all RPC routes based on database functions
func (h *RPCHandler) RegisterRoutes(router fiber.Router) error {
	ctx := context.Background()
	inspector := database.NewSchemaInspector(h.db)

	// Get all functions from public and auth schemas
	functions, err := inspector.GetAllFunctions(ctx, "public", "auth")
	if err != nil {
		return fmt.Errorf("failed to get functions: %w", err)
	}

	log.Info().Int("count", len(functions)).Msg("Registering RPC endpoints")

	for _, fn := range functions {
		h.RegisterFunctionRoute(router, fn)
	}

	return nil
}

// RegisterFunctionRoute registers a single function as an RPC endpoint
func (h *RPCHandler) RegisterFunctionRoute(router fiber.Router, fn database.FunctionInfo) {
	path := h.buildFunctionPath(fn)

	log.Info().
		Str("function", fmt.Sprintf("%s.%s", fn.Schema, fn.Name)).
		Str("path", path).
		Str("return_type", fn.ReturnType).
		Int("params", len(fn.Parameters)).
		Msg("Registering RPC endpoint")

	// RPC endpoints are POST only
	router.Post(path, h.makeFunctionHandler(fn))
}

// buildFunctionPath builds the API path for a function
func (h *RPCHandler) buildFunctionPath(fn database.FunctionInfo) string {
	// Paths are relative to the router group, no /api/rpc prefix needed
	if fn.Schema != "public" {
		return fmt.Sprintf("/%s/%s", fn.Schema, fn.Name)
	}
	return fmt.Sprintf("/%s", fn.Name)
}

// makeFunctionHandler creates a handler for calling a PostgreSQL function
func (h *RPCHandler) makeFunctionHandler(fn database.FunctionInfo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		// Parse request body as JSON
		var params map[string]interface{}
		if err := c.BodyParser(&params); err != nil {
			// If body is empty, use empty params
			params = make(map[string]interface{})
		}

		// Build function call
		query, args, err := h.buildFunctionCall(fn, params)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		log.Debug().
			Str("function", fmt.Sprintf("%s.%s", fn.Schema, fn.Name)).
			Str("query", query).
			Interface("args", args).
			Msg("Executing RPC call")

		// Execute function
		if fn.IsSetOf {
			// Function returns a set of rows
			rows, err := h.db.Query(ctx, query, args...)
			if err != nil {
				log.Error().Err(err).Str("function", fn.Name).Msg("Failed to execute function")
				return c.Status(500).JSON(fiber.Map{
					"error": "Failed to execute function",
				})
			}
			defer rows.Close()

			// Collect all rows
			var results []map[string]interface{}
			for rows.Next() {
				values, err := rows.Values()
				if err != nil {
					return c.Status(500).JSON(fiber.Map{
						"error": "Failed to read function results",
					})
				}

				// If function returns single column, return value directly
				if len(values) == 1 {
					// Try to parse as JSON if it's a composite type
					if jsonStr, ok := values[0].(string); ok {
						var jsonData map[string]interface{}
						if err := json.Unmarshal([]byte(jsonStr), &jsonData); err == nil {
							results = append(results, jsonData)
							continue
						}
					}
					// Otherwise return as single-key map
					results = append(results, map[string]interface{}{
						fn.ReturnType: values[0],
					})
				} else {
					// Multiple columns - build map from column names
					columns := rows.FieldDescriptions()
					row := make(map[string]interface{})
					for i, col := range columns {
						row[string(col.Name)] = values[i]
					}
					results = append(results, row)
				}
			}

			return c.JSON(results)
		} else {
			// Function returns a single value
			row := h.db.QueryRow(ctx, query, args...)

			var result interface{}
			if err := row.Scan(&result); err != nil {
				log.Error().Err(err).Str("function", fn.Name).Msg("Failed to execute function")
				return c.Status(500).JSON(fiber.Map{
					"error": "Failed to execute function",
				})
			}

			// Try to parse as JSON if it's a composite type
			if jsonStr, ok := result.(string); ok {
				var jsonData map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &jsonData); err == nil {
					return c.JSON(jsonData)
				}
			}

			// Return scalar value
			return c.JSON(map[string]interface{}{
				"result": result,
			})
		}
	}
}

// buildFunctionCall builds the SQL query to call a function with parameters
func (h *RPCHandler) buildFunctionCall(fn database.FunctionInfo, params map[string]interface{}) (string, []interface{}, error) {
	var args []interface{}
	var placeholders []string
	argCounter := 1

	// Build parameter list
	for _, param := range fn.Parameters {
		// Skip OUT parameters (they're for return values)
		if param.Mode == "OUT" {
			continue
		}

		var value interface{}
		var found bool

		// Try to find parameter by name
		if param.Name != "" {
			value, found = params[param.Name]
		}

		// If not found and no default, check positional parameter
		if !found && !param.HasDefault {
			// Try positional parameter (zero-indexed in array)
			value, found = params[fmt.Sprintf("arg%d", param.Position)]
		}

		// If still not found and no default, error
		if !found && !param.HasDefault {
			return "", nil, fmt.Errorf("missing required parameter: %s (type: %s)", param.Name, param.Type)
		}

		if found {
			args = append(args, value)
			placeholders = append(placeholders, fmt.Sprintf("$%d", argCounter))
			argCounter++
		}
	}

	// Build function call
	var query string
	if len(placeholders) > 0 {
		query = fmt.Sprintf("SELECT * FROM %s.%s(%s)", fn.Schema, fn.Name, strings.Join(placeholders, ", "))
	} else {
		query = fmt.Sprintf("SELECT * FROM %s.%s()", fn.Schema, fn.Name)
	}

	return query, args, nil
}

// GetFunctionInfo retrieves information about a specific function
func (h *RPCHandler) GetFunctionInfo(ctx context.Context, schema, name string) (*database.FunctionInfo, error) {
	inspector := database.NewSchemaInspector(h.db)
	functions, err := inspector.GetAllFunctions(ctx, schema)
	if err != nil {
		return nil, err
	}

	for _, fn := range functions {
		if fn.Name == name {
			return &fn, nil
		}
	}

	return nil, fmt.Errorf("function not found: %s.%s", schema, name)
}
