import { useState, useEffect, useMemo } from 'react'
import { formatDistanceToNow } from 'date-fns'
import { createFileRoute } from '@tanstack/react-router'
import {
  type ColumnDef,
  type SortingState,
  type ColumnFiltersState,
  flexRender,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import {
  Bot,
  RefreshCw,
  HardDrive,
  Trash2,
  Settings,
  MessageSquare,
} from 'lucide-react'
import { toast } from 'sonner'
import { chatbotsApi, type AIChatbotSummary } from '@/lib/api'
import { cn } from '@/lib/utils'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { ChatbotSettingsDialog } from '@/components/chatbots/chatbot-settings-dialog'
import { ChatbotTestDialog } from '@/components/chatbots/chatbot-test-dialog'
import {
  DataTablePagination,
  DataTableToolbar,
  DataTableColumnHeader,
} from '@/components/data-table'

export const Route = createFileRoute('/_authenticated/chatbots/')({
  component: ChatbotsPage,
})

function ChatbotsPage() {
  const [chatbots, setChatbots] = useState<AIChatbotSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [reloading, setReloading] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [settingsChatbot, setSettingsChatbot] =
    useState<AIChatbotSummary | null>(null)
  const [testChatbot, setTestChatbot] = useState<AIChatbotSummary | null>(null)

  // Table state
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])

  const fetchChatbots = async () => {
    setLoading(true)
    try {
      const data = await chatbotsApi.list()
      setChatbots(data || [])
    } catch {
      toast.error('Failed to fetch chatbots')
    } finally {
      setLoading(false)
    }
  }

  const handleReloadClick = async () => {
    setReloading(true)
    try {
      const result = await chatbotsApi.sync()
      const { created, updated, deleted, errors } = result.summary

      if (created > 0 || updated > 0 || deleted > 0) {
        const messages = []
        if (created > 0) messages.push(`${created} created`)
        if (updated > 0) messages.push(`${updated} updated`)
        if (deleted > 0) messages.push(`${deleted} deleted`)

        toast.success(`Chatbots synced: ${messages.join(', ')}`)
      } else if (errors > 0) {
        toast.error(`Failed to sync chatbots: ${errors} errors`)
      } else {
        toast.info('No changes detected')
      }

      await fetchChatbots()
    } catch {
      toast.error('Failed to sync chatbots from filesystem')
    } finally {
      setReloading(false)
    }
  }

  const toggleChatbot = async (chatbot: AIChatbotSummary) => {
    const newEnabledState = !chatbot.enabled

    try {
      await chatbotsApi.toggle(chatbot.id, newEnabledState)
      toast.success(`Chatbot ${newEnabledState ? 'enabled' : 'disabled'}`)
      await fetchChatbots()
    } catch {
      toast.error('Failed to toggle chatbot')
    }
  }

  const deleteChatbot = async (id: string) => {
    try {
      await chatbotsApi.delete(id)
      toast.success('Chatbot deleted successfully')
      await fetchChatbots()
    } catch {
      toast.error('Failed to delete chatbot')
    } finally {
      setDeleteConfirm(null)
    }
  }

  // Get unique namespaces for filter options
  const namespaceOptions = useMemo(() => {
    const namespaces = [...new Set(chatbots.map((cb) => cb.namespace))]
    return namespaces.map((ns) => ({ label: ns, value: ns }))
  }, [chatbots])

  // Define columns
  const columns: ColumnDef<AIChatbotSummary>[] = useMemo(
    () => [
      {
        accessorKey: 'name',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title='Name' />
        ),
        cell: ({ row }) => (
          <div className='flex items-center gap-2'>
            <Bot className='text-muted-foreground h-4 w-4 shrink-0' />
            <span className='font-medium'>{row.getValue('name')}</span>
          </div>
        ),
        enableHiding: false,
      },
      {
        accessorKey: 'namespace',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title='Namespace' />
        ),
        cell: ({ row }) => (
          <Badge variant='outline'>{row.getValue('namespace')}</Badge>
        ),
        filterFn: (row, id, value) => {
          return value.includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'version',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title='Version' />
        ),
        cell: ({ row }) => {
          const version = row.getValue('version') as number
          return version > 0 ? (
            <Badge variant='outline' className='text-xs'>
              v{version}
            </Badge>
          ) : (
            <span className='text-muted-foreground'>-</span>
          )
        },
      },
      {
        accessorKey: 'source',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title='Source' />
        ),
        cell: ({ row }) => (
          <Badge variant='secondary' className='text-xs'>
            {row.getValue('source')}
          </Badge>
        ),
      },
      {
        accessorKey: 'enabled',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title='Status' />
        ),
        cell: ({ row }) => (
          <Switch
            checked={row.getValue('enabled')}
            onCheckedChange={() => toggleChatbot(row.original)}
            className='scale-90'
          />
        ),
        enableSorting: false,
      },
      {
        accessorKey: 'updated_at',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title='Updated' />
        ),
        cell: ({ row }) => {
          const updatedAt = row.getValue('updated_at') as string
          return (
            <span className='text-muted-foreground text-sm text-nowrap'>
              {formatDistanceToNow(new Date(updatedAt), { addSuffix: true })}
            </span>
          )
        },
      },
      {
        id: 'actions',
        cell: ({ row }) => (
          <div className='flex items-center justify-end gap-1'>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  onClick={() => setTestChatbot(row.original)}
                  size='sm'
                  variant='ghost'
                  className='h-7 w-7 p-0'
                >
                  <MessageSquare className='h-4 w-4' />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Test chatbot</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  onClick={() => setSettingsChatbot(row.original)}
                  size='sm'
                  variant='ghost'
                  className='h-7 w-7 p-0'
                >
                  <Settings className='h-4 w-4' />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Settings</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  onClick={() => setDeleteConfirm(row.original.id)}
                  size='sm'
                  variant='ghost'
                  className='text-destructive hover:text-destructive hover:bg-destructive/10 h-7 w-7 p-0'
                >
                  <Trash2 className='h-4 w-4' />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Delete chatbot</TooltipContent>
            </Tooltip>
          </div>
        ),
        enableSorting: false,
        enableHiding: false,
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    []
  )

  const table = useReactTable({
    data: chatbots,
    columns,
    state: {
      sorting,
      columnFilters,
    },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  })

  useEffect(() => {
    fetchChatbots()
  }, [])

  if (loading) {
    return (
      <div className='flex h-96 items-center justify-center'>
        <RefreshCw className='text-muted-foreground h-8 w-8 animate-spin' />
      </div>
    )
  }

  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-3xl font-bold'>AI Chatbots</h1>
          <p className='text-muted-foreground'>
            Manage AI-powered chatbots for database interactions
          </p>
        </div>
      </div>

      <div className='flex items-center justify-between'>
        <div className='flex gap-4 text-sm'>
          <div className='flex items-center gap-1.5'>
            <span className='text-muted-foreground'>Total:</span>
            <Badge variant='secondary' className='h-5 px-2'>
              {chatbots.length}
            </Badge>
          </div>
          <div className='flex items-center gap-1.5'>
            <span className='text-muted-foreground'>Active:</span>
            <Badge
              variant='secondary'
              className='h-5 bg-green-500/10 px-2 text-green-600 dark:text-green-400'
            >
              {chatbots.filter((c) => c.enabled).length}
            </Badge>
          </div>
        </div>
        <div className='flex items-center gap-2'>
          <Button
            onClick={handleReloadClick}
            variant='outline'
            size='sm'
            disabled={reloading}
          >
            {reloading ? (
              <>
                <RefreshCw className='mr-2 h-4 w-4 animate-spin' />
                Syncing...
              </>
            ) : (
              <>
                <HardDrive className='mr-2 h-4 w-4' />
                Sync from Filesystem
              </>
            )}
          </Button>
          <Button onClick={() => fetchChatbots()} variant='outline' size='sm'>
            <RefreshCw className='mr-2 h-4 w-4' />
            Refresh
          </Button>
        </div>
      </div>

      {chatbots.length === 0 ? (
        <Card>
          <CardContent className='p-12 text-center'>
            <Bot className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
            <p className='mb-2 text-lg font-medium'>No chatbots yet</p>
            <p className='text-muted-foreground mb-4 text-sm'>
              Create chatbot files in the ./chatbots directory and sync them to
              get started
            </p>
            <Button onClick={handleReloadClick}>
              <HardDrive className='mr-2 h-4 w-4' />
              Sync from Filesystem
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className='flex flex-1 flex-col gap-4'>
          <DataTableToolbar
            table={table}
            searchPlaceholder='Filter by name...'
            searchKey='name'
            filters={[
              {
                columnId: 'namespace',
                title: 'Namespace',
                options: namespaceOptions,
              },
            ]}
          />
          <div className='overflow-hidden rounded-md border'>
            <Table>
              <TableHeader>
                {table.getHeaderGroups().map((headerGroup) => (
                  <TableRow key={headerGroup.id} className='group/row'>
                    {headerGroup.headers.map((header) => (
                      <TableHead
                        key={header.id}
                        colSpan={header.colSpan}
                        className={cn(
                          'bg-background group-hover/row:bg-muted group-data-[state=selected]/row:bg-muted'
                        )}
                      >
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
                {table.getRowModel().rows?.length ? (
                  table.getRowModel().rows.map((row) => (
                    <TableRow
                      key={row.id}
                      data-state={row.getIsSelected() && 'selected'}
                      className='group/row'
                    >
                      {row.getVisibleCells().map((cell) => (
                        <TableCell
                          key={cell.id}
                          className={cn(
                            'bg-background group-hover/row:bg-muted group-data-[state=selected]/row:bg-muted'
                          )}
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
                      className='h-24 text-center'
                    >
                      No results.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
          <DataTablePagination table={table} className='mt-auto' />
        </div>
      )}

      {/* Delete Confirmation Dialog */}
      <AlertDialog
        open={deleteConfirm !== null}
        onOpenChange={(open) => !open && setDeleteConfirm(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Chatbot</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this chatbot? This action cannot
              be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteConfirm && deleteChatbot(deleteConfirm)}
              className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Settings Dialog */}
      {settingsChatbot && (
        <ChatbotSettingsDialog
          chatbot={settingsChatbot}
          open={settingsChatbot !== null}
          onOpenChange={(open) => !open && setSettingsChatbot(null)}
        />
      )}

      {/* Test Dialog */}
      {testChatbot && (
        <ChatbotTestDialog
          chatbot={testChatbot}
          open={testChatbot !== null}
          onOpenChange={(open) => !open && setTestChatbot(null)}
        />
      )}
    </div>
  )
}
