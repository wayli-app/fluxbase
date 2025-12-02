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

// Optional esbuild import - will be dynamically loaded if available
let esbuild: typeof import("esbuild") | null = null;

// Optional fs import for reading deno.json
let fs: typeof import("fs") | null = null;

/**
 * Try to load esbuild for client-side bundling
 * Returns true if esbuild is available, false otherwise
 */
async function loadEsbuild(): Promise<boolean> {
  if (esbuild) return true;
  try {
    esbuild = await import("esbuild");
    return true;
  } catch {
    return false;
  }
}

/**
 * Try to load fs module
 */
async function loadFs(): Promise<boolean> {
  if (fs) return true;
  try {
    fs = await import("fs");
    return true;
  } catch {
    return false;
  }
}

/**
 * Load import map from a deno.json file
 *
 * @param denoJsonPath - Path to deno.json file
 * @returns Import map object or null if not found
 *
 * @example
 * ```typescript
 * const importMap = await loadImportMap('./deno.json')
 * const bundled = await FluxbaseAdminJobs.bundleCode({
 *   code: myCode,
 *   importMap,
 * })
 * ```
 */
/**
 * esbuild plugin that marks Deno-specific imports as external
 * Use this when bundling jobs with esbuild to handle npm:, https://, and jsr: imports
 *
 * @example
 * ```typescript
 * import { denoExternalPlugin } from '@fluxbase/sdk'
 * import * as esbuild from 'esbuild'
 *
 * const result = await esbuild.build({
 *   entryPoints: ['./my-job.ts'],
 *   bundle: true,
 *   plugins: [denoExternalPlugin],
 *   // ... other options
 * })
 * ```
 */
export const denoExternalPlugin = {
  name: "deno-external",
  setup(build: {
    onResolve: (
      opts: { filter: RegExp },
      cb: (args: { path: string }) => { path: string; external: boolean },
    ) => void;
  }) {
    // Mark npm: imports as external - Deno will resolve them at runtime
    build.onResolve({ filter: /^npm:/ }, (args) => ({
      path: args.path,
      external: true,
    }));

    // Mark https:// and http:// imports as external
    build.onResolve({ filter: /^https?:\/\// }, (args) => ({
      path: args.path,
      external: true,
    }));

    // Mark jsr: imports as external (Deno's JSR registry)
    build.onResolve({ filter: /^jsr:/ }, (args) => ({
      path: args.path,
      external: true,
    }));
  },
};

export async function loadImportMap(
  denoJsonPath: string,
): Promise<Record<string, string> | null> {
  const hasFs = await loadFs();
  if (!hasFs || !fs) {
    console.warn("fs module not available, cannot load import map");
    return null;
  }

  try {
    const content = fs.readFileSync(denoJsonPath, "utf-8");
    const config = JSON.parse(content);
    return config.imports || null;
  } catch (error) {
    console.warn(`Failed to load import map from ${denoJsonPath}:`, error);
    return null;
  }
}

/**
 * Options for bundling job code
 */
export interface BundleOptions {
  /** Entry point code */
  code: string;
  /** External modules to exclude from bundle */
  external?: string[];
  /** Source map generation */
  sourcemap?: boolean;
  /** Minify output */
  minify?: boolean;
  /** Import map from deno.json (maps aliases to npm: or file paths) */
  importMap?: Record<string, string>;
  /** Base directory for resolving relative imports (resolveDir in esbuild) */
  baseDir?: string;
  /** Additional paths to search for node_modules (useful when importing from parent directories) */
  nodePaths?: string[];
  /** Custom define values for esbuild (e.g., { 'process.env.NODE_ENV': '"production"' }) */
  define?: Record<string, string>;
}

/**
 * Result of bundling job code
 */
export interface BundleResult {
  /** Bundled code */
  code: string;
  /** Source map (if enabled) */
  sourceMap?: string;
}

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
    const hasEsbuild = await loadEsbuild();
    if (!hasEsbuild || !esbuild) {
      throw new Error(
        "esbuild is required for bundling. Install it with: npm install esbuild",
      );
    }

    // Process import map to extract externals and aliases
    const externals = [...(options.external ?? [])];
    const alias: Record<string, string> = {};

    if (options.importMap) {
      for (const [key, value] of Object.entries(options.importMap)) {
        // npm: imports should be marked as external - Deno will resolve them at runtime
        if (value.startsWith("npm:")) {
          // Add the import key as external (e.g., "@streamparser/json")
          externals.push(key);
        } else if (
          value.startsWith("https://") ||
          value.startsWith("http://")
        ) {
          // URL imports should also be external - Deno will fetch them at runtime
          externals.push(key);
        } else if (
          value.startsWith("/") ||
          value.startsWith("./") ||
          value.startsWith("../")
        ) {
          // Local file paths - create alias for esbuild
          alias[key] = value;
        } else {
          // Other imports (bare specifiers) - mark as external
          externals.push(key);
        }
      }
    }

    // Create a plugin to handle Deno-specific imports (npm:, https://, http://)
    const denoExternalPlugin: import("esbuild").Plugin = {
      name: "deno-external",
      setup(build) {
        // Mark npm: imports as external
        build.onResolve({ filter: /^npm:/ }, (args) => ({
          path: args.path,
          external: true,
        }));

        // Mark https:// and http:// imports as external
        build.onResolve({ filter: /^https?:\/\// }, (args) => ({
          path: args.path,
          external: true,
        }));

        // Mark jsr: imports as external (Deno's JSR registry)
        build.onResolve({ filter: /^jsr:/ }, (args) => ({
          path: args.path,
          external: true,
        }));
      },
    };

    const resolveDir = options.baseDir || process.cwd?.() || "/";

    const buildOptions: import("esbuild").BuildOptions = {
      stdin: {
        contents: options.code,
        loader: "ts",
        resolveDir,
      },
      // Set absWorkingDir for consistent path resolution
      absWorkingDir: resolveDir,
      bundle: true,
      write: false,
      format: "esm",
      // Use 'node' platform for better node_modules resolution (Deno supports Node APIs)
      platform: "node",
      target: "esnext",
      minify: options.minify ?? false,
      sourcemap: options.sourcemap ? "inline" : false,
      external: externals,
      plugins: [denoExternalPlugin],
      // Preserve handler export
      treeShaking: true,
      // Resolve .ts, .js, .mjs extensions
      resolveExtensions: [".ts", ".tsx", ".js", ".mjs", ".json"],
      // ESM conditions for better module resolution
      conditions: ["import", "module"],
    };

    // Add alias if we have any
    if (Object.keys(alias).length > 0) {
      buildOptions.alias = alias;
    }

    // Add nodePaths for resolving modules from additional directories
    if (options.nodePaths && options.nodePaths.length > 0) {
      buildOptions.nodePaths = options.nodePaths;
    }

    // Add custom define values
    if (options.define) {
      buildOptions.define = options.define;
    }

    const result = await esbuild.build(buildOptions);

    const output = result.outputFiles?.[0];
    if (!output) {
      throw new Error("Bundling failed: no output generated");
    }

    return {
      code: output.text,
      sourceMap: options.sourcemap ? output.text : undefined,
    };
  }
}
