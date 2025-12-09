/**
 * Admin RPC module for managing RPC procedures
 * Provides administrative operations for RPC procedure lifecycle management
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  RPCProcedure,
  RPCProcedureSummary,
  RPCExecution,
  RPCExecutionLog,
  SyncRPCOptions,
  SyncRPCResult,
  UpdateRPCProcedureRequest,
  RPCExecutionFilters,
} from "./types";

/**
 * Admin RPC manager for managing RPC procedures
 * Provides sync, CRUD, and execution monitoring operations
 *
 * @category Admin
 */
export class FluxbaseAdminRPC {
  private fetch: FluxbaseFetch;

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
  }

  // ============================================================================
  // PROCEDURE MANAGEMENT
  // ============================================================================

  /**
   * Sync RPC procedures from filesystem or API payload
   *
   * Can sync from:
   * 1. Filesystem (if no procedures provided) - loads from configured procedures directory
   * 2. API payload (if procedures array provided) - syncs provided procedure specifications
   *
   * Requires service_role or admin authentication.
   *
   * @param options - Sync options including namespace and optional procedures array
   * @returns Promise resolving to { data, error } tuple with sync results
   *
   * @example
   * ```typescript
   * // Sync from filesystem
   * const { data, error } = await client.admin.rpc.sync()
   *
   * // Sync with provided procedure code
   * const { data, error } = await client.admin.rpc.sync({
   *   namespace: 'default',
   *   procedures: [{
   *     name: 'get-user-orders',
   *     code: myProcedureSQL,
   *   }],
   *   options: {
   *     delete_missing: false, // Don't remove procedures not in this sync
   *     dry_run: false,        // Preview changes without applying
   *   }
   * })
   *
   * if (data) {
   *   console.log(`Synced: ${data.summary.created} created, ${data.summary.updated} updated`)
   * }
   * ```
   */
  async sync(
    options?: SyncRPCOptions,
  ): Promise<{ data: SyncRPCResult | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<SyncRPCResult>(
        "/api/v1/admin/rpc/sync",
        {
          namespace: options?.namespace || "default",
          procedures: options?.procedures,
          options: {
            delete_missing: options?.options?.delete_missing ?? false,
            dry_run: options?.options?.dry_run ?? false,
          },
        },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * List all RPC procedures (admin view)
   *
   * @param namespace - Optional namespace filter
   * @returns Promise resolving to { data, error } tuple with array of procedure summaries
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.rpc.list()
   * if (data) {
   *   console.log('Procedures:', data.map(p => p.name))
   * }
   * ```
   */
  async list(
    namespace?: string,
  ): Promise<{ data: RPCProcedureSummary[] | null; error: Error | null }> {
    try {
      const params = namespace ? `?namespace=${encodeURIComponent(namespace)}` : "";
      const response = await this.fetch.get<{ procedures: RPCProcedureSummary[]; count: number }>(
        `/api/v1/admin/rpc/procedures${params}`,
      );
      return { data: response.procedures || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * List all namespaces
   *
   * @returns Promise resolving to { data, error } tuple with array of namespace names
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.rpc.listNamespaces()
   * if (data) {
   *   console.log('Namespaces:', data)
   * }
   * ```
   */
  async listNamespaces(): Promise<{ data: string[] | null; error: Error | null }> {
    try {
      const response = await this.fetch.get<{ namespaces: string[] }>(
        "/api/v1/admin/rpc/namespaces",
      );
      return { data: response.namespaces || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get details of a specific RPC procedure
   *
   * @param namespace - Procedure namespace
   * @param name - Procedure name
   * @returns Promise resolving to { data, error } tuple with procedure details
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.rpc.get('default', 'get-user-orders')
   * if (data) {
   *   console.log('Procedure:', data.name)
   *   console.log('SQL:', data.sql_query)
   * }
   * ```
   */
  async get(
    namespace: string,
    name: string,
  ): Promise<{ data: RPCProcedure | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<RPCProcedure>(
        `/api/v1/admin/rpc/procedures/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Update an RPC procedure
   *
   * @param namespace - Procedure namespace
   * @param name - Procedure name
   * @param updates - Fields to update
   * @returns Promise resolving to { data, error } tuple with updated procedure
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.rpc.update('default', 'get-user-orders', {
   *   enabled: false,
   *   max_execution_time_seconds: 60,
   * })
   * ```
   */
  async update(
    namespace: string,
    name: string,
    updates: UpdateRPCProcedureRequest,
  ): Promise<{ data: RPCProcedure | null; error: Error | null }> {
    try {
      const data = await this.fetch.put<RPCProcedure>(
        `/api/v1/admin/rpc/procedures/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}`,
        updates,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Enable or disable an RPC procedure
   *
   * @param namespace - Procedure namespace
   * @param name - Procedure name
   * @param enabled - Whether to enable or disable
   * @returns Promise resolving to { data, error } tuple with updated procedure
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.rpc.toggle('default', 'get-user-orders', true)
   * ```
   */
  async toggle(
    namespace: string,
    name: string,
    enabled: boolean,
  ): Promise<{ data: RPCProcedure | null; error: Error | null }> {
    return this.update(namespace, name, { enabled });
  }

  /**
   * Delete an RPC procedure
   *
   * @param namespace - Procedure namespace
   * @param name - Procedure name
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.rpc.delete('default', 'get-user-orders')
   * ```
   */
  async delete(
    namespace: string,
    name: string,
  ): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.delete(
        `/api/v1/admin/rpc/procedures/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}`,
      );
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  // ============================================================================
  // EXECUTION MONITORING
  // ============================================================================

  /**
   * List RPC executions with optional filters
   *
   * @param filters - Optional filters for namespace, procedure, status, user
   * @returns Promise resolving to { data, error } tuple with array of executions
   *
   * @example
   * ```typescript
   * // List all executions
   * const { data, error } = await client.admin.rpc.listExecutions()
   *
   * // List failed executions for a specific procedure
   * const { data, error } = await client.admin.rpc.listExecutions({
   *   namespace: 'default',
   *   procedure: 'get-user-orders',
   *   status: 'failed',
   * })
   * ```
   */
  async listExecutions(
    filters?: RPCExecutionFilters,
  ): Promise<{ data: RPCExecution[] | null; error: Error | null }> {
    try {
      const params = new URLSearchParams();
      if (filters?.namespace) params.set("namespace", filters.namespace);
      if (filters?.procedure) params.set("procedure_name", filters.procedure);
      if (filters?.status) params.set("status", filters.status);
      if (filters?.user_id) params.set("user_id", filters.user_id);
      if (filters?.limit) params.set("limit", filters.limit.toString());
      if (filters?.offset) params.set("offset", filters.offset.toString());

      const queryString = params.toString();
      const path = queryString
        ? `/api/v1/admin/rpc/executions?${queryString}`
        : "/api/v1/admin/rpc/executions";

      const response = await this.fetch.get<{ executions: RPCExecution[]; count: number }>(path);
      return { data: response.executions || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get details of a specific execution
   *
   * @param executionId - Execution ID
   * @returns Promise resolving to { data, error } tuple with execution details
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.rpc.getExecution('execution-uuid')
   * if (data) {
   *   console.log('Status:', data.status)
   *   console.log('Duration:', data.duration_ms, 'ms')
   * }
   * ```
   */
  async getExecution(
    executionId: string,
  ): Promise<{ data: RPCExecution | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<RPCExecution>(
        `/api/v1/admin/rpc/executions/${encodeURIComponent(executionId)}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get execution logs for a specific execution
   *
   * @param executionId - Execution ID
   * @param afterLine - Optional line number to get logs after (for polling)
   * @returns Promise resolving to { data, error } tuple with execution logs
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.rpc.getExecutionLogs('execution-uuid')
   * if (data) {
   *   for (const log of data) {
   *     console.log(`[${log.level}] ${log.message}`)
   *   }
   * }
   * ```
   */
  async getExecutionLogs(
    executionId: string,
    afterLine?: number,
  ): Promise<{ data: RPCExecutionLog[] | null; error: Error | null }> {
    try {
      const params = afterLine !== undefined ? `?after=${afterLine}` : "";
      const response = await this.fetch.get<{ logs: RPCExecutionLog[]; count: number }>(
        `/api/v1/admin/rpc/executions/${encodeURIComponent(executionId)}/logs${params}`,
      );
      return { data: response.logs || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Cancel a running execution
   *
   * @param executionId - Execution ID
   * @returns Promise resolving to { data, error } tuple with updated execution
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.rpc.cancelExecution('execution-uuid')
   * ```
   */
  async cancelExecution(
    executionId: string,
  ): Promise<{ data: RPCExecution | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<RPCExecution>(
        `/api/v1/admin/rpc/executions/${encodeURIComponent(executionId)}/cancel`,
        {},
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }
}
