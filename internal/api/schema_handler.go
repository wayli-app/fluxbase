package api

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// SchemaRelationship represents a foreign key relationship for ERD visualization
type SchemaRelationship struct {
	ID               string `json:"id"`
	SourceSchema     string `json:"source_schema"`
	SourceTable      string `json:"source_table"`
	SourceColumn     string `json:"source_column"`
	TargetSchema     string `json:"target_schema"`
	TargetTable      string `json:"target_table"`
	TargetColumn     string `json:"target_column"`
	ConstraintName   string `json:"constraint_name"`
	OnDelete         string `json:"on_delete"`
	OnUpdate         string `json:"on_update"`
}

// SchemaNode represents a table for ERD visualization
type SchemaNode struct {
	Schema      string             `json:"schema"`
	Name        string             `json:"name"`
	Columns     []SchemaNodeColumn `json:"columns"`
	PrimaryKey  []string           `json:"primary_key"`
	RLSEnabled  bool               `json:"rls_enabled"`
	RowEstimate *int64             `json:"row_estimate,omitempty"`
}

// SchemaNodeColumn represents a column in a schema node
type SchemaNodeColumn struct {
	Name         string  `json:"name"`
	DataType     string  `json:"data_type"`
	Nullable     bool    `json:"nullable"`
	IsPrimaryKey bool    `json:"is_primary_key"`
	IsForeignKey bool    `json:"is_foreign_key"`
	FKTarget     *string `json:"fk_target,omitempty"` // "schema.table.column"
	DefaultValue *string `json:"default_value,omitempty"`
}

// SchemaGraphResponse is the response for the schema graph endpoint
type SchemaGraphResponse struct {
	Nodes   []SchemaNode         `json:"nodes"`
	Edges   []SchemaRelationship `json:"edges"`
	Schemas []string             `json:"schemas"`
}

// GetSchemaGraph returns all tables and relationships for ERD visualization
// GET /api/v1/admin/schema/graph
func (s *Server) GetSchemaGraph(c *fiber.Ctx) error {
	ctx := c.Context()
	schemasParam := c.Query("schemas", "public")
	schemaList := strings.Split(schemasParam, ",")

	// Trim whitespace from schema names
	for i, schema := range schemaList {
		schemaList[i] = strings.TrimSpace(schema)
	}

	// Query all tables with their columns, primary keys, and RLS status
	tablesQuery := `
		WITH table_info AS (
			SELECT
				t.table_schema,
				t.table_name,
				c.relrowsecurity as rls_enabled,
				c.reltuples::bigint as row_estimate
			FROM information_schema.tables t
			JOIN pg_class c ON c.relname = t.table_name
			JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
			WHERE t.table_schema = ANY($1)
			AND t.table_type = 'BASE TABLE'
		),
		columns_info AS (
			SELECT
				c.table_schema,
				c.table_name,
				c.column_name,
				c.data_type,
				c.is_nullable = 'YES' as is_nullable,
				c.column_default,
				c.ordinal_position
			FROM information_schema.columns c
			WHERE c.table_schema = ANY($1)
		),
		pk_info AS (
			SELECT
				tc.table_schema,
				tc.table_name,
				kcu.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			WHERE tc.constraint_type = 'PRIMARY KEY'
			AND tc.table_schema = ANY($1)
		),
		fk_columns AS (
			SELECT DISTINCT
				tc.table_schema,
				tc.table_name,
				kcu.column_name,
				ccu.table_schema as ref_schema,
				ccu.table_name as ref_table,
				ccu.column_name as ref_column
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage ccu
				ON ccu.constraint_name = tc.constraint_name
			WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = ANY($1)
		)
		SELECT
			ti.table_schema,
			ti.table_name,
			ti.rls_enabled,
			ti.row_estimate,
			ci.column_name,
			ci.data_type,
			ci.is_nullable,
			ci.column_default,
			ci.ordinal_position,
			pk.column_name IS NOT NULL as is_primary_key,
			fk.column_name IS NOT NULL as is_foreign_key,
			CASE WHEN fk.column_name IS NOT NULL
				THEN fk.ref_schema || '.' || fk.ref_table || '.' || fk.ref_column
				ELSE NULL
			END as fk_target
		FROM table_info ti
		JOIN columns_info ci ON ti.table_schema = ci.table_schema AND ti.table_name = ci.table_name
		LEFT JOIN pk_info pk ON ci.table_schema = pk.table_schema
			AND ci.table_name = pk.table_name
			AND ci.column_name = pk.column_name
		LEFT JOIN fk_columns fk ON ci.table_schema = fk.table_schema
			AND ci.table_name = fk.table_name
			AND ci.column_name = fk.column_name
		ORDER BY ti.table_schema, ti.table_name, ci.ordinal_position
	`

	rows, err := s.db.Query(ctx, tablesQuery, schemaList)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "SCHEMA_QUERY_FAILED", err.Error())
	}
	defer rows.Close()

	// Build nodes map (keyed by schema.table)
	nodesMap := make(map[string]*SchemaNode)
	pkMap := make(map[string][]string) // schema.table -> primary key columns

	for rows.Next() {
		var (
			tableSchema  string
			tableName    string
			rlsEnabled   bool
			rowEstimate  *int64
			columnName   string
			dataType     string
			isNullable   bool
			defaultValue *string
			ordinalPos   int
			isPrimaryKey bool
			isForeignKey bool
			fkTarget     *string
		)

		err := rows.Scan(
			&tableSchema, &tableName, &rlsEnabled, &rowEstimate,
			&columnName, &dataType, &isNullable, &defaultValue, &ordinalPos,
			&isPrimaryKey, &isForeignKey, &fkTarget,
		)
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "SCAN_FAILED", err.Error())
		}

		key := tableSchema + "." + tableName

		// Create or update node
		if _, exists := nodesMap[key]; !exists {
			nodesMap[key] = &SchemaNode{
				Schema:      tableSchema,
				Name:        tableName,
				RLSEnabled:  rlsEnabled,
				RowEstimate: rowEstimate,
				Columns:     []SchemaNodeColumn{},
				PrimaryKey:  []string{},
			}
		}

		// Add column
		nodesMap[key].Columns = append(nodesMap[key].Columns, SchemaNodeColumn{
			Name:         columnName,
			DataType:     dataType,
			Nullable:     isNullable,
			IsPrimaryKey: isPrimaryKey,
			IsForeignKey: isForeignKey,
			FKTarget:     fkTarget,
			DefaultValue: defaultValue,
		})

		// Track primary keys
		if isPrimaryKey {
			pkMap[key] = append(pkMap[key], columnName)
		}
	}

	if err := rows.Err(); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "ROWS_ERROR", err.Error())
	}

	// Set primary keys on nodes
	for key, pks := range pkMap {
		if node, exists := nodesMap[key]; exists {
			node.PrimaryKey = pks
		}
	}

	// Convert map to slice
	nodes := make([]SchemaNode, 0, len(nodesMap))
	for _, node := range nodesMap {
		nodes = append(nodes, *node)
	}

	// Query all foreign key relationships
	relationsQuery := `
		SELECT
			tc.constraint_name || '_' || kcu.column_name as id,
			tc.table_schema as source_schema,
			tc.table_name as source_table,
			kcu.column_name as source_column,
			ccu.table_schema as target_schema,
			ccu.table_name as target_table,
			ccu.column_name as target_column,
			tc.constraint_name,
			COALESCE(rc.delete_rule, 'NO ACTION') as on_delete,
			COALESCE(rc.update_rule, 'NO ACTION') as on_update
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
		LEFT JOIN information_schema.referential_constraints rc
			ON rc.constraint_name = tc.constraint_name
			AND rc.constraint_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_schema = ANY($1)
		ORDER BY tc.table_schema, tc.table_name, tc.constraint_name
	`

	relRows, err := s.db.Query(ctx, relationsQuery, schemaList)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "RELATIONS_QUERY_FAILED", err.Error())
	}
	defer relRows.Close()

	edges := []SchemaRelationship{}
	for relRows.Next() {
		var rel SchemaRelationship
		err := relRows.Scan(
			&rel.ID, &rel.SourceSchema, &rel.SourceTable, &rel.SourceColumn,
			&rel.TargetSchema, &rel.TargetTable, &rel.TargetColumn,
			&rel.ConstraintName, &rel.OnDelete, &rel.OnUpdate,
		)
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "REL_SCAN_FAILED", err.Error())
		}
		edges = append(edges, rel)
	}

	if err := relRows.Err(); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "REL_ROWS_ERROR", err.Error())
	}

	return c.JSON(SchemaGraphResponse{
		Nodes:   nodes,
		Edges:   edges,
		Schemas: schemaList,
	})
}

// GetTableRelationships returns relationships for a specific table
// GET /api/v1/admin/tables/:schema/:table/relationships
func (s *Server) GetTableRelationships(c *fiber.Ctx) error {
	ctx := c.Context()
	schema := c.Params("schema")
	table := c.Params("table")

	if schema == "" || table == "" {
		return SendBadRequest(c, "schema and table are required")
	}

	query := `
		WITH outgoing AS (
			SELECT
				'outgoing' as direction,
				tc.constraint_name,
				kcu.column_name as local_column,
				ccu.table_schema as foreign_schema,
				ccu.table_name as foreign_table,
				ccu.column_name as foreign_column,
				COALESCE(rc.delete_rule, 'NO ACTION') as delete_rule,
				COALESCE(rc.update_rule, 'NO ACTION') as update_rule
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage ccu
				ON ccu.constraint_name = tc.constraint_name
			LEFT JOIN information_schema.referential_constraints rc
				ON rc.constraint_name = tc.constraint_name
			WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = $1 AND tc.table_name = $2
		),
		incoming AS (
			SELECT
				'incoming' as direction,
				tc.constraint_name,
				ccu.column_name as local_column,
				tc.table_schema as foreign_schema,
				tc.table_name as foreign_table,
				kcu.column_name as foreign_column,
				COALESCE(rc.delete_rule, 'NO ACTION') as delete_rule,
				COALESCE(rc.update_rule, 'NO ACTION') as update_rule
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage ccu
				ON ccu.constraint_name = tc.constraint_name
			LEFT JOIN information_schema.referential_constraints rc
				ON rc.constraint_name = tc.constraint_name
			WHERE tc.constraint_type = 'FOREIGN KEY'
			AND ccu.table_schema = $1 AND ccu.table_name = $2
		)
		SELECT * FROM outgoing
		UNION ALL
		SELECT * FROM incoming
		ORDER BY direction, constraint_name
	`

	rows, err := s.db.Query(ctx, query, schema, table)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "QUERY_FAILED", err.Error())
	}
	defer rows.Close()

	type RelationshipDetail struct {
		Direction      string `json:"direction"`
		ConstraintName string `json:"constraint_name"`
		LocalColumn    string `json:"local_column"`
		ForeignSchema  string `json:"foreign_schema"`
		ForeignTable   string `json:"foreign_table"`
		ForeignColumn  string `json:"foreign_column"`
		DeleteRule     string `json:"delete_rule"`
		UpdateRule     string `json:"update_rule"`
	}

	outgoing := []RelationshipDetail{}
	incoming := []RelationshipDetail{}

	for rows.Next() {
		var rel RelationshipDetail
		err := rows.Scan(
			&rel.Direction, &rel.ConstraintName, &rel.LocalColumn,
			&rel.ForeignSchema, &rel.ForeignTable, &rel.ForeignColumn,
			&rel.DeleteRule, &rel.UpdateRule,
		)
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "SCAN_FAILED", err.Error())
		}

		if rel.Direction == "outgoing" {
			outgoing = append(outgoing, rel)
		} else {
			incoming = append(incoming, rel)
		}
	}

	return c.JSON(fiber.Map{
		"schema":   schema,
		"table":    table,
		"outgoing": outgoing,
		"incoming": incoming,
	})
}
