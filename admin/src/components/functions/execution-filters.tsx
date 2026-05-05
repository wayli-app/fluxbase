import { Search, Filter, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface ExecutionFiltersProps {
  namespaces: string[];
  selectedNamespace: string;
  onNamespaceChange: (namespace: string) => void;
  searchQuery: string;
  onSearchChange: (query: string) => void;
  statusFilter: string;
  onStatusFilterChange: (status: string) => void;
  onRefresh: () => void;
}

export function ExecutionFilters({
  namespaces,
  selectedNamespace,
  onNamespaceChange,
  searchQuery,
  onSearchChange,
  statusFilter,
  onStatusFilterChange,
  onRefresh,
}: ExecutionFiltersProps) {
  return (
    <div className="mb-4 flex items-center gap-3">
      <div className="flex items-center gap-2">
        <Label
          htmlFor="exec-namespace-select"
          className="text-muted-foreground text-sm whitespace-nowrap"
        >
          Namespace:
        </Label>
        <Select
          value={selectedNamespace}
          onValueChange={(value) => {
            onNamespaceChange(value);
          }}
        >
          <SelectTrigger id="exec-namespace-select" className="w-[150px]">
            <SelectValue placeholder="Select namespace" />
          </SelectTrigger>
          <SelectContent>
            {namespaces.map((ns) => (
              <SelectItem key={ns} value={ns}>
                {ns}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="relative max-w-xs flex-1">
        <Search className="text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
        <Input
          placeholder="Search by function name..."
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          className="pl-9"
        />
      </div>
      <Select
        value={statusFilter}
        onValueChange={(value) => {
          onStatusFilterChange(value);
        }}
      >
        <SelectTrigger className="w-[150px]">
          <Filter className="mr-2 h-4 w-4" />
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All Status</SelectItem>
          <SelectItem value="running">Running</SelectItem>
          <SelectItem value="success">Success</SelectItem>
          <SelectItem value="error">Error</SelectItem>
          <SelectItem value="timeout">Timeout</SelectItem>
          <SelectItem value="cancelled">Cancelled</SelectItem>
        </SelectContent>
      </Select>
      <Button onClick={onRefresh} variant="outline" size="sm">
        <RefreshCw className="mr-2 h-4 w-4" />
        Refresh
      </Button>
    </div>
  );
}
