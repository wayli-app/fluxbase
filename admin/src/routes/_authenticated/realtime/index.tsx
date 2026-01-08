import { createFileRoute } from '@tanstack/react-router'
import { useState, useEffect, useCallback, useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Radio,
  RefreshCw,
  Users,
  Activity,
  Clock,
  Search,
  User,
  Globe,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDistanceToNow } from 'date-fns'
import api from '@/lib/api'
import { getPageNumbers } from '@/lib/utils'
import { useRealtimeConnections } from '@/hooks/use-realtime-connections'

export const Route = createFileRoute('/_authenticated/realtime/')({
  component: RealtimePage,
})

// Types
interface ConnectionInfo {
  id: string
  user_id: string | null
  email: string | null
  display_name?: string | null
  remote_addr: string
  connected_at: string
}

interface RealtimeStats {
  total_connections: number
  connections: ConnectionInfo[]
  limit: number
  offset: number
}

function RealtimePage() {
  // State
  const [initialConnections, setInitialConnections] = useState<ConnectionInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')

  // Pagination state
  const [currentPage, setCurrentPage] = useState(0)
  const [pageSize, setPageSize] = useState(25)

  // Debounce search query
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(searchQuery)
      setCurrentPage(0) // Reset to first page on search change
    }, 300)
    return () => clearTimeout(timer)
  }, [searchQuery])

  // Fetch initial realtime stats
  const fetchStats = useCallback(async () => {
    try {
      // Fetch all connections for initial load (we'll do client-side filtering/pagination)
      const params = new URLSearchParams({
        limit: '1000', // Get a large batch for client-side management
        offset: '0',
      })

      const response = await api.get<RealtimeStats>(`/api/v1/realtime/stats?${params}`)
      setInitialConnections(response.data.connections || [])
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error fetching realtime stats:', error)
      toast.error('Failed to load realtime statistics')
    } finally {
      setLoading(false)
    }
  }, [])

  // Use realtime connections hook for live updates
  const {
    connections: liveConnections,
    totalConnections,
    isSubscribed,
  } = useRealtimeConnections({
    initialConnections,
    enabled: true,
    onSubscribed: fetchStats, // Fetch after WebSocket connects
  })

  // Client-side filtering
  const filteredConnections = useMemo(() => {
    if (!debouncedSearch) return liveConnections

    const search = debouncedSearch.toLowerCase()
    return liveConnections.filter(
      (conn) =>
        conn.id.toLowerCase().includes(search) ||
        conn.remote_addr.toLowerCase().includes(search) ||
        (conn.user_id && conn.user_id.toLowerCase().includes(search)) ||
        (conn.email && conn.email.toLowerCase().includes(search)) ||
        (conn.display_name && conn.display_name.toLowerCase().includes(search))
    )
  }, [liveConnections, debouncedSearch])

  // Client-side pagination
  const paginatedConnections = useMemo(() => {
    const start = currentPage * pageSize
    const end = start + pageSize
    return filteredConnections.slice(start, end)
  }, [filteredConnections, currentPage, pageSize])

  // Calculate connection duration
  const getConnectionDuration = (connectedAt: string) => {
    try {
      return formatDistanceToNow(new Date(connectedAt), { addSuffix: true })
    } catch {
      return 'Unknown'
    }
  }

  // Format user display with priority: display_name > email > user_id > Anonymous
  const formatUserDisplay = (conn: ConnectionInfo) => {
    if (conn.display_name) return conn.display_name
    if (conn.email) return conn.email
    if (conn.user_id) return `${conn.user_id.substring(0, 8)}...`
    return null
  }

  // Pagination calculations
  const totalPages = Math.ceil(filteredConnections.length / pageSize) || 1

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="flex flex-col items-center gap-2">
          <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
          <p className="text-sm text-muted-foreground">Loading realtime stats...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b bg-background px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <Radio className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h1 className="text-xl font-semibold">Realtime Dashboard</h1>
            <p className="text-sm text-muted-foreground">
              Monitor WebSocket connections and subscriptions
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={fetchStats}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 gap-4 p-6 md:grid-cols-2">
        <Card className="p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10">
              <Users className="h-5 w-5 text-blue-500" />
            </div>
            <div>
              <p className="text-2xl font-bold">{totalConnections}</p>
              <p className="text-sm text-muted-foreground">Active Connections</p>
            </div>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center gap-3">
            <div
              className={`flex h-10 w-10 items-center justify-center rounded-lg ${
                isSubscribed ? 'bg-green-500/10' : 'bg-red-500/10'
              }`}
            >
              <Activity className={`h-5 w-5 ${isSubscribed ? 'text-green-500' : 'text-red-500'}`} />
            </div>
            <div>
              <p className="text-2xl font-bold">
                {isSubscribed ? (
                  <span className="flex items-center gap-2">
                    <span className="h-2 w-2 rounded-full bg-green-500 animate-pulse" />
                    Live
                  </span>
                ) : (
                  <span className="flex items-center gap-2">
                    <span className="h-2 w-2 rounded-full bg-red-500" />
                    Disconnected
                  </span>
                )}
              </p>
              <p className="text-sm text-muted-foreground">WebSocket Status</p>
            </div>
          </div>
        </Card>
      </div>

      {/* Connections */}
      <div className="flex-1 overflow-hidden px-6 pb-6 flex flex-col">
        <div className="mb-4 flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search connections by ID, email, or IP address..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>

        <Card className="flex-1 overflow-hidden flex flex-col">
          <ScrollArea className="flex-1">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Connection ID</TableHead>
                  <TableHead>User</TableHead>
                  <TableHead>IP Address</TableHead>
                  <TableHead>Connected</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paginatedConnections.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center py-8">
                      <div className="flex flex-col items-center gap-2">
                        <Users className="h-8 w-8 text-muted-foreground" />
                        <p className="text-sm text-muted-foreground">
                          {searchQuery ? 'No connections found' : 'No active connections'}
                        </p>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  paginatedConnections.map((conn) => (
                    <TableRow key={conn.id}>
                      <TableCell className="font-mono text-xs">
                        {conn.id.substring(0, 8)}...
                      </TableCell>
                      <TableCell>
                        {conn.user_id ? (
                          <div className="flex items-center gap-2">
                            <User className="h-4 w-4 text-muted-foreground" />
                            <span className="text-sm">{formatUserDisplay(conn)}</span>
                          </div>
                        ) : (
                          <Badge variant="secondary">Anonymous</Badge>
                        )}
                      </TableCell>
                      <TableCell className="font-mono text-xs">
                        <div className="flex items-center gap-2">
                          <Globe className="h-4 w-4 text-muted-foreground" />
                          {conn.remote_addr}
                        </div>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        <div className="flex items-center gap-2">
                          <Clock className="h-4 w-4" />
                          {getConnectionDuration(conn.connected_at)}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </ScrollArea>

          {/* Pagination */}
          {filteredConnections.length > 0 && (
            <div className="flex items-center justify-between border-t px-4 py-3">
              <div className="flex items-center gap-2">
                <span className="text-sm text-muted-foreground">Rows per page:</span>
                <Select
                  value={`${pageSize}`}
                  onValueChange={(value) => {
                    setPageSize(Number(value))
                    setCurrentPage(0)
                  }}
                >
                  <SelectTrigger className="h-8 w-[70px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent side="top">
                    {[10, 25, 50, 100].map((size) => (
                      <SelectItem key={size} value={`${size}`}>
                        {size}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="flex items-center gap-2">
                <span className="text-sm text-muted-foreground">
                  Page {currentPage + 1} of {totalPages} ({filteredConnections.length}{' '}
                  {debouncedSearch ? 'filtered' : 'total'})
                </span>

                {/* First page */}
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 w-8 p-0"
                  onClick={() => setCurrentPage(0)}
                  disabled={currentPage === 0}
                >
                  <ChevronsLeft className="h-4 w-4" />
                </Button>

                {/* Previous page */}
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 w-8 p-0"
                  onClick={() => setCurrentPage((prev) => Math.max(0, prev - 1))}
                  disabled={currentPage === 0}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>

                {/* Page numbers */}
                {getPageNumbers(currentPage + 1, totalPages).map((pageNum, idx) =>
                  pageNum === '...' ? (
                    <span key={`ellipsis-${idx}`} className="px-1 text-muted-foreground">
                      ...
                    </span>
                  ) : (
                    <Button
                      key={pageNum}
                      variant={currentPage + 1 === pageNum ? 'default' : 'outline'}
                      size="sm"
                      className="h-8 min-w-8 px-2"
                      onClick={() => setCurrentPage((pageNum as number) - 1)}
                    >
                      {pageNum}
                    </Button>
                  )
                )}

                {/* Next page */}
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 w-8 p-0"
                  onClick={() => setCurrentPage((prev) => Math.min(totalPages - 1, prev + 1))}
                  disabled={currentPage >= totalPages - 1}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>

                {/* Last page */}
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 w-8 p-0"
                  onClick={() => setCurrentPage(totalPages - 1)}
                  disabled={currentPage >= totalPages - 1}
                >
                  <ChevronsRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </Card>
      </div>
    </div>
  )
}
