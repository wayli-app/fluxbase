/**
 * Vector hooks for Fluxbase React SDK
 * Provides hooks for vector embedding and similarity search
 */

import { useMutation } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type { EmbedRequest, VectorSearchOptions } from "@nimbleflux/fluxbase-sdk";

/**
 * Hook to generate embeddings for text
 *
 * @example
 * ```tsx
 * const { mutateAsync } = useVectorEmbed()
 * const { data } = await mutateAsync({ text: 'Hello world' })
 * ```
 */
export function useVectorEmbed() {
  const client = useFluxbaseClient();

  return useMutation({
    mutationFn: async (request: EmbedRequest) => {
      const { data, error } = await client.vector.embed(request);
      if (error) throw error;
      return data;
    },
  });
}

/**
 * Hook for vector similarity search with automatic text embedding
 *
 * @example
 * ```tsx
 * const { mutateAsync } = useVectorSearch()
 * const { data } = await mutateAsync({
 *   table: 'documents',
 *   column: 'embedding',
 *   query: 'How to use TypeScript?',
 *   match_count: 10,
 * })
 * ```
 */
export function useVectorSearch() {
  const client = useFluxbaseClient();

  return useMutation({
    mutationFn: async (options: VectorSearchOptions) => {
      const { data, error } = await client.vector.search(options);
      if (error) throw error;
      return data;
    },
  });
}
