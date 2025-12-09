/**
 * Admin Jobs module for managing job functions and executions
 * Provides administrative operations for job lifecycle management
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  JobFunction,
  CreateJobFunctionRequest,
  UpdateJobFunctionRequest,
  Job,
  JobStats,
  JobWorker,
  SyncJobsResult,
  JobFunctionSpec,
  SyncJobsOptions,
} from "./types";
import {
  bundleCode,
  loadEsbuild,
  loadImportMap,
  denoExternalPlugin,
  type BundleOptions,
  type BundleResult,
} from "./bundling";

// Re-export bundling utilities for backwards compatibility
export { denoExternalPlugin, loadImportMap };
export type { BundleOptions, BundleResult };

/**
 * Admin Jobs manager for managing background job functions
 * Provides create, update, delete, sync, and monitoring operations
 *
 * @category Admin
 */
export class FluxbaseAdminJobs {
  private fetch: FluxbaseFetch;

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
  }

  /**
   * Create a new job function
   *
   * @param request - Job function configuration and code
   * @returns Promise resolving to { data, error } tuple with created job function metadata
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.create({
   *   name: 'process-data',
   *   code: 'export async function handler(req) { return { success: true } }',
   *   enabled: true,
   *   timeout_seconds: 300
   * })
   * ```
   */
  async create(
    request: CreateJobFunctionRequest,
  ): Promise<{ data: JobFunction | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<JobFunction>(
        "/api/v1/admin/jobs/functions",
        request,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * List all namespaces that have job functions
   *
   * @returns Promise resolving to { data, error } tuple with array of namespace strings
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.listNamespaces()
   * if (data) {
   *   console.log('Available namespaces:', data)
   * }
   * ```
   */
  async listNamespaces(): Promise<{
    data: string[] | null;
    error: Error | null;
  }> {
    try {
      const response = await this.fetch.get<{ namespaces: string[] }>(
        "/api/v1/admin/jobs/namespaces",
      );
      return { data: response.namespaces || ["default"], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * List all job functions (admin view)
   *
   * @param namespace - Optional namespace filter
   * @returns Promise resolving to { data, error } tuple with array of job functions
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.list('default')
   * if (data) {
   *   console.log('Job functions:', data.map(f => f.name))
   * }
   * ```
   */
  async list(
    namespace?: string,
  ): Promise<{ data: JobFunction[] | null; error: Error | null }> {
    try {
      const params = namespace ? `?namespace=${namespace}` : "";
      const data = await this.fetch.get<JobFunction[]>(
        `/api/v1/admin/jobs/functions${params}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get details of a specific job function
   *
   * @param namespace - Namespace
   * @param name - Job function name
   * @returns Promise resolving to { data, error } tuple with job function metadata
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.get('default', 'process-data')
   * if (data) {
   *   console.log('Job function version:', data.version)
   * }
   * ```
   */
  async get(
    namespace: string,
    name: string,
  ): Promise<{ data: JobFunction | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<JobFunction>(
        `/api/v1/admin/jobs/functions/${namespace}/${name}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Update an existing job function
   *
   * @param namespace - Namespace
   * @param name - Job function name
   * @param updates - Fields to update
   * @returns Promise resolving to { data, error } tuple with updated job function metadata
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.update('default', 'process-data', {
   *   enabled: false,
   *   timeout_seconds: 600
   * })
   * ```
   */
  async update(
    namespace: string,
    name: string,
    updates: UpdateJobFunctionRequest,
  ): Promise<{ data: JobFunction | null; error: Error | null }> {
    try {
      const data = await this.fetch.put<JobFunction>(
        `/api/v1/admin/jobs/functions/${namespace}/${name}`,
        updates,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Delete a job function
   *
   * @param namespace - Namespace
   * @param name - Job function name
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.delete('default', 'process-data')
   * ```
   */
  async delete(
    namespace: string,
    name: string,
  ): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.delete(
        `/api/v1/admin/jobs/functions/${namespace}/${name}`,
      );
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * List all jobs (executions) across all namespaces (admin view)
   *
   * @param filters - Optional filters (status, namespace, limit, offset)
   * @returns Promise resolving to { data, error } tuple with array of jobs
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.listJobs({
   *   status: 'running',
   *   namespace: 'default',
   *   limit: 50
   * })
   * if (data) {
   *   data.forEach(job => {
   *     console.log(`${job.job_name}: ${job.status}`)
   *   })
   * }
   * ```
   */
  async listJobs(filters?: {
    status?: string;
    namespace?: string;
    limit?: number;
    offset?: number;
    includeResult?: boolean;
  }): Promise<{ data: Job[] | null; error: Error | null }> {
    try {
      const params = new URLSearchParams();
      if (filters?.status) params.append("status", filters.status);
      if (filters?.namespace) params.append("namespace", filters.namespace);
      if (filters?.limit) params.append("limit", filters.limit.toString());
      if (filters?.offset) params.append("offset", filters.offset.toString());
      if (filters?.includeResult) params.append("include_result", "true");

      const queryString = params.toString();
      const data = await this.fetch.get<Job[]>(
        `/api/v1/admin/jobs/queue${queryString ? `?${queryString}` : ""}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get details of a specific job (execution)
   *
   * @param jobId - Job ID
   * @returns Promise resolving to { data, error } tuple with job details
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.getJob('550e8400-e29b-41d4-a716-446655440000')
   * if (data) {
   *   console.log(`Job ${data.job_name}: ${data.status}`)
   * }
   * ```
   */
  async getJob(
    jobId: string,
  ): Promise<{ data: Job | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<Job>(
        `/api/v1/admin/jobs/queue/${jobId}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Cancel a running or pending job
   *
   * @param jobId - Job ID
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.cancel('550e8400-e29b-41d4-a716-446655440000')
   * ```
   */
  async cancel(jobId: string): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.post(`/api/v1/admin/jobs/queue/${jobId}/cancel`, {});
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Terminate a running job immediately
   *
   * @param jobId - Job ID
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.terminate('550e8400-e29b-41d4-a716-446655440000')
   * ```
   */
  async terminate(jobId: string): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.post(`/api/v1/admin/jobs/queue/${jobId}/terminate`, {});
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Retry a failed job
   *
   * @param jobId - Job ID
   * @returns Promise resolving to { data, error } tuple with new job
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.retry('550e8400-e29b-41d4-a716-446655440000')
   * ```
   */
  async retry(
    jobId: string,
  ): Promise<{ data: Job | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<Job>(
        `/api/v1/admin/jobs/queue/${jobId}/retry`,
        {},
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get job statistics
   *
   * @param namespace - Optional namespace filter
   * @returns Promise resolving to { data, error } tuple with job stats
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.getStats('default')
   * if (data) {
   *   console.log(`Pending: ${data.pending}, Running: ${data.running}`)
   * }
   * ```
   */
  async getStats(
    namespace?: string,
  ): Promise<{ data: JobStats | null; error: Error | null }> {
    try {
      const params = namespace ? `?namespace=${namespace}` : "";
      const data = await this.fetch.get<JobStats>(
        `/api/v1/admin/jobs/stats${params}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * List active workers
   *
   * @returns Promise resolving to { data, error } tuple with array of workers
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.listWorkers()
   * if (data) {
   *   data.forEach(worker => {
   *     console.log(`Worker ${worker.id}: ${worker.current_jobs} jobs`)
   *   })
   * }
   * ```
   */
  async listWorkers(): Promise<{
    data: JobWorker[] | null;
    error: Error | null;
  }> {
    try {
      const data = await this.fetch.get<JobWorker[]>(
        "/api/v1/admin/jobs/workers",
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Sync multiple job functions to a namespace
   *
   * Can sync from:
   * 1. Filesystem (if no jobs provided) - loads from configured jobs directory
   * 2. API payload (if jobs array provided) - syncs provided job specifications
   *
   * Requires service_role or admin authentication.
   *
   * @param options - Sync options including namespace and optional jobs array
   * @returns Promise resolving to { data, error } tuple with sync results
   *
   * @example
   * ```typescript
   * // Sync from filesystem
   * const { data, error } = await client.admin.jobs.sync({ namespace: 'default' })
   *
   * // Sync with pre-bundled code (client-side bundling)
   * const bundled = await FluxbaseAdminJobs.bundleCode({ code: myJobCode })
   * const { data, error } = await client.admin.jobs.sync({
   *   namespace: 'default',
   *   functions: [{
   *     name: 'my-job',
   *     code: bundled.code,
   *     is_pre_bundled: true,
   *     original_code: myJobCode,
   *   }],
   *   options: {
   *     delete_missing: true, // Remove jobs not in this sync
   *     dry_run: false,       // Preview changes without applying
   *   }
   * })
   *
   * if (data) {
   *   console.log(`Synced: ${data.summary.created} created, ${data.summary.updated} updated`)
   * }
   * ```
   */
  async sync(
    options: SyncJobsOptions | string,
  ): Promise<{ data: SyncJobsResult | null; error: Error | null }> {
    try {
      // Support legacy string-only namespace argument
      const syncOptions: SyncJobsOptions =
        typeof options === "string" ? { namespace: options } : options;

      const data = await this.fetch.post<SyncJobsResult>(
        "/api/v1/admin/jobs/sync",
        {
          namespace: syncOptions.namespace,
          jobs: syncOptions.functions,
          options: {
            delete_missing: syncOptions.options?.delete_missing ?? false,
            dry_run: syncOptions.options?.dry_run ?? false,
          },
        },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Sync job functions with automatic client-side bundling
   *
   * This is a convenience method that bundles all job code using esbuild
   * before sending to the server. Requires esbuild as a peer dependency.
   *
   * @param options - Sync options including namespace and jobs array
   * @param bundleOptions - Optional bundling configuration
   * @returns Promise resolving to { data, error } tuple with sync results
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.jobs.syncWithBundling({
   *   namespace: 'default',
   *   functions: [
   *     { name: 'process-data', code: processDataCode },
   *     { name: 'send-email', code: sendEmailCode },
   *   ],
   *   options: { delete_missing: true }
   * })
   * ```
   */
  async syncWithBundling(
    options: SyncJobsOptions,
    bundleOptions?: Partial<BundleOptions>,
  ): Promise<{ data: SyncJobsResult | null; error: Error | null }> {
    if (!options.functions || options.functions.length === 0) {
      return this.sync(options);
    }

    // Check if esbuild is available
    const hasEsbuild = await loadEsbuild();
    if (!hasEsbuild) {
      return {
        data: null,
        error: new Error(
          "esbuild is required for client-side bundling. Install it with: npm install esbuild",
        ),
      };
    }

    try {
      // Bundle each function
      const bundledFunctions: JobFunctionSpec[] = await Promise.all(
        options.functions.map(async (fn) => {
          // Skip if already pre-bundled
          if (fn.is_pre_bundled) {
            return fn;
          }

          const bundled = await FluxbaseAdminJobs.bundleCode({
            // Apply global bundle options first
            ...bundleOptions,
            // Then override with per-function values (these take priority)
            code: fn.code,
            // Use function's sourceDir for resolving relative imports
            baseDir: fn.sourceDir || bundleOptions?.baseDir,
            // Use function's nodePaths for additional module resolution
            nodePaths: fn.nodePaths || bundleOptions?.nodePaths,
          });

          return {
            ...fn,
            code: bundled.code,
            original_code: fn.code,
            is_pre_bundled: true,
          };
        }),
      );

      // Sync with pre-bundled code
      return this.sync({
        ...options,
        functions: bundledFunctions,
      });
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Bundle job code using esbuild (client-side)
   *
   * Transforms and bundles TypeScript/JavaScript code into a single file
   * that can be executed by the Fluxbase jobs runtime.
   *
   * Requires esbuild as a peer dependency.
   *
   * @param options - Bundle options including source code
   * @returns Promise resolving to bundled code
   * @throws Error if esbuild is not available
   *
   * @example
   * ```typescript
   * const bundled = await FluxbaseAdminJobs.bundleCode({
   *   code: `
   *     import { helper } from './utils'
   *     export async function handler(req) {
   *       return helper(req.payload)
   *     }
   *   `,
   *   minify: true,
   * })
   *
   * // Use bundled code in sync
   * await client.admin.jobs.sync({
   *   namespace: 'default',
   *   functions: [{
   *     name: 'my-job',
   *     code: bundled.code,
   *     is_pre_bundled: true,
   *   }]
   * })
   * ```
   */
  static async bundleCode(options: BundleOptions): Promise<BundleResult> {
    return bundleCode(options);
  }
}
