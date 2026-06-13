import { useState, useRef, useEffect } from "react";
import { createFileRoute } from "@tanstack/react-router";
import Editor from "@monaco-editor/react";
import {
  Database,
  Play,
  Trash2,
  Braces,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import type { editor, IDisposable } from "monaco-editor";
import { Panel, Group, Separator } from "react-resizable-panels";
import { toast } from "sonner";
import { useImpersonationStore } from "@/stores/impersonation-store";
import { BranchSelector } from "@/components/branch-selector";
import { PageHeader } from "@/components/layout/page-header";
import api from "@/lib/api";
import { useTheme } from "@/context/theme-provider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useSchemaMetadata } from "@/features/sql-editor/hooks/use-schema-metadata";
import { createGraphQLCompletionProvider } from "@/features/sql-editor/utils/graphql-completion-provider";
import { createSqlCompletionProvider } from "@/features/sql-editor/utils/sql-completion-provider";
import {
  HistoryItem,
  SQLResultView,
  GraphQLResultView,
  type EditorMode,
  type SQLResult,
  type SQLExecutionResponse,
  type GraphQLResponse,
  type QueryHistory,
} from "@/components/sql-editor";

export const Route = createFileRoute("/_authenticated/sql-editor/")({
  component: SQLEditorPage,
});

const ROWS_PER_PAGE = 100;

const DEFAULT_SQL =
  "-- Write your SQL query here\nSELECT * FROM auth.users LIMIT 10;";
const DEFAULT_GRAPHQL = `# Write your GraphQL query here
# GraphQL exposes tables from the 'public' schema
# Use the _health query to test connectivity
query {
  _health
}`;

function SQLEditorPage() {
  const { resolvedTheme } = useTheme();
  const [editorMode, setEditorMode] = useState<EditorMode>("sql");
  const [sqlQuery, setSqlQuery] = useState(DEFAULT_SQL);
  const [graphqlQuery, setGraphqlQuery] = useState(DEFAULT_GRAPHQL);
  const query = editorMode === "sql" ? sqlQuery : graphqlQuery;
  const setQuery = editorMode === "sql" ? setSqlQuery : setGraphqlQuery;
  const [isExecuting, setIsExecuting] = useState(false);
  const [queryHistory, setQueryHistory] = useState<QueryHistory[]>([]);
  const [selectedHistoryId, setSelectedHistoryId] = useState<string | null>(
    null,
  );
  const [historyOpen, setHistoryOpen] = useState(false);
  const [currentPages, setCurrentPages] = useState<Record<string, number>>({});
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);
  const monacoRef = useRef<typeof import("monaco-editor") | null>(null);
  const sqlCompletionProviderRef = useRef<IDisposable | null>(null);
  const graphqlCompletionProviderRef = useRef<IDisposable | null>(null);
  const executeQueryRef = useRef<() => void>(() => {});

  const { schemas, tables } = useSchemaMetadata();

  useEffect(() => {
    if (monacoRef.current && (schemas.length > 0 || tables.length > 0)) {
      if (sqlCompletionProviderRef.current) {
        sqlCompletionProviderRef.current.dispose();
      }
      sqlCompletionProviderRef.current =
        monacoRef.current.languages.registerCompletionItemProvider(
          "sql",
          createSqlCompletionProvider(monacoRef.current, { schemas, tables }),
        );
    }
    return () => {
      if (sqlCompletionProviderRef.current) {
        sqlCompletionProviderRef.current.dispose();
      }
    };
  }, [schemas, tables]);

  useEffect(() => {
    if (monacoRef.current && tables.length > 0) {
      if (graphqlCompletionProviderRef.current) {
        graphqlCompletionProviderRef.current.dispose();
      }
      graphqlCompletionProviderRef.current =
        monacoRef.current.languages.registerCompletionItemProvider(
          "graphql",
          createGraphQLCompletionProvider(monacoRef.current, { tables }),
        );
    }
    return () => {
      if (graphqlCompletionProviderRef.current) {
        graphqlCompletionProviderRef.current.dispose();
      }
    };
  }, [tables]);

  useEffect(() => {
    if (monacoRef.current) {
      monacoRef.current.editor.setTheme(
        resolvedTheme === "dark" ? "fluxbase-dark" : "fluxbase-light",
      );
    }
  }, [resolvedTheme]);

  const currentHistory = selectedHistoryId
    ? queryHistory.find((h) => h.id === selectedHistoryId)
    : queryHistory[0];

  const executeQuery = async () => {
    const currentQuery = editorRef.current?.getValue() || query;

    if (!currentQuery.trim()) {
      toast.error(
        `Please enter a ${editorMode === "sql" ? "SQL" : "GraphQL"} query`,
      );
      return;
    }

    setQuery(currentQuery);
    setIsExecuting(true);
    const startTime = performance.now();

    try {
      const {
        isImpersonating: isImpersonatingNow,
        impersonationToken: tokenNow,
      } = useImpersonationStore.getState();
      const config: { headers?: Record<string, string> } = {};
      if (isImpersonatingNow && tokenNow) {
        config.headers = {
          "X-Impersonation-Token": tokenNow,
        };
      }

      if (editorMode === "sql") {
        const response = await api.post<SQLExecutionResponse>(
          "/api/v1/admin/sql/execute",
          { query: currentQuery },
          config,
        );

        const executionTime = performance.now() - startTime;

        const historyItem: QueryHistory = {
          id: Date.now().toString(),
          timestamp: new Date(),
          mode: "sql",
          results: response.data.results,
          query: currentQuery,
          executionTime,
        };
        setQueryHistory((prev) => [historyItem, ...prev.slice(0, 9)]);
        setSelectedHistoryId(historyItem.id);
        setHistoryOpen(false);

        const pages: Record<string, number> = {};
        response.data.results.forEach((_, idx) => {
          pages[`${historyItem.id}-${idx}`] = 1;
        });
        setCurrentPages((prev) => ({ ...prev, ...pages }));

        const hasErrors = response.data.results.some((r) => r.error);
        if (hasErrors) {
          toast.warning("Query executed with errors");
        } else {
          toast.success("Query executed successfully");
        }
      } else {
        const response = await api.post<GraphQLResponse>(
          "/api/v1/graphql",
          { query: currentQuery },
          config,
        );

        const executionTime = performance.now() - startTime;

        const historyItem: QueryHistory = {
          id: Date.now().toString(),
          timestamp: new Date(),
          mode: "graphql",
          graphqlResponse: response.data,
          query: currentQuery,
          executionTime,
        };
        setQueryHistory((prev) => [historyItem, ...prev.slice(0, 9)]);
        setSelectedHistoryId(historyItem.id);
        setHistoryOpen(false);

        if (response.data.errors && response.data.errors.length > 0) {
          toast.warning("GraphQL query returned errors");
        } else {
          toast.success("GraphQL query executed successfully");
        }
      }
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error && "response" in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error
          : undefined;
      toast.error(
        errorMessage ||
          `Failed to execute ${editorMode === "sql" ? "SQL" : "GraphQL"} query`,
      );
    } finally {
      setIsExecuting(false);
    }
  };

  executeQueryRef.current = executeQuery;

  const clearHistory = () => {
    setQueryHistory([]);
    setSelectedHistoryId(null);
    setCurrentPages({});
    toast.success("Query history cleared");
  };

  const removeHistoryItem = (id: string) => {
    setQueryHistory((prev) => prev.filter((h) => h.id !== id));
    if (selectedHistoryId === id) {
      setSelectedHistoryId(queryHistory[0]?.id || null);
    }
  };

  const exportAsCSV = (result: SQLResult) => {
    if (!result.rows || result.rows.length === 0) {
      toast.error("No data to export");
      return;
    }

    const csv = [
      result.columns!.join(","),
      ...result.rows.map((row) =>
        result
          .columns!.map((col) => {
            const value = row[col];
            return typeof value === "string" && value.includes(",")
              ? `"${value}"`
              : value;
          })
          .join(","),
      ),
    ].join("\n");

    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `query-result-${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);

    toast.success("Exported as CSV");
  };

  const exportAsJSON = (result: SQLResult) => {
    if (!result.rows || result.rows.length === 0) {
      toast.error("No data to export");
      return;
    }

    const json = JSON.stringify(result.rows, null, 2);
    const blob = new Blob([json], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `query-result-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);

    toast.success("Exported as JSON");
  };

  const handleEditorDidMount = (
    editor: editor.IStandaloneCodeEditor,
    monaco: typeof import("monaco-editor"),
  ) => {
    editorRef.current = editor;
    monacoRef.current = monaco;

    if (
      !monaco.languages.getLanguages().some((lang) => lang.id === "graphql")
    ) {
      monaco.languages.register({ id: "graphql" });

      monaco.languages.setMonarchTokensProvider("graphql", {
        keywords: [
          "query",
          "mutation",
          "subscription",
          "fragment",
          "on",
          "type",
          "interface",
          "union",
          "enum",
          "input",
          "scalar",
          "directive",
          "extend",
          "schema",
          "implements",
        ],
        operators: ["=", "!", ":", "@", "|", "&", "..."],
        symbols: /[=!:@|&]+/,
        tokenizer: {
          root: [
            [/#.*$/, "comment"],
            [/"([^"\\]|\\.)*$/, "string.invalid"],
            [/"/, { token: "string.quote", bracket: "@open", next: "@string" }],
            [/\$[a-zA-Z_]\w*/, "variable"],
            [/@[a-zA-Z_]\w*/, "annotation"],
            [
              /[a-zA-Z_]\w*/,
              {
                cases: {
                  "@keywords": "keyword",
                  "@default": "identifier",
                },
              },
            ],
            [/[{}()[\]]/, "@brackets"],
            [/[0-9]+/, "number"],
            [/@symbols/, "operator"],
            [/[,:]/, "delimiter"],
          ],
          string: [
            [/[^\\"]+/, "string"],
            [/\\./, "string.escape"],
            [/"/, { token: "string.quote", bracket: "@close", next: "@pop" }],
          ],
        },
      });
    }

    if (schemas.length > 0 || tables.length > 0) {
      sqlCompletionProviderRef.current =
        monaco.languages.registerCompletionItemProvider(
          "sql",
          createSqlCompletionProvider(monaco, { schemas, tables }),
        );
    }

    if (tables.length > 0) {
      graphqlCompletionProviderRef.current =
        monaco.languages.registerCompletionItemProvider(
          "graphql",
          createGraphQLCompletionProvider(monaco, { tables }),
        );
    }

    monaco.editor.defineTheme("fluxbase-dark", {
      base: "vs-dark",
      inherit: true,
      rules: [
        { token: "comment", foreground: "6A9955" },
        { token: "keyword", foreground: "569CD6", fontStyle: "bold" },
        { token: "string", foreground: "CE9178" },
        { token: "number", foreground: "B5CEA8" },
        { token: "operator", foreground: "D4D4D4" },
        { token: "variable", foreground: "9CDCFE" },
        { token: "annotation", foreground: "DCDCAA" },
      ],
      colors: {
        "editor.background": "#09090b",
        "editor.foreground": "#e4e4e7",
        "editor.lineHighlightBackground": "#18181b",
        "editorLineNumber.foreground": "#71717a",
        "editorLineNumber.activeForeground": "#a1a1aa",
        "editor.selectionBackground": "#3f3f46",
        "editorCursor.foreground": "#a1a1aa",
      },
    });

    monaco.editor.defineTheme("fluxbase-light", {
      base: "vs",
      inherit: true,
      rules: [
        { token: "comment", foreground: "008000" },
        { token: "keyword", foreground: "0000FF", fontStyle: "bold" },
        { token: "string", foreground: "A31515" },
        { token: "number", foreground: "098658" },
        { token: "variable", foreground: "001080" },
        { token: "annotation", foreground: "795E26" },
      ],
      colors: {
        "editor.background": "#ffffff",
        "editor.foreground": "#09090b",
        "editor.lineHighlightBackground": "#f4f4f5",
        "editorLineNumber.foreground": "#a1a1aa",
        "editorLineNumber.activeForeground": "#71717a",
        "editor.selectionBackground": "#e4e4e7",
        "editorCursor.foreground": "#09090b",
      },
    });

    monaco.editor.setTheme(
      resolvedTheme === "dark" ? "fluxbase-dark" : "fluxbase-light",
    );

    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
      executeQueryRef.current();
    });
  };

  const handleModeChange = (mode: EditorMode) => {
    setEditorMode(mode);
    if (editorRef.current && monacoRef.current) {
      const model = editorRef.current.getModel();
      if (model) {
        monacoRef.current.editor.setModelLanguage(
          model,
          mode === "sql" ? "sql" : "graphql",
        );
      }
    }
  };

  const getPaginatedRows = (
    rows: Record<string, unknown>[],
    pageKey: string,
  ) => {
    const page = currentPages[pageKey] || 1;
    const start = (page - 1) * ROWS_PER_PAGE;
    const end = start + ROWS_PER_PAGE;
    return rows.slice(start, end);
  };

  const getTotalPages = (rowCount: number) => {
    return Math.ceil(rowCount / ROWS_PER_PAGE);
  };

  const setPage = (pageKey: string, page: number) => {
    setCurrentPages((prev) => ({ ...prev, [pageKey]: page }));
  };

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        icon={editorMode === "sql" ? <Database /> : <Braces />}
        title="Query Editor"
        description={`Execute ${editorMode === "sql" ? "SQL" : "GraphQL"} queries on the database`}
        actions={
          <>
            <Tabs
              value={editorMode}
              onValueChange={(v) => handleModeChange(v as EditorMode)}
            >
              <TabsList>
                <TabsTrigger value="sql" className="gap-2">
                  <Database className="h-4 w-4" />
                  SQL
                </TabsTrigger>
                <TabsTrigger value="graphql" className="gap-2">
                  <Braces className="h-4 w-4" />
                  GraphQL
                </TabsTrigger>
              </TabsList>
            </Tabs>

            <div className="flex items-center gap-2">
              <BranchSelector />
              {queryHistory.length > 0 && (
                <Button variant="outline" size="sm" onClick={clearHistory}>
                  <Trash2 className="mr-2 h-4 w-4" />
                  Clear History
                </Button>
              )}
              <Button size="sm" onClick={executeQuery} disabled={isExecuting}>
                <Play className="mr-2 h-4 w-4" />
                {isExecuting ? "Executing..." : "Execute (Ctrl+Enter)"}
              </Button>
            </div>
          </>
        }
      />

      <div className="flex flex-1 overflow-hidden p-6">
        <Group orientation="vertical" id="sql-editor-group-v2">
          <Panel id="query-editor" defaultSize="35" minSize="20" maxSize="80">
            <Card className="h-full overflow-hidden">
              <Editor
                height="100%"
                language={editorMode === "sql" ? "sql" : "graphql"}
                value={query}
                onChange={(value) => setQuery(value || "")}
                theme={
                  resolvedTheme === "dark" ? "fluxbase-dark" : "fluxbase-light"
                }
                onMount={handleEditorDidMount}
                options={{
                  minimap: { enabled: true },
                  fontSize: 14,
                  lineNumbers: "on",
                  scrollBeyondLastLine: false,
                  automaticLayout: true,
                  tabSize: 2,
                  quickSuggestions: true,
                  suggestOnTriggerCharacters: true,
                  acceptSuggestionOnCommitCharacter: true,
                  wordBasedSuggestions: "off",
                }}
              />
            </Card>
          </Panel>

          <Separator className="bg-border hover:bg-primary my-2 h-2 cursor-row-resize transition-colors" />

          <Panel id="query-results" defaultSize="65" minSize="20">
            <Card className="flex h-full flex-col overflow-hidden">
              {queryHistory.length === 0 ? (
                <div className="flex h-full items-center justify-center">
                  <div className="flex flex-col items-center gap-2 text-center">
                    <Database className="text-muted-foreground h-12 w-12" />
                    <p className="text-muted-foreground text-sm">
                      No queries executed yet
                    </p>
                    <p className="text-muted-foreground text-xs">
                      Write a query and press Execute or Ctrl+Enter
                    </p>
                  </div>
                </div>
              ) : (
                <div className="flex h-full flex-col">
                  <Collapsible open={historyOpen} onOpenChange={setHistoryOpen}>
                    <div className="flex items-center justify-between border-b px-4 py-2">
                      <CollapsibleTrigger asChild>
                        <Button variant="ghost" size="sm" className="gap-2">
                          {historyOpen ? (
                            <ChevronDown className="h-4 w-4" />
                          ) : (
                            <ChevronRight className="h-4 w-4" />
                          )}
                          Query History ({queryHistory.length})
                        </Button>
                      </CollapsibleTrigger>
                      {currentHistory && (
                        <div className="text-muted-foreground flex items-center gap-2 text-xs">
                          {currentHistory.timestamp.toLocaleString()}
                        </div>
                      )}
                    </div>
                    <CollapsibleContent>
                      <ScrollArea className="max-h-48 border-b">
                        <div className="space-y-1 p-2">
                          {queryHistory.map((history) => (
                            <HistoryItem
                              key={history.id}
                              history={history}
                              isSelected={selectedHistoryId === history.id}
                              onSelect={() => {
                                setSelectedHistoryId(history.id);
                                setHistoryOpen(false);
                              }}
                              onRemove={() => removeHistoryItem(history.id)}
                            />
                          ))}
                        </div>
                      </ScrollArea>
                    </CollapsibleContent>
                  </Collapsible>

                  {currentHistory && (
                    <div className="flex-1 overflow-auto">
                      <div className="space-y-4 p-4">
                        {currentHistory.mode === "sql" &&
                          currentHistory.results?.map((result, idx) => {
                            const pageKey = `${currentHistory.id}-${idx}`;
                            const currentPage = currentPages[pageKey] || 1;
                            const totalPages = result.rows
                              ? getTotalPages(result.rows.length)
                              : 0;
                            const paginatedRows = result.rows
                              ? getPaginatedRows(result.rows, pageKey)
                              : [];

                            return (
                              <SQLResultView
                                key={idx}
                                result={result}
                                currentPage={currentPage}
                                totalPages={totalPages}
                                paginatedRows={paginatedRows}
                                onExportCSV={() => exportAsCSV(result)}
                                onExportJSON={() => exportAsJSON(result)}
                                onPrevPage={() =>
                                  setPage(pageKey, currentPage - 1)
                                }
                                onNextPage={() =>
                                  setPage(pageKey, currentPage + 1)
                                }
                              />
                            );
                          })}

                        {currentHistory.mode === "graphql" &&
                          currentHistory.graphqlResponse && (
                            <GraphQLResultView
                              response={currentHistory.graphqlResponse}
                              executionTime={currentHistory.executionTime}
                            />
                          )}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </Card>
          </Panel>
        </Group>
      </div>
    </div>
  );
}
