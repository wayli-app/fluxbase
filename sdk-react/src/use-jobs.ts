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
