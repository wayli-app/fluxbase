import { useState, useEffect, useCallback } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { ArrowLeft, Search, RefreshCw, FileText, Bug, ChevronLeft, ChevronRight } from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  knowledgeBasesApi,
  type KnowledgeBase,
  type SearchResult,
  type DebugSearchResult,
} from '@/lib/api'

const RESULTS_PER_PAGE = 5

function truncateText(text: string, maxLength: number = 150): string {
  if (text.length <= maxLength) return text
  return text.slice(0, maxLength).trim() + '...'
}

type SearchMode = 'semantic' | 'keyword' | 'hybrid'

export const Route = createFileRoute(
  '/_authenticated/knowledge-bases/$id/search'
)({
  component: KnowledgeBaseSearchPage,
})

function KnowledgeBaseSearchPage() {
  const { id } = Route.useParams()
  const navigate = useNavigate()
  const [knowledgeBase, setKnowledgeBase] = useState<KnowledgeBase | null>(null)
  const [loading, setLoading] = useState(true)
  const [searching, setSearching] = useState(false)
  const [debugging, setDebugging] = useState(false)
  const [query, setQuery] = useState('')
  const [maxChunks, setMaxChunks] = useState(5)
  const [threshold, setThreshold] = useState(0.2)
  const [searchMode, setSearchMode] = useState<SearchMode>('hybrid')
  const [results, setResults] = useState<SearchResult[]>([])
  const [debugResult, setDebugResult] = useState<DebugSearchResult | null>(null)
  const [hasSearched, setHasSearched] = useState(false)
  const [currentPage, setCurrentPage] = useState(1)
  const [selectedResult, setSelectedResult] = useState<SearchResult | null>(null)

  const fetchKnowledgeBase = useCallback(async () => {
    setLoading(true)
    try {
      const kb = await knowledgeBasesApi.get(id)
      setKnowledgeBase(kb)
    } catch {
      toast.error('Failed to fetch knowledge base')
    } finally {
      setLoading(false)
    }
  }, [id])

  const handleSearch = async () => {
    if (!query.trim()) {
      toast.error('Please enter a search query')
      return
    }

    setSearching(true)
    setHasSearched(true)
    setDebugResult(null)
    setCurrentPage(1)
    try {
      const response = await knowledgeBasesApi.search(id, query, {
        max_chunks: maxChunks,
        threshold: threshold,
        mode: searchMode,
      })
      setResults(response.results || [])
      toast.success(`Found ${response.results?.length || 0} results (${response.mode} search)`)
    } catch {
      toast.error('Search failed')
      setResults([])
    } finally {
      setSearching(false)
    }
  }

  const handleDebugSearch = async () => {
    if (!query.trim()) {
      toast.error('Please enter a search query')
      return
    }

    setDebugging(true)
    setHasSearched(true)
    try {
      const result = await knowledgeBasesApi.debugSearch(id, query)
      setDebugResult(result)
      toast.success(`Debug search found ${result.chunks_found} chunks`)
    } catch {
      toast.error('Debug search failed')
      setDebugResult(null)
    } finally {
      setDebugging(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch()
    }
  }

  useEffect(() => {
    fetchKnowledgeBase()
  }, [fetchKnowledgeBase])

  if (loading) {
    return (
      <div className="flex h-96 items-center justify-center">
        <RefreshCw className="text-muted-foreground h-8 w-8 animate-spin" />
      </div>
    )
  }

  if (!knowledgeBase) {
    return (
      <div className="flex h-96 flex-col items-center justify-center gap-4">
        <p className="text-muted-foreground">Knowledge base not found</p>
        <Button
          variant="outline"
          onClick={() => navigate({ to: '/knowledge-bases' })}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Knowledge Bases
        </Button>
      </div>
    )
  }

  return (
    <div className="flex flex-1 flex-col gap-6 p-6">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate({ to: '/knowledge-bases' })}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
      </div>

      <div>
        <h1 className="text-3xl font-bold">Search: {knowledgeBase.name}</h1>
        <p className="text-muted-foreground">
          Search documents using semantic similarity, keyword matching, or hybrid
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Search Query</CardTitle>
          <CardDescription>
            Enter a query to search for relevant document chunks
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="query">Query</Label>
              <div className="flex gap-2">
                <Input
                  id="query"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder="What would you like to search for?"
                  className="flex-1"
                />
                <Button onClick={handleSearch} disabled={searching || debugging}>
                  {searching ? (
                    <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <Search className="mr-2 h-4 w-4" />
                  )}
                  Search
                </Button>
                <Button
                  variant="outline"
                  onClick={handleDebugSearch}
                  disabled={searching || debugging}
                  title="Debug search - shows raw similarity scores and embedding info"
                >
                  {debugging ? (
                    <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <Bug className="mr-2 h-4 w-4" />
                  )}
                  Debug
                </Button>
              </div>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="searchMode">Search Mode</Label>
                <Select
                  value={searchMode}
                  onValueChange={(value: SearchMode) => setSearchMode(value)}
                >
                  <SelectTrigger id="searchMode">
                    <SelectValue placeholder="Select mode" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="hybrid">
                      Hybrid (recommended)
                    </SelectItem>
                    <SelectItem value="semantic">Semantic only</SelectItem>
                    <SelectItem value="keyword">Keyword only</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="maxChunks">Max Results</Label>
                <Input
                  id="maxChunks"
                  type="number"
                  min={1}
                  max={20}
                  value={maxChunks}
                  onChange={(e) => setMaxChunks(parseInt(e.target.value) || 5)}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="threshold">Similarity Threshold</Label>
                <Input
                  id="threshold"
                  type="number"
                  min={0}
                  max={1}
                  step={0.1}
                  value={threshold}
                  onChange={(e) =>
                    setThreshold(parseFloat(e.target.value) || 0.2)
                  }
                />
              </div>
            </div>
            <p className="text-muted-foreground text-xs">
              <strong>Hybrid:</strong> Combines semantic meaning + keyword matching.{' '}
              <strong>Semantic:</strong> Finds conceptually similar content.{' '}
              <strong>Keyword:</strong> Finds exact text matches.
            </p>
          </div>
        </CardContent>
      </Card>

      {debugResult && (
        <Card className="border-amber-500/50 bg-amber-500/5">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Bug className="h-5 w-5" />
              Debug Information
            </CardTitle>
            <CardDescription>
              Raw similarity scores and embedding details (no threshold filtering)
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 font-mono text-sm">
              {debugResult.error_message && (
                <div className="rounded border border-red-500/50 bg-red-500/10 p-3 text-red-600 dark:text-red-400">
                  <strong>Issue Detected:</strong> {debugResult.error_message}
                </div>
              )}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <span className="text-muted-foreground">Query:</span>{' '}
                  <span className="font-semibold">{debugResult.query}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Chunks Found:</span>{' '}
                  <span className="font-semibold">{debugResult.chunks_found}</span>
                </div>
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <span className="text-muted-foreground">Total Chunks:</span>{' '}
                  <span className="font-semibold">{debugResult.total_chunks}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">With Embedding:</span>{' '}
                  <span className={`font-semibold ${debugResult.chunks_with_embedding === 0 ? 'text-red-500' : 'text-green-500'}`}>
                    {debugResult.chunks_with_embedding}
                  </span>
                </div>
                <div>
                  <span className="text-muted-foreground">Without Embedding:</span>{' '}
                  <span className={`font-semibold ${debugResult.chunks_without_embedding > 0 ? 'text-red-500' : ''}`}>
                    {debugResult.chunks_without_embedding}
                  </span>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <span className="text-muted-foreground">Query Embedding Model:</span>{' '}
                  <span className="font-semibold">{debugResult.embedding_model}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">KB Embedding Model:</span>{' '}
                  <span className={`font-semibold ${debugResult.embedding_model !== debugResult.kb_embedding_model ? 'text-amber-500' : ''}`}>
                    {debugResult.kb_embedding_model}
                  </span>
                  {debugResult.embedding_model !== debugResult.kb_embedding_model && (
                    <span className="ml-2 text-amber-500">(mismatch!)</span>
                  )}
                </div>
              </div>
              <div>
                <span className="text-muted-foreground">Query Embedding Dims:</span>{' '}
                <span className="font-semibold">{debugResult.query_embedding_dims}</span>
              </div>
              <div>
                <span className="text-muted-foreground">Query Embedding Preview (first 10):</span>
                <pre className="mt-1 overflow-x-auto rounded bg-black/20 p-2 text-xs">
                  [{debugResult.query_embedding_preview.map(v => v.toFixed(6)).join(', ')}]
                </pre>
              </div>
              {debugResult.stored_embedding_preview && (
                <div>
                  <span className="text-muted-foreground">Stored Embedding Preview (first 10):</span>
                  <pre className="mt-1 overflow-x-auto rounded bg-black/20 p-2 text-xs">
                    [{debugResult.stored_embedding_preview.map(v => v.toFixed(6)).join(', ')}]
                  </pre>
                </div>
              )}
              <div>
                <span className="text-muted-foreground">Raw Similarities:</span>
                {debugResult.raw_similarities.length === 0 ? (
                  <span className="ml-2 text-red-500">No results - all chunks may have NULL embeddings</span>
                ) : (
                  <pre className="mt-1 overflow-x-auto rounded bg-black/20 p-2 text-xs">
                    [{debugResult.raw_similarities.map(v => v.toFixed(6)).join(', ')}]
                  </pre>
                )}
              </div>
              {debugResult.top_chunk_content_preview && (
                <div>
                  <span className="text-muted-foreground">Top Chunk Content:</span>
                  <pre className="mt-1 overflow-x-auto rounded bg-black/20 p-2 text-xs whitespace-pre-wrap">
                    {debugResult.top_chunk_content_preview}
                  </pre>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {hasSearched && !debugResult && (
        <Card>
          <CardHeader>
            <CardTitle>Results</CardTitle>
            <CardDescription>
              {searching ? (
                <span className="flex items-center gap-2">
                  <RefreshCw className="h-3 w-3 animate-spin" />
                  Searching...
                </span>
              ) : (
                `${results.length} result${results.length !== 1 ? 's' : ''} found`
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {searching ? (
              <div className="flex h-[300px] items-center justify-center">
                <RefreshCw className="text-muted-foreground h-8 w-8 animate-spin" />
              </div>
            ) : results.length === 0 ? (
              <div className="py-12 text-center">
                <Search className="text-muted-foreground mx-auto mb-4 h-12 w-12" />
                <p className="mb-2 text-lg font-medium">No results found</p>
                <p className="text-muted-foreground text-sm">
                  Try adjusting your query or lowering the similarity threshold
                </p>
              </div>
            ) : (
              <>
                <div className="space-y-3">
                  {results
                    .slice((currentPage - 1) * RESULTS_PER_PAGE, currentPage * RESULTS_PER_PAGE)
                    .map((result, index) => (
                    <Card
                      key={result.chunk_id || index}
                      className="cursor-pointer transition-colors hover:bg-muted/50"
                      onClick={() => setSelectedResult(result)}
                    >
                      <CardHeader className="pb-2">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <FileText className="h-4 w-4" />
                            <span className="font-medium">
                              {result.document_title || 'Untitled'}
                            </span>
                          </div>
                          <Badge
                            variant="secondary"
                            className="bg-green-500/10 text-green-600 dark:text-green-400"
                          >
                            {(result.similarity * 100).toFixed(1)}% match
                          </Badge>
                        </div>
                      </CardHeader>
                      <CardContent>
                        <p className="text-muted-foreground text-sm">
                          {truncateText(result.content)}
                        </p>
                        {result.content.length > 150 && (
                          <p className="text-primary mt-1 text-xs">Click to view full content</p>
                        )}
                      </CardContent>
                    </Card>
                  ))}
                </div>
                {results.length > RESULTS_PER_PAGE && (
                  <div className="flex items-center justify-between pt-4 border-t mt-4">
                    <p className="text-muted-foreground text-sm">
                      Showing {(currentPage - 1) * RESULTS_PER_PAGE + 1}-{Math.min(currentPage * RESULTS_PER_PAGE, results.length)} of {results.length}
                    </p>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                        disabled={currentPage === 1}
                      >
                        <ChevronLeft className="h-4 w-4" />
                        Previous
                      </Button>
                      <span className="text-muted-foreground text-sm px-2">
                        Page {currentPage} of {Math.ceil(results.length / RESULTS_PER_PAGE)}
                      </span>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPage(p => Math.min(Math.ceil(results.length / RESULTS_PER_PAGE), p + 1))}
                        disabled={currentPage >= Math.ceil(results.length / RESULTS_PER_PAGE)}
                      >
                        Next
                        <ChevronRight className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>
      )}

      <Dialog open={!!selectedResult} onOpenChange={(open) => !open && setSelectedResult(null)}>
        <DialogContent className="max-w-2xl max-h-[80vh]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5" />
              {selectedResult?.document_title || 'Untitled'}
            </DialogTitle>
            <DialogDescription>
              {selectedResult && (
                <Badge
                  variant="secondary"
                  className="bg-green-500/10 text-green-600 dark:text-green-400"
                >
                  {(selectedResult.similarity * 100).toFixed(1)}% match
                </Badge>
              )}
            </DialogDescription>
          </DialogHeader>
          <ScrollArea className="max-h-[60vh]">
            <p className="text-sm whitespace-pre-wrap pr-4">
              {selectedResult?.content}
            </p>
          </ScrollArea>
        </DialogContent>
      </Dialog>
    </div>
  )
}
