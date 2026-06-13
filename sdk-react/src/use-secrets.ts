/**
 * Secrets hooks for Fluxbase React SDK
 * Provides hooks for managing edge function and job secrets
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type {
  ListSecretsOptions,
  CreateSecretRequest,
  UpdateSecretRequest,
} from "@nimbleflux/fluxbase-sdk";

/**
 * Hook to list all secrets (metadata only, never includes values)
 */
export function useSecrets(options?: ListSecretsOptions) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "secrets", options],
    queryFn: async () => {
      return await client.secrets.list(options);
    },
  });
}

/**
 * Hook to create a new secret
 */
export function useCreateSecret() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: CreateSecretRequest) => {
      return await client.secrets.create(request);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["fluxbase", "secrets"] });
    },
  });
}

/**
 * Hook to update a secret by name
 */
export function useUpdateSecret() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: {
      name: string;
      request: UpdateSecretRequest;
      namespace?: string;
    }) => {
      return await client.secrets.update(params.name, params.request, {
        namespace: params.namespace,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["fluxbase", "secrets"] });
    },
  });
}

/**
 * Hook to delete a secret by name
 */
export function useDeleteSecret() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { name: string; namespace?: string }) => {
      await client.secrets.delete(params.name, {
        namespace: params.namespace,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["fluxbase", "secrets"] });
    },
  });
}
