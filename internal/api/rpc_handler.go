package api

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/wayli-app/fluxbase/internal/middleware"
)

// RPCHandler handles RPC (Remote Procedure Call) endpoints for PostgreSQL functions
type RPCHandler struct {
	db *database.Connection
}

// NewRPCHandler creates a new RPC handler
func NewRPCHandler(db *database.Connection) *RPCHandler {
	return &RPCHandler{db: db}
}

// convertBytesToValue converts byte arrays to appropriate types
// Handles UUID (16 bytes), and falls back to string for other bytea
func convertBytesToValue(bytes []byte) interface{} {
	// Check if it's a UUID (exactly 16 bytes)
	if len(bytes) == 16 {
		// Format as UUID string: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
		uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
			bytes[0:4],
			bytes[4:6],
			bytes[6:8],
			bytes[8:10],
			bytes[10:16])
		return uuid
	}

	// For other byte arrays, check if it's valid UTF-8
	str := string(bytes)
	if isValidUTF8(str) {
		return str
	}

	// If not valid UTF-8, return as hex string
	return "0x" + hex.EncodeToString(bytes)
}

// isValidUTF8 checks if a string contains valid UTF-8
func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '\ufffd' {
			return false
		}
	}
	return true
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

	// Filter out internal and non-public functions before registering
	userFunctions := make([]database.FunctionInfo, 0)
	for _, fn := range functions {
		if !h.isInternalFunction(fn) && h.isFunctionPublic(ctx, fn) {
			userFunctions = append(userFunctions, fn)
		}
	}

	log.Info().Int("count", len(userFunctions)).Msg("Registering RPC endpoints")

	// Register GET endpoint to list all functions
	router.Get("/", h.ListFunctions)

	// Register POST endpoints for each function
	for _, fn := range userFunctions {
		h.RegisterFunctionRoute(router, fn)
	}

	return nil
}

// ListFunctions returns a list of all available RPC functions
func (h *RPCHandler) ListFunctions(c *fiber.Ctx) error {
	ctx := c.Context()
	inspector := database.NewSchemaInspector(h.db)

	// Get all functions from public and auth schemas
	functions, err := inspector.GetAllFunctions(ctx, "public", "auth")
	if err != nil {
		log.Error().Err(err).Msg("Failed to get functions")
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve functions",
		})
	}

	// Filter out internal and non-public functions
	filteredFunctions := make([]database.FunctionInfo, 0)
	for _, fn := range functions {
		if !h.isInternalFunction(fn) && h.isFunctionPublic(ctx, fn) {
			filteredFunctions = append(filteredFunctions, fn)
		}
	}

	return c.JSON(filteredFunctions)
}

// isInternalFunction checks if a function is an internal PostgreSQL extension function
// that should not be exposed as an RPC endpoint
func (h *RPCHandler) isInternalFunction(fn database.FunctionInfo) bool {
	// List of internal function prefixes that should be filtered out
	internalPrefixes := []string{
		"gin_",     // GIN index functions
		"gtrgm_",   // pg_trgm extension functions
		"uuid_ns_", // UUID namespace functions (usually internal)
	}

	// List of internal function names that should be filtered out
	internalFunctions := map[string]bool{
		"notify_table_change":      true, // Internal trigger function
		"update_updated_at_column": true, // Internal trigger function
		"enable_realtime":          true, // Internal realtime setup function
		"disable_realtime":         true, // Internal realtime setup function
	}

	// Check if function name matches internal prefixes
	for _, prefix := range internalPrefixes {
		if len(fn.Name) >= len(prefix) && fn.Name[:len(prefix)] == prefix {
			return true
		}
	}

	// Check if function is in the internal functions list
	if internalFunctions[fn.Name] {
		return true
	}

	// Keep user-facing utility functions
	return false
}

// isFunctionPublic checks if a function should be exposed as a public RPC endpoint
// Returns true if:
//   - Function NOT in config table (user-created, public by default)
//   - Function in config table with is_public=true
//
// Returns false if:
//   - Function in config table with is_public=false (system function)
func (h *RPCHandler) isFunctionPublic(ctx context.Context, fn database.FunctionInfo) bool {
	var isPublic bool

	query := `
		SELECT is_public
		FROM functions.rpc_function_config
		WHERE schema_name = $1 AND function_name = $2
	`

	err := h.db.QueryRow(ctx, query, fn.Schema, fn.Name).Scan(&isPublic)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Function not in config table -> user-created -> public by default
			return true
		}
		// Database error -> log and default to public
		log.Warn().Err(err).
			Str("schema", fn.Schema).
			Str("function", fn.Name).
			Msg("Failed to check function config, defaulting to public")
		return true
	}

	// Function found in config table, use its is_public value
	return isPublic
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

		log.Info().
			Str("function", fmt.Sprintf("%s.%s", fn.Schema, fn.Name)).
			Str("query", query).
			Interface("args", args).
			Msg("Executing RPC call")

		// Execute function within RLS transaction
		var responseData interface{}
		err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
			// Execute function
			if fn.IsSetOf {
				// Function returns a set of rows
				rows, err := tx.Query(ctx, query, args...)
				if err != nil {
					log.Error().Err(err).Str("function", fn.Name).Msg("Failed to execute function")
					return err
				}
				defer rows.Close()

				// Collect all rows
				var results []map[string]interface{}
				for rows.Next() {
					values, err := rows.Values()
					if err != nil {
						return err
					}

					// Convert byte arrays to appropriate types (UUID, string, etc.)
					for i, val := range values {
						if bytes, ok := val.([]byte); ok {
							values[i] = convertBytesToValue(bytes)
						} else if uuidArray, ok := val.([16]uint8); ok {
							// Convert [16]uint8 array to []byte slice
							bytes := make([]byte, 16)
							copy(bytes, uuidArray[:])
							values[i] = convertBytesToValue(bytes)
						}
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

				responseData = results
				return nil
			} else {
				// Function returns a single value
				row := tx.QueryRow(ctx, query, args...)

				var result interface{}
				if err := row.Scan(&result); err != nil {
					log.Error().Err(err).Str("function", fn.Name).Msg("Failed to execute function")
					return err
				}

				// Convert byte arrays to appropriate types (UUID, bytea, etc.)
				// Handle both []byte (slice) and [16]uint8 (fixed array for UUIDs)
				if bytes, ok := result.([]byte); ok {
					result = convertBytesToValue(bytes)
				} else if uuidArray, ok := result.([16]uint8); ok {
					// Convert [16]uint8 array to []byte slice
					bytes := make([]byte, 16)
					copy(bytes, uuidArray[:])
					result = convertBytesToValue(bytes)
				}

				// Try to parse as JSON if it's a composite type
				if jsonStr, ok := result.(string); ok {
					var jsonData map[string]interface{}
					if err := json.Unmarshal([]byte(jsonStr), &jsonData); err == nil {
						responseData = jsonData
						return nil
					}
				}

				// Return scalar value
				responseData = map[string]interface{}{
					"result": result,
				}
				return nil
			}
		})

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to execute function",
			})
		}

		return c.JSON(responseData)
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
