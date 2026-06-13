import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useFluxbaseClient } from "./context"

export interface UseCreateBranchParams {
  name: string;
  parentBranchId?: string;
  dataCloneMode?: "schema_only" | "full_clone" | "seed_data";
  type?: "main" | "preview" | "persistent";
  githubPRNumber?: number;
  githubPRUrl?: string;
  githubRepo?: string;
  expiresIn?: string;
}

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

export function useCreateBranch() {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (params: UseCreateBranchParams): Promise<unknown> => {
      const { name, ...options } = params
      const { data, error } = await client.branching.create(name, options)
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["fluxbase", "branches"] })
    },
  })
}

export function useDeleteBranch() {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (slug: string): Promise<void> => {
      const { error } = await client.branching.delete(slug)
      if (error) throw error
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["fluxbase", "branches"] })
    },
  })
}

export function useResetBranch() {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (slug: string): Promise<unknown> => {
      const { data, error } = await client.branching.reset(slug)
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["fluxbase", "branches"] })
    },
  })
}
