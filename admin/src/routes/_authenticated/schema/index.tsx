import { useState, useCallback, useMemo } from "react";
import { z } from "zod";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, getRouteApi } from "@tanstack/react-router";
import {
  GitFork,
  Loader2,
  AlertCircle,
  Search,
  LayoutGrid,
  List,
  ZoomIn,
  ZoomOut,
  Maximize2,
} from "lucide-react";
import {
  databaseApi,
  schemaApi,
  policyApi,
  type SchemaNode,
  type SecurityWarning,
} from "@/lib/api";
import { BranchSelector } from "@/components/branch-selector";
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  ERDCanvas,
  TableDetailsPanel,
  ListView,
  type ViewMode,
  type WarningHelpers,
} from "@/components/schema";

const schemaSearchSchema = z.object({
  schema: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/schema/")({
  validateSearch: schemaSearchSchema,
  component: SchemaViewerPage,
});

const routeApi = getRouteApi("/_authenticated/schema/");

function SchemaViewerPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();

  const [viewMode, setViewMode] = useState<ViewMode>("erd");
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedTable, setSelectedTable] = useState<string | null>(null);
  const [zoom, setZoom] = useState(1);

  const { data: availableSchemas = ["public"], isLoading: schemasLoading } =
    useQuery({
      queryKey: ["available-schemas"],
      queryFn: databaseApi.getSchemas,
      staleTime: 5 * 60 * 1000,
    });

  const selectedSchema = search.schema || "public";

  const {
    data,
    isLoading: graphLoading,
    error,
  } = useQuery({
    queryKey: ["schema-graph", selectedSchema],
    queryFn: () => schemaApi.getGraph([selectedSchema]),
    enabled: !schemasLoading,
  });

  const { data: warningsData } = useQuery({
    queryKey: ["security-warnings"],
    queryFn: () => policyApi.getSecurityWarnings(),
    staleTime: 60000,
  });

  const warningsByTable = useMemo(() => {
    if (!warningsData?.warnings) return new Map<string, SecurityWarning[]>();

    const grouped = new Map<string, SecurityWarning[]>();
    for (const warning of warningsData.warnings) {
      const key = `${warning.schema}.${warning.table}`;
      const existing = grouped.get(key) || [];
      existing.push(warning);
      grouped.set(key, existing);
    }
    return grouped;
  }, [warningsData]);

  const getTableWarningCount = useCallback(
    (schema: string, table: string): number => {
      return warningsByTable.get(`${schema}.${table}`)?.length || 0;
    },
    [warningsByTable],
  );

  const getTableWarningSeverity = useCallback(
    (
      schema: string,
      table: string,
    ): "critical" | "high" | "medium" | "low" | null => {
      const warnings = warningsByTable.get(`${schema}.${table}`);
      if (!warnings?.length) return null;

      const severityOrder = ["critical", "high", "medium", "low"] as const;
      for (const severity of severityOrder) {
        if (warnings.some((w) => w.severity === severity)) return severity;
      }
      return "low";
    },
    [warningsByTable],
  );

  const getTableWarnings = useCallback(
    (schema: string, table: string): SecurityWarning[] => {
      return warningsByTable.get(`${schema}.${table}`) || [];
    },
    [warningsByTable],
  );

  const warningHelpers: WarningHelpers = {
    getTableWarningCount,
    getTableWarningSeverity,
    getTableWarnings,
  };

  const isLoading = schemasLoading || graphLoading;

  const getFullName = (node: SchemaNode) => `${node.schema}.${node.name}`;

  const filteredNodes = useMemo(() => {
    if (!data?.nodes) return [];
    if (!searchQuery) return data.nodes;
    const query = searchQuery.toLowerCase();
    return data.nodes.filter(
      (n) =>
        n.name.toLowerCase().includes(query) ||
        getFullName(n).toLowerCase().includes(query) ||
        n.columns.some((c) => c.name.toLowerCase().includes(query)),
    );
  }, [data, searchQuery]);

  const filteredRelationships = useMemo(() => {
    if (!data?.edges || !filteredNodes.length) return [];
    const nodeNames = new Set(filteredNodes.map((n) => getFullName(n)));
    return data.edges.filter(
      (r) =>
        nodeNames.has(`${r.source_schema}.${r.source_table}`) &&
        nodeNames.has(`${r.target_schema}.${r.target_table}`),
    );
  }, [data, filteredNodes]);

  const selectedTableData = useMemo(() => {
    if (!selectedTable || !data?.nodes) return null;
    const node = data.nodes.find((n) => getFullName(n) === selectedTable);
    if (!node) return null;

    const seen = new Set<string>();
    const uniqueColumns = node.columns.filter((col) => {
      if (seen.has(col.name)) return false;
      seen.add(col.name);
      return true;
    });

    return { ...node, columns: uniqueColumns };
  }, [selectedTable, data]);

  const selectedTableRelationships = useMemo(() => {
    if (!selectedTable || !data?.edges) return { incoming: [], outgoing: [] };
    const [schema, table] = selectedTable.split(".");
    return {
      incoming: data.edges.filter(
        (r) => r.target_schema === schema && r.target_table === table,
      ),
      outgoing: data.edges.filter(
        (r) => r.source_schema === schema && r.source_table === table,
      ),
    };
  }, [selectedTable, data]);

  const handleSchemaChange = (schema: string) => {
    navigate({ search: { schema } });
  };

  const handleSelectTableFromList = (fullName: string) => {
    setSelectedTable(fullName);
    setViewMode("erd");
  };

  if (error) {
    return (
      <div className="flex flex-1 flex-col gap-6 p-6">
        <div className="text-destructive flex items-center gap-2">
          <AlertCircle className="h-5 w-5" />
          <span>Failed to load schema graph</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        icon={<GitFork />}
        title="Schema Viewer"
        description="Visualize database tables and their relationships"
        actions={
          <>
            <BranchSelector />
            <Button
              variant={viewMode === "erd" ? "default" : "outline"}
              size="sm"
              onClick={() => setViewMode("erd")}
            >
              <LayoutGrid className="mr-2 h-4 w-4" />
              ERD View
            </Button>
            <Button
              variant={viewMode === "list" ? "default" : "outline"}
              size="sm"
              onClick={() => setViewMode("list")}
            >
              <List className="mr-2 h-4 w-4" />
              List View
            </Button>
          </>
        }
      />

      <div className="flex-1 overflow-auto p-6">
        <div className="flex items-center gap-4">
          <div className="relative max-w-sm flex-1">
            <Search className="text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
            <Input
              placeholder="Search tables, columns..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>
          <Select
            value={selectedSchema}
            onValueChange={handleSchemaChange}
            disabled={schemasLoading}
          >
            <SelectTrigger className="w-[180px]">
              <SelectValue
                placeholder={schemasLoading ? "Loading..." : "Select schema"}
              />
            </SelectTrigger>
            <SelectContent>
              {availableSchemas.map((schema) => (
                <SelectItem key={schema} value={schema}>
                  {schema}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {viewMode === "erd" && (
            <div className="flex items-center gap-1">
              <Button
                variant="outline"
                size="icon"
                onClick={() => setZoom((z) => Math.max(0.25, z - 0.25))}
              >
                <ZoomOut className="h-4 w-4" />
              </Button>
              <span className="text-muted-foreground w-12 text-center text-sm">
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
            <Loader2 className="text-muted-foreground h-8 w-8 animate-spin" />
          </div>
        ) : viewMode === "erd" ? (
          <div
            className="flex flex-1 gap-6"
            style={{ minHeight: "calc(100vh - 16rem)" }}
          >
            <div
              className="bg-muted/20 flex-1 overflow-auto rounded-lg border"
              style={{ minWidth: 0 }}
            >
              <ERDCanvas
                nodes={filteredNodes}
                relationships={filteredRelationships}
                zoom={zoom}
                onZoomChange={setZoom}
                selectedTable={selectedTable}
                onSelectTable={setSelectedTable}
                {...warningHelpers}
              />
            </div>

            {selectedTable && selectedTableData && (
              <TableDetailsPanel
                selectedTableData={selectedTableData}
                selectedTableRelationships={selectedTableRelationships}
                onSelectTable={setSelectedTable}
                {...warningHelpers}
              />
            )}
          </div>
        ) : (
          <ListView
            nodes={filteredNodes}
            relationships={filteredRelationships}
            onSelectTable={handleSelectTableFromList}
            {...warningHelpers}
          />
        )}
      </div>
    </div>
  );
}
