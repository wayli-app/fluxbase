import { Copy, Loader2 } from "lucide-react";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { EdgeFunctionExecution } from "@/lib/api";
import type { ExecutionLog } from "@/hooks/use-execution-logs";

interface ExecutionDetailDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  selectedExecution: EdgeFunctionExecution | null;
  executionLogs: ExecutionLog[];
  executionLogsLoading: boolean;
  logLevelFilter: string;
  onLogLevelFilterChange: (level: string) => void;
  filteredLogs: ExecutionLog[];
}

export function ExecutionDetailDialog({
  open,
  onOpenChange,
  selectedExecution,
  executionLogsLoading,
  logLevelFilter,
  onLogLevelFilterChange,
  filteredLogs,
}: ExecutionDetailDialogProps) {
  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    toast.success(`${label} copied to clipboard`);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[90vh] w-[90vw] max-w-[1600px] flex-col overflow-hidden sm:max-w-none">
        <DialogHeader>
          <DialogTitle>Execution Details</DialogTitle>
          <DialogDescription>
            {selectedExecution?.function_name || "Unknown Function"} -{" "}
            {selectedExecution?.id?.slice(0, 8)}
          </DialogDescription>
        </DialogHeader>

        {selectedExecution && (
          <div className="flex flex-1 flex-col gap-4 overflow-y-auto pr-2">
            <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
              <div>
                <Label className="text-muted-foreground text-xs">Status</Label>
                <div className="mt-1">
                  <Badge
                    variant={
                      selectedExecution.status === "success"
                        ? "secondary"
                        : selectedExecution.status === "error" ||
                            selectedExecution.status === "timeout" ||
                            selectedExecution.status === "cancelled"
                          ? "destructive"
                          : "outline"
                    }
                  >
                    {selectedExecution.status}
                  </Badge>
                </div>
              </div>
              <div>
                <Label className="text-muted-foreground text-xs">
                  Status Code
                </Label>
                <p className="font-mono text-sm">
                  {selectedExecution.status_code ?? "-"}
                </p>
              </div>
              <div>
                <Label className="text-muted-foreground text-xs">
                  Duration
                </Label>
                <p className="font-mono text-sm">
                  {selectedExecution.duration_ms
                    ? `${selectedExecution.duration_ms}ms`
                    : "-"}
                </p>
              </div>
              <div>
                <Label className="text-muted-foreground text-xs">Trigger</Label>
                <p className="font-mono text-sm">
                  {selectedExecution.trigger_type}
                </p>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label className="text-muted-foreground text-xs">Started</Label>
                <p className="font-mono text-sm">
                  {new Date(selectedExecution.executed_at).toLocaleString()}
                </p>
              </div>
              <div>
                <Label className="text-muted-foreground text-xs">
                  Completed
                </Label>
                <p className="font-mono text-sm">
                  {selectedExecution.completed_at
                    ? new Date(selectedExecution.completed_at).toLocaleString()
                    : "-"}
                </p>
              </div>
            </div>

            {selectedExecution.result && (
              <div>
                <div className="mb-2 flex items-center justify-between">
                  <Label>Result</Label>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() =>
                      copyToClipboard(selectedExecution.result || "", "Result")
                    }
                  >
                    <Copy className="h-3 w-3" />
                  </Button>
                </div>
                <pre className="bg-muted max-h-32 overflow-auto rounded-md p-3 text-xs">
                  {selectedExecution.result}
                </pre>
              </div>
            )}

            {selectedExecution.error_message && (
              <div>
                <Label className="text-destructive">Error</Label>
                <pre className="bg-destructive/10 text-destructive mt-2 max-h-32 overflow-auto rounded-md p-3 text-xs">
                  {selectedExecution.error_message}
                </pre>
              </div>
            )}

            <div className="min-h-0 flex-1">
              <div className="mb-2 flex items-center justify-between">
                <Label>Logs</Label>
                <div className="flex items-center gap-2">
                  <Select
                    value={logLevelFilter}
                    onValueChange={onLogLevelFilterChange}
                  >
                    <SelectTrigger className="h-8 w-[120px] text-xs">
                      <SelectValue placeholder="Filter level" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Levels</SelectItem>
                      <SelectItem value="debug">Debug</SelectItem>
                      <SelectItem value="info">Info</SelectItem>
                      <SelectItem value="warn">Warn</SelectItem>
                      <SelectItem value="error">Error</SelectItem>
                    </SelectContent>
                  </Select>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      const logsText =
                        filteredLogs.length > 0
                          ? filteredLogs
                              .map(
                                (l) =>
                                  `[${l.level.toUpperCase()}] ${l.message}`,
                              )
                              .join("\n")
                          : selectedExecution.logs || "";
                      copyToClipboard(logsText, "Logs");
                    }}
                  >
                    <Copy className="h-3 w-3" />
                  </Button>
                </div>
              </div>

              {executionLogsLoading ? (
                <div className="flex h-48 items-center justify-center">
                  <Loader2 className="text-muted-foreground h-6 w-6 animate-spin" />
                </div>
              ) : filteredLogs.length > 0 ? (
                <ScrollArea className="bg-muted h-64 rounded-md">
                  <div className="space-y-1 p-3">
                    {filteredLogs.map((log) => (
                      <div
                        key={log.id}
                        className="flex gap-2 font-mono text-xs"
                      >
                        <Badge
                          variant={
                            log.level === "error"
                              ? "destructive"
                              : log.level === "warn"
                                ? "secondary"
                                : log.level === "debug"
                                  ? "outline"
                                  : "default"
                          }
                          className="h-4 shrink-0 px-1 py-0 text-[10px]"
                        >
                          {log.level.toUpperCase()}
                        </Badge>
                        <span className="break-all">{log.message}</span>
                      </div>
                    ))}
                  </div>
                </ScrollArea>
              ) : selectedExecution.logs ? (
                <pre className="bg-muted h-64 overflow-auto rounded-md p-3 text-xs">
                  {selectedExecution.logs}
                </pre>
              ) : (
                <div className="text-muted-foreground flex h-48 items-center justify-center text-sm">
                  No logs available
                </div>
              )}
            </div>
          </div>
        )}

        <DialogFooter className="mt-4">
          <Button onClick={() => onOpenChange(false)}>Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
