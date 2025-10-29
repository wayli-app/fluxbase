import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTable as useFluxbaseTable, useUpdate, useDelete } from '@fluxbase/sdk-react'
import { getRouteApi } from '@tanstack/react-router'
import {
  type ColumnDef,
  type SortingState,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { Plus } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { DataTablePagination } from '@/components/data-table'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from 'sonner'
import { DataTableColumnHeader } from '@/components/data-table'
import { TableRowActions } from './table-row-actions'
import { RecordEditDialog } from './record-edit-dialog'
import { EditableCell } from './editable-cell'

const route = getRouteApi('/_authenticated/tables/')

interface TableViewerProps {
  tableName: string
}

export function TableViewer({ tableName }: TableViewerProps) {
  const queryClient = useQueryClient()
  const [sorting, setSorting] = useState<SortingState>([])
  const [editingRecord, setEditingRecord] = useState<Record<
    string,
    unknown
  > | null>(null)
  const [isCreating, setIsCreating] = useState(false)

  // For REST API, convert schema.table to the appropriate path
  // Backend uses: /products for public.products, /auth/users for auth.users
  const tableApiPath = tableName.includes('.')
    ? (() => {
        const [schema, name] = tableName.split('.')
        return schema === 'public' ? name : `${schema}/${name}`
      })()
    : tableName

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
      let q = query.select('*').limit(pagination.pageSize).offset(pagination.pageIndex * pagination.pageSize)

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
    }) => updateFluxbase.mutateAsync({
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

  // Generate columns dynamically
  const columns = useMemo<ColumnDef<Record<string, unknown>>[]>(() => {
    if (!data || data.length === 0) return []

    const firstRow = data[0] as Record<string, unknown>
    const columnKeys = Object.keys(firstRow)

    const dataColumns: ColumnDef<Record<string, unknown>>[] = columnKeys.map(
      (key) => ({
        accessorKey: key,
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={key} />
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
                await updateMutation.mutateAsync({
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

    // Add actions column
    dataColumns.push({
      id: 'actions',
      cell: ({ row }) => (
        <TableRowActions
          row={row}
          onEdit={(record) => setEditingRecord(record)}
          onDelete={(record) => deleteMutation.mutate(record)}
        />
      ),
    })

    return dataColumns
  }, [data, deleteMutation])

  const table = useReactTable<Record<string, unknown>>({
    data: (data || []) as Record<string, unknown>[],
    columns,
    state: {
      sorting,
      pagination,
      globalFilter,
    },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    onPaginationChange,
    onGlobalFilterChange,
    manualPagination: true,
    pageCount: -1, // Unknown, will use hasNextPage
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

  if (!data || data.length === 0) {
    return (
      <div className='flex h-full flex-col items-center justify-center p-6'>
        <p className='text-muted-foreground mb-4'>No records found in this table</p>
        <Button onClick={() => setIsCreating(true)}>
          <Plus className='mr-2 size-4' />
          Add Record
        </Button>
      </div>
    )
  }

  return (
    <>
      <div className='flex h-full flex-col gap-4 p-6'>
        <div className='flex items-center justify-between'>
          <div>
            <h2 className='text-2xl font-bold'>{tableName}</h2>
            <p className='text-muted-foreground text-sm'>
              {data.length} record{data.length !== 1 ? 's' : ''}
            </p>
          </div>
          <Button onClick={() => setIsCreating(true)}>
            <Plus className='mr-2 size-4' />
            Add Record
          </Button>
        </div>

        <div className='overflow-hidden rounded-md border'>
          <Table>
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
            <TableBody>
              {table.getRowModel().rows.map((row) => (
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
              ))}
            </TableBody>
          </Table>
        </div>

        <DataTablePagination table={table} className='mt-auto' />
      </div>

      <RecordEditDialog
        tableName={tableApiPath}
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
