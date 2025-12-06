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
