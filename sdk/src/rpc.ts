/**
 * RPC (Remote Procedure Call) module for invoking SQL-based procedures
 */

import type {
  RPCProcedureSummary,
  RPCExecution,
  RPCInvokeResponse,
  RPCExecutionLog,
} from "./types";

/**
 * Options for invoking an RPC procedure
 */
export interface RPCInvokeOptions {
  /** Namespace of the procedure (defaults to 'default') */
  namespace?: string;
  /** Execute asynchronously (returns execution ID immediately) */
  async?: boolean;
  /** Request timeout in milliseconds (default: 30000) */
  timeout?: number;
}

/**
 * Fetch interface for RPC operations
 */
interface RPCFetch {
  get: <T>(path: string) => Promise<T>;
  post: <T>(path: string, body?: unknown) => Promise<T>;
}

/**
 * FluxbaseRPC provides methods for invoking RPC procedures
 *
 * @example
 * ```typescript
 * // Invoke a procedure synchronously
 * const { data, error } = await fluxbase.rpc.invoke('get-user-orders', {
 *   user_id: '123',
 *   limit: 10
 * });
 *
 * // Invoke asynchronously
 * const { data: asyncResult } = await fluxbase.rpc.invoke('long-running-report', {
 *   start_date: '2024-01-01'
 * }, { async: true });
 *
 * // Poll for status
 * const { data: status } = await fluxbase.rpc.getStatus(asyncResult.execution_id);
 * ```
 */
export class FluxbaseRPC {
  private fetch: RPCFetch;

  constructor(fetch: RPCFetch) {
    this.fetch = fetch;
  }

  /**
   * List available RPC procedures (public, enabled)
   *
   * @param namespace - Optional namespace filter
   * @returns Promise resolving to { data, error } tuple with array of procedure summaries
   */
  async list(namespace?: string): Promise<{
    data: RPCProcedureSummary[] | null;
    error: Error | null;
  }> {
    try {
      const params = namespace
        ? `?namespace=${encodeURIComponent(namespace)}`
        : "";
      const response = await this.fetch.get<{
        procedures: RPCProcedureSummary[];
        count: number;
      }>(`/api/v1/rpc/procedures${params}`);
      return { data: response.procedures || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Invoke an RPC procedure
   *
   * @param name - Procedure name
   * @param params - Optional parameters to pass to the procedure
   * @param options - Optional invocation options
   * @returns Promise resolving to { data, error } tuple with invocation response
   *
   * @example
   * ```typescript
   * // Synchronous invocation
   * const { data, error } = await fluxbase.rpc.invoke('get-user-orders', {
   *   user_id: '123',
   *   limit: 10
   * });
   * console.log(data.result); // Query results
   *
   * // Asynchronous invocation
   * const { data: asyncData } = await fluxbase.rpc.invoke('generate-report', {
   *   year: 2024
   * }, { async: true });
   * console.log(asyncData.execution_id); // Use to poll status
   * ```
   */
  async invoke<T = unknown>(
    name: string,
    params?: Record<string, unknown>,
    options?: RPCInvokeOptions,
  ): Promise<{ data: RPCInvokeResponse<T> | null; error: Error | null }> {
    try {
      const namespace = options?.namespace || "default";
      const response = await this.fetch.post<RPCInvokeResponse<T>>(
        `/api/v1/rpc/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}`,
        {
          params,
          async: options?.async,
        },
        { timeout: options?.timeout },
      );
      return { data: response, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get execution status (for async invocations or checking history)
   *
   * @param executionId - The execution ID returned from async invoke
   * @returns Promise resolving to { data, error } tuple with execution details
   *
   * @example
   * ```typescript
   * const { data, error } = await fluxbase.rpc.getStatus('execution-uuid');
   * if (data.status === 'completed') {
   *   console.log('Result:', data.result);
   * } else if (data.status === 'running') {
   *   console.log('Still running...');
   * }
   * ```
   */
  async getStatus(executionId: string): Promise<{
    data: RPCExecution | null;
    error: Error | null;
  }> {
    try {
      const data = await this.fetch.get<RPCExecution>(
        `/api/v1/rpc/executions/${encodeURIComponent(executionId)}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get execution logs (for debugging and monitoring)
   *
   * @param executionId - The execution ID
   * @param afterLine - Optional line number to get logs after (for polling)
   * @returns Promise resolving to { data, error } tuple with execution logs
   *
   * @example
   * ```typescript
   * const { data: logs } = await fluxbase.rpc.getLogs('execution-uuid');
   * for (const log of logs) {
   *   console.log(`[${log.level}] ${log.message}`);
   * }
   * ```
   */
  async getLogs(
    executionId: string,
    afterLine?: number,
  ): Promise<{ data: RPCExecutionLog[] | null; error: Error | null }> {
    try {
      const params = afterLine !== undefined ? `?after=${afterLine}` : "";
      const response = await this.fetch.get<{
        logs: RPCExecutionLog[];
        count: number;
      }>(
        `/api/v1/rpc/executions/${encodeURIComponent(executionId)}/logs${params}`,
      );
      return { data: response.logs || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Poll for execution completion with exponential backoff
   *
   * @param executionId - The execution ID to poll
   * @param options - Polling options
   * @returns Promise resolving to final execution state
   *
   * @example
   * ```typescript
   * const { data: result } = await fluxbase.rpc.invoke('long-task', {}, { async: true });
   * const { data: final } = await fluxbase.rpc.waitForCompletion(result.execution_id, {
   *   maxWaitMs: 60000, // Wait up to 1 minute
   *   onProgress: (exec) => console.log(`Status: ${exec.status}`)
   * });
   * console.log('Final result:', final.result);
   * ```
   */
  async waitForCompletion(
    executionId: string,
    options?: {
      /** Maximum time to wait in milliseconds (default: 30000) */
      maxWaitMs?: number;
      /** Initial polling interval in milliseconds (default: 500) */
      initialIntervalMs?: number;
      /** Maximum polling interval in milliseconds (default: 5000) */
      maxIntervalMs?: number;
      /** Callback for progress updates */
      onProgress?: (execution: RPCExecution) => void;
    },
  ): Promise<{ data: RPCExecution | null; error: Error | null }> {
    const maxWait = options?.maxWaitMs || 30000;
    const initialInterval = options?.initialIntervalMs || 500;
    const maxInterval = options?.maxIntervalMs || 5000;

    const startTime = Date.now();
    let interval = initialInterval;

    while (Date.now() - startTime < maxWait) {
      const { data: execution, error } = await this.getStatus(executionId);

      if (error) {
        return { data: null, error };
      }

      if (!execution) {
        return { data: null, error: new Error("Execution not found") };
      }

      if (options?.onProgress) {
        options.onProgress(execution);
      }

      // Check for terminal states
      if (
        execution.status === "completed" ||
        execution.status === "failed" ||
        execution.status === "cancelled" ||
        execution.status === "timeout"
      ) {
        return { data: execution, error: null };
      }

      // Wait before next poll with exponential backoff
      await new Promise((resolve) => setTimeout(resolve, interval));
      interval = Math.min(interval * 1.5, maxInterval);
    }

    return {
      data: null,
      error: new Error("Timeout waiting for execution to complete"),
    };
  }
}
