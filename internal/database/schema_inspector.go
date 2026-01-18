package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// SchemaInspector provides PostgreSQL schema introspection capabilities
type SchemaInspector struct {
	conn *Connection
}

// TableInfo represents metadata about a database table, view, or materialized view
type TableInfo struct {
	Schema      string       `json:"schema"`
	Name        string       `json:"name"`
	Type        string       `json:"type"`                // "table", "view", or "materialized_view"
	RESTPath    string       `json:"rest_path,omitempty"` // The REST API path for this table (e.g., "/auth/users")
	Columns     []ColumnInfo `json:"columns"`
	PrimaryKey  []string     `json:"primary_key"`
	ForeignKeys []ForeignKey `json:"foreign_keys"`
	Indexes     []IndexInfo  `json:"indexes"`
	RLSEnabled  bool         `json:"rls_enabled"`
}

// ColumnInfo represents metadata about a table column
type ColumnInfo struct {
	Name         string  `json:"name"`
	DataType     string  `json:"data_type"`
	IsNullable   bool    `json:"is_nullable"`
	DefaultValue *string `json:"default_value"`
	IsPrimaryKey bool    `json:"is_primary_key"`
	IsForeignKey bool    `json:"is_foreign_key"`
	IsUnique     bool    `json:"is_unique"`
	MaxLength    *int    `json:"max_length"`
	Position     int     `json:"position"`
}

// ForeignKey represents a foreign key relationship
type ForeignKey struct {
	Name             string `json:"name"`
	ColumnName       string `json:"column_name"`
	ReferencedTable  string `json:"referenced_table"`
	ReferencedColumn string `json:"referenced_column"`
	OnDelete         string `json:"on_delete"`
	OnUpdate         string `json:"on_update"`
}

// IndexInfo represents an index on a table
type IndexInfo struct {
	Name      string   `json:"name"`
	Columns   []string `json:"columns"`
	IsUnique  bool     `json:"is_unique"`
	IsPrimary bool     `json:"is_primary"`
}

// NewSchemaInspector creates a new schema inspector
func NewSchemaInspector(conn *Connection) *SchemaInspector {
	return &SchemaInspector{conn: conn}
}

// GetAllTables retrieves information about all tables in the specified schemas.
// This uses batched queries to avoid N+1 query patterns.
func (si *SchemaInspector) GetAllTables(ctx context.Context, schemas ...string) ([]TableInfo, error) {
	if len(schemas) == 0 {
		schemas = []string{"public"}
	}

	// Query to get all tables from specified schemas
	query := `
		SELECT
			schemaname,
			tablename,
			CASE
				WHEN relrowsecurity THEN true
				ELSE false
			END as rls_enabled
		FROM pg_tables t
		JOIN pg_class c ON c.relname = t.tablename AND c.relnamespace = (
			SELECT oid FROM pg_namespace WHERE nspname = t.schemaname
		)
		WHERE schemaname = ANY($1)
			AND tablename NOT LIKE 'pg_%'
			AND tablename NOT LIKE '_fluxbase.%'
			AND schemaname NOT IN ('information_schema', 'pg_catalog', '_fluxbase')
		ORDER BY schemaname, tablename
	`

	rows, err := si.conn.Query(ctx, query, schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	// Collect all table names and build initial TableInfo map
	tableMap := make(map[string]*TableInfo) // key: "schema.table"
	var tableKeys []string

	for rows.Next() {
		var schema, name string
		var rlsEnabled bool

		if err := rows.Scan(&schema, &name, &rlsEnabled); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}

		key := fmt.Sprintf("%s.%s", schema, name)
		tableMap[key] = &TableInfo{
			Schema:     schema,
			Name:       name,
			Type:       "table",
			RLSEnabled: rlsEnabled,
		}
		tableKeys = append(tableKeys, key)
	}

	if len(tableMap) == 0 {
		return []TableInfo{}, nil
	}

	// Batch fetch all metadata
	if err := si.batchFetchTableMetadata(ctx, schemas, tableMap, "table"); err != nil {
		return nil, err
	}

	// Build result slice in original order
	tables := make([]TableInfo, 0, len(tableKeys))
	for _, key := range tableKeys {
		if info, ok := tableMap[key]; ok {
			tables = append(tables, *info)
		}
	}

	return tables, nil
}

// GetTableInfo retrieves detailed information about a specific table
func (si *SchemaInspector) GetTableInfo(ctx context.Context, schema, table string) (*TableInfo, error) {
	tableInfo := &TableInfo{
		Schema: schema,
		Name:   table,
	}

	// Get columns
	columns, err := si.getColumns(ctx, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	tableInfo.Columns = columns

	// Get primary key
	primaryKey, err := si.getPrimaryKey(ctx, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary key: %w", err)
	}
	tableInfo.PrimaryKey = primaryKey

	// Get foreign keys
	foreignKeys, err := si.getForeignKeys(ctx, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}
	tableInfo.ForeignKeys = foreignKeys

	// Get indexes
	indexes, err := si.getIndexes(ctx, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	tableInfo.Indexes = indexes

	// Mark primary key columns
	for i := range tableInfo.Columns {
		for _, pk := range tableInfo.PrimaryKey {
			if tableInfo.Columns[i].Name == pk {
				tableInfo.Columns[i].IsPrimaryKey = true
				break
			}
		}
	}

	// Mark foreign key columns
	for i := range tableInfo.Columns {
		for _, fk := range tableInfo.ForeignKeys {
			if tableInfo.Columns[i].Name == fk.ColumnName {
				tableInfo.Columns[i].IsForeignKey = true
				break
			}
		}
	}

	return tableInfo, nil
}

// getColumns retrieves column information for a table, view, or materialized view
func (si *SchemaInspector) getColumns(ctx context.Context, schema, table string) ([]ColumnInfo, error) {
	// First try information_schema.columns (works for tables and regular views)
	query := `
		SELECT
			column_name,
			CASE
				WHEN data_type = 'USER-DEFINED' THEN udt_name
				ELSE data_type
			END as data_type,
			is_nullable,
			column_default,
			character_maximum_length,
			ordinal_position
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := si.conn.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullable string
		var maxLength *int32

		err := rows.Scan(
			&col.Name,
			&col.DataType,
			&isNullable,
			&col.DefaultValue,
			&maxLength,
			&col.Position,
		)
		if err != nil {
			return nil, err
		}

		col.IsNullable = isNullable == "YES"
		if maxLength != nil {
			length := int(*maxLength)
			col.MaxLength = &length
		}

		columns = append(columns, col)
	}

	// If no columns found, it might be a materialized view
	// Materialized views are NOT in information_schema.columns, use pg_attribute instead
	if len(columns) == 0 {
		columns, err = si.getMaterializedViewColumns(ctx, schema, table)
		if err != nil {
			return nil, err
		}
	}

	return columns, nil
}

// getMaterializedViewColumns retrieves column information for a materialized view using pg_catalog
func (si *SchemaInspector) getMaterializedViewColumns(ctx context.Context, schema, table string) ([]ColumnInfo, error) {
	query := `
		SELECT
			a.attname AS column_name,
			pg_catalog.format_type(a.atttypid, a.atttypmod) AS data_type,
			NOT a.attnotnull AS is_nullable,
			pg_get_expr(d.adbin, d.adrelid) AS column_default,
			a.attnum AS ordinal_position
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE n.nspname = $1
		  AND c.relname = $2
		  AND c.relkind = 'm'  -- 'm' = materialized view
		  AND a.attnum > 0     -- skip system columns
		  AND NOT a.attisdropped
		ORDER BY a.attnum
	`

	rows, err := si.conn.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullable bool

		err := rows.Scan(
			&col.Name,
			&col.DataType,
			&isNullable,
			&col.DefaultValue,
			&col.Position,
		)
		if err != nil {
			return nil, err
		}

		col.IsNullable = isNullable
		columns = append(columns, col)
	}

	return columns, nil
}

// getPrimaryKey retrieves primary key columns for a table
func (si *SchemaInspector) getPrimaryKey(ctx context.Context, schema, table string) ([]string, error) {
	query := `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		JOIN pg_class c ON c.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1
			AND c.relname = $2
			AND i.indisprimary
		ORDER BY array_position(i.indkey, a.attnum)
	`

	rows, err := si.conn.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var primaryKey []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		primaryKey = append(primaryKey, column)
	}

	return primaryKey, nil
}

// getForeignKeys retrieves foreign key information for a table
func (si *SchemaInspector) getForeignKeys(ctx context.Context, schema, table string) ([]ForeignKey, error) {
	query := `
		SELECT
			tc.constraint_name,
			kcu.column_name,
			ccu.table_schema || '.' || ccu.table_name AS referenced_table,
			ccu.column_name AS referenced_column,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		JOIN information_schema.referential_constraints AS rc
			ON rc.constraint_name = tc.constraint_name
			AND rc.constraint_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = $1
			AND tc.table_name = $2
	`

	rows, err := si.conn.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var foreignKeys []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		err := rows.Scan(
			&fk.Name,
			&fk.ColumnName,
			&fk.ReferencedTable,
			&fk.ReferencedColumn,
			&fk.OnDelete,
			&fk.OnUpdate,
		)
		if err != nil {
			return nil, err
		}
		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, nil
}

// getIndexes retrieves index information for a table
func (si *SchemaInspector) getIndexes(ctx context.Context, schema, table string) ([]IndexInfo, error) {
	query := `
		SELECT
			i.relname AS index_name,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum)) AS columns,
			ix.indisunique,
			ix.indisprimary
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = $1
			AND t.relname = $2
		GROUP BY i.relname, ix.indisunique, ix.indisprimary
		ORDER BY i.relname
	`

	rows, err := si.conn.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		err := rows.Scan(
			&idx.Name,
			&idx.Columns,
			&idx.IsUnique,
			&idx.IsPrimary,
		)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

// batchFetchTableMetadata fetches columns, primary keys, foreign keys, and indexes
// for all tables in the map using batched queries. This avoids the N+1 query problem.
// The objectType parameter specifies whether we're fetching for "table", "view", or "materialized_view".
func (si *SchemaInspector) batchFetchTableMetadata(ctx context.Context, schemas []string, tableMap map[string]*TableInfo, objectType string) error {
	// Batch fetch columns
	columns, err := si.batchGetColumns(ctx, schemas, objectType)
	if err != nil {
		return fmt.Errorf("failed to batch get columns: %w", err)
	}

	// Assign columns to tables
	for key, cols := range columns {
		if info, ok := tableMap[key]; ok {
			info.Columns = cols
		}
	}

	// For tables, also fetch primary keys, foreign keys, and indexes
	if objectType == "table" {
		// Batch fetch primary keys
		primaryKeys, err := si.batchGetPrimaryKeys(ctx, schemas)
		if err != nil {
			return fmt.Errorf("failed to batch get primary keys: %w", err)
		}

		for key, pks := range primaryKeys {
			if info, ok := tableMap[key]; ok {
				info.PrimaryKey = pks
				// Mark primary key columns
				for i := range info.Columns {
					for _, pk := range pks {
						if info.Columns[i].Name == pk {
							info.Columns[i].IsPrimaryKey = true
							break
						}
					}
				}
			}
		}

		// Batch fetch foreign keys
		foreignKeys, err := si.batchGetForeignKeys(ctx, schemas)
		if err != nil {
			return fmt.Errorf("failed to batch get foreign keys: %w", err)
		}

		for key, fks := range foreignKeys {
			if info, ok := tableMap[key]; ok {
				info.ForeignKeys = fks
				// Mark foreign key columns
				for i := range info.Columns {
					for _, fk := range fks {
						if info.Columns[i].Name == fk.ColumnName {
							info.Columns[i].IsForeignKey = true
							break
						}
					}
				}
			}
		}

		// Batch fetch indexes
		indexes, err := si.batchGetIndexes(ctx, schemas)
		if err != nil {
			return fmt.Errorf("failed to batch get indexes: %w", err)
		}

		for key, idxs := range indexes {
			if info, ok := tableMap[key]; ok {
				info.Indexes = idxs
			}
		}
	}

	// For materialized views, fetch indexes (they can have indexes)
	if objectType == "materialized_view" {
		indexes, err := si.batchGetIndexes(ctx, schemas)
		if err != nil {
			return fmt.Errorf("failed to batch get indexes: %w", err)
		}

		for key, idxs := range indexes {
			if info, ok := tableMap[key]; ok {
				info.Indexes = idxs
			}
		}
	}

	return nil
}

// batchGetColumns retrieves columns for all tables/views in the specified schemas in a single query.
// Returns a map from "schema.table" to column list.
func (si *SchemaInspector) batchGetColumns(ctx context.Context, schemas []string, objectType string) (map[string][]ColumnInfo, error) {
	result := make(map[string][]ColumnInfo)

	if objectType == "materialized_view" {
		// Materialized views need pg_attribute query
		return si.batchGetMaterializedViewColumns(ctx, schemas)
	}

	// For tables and regular views, use information_schema
	query := `
		SELECT
			table_schema,
			table_name,
			column_name,
			CASE
				WHEN data_type = 'USER-DEFINED' THEN udt_name
				ELSE data_type
			END as data_type,
			is_nullable,
			column_default,
			character_maximum_length,
			ordinal_position
		FROM information_schema.columns
		WHERE table_schema = ANY($1)
		ORDER BY table_schema, table_name, ordinal_position
	`

	rows, err := si.conn.Query(ctx, query, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table string
		var col ColumnInfo
		var isNullable string
		var maxLength *int32

		err := rows.Scan(
			&schema,
			&table,
			&col.Name,
			&col.DataType,
			&isNullable,
			&col.DefaultValue,
			&maxLength,
			&col.Position,
		)
		if err != nil {
			return nil, err
		}

		col.IsNullable = isNullable == "YES"
		if maxLength != nil {
			length := int(*maxLength)
			col.MaxLength = &length
		}

		key := fmt.Sprintf("%s.%s", schema, table)
		result[key] = append(result[key], col)
	}

	return result, nil
}

// batchGetMaterializedViewColumns retrieves columns for all materialized views using pg_attribute.
func (si *SchemaInspector) batchGetMaterializedViewColumns(ctx context.Context, schemas []string) (map[string][]ColumnInfo, error) {
	result := make(map[string][]ColumnInfo)

	query := `
		SELECT
			n.nspname AS schema_name,
			c.relname AS table_name,
			a.attname AS column_name,
			pg_catalog.format_type(a.atttypid, a.atttypmod) AS data_type,
			NOT a.attnotnull AS is_nullable,
			pg_get_expr(d.adbin, d.adrelid) AS column_default,
			a.attnum AS ordinal_position
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE n.nspname = ANY($1)
		  AND c.relkind = 'm'  -- 'm' = materialized view
		  AND a.attnum > 0     -- skip system columns
		  AND NOT a.attisdropped
		ORDER BY n.nspname, c.relname, a.attnum
	`

	rows, err := si.conn.Query(ctx, query, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table string
		var col ColumnInfo
		var isNullable bool

		err := rows.Scan(
			&schema,
			&table,
			&col.Name,
			&col.DataType,
			&isNullable,
			&col.DefaultValue,
			&col.Position,
		)
		if err != nil {
			return nil, err
		}

		col.IsNullable = isNullable

		key := fmt.Sprintf("%s.%s", schema, table)
		result[key] = append(result[key], col)
	}

	return result, nil
}

// batchGetPrimaryKeys retrieves primary keys for all tables in the specified schemas.
func (si *SchemaInspector) batchGetPrimaryKeys(ctx context.Context, schemas []string) (map[string][]string, error) {
	result := make(map[string][]string)

	query := `
		SELECT
			n.nspname AS schema_name,
			c.relname AS table_name,
			a.attname AS column_name,
			array_position(i.indkey, a.attnum) AS key_position
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		JOIN pg_class c ON c.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1)
			AND i.indisprimary
		ORDER BY n.nspname, c.relname, array_position(i.indkey, a.attnum)
	`

	rows, err := si.conn.Query(ctx, query, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table, column string
		var position int

		if err := rows.Scan(&schema, &table, &column, &position); err != nil {
			return nil, err
		}

		key := fmt.Sprintf("%s.%s", schema, table)
		result[key] = append(result[key], column)
	}

	return result, nil
}

// batchGetForeignKeys retrieves foreign keys for all tables in the specified schemas.
func (si *SchemaInspector) batchGetForeignKeys(ctx context.Context, schemas []string) (map[string][]ForeignKey, error) {
	result := make(map[string][]ForeignKey)

	query := `
		SELECT
			tc.table_schema,
			tc.table_name,
			tc.constraint_name,
			kcu.column_name,
			ccu.table_schema || '.' || ccu.table_name AS referenced_table,
			ccu.column_name AS referenced_column,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		JOIN information_schema.referential_constraints AS rc
			ON rc.constraint_name = tc.constraint_name
			AND rc.constraint_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = ANY($1)
		ORDER BY tc.table_schema, tc.table_name, tc.constraint_name
	`

	rows, err := si.conn.Query(ctx, query, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table string
		var fk ForeignKey
		err := rows.Scan(
			&schema,
			&table,
			&fk.Name,
			&fk.ColumnName,
			&fk.ReferencedTable,
			&fk.ReferencedColumn,
			&fk.OnDelete,
			&fk.OnUpdate,
		)
		if err != nil {
			return nil, err
		}

		key := fmt.Sprintf("%s.%s", schema, table)
		result[key] = append(result[key], fk)
	}

	return result, nil
}

// batchGetIndexes retrieves indexes for all tables in the specified schemas.
func (si *SchemaInspector) batchGetIndexes(ctx context.Context, schemas []string) (map[string][]IndexInfo, error) {
	result := make(map[string][]IndexInfo)

	query := `
		SELECT
			n.nspname AS schema_name,
			t.relname AS table_name,
			i.relname AS index_name,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum)) AS columns,
			ix.indisunique,
			ix.indisprimary
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = ANY($1)
		GROUP BY n.nspname, t.relname, i.relname, ix.indisunique, ix.indisprimary
		ORDER BY n.nspname, t.relname, i.relname
	`

	rows, err := si.conn.Query(ctx, query, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table string
		var idx IndexInfo
		err := rows.Scan(
			&schema,
			&table,
			&idx.Name,
			&idx.Columns,
			&idx.IsUnique,
			&idx.IsPrimary,
		)
		if err != nil {
			return nil, err
		}

		key := fmt.Sprintf("%s.%s", schema, table)
		result[key] = append(result[key], idx)
	}

	return result, nil
}

// GetSchemas retrieves all available schemas
func (si *SchemaInspector) GetSchemas(ctx context.Context) ([]string, error) {
	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema')
			AND schema_name NOT LIKE 'pg_%'
		ORDER BY schema_name
	`

	rows, err := si.conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}

	return schemas, nil
}

// GetAllViews retrieves information about all views in the specified schemas.
// This uses batched queries to avoid N+1 query patterns.
func (si *SchemaInspector) GetAllViews(ctx context.Context, schemas ...string) ([]TableInfo, error) {
	if len(schemas) == 0 {
		schemas = []string{"public"}
	}

	// Query to get all views from specified schemas
	query := `
		SELECT
			schemaname,
			viewname
		FROM pg_views
		WHERE schemaname = ANY($1)
			AND schemaname NOT IN ('information_schema', 'pg_catalog')
		ORDER BY schemaname, viewname
	`

	rows, err := si.conn.Query(ctx, query, schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to query views: %w", err)
	}
	defer rows.Close()

	// Collect all view names and build initial TableInfo map
	viewMap := make(map[string]*TableInfo)
	var viewKeys []string

	for rows.Next() {
		var schema, name string

		if err := rows.Scan(&schema, &name); err != nil {
			return nil, fmt.Errorf("failed to scan view: %w", err)
		}

		key := fmt.Sprintf("%s.%s", schema, name)
		viewMap[key] = &TableInfo{
			Schema:     schema,
			Name:       name,
			Type:       "view",
			RLSEnabled: false,
		}
		viewKeys = append(viewKeys, key)
	}

	if len(viewMap) == 0 {
		return []TableInfo{}, nil
	}

	// Batch fetch all metadata (views don't have primary keys, foreign keys, or indexes)
	if err := si.batchFetchTableMetadata(ctx, schemas, viewMap, "view"); err != nil {
		return nil, err
	}

	// Build result slice in original order
	views := make([]TableInfo, 0, len(viewKeys))
	for _, key := range viewKeys {
		if info, ok := viewMap[key]; ok {
			views = append(views, *info)
		}
	}

	return views, nil
}

// GetAllMaterializedViews retrieves information about all materialized views in the specified schemas.
// This uses batched queries to avoid N+1 query patterns.
func (si *SchemaInspector) GetAllMaterializedViews(ctx context.Context, schemas ...string) ([]TableInfo, error) {
	if len(schemas) == 0 {
		schemas = []string{"public"}
	}

	// Query to get all materialized views from specified schemas
	query := `
		SELECT
			schemaname,
			matviewname
		FROM pg_matviews
		WHERE schemaname = ANY($1)
			AND schemaname NOT IN ('information_schema', 'pg_catalog')
		ORDER BY schemaname, matviewname
	`

	rows, err := si.conn.Query(ctx, query, schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to query materialized views: %w", err)
	}
	defer rows.Close()

	// Collect all materialized view names and build initial TableInfo map
	matviewMap := make(map[string]*TableInfo)
	var matviewKeys []string

	for rows.Next() {
		var schema, name string

		if err := rows.Scan(&schema, &name); err != nil {
			return nil, fmt.Errorf("failed to scan materialized view: %w", err)
		}

		key := fmt.Sprintf("%s.%s", schema, name)
		matviewMap[key] = &TableInfo{
			Schema:     schema,
			Name:       name,
			Type:       "materialized_view",
			RLSEnabled: false,
		}
		matviewKeys = append(matviewKeys, key)
	}

	if len(matviewMap) == 0 {
		return []TableInfo{}, nil
	}

	// Batch fetch all metadata (materialized views have indexes but no primary/foreign keys)
	if err := si.batchFetchTableMetadata(ctx, schemas, matviewMap, "materialized_view"); err != nil {
		return nil, err
	}

	// Build result slice in original order
	matviews := make([]TableInfo, 0, len(matviewKeys))
	for _, key := range matviewKeys {
		if info, ok := matviewMap[key]; ok {
			matviews = append(matviews, *info)
		}
	}

	return matviews, nil
}

// FunctionInfo represents metadata about a database function
type FunctionInfo struct {
	Schema      string          `json:"schema"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []FunctionParam `json:"parameters"`
	ReturnType  string          `json:"return_type"`
	IsSetOf     bool            `json:"is_set_of"`
	Volatility  string          `json:"volatility"` // VOLATILE, STABLE, IMMUTABLE
	Language    string          `json:"language"`
}

// FunctionParam represents a function parameter
type FunctionParam struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Mode       string `json:"mode"` // IN, OUT, INOUT
	HasDefault bool   `json:"has_default"`
	Position   int    `json:"position"`
}

// GetAllFunctions retrieves information about all functions in the specified schemas
func (si *SchemaInspector) GetAllFunctions(ctx context.Context, schemas ...string) ([]FunctionInfo, error) {
	if len(schemas) == 0 {
		schemas = []string{"public"}
	}

	var functions []FunctionInfo

	// Query to get all functions from specified schemas
	// Excludes extension functions (PostGIS, pg_trgm, etc.)
	query := `
		SELECT
			n.nspname as schema_name,
			p.proname as function_name,
			pg_catalog.obj_description(p.oid, 'pg_proc') as description,
			pg_catalog.pg_get_function_result(p.oid) as return_type,
			p.proretset as is_set_of,
			CASE p.provolatile
				WHEN 'i' THEN 'IMMUTABLE'
				WHEN 's' THEN 'STABLE'
				WHEN 'v' THEN 'VOLATILE'
			END as volatility,
			l.lanname as language
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		JOIN pg_language l ON l.oid = p.prolang
		LEFT JOIN pg_depend d ON d.objid = p.oid AND d.deptype = 'e'
		WHERE n.nspname = ANY($1)
			AND n.nspname NOT IN ('pg_catalog', 'information_schema')
			AND p.prokind = 'f'  -- Only functions, not procedures
			AND d.objid IS NULL  -- Exclude extension functions
		ORDER BY n.nspname, p.proname
	`

	rows, err := si.conn.Query(ctx, query, schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to query functions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fn FunctionInfo
		var description *string

		if err := rows.Scan(
			&fn.Schema,
			&fn.Name,
			&description,
			&fn.ReturnType,
			&fn.IsSetOf,
			&fn.Volatility,
			&fn.Language,
		); err != nil {
			return nil, fmt.Errorf("failed to scan function: %w", err)
		}

		if description != nil {
			fn.Description = *description
		}

		// Get function parameters
		params, err := si.getFunctionParameters(ctx, fn.Schema, fn.Name)
		if err != nil {
			log.Warn().Err(err).Str("function", fmt.Sprintf("%s.%s", fn.Schema, fn.Name)).Msg("Failed to get function parameters")
			continue
		}
		fn.Parameters = params

		functions = append(functions, fn)
	}

	return functions, nil
}

// getFunctionParameters retrieves parameter information for a function
func (si *SchemaInspector) getFunctionParameters(ctx context.Context, schema, function string) ([]FunctionParam, error) {
	query := `
		SELECT
			COALESCE(p.parameter_name, '') as param_name,
			p.data_type,
			p.parameter_mode,
			COALESCE(p.parameter_default, '') != '' as has_default,
			p.ordinal_position
		FROM information_schema.parameters p
		WHERE p.specific_schema = $1
			AND p.specific_name IN (
				SELECT pg_proc.proname || '_' || pg_proc.oid
				FROM pg_proc
				JOIN pg_namespace ON pg_namespace.oid = pg_proc.pronamespace
				WHERE pg_namespace.nspname = $1 AND pg_proc.proname = $2
			)
		ORDER BY p.ordinal_position
	`

	rows, err := si.conn.Query(ctx, query, schema, function)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var params []FunctionParam
	for rows.Next() {
		var param FunctionParam
		if err := rows.Scan(
			&param.Name,
			&param.Type,
			&param.Mode,
			&param.HasDefault,
			&param.Position,
		); err != nil {
			return nil, err
		}
		params = append(params, param)
	}

	return params, nil
}

// BuildRESTPath builds a REST API path for a table
func (si *SchemaInspector) BuildRESTPath(table TableInfo) string {
	// Convert table name to plural form (simple pluralization)
	tableName := table.Name
	if !strings.HasSuffix(tableName, "s") {
		switch {
		case strings.HasSuffix(tableName, "x"),
			strings.HasSuffix(tableName, "ch"),
			strings.HasSuffix(tableName, "sh"):
			// box -> boxes, match -> matches, dish -> dishes
			tableName += "es"
		case strings.HasSuffix(tableName, "y") && len(tableName) >= 2:
			// Check if preceded by consonant (y -> ies) or vowel (y -> ys)
			beforeY := tableName[len(tableName)-2]
			if beforeY == 'a' || beforeY == 'e' || beforeY == 'i' || beforeY == 'o' || beforeY == 'u' {
				tableName += "s" // vowel + y: key -> keys, day -> days
			} else {
				tableName = strings.TrimSuffix(tableName, "y") + "ies" // consonant + y: story -> stories
			}
		default:
			tableName += "s"
		}
	}

	if table.Schema != "public" {
		return fmt.Sprintf("/api/rest/%s/%s", table.Schema, tableName)
	}
	return fmt.Sprintf("/api/rest/%s", tableName)
}

// VectorColumnInfo represents metadata about a vector column (pgvector)
type VectorColumnInfo struct {
	SchemaName string `json:"schema_name"`
	TableName  string `json:"table_name"`
	ColumnName string `json:"column_name"`
	Dimensions int    `json:"dimensions"` // -1 if variable/unspecified
}

// GetVectorColumns retrieves all vector columns in the specified schema and table
// If table is empty, returns all vector columns in the schema
// If both schema and table are empty, returns all vector columns in public schema
func (si *SchemaInspector) GetVectorColumns(ctx context.Context, schema, table string) ([]VectorColumnInfo, error) {
	if schema == "" {
		schema = "public"
	}

	// First check if pgvector extension is installed
	var hasVector bool
	err := si.conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'vector')").Scan(&hasVector)
	if err != nil || !hasVector {
		// pgvector not installed, return empty list
		return []VectorColumnInfo{}, nil
	}

	// Query to find all vector columns
	// The dimensions are stored in the typmod field of pg_attribute
	query := `
		SELECT
			n.nspname as schema_name,
			c.relname as table_name,
			a.attname as column_name,
			CASE
				WHEN a.atttypmod = -1 THEN -1
				ELSE a.atttypmod
			END as dimensions
		FROM pg_attribute a
		JOIN pg_class c ON a.attrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		JOIN pg_type t ON a.atttypid = t.oid
		WHERE t.typname = 'vector'
			AND a.attnum > 0
			AND NOT a.attisdropped
			AND n.nspname = $1
	`

	args := []interface{}{schema}
	if table != "" {
		query += " AND c.relname = $2"
		args = append(args, table)
	}

	query += " ORDER BY n.nspname, c.relname, a.attnum"

	rows, err := si.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query vector columns: %w", err)
	}
	defer rows.Close()

	var columns []VectorColumnInfo
	for rows.Next() {
		var col VectorColumnInfo
		if err := rows.Scan(&col.SchemaName, &col.TableName, &col.ColumnName, &col.Dimensions); err != nil {
			return nil, fmt.Errorf("failed to scan vector column: %w", err)
		}
		columns = append(columns, col)
	}

	return columns, nil
}

// IsPgVectorInstalled checks if the pgvector extension is installed
func (si *SchemaInspector) IsPgVectorInstalled(ctx context.Context) (bool, string, error) {
	var version *string
	err := si.conn.QueryRow(ctx, `
		SELECT installed_version
		FROM pg_available_extensions
		WHERE name = 'vector'
	`).Scan(&version)

	if err != nil {
		// Extension not in catalog
		return false, "", nil
	}

	if version == nil {
		// Extension available but not installed
		return false, "", nil
	}

	return true, *version, nil
}
