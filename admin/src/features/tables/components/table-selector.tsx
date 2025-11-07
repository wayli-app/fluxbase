import { useState } from 'react'
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query'
import { Database, MoreVertical, Pencil, Trash2, Plus } from 'lucide-react'
import { toast } from 'sonner'
import { databaseApi } from '@/lib/api'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
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
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'

interface TableSelectorProps {
  selectedTable?: string
  selectedSchema: string
  onTableSelect: (table: string) => void
  onSchemaChange: (schema: string) => void
}

export function TableSelector({
  selectedTable,
  selectedSchema,
  onTableSelect,
  onSchemaChange,
}: TableSelectorProps) {
  const queryClient = useQueryClient()
  const [showCreateSchema, setShowCreateSchema] = useState(false)
  const [newSchemaName, setNewSchemaName] = useState('')
  const [showCreateTable, setShowCreateTable] = useState(false)
  const [newTableName, setNewTableName] = useState('')
  const [newTableSchema, setNewTableSchema] = useState(selectedSchema)
  const [columns, setColumns] = useState<
    Array<{
      name: string
      type: string
      nullable: boolean
      primaryKey: boolean
      defaultValue: string
    }>
  >([
    {
      name: 'id',
      type: 'uuid',
      nullable: false,
      primaryKey: true,
      defaultValue: 'gen_random_uuid()',
    },
  ])

  const { data: schemas, isLoading: schemasLoading } = useQuery({
    queryKey: ['schemas'],
    queryFn: databaseApi.getSchemas,
  })

  const { data: tables, isLoading: tablesLoading } = useQuery({
    queryKey: ['tables', selectedSchema],
    queryFn: () => databaseApi.getTables(selectedSchema),
    staleTime: 0, // Always refetch when component mounts
    refetchOnMount: 'always', // Force refetch on mount
  })

  // Create Schema Mutation
  const createSchemaMutation = useMutation({
    mutationFn: (name: string) => databaseApi.createSchema(name),
    onSuccess: (data) => {
      toast.success(data.message)
      queryClient.invalidateQueries({ queryKey: ['schemas'] })
      setShowCreateSchema(false)
      setNewSchemaName('')
      onSchemaChange(data.schema)
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error
          : undefined
      toast.error(errorMessage || 'Failed to create schema')
    },
  })

  // Create Table Mutation
  const createTableMutation = useMutation({
    mutationFn: (data: {
      schema: string
      name: string
      columns: Array<{
        name: string
        type: string
        nullable: boolean
        primaryKey: boolean
        defaultValue: string
      }>
    }) => databaseApi.createTable(data),
    onSuccess: (data) => {
      toast.success(data.message)
      // Invalidate queries for the affected schema
      queryClient.invalidateQueries({ queryKey: ['tables', data.schema] })
      setShowCreateTable(false)
      setNewTableName('')
      setColumns([
        {
          name: 'id',
          type: 'uuid',
          nullable: false,
          primaryKey: true,
          defaultValue: 'gen_random_uuid()',
        },
      ])
      onTableSelect(`${data.schema}.${data.table}`)
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error
          : undefined
      toast.error(errorMessage || 'Failed to create table')
    },
  })

  // Delete Table Mutation
  const deleteTableMutation = useMutation({
    mutationFn: ({ schema, table }: { schema: string; table: string }) =>
      databaseApi.deleteTable(schema, table),
    onSuccess: (data, variables) => {
      toast.success(data.message)
      // If the deleted table is currently selected, clear the selection
      const deletedTableFull = `${variables.schema}.${variables.table}`
      if (selectedTable === deletedTableFull) {
        onTableSelect('')
      }
      // Invalidate queries for the affected schema
      queryClient.invalidateQueries({ queryKey: ['tables', variables.schema] })
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error
          : undefined
      toast.error(errorMessage || 'Failed to delete table')
    },
  })

  const isLoading = schemasLoading || tablesLoading

  if (isLoading) {
    return (
      <div className='space-y-2 p-4'>
        <Skeleton className='h-4 w-32' />
        <Skeleton className='h-10 w-full' />
        {[...Array(5)].map((_, i) => (
          <Skeleton key={i} className='h-9 w-full' />
        ))}
      </div>
    )
  }

  // Map tables to display format (already filtered by schema in API call)
  const filteredTables = (tables || []).map((table) => ({
    full: `${table.schema}.${table.name}`,
    name: table.name,
  }))

  return (
    <div className='flex h-full flex-col border-r'>
      <div className='border-b p-4'>
        <h2 className='mb-3 flex items-center gap-2 text-lg font-semibold'>
          <Database className='size-5' />
          Tables
        </h2>
        <div className='flex gap-2'>
          <Select value={selectedSchema} onValueChange={onSchemaChange}>
            <SelectTrigger className='flex-1'>
              <SelectValue placeholder='Select schema' />
            </SelectTrigger>
            <SelectContent>
              {(schemas || []).map((schema) => (
                <SelectItem key={schema} value={schema}>
                  {schema}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            variant='outline'
            size='icon'
            onClick={() => setShowCreateSchema(true)}
            title='Create Schema'
          >
            <Plus className='h-4 w-4' />
          </Button>
        </div>
      </div>
      <ScrollArea className='flex-1'>
        <div className='p-2'>
          <Button
            variant='outline'
            className='mb-2 w-full'
            onClick={() => {
              setNewTableSchema(selectedSchema)
              setShowCreateTable(true)
            }}
          >
            <Plus className='mr-2 h-4 w-4' />
            Create Table
          </Button>
          <div className='space-y-1'>
            {filteredTables.map(({ full, name }) => (
              <div key={full} className='group relative flex items-center'>
                <Button
                  variant={selectedTable === full ? 'secondary' : 'ghost'}
                  className={cn(
                    'flex-1 justify-start pr-8 font-normal',
                    selectedTable === full && 'bg-secondary'
                  )}
                  onClick={() => onTableSelect(full)}
                >
                  {name}
                </Button>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant='ghost'
                      size='sm'
                      className='absolute right-1 h-7 w-7 p-0 opacity-0 group-hover:opacity-100'
                      onClick={(e) => e.stopPropagation()}
                    >
                      <MoreVertical className='h-4 w-4' />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align='end'>
                    <DropdownMenuItem
                      onClick={(e) => {
                        e.stopPropagation()
                        toast.info('Table editing UI coming soon')
                      }}
                    >
                      <Pencil className='mr-2 h-4 w-4' />
                      Edit Table
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      className='text-destructive'
                      onClick={(e) => {
                        e.stopPropagation()
                        const [schema, table] = full.split('.')
                        if (
                          confirm(
                            `Are you sure you want to delete table "${full}"? This action cannot be undone.`
                          )
                        ) {
                          deleteTableMutation.mutate({ schema, table })
                        }
                      }}
                    >
                      <Trash2 className='mr-2 h-4 w-4' />
                      Delete Table
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            ))}
          </div>
        </div>
      </ScrollArea>

      {/* Create Schema Dialog */}
      <Dialog open={showCreateSchema} onOpenChange={setShowCreateSchema}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Schema</DialogTitle>
            <DialogDescription>
              Create a new database schema to organize your tables.
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid gap-2'>
              <Label htmlFor='schemaName'>Schema Name</Label>
              <Input
                id='schemaName'
                placeholder='e.g., my_schema'
                value={newSchemaName}
                onChange={(e) => setNewSchemaName(e.target.value)}
                autoFocus
              />
              <p className='text-muted-foreground text-xs'>
                Must start with a letter and contain only letters, numbers, and
                underscores.
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => {
                setShowCreateSchema(false)
                setNewSchemaName('')
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={() => {
                if (!newSchemaName.trim()) {
                  toast.error('Please enter a schema name')
                  return
                }
                createSchemaMutation.mutate(newSchemaName)
              }}
              disabled={createSchemaMutation.isPending}
            >
              {createSchemaMutation.isPending ? 'Creating...' : 'Create Schema'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Create Table Sheet */}
      <Sheet open={showCreateTable} onOpenChange={setShowCreateTable}>
        <SheetContent className='w-full overflow-y-auto px-8 sm:max-w-2xl'>
          <SheetHeader>
            <SheetTitle>Create New Table</SheetTitle>
            <SheetDescription>
              Define a new table with columns, types, and constraints.
            </SheetDescription>
          </SheetHeader>

          <div className='flex flex-col gap-6 py-6'>
            {/* Table Name and Schema */}
            <div className='space-y-4'>
              <div className='space-y-2'>
                <Label htmlFor='tableName'>
                  Table Name <span className='text-destructive'>*</span>
                </Label>
                <Input
                  id='tableName'
                  placeholder='e.g., users, products, orders'
                  value={newTableName}
                  onChange={(e) => setNewTableName(e.target.value)}
                  autoFocus
                />
                <p className='text-muted-foreground text-xs'>
                  Use lowercase with underscores (snake_case)
                </p>
              </div>

              <div className='space-y-2'>
                <Label htmlFor='tableSchema'>Schema</Label>
                <Select
                  value={newTableSchema}
                  onValueChange={setNewTableSchema}
                >
                  <SelectTrigger id='tableSchema'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {(schemas || []).map((schema) => (
                      <SelectItem key={schema} value={schema}>
                        {schema}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className='text-muted-foreground text-xs'>
                  The schema where this table will be created
                </p>
              </div>
            </div>

            {/* Columns Section */}
            <div className='space-y-4'>
              <div className='flex items-center justify-between'>
                <div>
                  <Label className='text-base'>Columns</Label>
                  <p className='text-muted-foreground mt-1 text-xs'>
                    Define the structure of your table
                  </p>
                </div>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => {
                    setColumns([
                      ...columns,
                      {
                        name: '',
                        type: 'text',
                        nullable: true,
                        primaryKey: false,
                        defaultValue: '',
                      },
                    ])
                  }}
                >
                  <Plus className='mr-2 h-4 w-4' />
                  Add Column
                </Button>
              </div>

              <div className='space-y-4'>
                {columns.map((column, index) => (
                  <div
                    key={index}
                    className='bg-muted/30 relative space-y-4 rounded-lg border p-4'
                  >
                    {/* Delete button in top-right */}
                    <Button
                      type='button'
                      variant='ghost'
                      size='sm'
                      className='absolute top-2 right-2 h-8 w-8 p-0'
                      onClick={() => {
                        setColumns(columns.filter((_, i) => i !== index))
                      }}
                      disabled={columns.length === 1}
                      title='Remove column'
                    >
                      <Trash2 className='h-4 w-4' />
                    </Button>

                    {/* Column Name */}
                    <div className='space-y-2 pr-10'>
                      <Label htmlFor={`column-name-${index}`}>
                        Column Name <span className='text-destructive'>*</span>
                      </Label>
                      <Input
                        id={`column-name-${index}`}
                        placeholder='e.g., email, created_at, user_id'
                        value={column.name}
                        onChange={(e) => {
                          const newColumns = [...columns]
                          newColumns[index].name = e.target.value
                          setColumns(newColumns)
                        }}
                      />
                    </div>

                    {/* Data Type */}
                    <div className='space-y-2'>
                      <Label htmlFor={`column-type-${index}`}>Data Type</Label>
                      <Select
                        value={column.type}
                        onValueChange={(value) => {
                          const newColumns = [...columns]
                          newColumns[index].type = value
                          setColumns(newColumns)
                        }}
                      >
                        <SelectTrigger id={`column-type-${index}`}>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value='text'>
                            text - Unlimited length text
                          </SelectItem>
                          <SelectItem value='varchar'>
                            varchar - Variable length text
                          </SelectItem>
                          <SelectItem value='integer'>
                            integer - Whole numbers
                          </SelectItem>
                          <SelectItem value='bigint'>
                            bigint - Large whole numbers
                          </SelectItem>
                          <SelectItem value='uuid'>
                            uuid - Unique identifier
                          </SelectItem>
                          <SelectItem value='boolean'>
                            boolean - True/false
                          </SelectItem>
                          <SelectItem value='timestamp'>
                            timestamp - Date and time
                          </SelectItem>
                          <SelectItem value='timestamptz'>
                            timestamptz - Date and time with timezone
                          </SelectItem>
                          <SelectItem value='date'>date - Date only</SelectItem>
                          <SelectItem value='jsonb'>
                            jsonb - JSON data
                          </SelectItem>
                          <SelectItem value='numeric'>
                            numeric - Precise decimal numbers
                          </SelectItem>
                        </SelectContent>
                      </Select>
                      <p className='text-muted-foreground text-xs'>
                        The PostgreSQL data type for this column
                      </p>
                    </div>

                    {/* Default Value */}
                    <div className='space-y-2'>
                      <Label htmlFor={`column-default-${index}`}>
                        Default Value
                      </Label>
                      <Input
                        id={`column-default-${index}`}
                        placeholder='e.g., gen_random_uuid(), now(), 0'
                        value={column.defaultValue}
                        onChange={(e) => {
                          const newColumns = [...columns]
                          newColumns[index].defaultValue = e.target.value
                          setColumns(newColumns)
                        }}
                      />
                      <p className='text-muted-foreground text-xs'>
                        Optional default value or function
                      </p>
                    </div>

                    {/* Constraints */}
                    <div className='space-y-3'>
                      <Label className='text-sm'>Constraints</Label>
                      <div className='flex flex-col gap-2'>
                        <div className='flex items-center gap-2'>
                          <Checkbox
                            id={`column-pk-${index}`}
                            checked={column.primaryKey}
                            onCheckedChange={(checked) => {
                              const newColumns = [...columns]
                              newColumns[index].primaryKey = checked === true
                              if (checked) {
                                newColumns[index].nullable = false
                              }
                              setColumns(newColumns)
                            }}
                          />
                          <Label
                            htmlFor={`column-pk-${index}`}
                            className='cursor-pointer text-sm font-normal'
                          >
                            Primary Key - Unique identifier for each row
                          </Label>
                        </div>
                        <div className='flex items-center gap-2'>
                          <Checkbox
                            id={`column-nullable-${index}`}
                            checked={column.nullable}
                            disabled={column.primaryKey}
                            onCheckedChange={(checked) => {
                              const newColumns = [...columns]
                              newColumns[index].nullable = checked === true
                              setColumns(newColumns)
                            }}
                          />
                          <Label
                            htmlFor={`column-nullable-${index}`}
                            className={cn(
                              'cursor-pointer text-sm font-normal',
                              column.primaryKey && 'opacity-50'
                            )}
                          >
                            Nullable - Allow NULL values
                          </Label>
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <SheetFooter className='flex flex-row gap-2 border-t pt-4'>
            <Button
              variant='outline'
              onClick={() => {
                setShowCreateTable(false)
                setNewTableName('')
                setColumns([
                  {
                    name: 'id',
                    type: 'uuid',
                    nullable: false,
                    primaryKey: true,
                    defaultValue: 'gen_random_uuid()',
                  },
                ])
              }}
              className='flex-1'
            >
              Cancel
            </Button>
            <Button
              onClick={() => {
                if (!newTableName.trim()) {
                  toast.error('Please enter a table name')
                  return
                }
                if (columns.length === 0) {
                  toast.error('Please add at least one column')
                  return
                }
                const hasInvalidColumn = columns.some((c) => !c.name.trim())
                if (hasInvalidColumn) {
                  toast.error('All columns must have a name')
                  return
                }
                createTableMutation.mutate({
                  schema: newTableSchema,
                  name: newTableName,
                  columns,
                })
              }}
              disabled={createTableMutation.isPending}
              className='flex-1'
            >
              {createTableMutation.isPending ? 'Creating...' : 'Create Table'}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  )
}
