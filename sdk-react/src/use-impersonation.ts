/**
 * Impersonation hooks for Fluxbase React SDK
 * Provides hooks for managing user impersonation sessions
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type { ListImpersonationSessionsOptions } from "@nimbleflux/fluxbase-sdk";

/**
 * Hook to list impersonation sessions (audit trail)
 */
export function useImpersonationSessions(
  options?: ListImpersonationSessionsOptions,
) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "impersonation", "sessions", options],
    queryFn: async () => {
      return await client.admin.impersonation.listSessions(options);
    },
  });
}

/**
 * Hook to get the current impersonation session
 */
export function useCurrentImpersonation() {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "impersonation", "current"],
    queryFn: async () => {
      return await client.admin.impersonation.getCurrent();
    },
  });
}

/**
 * Hook to start impersonating a specific user
 */
export function useImpersonateUser() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: {
      target_user_id: string;
      reason: string;
    }) => {
      return await client.admin.impersonation.impersonateUser(request);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "impersonation"],
      });
    },
  });
}

/**
 * Hook to stop the current impersonation session
 */
export function useStopImpersonation() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      return await client.admin.impersonation.stop();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "impersonation"],
      });
    },
  });
}
