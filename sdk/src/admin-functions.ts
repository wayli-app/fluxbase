/**
 * Admin Functions module for managing edge functions
 * Provides administrative operations for function lifecycle management
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  EdgeFunction,
  CreateFunctionRequest,
  UpdateFunctionRequest,
  EdgeFunctionExecution,
  SyncFunctionsOptions,
  SyncFunctionsResult,
} from "./types";

/**
 * Admin Functions manager for managing edge functions
 * Provides create, update, delete, and bulk sync operations
 *
 * @category Admin
 */
export class FluxbaseAdminFunctions {
  private fetch: FluxbaseFetch;

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
  }

  /**
   * Create a new edge function
   *
   * @param request - Function configuration and code
   * @returns Promise resolving to { data, error } tuple with created function metadata
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.functions.create({
   *   name: 'my-function',
   *   code: 'export default async function handler(req) { return { hello: "world" } }',
   *   enabled: true
   * })
   * ```
   */
  async create(
    request: CreateFunctionRequest,
  ): Promise<{ data: EdgeFunction | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<EdgeFunction>(
        "/api/v1/functions",
        request,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * List all edge functions (admin view)
   *
   * @returns Promise resolving to { data, error } tuple with array of functions
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.functions.list()
   * if (data) {
   *   console.log('Functions:', data.map(f => f.name))
   * }
   * ```
   */
  async list(): Promise<{ data: EdgeFunction[] | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<EdgeFunction[]>("/api/v1/functions");
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get details of a specific edge function
   *
   * @param name - Function name
   * @returns Promise resolving to { data, error } tuple with function metadata
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.functions.get('my-function')
   * if (data) {
   *   console.log('Function version:', data.version)
   * }
   * ```
   */
  async get(
    name: string,
  ): Promise<{ data: EdgeFunction | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<EdgeFunction>(
        `/api/v1/functions/${name}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Update an existing edge function
   *
   * @param name - Function name
   * @param updates - Fields to update
   * @returns Promise resolving to { data, error } tuple with updated function metadata
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.functions.update('my-function', {
   *   enabled: false,
   *   description: 'Updated description'
   * })
   * ```
   */
  async update(
    name: string,
    updates: UpdateFunctionRequest,
  ): Promise<{ data: EdgeFunction | null; error: Error | null }> {
    try {
      const data = await this.fetch.put<EdgeFunction>(
        `/api/v1/functions/${name}`,
        updates,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Delete an edge function
   *
   * @param name - Function name
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.functions.delete('my-function')
   * ```
   */
  async delete(name: string): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.delete(`/api/v1/functions/${name}`);
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get execution history for an edge function
   *
   * @param name - Function name
   * @param limit - Maximum number of executions to return (optional)
   * @returns Promise resolving to { data, error } tuple with execution records
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.functions.getExecutions('my-function', 10)
   * if (data) {
   *   data.forEach(exec => {
   *     console.log(`${exec.executed_at}: ${exec.status} (${exec.duration_ms}ms)`)
   *   })
   * }
   * ```
   */
  async getExecutions(
    name: string,
    limit?: number,
  ): Promise<{ data: EdgeFunctionExecution[] | null; error: Error | null }> {
    try {
      const params = limit ? `?limit=${limit}` : "";
      const data = await this.fetch.get<EdgeFunctionExecution[]>(
        `/api/v1/functions/${name}/executions${params}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Sync multiple functions to a namespace
   *
   * Bulk create/update/delete functions in a specific namespace. This is useful for
   * deploying functions from your application to Fluxbase in Kubernetes or other
   * container environments.
   *
   * Requires service_role or admin authentication.
   *
   * @param options - Sync configuration including namespace, functions, and options
   * @returns Promise resolving to { data, error } tuple with sync results
   *
   * @example
   * ```typescript
   * // Sync functions to "payment-service" namespace
   * const { data, error } = await client.admin.functions.sync({
   *   namespace: 'payment-service',
   *   functions: [
   *     {
   *       name: 'process-payment',
   *       code: 'export default async function handler(req) { ... }',
   *       enabled: true,
   *       allow_net: true
   *     },
   *     {
   *       name: 'refund-payment',
   *       code: 'export default async function handler(req) { ... }',
   *       enabled: true
   *     }
   *   ],
   *   options: {
   *     delete_missing: true  // Remove functions not in this list
   *   }
   * })
   *
   * if (data) {
   *   console.log(`Synced: ${data.summary.created} created, ${data.summary.updated} updated`)
   * }
   *
   * // Dry run to preview changes
   * const { data, error } = await client.admin.functions.sync({
   *   namespace: 'myapp',
   *   functions: [...],
   *   options: { dry_run: true }
   * })
   * ```
   */
  async sync(
    options: SyncFunctionsOptions,
  ): Promise<{ data: SyncFunctionsResult | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<SyncFunctionsResult>(
        "/api/v1/admin/functions/sync",
        options,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }
}
