import { useMutation } from "@tanstack/react-query"
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
