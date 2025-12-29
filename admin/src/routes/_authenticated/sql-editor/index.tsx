import { createFileRoute } from '@tanstack/react-router'
import { useState, useRef, useEffect } from 'react'
import Editor from '@monaco-editor/react'
import type { editor, IDisposable } from 'monaco-editor'
import { Panel, PanelGroup, PanelResizeHandle } from 'react-resizable-panels'
import { useSchemaMetadata } from '@/features/sql-editor/hooks/use-schema-metadata'
import { createSqlCompletionProvider } from '@/features/sql-editor/utils/sql-completion-provider'
import { createGraphQLCompletionProvider } from '@/features/sql-editor/utils/graphql-completion-provider'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Database,
  Play,
  Trash2,
  Download,
  AlertCircle,
  CheckCircle,
  Clock,
  ChevronDown,
  ChevronRight,
  ChevronLeft,
  ChevronRight as ChevronRightIcon,
  X,
  Braces,
} from 'lucide-react'
import { toast } from 'sonner'
import api from '@/lib/api'
import { useTheme } from '@/context/theme-provider'
import { useImpersonationStore } from '@/stores/impersonation-store'
import { syncAuthToken } from '@/lib/fluxbase-client'
import { ImpersonationPopover } from '@/features/impersonation/components/impersonation-popover'

export const Route = createFileRoute('/_authenticated/sql-editor/')({
  component: SQLEditorPage,
})

type EditorMode = 'sql' | 'graphql'

interface SQLResult {
  columns?: string[]
  rows?: Record<string, unknown>[]
  row_count: number
  affected_rows?: number
  execution_time_ms: number
  error?: string
  statement: string
}

interface SQLExecutionResponse {
  results: SQLResult[]
}

interface GraphQLResponse {
  data?: unknown
  errors?: Array<{
    message: string
    locations?: Array<{ line: number; column: number }>
    path?: (string | number)[]
  }>
}

interface QueryHistory {
  id: string
  timestamp: Date
  mode: EditorMode
  results?: SQLResult[]
  graphqlResponse?: GraphQLResponse
  query: string
  executionTime?: number
}

const ROWS_PER_PAGE = 100

const DEFAULT_SQL = '-- Write your SQL query here\nSELECT * FROM auth.users LIMIT 10;'
const DEFAULT_GRAPHQL = `# Write your GraphQL query here
query {
  users(limit: 10) {
    id
    email
    createdAt
  }
}`

function SQLEditorPage() {
  const { resolvedTheme } = useTheme()
  const [editorMode, setEditorMode] = useState<EditorMode>('sql')
  const [sqlQuery, setSqlQuery] = useState(DEFAULT_SQL)
  const [graphqlQuery, setGraphqlQuery] = useState(DEFAULT_GRAPHQL)
  const query = editorMode === 'sql' ? sqlQuery : graphqlQuery
  const setQuery = editorMode === 'sql' ? setSqlQuery : setGraphqlQuery
  const [isExecuting, setIsExecuting] = useState(false)
  const [queryHistory, setQueryHistory] = useState<QueryHistory[]>([])
  const [selectedHistoryId, setSelectedHistoryId] = useState<string | null>(null)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [currentPages, setCurrentPages] = useState<Record<string, number>>({})
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null)
  const monacoRef = useRef<typeof import('monaco-editor') | null>(null)
  const sqlCompletionProviderRef = useRef<IDisposable | null>(null)
  const graphqlCompletionProviderRef = useRef<IDisposable | null>(null)

  // Fetch schema metadata for autocompletion
  const { schemas, tables } = useSchemaMetadata()

  // Update SQL completion provider when metadata changes
  useEffect(() => {
    if (monacoRef.current && (schemas.length > 0 || tables.length > 0)) {
      // Dispose old SQL provider
      if (sqlCompletionProviderRef.current) {
        sqlCompletionProviderRef.current.dispose()
      }

      // Register SQL provider with updated metadata
      sqlCompletionProviderRef.current = monacoRef.current.languages.registerCompletionItemProvider(
        'sql',
        createSqlCompletionProvider(monacoRef.current, { schemas, tables })
      )
    }

    return () => {
      if (sqlCompletionProviderRef.current) {
        sqlCompletionProviderRef.current.dispose()
      }
    }
  }, [schemas, tables])

  // Update GraphQL completion provider when metadata changes
  useEffect(() => {
    if (monacoRef.current && tables.length > 0) {
      // Dispose old GraphQL provider
      if (graphqlCompletionProviderRef.current) {
        graphqlCompletionProviderRef.current.dispose()
      }

      // Register GraphQL provider with updated metadata
      graphqlCompletionProviderRef.current = monacoRef.current.languages.registerCompletionItemProvider(
        'graphql',
        createGraphQLCompletionProvider(monacoRef.current, { tables })
      )
    }

    return () => {
      if (graphqlCompletionProviderRef.current) {
        graphqlCompletionProviderRef.current.dispose()
      }
    }
  }, [tables])

  // Update Monaco theme when app theme changes
  useEffect(() => {
    if (monacoRef.current) {
      monacoRef.current.editor.setTheme(
        resolvedTheme === 'dark' ? 'fluxbase-dark' : 'fluxbase-light'
      )
    }
  }, [resolvedTheme])

  // Get current history item (most recent or selected)
  const currentHistory = selectedHistoryId
    ? queryHistory.find((h) => h.id === selectedHistoryId)
    : queryHistory[0]

  // Execute SQL or GraphQL query
  const executeQuery = async () => {
    // Get current value from editor if available, otherwise use state
    const currentQuery = editorRef.current?.getValue() || query

    if (!currentQuery.trim()) {
      toast.error(`Please enter a ${editorMode === 'sql' ? 'SQL' : 'GraphQL'} query`)
      return
    }

    // Update state to match editor
    setQuery(currentQuery)

    setIsExecuting(true)
    const startTime = performance.now()

    try {
      // Build request config with optional impersonation context
      const { isImpersonating: isImpersonatingNow, impersonationToken: tokenNow } =
        useImpersonationStore.getState()
      const config: { headers?: Record<string, string> } = {}
      if (isImpersonatingNow && tokenNow) {
        config.headers = {
          'X-Impersonation-Token': tokenNow,
        }
      }

      if (editorMode === 'sql') {
        // Execute SQL query
        const response = await api.post<SQLExecutionResponse>(
          '/api/v1/admin/sql/execute',
          { query: currentQuery },
          config
        )

        const executionTime = performance.now() - startTime

        // Add to history
        const historyItem: QueryHistory = {
          id: Date.now().toString(),
          timestamp: new Date(),
          mode: 'sql',
          results: response.data.results,
          query: currentQuery,
          executionTime,
        }
        setQueryHistory((prev) => [historyItem, ...prev.slice(0, 9)])
        setSelectedHistoryId(historyItem.id)
        setHistoryOpen(false)

        // Initialize pagination for each result
        const pages: Record<string, number> = {}
        response.data.results.forEach((_, idx) => {
          pages[`${historyItem.id}-${idx}`] = 1
        })
        setCurrentPages((prev) => ({ ...prev, ...pages }))

        // Show success toast
        const hasErrors = response.data.results.some((r) => r.error)
        if (hasErrors) {
          toast.warning('Query executed with errors')
        } else {
          toast.success('Query executed successfully')
        }
      } else {
        // Execute GraphQL query
        const response = await api.post<GraphQLResponse>(
          '/api/v1/graphql',
          { query: currentQuery },
          config
        )

        const executionTime = performance.now() - startTime

        // Add to history
        const historyItem: QueryHistory = {
          id: Date.now().toString(),
          timestamp: new Date(),
          mode: 'graphql',
          graphqlResponse: response.data,
          query: currentQuery,
          executionTime,
        }
        setQueryHistory((prev) => [historyItem, ...prev.slice(0, 9)])
        setSelectedHistoryId(historyItem.id)
        setHistoryOpen(false)

        // Show success/error toast
        if (response.data.errors && response.data.errors.length > 0) {
          toast.warning('GraphQL query returned errors')
        } else {
          toast.success('GraphQL query executed successfully')
        }
      }
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response?.data
              ?.error
          : undefined
      toast.error(errorMessage || `Failed to execute ${editorMode === 'sql' ? 'SQL' : 'GraphQL'} query`)
    } finally {
      setIsExecuting(false)
    }
  }

  // Clear all history
  const clearHistory = () => {
    setQueryHistory([])
    setSelectedHistoryId(null)
    setCurrentPages({})
    toast.success('Query history cleared')
  }

  // Remove single history item
  const removeHistoryItem = (id: string) => {
    setQueryHistory((prev) => prev.filter((h) => h.id !== id))
    if (selectedHistoryId === id) {
      setSelectedHistoryId(queryHistory[0]?.id || null)
    }
  }

  // Export result as CSV
  const exportAsCSV = (result: SQLResult) => {
    if (!result.rows || result.rows.length === 0) {
      toast.error('No data to export')
      return
    }

    const csv = [
      result.columns!.join(','),
      ...result.rows.map((row) =>
        result.columns!.map((col) => {
          const value = row[col]
          return typeof value === 'string' && value.includes(',')
            ? `"${value}"`
            : value
        }).join(',')
      ),
    ].join('\n')

    const blob = new Blob([csv], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `query-result-${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)

    toast.success('Exported as CSV')
  }

  // Export result as JSON
  const exportAsJSON = (result: SQLResult) => {
    if (!result.rows || result.rows.length === 0) {
      toast.error('No data to export')
      return
    }

    const json = JSON.stringify(result.rows, null, 2)
    const blob = new Blob([json], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `query-result-${Date.now()}.json`
    a.click()
    URL.revokeObjectURL(url)

    toast.success('Exported as JSON')
  }

  // Handle editor mount - register keyboard shortcut and completion providers
  const handleEditorDidMount = (
    editor: editor.IStandaloneCodeEditor,
    monaco: typeof import('monaco-editor')
  ) => {
    editorRef.current = editor
    monacoRef.current = monaco

    // Register GraphQL language if not already registered
    if (!monaco.languages.getLanguages().some(lang => lang.id === 'graphql')) {
      monaco.languages.register({ id: 'graphql' })

      // GraphQL syntax highlighting
      monaco.languages.setMonarchTokensProvider('graphql', {
        keywords: ['query', 'mutation', 'subscription', 'fragment', 'on', 'type', 'interface', 'union', 'enum', 'input', 'scalar', 'directive', 'extend', 'schema', 'implements'],
        operators: ['=', '!', ':', '@', '|', '&', '...'],
        symbols: /[=!:@|&]+/,
        tokenizer: {
          root: [
            [/#.*$/, 'comment'],
            [/"([^"\\]|\\.)*$/, 'string.invalid'],
            [/"/, { token: 'string.quote', bracket: '@open', next: '@string' }],
            [/\$[a-zA-Z_]\w*/, 'variable'],
            [/@[a-zA-Z_]\w*/, 'annotation'],
            [/[a-zA-Z_]\w*/, {
              cases: {
                '@keywords': 'keyword',
                '@default': 'identifier'
              }
            }],
            [/[{}()[\]]/, '@brackets'],
            [/[0-9]+/, 'number'],
            [/@symbols/, 'operator'],
            [/[,:]/, 'delimiter'],
          ],
          string: [
            [/[^\\"]+/, 'string'],
            [/\\./, 'string.escape'],
            [/"/, { token: 'string.quote', bracket: '@close', next: '@pop' }]
          ],
        }
      })
    }

    // Register initial SQL completion provider if metadata is already loaded
    if (schemas.length > 0 || tables.length > 0) {
      sqlCompletionProviderRef.current = monaco.languages.registerCompletionItemProvider(
        'sql',
        createSqlCompletionProvider(monaco, { schemas, tables })
      )
    }

    // Register initial GraphQL completion provider if metadata is already loaded
    if (tables.length > 0) {
      graphqlCompletionProviderRef.current = monaco.languages.registerCompletionItemProvider(
        'graphql',
        createGraphQLCompletionProvider(monaco, { tables })
      )
    }

    // Define custom theme that matches dashboard
    monaco.editor.defineTheme('fluxbase-dark', {
      base: 'vs-dark',
      inherit: true,
      rules: [
        { token: 'comment', foreground: '6A9955' },
        { token: 'keyword', foreground: '569CD6', fontStyle: 'bold' },
        { token: 'string', foreground: 'CE9178' },
        { token: 'number', foreground: 'B5CEA8' },
        { token: 'operator', foreground: 'D4D4D4' },
        { token: 'variable', foreground: '9CDCFE' },
        { token: 'annotation', foreground: 'DCDCAA' },
      ],
      colors: {
        'editor.background': '#09090b',
        'editor.foreground': '#e4e4e7',
        'editor.lineHighlightBackground': '#18181b',
        'editorLineNumber.foreground': '#71717a',
        'editorLineNumber.activeForeground': '#a1a1aa',
        'editor.selectionBackground': '#3f3f46',
        'editorCursor.foreground': '#a1a1aa',
      }
    })

    monaco.editor.defineTheme('fluxbase-light', {
      base: 'vs',
      inherit: true,
      rules: [
        { token: 'comment', foreground: '008000' },
        { token: 'keyword', foreground: '0000FF', fontStyle: 'bold' },
        { token: 'string', foreground: 'A31515' },
        { token: 'number', foreground: '098658' },
        { token: 'variable', foreground: '001080' },
        { token: 'annotation', foreground: '795E26' },
      ],
      colors: {
        'editor.background': '#ffffff',
        'editor.foreground': '#09090b',
        'editor.lineHighlightBackground': '#f4f4f5',
        'editorLineNumber.foreground': '#a1a1aa',
        'editorLineNumber.activeForeground': '#71717a',
        'editor.selectionBackground': '#e4e4e7',
        'editorCursor.foreground': '#09090b',
      }
    })

    // Set the appropriate theme
    monaco.editor.setTheme(resolvedTheme === 'dark' ? 'fluxbase-dark' : 'fluxbase-light')

    // Register Ctrl/Cmd + Enter to execute query
    editor.addCommand(
      monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter,
      () => {
        executeQuery()
      }
    )
  }

  // Handle mode switch
  const handleModeChange = (mode: EditorMode) => {
    setEditorMode(mode)
    // Update editor language if mounted
    if (editorRef.current && monacoRef.current) {
      const model = editorRef.current.getModel()
      if (model) {
        monacoRef.current.editor.setModelLanguage(model, mode === 'sql' ? 'sql' : 'graphql')
      }
    }
  }

  // Get paginated rows for a result
  const getPaginatedRows = (rows: Record<string, unknown>[], pageKey: string) => {
    const page = currentPages[pageKey] || 1
    const start = (page - 1) * ROWS_PER_PAGE
    const end = start + ROWS_PER_PAGE
    return rows.slice(start, end)
  }

  // Calculate total pages
  const getTotalPages = (rowCount: number) => {
    return Math.ceil(rowCount / ROWS_PER_PAGE)
  }

  // Change page
  const setPage = (pageKey: string, page: number) => {
    setCurrentPages((prev) => ({ ...prev, [pageKey]: page }))
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b bg-background px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            {editorMode === 'sql' ? (
              <Database className="h-5 w-5 text-primary" />
            ) : (
              <Braces className="h-5 w-5 text-primary" />
            )}
          </div>
          <div>
            <h1 className="text-xl font-semibold">Query Editor</h1>
            <p className="text-sm text-muted-foreground">
              Execute {editorMode === 'sql' ? 'SQL' : 'GraphQL'} queries on the database
            </p>
          </div>
        </div>

        {/* Mode Toggle */}
        <Tabs value={editorMode} onValueChange={(v) => handleModeChange(v as EditorMode)}>
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

        {/* Impersonation UI */}
        <ImpersonationPopover
          contextLabel="Executing as"
          defaultReason={`Testing RLS policies in ${editorMode === 'sql' ? 'SQL' : 'GraphQL'} Editor`}
          onImpersonationStart={() => syncAuthToken()}
          onImpersonationStop={() => syncAuthToken()}
        />

        <div className="flex items-center gap-2">
          {queryHistory.length > 0 && (
            <Button variant="outline" size="sm" onClick={clearHistory}>
              <Trash2 className="mr-2 h-4 w-4" />
              Clear History
            </Button>
          )}
          <Button
            size="sm"
            onClick={executeQuery}
            disabled={isExecuting}
          >
            <Play className="mr-2 h-4 w-4" />
            {isExecuting ? 'Executing...' : 'Execute (Ctrl+Enter)'}
          </Button>
        </div>
      </div>

      {/* Editor and Results */}
      <div className="flex flex-1 overflow-hidden p-6">
        <PanelGroup direction="vertical">
          {/* Query Editor */}
          <Panel defaultSize={35} minSize={20}>
            <Card className="h-full overflow-hidden">
              <Editor
                height="100%"
                language={editorMode === 'sql' ? 'sql' : 'graphql'}
                value={query}
                onChange={(value) => setQuery(value || '')}
                theme={resolvedTheme === 'dark' ? 'fluxbase-dark' : 'fluxbase-light'}
                onMount={handleEditorDidMount}
                options={{
                  minimap: { enabled: true },
                  fontSize: 14,
                  lineNumbers: 'on',
                  scrollBeyondLastLine: false,
                  automaticLayout: true,
                  tabSize: 2,
                  quickSuggestions: true,
                  suggestOnTriggerCharacters: true,
                  acceptSuggestionOnCommitCharacter: true,
                  wordBasedSuggestions: 'off',
                }}
              />
            </Card>
          </Panel>

          <PanelResizeHandle className="my-2 h-1 bg-border hover:bg-primary transition-colors" />

          {/* Results */}
          <Panel defaultSize={65} minSize={30}>
            <Card className="h-full overflow-hidden flex flex-col">
              {queryHistory.length === 0 ? (
                <div className="flex h-full items-center justify-center">
                  <div className="flex flex-col items-center gap-2 text-center">
                    <Database className="h-12 w-12 text-muted-foreground" />
                    <p className="text-sm text-muted-foreground">
                      No queries executed yet
                    </p>
                    <p className="text-xs text-muted-foreground">
                      Write a query and press Execute or Ctrl+Enter
                    </p>
                  </div>
                </div>
              ) : (
                <div className="flex flex-col h-full">
                  {/* History Panel */}
                  <Collapsible open={historyOpen} onOpenChange={setHistoryOpen}>
                    <div className="border-b px-4 py-2 flex items-center justify-between">
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
                        <div className="text-xs text-muted-foreground flex items-center gap-2">
                          <Clock className="h-3 w-3" />
                          {currentHistory.timestamp.toLocaleString()}
                        </div>
                      )}
                    </div>
                    <CollapsibleContent>
                      <ScrollArea className="max-h-48 border-b">
                        <div className="p-2 space-y-1">
                          {queryHistory.map((history) => (
                            <div
                              key={history.id}
                              className={`flex items-center justify-between p-2 rounded-md hover:bg-accent cursor-pointer ${
                                selectedHistoryId === history.id ? 'bg-accent' : ''
                              }`}
                              onClick={() => {
                                setSelectedHistoryId(history.id)
                                setHistoryOpen(false)
                              }}
                            >
                              <div className="flex-1 min-w-0">
                                <div className="flex items-center gap-2">
                                  {history.mode === 'sql' ? (
                                    <Database className="h-3 w-3 text-muted-foreground flex-shrink-0" />
                                  ) : (
                                    <Braces className="h-3 w-3 text-muted-foreground flex-shrink-0" />
                                  )}
                                  <Badge variant="outline" className="text-xs">
                                    {history.mode.toUpperCase()}
                                  </Badge>
                                  <span className="text-xs text-muted-foreground">
                                    {history.timestamp.toLocaleString()}
                                  </span>
                                  {history.mode === 'sql' && history.results && (
                                    <Badge variant="secondary" className="text-xs">
                                      {history.results.length} result(s)
                                    </Badge>
                                  )}
                                  {history.executionTime && (
                                    <Badge variant="secondary" className="text-xs">
                                      {history.executionTime.toFixed(0)}ms
                                    </Badge>
                                  )}
                                </div>
                                <code className="text-xs text-muted-foreground block truncate mt-1">
                                  {history.query.split('\n').find(l => l.trim() && !l.trim().startsWith('--') && !l.trim().startsWith('#'))?.substring(0, 80) || history.query.split('\n')[0].substring(0, 80)}
                                </code>
                              </div>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-6 w-6 p-0 ml-2"
                                onClick={(e) => {
                                  e.stopPropagation()
                                  removeHistoryItem(history.id)
                                }}
                              >
                                <X className="h-3 w-3" />
                              </Button>
                            </div>
                          ))}
                        </div>
                      </ScrollArea>
                    </CollapsibleContent>
                  </Collapsible>

                  {/* Current Result */}
                  {currentHistory && (
                    <div className="flex-1 overflow-auto">
                      <div className="p-4 space-y-4">
                        {/* SQL Results */}
                        {currentHistory.mode === 'sql' && currentHistory.results?.map((result, idx) => {
                          const pageKey = `${currentHistory.id}-${idx}`
                          const currentPage = currentPages[pageKey] || 1
                          const totalPages = result.rows ? getTotalPages(result.rows.length) : 0
                          const paginatedRows = result.rows ? getPaginatedRows(result.rows, pageKey) : []

                          return (
                            <div key={idx} className="space-y-2">
                              {/* Statement Header */}
                              <div className="flex items-center justify-between">
                                <div className="flex items-center gap-2">
                                  {result.error ? (
                                    <AlertCircle className="h-4 w-4 text-destructive" />
                                  ) : (
                                    <CheckCircle className="h-4 w-4 text-green-500" />
                                  )}
                                  <code className="text-xs text-muted-foreground">
                                    {result.statement.length > 60
                                      ? result.statement.substring(0, 60) + '...'
                                      : result.statement}
                                  </code>
                                </div>
                                <div className="flex items-center gap-2">
                                  <Badge variant="outline">
                                    {result.execution_time_ms.toFixed(2)}ms
                                  </Badge>
                                  {result.rows && result.rows.length > 0 && (
                                    <>
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => exportAsCSV(result)}
                                      >
                                        <Download className="mr-1 h-3 w-3" />
                                        CSV
                                      </Button>
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => exportAsJSON(result)}
                                      >
                                        <Download className="mr-1 h-3 w-3" />
                                        JSON
                                      </Button>
                                    </>
                                  )}
                                </div>
                              </div>

                              {/* Error Message */}
                              {result.error && (
                                <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                                  {result.error}
                                </div>
                              )}

                              {/* Results Table */}
                              {result.rows && result.rows.length > 0 && (
                                <>
                                  <div className="rounded-md border overflow-auto max-w-full">
                                    <Table className="w-max min-w-full">
                                      <TableHeader>
                                        <TableRow>
                                          {result.columns!.map((col) => (
                                            <TableHead key={col} className="font-mono text-xs whitespace-nowrap">
                                              {col}
                                            </TableHead>
                                          ))}
                                        </TableRow>
                                      </TableHeader>
                                      <TableBody>
                                        {paginatedRows.map((row, rowIdx) => (
                                          <TableRow key={rowIdx}>
                                            {result.columns!.map((col) => (
                                              <TableCell key={col} className="font-mono text-xs whitespace-nowrap">
                                                {row[col] === null
                                                  ? <span className="text-muted-foreground italic">null</span>
                                                  : typeof row[col] === 'object'
                                                  ? JSON.stringify(row[col])
                                                  : String(row[col])}
                                              </TableCell>
                                            ))}
                                          </TableRow>
                                        ))}
                                      </TableBody>
                                    </Table>
                                  </div>

                                  {/* Pagination */}
                                  {totalPages > 1 && (
                                    <div className="flex items-center justify-between">
                                      <p className="text-xs text-muted-foreground">
                                        Page {currentPage} of {totalPages} ({result.rows.length} total rows)
                                      </p>
                                      <div className="flex items-center gap-2">
                                        <Button
                                          variant="outline"
                                          size="sm"
                                          onClick={() => setPage(pageKey, currentPage - 1)}
                                          disabled={currentPage === 1}
                                        >
                                          <ChevronLeft className="h-4 w-4" />
                                          Previous
                                        </Button>
                                        <Button
                                          variant="outline"
                                          size="sm"
                                          onClick={() => setPage(pageKey, currentPage + 1)}
                                          disabled={currentPage === totalPages}
                                        >
                                          Next
                                          <ChevronRightIcon className="h-4 w-4" />
                                        </Button>
                                      </div>
                                    </div>
                                  )}

                                  {totalPages <= 1 && (
                                    <p className="text-xs text-muted-foreground">
                                      Showing {result.rows.length} row(s)
                                    </p>
                                  )}
                                </>
                              )}

                              {/* Success message for non-SELECT queries */}
                              {!result.rows && !result.error && (
                                <div className="rounded-md bg-green-500/10 p-3 text-sm text-green-600">
                                  {result.affected_rows !== undefined
                                    ? `Success: ${result.affected_rows} row(s) affected`
                                    : 'Query executed successfully'}
                                </div>
                              )}
                            </div>
                          )
                        })}

                        {/* GraphQL Results */}
                        {currentHistory.mode === 'graphql' && currentHistory.graphqlResponse && (
                          <div className="space-y-4">
                            {/* Header */}
                            <div className="flex items-center justify-between">
                              <div className="flex items-center gap-2">
                                {currentHistory.graphqlResponse.errors && currentHistory.graphqlResponse.errors.length > 0 ? (
                                  <AlertCircle className="h-4 w-4 text-destructive" />
                                ) : (
                                  <CheckCircle className="h-4 w-4 text-green-500" />
                                )}
                                <span className="text-sm font-medium">GraphQL Response</span>
                              </div>
                              <div className="flex items-center gap-2">
                                {currentHistory.executionTime && (
                                  <Badge variant="outline">
                                    {currentHistory.executionTime.toFixed(0)}ms
                                  </Badge>
                                )}
                                {currentHistory.graphqlResponse.data !== undefined && currentHistory.graphqlResponse.data !== null ? (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => {
                                      const json = JSON.stringify(currentHistory.graphqlResponse, null, 2)
                                      const blob = new Blob([json], { type: 'application/json' })
                                      const url = URL.createObjectURL(blob)
                                      const a = document.createElement('a')
                                      a.href = url
                                      a.download = `graphql-result-${Date.now()}.json`
                                      a.click()
                                      URL.revokeObjectURL(url)
                                      toast.success('Exported as JSON')
                                    }}
                                  >
                                    <Download className="mr-1 h-3 w-3" />
                                    JSON
                                  </Button>
                                ) : null}
                              </div>
                            </div>

                            {/* GraphQL Errors */}
                            {currentHistory.graphqlResponse.errors && currentHistory.graphqlResponse.errors.length > 0 ? (
                              <div className="space-y-2">
                                {currentHistory.graphqlResponse.errors.map((error, idx) => (
                                  <div key={idx} className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                                    <div className="font-medium">{error.message}</div>
                                    {error.locations && error.locations.length > 0 && (
                                      <div className="text-xs mt-1 text-destructive/80">
                                        Location: Line {error.locations[0].line}, Column {error.locations[0].column}
                                      </div>
                                    )}
                                    {error.path && error.path.length > 0 && (
                                      <div className="text-xs mt-1 text-destructive/80">
                                        Path: {error.path.join(' → ')}
                                      </div>
                                    )}
                                  </div>
                                ))}
                              </div>
                            ) : null}

                            {/* GraphQL Data */}
                            {currentHistory.graphqlResponse.data !== undefined && currentHistory.graphqlResponse.data !== null ? (
                              <div className="rounded-md border bg-muted/30 p-4 overflow-auto max-h-[500px]">
                                <pre className="text-xs font-mono whitespace-pre-wrap">
                                  {JSON.stringify(currentHistory.graphqlResponse.data, null, 2)}
                                </pre>
                              </div>
                            ) : null}

                            {/* Success with no data */}
                            {currentHistory.graphqlResponse.data === undefined && (!currentHistory.graphqlResponse.errors || currentHistory.graphqlResponse.errors.length === 0) ? (
                              <div className="rounded-md bg-green-500/10 p-3 text-sm text-green-600">
                                GraphQL query executed successfully (no data returned)
                              </div>
                            ) : null}
                          </div>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </Card>
          </Panel>
        </PanelGroup>
      </div>
    </div>
  )
}
