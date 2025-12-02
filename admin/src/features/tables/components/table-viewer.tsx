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
import { Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { apiClient } from '@/lib/api'
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
  const { data, isLoading } = useFluxbaseTable(
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

  // eslint-disable-next-line react-hooks/incompatible-library
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
    pageCount: -1, // Unknown, will use hasNextPage
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
        <div className='flex items-center justify-between'>
          <div>
            <h2 className='text-2xl font-bold'>{tableName}</h2>
            <p className='text-muted-foreground text-sm'>
              {hasData
                ? `${data.length} record${data.length !== 1 ? 's' : ''}`
                : 'No records'}
              {selectedCount > 0 && ` (${selectedCount} selected)`}
            </p>
          </div>
          <div className='flex gap-2'>
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
              {hasData ? (
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
