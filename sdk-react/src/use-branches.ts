import { useQuery } from "@tanstack/react-query"
import { useFluxbaseClient } from "./context"

export interface UseBranchesOptions {
  status?: "creating" | "ready" | "migrating" | "error" | "deleting" | "deleted";
  type?: "main" | "preview" | "persistent";
  limit?: number;
  offset?: number;
}

export function useBranches(options?: UseBranchesOptions) {
  const client = useFluxbaseClient()
  return useQuery({
    queryKey: ["fluxbase", "branches", options],
    queryFn: async (): Promise<unknown> => {
      const { data, error } = await client.branching.list(options)
      if (error) throw error
      return data
    },
  })
}
