package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// PostQueryRequest represents the JSON body for POST-based queries
// Used when query parameters would exceed URL length limits
type PostQueryRequest struct {
	Select         string                   `json:"select,omitempty"`
	Filters        []PostQueryFilter        `json:"filters,omitempty"`
	OrFilters      []string                 `json:"orFilters,omitempty"`
	AndFilters     []string                 `json:"andFilters,omitempty"`
	BetweenFilters []PostQueryBetweenFilter `json:"betweenFilters,omitempty"`
	Order          []PostQueryOrderBy       `json:"order,omitempty"`
	Limit          *int                     `json:"limit,omitempty"`
	Offset         *int                     `json:"offset,omitempty"`
	Count          string                   `json:"count,omitempty"`
	GroupBy        []string                 `json:"groupBy,omitempty"`
}

// PostQueryFilter represents a single filter in the POST body
type PostQueryFilter struct {
	Column   string      `json:"column"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// PostQueryBetweenFilter represents a between filter in the POST body
type PostQueryBetweenFilter struct {
	Column  string      `json:"column"`
	Min     interface{} `json:"min"`
	Max     interface{} `json:"max"`
	Negated bool        `json:"negated"`
}

// PostQueryOrderBy represents an order clause in the POST body
type PostQueryOrderBy struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
	Nulls     string `json:"nulls,omitempty"`
}

// makePostQueryHandler creates a handler for POST-based queries
// This allows complex filter expressions that would exceed URL length limits
func (h *RESTHandler) makePostQueryHandler(table database.TableInfo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		// Parse request body
		var req PostQueryRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Invalid request body",
				"details": err.Error(),
			})
		}

		// Convert POST request to QueryParams
		params, err := h.convertPostQueryToParams(&req)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Invalid query parameters",
				"details": err.Error(),
			})
		}

		// Build and execute query (reuse existing logic from GET handler)
		query, args := h.buildSelectQuery(table, params)

		// Execute query with RLS context
		var results []map[string]interface{}
		err = middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, query, args...)
			if err != nil {
				return err
			}
			defer rows.Close()

			results, err = pgxRowsToJSON(rows)
			return err
		})

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to execute query",
			})
		}

		// Handle count if requested
		if params.Count != "" {
			count, err := h.getCount(ctx, c, table, params)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to get count",
				})
			}

			// Set Content-Range header
			start := 0
			if params.Offset != nil {
				start = *params.Offset
			}
			end := start + len(results) - 1
			if end < start {
				end = start
			}
			c.Set("Content-Range", fmt.Sprintf("%d-%d/%d", start, end, count))
		}

		return c.JSON(results)
	}
}

// convertPostQueryToParams converts a POST query request to QueryParams
func (h *RESTHandler) convertPostQueryToParams(req *PostQueryRequest) (*QueryParams, error) {
	params := &QueryParams{
		Filters: []Filter{},
		Order:   []OrderBy{},
	}

	// Parse select
	if req.Select != "" {
		params.Select = strings.Split(req.Select, ",")
		for i := range params.Select {
			params.Select[i] = strings.TrimSpace(params.Select[i])
		}
	}

	// Convert regular filters
	for _, f := range req.Filters {
		params.Filters = append(params.Filters, Filter{
			Column:   f.Column,
			Operator: FilterOperator(f.Operator),
			Value:    f.Value,
			IsOr:     false,
		})
	}

	// Convert between filters
	for _, bf := range req.BetweenFilters {
		if bf.Negated {
			// not.between: (column < min OR column > max)
			// Create a new OR group for this not.between
			params.orGroupCounter++
			groupID := params.orGroupCounter

			params.Filters = append(params.Filters, Filter{
				Column:    bf.Column,
				Operator:  OpLessThan,
				Value:     bf.Min,
				IsOr:      true,
				OrGroupID: groupID,
			})
			params.Filters = append(params.Filters, Filter{
				Column:    bf.Column,
				Operator:  OpGreaterThan,
				Value:     bf.Max,
				IsOr:      true,
				OrGroupID: groupID,
			})
		} else {
			// between: (column >= min AND column <= max)
			params.Filters = append(params.Filters, Filter{
				Column:   bf.Column,
				Operator: OpGreaterOrEqual,
				Value:    bf.Min,
				IsOr:     false,
			})
			params.Filters = append(params.Filters, Filter{
				Column:   bf.Column,
				Operator: OpLessOrEqual,
				Value:    bf.Max,
				IsOr:     false,
			})
		}
	}

	// Parse OR filters (string format like "column.operator.value,column.operator.value")
	for _, orFilter := range req.OrFilters {
		// Create a new OR group
		params.orGroupCounter++
		groupID := params.orGroupCounter

		parts := strings.Split(orFilter, ",")
		for _, part := range parts {
			filterParts := strings.SplitN(strings.TrimSpace(part), ".", 3)
			if len(filterParts) != 3 {
				return nil, fmt.Errorf("invalid OR filter format: %s", part)
			}
			params.Filters = append(params.Filters, Filter{
				Column:    filterParts[0],
				Operator:  FilterOperator(filterParts[1]),
				Value:     filterParts[2],
				IsOr:      true,
				OrGroupID: groupID,
			})
		}
	}

	// Parse AND filters (string format)
	for _, andFilter := range req.AndFilters {
		parts := strings.Split(andFilter, ",")
		for _, part := range parts {
			filterParts := strings.SplitN(strings.TrimSpace(part), ".", 3)
			if len(filterParts) != 3 {
				return nil, fmt.Errorf("invalid AND filter format: %s", part)
			}
			params.Filters = append(params.Filters, Filter{
				Column:   filterParts[0],
				Operator: FilterOperator(filterParts[1]),
				Value:    filterParts[2],
				IsOr:     false,
			})
		}
	}

	// Convert order
	for _, o := range req.Order {
		orderBy := OrderBy{
			Column: o.Column,
			Desc:   strings.ToLower(o.Direction) == "desc",
			Nulls:  strings.ToLower(o.Nulls),
		}
		params.Order = append(params.Order, orderBy)
	}

	// Set limit and offset
	params.Limit = req.Limit
	params.Offset = req.Offset

	// Set count type
	if req.Count != "" {
		params.Count = CountType(req.Count)
	}

	// Set group by
	params.GroupBy = req.GroupBy

	return params, nil
}

// buildSelectQuery builds a SELECT query from parameters
func (h *RESTHandler) buildSelectQuery(table database.TableInfo, params *QueryParams) (string, []interface{}) {
	var selectClause string

	// If we have aggregations, use BuildSelectClause (handles aggregations)
	if len(params.Aggregations) > 0 || len(params.GroupBy) > 0 {
		selectClause = params.BuildSelectClause(table.Name)
	} else if len(params.Select) > 0 {
		// Validate and sanitize column names for regular selects
		validColumns := []string{}
		for _, col := range params.Select {
			if h.columnExists(table, col) {
				// Check if this column needs geometry conversion
				for _, tableCol := range table.Columns {
					if tableCol.Name == col && isGeometryColumn(tableCol.DataType) {
						validColumns = append(validColumns, fmt.Sprintf("ST_AsGeoJSON(%s)::jsonb AS %s", col, col))
						break
					} else if tableCol.Name == col {
						validColumns = append(validColumns, col)
						break
					}
				}
			}
		}
		if len(validColumns) > 0 {
			selectClause = strings.Join(validColumns, ", ")
		} else {
			selectClause = buildSelectColumns(table)
		}
	} else {
		// Use buildSelectColumns to handle geometry columns
		selectClause = buildSelectColumns(table)
	}

	// Start building query
	query := fmt.Sprintf("SELECT %s FROM %s.%s", selectClause, table.Schema, table.Name)

	// Add WHERE, ORDER BY, LIMIT, OFFSET
	whereAndMore, args := params.ToSQL(table.Name)
	if whereAndMore != "" {
		query += " " + whereAndMore
	}

	// Add GROUP BY clause
	groupByClause := params.BuildGroupByClause()
	if groupByClause != "" {
		query += groupByClause
	}

	return query, args
}

// columnExists checks if a column exists in the table
func (h *RESTHandler) columnExists(table database.TableInfo, columnName string) bool {
	for _, col := range table.Columns {
		if col.Name == columnName {
			return true
		}
	}
	return false
}

// getCount gets the row count for a query
func (h *RESTHandler) getCount(ctx context.Context, c *fiber.Ctx, table database.TableInfo, params *QueryParams) (int, error) {
	// Build count query
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", table.Schema, table.Name)

	// Build WHERE clause
	var args []interface{}
	if len(params.Filters) > 0 {
		argCounter := 1
		whereClause, whereArgs := params.buildWhereClause(&argCounter)
		if whereClause != "" {
			query += " WHERE " + whereClause
			args = whereArgs
		}
	}

	// Execute count query with RLS context
	var count int
	err := middleware.WrapWithRLS(ctx, h.db, c, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(&count)
	})

	return count, err
}
