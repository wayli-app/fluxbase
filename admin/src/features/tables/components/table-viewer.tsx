import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQueryClient, useQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import {
  type ColumnDef,
  type SortingState,
  type RowSelectionState,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import {
  useTable as useFluxbaseTable,
  useUpdate,
  useDelete,
} from '@fluxbase/sdk-react'
import { Plus, Trash2, User, X, ShieldAlert } from 'lucide-react'
import { toast } from 'sonner'
import { apiClient } from '@/lib/api'
import { useImpersonationStore } from '@/stores/impersonation-store'
import { UserSearch } from '@/features/impersonation/components/user-search'
import { impersonationApi } from '@/lib/impersonation-api'
import { cn } from '@/lib/utils'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  DataTablePagination,
  DataTableColumnHeader,
} from '@/components/data-table'
import { EditableCell } from './editable-cell'
import { RecordEditDialog } from './record-edit-dialog'
import { TableRowActions } from './table-row-actions'

const route = getRouteApi('/_authenticated/tables/')

interface TableViewerProps {
  tableName: string
  schema: string
}

export function TableViewer({ tableName, schema }: TableViewerProps) {
  const queryClient = useQueryClient()
  const [sorting, setSorting] = useState<SortingState>([])
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({})
  const [editingRecord, setEditingRecord] = useState<Record<
    string,
    unknown
  > | null>(null)
  const [isCreating, setIsCreating] = useState(false)

  // Impersonation state - use global store directly
  const [isStartingImpersonation, setIsStartingImpersonation] = useState(false)
  const {
    isImpersonating,
    impersonationType,
    impersonatedUser,
    stopImpersonation: clearImpersonationStore,
  } = useImpersonationStore()

  // Handle starting user impersonation
  const handleStartUserImpersonation = async (userId: string, userEmail: string) => {
    try {
      setIsStartingImpersonation(true)
      const response = await impersonationApi.startUserImpersonation(
        userId,
        'Testing RLS policies in Tables view'
      )

      // Update global impersonation store
      useImpersonationStore.getState().startImpersonation(
        response.access_token,
        response.refresh_token,
        response.target_user,
        response.session,
        'user'
      )

      // Invalidate queries to refetch with new token
      queryClient.invalidateQueries()
      toast.success(`Now viewing as ${userEmail}`)
    } catch {
      toast.error('Failed to start impersonation')
    } finally {
      setIsStartingImpersonation(false)
    }
  }

  // Handle starting anonymous impersonation
  const handleStartAnonImpersonation = async () => {
    try {
      setIsStartingImpersonation(true)
      const response = await impersonationApi.startAnonImpersonation(
        'Testing anonymous access in Tables view'
      )

      useImpersonationStore.getState().startImpersonation(
        response.access_token,
        response.refresh_token,
        response.target_user,
        response.session,
        'anon'
      )

      queryClient.invalidateQueries()
      toast.success('Now viewing as Anonymous user')
    } catch {
      toast.error('Failed to start anonymous impersonation')
    } finally {
      setIsStartingImpersonation(false)
    }
  }

  // Handle starting service role impersonation
  const handleStartServiceImpersonation = async () => {
    try {
      setIsStartingImpersonation(true)
      const response = await impersonationApi.startServiceImpersonation(
        'Testing service role access in Tables view'
      )

      useImpersonationStore.getState().startImpersonation(
        response.access_token,
        response.refresh_token,
        response.target_user,
        response.session,
        'service'
      )

      queryClient.invalidateQueries()
      toast.success('Now viewing as Service Role')
    } catch {
      toast.error('Failed to start service role impersonation')
    } finally {
      setIsStartingImpersonation(false)
    }
  }

  // Handle stopping impersonation
  const handleStopImpersonation = async () => {
    try {
      await impersonationApi.stopImpersonation()
      toast.success('Stopped impersonation')
    } catch {
      // Still clear local state so user can recover
      toast.info('Cleared impersonation (session may have already expired)')
    } finally {
      // Always clear local state so user isn't stuck
      clearImpersonationStore()
      queryClient.invalidateQueries()
    }
  }

  // Get display label for current impersonation
  const getImpersonationLabel = () => {
    if (!isImpersonating) return null
    switch (impersonationType) {
      case 'anon':
        return 'Anonymous'
      case 'service':
        return 'Service Role'
      default:
        return impersonatedUser?.email || 'Unknown User'
    }
  }

  // Get table metadata from React Query cache (already fetched by TableSelector)
  // Fetch specific table schema to ensure columns are available even when table is empty
  const { data: tableInfo } = useQuery<{
    schema: string
    name: string
    rest_path?: string
    columns: Array<{
      name: string
      data_type: string
      is_nullable: boolean
      default_value: string | null
      is_primary_key: boolean
    }>
  }>({
    queryKey: ['table-schema', schema, tableName],
    queryFn: async () => {
      // Extract table name from the format "schema.table" or just use tableName
      const tableNameOnly = tableName.includes('.')
        ? tableName.split('.')[1]
        : tableName
      const response = await apiClient.get(
        `/api/v1/admin/tables/${schema}/${tableNameOnly}`
      )
      return response.data
    },
    staleTime: 60000, // Cache for 1 minute
  })
  // Use schema/table format for REST API path to match backend expectations
  const tableApiPath =
    schema === 'public'
      ? tableInfo?.name || tableName.split('.')[1]
      : `${schema}/${tableInfo?.name || tableName.split('.')[1]}`

  const {
    pagination,
    onPaginationChange,
    globalFilter,
    onGlobalFilterChange,
    ensurePageInRange,
  } = useTableUrlState({
    search: route.useSearch(),
    navigate: route.useNavigate(),
    pagination: { defaultPage: 1, defaultPageSize: 10 },
    globalFilter: { enabled: true, key: 'filter' },
  })

  // Fetch table data using Fluxbase SDK
  const { data, isLoading, error: tableError } = useFluxbaseTable(
    tableApiPath,
    (query) => {
      let q = query
        .select('*')
        .limit(pagination.pageSize)
        .offset(pagination.pageIndex * pagination.pageSize)

      if (sorting[0]) {
        q = q.order(sorting[0].id, { ascending: !sorting[0].desc })
      }

      return q
    },
    {
      queryKey: ['table-data', tableName, pagination, sorting],
      enabled: !!tableName,
    }
  )

  // Fetch total count for pagination
  const { data: countData } = useQuery<number>({
    queryKey: ['table-count', tableName, tableApiPath],
    queryFn: async () => {
      const response = await apiClient.get(
        `/rest/v1/${tableApiPath}?select=count&count=exact`,
        { headers: { 'Prefer': 'count=exact' } }
      )
      // Parse count from Content-Range header: "0-0/123" -> 123
      const contentRange = response.headers['content-range']
      if (contentRange) {
        const match = contentRange.match(/\/(\d+)$/)
        if (match && match[1]) {
          return parseInt(match[1], 10)
        }
      }
      // Fallback to data length if no header
      return Array.isArray(response.data) ? response.data.length : 0
    },
    enabled: !!tableName,
    staleTime: 30000, // Cache for 30 seconds
  })

  // Calculate page count from total rows
  const totalRowCount = countData ?? 0
  const calculatedPageCount = Math.ceil(totalRowCount / pagination.pageSize)

  // Check if error is a 403 Forbidden (RLS/permission issue)
  const isForbidden = tableError &&
    typeof tableError === 'object' &&
    'status' in tableError &&
    (tableError as { status?: number }).status === 403

  // Update mutation using Fluxbase SDK
  const updateFluxbase = useUpdate(tableApiPath)
  const updateMutation = useMutation({
    mutationFn: ({
      id,
      field,
      value,
    }: {
      id: string | number
      field: string
      value: unknown
    }) =>
      updateFluxbase.mutateAsync({
        data: { [field]: value },
        buildQuery: (q) => q.eq('id', id),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['table-data', tableName] })
      toast.success('Record updated successfully')
    },
    onError: (error: Error) => {
      toast.error(`Failed to update record: ${error.message}`)
    },
  })

  // Delete mutation using Fluxbase SDK
  const deleteFluxbase = useDelete(tableApiPath)
  const deleteMutation = useMutation({
    mutationFn: (record: Record<string, unknown>) =>
      deleteFluxbase.mutateAsync((q) => q.eq('id', record.id)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['table-data', tableName] })
      queryClient.invalidateQueries({ queryKey: ['table-count', tableName] })
      toast.success('Record deleted successfully')
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete record: ${error.message}`)
    },
  })

  // Bulk delete mutation
  const bulkDeleteMutation = useMutation({
    mutationFn: async (ids: string[]) => {
      // Delete each record individually
      await Promise.all(
        ids.map((id) => deleteFluxbase.mutateAsync((q) => q.eq('id', id)))
      )
    },
    onSuccess: (_, ids) => {
      queryClient.invalidateQueries({ queryKey: ['table-data', tableName] })
      queryClient.invalidateQueries({ queryKey: ['table-count', tableName] })
      toast.success(
        `${ids.length} record${ids.length !== 1 ? 's' : ''} deleted successfully`
      )
      setRowSelection({})
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete records: ${error.message}`)
    },
  })

  // Get column metadata from table info (fetched directly from schema endpoint)
  const tableColumns = useMemo(
    () => tableInfo?.columns || [],
    [tableInfo?.columns]
  )

  // Destructure stable functions from mutations
  const updateMutateAsync = updateMutation.mutateAsync
  const deleteMutate = deleteMutation.mutate

  // Generate columns dynamically
  const columns = useMemo<ColumnDef<Record<string, unknown>>[]>(() => {
    // Use table schema if available, otherwise fall back to data keys
    let columnKeys: string[]
    const columnTypes: Record<string, string> = {}

    if (tableColumns.length > 0) {
      columnKeys = tableColumns.map((col) => col.name)
      tableColumns.forEach((col) => {
        columnTypes[col.name] = col.data_type
      })
    } else if (data && data.length > 0) {
      const firstRow = data[0] as Record<string, unknown>
      columnKeys = Object.keys(firstRow)
    } else {
      // No schema and no data - show empty state
      return []
    }

    // Add selection column at the beginning
    const allColumns: ColumnDef<Record<string, unknown>>[] = [
      {
        id: 'select',
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected()}
            onCheckedChange={(value) =>
              table.toggleAllPageRowsSelected(!!value)
            }
            aria-label='Select all'
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label='Select row'
          />
        ),
        enableSorting: false,
        enableHiding: false,
      },
    ]

    const dataColumns: ColumnDef<Record<string, unknown>>[] = columnKeys.map(
      (key) => ({
        accessorKey: key,
        header: ({ column }) => (
          <div className='flex flex-col gap-0.5'>
            <DataTableColumnHeader column={column} title={key} />
            {columnTypes[key] && (
              <span className='text-muted-foreground text-xs font-normal'>
                {columnTypes[key]}
              </span>
            )}
          </div>
        ),
        cell: ({ row }) => {
          const value = row.getValue(key)
          const recordId = row.original.id
          const isIdColumn = key === 'id'

          return (
            <EditableCell
              value={value}
              isReadOnly={isIdColumn}
              onSave={async (newValue) => {
                await updateMutateAsync({
                  id: recordId as string | number,
                  field: key,
                  value: newValue,
                })
              }}
            />
          )
        },
      })
    )

    // Add data columns
    allColumns.push(...dataColumns)

    // Add actions column
    allColumns.push({
      id: 'actions',
      cell: ({ row }) => (
        <TableRowActions
          row={row}
          onEdit={(record) => setEditingRecord(record)}
          onDelete={(record) => deleteMutate(record)}
        />
      ),
    })

    return allColumns
  }, [data, deleteMutate, updateMutateAsync, tableColumns])

  const table = useReactTable<Record<string, unknown>>({
    data: (data || []) as Record<string, unknown>[],
    columns,
    state: {
      sorting,
      pagination,
      globalFilter,
      rowSelection,
    },
    enableRowSelection: true,
    onRowSelectionChange: setRowSelection,
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    onPaginationChange,
    onGlobalFilterChange,
    manualPagination: true,
    pageCount: calculatedPageCount || -1,
    getRowId: (row) => String(row.id), // Use id as row identifier
  })

  const pageCount = table.getPageCount()
  useEffect(() => {
    ensurePageInRange(pageCount)
  }, [pageCount, ensurePageInRange])

  if (isLoading) {
    return (
      <div className='space-y-4 p-6'>
        <Skeleton className='h-8 w-64' />
        <Skeleton className='h-96 w-full' />
      </div>
    )
  }

  const hasData = data && data.length > 0
  const hasColumns = columns.length > 0
  const selectedRows = table.getFilteredSelectedRowModel().rows
  const selectedCount = selectedRows.length

  const handleBulkDelete = () => {
    const selectedIds = selectedRows.map((row) => String(row.original.id))
    if (selectedIds.length === 0) return

    if (
      confirm(
        `Are you sure you want to delete ${selectedIds.length} record${selectedIds.length !== 1 ? 's' : ''}? This action cannot be undone.`
      )
    ) {
      bulkDeleteMutation.mutate(selectedIds)
    }
  }

  return (
    <>
      <div className='flex h-full flex-col gap-4 p-6'>
        <div className='flex items-center justify-between gap-4'>
          <div className='min-w-0 flex-shrink-0'>
            <h2 className='text-2xl font-bold'>{tableName}</h2>
            <p className='text-muted-foreground text-sm'>
              {hasData
                ? `${data.length} record${data.length !== 1 ? 's' : ''}`
                : 'No records'}
              {selectedCount > 0 && ` (${selectedCount} selected)`}
            </p>
          </div>

          {/* Impersonation selector */}
          <div className='flex-1 max-w-md'>
            <div className='flex items-center gap-2'>
              <User className='h-4 w-4 text-muted-foreground shrink-0' />
              {isImpersonating ? (
                <div className={cn(
                  'flex items-center gap-2 px-3 py-1.5 rounded-md text-sm flex-1',
                  'bg-amber-50 border border-amber-200 dark:bg-amber-950 dark:border-amber-800'
                )}>
                  <span className='truncate'>
                    Viewing as: <strong>{getImpersonationLabel()}</strong>
                  </span>
                  <Button
                    variant='ghost'
                    size='sm'
                    className='h-6 w-6 p-0 shrink-0'
                    onClick={handleStopImpersonation}
                    title='Stop impersonation'
                  >
                    <X className='h-3 w-3' />
                  </Button>
                </div>
              ) : (
                <div className='flex items-center gap-2 flex-1'>
                  <div className='flex-1 min-w-[180px]'>
                    <UserSearch
                      onSelect={handleStartUserImpersonation}
                      disabled={isStartingImpersonation}
                    />
                  </div>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={handleStartAnonImpersonation}
                    disabled={isStartingImpersonation}
                    title='Test as anonymous user'
                  >
                    Anon
                  </Button>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={handleStartServiceImpersonation}
                    disabled={isStartingImpersonation}
                    title='Test with service role'
                  >
                    Service
                  </Button>
                </div>
              )}
            </div>
            {isImpersonating && (
              <p className='text-xs text-muted-foreground mt-1 ml-6'>
                {impersonationType === 'service'
                  ? 'Viewing with full service role access (bypasses RLS)'
                  : impersonationType === 'anon'
                    ? 'Viewing as anonymous user (anon key)'
                    : 'Data filtered by this user\'s RLS policies'}
              </p>
            )}
          </div>

          <div className='flex gap-2 shrink-0'>
            {selectedCount > 0 && (
              <Button
                variant='destructive'
                onClick={handleBulkDelete}
                disabled={bulkDeleteMutation.isPending}
              >
                <Trash2 className='mr-2 size-4' />
                Delete {selectedCount}{' '}
                {selectedCount === 1 ? 'record' : 'records'}
              </Button>
            )}
            <Button onClick={() => setIsCreating(true)}>
              <Plus className='mr-2 size-4' />
              Add Record
            </Button>
          </div>
        </div>

        <div className='overflow-hidden rounded-md border'>
          <Table>
            {hasColumns && (
              <TableHeader>
                {table.getHeaderGroups().map((headerGroup) => (
                  <TableRow key={headerGroup.id}>
                    {headerGroup.headers.map((header) => (
                      <TableHead key={header.id} colSpan={header.colSpan}>
                        {header.isPlaceholder
                          ? null
                          : flexRender(
                              header.column.columnDef.header,
                              header.getContext()
                            )}
                      </TableHead>
                    ))}
                  </TableRow>
                ))}
              </TableHeader>
            )}
            <TableBody>
              {isForbidden ? (
                <TableRow>
                  <TableCell
                    colSpan={columns.length || 1}
                    className='h-64 text-center'
                  >
                    <div className='flex flex-col items-center justify-center gap-3'>
                      <ShieldAlert className='h-12 w-12 text-amber-500' />
                      <div>
                        <p className='font-medium text-foreground'>Access Denied</p>
                        <p className='text-muted-foreground text-sm mt-1'>
                          {isImpersonating
                            ? `The ${impersonationType === 'anon' ? 'anonymous user' : impersonationType === 'service' ? 'service role' : 'impersonated user'} does not have permission to view this table.`
                            : 'You do not have permission to view this table.'}
                        </p>
                      </div>
                      {isImpersonating && (
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={handleStopImpersonation}
                        >
                          Stop Impersonation
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ) : hasData ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell
                        key={cell.id}
                        className={cn(cell.column.columnDef.meta?.className)}
                      >
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext()
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell
                    colSpan={columns.length}
                    className='h-64 text-center'
                  >
                    <div className='flex flex-col items-center justify-center gap-2'>
                      <p className='text-muted-foreground'>
                        No records in this table
                      </p>
                      <p className='text-muted-foreground text-xs'>
                        Click "Add Record" to create the first entry
                      </p>
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>

        {hasData && <DataTablePagination table={table} className='mt-auto' />}
      </div>

      <RecordEditDialog
        tableName={tableApiPath}
        tableDisplayName={tableName}
        tableSchema={tableColumns}
        record={editingRecord}
        isOpen={!!editingRecord || isCreating}
        onClose={() => {
          setEditingRecord(null)
          setIsCreating(false)
        }}
        isCreate={isCreating}
      />
    </>
  )
}
