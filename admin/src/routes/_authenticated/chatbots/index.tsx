import { useState, useMemo } from "react";
import { formatDistanceToNow } from "date-fns";
import { createFileRoute } from "@tanstack/react-router";
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
} from "@tanstack/react-table";
import {
  Bot,
  RefreshCw,
  HardDrive,
  Trash2,
  Settings,
  MessageSquare,
  History,
} from "lucide-react";
import { type AIChatbotSummary } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ChatbotSettingsDialog } from "@/components/chatbots/chatbot-settings-dialog";
import { ChatbotTestDialog } from "@/components/chatbots/chatbot-test-dialog";
import { ChatbotConversationsDialog } from "@/components/chatbots/chatbot-conversations-dialog";
import { DeleteChatbotDialog } from "@/components/chatbots/delete-chatbot-dialog";
import { useChatbots } from "@/components/chatbots/use-chatbots";
import { PageHeader } from "@/components/layout/page-header";
import {
  DataTablePagination,
  DataTableToolbar,
  DataTableColumnHeader,
} from "@/components/data-table";

interface ChatbotActionsProps {
  onTest: () => void;
  onConversations: () => void;
  onSettings: () => void;
  onDelete: () => void;
}

function ChatbotActions({
  onTest,
  onConversations,
  onSettings,
  onDelete,
}: ChatbotActionsProps) {
  return (
    <div className="flex items-center justify-end gap-1">
      <Tooltip>
        <TooltipTrigger asChild>
          <Button onClick={onTest} size="sm" variant="ghost" className="h-7 w-7 p-0">
            <MessageSquare className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Test chatbot</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button onClick={onConversations} size="sm" variant="ghost" className="h-7 w-7 p-0">
            <History className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>View conversations</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button onClick={onSettings} size="sm" variant="ghost" className="h-7 w-7 p-0">
            <Settings className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Settings</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            onClick={onDelete}
            size="sm"
            variant="ghost"
            className="text-destructive hover:text-destructive hover:bg-destructive/10 h-7 w-7 p-0"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Delete chatbot</TooltipContent>
      </Tooltip>
    </div>
  );
}

function useChatbotColumns(toggle: (chatbot: AIChatbotSummary) => void) {
  return useMemo<ColumnDef<AIChatbotSummary>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Name" />
        ),
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <Bot className="text-muted-foreground h-4 w-4 shrink-0" />
            <span className="font-medium">{row.getValue("name")}</span>
          </div>
        ),
        enableHiding: false,
      },
      {
        accessorKey: "namespace",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Namespace" />
        ),
        cell: ({ row }) => (
          <Badge variant="outline">{row.getValue("namespace")}</Badge>
        ),
        filterFn: (row, id, value) => {
          return value.includes(row.getValue(id));
        },
      },
      {
        accessorKey: "version",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Version" />
        ),
        cell: ({ row }) => {
          const version = row.getValue("version") as number;
          return version > 0 ? (
            <Badge variant="outline" className="text-xs">
              v{version}
            </Badge>
          ) : (
            <span className="text-muted-foreground">-</span>
          );
        },
      },
      {
        accessorKey: "source",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Source" />
        ),
        cell: ({ row }) => (
          <Badge variant="secondary" className="text-xs">
            {row.getValue("source")}
          </Badge>
        ),
      },
      {
        accessorKey: "enabled",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Status" />
        ),
        cell: ({ row }) => (
          <Switch
            checked={row.getValue("enabled")}
            onCheckedChange={() => toggle(row.original)}
            className="scale-90"
          />
        ),
        enableSorting: false,
      },
      {
        accessorKey: "updated_at",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Updated" />
        ),
        cell: ({ row }) => {
          const updatedAt = row.getValue("updated_at") as string;
          return (
            <span className="text-muted-foreground text-sm text-nowrap">
              {formatDistanceToNow(new Date(updatedAt), { addSuffix: true })}
            </span>
          );
        },
      },
      {
        id: "actions",
        cell: ({ row }) => <ChatbotActionsCell chatbot={row.original} />,
        enableSorting: false,
        enableHiding: false,
      },
    ],
    [toggle],
  );
}

const ChatbotsPage = () => {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);

  const {
    chatbots,
    loading,
    reloading,
    sync,
    toggle,
    refetch,
  } = useChatbots();

  const columns = useChatbotColumns(toggle);

  const namespaceOptions = useMemo(() => {
    const namespaces = [...new Set(chatbots.map((cb) => cb.namespace))];
    return namespaces.map((ns) => ({ label: ns, value: ns }));
  }, [chatbots]);

  const table = useReactTable({
    data: chatbots,
    columns,
    state: { sorting, columnFilters },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  });

  if (loading) {
    return (
      <div className="flex h-96 items-center justify-center">
        <RefreshCw className="text-muted-foreground h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        icon={<Bot />}
        title="AI Chatbots"
        description="Manage AI-powered chatbots for database interactions"
      />

      <div className="flex-1 overflow-auto p-6">
        <div className="flex flex-col gap-6">
          <div className="flex items-center justify-between">
            <div className="flex gap-4 text-sm">
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground">Total:</span>
                <Badge variant="secondary" className="h-5 px-2">
                  {chatbots.length}
                </Badge>
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground">Active:</span>
                <Badge
                  variant="secondary"
                  className="h-5 bg-green-500/10 px-2 text-green-600 dark:text-green-400"
                >
                  {chatbots.filter((c) => c.enabled).length}
                </Badge>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Button
                onClick={() => sync()}
                variant="outline"
                size="sm"
                disabled={reloading}
              >
                {reloading ? (
                  <>
                    <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                    Syncing...
                  </>
                ) : (
                  <>
                    <HardDrive className="mr-2 h-4 w-4" />
                    Sync from Filesystem
                  </>
                )}
              </Button>
              <Button onClick={() => refetch()} variant="outline" size="sm">
                <RefreshCw className="mr-2 h-4 w-4" />
                Refresh
              </Button>
            </div>
          </div>

          {chatbots.length === 0 ? (
            <Card>
              <CardContent className="p-12 text-center">
                <Bot className="text-muted-foreground mx-auto mb-4 h-12 w-12" />
                <p className="mb-2 text-lg font-medium">No chatbots yet</p>
                <p className="text-muted-foreground mb-4 text-sm">
                  Create chatbot files and sync them to get started
                </p>
              </CardContent>
            </Card>
          ) : (
            <div className="flex flex-1 flex-col gap-4">
              <DataTableToolbar
                table={table}
                searchPlaceholder="Filter by name..."
                searchKey="name"
                filters={[
                  {
                    columnId: "namespace",
                    title: "Namespace",
                    options: namespaceOptions,
                  },
                ]}
              />
              <div className="overflow-hidden rounded-md border">
                <Table>
                  <TableHeader>
                    {table.getHeaderGroups().map((headerGroup) => (
                      <TableRow key={headerGroup.id} className="group/row">
                        {headerGroup.headers.map((header) => (
                          <TableHead
                            key={header.id}
                            colSpan={header.colSpan}
                            className={cn(
                              "bg-background group-hover/row:bg-muted group-data-[state=selected]/row:bg-muted",
                            )}
                          >
                            {header.isPlaceholder
                              ? null
                              : flexRender(
                                  header.column.columnDef.header,
                                  header.getContext(),
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
                          data-state={row.getIsSelected() && "selected"}
                          className="group/row"
                        >
                          {row.getVisibleCells().map((cell) => (
                            <TableCell
                              key={cell.id}
                              className={cn(
                                "bg-background group-hover/row:bg-muted group-data-[state=selected]/row:bg-muted",
                              )}
                            >
                              {flexRender(
                                cell.column.columnDef.cell,
                                cell.getContext(),
                              )}
                            </TableCell>
                          ))}
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell
                          colSpan={columns.length}
                          className="h-24 text-center"
                        >
                          No results.
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>
              <DataTablePagination table={table} className="mt-auto" />
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

function ChatbotActionsCell({ chatbot }: { chatbot: AIChatbotSummary }) {
  const [settings, setSettings] = useState(false);
  const [test, setTest] = useState(false);
  const [conversations, setConversations] = useState(false);
  const [del, setDel] = useState(false);
  const { deleteChatbot } = useChatbots();

  return (
    <>
      <ChatbotActions
        onTest={() => setTest(true)}
        onConversations={() => setConversations(true)}
        onSettings={() => setSettings(true)}
        onDelete={() => setDel(true)}
      />
      {settings && (
        <ChatbotSettingsDialog
          chatbot={chatbot}
          open={settings}
          onOpenChange={setSettings}
        />
      )}
      {test && (
        <ChatbotTestDialog
          chatbot={chatbot}
          open={test}
          onOpenChange={setTest}
        />
      )}
      {conversations && (
        <ChatbotConversationsDialog
          open={conversations}
          onOpenChange={setConversations}
          chatbotId={chatbot.id}
          chatbotName={chatbot.name}
        />
      )}
      <DeleteChatbotDialog
        open={del}
        onOpenChange={setDel}
        onConfirm={() => {
          deleteChatbot(chatbot.id);
          setDel(false);
        }}
      />
    </>
  );
}

export const Route = createFileRoute("/_authenticated/chatbots/")({
  component: ChatbotsPage,
});
