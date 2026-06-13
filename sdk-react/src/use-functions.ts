import { useMutation, useQuery } from "@tanstack/react-query"
import { useFluxbaseClient } from "./context"

export function useInvokeFunction() {
  const client = useFluxbaseClient()
  return useMutation({
    mutationFn: async ({ name, payload, method }: { name: string; payload?: unknown; method?: "GET" | "POST" | "PUT" | "DELETE" | "PATCH" }) => {
      const { data, error } = await client.functions.invoke(name, { body: payload, method })
      if (error) throw error
      return data
    },
  })
}

export function useFunctions() {
  const client = useFluxbaseClient()
  return useQuery({
    queryKey: ["fluxbase", "functions"],
    queryFn: async (): Promise<unknown> => {
      const { data, error } = await client.functions.list()
      if (error) throw error
      return data
    },
  })
}
