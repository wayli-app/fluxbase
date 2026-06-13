import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useFluxbaseClient } from "./context"

export function useSubmitJob() {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (params: { name: string; payload?: unknown }): Promise<unknown> => {
      const { data, error } = await client.jobs.submit(params.name, params.payload)
      if (error) throw error
      return data
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["fluxbase", "jobs"] }) },
  })
}

export function useJobStatus(jobId: string | null) {
  const client = useFluxbaseClient()
  return useQuery({
    queryKey: ["fluxbase", "jobs", jobId],
    queryFn: async (): Promise<unknown> => {
      if (!jobId) return null
      const { data, error } = await client.jobs.get(jobId)
      if (error) throw error
      return data
    },
    enabled: !!jobId,
    refetchInterval: (query) => {
      const status = (query.state.data as Record<string, unknown> | null)?.status as string | undefined
      return status === "pending" || status === "running" ? 2000 : false
    },
  })
}

export interface UseJobsOptions {
  namespace?: string;
  limit?: number;
  offset?: number;
  status?: string;
}

export function useJobs(options?: UseJobsOptions) {
  const client = useFluxbaseClient()
  return useQuery({
    queryKey: ["fluxbase", "jobs", options],
    queryFn: async (): Promise<unknown> => {
      const { data, error } = await client.jobs.list(options)
      if (error) throw error
      return data
    },
  })
}

export function useCancelJob() {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (jobId: string): Promise<void> => {
      const { error } = await client.jobs.cancel(jobId)
      if (error) throw error
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["fluxbase", "jobs"] })
    },
  })
}

export function useRetryJob() {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (jobId: string): Promise<unknown> => {
      const { data, error } = await client.jobs.retry(jobId)
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["fluxbase", "jobs"] })
    },
  })
}
