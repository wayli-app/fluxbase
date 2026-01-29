import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useState, useCallback, useMemo } from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  GitFork,
  Loader2,
  AlertCircle,
  Search,
  Database,
  Key,
  Link as LinkIcon,
  Eye,
  Shield,
  ShieldOff,
  ArrowRight,
  ArrowLeft,
  Columns,
  LayoutGrid,
  List,
  ZoomIn,
  ZoomOut,
  Maximize2,
} from 'lucide-react'
import {
  schemaApi,
  type SchemaNode,
  type SchemaRelationship,
  type SchemaGraphResponse,
} from '@/lib/api'
import { cn } from '@/lib/utils'

export const Route = createFileRoute('/_authenticated/schema/')({
  component: SchemaViewerPage,
})

type ViewMode = 'erd' | 'list'

function SchemaViewerPage() {
  const [viewMode, setViewMode] = useState<ViewMode>('erd')
  const [selectedSchemas, setSelectedSchemas] = useState<string[]>(['public'])
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedTable, setSelectedTable] = useState<string | null>(null)
  const [zoom, setZoom] = useState(1)

  const { data, isLoading, error } = useQuery({
    queryKey: ['schema-graph', selectedSchemas],
    queryFn: () => schemaApi.getGraph(selectedSchemas),
  })

  // Helper to get full name for a node
  const getFullName = (node: SchemaNode) => `${node.schema}.${node.name}`

  // Get unique schemas from the data
  const availableSchemas = useMemo(() => {
    if (!data?.nodes) return ['public']
    const schemas = new Set(data.nodes.map((n) => n.schema))
    return Array.from(schemas).sort()
  }, [data])

  // Filter nodes based on search
  const filteredNodes = useMemo(() => {
    if (!data?.nodes) return []
    if (!searchQuery) return data.nodes
    const query = searchQuery.toLowerCase()
    return data.nodes.filter(
      (n) =>
        n.name.toLowerCase().includes(query) ||
        getFullName(n).toLowerCase().includes(query) ||
        n.columns.some((c) => c.name.toLowerCase().includes(query))
    )
  }, [data?.nodes, searchQuery])

  // Get relationships for filtered nodes (use edges from API)
  const filteredRelationships = useMemo(() => {
    if (!data?.edges || !filteredNodes.length) return []
    const nodeNames = new Set(filteredNodes.map((n) => getFullName(n)))
    return data.edges.filter(
      (r) =>
        nodeNames.has(`${r.source_schema}.${r.source_table}`) ||
        nodeNames.has(`${r.target_schema}.${r.target_table}`)
    )
  }, [data?.edges, filteredNodes])

  // Get selected table details
  const selectedTableData = useMemo(() => {
    if (!selectedTable || !data?.nodes) return null
    return data.nodes.find((n) => getFullName(n) === selectedTable)
  }, [selectedTable, data?.nodes])

  // Get relationships for selected table
  const selectedTableRelationships = useMemo(() => {
    if (!selectedTable || !data?.edges) return { incoming: [], outgoing: [] }
    const [schema, table] = selectedTable.split('.')
    return {
      incoming: data.edges.filter(
        (r) => r.target_schema === schema && r.target_table === table
      ),
      outgoing: data.edges.filter(
        (r) => r.source_schema === schema && r.source_table === table
      ),
    }
  }, [selectedTable, data?.edges])

  const handleSchemaChange = (schema: string) => {
    if (schema === 'all') {
      setSelectedSchemas(availableSchemas)
    } else {
      setSelectedSchemas([schema])
    }
  }

  if (error) {
    return (
      <div className="flex flex-1 flex-col gap-6 p-6">
        <div className="flex items-center gap-2 text-destructive">
          <AlertCircle className="h-5 w-5" />
          <span>Failed to load schema graph</span>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-1 flex-col gap-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <GitFork className="h-8 w-8" />
            Schema Viewer
          </h1>
          <p className="text-sm text-muted-foreground mt-2">
            Visualize database tables and their relationships
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant={viewMode === 'erd' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setViewMode('erd')}
          >
            <LayoutGrid className="h-4 w-4 mr-2" />
            ERD View
          </Button>
          <Button
            variant={viewMode === 'list' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setViewMode('list')}
          >
            <List className="h-4 w-4 mr-2" />
            List View
          </Button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search tables, columns..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select
          value={selectedSchemas.length === availableSchemas.length ? 'all' : selectedSchemas[0]}
          onValueChange={handleSchemaChange}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Select schema" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Schemas</SelectItem>
            {availableSchemas.map((schema) => (
              <SelectItem key={schema} value={schema}>
                {schema}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {viewMode === 'erd' && (
          <div className="flex items-center gap-1">
            <Button
              variant="outline"
              size="icon"
              onClick={() => setZoom((z) => Math.max(0.25, z - 0.25))}
            >
              <ZoomOut className="h-4 w-4" />
            </Button>
            <span className="text-sm text-muted-foreground w-12 text-center">
              {Math.round(zoom * 100)}%
            </span>
            <Button
              variant="outline"
              size="icon"
              onClick={() => setZoom((z) => Math.min(2, z + 0.25))}
            >
              <ZoomIn className="h-4 w-4" />
            </Button>
            <Button variant="outline" size="icon" onClick={() => setZoom(1)}>
              <Maximize2 className="h-4 w-4" />
            </Button>
          </div>
        )}
      </div>

      {isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : viewMode === 'erd' ? (
        <div className="flex gap-6 flex-1">
          {/* ERD Canvas */}
          <div className="flex-1 border rounded-lg bg-muted/20 overflow-auto relative">
            <ERDCanvas
              nodes={filteredNodes}
              relationships={filteredRelationships}
              zoom={zoom}
              selectedTable={selectedTable}
              onSelectTable={setSelectedTable}
            />
          </div>

          {/* Table Details Panel */}
          {selectedTable && selectedTableData && (
            <Card className="w-96 shrink-0">
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Database className="h-4 w-4" />
                    {selectedTableData.name}
                  </CardTitle>
                  {selectedTableData.rls_enabled ? (
                    <Badge variant="default" className="gap-1">
                      <Shield className="h-3 w-3" />
                      RLS
                    </Badge>
                  ) : (
                    <Badge variant="secondary" className="gap-1">
                      <ShieldOff className="h-3 w-3" />
                      No RLS
                    </Badge>
                  )}
                </div>
                <CardDescription>{getFullName(selectedTableData)}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Columns */}
                <div>
                  <h4 className="text-sm font-medium mb-2 flex items-center gap-2">
                    <Columns className="h-4 w-4" />
                    Columns ({selectedTableData.columns.length})
                  </h4>
                  <div className="space-y-1 max-h-48 overflow-auto">
                    {selectedTableData.columns.map((col) => (
                      <div
                        key={col.name}
                        className="flex items-center justify-between text-sm py-1 px-2 rounded hover:bg-muted"
                      >
                        <div className="flex items-center gap-2">
                          {col.is_primary_key && (
                            <Key className="h-3 w-3 text-yellow-500" />
                          )}
                          {col.is_foreign_key && (
                            <LinkIcon className="h-3 w-3 text-blue-500" />
                          )}
                          <span className={cn(col.is_primary_key && 'font-medium')}>
                            {col.name}
                          </span>
                        </div>
                        <span className="text-muted-foreground text-xs">
                          {col.data_type}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Relationships */}
                {(selectedTableRelationships.incoming.length > 0 ||
                  selectedTableRelationships.outgoing.length > 0) && (
                  <div>
                    <h4 className="text-sm font-medium mb-2 flex items-center gap-2">
                      <GitFork className="h-4 w-4" />
                      Relationships
                    </h4>
                    <div className="space-y-2">
                      {selectedTableRelationships.outgoing.map((rel) => (
                        <div
                          key={rel.id}
                          className="flex items-center gap-2 text-sm p-2 rounded bg-muted/50 cursor-pointer hover:bg-muted"
                          onClick={() =>
                            setSelectedTable(
                              `${rel.target_schema}.${rel.target_table}`
                            )
                          }
                        >
                          <ArrowRight className="h-4 w-4 text-blue-500" />
                          <span className="text-muted-foreground">
                            {rel.source_column}
                          </span>
                          <ArrowRight className="h-3 w-3" />
                          <span className="font-medium">
                            {rel.target_schema}.{rel.target_table}
                          </span>
                          <span className="text-muted-foreground">
                            ({rel.target_column})
                          </span>
                        </div>
                      ))}
                      {selectedTableRelationships.incoming.map((rel) => (
                        <div
                          key={rel.id}
                          className="flex items-center gap-2 text-sm p-2 rounded bg-muted/50 cursor-pointer hover:bg-muted"
                          onClick={() =>
                            setSelectedTable(
                              `${rel.source_schema}.${rel.source_table}`
                            )
                          }
                        >
                          <ArrowLeft className="h-4 w-4 text-green-500" />
                          <span className="font-medium">
                            {rel.source_schema}.{rel.source_table}
                          </span>
                          <span className="text-muted-foreground">
                            ({rel.source_column})
                          </span>
                          <ArrowRight className="h-3 w-3" />
                          <span className="text-muted-foreground">
                            {rel.target_column}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Row count */}
                {selectedTableData.row_estimate !== undefined && (
                  <div className="text-sm text-muted-foreground">
                    ~{selectedTableData.row_estimate.toLocaleString()} rows
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      ) : (
        /* List View */
        <Card>
          <CardHeader>
            <CardTitle>Tables and Views</CardTitle>
            <CardDescription>
              {filteredNodes.length} items found
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Schema</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Columns</TableHead>
                  <TableHead>RLS</TableHead>
                  <TableHead>Relationships</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredNodes.map((node) => {
                  const fullName = getFullName(node)
                  const relationships = data?.edges?.filter(
                    (r) =>
                      (r.source_schema === node.schema &&
                        r.source_table === node.name) ||
                      (r.target_schema === node.schema &&
                        r.target_table === node.name)
                  )
                  return (
                    <TableRow
                      key={fullName}
                      className="cursor-pointer"
                      onClick={() => {
                        setSelectedTable(fullName)
                        setViewMode('erd')
                      }}
                    >
                      <TableCell className="font-medium">{node.name}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{node.schema}</Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant="default">table</Badge>
                      </TableCell>
                      <TableCell>{node.columns.length}</TableCell>
                      <TableCell>
                        {node.rls_enabled ? (
                          <Shield className="h-4 w-4 text-green-500" />
                        ) : (
                          <ShieldOff className="h-4 w-4 text-muted-foreground" />
                        )}
                      </TableCell>
                      <TableCell>
                        {relationships?.length ? (
                          <Badge variant="outline">{relationships.length}</Badge>
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

// ERD Canvas Component - Simple CSS-based visualization
interface ERDCanvasProps {
  nodes: SchemaNode[]
  relationships: SchemaRelationship[]
  zoom: number
  selectedTable: string | null
  onSelectTable: (table: string | null) => void
}

function ERDCanvas({
  nodes,
  relationships,
  zoom,
  selectedTable,
  onSelectTable,
}: ERDCanvasProps) {
  // Helper to get full name for a node
  const getNodeFullName = (node: SchemaNode) => `${node.schema}.${node.name}`

  // Calculate node positions in a grid layout
  const nodePositions = useMemo(() => {
    const positions: Record<string, { x: number; y: number }> = {}
    const cols = Math.ceil(Math.sqrt(nodes.length))
    const nodeWidth = 280
    const nodeHeight = 200
    const gap = 80

    nodes.forEach((node, index) => {
      const col = index % cols
      const row = Math.floor(index / cols)
      positions[getNodeFullName(node)] = {
        x: col * (nodeWidth + gap) + 40,
        y: row * (nodeHeight + gap) + 40,
      }
    })

    return positions
  }, [nodes])

  // Calculate SVG dimensions
  const svgDimensions = useMemo(() => {
    const positions = Object.values(nodePositions)
    if (!positions.length) return { width: 800, height: 600 }
    const maxX = Math.max(...positions.map((p) => p.x)) + 320
    const maxY = Math.max(...positions.map((p) => p.y)) + 240
    return { width: maxX, height: maxY }
  }, [nodePositions])

  // Draw relationship lines
  const renderRelationships = useCallback(() => {
    return relationships.map((rel) => {
      const sourcePos = nodePositions[`${rel.source_schema}.${rel.source_table}`]
      const targetPos = nodePositions[`${rel.target_schema}.${rel.target_table}`]
      if (!sourcePos || !targetPos) return null

      const sourceX = sourcePos.x + 280
      const sourceY = sourcePos.y + 100
      const targetX = targetPos.x
      const targetY = targetPos.y + 100

      // Create a curved path
      const midX = (sourceX + targetX) / 2
      const path = `M ${sourceX} ${sourceY} C ${midX} ${sourceY}, ${midX} ${targetY}, ${targetX} ${targetY}`

      const isHighlighted =
        selectedTable === `${rel.source_schema}.${rel.source_table}` ||
        selectedTable === `${rel.target_schema}.${rel.target_table}`

      return (
        <g key={rel.id}>
          <path
            d={path}
            fill="none"
            stroke={isHighlighted ? 'hsl(var(--primary))' : 'hsl(var(--muted-foreground))'}
            strokeWidth={isHighlighted ? 2 : 1}
            strokeDasharray={isHighlighted ? undefined : '4,4'}
            opacity={isHighlighted ? 1 : 0.5}
          />
          {/* Arrow marker */}
          <circle
            cx={targetX}
            cy={targetY}
            r={4}
            fill={isHighlighted ? 'hsl(var(--primary))' : 'hsl(var(--muted-foreground))'}
          />
        </g>
      )
    })
  }, [relationships, nodePositions, selectedTable])

  if (!nodes.length) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        No tables found
      </div>
    )
  }

  return (
    <div
      className="relative min-h-[500px]"
      style={{
        transform: `scale(${zoom})`,
        transformOrigin: 'top left',
        width: svgDimensions.width,
        height: svgDimensions.height,
      }}
    >
      {/* SVG for relationship lines */}
      <svg
        className="absolute inset-0 pointer-events-none"
        width={svgDimensions.width}
        height={svgDimensions.height}
      >
        {renderRelationships()}
      </svg>

      {/* Table nodes */}
      {nodes.map((node) => {
        const fullName = getNodeFullName(node)
        const pos = nodePositions[fullName]
        if (!pos) return null

        const isSelected = selectedTable === fullName

        return (
          <div
            key={fullName}
            className={cn(
              'absolute w-[280px] bg-card border rounded-lg shadow-sm cursor-pointer transition-all',
              isSelected && 'ring-2 ring-primary shadow-lg',
              !isSelected && 'hover:shadow-md hover:border-primary/50'
            )}
            style={{ left: pos.x, top: pos.y }}
            onClick={() => onSelectTable(isSelected ? null : fullName)}
          >
            {/* Header */}
            <div className="px-3 py-2 border-b flex items-center justify-between rounded-t-lg bg-blue-500/10">
              <div className="flex items-center gap-2">
                <Database className="h-4 w-4" />
                <span className="font-medium text-sm truncate">{node.name}</span>
              </div>
              <div className="flex items-center gap-1">
                {node.rls_enabled && (
                  <TooltipProvider>
                    <Tooltip>
                      <TooltipTrigger>
                        <Shield className="h-3 w-3 text-green-500" />
                      </TooltipTrigger>
                      <TooltipContent>RLS Enabled</TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                )}
                <Badge variant="outline" className="text-xs">
                  {node.type}
                </Badge>
              </div>
            </div>

            {/* Columns */}
            <div className="px-2 py-1 max-h-[160px] overflow-auto">
              {node.columns.slice(0, 8).map((col) => (
                <div
                  key={col.name}
                  className="flex items-center justify-between text-xs py-1 px-1 hover:bg-muted rounded"
                >
                  <div className="flex items-center gap-1.5 truncate">
                    {col.is_primary_key && (
                      <Key className="h-3 w-3 text-yellow-500 shrink-0" />
                    )}
                    {col.is_foreign_key && (
                      <LinkIcon className="h-3 w-3 text-blue-500 shrink-0" />
                    )}
                    <span
                      className={cn(
                        'truncate',
                        col.is_primary_key && 'font-medium'
                      )}
                    >
                      {col.name}
                    </span>
                  </div>
                  <span className="text-muted-foreground ml-2 truncate">
                    {col.data_type}
                  </span>
                </div>
              ))}
              {node.columns.length > 8 && (
                <div className="text-xs text-muted-foreground py-1 px-1">
                  +{node.columns.length - 8} more columns
                </div>
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}
