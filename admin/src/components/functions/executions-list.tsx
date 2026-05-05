import { CheckCircle, XCircle, Loader2, Activity, Clock, AlertCircle } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { EdgeFunctionExecution } from "@/lib/api";

interface ExecutionsListProps {
  executions: EdgeFunctionExecution[];
  loading: boolean;
  isInitialLoad: boolean;
  onExecutionClick: (exec: EdgeFunctionExecution) => void;
}

function getStatusIcon(status: string) {
  switch (status) {
    case "success":
      return <CheckCircle className="h-4 w-4 shrink-0 text-green-500" />;
    case "running":
      return <Loader2 className="h-4 w-4 shrink-0 animate-spin text-blue-500" />;
    case "pending":
      return <Clock className="h-4 w-4 shrink-0 text-yellow-500" />;
    case "error":
    case "timeout":
    case "cancelled":
      return <XCircle className="h-4 w-4 shrink-0 text-red-500" />;
    default:
      return <AlertCircle className="h-4 w-4 shrink-0 text-muted-foreground" />;
  }
}

function getStatusVariant(status: string): "secondary" | "destructive" | "outline" {
  switch (status) {
    case "success":
      return "secondary";
    case "error":
    case "timeout":
    case "cancelled":
      return "destructive";
    default:
      return "outline";
  }
}

export function ExecutionsList({
  executions,
  loading,
  isInitialLoad,
  onExecutionClick,
}: ExecutionsListProps) {
  return (
    <ScrollArea className="h-[calc(100vh-28rem)]">
      {loading && isInitialLoad ? (
        <div className="flex h-48 items-center justify-center">
          <Loader2 className="text-muted-foreground h-8 w-8 animate-spin" />
        </div>
      ) : executions.length === 0 ? (
        <Card>
          <CardContent className="p-12 text-center">
            <Activity className="text-muted-foreground mx-auto mb-4 h-12 w-12" />
            <p className="mb-2 text-lg font-medium">No executions found</p>
            <p className="text-muted-foreground text-sm">
              Execute some functions to see their logs here
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-1">
          {loading && !isInitialLoad && (
            <div className="text-muted-foreground flex items-center justify-center py-2">
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              <span className="text-xs">Refreshing...</span>
            </div>
          )}
          {executions.map((exec) => (
            <div
              key={exec.id}
              className="hover:border-primary/50 bg-card flex cursor-pointer items-center justify-between gap-2 rounded-md border px-3 py-2 transition-colors"
              onClick={() => onExecutionClick(exec)}
            >
              <div className="flex min-w-0 flex-1 items-center gap-3">
                {getStatusIcon(exec.status)}
                <span className="truncate text-sm font-medium">
                  {exec.function_name || "Unknown"}
                </span>
                <Badge
                  variant={getStatusVariant(exec.status)}
                  className="h-4 shrink-0 px-1.5 py-0 text-[10px]"
                >
                  {exec.status}
                </Badge>
                {exec.status_code && (
                  <Badge
                    variant="outline"
                    className="h-4 shrink-0 px-1.5 py-0 text-[10px]"
                  >
                    {exec.status_code}
                  </Badge>
                )}
              </div>
              <div className="flex shrink-0 items-center gap-3">
                <span className="text-muted-foreground text-xs">
                  {exec.duration_ms ? `${exec.duration_ms}ms` : "-"}
                </span>
                <span className="text-muted-foreground text-xs">
                  {new Date(exec.executed_at).toLocaleString()}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </ScrollArea>
  );
}
