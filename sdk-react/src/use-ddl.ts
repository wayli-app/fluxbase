/**
 * DDL hooks for Fluxbase React SDK
 * Provides hooks for managing database schemas and tables
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";

/**
 * Hook to list all database schemas
 */
export function useSchemas() {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "schemas"],
    queryFn: async () => {
      return await client.admin.ddl.listSchemas();
    },
  });
}

/**
 * Hook to list tables, optionally filtered by schema
 */
export function useTables(schema?: string) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "tables", schema],
    queryFn: async () => {
      return await client.admin.ddl.listTables(schema);
    },
  });
}

/**
 * Hook to create a new database schema
 */
export function useCreateSchema() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (name: string) => {
      return await client.admin.ddl.createSchema(name);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "schemas"],
      });
    },
  });
}

/**
 * Hook to delete a table from a schema
 */
export function useDeleteTable() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ schema, name }: { schema: string; name: string }) => {
      return await client.admin.ddl.deleteTable(schema, name);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "tables"],
      });
    },
  });
}
