/**
 * Service keys hooks for Fluxbase React SDK
 * Provides hooks for managing tenant service keys (anon and service)
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type {
  CreateServiceKeyRequest,
  RevokeServiceKeyRequest,
} from "@nimbleflux/fluxbase-sdk";

/**
 * Hook to list all service keys
 */
export function useServiceKeys() {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "service-keys"],
    queryFn: async () => {
      const { data, error } = await client.admin.serviceKeys.list();
      if (error) throw error;
      return data;
    },
  });
}

/**
 * Hook to create a new service key
 *
 * The full key value is only returned once — store it securely!
 */
export function useCreateServiceKey() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: CreateServiceKeyRequest) => {
      const { data, error } = await client.admin.serviceKeys.create(request);
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["fluxbase", "service-keys"] });
    },
  });
}

/**
 * Hook to rotate a service key (creates a replacement and deprecates the old one)
 */
export function useRotateServiceKey() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      const { data, error } = await client.admin.serviceKeys.rotate(id);
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["fluxbase", "service-keys"] });
    },
  });
}

/**
 * Hook to revoke a service key permanently (emergency)
 */
export function useRevokeServiceKey() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: {
      id: string;
      request?: RevokeServiceKeyRequest;
    }) => {
      const { error } = await client.admin.serviceKeys.revoke(
        params.id,
        params.request,
      );
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["fluxbase", "service-keys"] });
    },
  });
}
