import { useState, useEffect, useCallback } from "react";
import { formatDistanceToNow } from "date-fns";
import { createFileRoute } from "@tanstack/react-router";
import {
  Database,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  RefreshCw,
  Shield,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { PageHeader } from "@/components/layout/page-header";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { auditLogApi, type AuditLogEntry } from "@/lib/api";
import { useTenantStore } from "@/stores/tenant-store";

const AIAuditPage = () => {
  const [entries, setEntries] = useState<AuditLogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedEntry, setSelectedEntry] = useState<AuditLogEntry | null>(null);
  const [filterSuccess, setFilterSuccess] = useState<boolean | undefined>();
  const currentTenantId = useTenantStore((state) => state.currentTenant?.id);

  const fetchEntries = useCallback(async () => {
    setLoading(true);
    try {
      const data = await auditLogApi.list({
        success: filterSuccess,
        limit: 100,
      });
      setEntries(data.entries || []);
    } catch {
      setEntries([]);
    } finally {
      setLoading(false);
    }
  }, [filterSuccess]);

  useEffect(() => {
    fetchEntries();
  }, [fetchEntries, currentTenantId]);

  return (
    <div className="space-y-6">
      <PageHeader
        icon={<Shield />}
        title="AI Audit Log"
        description="SQL queries generated and executed by AI chatbots"
        actions={
          <>
            <Button
              variant={filterSuccess === undefined ? "default" : "outline"}
              size="sm"
              onClick={() => setFilterSuccess(undefined)}
            >
              All
            </Button>
            <Button
              variant={filterSuccess === true ? "default" : "outline"}
              size="sm"
              onClick={() => setFilterSuccess(true)}
            >
              <CheckCircle2 className="mr-1 h-4 w-4" />
              Success
            </Button>
            <Button
              variant={filterSuccess === false ? "default" : "outline"}
              size="sm"
              onClick={() => setFilterSuccess(false)}
            >
              <XCircle className="mr-1 h-4 w-4" />
              Failed
            </Button>
            <Button variant="outline" size="sm" onClick={fetchEntries}>
              <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            </Button>
          </>
        }
      />

      <div className="px-6">
        {loading ? (
          <div className="text-muted-foreground py-8 text-center">
            Loading audit log...
          </div>
        ) : entries.length === 0 ? (
          <Card>
            <CardContent className="text-muted-foreground py-12 text-center">
              <Database className="mx-auto mb-2 h-8 w-8 opacity-50" />
              No audit log entries found
            </CardContent>
          </Card>
        ) : (
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[100px]">Status</TableHead>
                  <TableHead>SQL Query</TableHead>
                  <TableHead>Chatbot</TableHead>
                  <TableHead>User</TableHead>
                  <TableHead className="w-[120px]">Tables</TableHead>
                  <TableHead className="w-[80px]">Rows</TableHead>
                  <TableHead className="w-[80px]">Duration</TableHead>
                  <TableHead className="w-[120px]">When</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {entries.map((entry) => (
                  <TableRow
                    key={entry.id}
                    className="cursor-pointer hover:bg-accent"
                    onClick={() => setSelectedEntry(entry)}
                  >
                    <TableCell>
                      {entry.success ? (
                        <CheckCircle2 className="h-4 w-4 text-green-500" />
                      ) : (
                        <XCircle className="h-4 w-4 text-red-500" />
                      )}
                    </TableCell>
                    <TableCell className="max-w-[300px]">
                      <code className="truncate block text-xs">
                        {entry.generated_sql.slice(0, 100)}
                        {entry.generated_sql.length > 100 ? "..." : ""}
                      </code>
                    </TableCell>
                    <TableCell className="text-sm">
                      {entry.chatbot_name || entry.chatbot_id?.slice(0, 8)}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-xs">
                      {entry.user_email || entry.user_id?.slice(0, 8) || "—"}
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {(entry.tables_accessed || []).slice(0, 2).map((t) => (
                          <Badge key={t} variant="outline" className="text-xs">
                            {t}
                          </Badge>
                        ))}
                        {(entry.tables_accessed || []).length > 2 && (
                          <Badge variant="outline" className="text-xs">
                            +{(entry.tables_accessed || []).length - 2}
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-sm">
                      {entry.rows_returned ?? "—"}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-xs">
                      {entry.execution_duration_ms
                        ? `${entry.execution_duration_ms}ms`
                        : "—"}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-xs">
                      {formatDistanceToNow(new Date(entry.created_at), {
                        addSuffix: true,
                      })}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      <Dialog
        open={!!selectedEntry}
        onOpenChange={(v) => !v && setSelectedEntry(null)}
      >
        <DialogContent className="max-w-3xl max-h-[80vh]">
          <DialogHeader>
            <DialogTitle>Audit Entry Details</DialogTitle>
          </DialogHeader>
          {selectedEntry && (
            <ScrollArea className="max-h-[60vh] pr-4">
              <div className="space-y-4">
                <div className="flex items-center gap-2">
                  {selectedEntry.success ? (
                    <Badge className="bg-green-500">
                      <CheckCircle2 className="mr-1 h-3 w-3" /> Success
                    </Badge>
                  ) : (
                    <Badge variant="destructive">
                      <XCircle className="mr-1 h-3 w-3" /> Failed
                    </Badge>
                  )}
                  {!selectedEntry.validation_passed && (
                    <Badge variant="destructive">
                      <AlertTriangle className="mr-1 h-3 w-3" /> Validation Failed
                    </Badge>
                  )}
                </div>

                {selectedEntry.generated_sql && (
                  <div>
                    <h4 className="mb-1 text-sm font-medium">Generated SQL</h4>
                    <pre className="bg-muted overflow-x-auto rounded p-3 text-xs">
                      {selectedEntry.generated_sql}
                    </pre>
                  </div>
                )}

                {selectedEntry.error_message && (
                  <div>
                    <h4 className="mb-1 text-sm font-medium text-destructive">
                      Error
                    </h4>
                    <pre className="bg-destructive/10 overflow-x-auto rounded p-3 text-xs text-destructive">
                      {selectedEntry.error_message}
                    </pre>
                  </div>
                )}

                {selectedEntry.validation_errors &&
                  selectedEntry.validation_errors.length > 0 && (
                    <div>
                      <h4 className="mb-1 text-sm font-medium">
                        Validation Errors
                      </h4>
                      <ul className="space-y-1">
                        {selectedEntry.validation_errors.map((err, i) => (
                          <li
                            key={i}
                            className="bg-destructive/10 rounded p-2 text-xs text-destructive"
                          >
                            {err}
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}

                <div className="grid grid-cols-2 gap-4">
                  {selectedEntry.chatbot_name && (
                    <div>
                      <span className="text-muted-foreground text-xs">
                        Chatbot
                      </span>
                      <p className="text-sm">{selectedEntry.chatbot_name}</p>
                    </div>
                  )}
                  {selectedEntry.user_email && (
                    <div>
                      <span className="text-muted-foreground text-xs">User</span>
                      <p className="text-sm">{selectedEntry.user_email}</p>
                    </div>
                  )}
                  {selectedEntry.tables_accessed &&
                    selectedEntry.tables_accessed.length > 0 && (
                      <div>
                        <span className="text-muted-foreground text-xs">
                          Tables Accessed
                        </span>
                        <div className="mt-1 flex flex-wrap gap-1">
                          {selectedEntry.tables_accessed.map((t) => (
                            <Badge key={t} variant="outline" className="text-xs">
                              {t}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    )}
                  {selectedEntry.operations_used &&
                    selectedEntry.operations_used.length > 0 && (
                      <div>
                        <span className="text-muted-foreground text-xs">
                          Operations
                        </span>
                        <div className="mt-1 flex flex-wrap gap-1">
                          {selectedEntry.operations_used.map((op) => (
                            <Badge key={op} variant="secondary" className="text-xs">
                              {op}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    )}
                  {selectedEntry.rows_returned !== undefined && (
                    <div>
                      <span className="text-muted-foreground text-xs">
                        Rows Returned
                      </span>
                      <p className="text-sm">{selectedEntry.rows_returned}</p>
                    </div>
                  )}
                  {selectedEntry.execution_duration_ms && (
                    <div>
                      <span className="text-muted-foreground text-xs">
                        Duration
                      </span>
                      <p className="text-sm">
                        {selectedEntry.execution_duration_ms}ms
                      </p>
                    </div>
                  )}
                  {selectedEntry.ip_address && (
                    <div>
                      <span className="text-muted-foreground text-xs">
                        IP Address
                      </span>
                      <p className="text-sm">{selectedEntry.ip_address}</p>
                    </div>
                  )}
                </div>
              </div>
            </ScrollArea>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
};

export const Route = createFileRoute(
  "/_authenticated/monitoring/ai-audit",
)({
  component: AIAuditPage,
});
