package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type querier interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

type SchemaInspector struct {
	conn *Connection
}

func (si *SchemaInspector) q() querier {
	return si.conn
}

func PoolQuerier(pool *pgxpool.Pool) querier {
	return poolQuerier{Pool: pool}
}

type poolQuerier struct {
	*pgxpool.Pool
}

type TableInfo struct {
	Schema      string                 `json:"schema"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	RESTPath    string                 `json:"rest_path,omitempty"`
	Columns     []ColumnInfo           `json:"columns"`
	PrimaryKey  []string               `json:"primary_key"`
	ForeignKeys []ForeignKey           `json:"foreign_keys"`
	Indexes     []IndexInfo            `json:"indexes"`
	RLSEnabled  bool                   `json:"rls_enabled"`
	ColumnMap   map[string]*ColumnInfo `json:"-"`
}

func (t *TableInfo) BuildColumnMap() {
	t.ColumnMap = make(map[string]*ColumnInfo, len(t.Columns))
	for i := range t.Columns {
		t.ColumnMap[t.Columns[i].Name] = &t.Columns[i]
	}
}

func (t *TableInfo) GetColumn(name string) *ColumnInfo {
	if t.ColumnMap != nil {
		return t.ColumnMap[name]
	}
	for i := range t.Columns {
		if t.Columns[i].Name == name {
			return &t.Columns[i]
		}
	}
	return nil
}

func (t *TableInfo) HasColumn(name string) bool {
	return t.GetColumn(name) != nil
}

type ColumnInfo struct {
	Name         string           `json:"name"`
	DataType     string           `json:"data_type"`
	IsNullable   bool             `json:"is_nullable"`
	DefaultValue *string          `json:"default_value"`
	IsPrimaryKey bool             `json:"is_primary_key"`
	IsForeignKey bool             `json:"is_foreign_key"`
	IsUnique     bool             `json:"is_unique"`
	MaxLength    *int             `json:"max_length"`
	Position     int              `json:"position"`
	Description  string           `json:"description,omitempty"`
	JSONBSchema  *JSONBSchemaInfo `json:"jsonb_schema,omitempty"`
}

type JSONBSchemaInfo struct {
	Properties map[string]JSONBProperty `json:"properties,omitempty"`
	Required   []string                 `json:"required,omitempty"`
}

type JSONBProperty struct {
	Type        string                   `json:"type"`
	Description string                   `json:"description,omitempty"`
	Properties  map[string]JSONBProperty `json:"properties,omitempty"`
	Items       *JSONBProperty           `json:"items,omitempty"`
}

type ForeignKey struct {
	Name             string `json:"name"`
	ColumnName       string `json:"column_name"`
	ReferencedTable  string `json:"referenced_table"`
	ReferencedColumn string `json:"referenced_column"`
	OnDelete         string `json:"on_delete"`
	OnUpdate         string `json:"on_update"`
}

type IndexInfo struct {
	Name      string   `json:"name"`
	Columns   []string `json:"columns"`
	IsUnique  bool     `json:"is_unique"`
	IsPrimary bool     `json:"is_primary"`
}

func NewSchemaInspector(conn *Connection) *SchemaInspector {
	return &SchemaInspector{conn: conn}
}

func (si *SchemaInspector) GetAllTables(ctx context.Context, schemas ...string) ([]TableInfo, error) {
	return si.GetAllTablesFromQ(ctx, si.q(), schemas...)
}

func (si *SchemaInspector) GetAllTablesFromQ(ctx context.Context, q querier, schemas ...string) ([]TableInfo, error) {
	LogSchemaIntrospection(ctx, "GetAllTables", map[string]interface{}{"schemas": schemas})
	if len(schemas) == 0 {
		schemas = []string{"public"}
	}

	query := `
		SELECT
			n.nspname as schemaname,
			c.relname as tablename,
			CASE
				WHEN c.relrowsecurity THEN true
				ELSE false
			END as rls_enabled
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1)
			AND c.relkind IN ('r', 'f')
			AND c.relname NOT LIKE 'pg_%'
			AND c.relname NOT LIKE '_fluxbase.%'
			AND n.nspname NOT IN ('information_schema', 'pg_catalog', '_fluxbase')
		ORDER BY n.nspname, c.relname
	`

	rows, err := q.Query(ctx, query, schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	tableMap := make(map[string]*TableInfo)
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

	if err := si.batchFetchTableMetadata(ctx, q, schemas, tableMap, "table"); err != nil {
		return nil, err
	}

	tables := make([]TableInfo, 0, len(tableKeys))
	for _, key := range tableKeys {
		if info, ok := tableMap[key]; ok {
			tables = append(tables, *info)
		}
	}

	return tables, nil
}

func (si *SchemaInspector) GetTableInfo(ctx context.Context, schema, table string) (*TableInfo, error) {
	return si.GetTableInfoFromQ(ctx, si.q(), schema, table)
}

func (si *SchemaInspector) GetTableInfoFromQ(ctx context.Context, q querier, schema, table string) (*TableInfo, error) {
	LogSchemaIntrospection(ctx, "GetTableInfo", map[string]interface{}{"schema": schema, "table": table})
	tableInfo := &TableInfo{
		Schema: schema,
		Name:   table,
	}

	columns, err := si.getColumns(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	tableInfo.Columns = columns

	primaryKey, err := si.getPrimaryKey(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary key: %w", err)
	}
	tableInfo.PrimaryKey = primaryKey

	foreignKeys, err := si.getForeignKeys(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}
	tableInfo.ForeignKeys = foreignKeys

	indexes, err := si.getIndexes(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	tableInfo.Indexes = indexes

	for i := range tableInfo.Columns {
		for _, pk := range tableInfo.PrimaryKey {
			if tableInfo.Columns[i].Name == pk {
				tableInfo.Columns[i].IsPrimaryKey = true
				break
			}
		}
	}

	for i := range tableInfo.Columns {
		for _, fk := range tableInfo.ForeignKeys {
			if tableInfo.Columns[i].Name == fk.ColumnName {
				tableInfo.Columns[i].IsForeignKey = true
				break
			}
		}
	}

	tableInfo.BuildColumnMap()

	return tableInfo, nil
}

func (si *SchemaInspector) getColumns(ctx context.Context, q querier, schema, table string) ([]ColumnInfo, error) {
	query := `
		SELECT
			c.column_name,
			CASE
				WHEN c.data_type = 'USER-DEFINED' THEN c.udt_name
				ELSE c.data_type
			END as data_type,
			c.is_nullable,
			c.column_default,
			c.character_maximum_length,
			c.ordinal_position,
			COALESCE(pg_catalog.col_description(
				(c.table_schema || '.' || c.table_name)::regclass::oid,
				c.ordinal_position
			), '') as column_comment
		FROM information_schema.columns c
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := q.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullable string
		var maxLength *int32
		var comment string

		err := rows.Scan(
			&col.Name,
			&col.DataType,
			&isNullable,
			&col.DefaultValue,
			&maxLength,
			&col.Position,
			&comment,
		)
		if err != nil {
			return nil, err
		}

		col.IsNullable = isNullable == "YES"
		if maxLength != nil {
			length := int(*maxLength)
			col.MaxLength = &length
		}

		col.Description, col.JSONBSchema = parseColumnComment(comment)

		columns = append(columns, col)
	}

	if len(columns) == 0 {
		columns, err = si.getMaterializedViewColumns(ctx, q, schema, table)
		if err != nil {
			return nil, err
		}
	}

	return columns, nil
}

func (si *SchemaInspector) getMaterializedViewColumns(ctx context.Context, q querier, schema, table string) ([]ColumnInfo, error) {
	query := `
		SELECT
			a.attname AS column_name,
			pg_catalog.format_type(a.atttypid, a.atttypmod) AS data_type,
			NOT a.attnotnull AS is_nullable,
			pg_get_expr(d.adbin, d.adrelid) AS column_default,
			a.attnum AS ordinal_position,
			COALESCE(pg_catalog.col_description(c.oid, a.attnum), '') as column_comment
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE n.nspname = $1
		  AND c.relname = $2
		  AND c.relkind = 'm'
		  AND a.attnum > 0
		  AND NOT a.attisdropped
		ORDER BY a.attnum
	`

	rows, err := q.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullable bool
		var comment string

		err := rows.Scan(
			&col.Name,
			&col.DataType,
			&isNullable,
			&col.DefaultValue,
			&col.Position,
			&comment,
		)
		if err != nil {
			return nil, err
		}

		col.IsNullable = isNullable
		col.Description, col.JSONBSchema = parseColumnComment(comment)

		columns = append(columns, col)
	}

	return columns, nil
}

func parseColumnComment(comment string) (description string, schema *JSONBSchemaInfo) {
	if comment == "" {
		return "", nil
	}

	if strings.Contains(comment, "_fluxbase_jsonb_schema") {
		var data struct {
			FluxbaseJSONBSchema *JSONBSchemaInfo `json:"_fluxbase_jsonb_schema"`
		}
		if err := json.Unmarshal([]byte(comment), &data); err == nil && data.FluxbaseJSONBSchema != nil {
			return "", data.FluxbaseJSONBSchema
		}
	}

	return comment, nil
}

func (si *SchemaInspector) getPrimaryKey(ctx context.Context, q querier, schema, table string) ([]string, error) {
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

	rows, err := q.Query(ctx, query, schema, table)
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

func (si *SchemaInspector) getForeignKeys(ctx context.Context, q querier, schema, table string) ([]ForeignKey, error) {
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

	rows, err := q.Query(ctx, query, schema, table)
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

func (si *SchemaInspector) getIndexes(ctx context.Context, q querier, schema, table string) ([]IndexInfo, error) {
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

	rows, err := q.Query(ctx, query, schema, table)
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

func (si *SchemaInspector) batchFetchTableMetadata(ctx context.Context, q querier, schemas []string, tableMap map[string]*TableInfo, objectType string) error {
	columns, err := si.batchGetColumns(ctx, q, schemas, objectType)
	if err != nil {
		return fmt.Errorf("failed to batch get columns: %w", err)
	}

	for key, cols := range columns {
		if info, ok := tableMap[key]; ok {
			info.Columns = cols
		}
	}

	if objectType == "table" {
		primaryKeys, err := si.batchGetPrimaryKeys(ctx, q, schemas)
		if err != nil {
			return fmt.Errorf("failed to batch get primary keys: %w", err)
		}

		for key, pks := range primaryKeys {
			if info, ok := tableMap[key]; ok {
				info.PrimaryKey = pks
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

		foreignKeys, err := si.batchGetForeignKeys(ctx, q, schemas)
		if err != nil {
			return fmt.Errorf("failed to batch get foreign keys: %w", err)
		}

		for key, fks := range foreignKeys {
			if info, ok := tableMap[key]; ok {
				info.ForeignKeys = fks
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

		indexes, err := si.batchGetIndexes(ctx, q, schemas)
		if err != nil {
			return fmt.Errorf("failed to batch get indexes: %w", err)
		}

		for key, idxs := range indexes {
			if info, ok := tableMap[key]; ok {
				info.Indexes = idxs
			}
		}
	}

	if objectType == "materialized_view" {
		indexes, err := si.batchGetIndexes(ctx, q, schemas)
		if err != nil {
			return fmt.Errorf("failed to batch get indexes: %w", err)
		}

		for key, idxs := range indexes {
			if info, ok := tableMap[key]; ok {
				info.Indexes = idxs
			}
		}
	}

	for _, info := range tableMap {
		info.BuildColumnMap()
	}

	return nil
}

func (si *SchemaInspector) batchGetColumns(ctx context.Context, q querier, schemas []string, objectType string) (map[string][]ColumnInfo, error) {
	result := make(map[string][]ColumnInfo)

	if objectType == "materialized_view" {
		return si.batchGetMaterializedViewColumns(ctx, q, schemas)
	}

	query := `
		SELECT
			c.table_schema,
			c.table_name,
			c.column_name,
			CASE
				WHEN c.data_type = 'USER-DEFINED' THEN c.udt_name
				ELSE c.data_type
			END as data_type,
			c.is_nullable,
			c.column_default,
			c.character_maximum_length,
			c.ordinal_position,
			COALESCE(pg_catalog.col_description(
				(c.table_schema || '.' || c.table_name)::regclass::oid,
				c.ordinal_position
			), '') as column_comment
		FROM information_schema.columns c
		WHERE c.table_schema = ANY($1)
		ORDER BY c.table_schema, c.table_name, c.ordinal_position
	`

	rows, err := q.Query(ctx, query, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table string
		var col ColumnInfo
		var isNullable string
		var maxLength *int32
		var comment string

		err := rows.Scan(
			&schema,
			&table,
			&col.Name,
			&col.DataType,
			&isNullable,
			&col.DefaultValue,
			&maxLength,
			&col.Position,
			&comment,
		)
		if err != nil {
			return nil, err
		}

		col.IsNullable = isNullable == "YES"
		if maxLength != nil {
			length := int(*maxLength)
			col.MaxLength = &length
		}

		col.Description, col.JSONBSchema = parseColumnComment(comment)

		key := fmt.Sprintf("%s.%s", schema, table)
		result[key] = append(result[key], col)
	}

	return result, nil
}

func (si *SchemaInspector) batchGetMaterializedViewColumns(ctx context.Context, q querier, schemas []string) (map[string][]ColumnInfo, error) {
	result := make(map[string][]ColumnInfo)

	query := `
		SELECT
			n.nspname AS schema_name,
			c.relname AS table_name,
			a.attname AS column_name,
			pg_catalog.format_type(a.atttypid, a.atttypmod) AS data_type,
			NOT a.attnotnull AS is_nullable,
			pg_get_expr(d.adbin, d.adrelid) AS column_default,
			a.attnum AS ordinal_position,
			COALESCE(pg_catalog.col_description(c.oid, a.attnum), '') as column_comment
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE n.nspname = ANY($1)
		  AND c.relkind = 'm'
		  AND a.attnum > 0
		  AND NOT a.attisdropped
		ORDER BY n.nspname, c.relname, a.attnum
	`

	rows, err := q.Query(ctx, query, schemas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table string
		var col ColumnInfo
		var isNullable bool
		var comment string

		err := rows.Scan(
			&schema,
			&table,
			&col.Name,
			&col.DataType,
			&isNullable,
			&col.DefaultValue,
			&col.Position,
			&comment,
		)
		if err != nil {
			return nil, err
		}

		col.IsNullable = isNullable
		col.Description, col.JSONBSchema = parseColumnComment(comment)

		key := fmt.Sprintf("%s.%s", schema, table)
		result[key] = append(result[key], col)
	}

	return result, nil
}

func (si *SchemaInspector) batchGetPrimaryKeys(ctx context.Context, q querier, schemas []string) (map[string][]string, error) {
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

	rows, err := q.Query(ctx, query, schemas)
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

func (si *SchemaInspector) batchGetForeignKeys(ctx context.Context, q querier, schemas []string) (map[string][]ForeignKey, error) {
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
		JOIN information_schema.referential_constraints AS rc
			ON rc.constraint_name = tc.constraint_name
			AND rc.constraint_schema = tc.table_schema
		JOIN information_schema.key_column_usage AS ccu
			ON ccu.constraint_name = rc.unique_constraint_name
			AND ccu.table_schema = rc.unique_constraint_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = ANY($1)
		ORDER BY tc.table_schema, tc.table_name, tc.constraint_name
	`

	rows, err := q.Query(ctx, query, schemas)
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

func (si *SchemaInspector) batchGetIndexes(ctx context.Context, q querier, schemas []string) (map[string][]IndexInfo, error) {
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

	rows, err := q.Query(ctx, query, schemas)
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

func (si *SchemaInspector) GetSchemas(ctx context.Context) ([]string, error) {
	return si.GetSchemasFromQ(ctx, si.q())
}

func (si *SchemaInspector) GetSchemasFromQ(ctx context.Context, q querier) ([]string, error) {
	LogSchemaIntrospection(ctx, "GetSchemas", nil)
	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema')
			AND schema_name NOT LIKE 'pg_%'
		ORDER BY schema_name
	`

	rows, err := q.Query(ctx, query)
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

func (si *SchemaInspector) GetAllViews(ctx context.Context, schemas ...string) ([]TableInfo, error) {
	return si.GetAllViewsFromQ(ctx, si.q(), schemas...)
}

func (si *SchemaInspector) GetAllViewsFromQ(ctx context.Context, q querier, schemas ...string) ([]TableInfo, error) {
	LogSchemaIntrospection(ctx, "GetAllViews", map[string]interface{}{"schemas": schemas})
	if len(schemas) == 0 {
		schemas = []string{"public"}
	}

	query := `
		SELECT schemaname, viewname
		FROM pg_views
		WHERE schemaname = ANY($1)
			AND schemaname NOT IN ('information_schema', 'pg_catalog')
		ORDER BY schemaname, viewname
	`

	rows, err := q.Query(ctx, query, schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to query views: %w", err)
	}
	defer rows.Close()

	viewMap := make(map[string]*TableInfo)
	var viewKeys []string

	for rows.Next() {
		var schema, name string
		if err := rows.Scan(&schema, &name); err != nil {
			return nil, fmt.Errorf("failed to scan view: %w", err)
		}

		key := fmt.Sprintf("%s.%s", schema, name)
		viewMap[key] = &TableInfo{
			Schema: schema,
			Name:   name,
			Type:   "view",
		}
		viewKeys = append(viewKeys, key)
	}

	if len(viewMap) == 0 {
		return []TableInfo{}, nil
	}

	if err := si.batchFetchTableMetadata(ctx, q, schemas, viewMap, "view"); err != nil {
		return nil, err
	}

	views := make([]TableInfo, 0, len(viewKeys))
	for _, key := range viewKeys {
		if info, ok := viewMap[key]; ok {
			views = append(views, *info)
		}
	}

	return views, nil
}

func (si *SchemaInspector) GetAllMaterializedViews(ctx context.Context, schemas ...string) ([]TableInfo, error) {
	return si.GetAllMaterializedViewsFromQ(ctx, si.q(), schemas...)
}

func (si *SchemaInspector) GetAllMaterializedViewsFromQ(ctx context.Context, q querier, schemas ...string) ([]TableInfo, error) {
	LogSchemaIntrospection(ctx, "GetAllMaterializedViews", map[string]interface{}{"schemas": schemas})
	if len(schemas) == 0 {
		schemas = []string{"public"}
	}

	query := `
		SELECT schemaname, matviewname
		FROM pg_matviews
		WHERE schemaname = ANY($1)
			AND schemaname NOT IN ('information_schema', 'pg_catalog')
		ORDER BY schemaname, matviewname
	`

	rows, err := q.Query(ctx, query, schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to query materialized views: %w", err)
	}
	defer rows.Close()

	matviewMap := make(map[string]*TableInfo)
	var matviewKeys []string

	for rows.Next() {
		var schema, name string
		if err := rows.Scan(&schema, &name); err != nil {
			return nil, fmt.Errorf("failed to scan materialized view: %w", err)
		}

		key := fmt.Sprintf("%s.%s", schema, name)
		matviewMap[key] = &TableInfo{
			Schema: schema,
			Name:   name,
			Type:   "materialized_view",
		}
		matviewKeys = append(matviewKeys, key)
	}

	if len(matviewMap) == 0 {
		return []TableInfo{}, nil
	}

	if err := si.batchFetchTableMetadata(ctx, q, schemas, matviewMap, "materialized_view"); err != nil {
		return nil, err
	}

	matviews := make([]TableInfo, 0, len(matviewKeys))
	for _, key := range matviewKeys {
		if info, ok := matviewMap[key]; ok {
			matviews = append(matviews, *info)
		}
	}

	return matviews, nil
}

type FunctionInfo struct {
	Schema      string          `json:"schema"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []FunctionParam `json:"parameters"`
	ReturnType  string          `json:"return_type"`
	IsSetOf     bool            `json:"is_set_of"`
	Volatility  string          `json:"volatility"`
	Language    string          `json:"language"`
}

type FunctionParam struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Mode       string `json:"mode"`
	HasDefault bool   `json:"has_default"`
	Position   int    `json:"position"`
}

func (si *SchemaInspector) GetAllFunctions(ctx context.Context, schemas ...string) ([]FunctionInfo, error) {
	LogSchemaIntrospection(ctx, "GetAllFunctions", map[string]interface{}{"schemas": schemas})
	if len(schemas) == 0 {
		schemas = []string{"public"}
	}

	var functions []FunctionInfo

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
			AND p.prokind = 'f'
			AND d.objid IS NULL
		ORDER BY n.nspname, p.proname
	`

	rows, err := si.q().Query(ctx, query, schemas)
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

	rows, err := si.q().Query(ctx, query, schema, function)
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

func (si *SchemaInspector) BuildRESTPath(table TableInfo) string {
	tableName := table.Name
	if !strings.HasSuffix(tableName, "s") {
		switch {
		case strings.HasSuffix(tableName, "x"),
			strings.HasSuffix(tableName, "ch"),
			strings.HasSuffix(tableName, "sh"):
			tableName += "es"
		case strings.HasSuffix(tableName, "y") && len(tableName) >= 2:
			beforeY := tableName[len(tableName)-2]
			if beforeY == 'a' || beforeY == 'e' || beforeY == 'i' || beforeY == 'o' || beforeY == 'u' {
				tableName += "s"
			} else {
				tableName = strings.TrimSuffix(tableName, "y") + "ies"
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

type VectorColumnInfo struct {
	SchemaName string `json:"schema_name"`
	TableName  string `json:"table_name"`
	ColumnName string `json:"column_name"`
	Dimensions int    `json:"dimensions"`
}

func (si *SchemaInspector) GetVectorColumns(ctx context.Context, schema, table string) ([]VectorColumnInfo, error) {
	if schema == "" {
		schema = "public"
	}

	var hasVector bool
	err := si.q().QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'vector')").Scan(&hasVector)
	if err != nil || !hasVector {
		return []VectorColumnInfo{}, nil
	}

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

	rows, err := si.q().Query(ctx, query, args...)
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

func (si *SchemaInspector) IsPgVectorInstalled(ctx context.Context) (bool, string, error) {
	var version *string
	err := si.q().QueryRow(ctx, `
		SELECT installed_version
		FROM pg_available_extensions
		WHERE name = 'vector'
	`).Scan(&version)
	if err != nil {
		return false, "", nil
	}

	if version == nil {
		return false, "", nil
	}

	return true, *version, nil
}
