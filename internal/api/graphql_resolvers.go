package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/graphql-go/graphql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// GraphQLResolverFactory creates resolvers for GraphQL queries and mutations
type GraphQLResolverFactory struct {
	db          *pgxpool.Pool
	schemaCache *database.SchemaCache
}

// NewGraphQLResolverFactory creates a new resolver factory
func NewGraphQLResolverFactory(db *pgxpool.Pool, schemaCache *database.SchemaCache) *GraphQLResolverFactory {
	return &GraphQLResolverFactory{
		db:          db,
		schemaCache: schemaCache,
	}
}

// GraphQL context keys
type graphqlContextKey string

const (
	// GraphQLRLSContextKey is used to store RLS context in the request context
	GraphQLRLSContextKey graphqlContextKey = "graphql_rls_context"
)

// RLSContext contains information needed for Row Level Security
type RLSContext struct {
	UserID string
	Role   string
	Claims map[string]interface{}
}

// mapAppRoleToDatabaseRole maps application-level roles to database-level roles
// This is a copy of the middleware function to avoid import cycles
func mapAppRoleToDatabaseRole(appRole string) string {
	switch appRole {
	case "service_role", "dashboard_admin":
		return "service_role"
	case "anon", "":
		return "anon"
	default:
		return "authenticated"
	}
}

// queryWithRLS executes a query with Row Level Security context
// This wraps the query in a transaction with SET LOCAL ROLE and request.jwt.claims set
func (g *GraphQLSchemaGenerator) queryWithRLS(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	db := g.getDBFromContext(ctx)
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Check if RLS context is present
	rlsCtx, ok := ctx.Value(GraphQLRLSContextKey).(*RLSContext)

	// If no RLS context, execute directly (anonymous access)
	if !ok || rlsCtx == nil {
		log.Debug().Msg("GraphQL query executing without RLS context (anonymous)")
		rows, err := db.Query(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanRowsToMaps(rows)
	}

	// Start a transaction to set RLS context
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Set RLS context
	if err := setGraphQLRLSContext(ctx, tx, rlsCtx); err != nil {
		return nil, err
	}

	// Execute query within transaction
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanRowsToMaps(rows)
	if err != nil {
		return nil, err
	}

	// Commit the transaction (read-only queries don't need explicit commit, but it's good practice)
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return results, nil
}

// execWithRLS executes a mutating query (INSERT/UPDATE/DELETE) with RLS context
func (g *GraphQLSchemaGenerator) execWithRLS(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	db := g.getDBFromContext(ctx)
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Check if RLS context is present
	rlsCtx, ok := ctx.Value(GraphQLRLSContextKey).(*RLSContext)

	// Start a transaction
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Set RLS context if present
	if ok && rlsCtx != nil {
		if err := setGraphQLRLSContext(ctx, tx, rlsCtx); err != nil {
			return nil, err
		}
	} else {
		// Anonymous access - set anon role
		if _, err := tx.Exec(ctx, "SET LOCAL ROLE anon"); err != nil {
			return nil, fmt.Errorf("failed to SET LOCAL ROLE anon: %w", err)
		}
	}

	// Execute query within transaction
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanRowsToMaps(rows)
	if err != nil {
		return nil, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return results, nil
}

// setGraphQLRLSContext sets PostgreSQL session variables for RLS
func setGraphQLRLSContext(ctx context.Context, tx pgx.Tx, rlsCtx *RLSContext) error {
	// Map application role to database role
	dbRole := mapAppRoleToDatabaseRole(rlsCtx.Role)

	// SET LOCAL ROLE for database-level security (quoteIdentifier for safety)
	setRoleQuery := fmt.Sprintf("SET LOCAL ROLE %s", quoteIdentifier(dbRole))
	if _, err := tx.Exec(ctx, setRoleQuery); err != nil {
		log.Error().Err(err).Str("db_role", dbRole).Msg("GraphQL: Failed to SET LOCAL ROLE")
		return fmt.Errorf("failed to SET LOCAL ROLE %s: %w", dbRole, err)
	}

	// Build JWT claims for RLS policies
	jwtClaims := map[string]interface{}{
		"sub":  rlsCtx.UserID,
		"role": rlsCtx.Role, // Original application role for fine-grained policies
	}

	// Add additional claims if present
	for k, v := range rlsCtx.Claims {
		jwtClaims[k] = v
	}

	jwtClaimsJSON, err := json.Marshal(jwtClaims)
	if err != nil {
		log.Error().Err(err).Msg("GraphQL: Failed to marshal JWT claims")
		return fmt.Errorf("failed to marshal JWT claims: %w", err)
	}

	// Set request.jwt.claims session variable (Supabase format)
	if _, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", string(jwtClaimsJSON)); err != nil {
		log.Error().Err(err).Msg("GraphQL: Failed to set request.jwt.claims")
		return fmt.Errorf("failed to set request.jwt.claims: %w", err)
	}

	log.Debug().
		Str("user_id", rlsCtx.UserID).
		Str("app_role", rlsCtx.Role).
		Str("db_role", dbRole).
		Msg("GraphQL: RLS context set for query")

	return nil
}

// makeCollectionResolver creates a resolver for querying a collection of records
func (g *GraphQLSchemaGenerator) makeCollectionResolver(table database.TableInfo) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context

		// Build the query
		qb := NewQueryBuilder(table.Schema, table.Name)

		// Apply filters
		if filter, ok := p.Args["filter"].(map[string]interface{}); ok && len(filter) > 0 {
			filters := g.buildFiltersFromArgs(table, filter)
			qb.WithFilters(filters)
		}

		// Apply ordering
		if orderBy, ok := p.Args["orderBy"].([]interface{}); ok && len(orderBy) > 0 {
			orders := g.buildOrderFromArgs(table, orderBy)
			qb.WithOrder(orders)
		}

		// Apply limit
		if limit, ok := p.Args["limit"].(int); ok {
			qb.WithLimit(limit)
		}

		// Apply offset
		if offset, ok := p.Args["offset"].(int); ok {
			qb.WithOffset(offset)
		}

		// Build and execute query with RLS
		sql, args := qb.BuildSelect()
		return g.queryWithRLS(ctx, sql, args...)
	}
}

// makeSingleResolver creates a resolver for querying a single record by primary key
func (g *GraphQLSchemaGenerator) makeSingleResolver(table database.TableInfo) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context

		// Build filters from primary key arguments
		filters := []Filter{}
		for _, pkCol := range table.PrimaryKey {
			fieldName := g.columnToFieldName(pkCol)
			if val, ok := p.Args[fieldName]; ok {
				filters = append(filters, Filter{
					Column:   pkCol,
					Operator: OpEqual,
					Value:    val,
				})
			}
		}

		if len(filters) == 0 {
			return nil, fmt.Errorf("primary key arguments required")
		}

		// Build the query
		qb := NewQueryBuilder(table.Schema, table.Name)
		qb.WithFilters(filters)
		qb.WithLimit(1)

		sql, args := qb.BuildSelect()

		// Execute with RLS
		results, err := g.queryWithRLS(ctx, sql, args...)
		if err != nil {
			return nil, fmt.Errorf("query failed: %w", err)
		}

		if len(results) == 0 {
			return nil, nil
		}
		return results[0], nil
	}
}

// makeInsertResolver creates a resolver for inserting a single record
func (g *GraphQLSchemaGenerator) makeInsertResolver(table database.TableInfo) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context

		data, ok := p.Args["data"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("data argument required")
		}

		// Convert GraphQL field names to database column names
		dbData := g.graphqlToDBColumnNames(table, data)

		// Build the insert query
		qb := NewQueryBuilder(table.Schema, table.Name)
		qb.WithReturning([]string{"*"})
		sql, args := qb.BuildInsert(dbData)

		// Execute with RLS
		results, err := g.execWithRLS(ctx, sql, args...)
		if err != nil {
			return nil, fmt.Errorf("insert failed: %w", err)
		}

		if len(results) == 0 {
			return nil, nil
		}
		return results[0], nil
	}
}

// makeInsertManyResolver creates a resolver for inserting multiple records
func (g *GraphQLSchemaGenerator) makeInsertManyResolver(table database.TableInfo) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context

		dataArr, ok := p.Args["data"].([]interface{})
		if !ok || len(dataArr) == 0 {
			return nil, fmt.Errorf("data array argument required")
		}

		db := g.getDBFromContext(ctx)
		if db == nil {
			return nil, fmt.Errorf("database connection not available")
		}

		// Check if RLS context is present
		rlsCtx, hasRLS := ctx.Value(GraphQLRLSContextKey).(*RLSContext)

		// Start a transaction for all inserts
		tx, err := db.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		// Set RLS context
		if hasRLS && rlsCtx != nil {
			if err := setGraphQLRLSContext(ctx, tx, rlsCtx); err != nil {
				return nil, err
			}
		} else {
			if _, err := tx.Exec(ctx, "SET LOCAL ROLE anon"); err != nil {
				return nil, fmt.Errorf("failed to SET LOCAL ROLE anon: %w", err)
			}
		}

		var results []map[string]interface{}
		for _, item := range dataArr {
			data, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			dbData := g.graphqlToDBColumnNames(table, data)

			qb := NewQueryBuilder(table.Schema, table.Name)
			qb.WithReturning([]string{"*"})
			sql, args := qb.BuildInsert(dbData)

			rows, err := tx.Query(ctx, sql, args...)
			if err != nil {
				return nil, fmt.Errorf("insert failed: %w", err)
			}

			insertedRows, err := scanRowsToMaps(rows)
			rows.Close()
			if err != nil {
				return nil, err
			}

			results = append(results, insertedRows...)
		}

		// Commit transaction
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		return results, nil
	}
}

// makeUpdateResolver creates a resolver for updating a single record by primary key
func (g *GraphQLSchemaGenerator) makeUpdateResolver(table database.TableInfo) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context

		data, ok := p.Args["data"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("data argument required")
		}

		// Build filters from primary key arguments
		filters := []Filter{}
		for _, pkCol := range table.PrimaryKey {
			fieldName := g.columnToFieldName(pkCol)
			if val, ok := p.Args[fieldName]; ok {
				filters = append(filters, Filter{
					Column:   pkCol,
					Operator: OpEqual,
					Value:    val,
				})
			}
		}

		if len(filters) == 0 {
			return nil, fmt.Errorf("primary key arguments required")
		}

		dbData := g.graphqlToDBColumnNames(table, data)

		qb := NewQueryBuilder(table.Schema, table.Name)
		qb.WithFilters(filters)
		qb.WithReturning([]string{"*"})
		sql, args := qb.BuildUpdate(dbData)

		// Execute with RLS
		results, err := g.execWithRLS(ctx, sql, args...)
		if err != nil {
			return nil, fmt.Errorf("update failed: %w", err)
		}

		if len(results) == 0 {
			return nil, nil
		}
		return results[0], nil
	}
}

// makeUpdateManyResolver creates a resolver for updating multiple records
func (g *GraphQLSchemaGenerator) makeUpdateManyResolver(table database.TableInfo) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context

		data, ok := p.Args["data"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("data argument required")
		}

		filter, ok := p.Args["filter"].(map[string]interface{})
		if !ok || len(filter) == 0 {
			return nil, fmt.Errorf("filter argument required")
		}

		dbData := g.graphqlToDBColumnNames(table, data)
		filters := g.buildFiltersFromArgs(table, filter)

		qb := NewQueryBuilder(table.Schema, table.Name)
		qb.WithFilters(filters)
		qb.WithReturning([]string{"*"})
		sql, args := qb.BuildUpdate(dbData)

		// Execute with RLS
		return g.execWithRLS(ctx, sql, args...)
	}
}

// makeDeleteResolver creates a resolver for deleting a single record by primary key
func (g *GraphQLSchemaGenerator) makeDeleteResolver(table database.TableInfo) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context

		// Build filters from primary key arguments
		filters := []Filter{}
		for _, pkCol := range table.PrimaryKey {
			fieldName := g.columnToFieldName(pkCol)
			if val, ok := p.Args[fieldName]; ok {
				filters = append(filters, Filter{
					Column:   pkCol,
					Operator: OpEqual,
					Value:    val,
				})
			}
		}

		if len(filters) == 0 {
			return nil, fmt.Errorf("primary key arguments required")
		}

		qb := NewQueryBuilder(table.Schema, table.Name)
		qb.WithFilters(filters)
		qb.WithReturning([]string{"*"})
		sql, args := qb.BuildDelete()

		// Execute with RLS
		results, err := g.execWithRLS(ctx, sql, args...)
		if err != nil {
			return nil, fmt.Errorf("delete failed: %w", err)
		}

		if len(results) == 0 {
			return nil, nil
		}
		return results[0], nil
	}
}

// makeDeleteManyResolver creates a resolver for deleting multiple records
func (g *GraphQLSchemaGenerator) makeDeleteManyResolver(table database.TableInfo) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context

		filter, ok := p.Args["filter"].(map[string]interface{})
		if !ok || len(filter) == 0 {
			return nil, fmt.Errorf("filter argument required")
		}

		filters := g.buildFiltersFromArgs(table, filter)

		db := g.getDBFromContext(ctx)
		if db == nil {
			return nil, fmt.Errorf("database connection not available")
		}

		// Check if RLS context is present
		rlsCtx, hasRLS := ctx.Value(GraphQLRLSContextKey).(*RLSContext)

		// Start a transaction for count + delete
		tx, err := db.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		// Set RLS context
		if hasRLS && rlsCtx != nil {
			if err := setGraphQLRLSContext(ctx, tx, rlsCtx); err != nil {
				return nil, err
			}
		} else {
			if _, err := tx.Exec(ctx, "SET LOCAL ROLE anon"); err != nil {
				return nil, fmt.Errorf("failed to SET LOCAL ROLE anon: %w", err)
			}
		}

		// First count how many will be deleted (within RLS context)
		qb := NewQueryBuilder(table.Schema, table.Name)
		qb.WithFilters(filters)
		countSQL, countArgs := qb.BuildCount()

		var count int
		err = tx.QueryRow(ctx, countSQL, countArgs...).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("count failed: %w", err)
		}

		// Now delete
		qb = NewQueryBuilder(table.Schema, table.Name)
		qb.WithFilters(filters)
		sql, args := qb.BuildDelete()

		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			return nil, fmt.Errorf("delete failed: %w", err)
		}

		// Commit transaction
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		return count, nil
	}
}

// makeForeignKeyResolver creates a resolver for fetching related records via foreign key
// IMPORTANT: This resolver must enforce RLS when following foreign key relationships
// to prevent unauthorized access via GraphQL traversals
func (g *GraphQLSchemaGenerator) makeForeignKeyResolver(table database.TableInfo, fk database.ForeignKey) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		source, ok := p.Source.(map[string]interface{})
		if !ok {
			return nil, nil
		}

		// Get the foreign key value from the source record
		fkValue := source[fk.ColumnName]
		if fkValue == nil {
			return nil, nil
		}

		ctx := p.Context

		// Query the related table
		qb := NewQueryBuilder(table.Schema, fk.ReferencedTable)
		qb.WithFilters([]Filter{{
			Column:   fk.ReferencedColumn,
			Operator: OpEqual,
			Value:    fkValue,
		}})
		qb.WithLimit(1)

		sql, args := qb.BuildSelect()

		// Execute with RLS - this is critical for security
		// Foreign key traversals must respect RLS policies
		results, err := g.queryWithRLS(ctx, sql, args...)
		if err != nil {
			return nil, fmt.Errorf("query failed: %w", err)
		}

		if len(results) == 0 {
			return nil, nil
		}
		return results[0], nil
	}
}

// buildFiltersFromArgs converts GraphQL filter arguments to query filters
func (g *GraphQLSchemaGenerator) buildFiltersFromArgs(table database.TableInfo, args map[string]interface{}) []Filter {
	orGroupCounter := 0
	return g.buildFiltersFromArgsWithCounter(table, args, &orGroupCounter, 0, false)
}

// buildFiltersFromArgsWithCounter converts GraphQL filter arguments to query filters with OR group tracking
func (g *GraphQLSchemaGenerator) buildFiltersFromArgsWithCounter(table database.TableInfo, args map[string]interface{}, orGroupCounter *int, currentOrGroup int, negated bool) []Filter {
	var filters []Filter

	// Create a map of field names to column names
	fieldToColumn := make(map[string]string)
	for _, col := range table.Columns {
		fieldName := g.columnToFieldName(col.Name)
		fieldToColumn[fieldName] = col.Name
	}

	for key, value := range args {
		if value == nil {
			continue
		}

		// Handle logical operators
		if key == "_and" {
			if andArr, ok := value.([]interface{}); ok {
				for _, andItem := range andArr {
					if andMap, ok := andItem.(map[string]interface{}); ok {
						subFilters := g.buildFiltersFromArgsWithCounter(table, andMap, orGroupCounter, currentOrGroup, negated)
						filters = append(filters, subFilters...)
					}
				}
			}
			continue
		}
		if key == "_or" {
			// OR: each element in the array gets the same OrGroupID
			if orArr, ok := value.([]interface{}); ok {
				*orGroupCounter++
				groupID := *orGroupCounter
				for _, orItem := range orArr {
					if orMap, ok := orItem.(map[string]interface{}); ok {
						subFilters := g.buildFiltersFromArgsWithCounter(table, orMap, orGroupCounter, groupID, negated)
						filters = append(filters, subFilters...)
					}
				}
			}
			continue
		}
		if key == "_not" {
			// NOT: negate the conditions inside
			if notMap, ok := value.(map[string]interface{}); ok {
				subFilters := g.buildFiltersFromArgsWithCounter(table, notMap, orGroupCounter, currentOrGroup, !negated)
				filters = append(filters, subFilters...)
			}
			continue
		}

		// Parse field name and operator
		parts := strings.Split(key, "_")
		if len(parts) < 2 {
			continue
		}

		// Extract operator (last part or last two parts for compound operators)
		operator := parts[len(parts)-1]
		fieldParts := parts[:len(parts)-1]

		// Handle compound operators like "is_null"
		if operator == "null" && len(parts) >= 3 && parts[len(parts)-2] == "is" {
			operator = "is_null"
			fieldParts = parts[:len(parts)-2]
		}

		fieldName := strings.Join(fieldParts, "_")
		columnName, ok := fieldToColumn[fieldName]
		if !ok {
			continue
		}

		// Map GraphQL operators to query builder operators
		var queryOp FilterOperator
		switch operator {
		case "eq":
			queryOp = OpEqual
		case "neq":
			queryOp = OpNotEqual
		case "gt":
			queryOp = OpGreaterThan
		case "gte":
			queryOp = OpGreaterOrEqual
		case "lt":
			queryOp = OpLessThan
		case "lte":
			queryOp = OpLessOrEqual
		case "like":
			queryOp = OpLike
		case "ilike":
			queryOp = OpILike
		case "in":
			queryOp = OpIn
		case "is_null":
			if isNull, ok := value.(bool); ok {
				if isNull {
					queryOp = OpIs
					value = nil
				} else {
					queryOp = OpNot
					value = nil
				}
			}
		case "contains":
			queryOp = OpContains
		case "contained_by":
			queryOp = OpContained
		default:
			continue
		}

		// Apply negation if in a _not block
		if negated {
			queryOp = negateOperator(queryOp)
		}

		filter := Filter{
			Column:   columnName,
			Operator: queryOp,
			Value:    value,
		}

		// Set OR group if we're inside an _or block
		if currentOrGroup > 0 {
			filter.IsOr = true
			filter.OrGroupID = currentOrGroup
		}

		filters = append(filters, filter)
	}

	return filters
}

// negateOperator returns the negated version of a filter operator
func negateOperator(op FilterOperator) FilterOperator {
	switch op {
	case OpEqual:
		return OpNotEqual
	case OpNotEqual:
		return OpEqual
	case OpGreaterThan:
		return OpLessOrEqual
	case OpGreaterOrEqual:
		return OpLessThan
	case OpLessThan:
		return OpGreaterOrEqual
	case OpLessOrEqual:
		return OpGreaterThan
	case OpIn:
		return OpNotIn
	case OpNotIn:
		return OpIn
	case OpIs:
		return OpIsNot
	case OpIsNot:
		return OpIs
	case OpContains:
		// NOT contains - use NOT operator with contains
		return OpContains // Will need special handling in query builder
	default:
		return op
	}
}

// buildOrderFromArgs converts GraphQL orderBy arguments to query ordering
func (g *GraphQLSchemaGenerator) buildOrderFromArgs(table database.TableInfo, args []interface{}) []OrderBy {
	var orders []OrderBy

	// Create a map of field names to column names
	fieldToColumn := make(map[string]string)
	for _, col := range table.Columns {
		fieldName := g.columnToFieldName(col.Name)
		fieldToColumn[fieldName] = col.Name
	}

	for _, arg := range args {
		orderMap, ok := arg.(map[string]interface{})
		if !ok {
			continue
		}

		for fieldName, direction := range orderMap {
			columnName, ok := fieldToColumn[fieldName]
			if !ok {
				continue
			}

			dirStr, ok := direction.(string)
			if !ok {
				continue
			}

			desc := false
			nulls := ""

			switch dirStr {
			case "ASC", "ASC_NULLS_LAST":
				desc = false
				nulls = "last"
			case "ASC_NULLS_FIRST":
				desc = false
				nulls = "first"
			case "DESC", "DESC_NULLS_FIRST":
				desc = true
				nulls = "first"
			case "DESC_NULLS_LAST":
				desc = true
				nulls = "last"
			}

			orders = append(orders, OrderBy{
				Column: columnName,
				Desc:   desc,
				Nulls:  nulls,
			})
		}
	}

	return orders
}

// graphqlToDBColumnNames converts GraphQL field names to database column names
func (g *GraphQLSchemaGenerator) graphqlToDBColumnNames(table database.TableInfo, data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Create a map of field names to column names
	fieldToColumn := make(map[string]string)
	for _, col := range table.Columns {
		fieldName := g.columnToFieldName(col.Name)
		fieldToColumn[fieldName] = col.Name
	}

	for fieldName, value := range data {
		if columnName, ok := fieldToColumn[fieldName]; ok {
			result[columnName] = value
		}
	}

	return result
}

// getDBFromContext gets the database connection from context
func (g *GraphQLSchemaGenerator) getDBFromContext(ctx context.Context) *pgxpool.Pool {
	if g.resolverFactory != nil {
		return g.resolverFactory.db
	}
	return nil
}

// scanRowsToMaps converts pgx rows to a slice of maps
func scanRowsToMaps(rows pgx.Rows) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	cols := rows.FieldDescriptions()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range cols {
			row[string(col.Name)] = values[i]
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
