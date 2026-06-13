/**
 * Migration hooks for Fluxbase React SDK
 * Provides hooks for listing, applying, rolling back, and syncing database migrations
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type { SyncMigrationsOptions } from "@nimbleflux/fluxbase-sdk";

/**
 * Hook to list migrations in a namespace
 */
export function useMigrations(namespace?: string) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "migrations", namespace],
    queryFn: async () => {
      const { data, error } = await client.admin.migrations.list(namespace);
      if (error) throw error;
      return data || [];
    },
  });
}

/**
 * Hook to apply a specific migration
 */
export function useApplyMigration() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      name,
      namespace,
    }: {
      name: string;
      namespace?: string;
    }) => {
      const { data, error } = await client.admin.migrations.apply(
        name,
        namespace,
      );
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "migrations"],
      });
    },
  });
}

/**
 * Hook to rollback a specific migration
 */
export function useRollbackMigration() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      name,
      namespace,
    }: {
      name: string;
      namespace?: string;
    }) => {
      const { data, error } = await client.admin.migrations.rollback(
        name,
        namespace,
      );
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "migrations"],
      });
    },
  });
}

/**
 * Hook to sync registered migrations
 */
export function useSyncMigrations() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (options: SyncMigrationsOptions | void) => {
      const { data, error } = await client.admin.migrations.sync(
        options ?? undefined,
      );
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "migrations"],
      });
    },
  });
}
