import { Check, ChevronsUpDown, Database, GitBranch } from "lucide-react";
import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { useBranchStore, type Branch } from "@/stores/branch-store";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { branchesApi } from "@/lib/api";
import { Badge } from "@/components/ui/badge";

export function BranchSelector() {
  const {
    branches,
    currentBranch,
    setCurrentBranch,
    setBranches,
    isBranchingEnabled,
    setIsBranchingEnabled,
  } = useBranchStore();
  const [open, setOpen] = useState(false);

  const query = useQuery({
    queryKey: ["branches", { status: "ready" }],
    queryFn: () => branchesApi.list({ status: "ready" }),
    retry: false,
    refetchOnWindowFocus: false,
  });

  const queryError =
    query.error &&
    ![
      (query.error as { response?: { status?: number } })?.response?.status,
    ].some((s) => s === 401 || s === 403 || s === 404 || s === 503)
      ? "Failed to load branches"
      : null;

  useEffect(() => {
    if (query.data) {
      setBranches(query.data.branches ?? []);
      setIsBranchingEnabled(true);
    }
  }, [query.data, setBranches, setIsBranchingEnabled]);

  useEffect(() => {
    if (query.error) {
      const status = (query.error as { response?: { status?: number } })
        ?.response?.status;
      if (
        status === 401 ||
        status === 403 ||
        status === 404 ||
        status === 503
      ) {
        setIsBranchingEnabled(false);
        setBranches([]);
      } else {
        // eslint-disable-next-line no-console
        console.error("Failed to fetch branches:", query.error);
      }
    }
  }, [query.error, setIsBranchingEnabled, setBranches]);

  const handleSelectBranch = (branch: Branch) => {
    setCurrentBranch(branch);
    setOpen(false);
  };

  const handleSelectMain = () => {
    // Find the main branch in the list
    const mainBranch = branches.find((b) => b.type === "main");
    if (mainBranch) {
      setCurrentBranch(mainBranch);
    } else {
      // Create a synthetic main branch object
      setCurrentBranch({
        id: "main",
        name: "Main Database",
        slug: "main",
        database_name: "main",
        status: "ready",
        type: "main",
        created_at: "",
        updated_at: "",
      });
    }
    setOpen(false);
  };

  // Don't show selector if:
  // 1. Still loading
  // 2. Branching is disabled
  // 3. Only main branch exists (no other branches)
  if (query.isPending || !isBranchingEnabled) {
    return null;
  }

  // Only show main branch and no others - hide selector
  const nonMainBranches = branches.filter((b) => b.type !== "main");
  if (nonMainBranches.length === 0) {
    return null;
  }

  const getBranchTypeLabel = (branch: Branch) => {
    switch (branch.type) {
      case "main":
        return null;
      case "preview":
        return (
          <Badge variant="secondary" className="text-xs">
            Preview
          </Badge>
        );
      case "persistent":
        return (
          <Badge variant="outline" className="text-xs">
            Persistent
          </Badge>
        );
      case "production":
        return (
          <Badge variant="default" className="bg-green-500 text-xs">
            Production
          </Badge>
        );
      default:
        return null;
    }
  };

  const isMainSelected =
    currentBranch?.type === "main" || currentBranch?.slug === "main";

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-label="Select database branch"
          size="sm"
          className={cn(
            "w-[180px] justify-between",
            !currentBranch && "text-muted-foreground",
            !isMainSelected && "border-blue-500 bg-blue-500/10",
          )}
        >
          {!isMainSelected ? (
            <GitBranch className="mr-2 h-4 w-4 text-blue-500" />
          ) : (
            <Database className="mr-2 h-4 w-4" />
          )}
          <span className="truncate">
            {currentBranch?.name || "Select branch..."}
          </span>
          <ChevronsUpDown className="ml-auto h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[220px] p-0">
        <Command>
          <CommandInput placeholder="Search branches..." />
          <CommandList>
            <CommandEmpty>{queryError || "No branches found."}</CommandEmpty>
            <CommandGroup>
              {/* Main database option */}
              <CommandItem onSelect={handleSelectMain}>
                <Check
                  className={cn(
                    "mr-2 h-4 w-4",
                    isMainSelected ? "opacity-100" : "opacity-0",
                  )}
                />
                <div className="flex items-center gap-2">
                  <Database className="h-4 w-4 text-muted-foreground" />
                  <span>Main Database</span>
                </div>
              </CommandItem>

              {/* Other branches */}
              {nonMainBranches.map((branch) => (
                <CommandItem
                  key={branch.id}
                  value={branch.name}
                  onSelect={() => handleSelectBranch(branch)}
                >
                  <Check
                    className={cn(
                      "mr-2 h-4 w-4",
                      currentBranch?.id === branch.id
                        ? "opacity-100"
                        : "opacity-0",
                    )}
                  />
                  <div className="flex flex-col gap-1">
                    <div className="flex items-center gap-2">
                      <GitBranch className="h-3 w-3" />
                      <span className="truncate">{branch.name}</span>
                      {getBranchTypeLabel(branch)}
                    </div>
                    {branch.github_pr_number && (
                      <span className="text-xs text-muted-foreground">
                        PR #{branch.github_pr_number}
                      </span>
                    )}
                  </div>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
