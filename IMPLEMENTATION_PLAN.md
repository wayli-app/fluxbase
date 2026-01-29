# Fluxbase Feature Implementation Plan

**Date:** January 2026
**Scope:** Complete implementation roadmap for all identified improvements

---

## Table of Contents

1. [Overview](#1-overview)
2. [Phase 1: Security Fixes (Week 1)](#2-phase-1-security-fixes-week-1)
3. [Phase 2: Visual Schema Viewer with Relations (Weeks 2-4)](#3-phase-2-visual-schema-viewer-with-relations-weeks-2-4)
4. [Phase 3: RLS Policy Editor & Security Warnings (Weeks 5-8)](#4-phase-3-rls-policy-editor--security-warnings-weeks-5-8)
5. [Phase 4: Enhanced DDL Operations (Weeks 9-10)](#5-phase-4-enhanced-ddl-operations-weeks-9-10)
6. [Phase 5: Developer Experience (Weeks 11-14)](#6-phase-5-developer-experience-weeks-11-14)
7. [Phase 6: Advanced Features (Weeks 15-20)](#7-phase-6-advanced-features-weeks-15-20)
8. [File-by-File Implementation Details](#8-file-by-file-implementation-details)
9. [Database Migrations](#9-database-migrations)
10. [Testing Strategy](#10-testing-strategy)

---

## 1. Overview

### Priority Order

| Priority | Feature | Impact | Effort |
|----------|---------|--------|--------|
| P0 | SQL Injection Fix | Critical | ✅ Done |
| P0 | Startup Security Validation | High | Small |
| P1 | Visual Schema Viewer | High | Large |
| P1 | RLS Policy Viewer | High | Medium |
| P1 | RLS Policy Editor | High | Large |
| P1 | Security Warning System | High | Medium |
| P2 | Enhanced DDL (FK, indexes) | Medium | Medium |
| P2 | Error Handling Standardization | Medium | Medium |
| P3 | Go SDK | Medium | Large |
| P3 | Migration UI | Medium | Medium |
| P4 | Performance Advisor | Low | Medium |
| P4 | Message Queue | Low | Large |

### Technology Stack Reference

**Backend:**
- Go 1.25+, Fiber v2, pgx/v5
- Handler pattern with dependency injection
- PostgreSQL system catalogs for introspection

**Frontend:**
- React 19, TanStack Router/Query, Tailwind v4
- shadcn/ui components, Zustand state management
- ReactFlow for diagram visualization

---

## 2. Phase 1: Security Fixes (Week 1)

### 2.1 SQL Injection Fix ✅ COMPLETED

Fixed in `internal/pubsub/postgres.go`:
- LISTEN now uses `quoteIdentifier()` for proper identifier escaping
- pg_notify now uses parameterized queries `$1, $2`

### 2.2 Startup Security Configuration Validation

**File:** `internal/config/validation.go` (new file)

```go
package config

import (
    "errors"
    "fmt"
    "os"
    "strings"
)

// SecurityValidationError contains all security configuration issues
type SecurityValidationError struct {
    Errors   []string
    Warnings []string
}

func (e *SecurityValidationError) Error() string {
    return fmt.Sprintf("security validation failed: %d errors, %d warnings",
        len(e.Errors), len(e.Warnings))
}

// ValidateSecurityConfig checks for insecure configurations at startup
func ValidateSecurityConfig(cfg *Config) *SecurityValidationError {
    result := &SecurityValidationError{}
    isProduction := os.Getenv("FLUXBASE_ENV") == "production"

    // Critical: Encryption key validation
    if cfg.Auth.EncryptionKey != "" {
        keyLen := len(cfg.Auth.EncryptionKey)
        if keyLen != 16 && keyLen != 24 && keyLen != 32 {
            result.Errors = append(result.Errors,
                fmt.Sprintf("encryption_key must be 16, 24, or 32 bytes (got %d)", keyLen))
        }
    } else if isProduction {
        result.Errors = append(result.Errors,
            "encryption_key is required in production for OAuth secret encryption")
    }

    // Critical: Setup token in production
    if cfg.Auth.SetupToken == "" && isProduction {
        result.Errors = append(result.Errors,
            "setup_token is required in production to protect dashboard setup")
    }

    // Warning: CORS wildcard
    for _, origin := range cfg.Server.CORS.AllowedOrigins {
        if origin == "*" {
            result.Warnings = append(result.Warnings,
                "CORS allows all origins (*) - this may expose the API to cross-origin attacks")
            break
        }
    }

    // Warning: Rate limiting in memory mode
    if cfg.RateLimit.Store == "memory" && isProduction {
        result.Warnings = append(result.Warnings,
            "rate limiting uses in-memory store - bypassed in multi-instance deployments")
    }

    // Warning: Debug mode in production
    if cfg.Server.Debug && isProduction {
        result.Warnings = append(result.Warnings,
            "debug mode enabled in production - may expose sensitive information")
    }

    if len(result.Errors) > 0 || len(result.Warnings) > 0 {
        return result
    }
    return nil
}
```

**Integration in `cmd/fluxbase/main.go`:**

```go
// After config loading, before server start
if err := config.ValidateSecurityConfig(cfg); err != nil {
    secErr := err.(*config.SecurityValidationError)

    // Log warnings
    for _, warn := range secErr.Warnings {
        log.Warn().Msg("Security warning: " + warn)
    }

    // Fail on errors
    if len(secErr.Errors) > 0 {
        for _, e := range secErr.Errors {
            log.Error().Msg("Security error: " + e)
        }
        log.Fatal().Msg("Server startup blocked due to security configuration errors")
    }
}
```

### 2.3 Rate Limiting Multi-Instance Warning

**File:** `internal/middleware/rate_limiter.go`

Add detection for multi-instance mode:

```go
func (r *RateLimiter) validateConfiguration() error {
    // Detect Kubernetes/Docker environment
    isMultiInstance := os.Getenv("KUBERNETES_SERVICE_HOST") != "" ||
                       os.Getenv("DOCKER_HOST") != "" ||
                       os.Getenv("FLUXBASE_MULTI_INSTANCE") == "true"

    if isMultiInstance && r.store.Type() == "memory" {
        log.Warn().Msg("Rate limiting uses in-memory store in multi-instance environment. " +
            "Configure Redis/Dragonfly for distributed rate limiting.")
    }
    return nil
}
```

---

## 3. Phase 2: Visual Schema Viewer with Relations (Weeks 2-4)

### 3.1 Backend: Schema Relationships API

#### New File: `internal/api/schema_relationships_handler.go`

```go
package api

import (
    "github.com/gofiber/fiber/v2"
)

// SchemaRelationship represents a foreign key relationship for visualization
type SchemaRelationship struct {
    ID                string `json:"id"`
    SourceSchema      string `json:"source_schema"`
    SourceTable       string `json:"source_table"`
    SourceColumn      string `json:"source_column"`
    TargetSchema      string `json:"target_schema"`
    TargetTable       string `json:"target_table"`
    TargetColumn      string `json:"target_column"`
    ConstraintName    string `json:"constraint_name"`
    OnDelete          string `json:"on_delete"`
    OnUpdate          string `json:"on_update"`
}

// SchemaNode represents a table for the ERD visualization
type SchemaNode struct {
    Schema       string                   `json:"schema"`
    Name         string                   `json:"name"`
    Columns      []SchemaNodeColumn       `json:"columns"`
    PrimaryKey   []string                 `json:"primary_key"`
    RLSEnabled   bool                     `json:"rls_enabled"`
    RowCount     *int64                   `json:"row_count,omitempty"`
    Indexes      []IndexInfo              `json:"indexes"`
}

type SchemaNodeColumn struct {
    Name         string  `json:"name"`
    DataType     string  `json:"data_type"`
    Nullable     bool    `json:"nullable"`
    IsPrimaryKey bool    `json:"is_primary_key"`
    IsForeignKey bool    `json:"is_foreign_key"`
    FKTarget     *string `json:"fk_target,omitempty"` // "schema.table.column"
    DefaultValue *string `json:"default_value,omitempty"`
}

// GetSchemaGraph returns all tables and relationships for ERD visualization
// GET /api/v1/admin/schema/graph
func (s *Server) GetSchemaGraph(c *fiber.Ctx) error {
    ctx := c.Context()
    schemas := c.Query("schemas", "public,auth,storage")

    // Query all tables with columns
    nodesQuery := `
        SELECT
            t.table_schema,
            t.table_name,
            c.relrowsecurity as rls_enabled,
            (SELECT reltuples::bigint FROM pg_class WHERE oid = c.oid) as row_estimate
        FROM information_schema.tables t
        JOIN pg_class c ON c.relname = t.table_name
        JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
        WHERE t.table_schema = ANY($1)
        AND t.table_type = 'BASE TABLE'
        ORDER BY t.table_schema, t.table_name
    `

    // Query all foreign key relationships
    relationsQuery := `
        SELECT
            tc.constraint_name,
            tc.table_schema as source_schema,
            tc.table_name as source_table,
            kcu.column_name as source_column,
            ccu.table_schema as target_schema,
            ccu.table_name as target_table,
            ccu.column_name as target_column,
            rc.delete_rule as on_delete,
            rc.update_rule as on_update
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu
            ON tc.constraint_name = kcu.constraint_name
            AND tc.table_schema = kcu.table_schema
        JOIN information_schema.constraint_column_usage ccu
            ON ccu.constraint_name = tc.constraint_name
        JOIN information_schema.referential_constraints rc
            ON rc.constraint_name = tc.constraint_name
        WHERE tc.constraint_type = 'FOREIGN KEY'
        AND tc.table_schema = ANY($1)
        ORDER BY tc.table_schema, tc.table_name
    `

    schemaList := strings.Split(schemas, ",")

    // Execute queries...
    nodes, err := s.querySchemaNodes(ctx, nodesQuery, schemaList)
    if err != nil {
        return SendError(c, fiber.StatusInternalServerError, "SCHEMA_QUERY_FAILED", err.Error())
    }

    relations, err := s.querySchemaRelations(ctx, relationsQuery, schemaList)
    if err != nil {
        return SendError(c, fiber.StatusInternalServerError, "RELATIONS_QUERY_FAILED", err.Error())
    }

    return c.JSON(fiber.Map{
        "nodes":     nodes,
        "edges":     relations,
        "schemas":   schemaList,
    })
}

// GetTableRelationships returns relationships for a specific table
// GET /api/v1/admin/tables/:schema/:table/relationships
func (s *Server) GetTableRelationships(c *fiber.Ctx) error {
    schema := c.Params("schema")
    table := c.Params("table")

    query := `
        WITH outgoing AS (
            SELECT
                'outgoing' as direction,
                tc.constraint_name,
                kcu.column_name as local_column,
                ccu.table_schema as foreign_schema,
                ccu.table_name as foreign_table,
                ccu.column_name as foreign_column,
                rc.delete_rule,
                rc.update_rule
            FROM information_schema.table_constraints tc
            JOIN information_schema.key_column_usage kcu
                ON tc.constraint_name = kcu.constraint_name
            JOIN information_schema.constraint_column_usage ccu
                ON ccu.constraint_name = tc.constraint_name
            JOIN information_schema.referential_constraints rc
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
                rc.delete_rule,
                rc.update_rule
            FROM information_schema.table_constraints tc
            JOIN information_schema.key_column_usage kcu
                ON tc.constraint_name = kcu.constraint_name
            JOIN information_schema.constraint_column_usage ccu
                ON ccu.constraint_name = tc.constraint_name
            JOIN information_schema.referential_constraints rc
                ON rc.constraint_name = tc.constraint_name
            WHERE tc.constraint_type = 'FOREIGN KEY'
            AND ccu.table_schema = $1 AND ccu.table_name = $2
        )
        SELECT * FROM outgoing
        UNION ALL
        SELECT * FROM incoming
        ORDER BY direction, constraint_name
    `

    // Execute and return...
}
```

#### Register Routes in `internal/api/server.go`

```go
// In setupRoutes() function, add:
admin.Get("/schema/graph", s.GetSchemaGraph)
admin.Get("/tables/:schema/:table/relationships", s.GetTableRelationships)
```

### 3.2 Frontend: Schema Viewer Page

#### New Route: `admin/src/routes/_authenticated/schema/index.tsx`

```typescript
import z from 'zod'
import { createFileRoute } from '@tanstack/react-router'
import { SchemaViewer } from '@/features/schema-viewer'

const schemaSearchSchema = z.object({
  schemas: z.string().optional().catch('public'),
  selectedTable: z.string().optional(),
  view: z.enum(['erd', 'list']).optional().catch('erd'),
  zoom: z.number().optional().catch(1),
})

export const Route = createFileRoute('/_authenticated/schema/')({
  validateSearch: schemaSearchSchema,
  component: SchemaViewer,
})
```

#### New Feature: `admin/src/features/schema-viewer/index.tsx`

```typescript
import { useCallback, useMemo, useState } from 'react'
import { getRouteApi } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import ReactFlow, {
  Node,
  Edge,
  Controls,
  Background,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
  Panel,
} from 'reactflow'
import dagre from 'dagre'
import 'reactflow/dist/style.css'

import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { schemaApi } from '@/lib/api'
import { TableNode } from './components/table-node'
import { RelationshipEdge } from './components/relationship-edge'
import { SchemaSelector } from './components/schema-selector'
import { TableDetailsPanel } from './components/table-details-panel'

const route = getRouteApi('/_authenticated/schema/')

// Custom node types
const nodeTypes = {
  table: TableNode,
}

// Custom edge types
const edgeTypes = {
  relationship: RelationshipEdge,
}

export function SchemaViewer() {
  const navigate = route.useNavigate()
  const { schemas, selectedTable, view, zoom } = route.useSearch()

  // Fetch schema graph data
  const { data: graphData, isLoading } = useQuery({
    queryKey: ['schema-graph', schemas],
    queryFn: () => schemaApi.getSchemaGraph(schemas),
    staleTime: 60000,
  })

  // Convert to ReactFlow nodes and edges
  const { initialNodes, initialEdges } = useMemo(() => {
    if (!graphData) return { initialNodes: [], initialEdges: [] }

    // Create nodes from tables
    const nodes: Node[] = graphData.nodes.map((table, index) => ({
      id: `${table.schema}.${table.name}`,
      type: 'table',
      position: { x: 0, y: 0 }, // Will be calculated by dagre
      data: {
        schema: table.schema,
        name: table.name,
        columns: table.columns,
        primaryKey: table.primary_key,
        rlsEnabled: table.rls_enabled,
        rowCount: table.row_count,
        isSelected: selectedTable === `${table.schema}.${table.name}`,
        onSelect: () => navigate({
          search: (prev) => ({ ...prev, selectedTable: `${table.schema}.${table.name}` })
        }),
      },
    }))

    // Create edges from relationships
    const edges: Edge[] = graphData.edges.map((rel) => ({
      id: rel.id,
      source: `${rel.source_schema}.${rel.source_table}`,
      target: `${rel.target_schema}.${rel.target_table}`,
      sourceHandle: rel.source_column,
      targetHandle: rel.target_column,
      type: 'relationship',
      animated: false,
      markerEnd: { type: MarkerType.ArrowClosed },
      data: {
        constraintName: rel.constraint_name,
        onDelete: rel.on_delete,
        onUpdate: rel.on_update,
        sourceColumn: rel.source_column,
        targetColumn: rel.target_column,
      },
    }))

    // Apply dagre layout
    const layoutedNodes = applyDagreLayout(nodes, edges)

    return { initialNodes: layoutedNodes, initialEdges: edges }
  }, [graphData, selectedTable])

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges)

  // Update nodes when data changes
  useEffect(() => {
    setNodes(initialNodes)
    setEdges(initialEdges)
  }, [initialNodes, initialEdges])

  const handleSchemaChange = (newSchemas: string) => {
    navigate({ search: (prev) => ({ ...prev, schemas: newSchemas, selectedTable: undefined }) })
  }

  const handleExport = useCallback(() => {
    // Export as PNG/SVG
    // Implementation using html-to-image or similar
  }, [])

  return (
    <>
      <Header>
        <div className="flex items-center gap-4">
          <h1 className="text-lg font-semibold">Schema Viewer</h1>
          <SchemaSelector value={schemas} onChange={handleSchemaChange} />
        </div>
        <div className="ms-auto flex items-center gap-2">
          <Tabs value={view} onValueChange={(v) => navigate({ search: (prev) => ({ ...prev, view: v }) })}>
            <TabsList>
              <TabsTrigger value="erd">ERD View</TabsTrigger>
              <TabsTrigger value="list">List View</TabsTrigger>
            </TabsList>
          </Tabs>
          <Button variant="outline" size="sm" onClick={handleExport}>
            Export
          </Button>
        </div>
      </Header>

      <Main className="h-[calc(100vh-4rem)] p-0">
        {view === 'erd' ? (
          <div className="flex h-full">
            {/* ERD Canvas */}
            <div className="flex-1">
              <ReactFlow
                nodes={nodes}
                edges={edges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                nodeTypes={nodeTypes}
                edgeTypes={edgeTypes}
                fitView
                minZoom={0.1}
                maxZoom={2}
              >
                <Controls />
                <MiniMap
                  nodeColor={(node) => {
                    const schemaColors: Record<string, string> = {
                      public: '#3b82f6',
                      auth: '#ef4444',
                      storage: '#22c55e',
                    }
                    return schemaColors[node.data?.schema] || '#6b7280'
                  }}
                />
                <Background />

                <Panel position="top-left">
                  <Card className="w-48">
                    <CardHeader className="py-2">
                      <CardTitle className="text-sm">Legend</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-1 py-2">
                      <div className="flex items-center gap-2">
                        <div className="h-3 w-3 rounded bg-blue-500" />
                        <span className="text-xs">public</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="h-3 w-3 rounded bg-red-500" />
                        <span className="text-xs">auth</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="h-3 w-3 rounded bg-green-500" />
                        <span className="text-xs">storage</span>
                      </div>
                    </CardContent>
                  </Card>
                </Panel>
              </ReactFlow>
            </div>

            {/* Details Panel */}
            {selectedTable && (
              <TableDetailsPanel
                tableId={selectedTable}
                onClose={() => navigate({ search: (prev) => ({ ...prev, selectedTable: undefined }) })}
              />
            )}
          </div>
        ) : (
          <SchemaListView data={graphData} />
        )}
      </Main>
    </>
  )
}

// Dagre layout helper
function applyDagreLayout(nodes: Node[], edges: Edge[]): Node[] {
  const dagreGraph = new dagre.graphlib.Graph()
  dagreGraph.setDefaultEdgeLabel(() => ({}))
  dagreGraph.setGraph({ rankdir: 'LR', nodesep: 50, ranksep: 100 })

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: 250, height: 200 })
  })

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target)
  })

  dagre.layout(dagreGraph)

  return nodes.map((node) => {
    const nodeWithPosition = dagreGraph.node(node.id)
    return {
      ...node,
      position: {
        x: nodeWithPosition.x - 125,
        y: nodeWithPosition.y - 100,
      },
    }
  })
}
```

#### Table Node Component: `admin/src/features/schema-viewer/components/table-node.tsx`

```typescript
import { memo } from 'react'
import { Handle, Position } from 'reactflow'
import { Key, Link, Shield, ShieldOff } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

interface TableNodeProps {
  data: {
    schema: string
    name: string
    columns: Array<{
      name: string
      data_type: string
      is_primary_key: boolean
      is_foreign_key: boolean
      fk_target?: string
      nullable: boolean
    }>
    primaryKey: string[]
    rlsEnabled: boolean
    rowCount?: number
    isSelected: boolean
    onSelect: () => void
  }
}

export const TableNode = memo(({ data }: TableNodeProps) => {
  const schemaColors: Record<string, string> = {
    public: 'border-blue-500 bg-blue-50 dark:bg-blue-950',
    auth: 'border-red-500 bg-red-50 dark:bg-red-950',
    storage: 'border-green-500 bg-green-50 dark:bg-green-950',
  }

  return (
    <div
      className={cn(
        'min-w-[220px] rounded-lg border-2 shadow-md cursor-pointer transition-all',
        schemaColors[data.schema] || 'border-gray-500 bg-gray-50 dark:bg-gray-950',
        data.isSelected && 'ring-2 ring-primary ring-offset-2'
      )}
      onClick={data.onSelect}
    >
      {/* Header */}
      <div className="flex items-center justify-between border-b px-3 py-2">
        <div>
          <span className="text-xs text-muted-foreground">{data.schema}.</span>
          <span className="font-semibold">{data.name}</span>
        </div>
        <div className="flex items-center gap-1">
          {data.rlsEnabled ? (
            <Tooltip>
              <TooltipTrigger>
                <Shield className="h-4 w-4 text-green-600" />
              </TooltipTrigger>
              <TooltipContent>RLS Enabled</TooltipContent>
            </Tooltip>
          ) : (
            <Tooltip>
              <TooltipTrigger>
                <ShieldOff className="h-4 w-4 text-amber-600" />
              </TooltipTrigger>
              <TooltipContent>RLS Disabled</TooltipContent>
            </Tooltip>
          )}
          {data.rowCount !== undefined && (
            <Badge variant="secondary" className="text-xs">
              {data.rowCount.toLocaleString()} rows
            </Badge>
          )}
        </div>
      </div>

      {/* Columns */}
      <div className="max-h-[200px] overflow-y-auto">
        {data.columns.slice(0, 10).map((col) => (
          <div
            key={col.name}
            className="relative flex items-center gap-2 border-b px-3 py-1.5 text-sm last:border-b-0"
          >
            {/* Left handle for incoming FKs */}
            <Handle
              type="target"
              position={Position.Left}
              id={col.name}
              className="!h-2 !w-2 !bg-primary"
            />

            {/* Column icons */}
            <div className="flex w-6 items-center justify-center">
              {col.is_primary_key && (
                <Tooltip>
                  <TooltipTrigger>
                    <Key className="h-3 w-3 text-amber-500" />
                  </TooltipTrigger>
                  <TooltipContent>Primary Key</TooltipContent>
                </Tooltip>
              )}
              {col.is_foreign_key && (
                <Tooltip>
                  <TooltipTrigger>
                    <Link className="h-3 w-3 text-blue-500" />
                  </TooltipTrigger>
                  <TooltipContent>FK → {col.fk_target}</TooltipContent>
                </Tooltip>
              )}
            </div>

            {/* Column name and type */}
            <span className={cn('flex-1', col.is_primary_key && 'font-medium')}>
              {col.name}
            </span>
            <span className="text-xs text-muted-foreground">
              {col.data_type}
              {!col.nullable && <span className="text-red-500">*</span>}
            </span>

            {/* Right handle for outgoing FKs */}
            {col.is_foreign_key && (
              <Handle
                type="source"
                position={Position.Right}
                id={col.name}
                className="!h-2 !w-2 !bg-primary"
              />
            )}
          </div>
        ))}
        {data.columns.length > 10 && (
          <div className="px-3 py-1.5 text-xs text-muted-foreground">
            +{data.columns.length - 10} more columns
          </div>
        )}
      </div>
    </div>
  )
})

TableNode.displayName = 'TableNode'
```

### 3.3 Add to Sidebar Navigation

**File:** `admin/src/components/layout/data/sidebar-data.ts`

```typescript
// Add to navGroups under "Database" section:
{
  title: 'Database',
  items: [
    { title: 'Tables', url: '/tables', icon: Table },
    { title: 'Schema Viewer', url: '/schema', icon: GitBranch }, // NEW
    { title: 'SQL Editor', url: '/sql-editor', icon: Code },
    // ...
  ],
}
```

### 3.4 API Client Update

**File:** `admin/src/lib/api.ts`

```typescript
// Add new API module
export const schemaApi = {
  getSchemaGraph: async (schemas?: string): Promise<SchemaGraphResponse> => {
    const params = schemas ? `?schemas=${encodeURIComponent(schemas)}` : ''
    const response = await api.get<SchemaGraphResponse>(`/api/v1/admin/schema/graph${params}`)
    return response.data
  },

  getTableRelationships: async (schema: string, table: string): Promise<TableRelationships> => {
    const response = await api.get<TableRelationships>(
      `/api/v1/admin/tables/${schema}/${table}/relationships`
    )
    return response.data
  },
}

// Add types
export interface SchemaGraphResponse {
  nodes: SchemaNode[]
  edges: SchemaRelationship[]
  schemas: string[]
}

export interface SchemaNode {
  schema: string
  name: string
  columns: SchemaColumn[]
  primary_key: string[]
  rls_enabled: boolean
  row_count?: number
  indexes: IndexInfo[]
}

export interface SchemaColumn {
  name: string
  data_type: string
  nullable: boolean
  is_primary_key: boolean
  is_foreign_key: boolean
  fk_target?: string
  default_value?: string
}

export interface SchemaRelationship {
  id: string
  source_schema: string
  source_table: string
  source_column: string
  target_schema: string
  target_table: string
  target_column: string
  constraint_name: string
  on_delete: string
  on_update: string
}
```

---

## 4. Phase 3: RLS Policy Editor & Security Warnings (Weeks 5-8)

### 4.1 Backend: Policy Introspection API

#### New File: `internal/api/policy_handler.go`

```go
package api

import (
    "context"
    "fmt"
    "strings"

    "github.com/gofiber/fiber/v2"
    "github.com/jackc/pgx/v5"
)

// Policy represents a PostgreSQL RLS policy
type Policy struct {
    Schema      string   `json:"schema"`
    Table       string   `json:"table"`
    PolicyName  string   `json:"policy_name"`
    Permissive  string   `json:"permissive"`  // "PERMISSIVE" or "RESTRICTIVE"
    Roles       []string `json:"roles"`
    Command     string   `json:"command"`     // ALL, SELECT, INSERT, UPDATE, DELETE
    Using       *string  `json:"using"`       // USING expression
    WithCheck   *string  `json:"with_check"`  // WITH CHECK expression
}

// TableRLSStatus represents RLS status for a table
type TableRLSStatus struct {
    Schema       string   `json:"schema"`
    Table        string   `json:"table"`
    RLSEnabled   bool     `json:"rls_enabled"`
    RLSForced    bool     `json:"rls_forced"`
    PolicyCount  int      `json:"policy_count"`
    Policies     []Policy `json:"policies"`
}

// SecurityWarning represents a security issue detected
type SecurityWarning struct {
    ID          string `json:"id"`
    Severity    string `json:"severity"`    // critical, high, medium, low
    Category    string `json:"category"`
    Schema      string `json:"schema"`
    Table       string `json:"table"`
    PolicyName  string `json:"policy_name,omitempty"`
    Message     string `json:"message"`
    Suggestion  string `json:"suggestion"`
    FixSQL      string `json:"fix_sql,omitempty"`
}

// ListPolicies returns all RLS policies
// GET /api/v1/admin/policies
func (s *Server) ListPolicies(c *fiber.Ctx) error {
    ctx := c.Context()
    schema := c.Query("schema", "")

    query := `
        SELECT
            schemaname,
            tablename,
            policyname,
            permissive,
            roles,
            cmd,
            qual,
            with_check
        FROM pg_policies
        WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
    `
    args := []interface{}{}

    if schema != "" {
        query += " AND schemaname = $1"
        args = append(args, schema)
    }
    query += " ORDER BY schemaname, tablename, policyname"

    rows, err := s.pool.Query(ctx, query, args...)
    if err != nil {
        return SendError(c, fiber.StatusInternalServerError, "QUERY_FAILED", err.Error())
    }
    defer rows.Close()

    policies := []Policy{}
    for rows.Next() {
        var p Policy
        var roles []string
        err := rows.Scan(
            &p.Schema, &p.Table, &p.PolicyName, &p.Permissive,
            &roles, &p.Command, &p.Using, &p.WithCheck,
        )
        if err != nil {
            return SendError(c, fiber.StatusInternalServerError, "SCAN_FAILED", err.Error())
        }
        p.Roles = roles
        policies = append(policies, p)
    }

    return c.JSON(policies)
}

// GetTableRLSStatus returns RLS status and policies for a specific table
// GET /api/v1/admin/tables/:schema/:table/rls
func (s *Server) GetTableRLSStatus(c *fiber.Ctx) error {
    ctx := c.Context()
    schema := c.Params("schema")
    table := c.Params("table")

    // Get RLS status
    var status TableRLSStatus
    status.Schema = schema
    status.Table = table

    err := s.pool.QueryRow(ctx, `
        SELECT relrowsecurity, relforcerowsecurity
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = $1 AND c.relname = $2
    `, schema, table).Scan(&status.RLSEnabled, &status.RLSForced)
    if err != nil {
        return SendError(c, fiber.StatusNotFound, "TABLE_NOT_FOUND", "Table not found")
    }

    // Get policies
    rows, err := s.pool.Query(ctx, `
        SELECT policyname, permissive, roles, cmd, qual, with_check
        FROM pg_policies
        WHERE schemaname = $1 AND tablename = $2
        ORDER BY policyname
    `, schema, table)
    if err != nil {
        return SendError(c, fiber.StatusInternalServerError, "QUERY_FAILED", err.Error())
    }
    defer rows.Close()

    for rows.Next() {
        var p Policy
        var roles []string
        err := rows.Scan(&p.PolicyName, &p.Permissive, &roles, &p.Command, &p.Using, &p.WithCheck)
        if err != nil {
            continue
        }
        p.Schema = schema
        p.Table = table
        p.Roles = roles
        status.Policies = append(status.Policies, p)
    }
    status.PolicyCount = len(status.Policies)

    return c.JSON(status)
}

// ToggleTableRLS enables or disables RLS on a table
// POST /api/v1/admin/tables/:schema/:table/rls/toggle
func (s *Server) ToggleTableRLS(c *fiber.Ctx) error {
    ctx := c.Context()
    schema := c.Params("schema")
    table := c.Params("table")

    var req struct {
        Enabled bool `json:"enabled"`
    }
    if err := c.BodyParser(&req); err != nil {
        return SendBadRequest(c, "Invalid request body")
    }

    // Validate table exists
    var exists bool
    err := s.pool.QueryRow(ctx, `
        SELECT EXISTS(
            SELECT 1 FROM pg_class c
            JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE n.nspname = $1 AND c.relname = $2
        )
    `, schema, table).Scan(&exists)
    if err != nil || !exists {
        return SendNotFound(c, "Table not found")
    }

    // Toggle RLS
    action := "DISABLE"
    if req.Enabled {
        action = "ENABLE"
    }

    _, err = s.pool.Exec(ctx, fmt.Sprintf(
        "ALTER TABLE %s.%s %s ROW LEVEL SECURITY",
        quoteIdentifier(schema),
        quoteIdentifier(table),
        action,
    ))
    if err != nil {
        return SendError(c, fiber.StatusInternalServerError, "RLS_TOGGLE_FAILED", err.Error())
    }

    return c.JSON(fiber.Map{
        "success": true,
        "rls_enabled": req.Enabled,
    })
}

// CreatePolicy creates a new RLS policy
// POST /api/v1/admin/policies
func (s *Server) CreatePolicy(c *fiber.Ctx) error {
    ctx := c.Context()

    var req struct {
        Schema     string   `json:"schema"`
        Table      string   `json:"table"`
        Name       string   `json:"name"`
        Command    string   `json:"command"`    // ALL, SELECT, INSERT, UPDATE, DELETE
        Permissive bool     `json:"permissive"` // true = PERMISSIVE, false = RESTRICTIVE
        Roles      []string `json:"roles"`
        Using      string   `json:"using"`
        WithCheck  string   `json:"with_check"`
    }
    if err := c.BodyParser(&req); err != nil {
        return SendBadRequest(c, "Invalid request body")
    }

    // Validate inputs
    if req.Schema == "" || req.Table == "" || req.Name == "" {
        return SendBadRequest(c, "schema, table, and name are required")
    }

    validCommands := map[string]bool{"ALL": true, "SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true}
    if !validCommands[strings.ToUpper(req.Command)] {
        return SendBadRequest(c, "command must be ALL, SELECT, INSERT, UPDATE, or DELETE")
    }

    // Build CREATE POLICY statement
    permissive := "PERMISSIVE"
    if !req.Permissive {
        permissive = "RESTRICTIVE"
    }

    roles := "PUBLIC"
    if len(req.Roles) > 0 {
        quotedRoles := make([]string, len(req.Roles))
        for i, r := range req.Roles {
            quotedRoles[i] = quoteIdentifier(r)
        }
        roles = strings.Join(quotedRoles, ", ")
    }

    sql := fmt.Sprintf(
        "CREATE POLICY %s ON %s.%s AS %s FOR %s TO %s",
        quoteIdentifier(req.Name),
        quoteIdentifier(req.Schema),
        quoteIdentifier(req.Table),
        permissive,
        strings.ToUpper(req.Command),
        roles,
    )

    if req.Using != "" {
        sql += fmt.Sprintf(" USING (%s)", req.Using)
    }
    if req.WithCheck != "" {
        sql += fmt.Sprintf(" WITH CHECK (%s)", req.WithCheck)
    }

    _, err := s.pool.Exec(ctx, sql)
    if err != nil {
        return SendError(c, fiber.StatusBadRequest, "POLICY_CREATE_FAILED", err.Error())
    }

    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "success": true,
        "sql": sql,
    })
}

// DeletePolicy drops an RLS policy
// DELETE /api/v1/admin/policies/:schema/:table/:policy
func (s *Server) DeletePolicy(c *fiber.Ctx) error {
    ctx := c.Context()
    schema := c.Params("schema")
    table := c.Params("table")
    policy := c.Params("policy")

    sql := fmt.Sprintf(
        "DROP POLICY %s ON %s.%s",
        quoteIdentifier(policy),
        quoteIdentifier(schema),
        quoteIdentifier(table),
    )

    _, err := s.pool.Exec(ctx, sql)
    if err != nil {
        return SendError(c, fiber.StatusBadRequest, "POLICY_DELETE_FAILED", err.Error())
    }

    return c.JSON(fiber.Map{"success": true})
}

// GetSecurityWarnings scans for security issues
// GET /api/v1/admin/security/warnings
func (s *Server) GetSecurityWarnings(c *fiber.Ctx) error {
    ctx := c.Context()
    warnings := []SecurityWarning{}

    // Check 1: Tables in public schema without RLS
    rows, err := s.pool.Query(ctx, `
        SELECT c.relname
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public'
        AND c.relkind = 'r'
        AND NOT c.relrowsecurity
        AND c.relname NOT LIKE 'pg_%'
        AND c.relname NOT LIKE '_pg_%'
    `)
    if err == nil {
        defer rows.Close()
        for rows.Next() {
            var tableName string
            rows.Scan(&tableName)
            warnings = append(warnings, SecurityWarning{
                ID:         fmt.Sprintf("no-rls-%s", tableName),
                Severity:   "critical",
                Category:   "rls",
                Schema:     "public",
                Table:      tableName,
                Message:    fmt.Sprintf("Table '%s' does not have Row Level Security enabled", tableName),
                Suggestion: "Enable RLS and create appropriate policies to restrict data access",
                FixSQL:     fmt.Sprintf("ALTER TABLE public.%s ENABLE ROW LEVEL SECURITY;", quoteIdentifier(tableName)),
            })
        }
    }

    // Check 2: RLS enabled but no policies
    rows2, err := s.pool.Query(ctx, `
        SELECT n.nspname, c.relname
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relrowsecurity = true
        AND c.relkind = 'r'
        AND NOT EXISTS (
            SELECT 1 FROM pg_policies p
            WHERE p.schemaname = n.nspname AND p.tablename = c.relname
        )
    `)
    if err == nil {
        defer rows2.Close()
        for rows2.Next() {
            var schema, tableName string
            rows2.Scan(&schema, &tableName)
            warnings = append(warnings, SecurityWarning{
                ID:         fmt.Sprintf("no-policies-%s-%s", schema, tableName),
                Severity:   "high",
                Category:   "rls",
                Schema:     schema,
                Table:      tableName,
                Message:    fmt.Sprintf("Table '%s.%s' has RLS enabled but no policies defined - all access is denied", schema, tableName),
                Suggestion: "Create at least one policy to allow intended access patterns",
            })
        }
    }

    // Check 3: Overly permissive policies (USING true)
    rows3, err := s.pool.Query(ctx, `
        SELECT schemaname, tablename, policyname, cmd
        FROM pg_policies
        WHERE qual = 'true'
        AND cmd != 'SELECT'
    `)
    if err == nil {
        defer rows3.Close()
        for rows3.Next() {
            var schema, tableName, policyName, cmd string
            rows3.Scan(&schema, &tableName, &policyName, &cmd)
            warnings = append(warnings, SecurityWarning{
                ID:         fmt.Sprintf("permissive-%s-%s-%s", schema, tableName, policyName),
                Severity:   "high",
                Category:   "policy",
                Schema:     schema,
                Table:      tableName,
                PolicyName: policyName,
                Message:    fmt.Sprintf("Policy '%s' on %s.%s uses 'USING (true)' for %s - allows unrestricted writes", policyName, schema, tableName, cmd),
                Suggestion: "Restrict the USING clause to appropriate conditions",
            })
        }
    }

    // Check 4: Anon role has write access
    rows4, err := s.pool.Query(ctx, `
        SELECT schemaname, tablename, policyname, cmd
        FROM pg_policies
        WHERE 'anon' = ANY(roles)
        AND cmd IN ('INSERT', 'UPDATE', 'DELETE', 'ALL')
    `)
    if err == nil {
        defer rows4.Close()
        for rows4.Next() {
            var schema, tableName, policyName, cmd string
            rows4.Scan(&schema, &tableName, &policyName, &cmd)
            warnings = append(warnings, SecurityWarning{
                ID:         fmt.Sprintf("anon-write-%s-%s-%s", schema, tableName, policyName),
                Severity:   "high",
                Category:   "policy",
                Schema:     schema,
                Table:      tableName,
                PolicyName: policyName,
                Message:    fmt.Sprintf("Policy '%s' grants %s access to anonymous users", policyName, cmd),
                Suggestion: "Review if anonymous write access is intentional",
            })
        }
    }

    // Check 5: Missing WITH CHECK on INSERT/UPDATE policies
    rows5, err := s.pool.Query(ctx, `
        SELECT schemaname, tablename, policyname, cmd
        FROM pg_policies
        WHERE cmd IN ('INSERT', 'UPDATE', 'ALL')
        AND with_check IS NULL
        AND permissive = 'PERMISSIVE'
    `)
    if err == nil {
        defer rows5.Close()
        for rows5.Next() {
            var schema, tableName, policyName, cmd string
            rows5.Scan(&schema, &tableName, &policyName, &cmd)
            warnings = append(warnings, SecurityWarning{
                ID:         fmt.Sprintf("no-check-%s-%s-%s", schema, tableName, policyName),
                Severity:   "medium",
                Category:   "policy",
                Schema:     schema,
                Table:      tableName,
                PolicyName: policyName,
                Message:    fmt.Sprintf("Policy '%s' has no WITH CHECK clause for %s operations", policyName, cmd),
                Suggestion: "Add WITH CHECK to validate data on insert/update",
            })
        }
    }

    return c.JSON(fiber.Map{
        "warnings": warnings,
        "summary": fiber.Map{
            "total":    len(warnings),
            "critical": countBySeverity(warnings, "critical"),
            "high":     countBySeverity(warnings, "high"),
            "medium":   countBySeverity(warnings, "medium"),
            "low":      countBySeverity(warnings, "low"),
        },
    })
}

func countBySeverity(warnings []SecurityWarning, severity string) int {
    count := 0
    for _, w := range warnings {
        if w.Severity == severity {
            count++
        }
    }
    return count
}
```

#### Register Routes

```go
// In server.go setupRoutes()
admin.Get("/policies", s.ListPolicies)
admin.Post("/policies", s.CreatePolicy)
admin.Delete("/policies/:schema/:table/:policy", s.DeletePolicy)
admin.Get("/tables/:schema/:table/rls", s.GetTableRLSStatus)
admin.Post("/tables/:schema/:table/rls/toggle", s.ToggleTableRLS)
admin.Get("/security/warnings", s.GetSecurityWarnings)
```

### 4.2 Frontend: Policies Page

#### New Route: `admin/src/routes/_authenticated/policies/index.tsx`

```typescript
import z from 'zod'
import { createFileRoute } from '@tanstack/react-router'
import { Policies } from '@/features/policies'

const policySearchSchema = z.object({
  schema: z.string().optional().catch('public'),
  table: z.string().optional(),
  tab: z.enum(['overview', 'warnings', 'templates']).optional().catch('overview'),
})

export const Route = createFileRoute('/_authenticated/policies/')({
  validateSearch: policySearchSchema,
  component: Policies,
})
```

#### New Feature: `admin/src/features/policies/index.tsx`

```typescript
import { useState } from 'react'
import { getRouteApi } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Shield,
  ShieldAlert,
  ShieldCheck,
  ShieldOff,
  Plus,
  AlertTriangle,
  CheckCircle,
  XCircle,
} from 'lucide-react'

import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { toast } from 'sonner'

import { policyApi, databaseApi } from '@/lib/api'
import { PolicyEditor } from './components/policy-editor'
import { SecurityWarnings } from './components/security-warnings'
import { PolicyTemplates } from './components/policy-templates'
import { PolicyCard } from './components/policy-card'

const route = getRouteApi('/_authenticated/policies/')

export function Policies() {
  const navigate = route.useNavigate()
  const { schema, table, tab } = route.useSearch()
  const queryClient = useQueryClient()

  const [editorOpen, setEditorOpen] = useState(false)
  const [editingPolicy, setEditingPolicy] = useState<Policy | null>(null)

  // Fetch tables with RLS status
  const { data: tables, isLoading: tablesLoading } = useQuery({
    queryKey: ['tables-rls', schema],
    queryFn: () => databaseApi.getTablesWithRLS(schema),
  })

  // Fetch security warnings
  const { data: warnings } = useQuery({
    queryKey: ['security-warnings'],
    queryFn: () => policyApi.getSecurityWarnings(),
  })

  // Toggle RLS mutation
  const toggleRLSMutation = useMutation({
    mutationFn: ({ schema, table, enabled }: { schema: string; table: string; enabled: boolean }) =>
      policyApi.toggleTableRLS(schema, table, enabled),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['tables-rls'] })
      queryClient.invalidateQueries({ queryKey: ['security-warnings'] })
      toast.success(`RLS ${variables.enabled ? 'enabled' : 'disabled'} on ${variables.table}`)
    },
    onError: (error: Error) => {
      toast.error(`Failed to toggle RLS: ${error.message}`)
    },
  })

  const handleCreatePolicy = () => {
    setEditingPolicy(null)
    setEditorOpen(true)
  }

  const handleEditPolicy = (policy: Policy) => {
    setEditingPolicy(policy)
    setEditorOpen(true)
  }

  const securityScore = calculateSecurityScore(warnings?.summary)

  return (
    <>
      <Header>
        <div className="flex items-center gap-4">
          <Shield className="h-6 w-6" />
          <h1 className="text-lg font-semibold">Row Level Security</h1>
        </div>
        <div className="ms-auto flex items-center gap-2">
          <Button onClick={handleCreatePolicy}>
            <Plus className="mr-2 h-4 w-4" />
            Create Policy
          </Button>
        </div>
      </Header>

      <Main>
        <Tabs value={tab} onValueChange={(v) => navigate({ search: (prev) => ({ ...prev, tab: v }) })}>
          <div className="mb-6 flex items-center justify-between">
            <TabsList>
              <TabsTrigger value="overview">Overview</TabsTrigger>
              <TabsTrigger value="warnings" className="relative">
                Warnings
                {warnings?.summary?.total > 0 && (
                  <Badge variant="destructive" className="ml-2">
                    {warnings.summary.total}
                  </Badge>
                )}
              </TabsTrigger>
              <TabsTrigger value="templates">Templates</TabsTrigger>
            </TabsList>

            {/* Security Score */}
            <Card className="w-48">
              <CardContent className="flex items-center gap-3 py-3">
                <SecurityScoreBadge score={securityScore} />
                <div>
                  <div className="text-sm font-medium">Security Score</div>
                  <div className="text-xs text-muted-foreground">
                    {warnings?.summary?.total || 0} issues
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          <TabsContent value="overview" className="space-y-4">
            {/* Schema selector */}
            <SchemaSelector
              value={schema}
              onChange={(s) => navigate({ search: (prev) => ({ ...prev, schema: s }) })}
            />

            {/* Table list with RLS status */}
            <div className="space-y-2">
              {tables?.map((table) => (
                <Accordion key={`${table.schema}.${table.name}`} type="single" collapsible>
                  <AccordionItem value={table.name} className="border rounded-lg">
                    <AccordionTrigger className="px-4 hover:no-underline">
                      <div className="flex items-center gap-4 flex-1">
                        {/* RLS Toggle */}
                        <Switch
                          checked={table.rls_enabled}
                          onCheckedChange={(enabled) => {
                            toggleRLSMutation.mutate({
                              schema: table.schema,
                              table: table.name,
                              enabled,
                            })
                          }}
                          onClick={(e) => e.stopPropagation()}
                        />

                        {/* Table name */}
                        <span className="font-medium">{table.name}</span>

                        {/* Status badges */}
                        <div className="flex items-center gap-2 ms-auto mr-4">
                          {table.rls_enabled ? (
                            <Badge variant="outline" className="text-green-600 border-green-600">
                              <ShieldCheck className="mr-1 h-3 w-3" />
                              RLS Enabled
                            </Badge>
                          ) : (
                            <Badge variant="outline" className="text-amber-600 border-amber-600">
                              <ShieldOff className="mr-1 h-3 w-3" />
                              RLS Disabled
                            </Badge>
                          )}

                          <Badge variant="secondary">
                            {table.policy_count} {table.policy_count === 1 ? 'policy' : 'policies'}
                          </Badge>

                          {/* Warning indicator */}
                          {table.has_warnings && (
                            <AlertTriangle className="h-4 w-4 text-amber-500" />
                          )}
                        </div>
                      </div>
                    </AccordionTrigger>

                    <AccordionContent className="px-4 pb-4">
                      {table.policies?.length > 0 ? (
                        <div className="space-y-3">
                          {table.policies.map((policy) => (
                            <PolicyCard
                              key={policy.policy_name}
                              policy={policy}
                              onEdit={() => handleEditPolicy(policy)}
                              onDelete={() => handleDeletePolicy(policy)}
                            />
                          ))}
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => {
                              setEditingPolicy({ schema: table.schema, table: table.name } as Policy)
                              setEditorOpen(true)
                            }}
                          >
                            <Plus className="mr-2 h-3 w-3" />
                            Add Policy
                          </Button>
                        </div>
                      ) : (
                        <div className="text-center py-6 text-muted-foreground">
                          <ShieldAlert className="mx-auto h-8 w-8 mb-2" />
                          <p>No policies defined</p>
                          <Button
                            variant="outline"
                            size="sm"
                            className="mt-2"
                            onClick={() => {
                              setEditingPolicy({ schema: table.schema, table: table.name } as Policy)
                              setEditorOpen(true)
                            }}
                          >
                            <Plus className="mr-2 h-3 w-3" />
                            Create First Policy
                          </Button>
                        </div>
                      )}
                    </AccordionContent>
                  </AccordionItem>
                </Accordion>
              ))}
            </div>
          </TabsContent>

          <TabsContent value="warnings">
            <SecurityWarnings warnings={warnings?.warnings || []} />
          </TabsContent>

          <TabsContent value="templates">
            <PolicyTemplates
              onApply={(template) => {
                setEditingPolicy(template)
                setEditorOpen(true)
              }}
            />
          </TabsContent>
        </Tabs>

        {/* Policy Editor Dialog */}
        <PolicyEditor
          open={editorOpen}
          onOpenChange={setEditorOpen}
          policy={editingPolicy}
          onSave={() => {
            queryClient.invalidateQueries({ queryKey: ['tables-rls'] })
            queryClient.invalidateQueries({ queryKey: ['security-warnings'] })
            setEditorOpen(false)
          }}
        />
      </Main>
    </>
  )
}

function SecurityScoreBadge({ score }: { score: string }) {
  const colors: Record<string, string> = {
    A: 'bg-green-500',
    B: 'bg-lime-500',
    C: 'bg-yellow-500',
    D: 'bg-orange-500',
    F: 'bg-red-500',
  }

  return (
    <div className={`h-10 w-10 rounded-full ${colors[score]} flex items-center justify-center text-white font-bold`}>
      {score}
    </div>
  )
}

function calculateSecurityScore(summary?: { critical: number; high: number; medium: number; low: number }): string {
  if (!summary) return 'A'
  const score = summary.critical * 25 + summary.high * 10 + summary.medium * 3 + summary.low * 1
  if (score === 0) return 'A'
  if (score <= 5) return 'B'
  if (score <= 15) return 'C'
  if (score <= 30) return 'D'
  return 'F'
}
```

#### Policy Editor Component: `admin/src/features/policies/components/policy-editor.tsx`

```typescript
import { useState, useEffect } from 'react'
import { useMutation } from '@tanstack/react-query'
import Editor from '@monaco-editor/react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { toast } from 'sonner'
import { policyApi } from '@/lib/api'

interface PolicyEditorProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  policy: Policy | null
  onSave: () => void
}

export function PolicyEditor({ open, onOpenChange, policy, onSave }: PolicyEditorProps) {
  const [formData, setFormData] = useState({
    schema: policy?.schema || 'public',
    table: policy?.table || '',
    name: policy?.policy_name || '',
    command: policy?.command || 'SELECT',
    permissive: policy?.permissive !== 'RESTRICTIVE',
    roles: policy?.roles || ['authenticated'],
    using: policy?.using || '',
    withCheck: policy?.with_check || '',
  })

  const [previewSQL, setPreviewSQL] = useState('')

  useEffect(() => {
    // Generate SQL preview
    const roles = formData.roles.length > 0 ? formData.roles.join(', ') : 'PUBLIC'
    let sql = `CREATE POLICY "${formData.name}"
  ON ${formData.schema}.${formData.table}
  AS ${formData.permissive ? 'PERMISSIVE' : 'RESTRICTIVE'}
  FOR ${formData.command}
  TO ${roles}`

    if (formData.using) {
      sql += `
  USING (${formData.using})`
    }
    if (formData.withCheck) {
      sql += `
  WITH CHECK (${formData.withCheck})`
    }
    sql += ';'
    setPreviewSQL(sql)
  }, [formData])

  const createMutation = useMutation({
    mutationFn: () => policyApi.createPolicy(formData),
    onSuccess: () => {
      toast.success('Policy created successfully')
      onSave()
    },
    onError: (error: Error) => {
      toast.error(`Failed to create policy: ${error.message}`)
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    createMutation.mutate()
  }

  // Common expression helpers
  const expressionHelpers = [
    { label: 'Current user ID', value: 'auth.uid()' },
    { label: 'Current user role', value: "auth.jwt() ->> 'role'" },
    { label: 'Is authenticated', value: "auth.role() = 'authenticated'" },
    { label: 'Is admin', value: 'auth.is_admin()' },
    { label: 'User owns row', value: 'auth.uid() = user_id' },
    { label: 'Always true', value: 'true' },
    { label: 'Always false', value: 'false' },
  ]

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {policy?.policy_name ? 'Edit Policy' : 'Create Policy'}
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          <Tabs defaultValue="basic">
            <TabsList>
              <TabsTrigger value="basic">Basic</TabsTrigger>
              <TabsTrigger value="expressions">Expressions</TabsTrigger>
              <TabsTrigger value="preview">SQL Preview</TabsTrigger>
            </TabsList>

            <TabsContent value="basic" className="space-y-4 pt-4">
              {/* Schema and Table */}
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Schema</Label>
                  <Select
                    value={formData.schema}
                    onValueChange={(v) => setFormData({ ...formData, schema: v })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="public">public</SelectItem>
                      <SelectItem value="auth">auth</SelectItem>
                      <SelectItem value="storage">storage</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label>Table</Label>
                  <Input
                    value={formData.table}
                    onChange={(e) => setFormData({ ...formData, table: e.target.value })}
                    placeholder="table_name"
                    required
                  />
                </div>
              </div>

              {/* Policy Name */}
              <div className="space-y-2">
                <Label>Policy Name</Label>
                <Input
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="e.g., users_can_read_own_data"
                  required
                />
              </div>

              {/* Command and Type */}
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Command</Label>
                  <Select
                    value={formData.command}
                    onValueChange={(v) => setFormData({ ...formData, command: v })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="ALL">ALL</SelectItem>
                      <SelectItem value="SELECT">SELECT</SelectItem>
                      <SelectItem value="INSERT">INSERT</SelectItem>
                      <SelectItem value="UPDATE">UPDATE</SelectItem>
                      <SelectItem value="DELETE">DELETE</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label>Type</Label>
                  <div className="flex items-center gap-2 pt-2">
                    <Switch
                      checked={formData.permissive}
                      onCheckedChange={(v) => setFormData({ ...formData, permissive: v })}
                    />
                    <span className="text-sm">
                      {formData.permissive ? 'Permissive (OR)' : 'Restrictive (AND)'}
                    </span>
                  </div>
                </div>
              </div>

              {/* Roles */}
              <div className="space-y-2">
                <Label>Roles</Label>
                <RoleMultiSelect
                  value={formData.roles}
                  onChange={(roles) => setFormData({ ...formData, roles })}
                />
              </div>
            </TabsContent>

            <TabsContent value="expressions" className="space-y-4 pt-4">
              {/* Expression helpers */}
              <div className="flex flex-wrap gap-2 mb-4">
                <span className="text-sm text-muted-foreground">Insert:</span>
                {expressionHelpers.map((helper) => (
                  <Button
                    key={helper.value}
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      // Insert at cursor position in active editor
                      setFormData({ ...formData, using: formData.using + helper.value })
                    }}
                  >
                    {helper.label}
                  </Button>
                ))}
              </div>

              {/* USING expression */}
              <div className="space-y-2">
                <Label>USING Expression (for SELECT, UPDATE, DELETE)</Label>
                <div className="h-32 border rounded-md overflow-hidden">
                  <Editor
                    height="100%"
                    language="sql"
                    value={formData.using}
                    onChange={(value) => setFormData({ ...formData, using: value || '' })}
                    options={{
                      minimap: { enabled: false },
                      lineNumbers: 'off',
                      scrollBeyondLastLine: false,
                    }}
                    theme="vs-dark"
                  />
                </div>
                <p className="text-xs text-muted-foreground">
                  Expression that must be true for existing rows to be visible/affected
                </p>
              </div>

              {/* WITH CHECK expression */}
              <div className="space-y-2">
                <Label>WITH CHECK Expression (for INSERT, UPDATE)</Label>
                <div className="h-32 border rounded-md overflow-hidden">
                  <Editor
                    height="100%"
                    language="sql"
                    value={formData.withCheck}
                    onChange={(value) => setFormData({ ...formData, withCheck: value || '' })}
                    options={{
                      minimap: { enabled: false },
                      lineNumbers: 'off',
                      scrollBeyondLastLine: false,
                    }}
                    theme="vs-dark"
                  />
                </div>
                <p className="text-xs text-muted-foreground">
                  Expression that must be true for new/updated rows to be accepted
                </p>
              </div>
            </TabsContent>

            <TabsContent value="preview" className="pt-4">
              <div className="h-64 border rounded-md overflow-hidden">
                <Editor
                  height="100%"
                  language="sql"
                  value={previewSQL}
                  options={{
                    readOnly: true,
                    minimap: { enabled: false },
                  }}
                  theme="vs-dark"
                />
              </div>
            </TabsContent>
          </Tabs>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? 'Creating...' : 'Create Policy'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
```

### 4.3 Add to Sidebar Navigation

```typescript
// In sidebar-data.ts
{
  title: 'Security',
  items: [
    { title: 'Policies', url: '/policies', icon: Shield }, // NEW
    { title: 'Client Keys', url: '/client-keys', icon: Key },
    { title: 'Service Keys', url: '/service-keys', icon: KeySquare },
  ],
}
```

---

## 5. Phase 4: Enhanced DDL Operations (Weeks 9-10)

### 5.1 Backend: Extended DDL Handler

**File:** `internal/api/ddl_handler.go` (extend existing)

Add support for:

```go
// CreateForeignKeyRequest for creating foreign key constraints
type CreateForeignKeyRequest struct {
    Schema           string `json:"schema"`
    Table            string `json:"table"`
    ConstraintName   string `json:"constraint_name"`
    Columns          []string `json:"columns"`
    ReferencedSchema string `json:"referenced_schema"`
    ReferencedTable  string `json:"referenced_table"`
    ReferencedColumns []string `json:"referenced_columns"`
    OnDelete         string `json:"on_delete"` // CASCADE, SET NULL, RESTRICT, NO ACTION
    OnUpdate         string `json:"on_update"`
}

// POST /api/v1/admin/tables/:schema/:table/foreign-keys
func (s *Server) CreateForeignKey(c *fiber.Ctx) error {
    // Implementation
}

// DELETE /api/v1/admin/tables/:schema/:table/foreign-keys/:name
func (s *Server) DropForeignKey(c *fiber.Ctx) error {
    // Implementation
}

// CreateIndexRequest for creating indexes
type CreateIndexRequest struct {
    Schema   string   `json:"schema"`
    Table    string   `json:"table"`
    Name     string   `json:"name"`
    Columns  []string `json:"columns"`
    Unique   bool     `json:"unique"`
    Method   string   `json:"method"` // btree, hash, gin, gist
    Where    string   `json:"where"`  // partial index condition
}

// POST /api/v1/admin/tables/:schema/:table/indexes
func (s *Server) CreateIndex(c *fiber.Ctx) error {
    // Implementation
}

// DELETE /api/v1/admin/tables/:schema/:table/indexes/:name
func (s *Server) DropIndex(c *fiber.Ctx) error {
    // Implementation
}
```

---

## 6. Phase 5: Developer Experience (Weeks 11-14)

### 6.1 Go SDK

Create `pkg/client/` directory with:

```
pkg/client/
  client.go          # Main client struct
  auth.go            # Authentication methods
  query.go           # Query builder
  storage.go         # Storage operations
  realtime.go        # WebSocket subscriptions
  functions.go       # Function invocation
  types.go           # Shared types
  errors.go          # Error types
```

### 6.2 Migration UI

Add route: `/migrations` with:
- Migration list (applied/pending)
- Apply/rollback buttons
- Migration editor (create new)
- Diff viewer

### 6.3 React SDK Expansion

Add hooks for missing features:
- `useFunctions()` - Function management
- `useJobs()` - Job management
- `useMigrations()` - Migration status
- `useRPC()` - RPC execution
- `useVectors()` - Vector search

---

## 7. Phase 6: Advanced Features (Weeks 15-20)

### 7.1 Performance Advisor

Backend scans for:
- Missing indexes on FK columns
- Slow query patterns
- Table bloat
- Unused indexes

### 7.2 Message Queue (pgmq-like)

New module: `internal/queue/`
- Queue management
- Message publish/consume
- Dead letter handling
- Admin UI

### 7.3 Audit Logging

New table: `admin.audit_log`
Track all admin operations with user, action, resource, timestamp.

---

## 8. File-by-File Implementation Details

### New Files to Create

| File | Purpose | Phase |
|------|---------|-------|
| `internal/config/validation.go` | Startup security validation | 1 |
| `internal/api/schema_relationships_handler.go` | Schema graph API | 2 |
| `internal/api/policy_handler.go` | RLS policy CRUD API | 3 |
| `admin/src/routes/_authenticated/schema/index.tsx` | Schema viewer route | 2 |
| `admin/src/features/schema-viewer/index.tsx` | Schema viewer page | 2 |
| `admin/src/features/schema-viewer/components/table-node.tsx` | ERD table node | 2 |
| `admin/src/features/schema-viewer/components/relationship-edge.tsx` | ERD edge | 2 |
| `admin/src/features/schema-viewer/components/table-details-panel.tsx` | Details sidebar | 2 |
| `admin/src/routes/_authenticated/policies/index.tsx` | Policies route | 3 |
| `admin/src/features/policies/index.tsx` | Policies page | 3 |
| `admin/src/features/policies/components/policy-editor.tsx` | Policy editor dialog | 3 |
| `admin/src/features/policies/components/policy-card.tsx` | Policy display card | 3 |
| `admin/src/features/policies/components/security-warnings.tsx` | Warnings list | 3 |
| `admin/src/features/policies/components/policy-templates.tsx` | Template library | 3 |
| `pkg/client/*.go` | Go SDK | 5 |

### Files to Modify

| File | Changes | Phase |
|------|---------|-------|
| `cmd/fluxbase/main.go` | Add security validation call | 1 |
| `internal/api/server.go` | Register new routes | 2, 3 |
| `internal/api/ddl_handler.go` | Add FK/index methods | 4 |
| `admin/src/lib/api.ts` | Add schema, policy API modules | 2, 3 |
| `admin/src/components/layout/data/sidebar-data.ts` | Add navigation items | 2, 3 |

---

## 9. Database Migrations

### Migration 072: Admin Audit Log

```sql
-- 072_admin_audit_log.up.sql
CREATE TABLE IF NOT EXISTS admin.audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.dashboard_users(id),
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id TEXT,
    resource_schema TEXT,
    resource_table TEXT,
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_user_id ON admin.audit_log(user_id);
CREATE INDEX idx_audit_log_action ON admin.audit_log(action);
CREATE INDEX idx_audit_log_resource ON admin.audit_log(resource_type, resource_id);
CREATE INDEX idx_audit_log_created_at ON admin.audit_log(created_at DESC);

-- 072_admin_audit_log.down.sql
DROP TABLE IF EXISTS admin.audit_log;
```

---

## 10. Testing Strategy

### Unit Tests

| Component | Test File | Coverage Target |
|-----------|-----------|-----------------|
| Security validation | `internal/config/validation_test.go` | 90% |
| Schema relationships API | `internal/api/schema_relationships_handler_test.go` | 80% |
| Policy CRUD API | `internal/api/policy_handler_test.go` | 85% |
| DDL extensions | `internal/api/ddl_handler_test.go` | 80% |

### E2E Tests

| Feature | Test File |
|---------|-----------|
| Schema viewer | `test/e2e/schema_viewer_test.go` |
| Policy management | `test/e2e/policy_management_test.go` |
| Security warnings | `test/e2e/security_advisor_test.go` |

### Frontend Tests

| Component | Test File |
|-----------|-----------|
| Schema viewer | `admin/src/features/schema-viewer/__tests__/index.test.tsx` |
| Policy editor | `admin/src/features/policies/__tests__/policy-editor.test.tsx` |

---

## Dependencies to Add

### Backend (go.mod)

None required - all functionality uses existing pgx/v5 and Fiber.

### Frontend (package.json)

```json
{
  "dependencies": {
    "reactflow": "^11.10.0",
    "dagre": "^0.8.5",
    "@types/dagre": "^0.7.52"
  }
}
```

---

## Summary

This implementation plan provides a complete roadmap for building:

1. **Security fixes** - SQL injection (done), startup validation, rate limiting warnings
2. **Visual Schema Viewer** - ERD canvas with ReactFlow, table nodes, relationship edges
3. **RLS Policy Editor** - Policy CRUD, security warnings, templates, SQL preview
4. **Enhanced DDL** - Foreign keys, indexes, constraints via API and UI
5. **Developer Experience** - Go SDK, migration UI, React SDK expansion
6. **Advanced Features** - Performance advisor, message queue, audit logging

Each phase builds on the previous, with clear file locations, code patterns matching the existing codebase, and comprehensive test coverage.
