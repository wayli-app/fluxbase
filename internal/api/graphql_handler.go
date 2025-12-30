package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// GraphQLHandler handles GraphQL HTTP requests
type GraphQLHandler struct {
	schemaGenerator *GraphQLSchemaGenerator
	db              *database.Connection
	config          *config.GraphQLConfig
	resolverFactory *GraphQLResolverFactory
}

// GraphQLRequest represents a GraphQL HTTP request body
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL HTTP response body
type GraphQLResponse struct {
	Data   interface{}    `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []GraphQLErrorLocation `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// GraphQLErrorLocation represents the location of a GraphQL error in the query
type GraphQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// NewGraphQLHandler creates a new GraphQL handler
func NewGraphQLHandler(db *database.Connection, schemaCache *database.SchemaCache, cfg *config.GraphQLConfig) *GraphQLHandler {
	// Create resolver factory
	resolverFactory := NewGraphQLResolverFactory(db.Pool(), schemaCache)

	// Create schema generator
	schemaGenerator := NewGraphQLSchemaGenerator(schemaCache, db, cfg.Introspection)
	schemaGenerator.SetResolverFactory(resolverFactory)

	return &GraphQLHandler{
		schemaGenerator: schemaGenerator,
		db:              db,
		config:          cfg,
		resolverFactory: resolverFactory,
	}
}

// HandleGraphQL handles POST /api/v1/graphql requests
func (h *GraphQLHandler) HandleGraphQL(c *fiber.Ctx) error {
	startTime := time.Now()
	ctx := c.UserContext()

	// Parse request body
	var req GraphQLRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Invalid JSON in request body",
			}},
		})
	}

	// Validate query is present
	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Query string is required",
			}},
		})
	}

	// Validate query depth
	if h.config.MaxDepth > 0 {
		depth, err := calculateQueryDepth(req.Query)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(GraphQLResponse{
				Errors: []GraphQLError{{
					Message: "Invalid query syntax",
				}},
			})
		}
		if depth > h.config.MaxDepth {
			return c.Status(fiber.StatusBadRequest).JSON(GraphQLResponse{
				Errors: []GraphQLError{{
					Message: fmt.Sprintf("query depth %d exceeds maximum allowed depth of %d", depth, h.config.MaxDepth),
				}},
			})
		}
	}

	// Validate query complexity
	if h.config.MaxComplexity > 0 {
		complexity := calculateQueryComplexity(req.Query)
		if complexity > h.config.MaxComplexity {
			return c.Status(fiber.StatusBadRequest).JSON(GraphQLResponse{
				Errors: []GraphQLError{{
					Message: fmt.Sprintf("query complexity %d exceeds maximum of %d", complexity, h.config.MaxComplexity),
				}},
			})
		}
	}

	// Get GraphQL schema
	schema, err := h.schemaGenerator.GetSchema(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get GraphQL schema")
		return c.Status(fiber.StatusInternalServerError).JSON(GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Failed to initialize GraphQL schema",
			}},
		})
	}

	// Set up RLS context if user is authenticated
	ctx = h.setupRLSContext(c, ctx)

	// Execute the query
	result := graphql.Do(graphql.Params{
		Schema:         *schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		OperationName:  req.OperationName,
		Context:        ctx,
	})

	// Log query execution
	duration := time.Since(startTime)
	log.Debug().
		Str("operation", req.OperationName).
		Int("errors", len(result.Errors)).
		Dur("duration", duration).
		Msg("GraphQL query executed")

	// Convert graphql-go errors to our format
	response := GraphQLResponse{
		Data: result.Data,
	}

	if len(result.Errors) > 0 {
		response.Errors = make([]GraphQLError, len(result.Errors))
		for i, err := range result.Errors {
			gqlErr := GraphQLError{
				Message: err.Message,
				Path:    err.Path,
			}
			if len(err.Locations) > 0 {
				gqlErr.Locations = make([]GraphQLErrorLocation, len(err.Locations))
				for j, loc := range err.Locations {
					gqlErr.Locations[j] = GraphQLErrorLocation{
						Line:   loc.Line,
						Column: loc.Column,
					}
				}
			}
			response.Errors[i] = gqlErr
		}
	}

	return c.JSON(response)
}

// setupRLSContext sets up Row Level Security context for the query
func (h *GraphQLHandler) setupRLSContext(c *fiber.Ctx, ctx context.Context) context.Context {
	// Get user from fiber context (set by auth middleware)
	user, ok := c.Locals("user").(*auth.User)
	if !ok || user == nil {
		return ctx
	}

	// Create RLS context
	rlsCtx := &RLSContext{
		UserID: user.ID,
		Role:   user.Role,
		Claims: make(map[string]interface{}),
	}

	// Add user metadata to claims if available
	if user.UserMetadata != nil {
		if metadata, ok := user.UserMetadata.(map[string]interface{}); ok {
			for k, v := range metadata {
				rlsCtx.Claims[k] = v
			}
		}
	}

	return context.WithValue(ctx, GraphQLRLSContextKey, rlsCtx)
}

// HandleIntrospection handles GET /api/v1/graphql (returns introspection data)
func (h *GraphQLHandler) HandleIntrospection(c *fiber.Ctx) error {
	if !h.config.Introspection {
		return c.Status(fiber.StatusForbidden).JSON(GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Introspection is disabled",
			}},
		})
	}

	ctx := c.UserContext()

	// Get GraphQL schema
	schema, err := h.schemaGenerator.GetSchema(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Failed to initialize GraphQL schema",
			}},
		})
	}

	// Execute introspection query
	result := graphql.Do(graphql.Params{
		Schema:        *schema,
		RequestString: introspectionQuery,
		Context:       ctx,
	})

	return c.JSON(GraphQLResponse{
		Data:   result.Data,
		Errors: convertErrors(result.Errors),
	})
}

// RegisterRoutes registers GraphQL routes with the Fiber app
func (h *GraphQLHandler) RegisterRoutes(app *fiber.App, authService *auth.Service, clientKeyService *auth.ClientKeyService, db *pgxpool.Pool, jwtManager *auth.JWTManager) {
	// Import middleware package
	// GraphQL endpoint - requires authentication with RLS support
	graphqlGroup := app.Group("/api/v1/graphql")

	// POST /api/v1/graphql - execute queries/mutations
	graphqlGroup.Post("/", h.HandleGraphQL)

	// GET /api/v1/graphql - introspection (if enabled)
	graphqlGroup.Get("/", h.HandleIntrospection)
}

// InvalidateSchema invalidates the cached GraphQL schema
func (h *GraphQLHandler) InvalidateSchema() {
	h.schemaGenerator.InvalidateSchema()
}

// convertErrors converts graphql-go errors to our format
func convertErrors(errors []gqlerrors.FormattedError) []GraphQLError {
	if len(errors) == 0 {
		return nil
	}

	result := make([]GraphQLError, len(errors))
	for i, err := range errors {
		gqlErr := GraphQLError{
			Message: err.Message,
			Path:    err.Path,
		}
		if len(err.Locations) > 0 {
			gqlErr.Locations = make([]GraphQLErrorLocation, len(err.Locations))
			for j, loc := range err.Locations {
				gqlErr.Locations[j] = GraphQLErrorLocation{
					Line:   loc.Line,
					Column: loc.Column,
				}
			}
		}
		result[i] = gqlErr
	}
	return result
}

// calculateQueryDepth returns the maximum depth of a GraphQL query
func calculateQueryDepth(query string) (int, error) {
	doc, err := parser.Parse(parser.ParseParams{Source: query})
	if err != nil {
		return 0, err
	}

	var maxDepth int
	for _, def := range doc.Definitions {
		if op, ok := def.(*ast.OperationDefinition); ok {
			depth := calculateSelectionSetDepth(op.SelectionSet, 0)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}
	return maxDepth, nil
}

// calculateSelectionSetDepth recursively calculates the depth of a selection set
func calculateSelectionSetDepth(selSet *ast.SelectionSet, currentDepth int) int {
	if selSet == nil || len(selSet.Selections) == 0 {
		return currentDepth
	}

	maxDepth := currentDepth + 1
	for _, sel := range selSet.Selections {
		switch s := sel.(type) {
		case *ast.Field:
			depth := calculateSelectionSetDepth(s.SelectionSet, currentDepth+1)
			if depth > maxDepth {
				maxDepth = depth
			}
		case *ast.InlineFragment:
			depth := calculateSelectionSetDepth(s.SelectionSet, currentDepth+1)
			if depth > maxDepth {
				maxDepth = depth
			}
		case *ast.FragmentSpread:
			// Fragment spreads need document context to resolve
			// For now, count as +1 depth
			if currentDepth+1 > maxDepth {
				maxDepth = currentDepth + 1
			}
		}
	}
	return maxDepth
}

// calculateQueryComplexity calculates a complexity score for a GraphQL query
// based on the number of fields and their types (lists have higher cost)
func calculateQueryComplexity(query string) int {
	doc, err := parser.Parse(parser.ParseParams{Source: query})
	if err != nil {
		return 0
	}

	var totalComplexity int
	for _, def := range doc.Definitions {
		if op, ok := def.(*ast.OperationDefinition); ok {
			// Add base cost for mutations
			baseCost := 0
			if op.Operation == ast.OperationTypeMutation {
				baseCost = 10
			}
			complexity := calculateSelectionComplexity(op.SelectionSet, 1)
			totalComplexity += baseCost + complexity
		}
	}
	return totalComplexity
}

// calculateSelectionComplexity recursively calculates the complexity of a selection set
func calculateSelectionComplexity(selSet *ast.SelectionSet, multiplier int) int {
	if selSet == nil {
		return 0
	}

	var complexity int
	for _, sel := range selSet.Selections {
		switch s := sel.(type) {
		case *ast.Field:
			fieldName := s.Name.Value
			fieldCost := 1 // Base cost per field

			// List fields (ending with 's' or common list names) have higher cost
			// This is a heuristic; actual type information would be better
			isListField := len(fieldName) > 1 && fieldName[len(fieldName)-1] == 's' &&
				fieldName != "status" && fieldName != "address"

			if isListField {
				fieldCost = 10

				// Check for 'first' or 'last' argument to adjust cost
				for _, arg := range s.Arguments {
					if arg.Name.Value == "first" || arg.Name.Value == "last" || arg.Name.Value == "limit" {
						if intVal, ok := arg.Value.GetValue().(int); ok && intVal > 0 {
							fieldCost = intVal
							if fieldCost < 10 {
								fieldCost = 10
							}
						}
					}
				}
			}

			complexity += fieldCost * multiplier

			// Recurse into nested selections
			if s.SelectionSet != nil {
				nestedMultiplier := multiplier
				if isListField {
					nestedMultiplier *= 10 // Assume 10 items per list by default
				}
				complexity += calculateSelectionComplexity(s.SelectionSet, nestedMultiplier)
			}

		case *ast.InlineFragment:
			complexity += calculateSelectionComplexity(s.SelectionSet, multiplier)
		}
	}
	return complexity
}

// Standard GraphQL introspection query
const introspectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
    directives {
      name
      description
      locations
      args {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type { ...TypeRef }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
      }
    }
  }
}
`
