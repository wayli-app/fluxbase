/**
 * React hooks for multi-tenant management
 *
 * @module use-tenant
 */

import { useState, useCallback } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type {
  Tenant,
  TenantWithRole,
  CreateTenantOptions,
  UpdateTenantOptions,
} from "@nimbleflux/fluxbase-sdk";

export interface UseTenantsOptions {
  /**
   * Whether to automatically fetch tenants on mount
   * @default true
   */
  autoFetch?: boolean;
}

export interface UseTenantsReturn {
  /**
   * Array of tenants the user has access to
   */
  tenants: TenantWithRole[];

  /**
   * Whether tenants are being fetched
   */
  isLoading: boolean;

  /**
   * Any error that occurred
   */
  error: Error | null;

  /**
   * Refetch tenants
   */
  refetch: () => Promise<void>;

  /**
   * Create a new tenant (instance admin only)
   */
  createTenant: (options: CreateTenantOptions) => Promise<Tenant>;

  /**
   * Update a tenant (tenant admin only)
   */
  updateTenant: (id: string, options: UpdateTenantOptions) => Promise<Tenant>;

  /**
   * Delete a tenant (instance admin only)
   */
  deleteTenant: (id: string) => Promise<void>;

  /**
   * Set the current tenant context
   */
  setCurrentTenant: (tenantId: string | undefined) => void;

  /**
   * Get the current tenant ID
   */
  currentTenantId: string | undefined;
}

/**
 * Hook for managing tenants
 *
 * Provides tenant list and management functions for multi-tenant applications.
 *
 * @example
 * ```tsx
 * function TenantManager() {
 *   const { tenants, isLoading, setCurrentTenant, currentTenantId } = useTenants()
 *
 *   if (isLoading) return <div>Loading...</div>
 *
 *   return (
 *     <select
 *       value={currentTenantId || ''}
 *       onChange={(e) => setCurrentTenant(e.target.value || undefined)}
 *     >
 *       {tenants.map(t => (
 *         <option key={t.id} value={t.id}>
 *           {t.name} ({t.my_role})
 *         </option>
 *       ))}
 *     </select>
 *   )
 * }
 * ```
 */
export function useTenants(options: UseTenantsOptions = {}): UseTenantsReturn {
  const { autoFetch = true } = options;
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  const [currentTenantId, setCurrentTenantId] = useState<string | undefined>(
    client.getTenantId(),
  );

  const query = useQuery({
    queryKey: ["fluxbase", "tenants", "mine"],
    queryFn: async () => {
      const { data, error } = await client.tenant.listMine();
      if (error) throw error;
      return data || ([] as TenantWithRole[]);
    },
    enabled: autoFetch,
  });

  const createTenant = useCallback(
    async (opts: CreateTenantOptions): Promise<Tenant> => {
      const { data, error: createError } = await client.tenant.create(opts);
      if (createError) throw createError;
      await queryClient.invalidateQueries({
        queryKey: ["fluxbase", "tenants"],
      });
      return data!;
    },
    [client, queryClient],
  );

  const updateTenant = useCallback(
    async (id: string, opts: UpdateTenantOptions): Promise<Tenant> => {
      const { data, error: updateError } = await client.tenant.update(id, opts);
      if (updateError) throw updateError;
      await queryClient.invalidateQueries({
        queryKey: ["fluxbase", "tenants"],
      });
      return data!;
    },
    [client, queryClient],
  );

  const deleteTenant = useCallback(
    async (id: string): Promise<void> => {
      const { error: deleteError } = await client.tenant.delete(id);
      if (deleteError) throw deleteError;
      await queryClient.invalidateQueries({
        queryKey: ["fluxbase", "tenants"],
      });
    },
    [client, queryClient],
  );

  const setCurrentTenant = useCallback(
    (tenantId: string | undefined) => {
      client.setTenant(tenantId);
      setCurrentTenantId(tenantId);
    },
    [client],
  );

  return {
    tenants: query.data ?? [],
    isLoading: autoFetch ? query.isLoading : false,
    error: query.error,
    refetch: async () => {
      await query.refetch();
    },
    createTenant,
    updateTenant,
    deleteTenant,
    setCurrentTenant,
    currentTenantId,
  };
}

export interface UseTenantOptions {
  /**
   * Tenant ID to fetch
   */
  tenantId: string;

  /**
   * Whether to automatically fetch tenant on mount
   * @default true
   */
  autoFetch?: boolean;
}

export interface UseTenantReturn {
  /**
   * Tenant data
   */
  tenant: Tenant | null;

  /**
   * Whether tenant is being fetched
   */
  isLoading: boolean;

  /**
   * Any error that occurred
   */
  error: Error | null;

  /**
   * Refetch tenant
   */
  refetch: () => Promise<void>;

  /**
   * Update the tenant
   */
  update: (options: UpdateTenantOptions) => Promise<Tenant>;

  /**
   * Delete the tenant
   */
  remove: () => Promise<void>;
}

/**
 * Hook for managing a single tenant
 *
 * @example
 * ```tsx
 * function TenantDetails({ tenantId }: { tenantId: string }) {
 *   const { tenant, isLoading, update } = useTenant({ tenantId })
 *
 *   if (isLoading) return <div>Loading...</div>
 *   if (!tenant) return <div>Tenant not found</div>
 *
 *   return (
 *     <div>
 *       <h1>{tenant.name}</h1>
 *       <button onClick={() => update({ name: 'New Name' })}>
 *         Rename
 *       </button>
 *     </div>
 *   )
 * }
 * ```
 */
export function useTenant(options: UseTenantOptions): UseTenantReturn {
  const { tenantId, autoFetch = true } = options;
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ["fluxbase", "tenant", tenantId],
    queryFn: async () => {
      const { data, error } = await client.tenant.get(tenantId);
      if (error) throw error;
      return data;
    },
    enabled: autoFetch && !!tenantId,
  });

  const update = useCallback(
    async (opts: UpdateTenantOptions): Promise<Tenant> => {
      const { data, error: updateError } = await client.tenant.update(
        tenantId,
        opts,
      );
      if (updateError) throw updateError;
      await queryClient.invalidateQueries({
        queryKey: ["fluxbase", "tenant", tenantId],
      });
      return data!;
    },
    [client, tenantId, queryClient],
  );

  const remove = useCallback(async (): Promise<void> => {
    const { error: deleteError } = await client.tenant.delete(tenantId);
    if (deleteError) throw deleteError;
    await queryClient.invalidateQueries({
      queryKey: ["fluxbase", "tenant", tenantId],
    });
  }, [client, tenantId, queryClient]);

  return {
    tenant: query.data ?? null,
    isLoading: autoFetch ? query.isLoading : false,
    error: query.error,
    refetch: async () => {
      await query.refetch();
    },
    update,
    remove,
  };
}
