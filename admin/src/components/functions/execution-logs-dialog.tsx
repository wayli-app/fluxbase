import { History } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Card, CardHeader, CardContent } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import type { EdgeFunction, EdgeFunctionExecution } from "@/lib/api";

interface ExecutionLogsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  selectedFunction: EdgeFunction | null;
  executions: EdgeFunctionExecution[];
  wordWrap: boolean;
  onWordWrapChange: (wrap: boolean) => void;
}

export function ExecutionLogsDialog({
  open,
  onOpenChange,
  selectedFunction,
  executions,
  wordWrap,
  onWordWrapChange,
}: ExecutionLogsDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[95vh] !w-[95vw] !max-w-[95vw] flex-col overflow-hidden">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle>Execution Logs</DialogTitle>
          <DialogDescription>
            Recent executions for {selectedFunction?.name}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-shrink-0 items-center space-x-2">
          <Switch
            id="logs-word-wrap"
            checked={wordWrap}
            onCheckedChange={onWordWrapChange}
          />
          <Label htmlFor="logs-word-wrap" className="cursor-pointer">
            Word wrap
          </Label>
        </div>

        <div className="min-h-0 flex-1 space-y-3 overflow-auto rounded-lg border p-4">
          {executions.length === 0 ? (
            <div className="text-muted-foreground py-12 text-center">
              <History className="mx-auto mb-4 h-12 w-12 opacity-50" />
              <p>No executions yet</p>
            </div>
          ) : (
            executions.map((exec) => (
              <Card key={exec.id} className="overflow-hidden">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="mb-1 flex items-center gap-2">
                        <Badge
                          variant={
                            exec.status === "success"
                              ? "default"
                              : exec.status === "error" ||
                                  exec.status === "timeout" ||
                                  exec.status === "cancelled"
                                ? "destructive"
                                : "outline"
                          }
                        >
                          {exec.status}
                        </Badge>
                        <Badge variant="outline">{exec.trigger_type}</Badge>
                        {exec.status_code && (
                          <Badge variant="secondary">{exec.status_code}</Badge>
                        )}
                        {exec.duration_ms && (
                          <span className="text-muted-foreground text-xs">
                            {exec.duration_ms}ms
                          </span>
                        )}
                      </div>
                      <p className="text-muted-foreground text-xs">
                        {new Date(exec.executed_at).toLocaleString()}
                      </p>
                    </div>
                  </div>
                </CardHeader>
                {(exec.logs || exec.error_message || exec.result) && (
                  <CardContent className="overflow-hidden pt-0">
                    {exec.error_message && (
                      <div className="mb-2 min-w-0">
                        <Label className="text-destructive text-xs">
                          Error:
                        </Label>
                        <div className="bg-destructive/10 mt-1 max-h-40 max-w-full overflow-auto rounded border">
                          <pre
                            className={`min-w-0 p-2 text-xs ${wordWrap ? "break-words whitespace-pre-wrap" : "whitespace-pre"}`}
                          >
                            {exec.error_message}
                          </pre>
                        </div>
                      </div>
                    )}
                    {exec.logs && (
                      <div className="mb-2 min-w-0">
                        <Label className="text-xs">Logs:</Label>
                        <div className="bg-muted mt-1 max-h-40 max-w-full overflow-auto rounded border">
                          <pre
                            className={`min-w-0 p-2 text-xs ${wordWrap ? "break-words whitespace-pre-wrap" : "whitespace-pre"}`}
                          >
                            {exec.logs}
                          </pre>
                        </div>
                      </div>
                    )}
                    {exec.result && !exec.error_message && (
                      <div className="min-w-0">
                        <Label className="text-xs">Result:</Label>
                        <div className="bg-muted mt-1 max-h-40 max-w-full overflow-auto rounded border">
                          <pre
                            className={`min-w-0 p-2 text-xs ${wordWrap ? "break-words whitespace-pre-wrap" : "whitespace-pre"}`}
                          >
                            {exec.result}
                          </pre>
                        </div>
                      </div>
                    )}
                  </CardContent>
                )}
              </Card>
            ))
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
