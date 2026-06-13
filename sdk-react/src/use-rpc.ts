/**
 * RPC hooks for Fluxbase React SDK
 * Provides hooks for listing and invoking RPC procedures
 */

import { useQuery, useMutation } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";

interface RPCInvokeOptions {
  namespace?: string;
  async?: boolean;
  timeout?: number;
}

/**
 * Hook to list available RPC procedures
 */
export function useRPCList(namespace?: string) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "rpc", namespace],
    queryFn: async () => {
      const { data, error } = await client.rpc.list(namespace);
      if (error) throw error;
      return data;
    },
  });
}

/**
 * Hook to invoke an RPC procedure
 *
 * @example
 * ```tsx
 * const { mutateAsync, data } = useInvokeRPC()
 * await mutateAsync({ name: 'get-user-orders', payload: { user_id: '123' } })
 * ```
 */
export function useInvokeRPC() {
  const client = useFluxbaseClient();

  return useMutation({
    mutationFn: async (params: {
      name: string;
      payload?: Record<string, unknown>;
      options?: RPCInvokeOptions;
    }) => {
      const { data, error } = await client.rpc.invoke(
        params.name,
        params.payload,
        params.options,
      );
      if (error) throw error;
      return data;
    },
  });
}
