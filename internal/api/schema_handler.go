package api

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// SchemaRelationship represents a foreign key relationship for ERD visualization
type SchemaRelationship struct {
	ID             string `json:"id"`
	SourceSchema   string `json:"source_schema"`
	SourceTable    string `json:"source_table"`
	SourceColumn   string `json:"source_column"`
	TargetSchema   string `json:"target_schema"`
	TargetTable    string `json:"target_table"`
	TargetColumn   string `json:"target_column"`
	ConstraintName string `json:"constraint_name"`
	OnDelete       string `json:"on_delete"`
	OnUpdate       string `json:"on_update"`
	Cardinality    string `json:"cardinality"`
}

// SchemaNode represents a table for ERD visualization
type SchemaNode struct {
	Schema           string             `json:"schema"`
	Name             string             `json:"name"`
	Columns          []SchemaNodeColumn `json:"columns"`
	PrimaryKey       []string           `json:"primary_key"`
	RLSEnabled       bool               `json:"rls_enabled"`
	ForceRLS         bool               `json:"force_rls"`
	RowEstimate      *int64             `json:"row_estimate,omitempty"`
	Comment          *string            `json:"comment,omitempty"`
	IncomingRelCount int                `json:"incoming_rel_count"`
	OutgoingRelCount int                `json:"outgoing_rel_count"`
}

// SchemaNodeColumn represents a column in a schema node
type SchemaNodeColumn struct {
	Name         string  `json:"name"`
	DataType     string  `json:"data_type"`
	Nullable     bool    `json:"nullable"`
	IsPrimaryKey bool    `json:"is_primary_key"`
	IsForeignKey bool    `json:"is_foreign_key"`
	FKTarget     *string `json:"fk_target,omitempty"`
	DefaultValue *string `json:"default_value,omitempty"`
	IsUnique     bool    `json:"is_unique"`
	IsIndexed    bool    `json:"is_indexed"`
	Comment      *string `json:"comment,omitempty"`
}

// SchemaGraphResponse is the response for the schema graph endpoint
type SchemaGraphResponse struct {
	Nodes   []SchemaNode         `json:"nodes"`
	Edges   []SchemaRelationship `json:"edges"`
	Schemas []string             `json:"schemas"`
}

type schemaGraphCacheEntry struct {
	response SchemaGraphResponse
	expiry   time.Time
}

type schemaGraphCache struct {
	mu      sync.RWMutex
	entries map[string]*schemaGraphCacheEntry
	ttl     time.Duration
}

func newSchemaGraphCache(ttl time.Duration) *schemaGraphCache {
	return &schemaGraphCache{
		entries: make(map[string]*schemaGraphCacheEntry),
		ttl:     ttl,
	}
}

func (c *schemaGraphCache) Get(key string) (*SchemaGraphResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiry) {
		return nil, false
	}
	return &entry.response, true
}

func (c *schemaGraphCache) Set(key string, resp SchemaGraphResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = &schemaGraphCacheEntry{
		response: resp,
		expiry:   time.Now().Add(c.ttl),
	}
}

func (c *schemaGraphCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*schemaGraphCacheEntry)
}

func schemaGraphCacheKey(tenantID string, schemas []string) string {
	sorted := make([]string, len(schemas))
	copy(sorted, schemas)
	sort.Strings(sorted)
	return tenantID + ":" + strings.Join(sorted, ",")
}

type SchemaGraphHandlers struct {
	db         *database.Connection
	graphCache *schemaGraphCache
}

func NewSchemaGraphHandlers(db *database.Connection, graphCache *schemaGraphCache) *SchemaGraphHandlers {
	return &SchemaGraphHandlers{db: db, graphCache: graphCache}
}

// GetSchemaGraph returns all tables and relationships for ERD visualization.
// Results are cached for 2 minutes per (tenant, schema list) combination.
// GET /api/v1/admin/schema/graph
func (h *SchemaGraphHandlers) GetSchemaGraph(c fiber.Ctx) error {
	if h.db == nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	ctx := middleware.CtxWithTenant(c)
	schemasParam := c.Query("schemas", "public")
	schemaList := strings.Split(schemasParam, ",")
	for i, schema := range schemaList {
		schemaList[i] = strings.TrimSpace(schema)
	}

	tenantID := middleware.GetTenantID(c)
	cacheKey := schemaGraphCacheKey(tenantID, schemaList)

	if cached, ok := h.graphCache.Get(cacheKey); ok {
		return c.JSON(cached)
	}

	pool := tenantPool(c, h.db)

	nodes, err := querySchemaNodes(ctx, pool, schemaList)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	edges, err := querySchemaEdges(ctx, pool, schemaList)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	for i := range edges {
		srcKey := edges[i].SourceSchema + "." + edges[i].SourceTable
		tgtKey := edges[i].TargetSchema + "." + edges[i].TargetTable
		if node, ok := nodes[srcKey]; ok {
			node.OutgoingRelCount++
		}
		if node, ok := nodes[tgtKey]; ok {
			node.IncomingRelCount++
		}
	}

	nodesSlice := make([]SchemaNode, 0, len(nodes))
	for _, node := range nodes {
		nodesSlice = append(nodesSlice, *node)
	}
	sort.Slice(nodesSlice, func(i, j int) bool {
		if nodesSlice[i].Schema != nodesSlice[j].Schema {
			return nodesSlice[i].Schema < nodesSlice[j].Schema
		}
		return nodesSlice[i].Name < nodesSlice[j].Name
	})

	resp := SchemaGraphResponse{
		Nodes:   nodesSlice,
		Edges:   edges,
		Schemas: schemaList,
	}

	h.graphCache.Set(cacheKey, resp)

	return c.JSON(resp)
}

// querySchemaNodes fetches all tables with columns, PKs, indexes, FKs, RLS, and comments
// using direct pg_catalog queries for optimal performance.
func querySchemaNodes(ctx context.Context, pool *pgxpool.Pool, schemaList []string) (map[string]*SchemaNode, error) {
	query := `
		WITH tables AS (
			SELECT
				n.nspname AS table_schema,
				c.relname AS table_name,
				c.relrowsecurity AS rls_enabled,
				c.relforcerowsecurity AS force_rls,
				c.reltuples::bigint AS row_estimate,
				obj_description(c.oid, 'pg_class') AS table_comment,
				c.oid
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			LEFT JOIN pg_depend d ON d.objid = c.oid AND d.deptype = 'e'
			WHERE n.nspname = ANY($1)
			  AND c.relkind IN ('r', 'f')
			  AND d.objid IS NULL
		),
		cols AS (
			SELECT
				t.table_schema,
				t.table_name,
				a.attname AS column_name,
				format_type(a.atttypid, a.atttypmod) AS data_type,
				NOT a.attnotnull AS is_nullable,
				pg_get_expr(d.adbin, d.adrelid) AS column_default,
				a.attnum AS ordinal_position,
				col_description(t.oid, a.attnum) AS column_comment
			FROM tables t
			JOIN pg_attribute a ON a.attrelid = t.oid
				AND a.attnum > 0
				AND NOT a.attisdropped
			LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid
				AND d.adnum = a.attnum
		),
		pk_cols AS (
			SELECT
				n.nspname AS table_schema,
				cl.relname AS table_name,
				a.attname AS column_name
			FROM pg_constraint con
			JOIN pg_class cl ON con.conrelid = cl.oid
			JOIN pg_namespace n ON cl.relnamespace = n.oid
			CROSS JOIN LATERAL unnest(con.conkey) WITH ORDINALITY AS u(attnum, ord)
			JOIN pg_attribute a ON a.attrelid = cl.oid AND a.attnum = u.attnum
			WHERE con.contype = 'p'
			  AND n.nspname = ANY($1)
		),
		uq_cols AS (
			SELECT
				n.nspname AS table_schema,
				cl.relname AS table_name,
				a.attname AS column_name
			FROM pg_constraint con
			JOIN pg_class cl ON con.conrelid = cl.oid
			JOIN pg_namespace n ON cl.relnamespace = n.oid
			CROSS JOIN LATERAL unnest(con.conkey) WITH ORDINALITY AS u(attnum, ord)
			JOIN pg_attribute a ON a.attrelid = cl.oid AND a.attnum = u.attnum
			WHERE con.contype = 'u'
			  AND n.nspname = ANY($1)
		),
		idx_cols AS (
			SELECT DISTINCT
				n.nspname AS table_schema,
				cl.relname AS table_name,
				a.attname AS column_name
			FROM pg_index i
			JOIN pg_class cl ON i.indrelid = cl.oid
			JOIN pg_namespace n ON cl.relnamespace = n.oid
			CROSS JOIN LATERAL unnest(i.indkey) WITH ORDINALITY AS u(attnum, ord)
			JOIN pg_attribute a ON a.attrelid = cl.oid AND a.attnum = u.attnum
			WHERE n.nspname = ANY($1)
			  AND NOT i.indisprimary
		),
		fk_cols AS (
			SELECT DISTINCT
				ns_src.nspname AS table_schema,
				cl_src.relname AS table_name,
				a_src.attname AS column_name,
				ns_tgt.nspname AS ref_schema,
				cl_tgt.relname AS ref_table,
				a_tgt.attname AS ref_column
			FROM pg_constraint con
			JOIN pg_class cl_src ON con.conrelid = cl_src.oid
			JOIN pg_namespace ns_src ON cl_src.relnamespace = ns_src.oid
			JOIN pg_class cl_tgt ON con.confrelid = cl_tgt.oid
			JOIN pg_namespace ns_tgt ON cl_tgt.relnamespace = ns_tgt.oid
			CROSS JOIN LATERAL unnest(con.conkey, con.confkey) WITH ORDINALITY AS u(src_attnum, tgt_attnum, ord)
			JOIN pg_attribute a_src ON a_src.attrelid = cl_src.oid AND a_src.attnum = u.src_attnum
			JOIN pg_attribute a_tgt ON a_tgt.attrelid = cl_tgt.oid AND a_tgt.attnum = u.tgt_attnum
			WHERE con.contype = 'f'
			  AND ns_src.nspname = ANY($1)
		)
		SELECT
			t.table_schema,
			t.table_name,
			t.rls_enabled,
			t.force_rls,
			t.row_estimate,
			t.table_comment,
			co.column_name,
			co.data_type,
			co.is_nullable,
			co.column_default,
			co.ordinal_position,
			co.column_comment,
			pk.column_name IS NOT NULL AS is_primary_key,
			fk.column_name IS NOT NULL AS is_foreign_key,
			uq.column_name IS NOT NULL AS is_unique,
			ix.column_name IS NOT NULL AS is_indexed,
			CASE WHEN fk.column_name IS NOT NULL
				THEN fk.ref_schema || '.' || fk.ref_table || '.' || fk.ref_column
				ELSE NULL
			END AS fk_target
		FROM tables t
		JOIN cols co
			ON t.table_schema = co.table_schema
			AND t.table_name = co.table_name
		LEFT JOIN pk_cols pk
			ON co.table_schema = pk.table_schema
			AND co.table_name = pk.table_name
			AND co.column_name = pk.column_name
		LEFT JOIN uq_cols uq
			ON co.table_schema = uq.table_schema
			AND co.table_name = uq.table_name
			AND co.column_name = uq.column_name
		LEFT JOIN idx_cols ix
			ON co.table_schema = ix.table_schema
			AND co.table_name = ix.table_name
			AND co.column_name = ix.column_name
		LEFT JOIN fk_cols fk
			ON co.table_schema = fk.table_schema
			AND co.table_name = fk.table_name
			AND co.column_name = fk.column_name
		ORDER BY t.table_schema, t.table_name, co.ordinal_position
	`

	rows, err := pool.Query(ctx, query, schemaList)
	if err != nil {
		return nil, fmt.Errorf("failed to query schema nodes: %w", err)
	}
	defer rows.Close()

	nodesMap := make(map[string]*SchemaNode)
	pkMap := make(map[string][]string)
	seenCols := make(map[string]struct{})

	for rows.Next() {
		var (
			tableSchema   string
			tableName     string
			rlsEnabled    bool
			forceRLS      bool
			rowEstimate   *int64
			tableComment  *string
			columnName    string
			dataType      string
			isNullable    bool
			defaultValue  *string
			ordinalPos    int
			columnComment *string
			isPrimaryKey  bool
			isForeignKey  bool
			isUnique      bool
			isIndexed     bool
			fkTarget      *string
		)

		if err := rows.Scan(
			&tableSchema, &tableName, &rlsEnabled, &forceRLS, &rowEstimate, &tableComment,
			&columnName, &dataType, &isNullable, &defaultValue, &ordinalPos, &columnComment,
			&isPrimaryKey, &isForeignKey, &isUnique, &isIndexed, &fkTarget,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schema node row: %w", err)
		}

		key := tableSchema + "." + tableName

		if _, exists := nodesMap[key]; !exists {
			nodesMap[key] = &SchemaNode{
				Schema:      tableSchema,
				Name:        tableName,
				RLSEnabled:  rlsEnabled,
				ForceRLS:    forceRLS,
				RowEstimate: rowEstimate,
				Comment:     tableComment,
				Columns:     []SchemaNodeColumn{},
				PrimaryKey:  []string{},
			}
		}

		colKey := key + "." + columnName
		if _, seen := seenCols[colKey]; !seen {
			seenCols[colKey] = struct{}{}
			nodesMap[key].Columns = append(nodesMap[key].Columns, SchemaNodeColumn{
				Name:         columnName,
				DataType:     dataType,
				Nullable:     isNullable,
				IsPrimaryKey: isPrimaryKey,
				IsForeignKey: isForeignKey,
				FKTarget:     fkTarget,
				DefaultValue: defaultValue,
				IsUnique:     isUnique,
				IsIndexed:    isIndexed,
				Comment:      columnComment,
			})
		}

		if isPrimaryKey {
			pkMap[key] = append(pkMap[key], columnName)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schema node rows: %w", err)
	}

	for key, pks := range pkMap {
		if node, ok := nodesMap[key]; ok {
			node.PrimaryKey = pks
		}
	}

	return nodesMap, nil
}

// querySchemaEdges fetches all foreign key relationships with cardinality.
func querySchemaEdges(ctx context.Context, pool *pgxpool.Pool, schemaList []string) ([]SchemaRelationship, error) {
	query := `
		WITH fk_info AS (
			SELECT
				c.conname || '_' || a_src.attname AS id,
				ns_src.nspname AS source_schema,
				cl_src.relname AS source_table,
				a_src.attname AS source_column,
				ns_tgt.nspname AS target_schema,
				cl_tgt.relname AS target_table,
				a_tgt.attname AS target_column,
				c.conname AS constraint_name,
				CASE c.confdeltype
					WHEN 'a' THEN 'NO ACTION'
					WHEN 'r' THEN 'RESTRICT'
					WHEN 'c' THEN 'CASCADE'
					WHEN 'n' THEN 'SET NULL'
					WHEN 'd' THEN 'SET DEFAULT'
					ELSE 'NO ACTION'
				END AS on_delete,
				CASE c.confupdtype
					WHEN 'a' THEN 'NO ACTION'
					WHEN 'r' THEN 'RESTRICT'
					WHEN 'c' THEN 'CASCADE'
					WHEN 'n' THEN 'SET NULL'
					WHEN 'd' THEN 'SET DEFAULT'
					ELSE 'NO ACTION'
				END AS on_update
			FROM pg_constraint c
			JOIN pg_class cl_src ON c.conrelid = cl_src.oid
			JOIN pg_namespace ns_src ON cl_src.relnamespace = ns_src.oid
			JOIN pg_class cl_tgt ON c.confrelid = cl_tgt.oid
			JOIN pg_namespace ns_tgt ON cl_tgt.relnamespace = ns_tgt.oid
			CROSS JOIN LATERAL unnest(c.conkey, c.confkey) WITH ORDINALITY AS cols(src_attnum, tgt_attnum, ord)
			JOIN pg_attribute a_src ON a_src.attrelid = cl_src.oid AND a_src.attnum = cols.src_attnum
			JOIN pg_attribute a_tgt ON a_tgt.attrelid = cl_tgt.oid AND a_tgt.attnum = cols.tgt_attnum
			WHERE c.contype = 'f'
			  AND ns_src.nspname = ANY($1)
		),
		source_unique AS (
			SELECT DISTINCT
				ns.nspname AS table_schema,
				cl.relname AS table_name,
				a.attname AS column_name
			FROM pg_constraint c
			JOIN pg_class cl ON c.conrelid = cl.oid
			JOIN pg_namespace ns ON cl.relnamespace = ns.oid
			CROSS JOIN LATERAL unnest(c.conkey) WITH ORDINALITY AS cols(attnum, ord)
			JOIN pg_attribute a ON a.attrelid = cl.oid AND a.attnum = cols.attnum
			WHERE c.contype IN ('u', 'p')
			  AND ns.nspname = ANY($1)
		)
		SELECT
			fk.id,
			fk.source_schema,
			fk.source_table,
			fk.source_column,
			fk.target_schema,
			fk.target_table,
			fk.target_column,
			fk.constraint_name,
			fk.on_delete,
			fk.on_update,
			CASE
				WHEN su.column_name IS NOT NULL THEN 'one-to-one'
				ELSE 'many-to-one'
			END AS cardinality
		FROM fk_info fk
		LEFT JOIN source_unique su
			ON fk.source_schema = su.table_schema
			AND fk.source_table = su.table_name
			AND fk.source_column = su.column_name
		ORDER BY fk.source_schema, fk.source_table, fk.constraint_name
	`

	rows, err := pool.Query(ctx, query, schemaList)
	if err != nil {
		return nil, fmt.Errorf("failed to query schema edges: %w", err)
	}
	defer rows.Close()

	var edges []SchemaRelationship
	for rows.Next() {
		var rel SchemaRelationship
		if err := rows.Scan(
			&rel.ID, &rel.SourceSchema, &rel.SourceTable, &rel.SourceColumn,
			&rel.TargetSchema, &rel.TargetTable, &rel.TargetColumn,
			&rel.ConstraintName, &rel.OnDelete, &rel.OnUpdate, &rel.Cardinality,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schema edge row: %w", err)
		}
		edges = append(edges, rel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schema edge rows: %w", err)
	}

	return edges, nil
}

// GetTableRelationships returns relationships for a specific table
// GET /api/v1/admin/tables/:schema/:table/relationships
func (h *SchemaGraphHandlers) GetTableRelationships(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	schema := c.Params("schema")
	table := c.Params("table")

	if schema == "" || table == "" {
		return SendBadRequest(c, "schema and table are required", "MISSING_PARAMS")
	}

	if h.db == nil {
		return SendInternalError(c, "Database connection not initialized")
	}

	pool := tenantPool(c, h.db)

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

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
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
		if err := rows.Scan(
			&rel.Direction, &rel.ConstraintName, &rel.LocalColumn,
			&rel.ForeignSchema, &rel.ForeignTable, &rel.ForeignColumn,
			&rel.DeleteRule, &rel.UpdateRule,
		); err != nil {
			return SendError(c, fiber.StatusInternalServerError, err.Error())
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

func tenantPool(c fiber.Ctx, db *database.Connection) *pgxpool.Pool {
	if pool := middleware.GetTenantPool(c); pool != nil {
		return pool
	}
	return db.Pool()
}
