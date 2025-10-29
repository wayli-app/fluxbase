import { useState, useEffect } from 'react'
import { Search, ChevronRight, ChevronDown, Loader2 } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import type { OpenAPISpec, EndpointGroup, EndpointInfo } from '../types'

interface EndpointBrowserProps {
  spec: OpenAPISpec | null
  onSelectEndpoint: (endpoint: EndpointInfo) => void
  selectedEndpoint?: EndpointInfo | null
}

const METHOD_COLORS = {
  GET: 'bg-blue-500/10 text-blue-700 hover:bg-blue-500/20',
  POST: 'bg-green-500/10 text-green-700 hover:bg-green-500/20',
  PUT: 'bg-orange-500/10 text-orange-700 hover:bg-orange-500/20',
  PATCH: 'bg-yellow-500/10 text-yellow-700 hover:bg-yellow-500/20',
  DELETE: 'bg-red-500/10 text-red-700 hover:bg-red-500/20',
}

export function EndpointBrowser({ spec, onSelectEndpoint, selectedEndpoint }: EndpointBrowserProps) {
  const [groups, setGroups] = useState<EndpointGroup[]>([])
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set())
  const [searchQuery, setSearchQuery] = useState('')
  const [filterTag, setFilterTag] = useState<string>('all')
  const [filterMethod, setFilterMethod] = useState<string>('all')

  useEffect(() => {
    if (!spec) return

    // Group endpoints by tags, then by resource
    const tagMap = new Map<string, Map<string, EndpointInfo[]>>()

    Object.entries(spec.paths).forEach(([path, methods]) => {
      Object.entries(methods).forEach(([method, operation]) => {
        if (typeof operation === 'object' && 'responses' in operation) {
          const endpoint: EndpointInfo = {
            path,
            method: method.toUpperCase(),
            summary: operation.summary,
            description: operation.description,
            operationId: operation.operationId,
            tags: operation.tags || ['Other'],
            parameters: operation.parameters,
            requestBody: operation.requestBody,
            responses: operation.responses,
          }

          const tags = operation.tags || ['Other']
          tags.forEach(tag => {
            if (!tagMap.has(tag)) {
              tagMap.set(tag, new Map())
            }

            // Group by base path (without {id}) for all endpoints
            let resourceKey = path.replace(/\/\{[^}]+\}$/, '')

            const resourceMap = tagMap.get(tag)!
            if (!resourceMap.has(resourceKey)) {
              resourceMap.set(resourceKey, [])
            }
            resourceMap.get(resourceKey)!.push(endpoint)
          })
        }
      })
    })

    // Convert to hierarchical structure: tag -> resources
    const groupArray: EndpointGroup[] = []

    Array.from(tagMap.entries())
      .sort((a, b) => {
        // Put Authentication and Tables first, then alphabetical
        if (a[0] === 'Authentication') return -1
        if (b[0] === 'Authentication') return 1
        if (a[0] === 'Tables') return -1
        if (b[0] === 'Tables') return 1
        return a[0].localeCompare(b[0])
      })
      .forEach(([tag, resourceMap]) => {
        // Create parent group for the tag
        const tagResources: EndpointGroup[] = []

        Array.from(resourceMap.entries())
          .sort((a, b) => a[0].localeCompare(b[0]))
          .forEach(([resource, endpoints]) => {
            // Sort endpoints by method order: GET, POST, PUT, PATCH, DELETE
            const methodOrder: Record<string, number> = {
              GET: 1,
              POST: 2,
              PUT: 3,
              PATCH: 4,
              DELETE: 5,
            }

            const sortedEndpoints = endpoints.sort((a, b) => {
              const orderA = methodOrder[a.method] || 99
              const orderB = methodOrder[b.method] || 99
              return orderA - orderB
            })

            // Use full path as display name (use the base path without {id})
            const displayName = resource

            // Keep all endpoint variants but organize them for display
            // We need to keep all variants (with/without {id}) because they have different docs
            tagResources.push({
              name: displayName,
              endpoints: sortedEndpoints, // Keep all variants
              expanded: false,
            })
          })

        // Add parent tag group with its resources as nested structure
        groupArray.push({
          name: tag,
          endpoints: [], // Parent has no direct endpoints
          expanded: false, // Collapse all categories by default
          children: tagResources,
        })
      })

    setGroups(groupArray)
    setExpandedGroups(new Set()) // Start with all categories collapsed
  }, [spec])

  const toggleGroup = (groupName: string) => {
    setExpandedGroups(prev => {
      const next = new Set(prev)
      if (next.has(groupName)) {
        next.delete(groupName)
      } else {
        next.add(groupName)
      }
      return next
    })
  }

  // Filter endpoints based on search and filters
  const filteredGroups = groups.map(group => {
    // For hierarchical groups with children
    if (group.children && group.children.length > 0) {
      const filteredChildren = group.children.map(child => {
        const filteredEndpoints = child.endpoints.filter(endpoint => {
          // Search filter
          if (searchQuery) {
            const query = searchQuery.toLowerCase()
            const matchesPath = endpoint.path.toLowerCase().includes(query)
            const matchesSummary = endpoint.summary?.toLowerCase().includes(query)
            const matchesDescription = endpoint.description?.toLowerCase().includes(query)
            const matchesOperationId = endpoint.operationId?.toLowerCase().includes(query)
            const matchesResourceName = child.name.toLowerCase().includes(query)
            if (!matchesPath && !matchesSummary && !matchesDescription && !matchesOperationId && !matchesResourceName) {
              return false
            }
          }

          // Method filter
          if (filterMethod !== 'all' && endpoint.method !== filterMethod) {
            return false
          }

          // Tag filter
          if (filterTag !== 'all' && !endpoint.tags?.includes(filterTag)) {
            return false
          }

          return true
        })

        return {
          ...child,
          endpoints: filteredEndpoints,
        }
      }).filter(child => child.endpoints.length > 0)

      return {
        ...group,
        children: filteredChildren,
      }
    }

    // For flat groups with direct endpoints
    const filteredEndpoints = group.endpoints.filter(endpoint => {
      if (searchQuery) {
        const query = searchQuery.toLowerCase()
        const matchesPath = endpoint.path.toLowerCase().includes(query)
        const matchesSummary = endpoint.summary?.toLowerCase().includes(query)
        const matchesDescription = endpoint.description?.toLowerCase().includes(query)
        const matchesOperationId = endpoint.operationId?.toLowerCase().includes(query)
        if (!matchesPath && !matchesSummary && !matchesDescription && !matchesOperationId) {
          return false
        }
      }

      if (filterMethod !== 'all' && endpoint.method !== filterMethod) {
        return false
      }

      if (filterTag !== 'all' && !endpoint.tags?.includes(filterTag)) {
        return false
      }

      return true
    })

    return {
      ...group,
      endpoints: filteredEndpoints,
    }
  }).filter(group => (group.children && group.children.length > 0) || group.endpoints.length > 0)

  const allTags = Array.from(new Set(
    groups.flatMap(g =>
      g.children
        ? g.children.flatMap(c => c.endpoints.flatMap(e => e.tags || []))
        : g.endpoints.flatMap(e => e.tags || [])
    )
  ))

  // Show loading state when spec is not available
  if (!spec) {
    return (
      <div className="flex flex-col h-full">
        <div className="p-4 flex items-center gap-3 text-muted-foreground">
          <Loader2 className="h-5 w-5 animate-spin" />
          <p className="text-sm">Loading API endpoints...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Search and Filters */}
      <div className="p-4 border-b space-y-3">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search endpoints..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        <div className="flex gap-2">
          <Select value={filterMethod} onValueChange={setFilterMethod}>
            <SelectTrigger className="flex-1">
              <SelectValue placeholder="Method" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Methods</SelectItem>
              <SelectItem value="GET">GET</SelectItem>
              <SelectItem value="POST">POST</SelectItem>
              <SelectItem value="PUT">PUT</SelectItem>
              <SelectItem value="PATCH">PATCH</SelectItem>
              <SelectItem value="DELETE">DELETE</SelectItem>
            </SelectContent>
          </Select>

          <Select value={filterTag} onValueChange={setFilterTag}>
            <SelectTrigger className="flex-1">
              <SelectValue placeholder="Tag" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Tags</SelectItem>
              {allTags.map(tag => (
                <SelectItem key={tag} value={tag}>{tag}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Endpoint List */}
      <ScrollArea className="flex-1">
        <div className="p-2">
          {filteredGroups.map(group => (
            <div key={group.name} className="mb-1">
              {/* Parent Tag Group (collapsible) */}
              <button
                className="w-full flex items-center gap-2 p-2 hover:bg-muted/50 rounded text-left"
                onClick={() => toggleGroup(group.name)}
              >
                {expandedGroups.has(group.name) ? (
                  <ChevronDown className="h-4 w-4 shrink-0" />
                ) : (
                  <ChevronRight className="h-4 w-4 shrink-0" />
                )}
                <span className="font-semibold text-sm">{group.name}</span>
                <span className="text-xs text-muted-foreground ml-auto">
                  {group.children?.length || 0}
                </span>
              </button>

              {/* Child Resources (with method badges) */}
              {expandedGroups.has(group.name) && group.children && (
                <div className="ml-6 mt-1 space-y-1">
                  {group.children.map(resource => {
                    const isSelected = resource.endpoints.some(
                      e => selectedEndpoint?.path === e.path && selectedEndpoint?.method === e.method
                    )

                    // Get unique methods and their corresponding endpoints
                    const methodMap = new Map<string, EndpointInfo>()
                    resource.endpoints.forEach(endpoint => {
                      // Prefer endpoints without {id} for list operations (GET, POST)
                      // Prefer endpoints with {id} for single operations (PUT, PATCH, DELETE)
                      const current = methodMap.get(endpoint.method)
                      if (!current) {
                        methodMap.set(endpoint.method, endpoint)
                      } else if (
                        (endpoint.method === 'GET' || endpoint.method === 'POST') &&
                        !endpoint.path.includes('{id}')
                      ) {
                        methodMap.set(endpoint.method, endpoint)
                      } else if (
                        (endpoint.method === 'PUT' || endpoint.method === 'PATCH' || endpoint.method === 'DELETE') &&
                        endpoint.path.includes('{id}')
                      ) {
                        methodMap.set(endpoint.method, endpoint)
                      }
                    })

                    const uniqueEndpoints = Array.from(methodMap.entries()).sort((a, b) => {
                      const methodOrder: Record<string, number> = {
                        GET: 1,
                        POST: 2,
                        PUT: 3,
                        PATCH: 4,
                        DELETE: 5,
                      }
                      return (methodOrder[a[0]] || 99) - (methodOrder[b[0]] || 99)
                    })

                    return (
                      <button
                        key={resource.name}
                        className={cn(
                          "w-full flex flex-col gap-1 p-2 rounded text-left transition-colors",
                          "hover:bg-muted/50",
                          isSelected && "bg-muted"
                        )}
                        onClick={() => {
                          // Select the first endpoint (usually GET)
                          if (uniqueEndpoints.length > 0) {
                            onSelectEndpoint(uniqueEndpoints[0][1])
                          }
                        }}
                      >
                        <div className="text-xs font-mono text-muted-foreground truncate">
                          {resource.name}
                        </div>
                        <div className="flex gap-1 flex-wrap">
                          {uniqueEndpoints.map(([method, endpoint]) => (
                            <Badge
                              key={method}
                              variant="outline"
                              className={cn(
                                "text-xs font-mono shrink-0 cursor-pointer",
                                METHOD_COLORS[method as keyof typeof METHOD_COLORS]
                              )}
                              onClick={(e) => {
                                e.stopPropagation()
                                onSelectEndpoint(endpoint)
                              }}
                            >
                              {method}
                            </Badge>
                          ))}
                        </div>
                      </button>
                    )
                  })}
                </div>
              )}
            </div>
          ))}

          {filteredGroups.length === 0 && (
            <div className="text-center text-muted-foreground py-8">
              {searchQuery || filterMethod !== 'all' || filterTag !== 'all'
                ? 'No endpoints match your filters'
                : 'No endpoints available'}
            </div>
          )}
        </div>
      </ScrollArea>

      {/* Stats */}
      <div className="p-3 border-t text-xs text-muted-foreground">
        {spec && (
          <div className="space-y-1">
            <div>
              {Object.keys(spec.paths).length} paths, {
                Object.values(spec.paths).reduce((acc, methods) =>
                  acc + Object.keys(methods).length, 0
                )
              } endpoints
            </div>
            <div>{filteredGroups.reduce((acc, g) => acc + g.endpoints.length, 0)} visible</div>
          </div>
        )}
      </div>
    </div>
  )
}