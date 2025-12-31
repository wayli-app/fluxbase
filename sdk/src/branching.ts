/**
 * Branching module for Fluxbase SDK
 * Provides database branching capabilities for development workflows
 *
 * @example
 * ```typescript
 * // List all branches
 * const { data, error } = await client.branching.list()
 *
 * // Create a new branch for feature development
 * const { data: branch } = await client.branching.create('feature/add-auth', {
 *   dataCloneMode: 'schema_only',
 *   expiresIn: '7d'
 * })
 *
 * // Get branch details
 * const { data: details } = await client.branching.get('feature/add-auth')
 *
 * // Reset branch to parent state
 * await client.branching.reset('feature/add-auth')
 *
 * // Delete branch when done
 * await client.branching.delete('feature/add-auth')
 * ```
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  Branch,
  BranchActivity,
  BranchPoolStats,
  CreateBranchOptions,
  ListBranchesOptions,
  ListBranchesResponse,
} from "./types";

/**
 * Branching client for database branch management
 *
 * Database branches allow you to create isolated copies of your database
 * for development, testing, and preview environments.
 *
 * @category Branching
 */
export class FluxbaseBranching {
  private fetch: FluxbaseFetch;

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
  }

  /**
   * List all database branches
   *
   * @param options - Filter and pagination options
   * @returns Promise resolving to { data, error } tuple with branches list
   *
   * @example
   * ```typescript
   * // List all branches
   * const { data, error } = await client.branching.list()
   *
   * // Filter by status
   * const { data } = await client.branching.list({ status: 'ready' })
   *
   * // Filter by type
   * const { data } = await client.branching.list({ type: 'preview' })
   *
   * // Only show my branches
   * const { data } = await client.branching.list({ mine: true })
   *
   * // Pagination
   * const { data } = await client.branching.list({ limit: 10, offset: 20 })
   * ```
   */
  async list(
    options?: ListBranchesOptions,
  ): Promise<{ data: ListBranchesResponse | null; error: Error | null }> {
    try {
      const params = new URLSearchParams();

      if (options?.status) {
        params.append("status", options.status);
      }
      if (options?.type) {
        params.append("type", options.type);
      }
      if (options?.githubRepo) {
        params.append("github_repo", options.githubRepo);
      }
      if (options?.mine) {
        params.append("mine", "true");
      }
      if (options?.limit !== undefined) {
        params.append("limit", options.limit.toString());
      }
      if (options?.offset !== undefined) {
        params.append("offset", options.offset.toString());
      }

      const queryString = params.toString();
      const url = `/api/v1/admin/branches${queryString ? `?${queryString}` : ""}`;

      const response = await this.fetch.get<ListBranchesResponse>(url);
      return { data: response, error: null };
    } catch (err) {
      return { data: null, error: err as Error };
    }
  }

  /**
   * Get a specific branch by ID or slug
   *
   * @param idOrSlug - Branch ID (UUID) or slug
   * @returns Promise resolving to { data, error } tuple with branch details
   *
   * @example
   * ```typescript
   * // Get by slug
   * const { data, error } = await client.branching.get('feature/add-auth')
   *
   * // Get by ID
   * const { data } = await client.branching.get('123e4567-e89b-12d3-a456-426614174000')
   * ```
   */
  async get(
    idOrSlug: string,
  ): Promise<{ data: Branch | null; error: Error | null }> {
    try {
      const response = await this.fetch.get<Branch>(
        `/api/v1/admin/branches/${encodeURIComponent(idOrSlug)}`,
      );
      return { data: response, error: null };
    } catch (err) {
      return { data: null, error: err as Error };
    }
  }

  /**
   * Create a new database branch
   *
   * @param name - Branch name (will be converted to a slug)
   * @param options - Branch creation options
   * @returns Promise resolving to { data, error } tuple with created branch
   *
   * @example
   * ```typescript
   * // Create a simple branch
   * const { data, error } = await client.branching.create('feature/add-auth')
   *
   * // Create with options
   * const { data } = await client.branching.create('feature/add-auth', {
   *   dataCloneMode: 'schema_only',  // Don't clone data
   *   expiresIn: '7d',               // Auto-delete after 7 days
   *   type: 'persistent'             // Won't auto-delete on PR merge
   * })
   *
   * // Create a PR preview branch
   * const { data } = await client.branching.create('pr-123', {
   *   type: 'preview',
   *   githubPRNumber: 123,
   *   githubRepo: 'owner/repo',
   *   expiresIn: '7d'
   * })
   *
   * // Clone with full data (for debugging)
   * const { data } = await client.branching.create('debug-issue-456', {
   *   dataCloneMode: 'full_clone'
   * })
   * ```
   */
  async create(
    name: string,
    options?: CreateBranchOptions,
  ): Promise<{ data: Branch | null; error: Error | null }> {
    try {
      const body: Record<string, unknown> = { name };

      if (options?.parentBranchId) {
        body.parent_branch_id = options.parentBranchId;
      }
      if (options?.dataCloneMode) {
        body.data_clone_mode = options.dataCloneMode;
      }
      if (options?.type) {
        body.type = options.type;
      }
      if (options?.githubPRNumber !== undefined) {
        body.github_pr_number = options.githubPRNumber;
      }
      if (options?.githubPRUrl) {
        body.github_pr_url = options.githubPRUrl;
      }
      if (options?.githubRepo) {
        body.github_repo = options.githubRepo;
      }
      if (options?.expiresIn) {
        body.expires_in = options.expiresIn;
      }

      const response = await this.fetch.post<Branch>(
        "/api/v1/admin/branches",
        body,
      );
      return { data: response, error: null };
    } catch (err) {
      return { data: null, error: err as Error };
    }
  }

  /**
   * Delete a database branch
   *
   * This permanently deletes the branch database and all its data.
   * Cannot delete the main branch.
   *
   * @param idOrSlug - Branch ID (UUID) or slug
   * @returns Promise resolving to { error } (null on success)
   *
   * @example
   * ```typescript
   * // Delete a branch
   * const { error } = await client.branching.delete('feature/add-auth')
   *
   * if (error) {
   *   console.error('Failed to delete branch:', error.message)
   * }
   * ```
   */
  async delete(idOrSlug: string): Promise<{ error: Error | null }> {
    try {
      await this.fetch.delete(
        `/api/v1/admin/branches/${encodeURIComponent(idOrSlug)}`,
      );
      return { error: null };
    } catch (err) {
      return { error: err as Error };
    }
  }

  /**
   * Reset a branch to its parent state
   *
   * This drops and recreates the branch database, resetting all data
   * to match the parent branch. Cannot reset the main branch.
   *
   * @param idOrSlug - Branch ID (UUID) or slug
   * @returns Promise resolving to { data, error } tuple with reset branch
   *
   * @example
   * ```typescript
   * // Reset a branch
   * const { data, error } = await client.branching.reset('feature/add-auth')
   *
   * if (data) {
   *   console.log('Branch reset, status:', data.status)
   * }
   * ```
   */
  async reset(
    idOrSlug: string,
  ): Promise<{ data: Branch | null; error: Error | null }> {
    try {
      const response = await this.fetch.post<Branch>(
        `/api/v1/admin/branches/${encodeURIComponent(idOrSlug)}/reset`,
        {},
      );
      return { data: response, error: null };
    } catch (err) {
      return { data: null, error: err as Error };
    }
  }

  /**
   * Get activity log for a branch
   *
   * @param idOrSlug - Branch ID (UUID) or slug
   * @param limit - Maximum number of entries to return (default: 50, max: 100)
   * @returns Promise resolving to { data, error } tuple with activity entries
   *
   * @example
   * ```typescript
   * // Get recent activity
   * const { data, error } = await client.branching.getActivity('feature/add-auth')
   *
   * if (data) {
   *   for (const entry of data) {
   *     console.log(`${entry.action}: ${entry.status}`)
   *   }
   * }
   *
   * // Get more entries
   * const { data } = await client.branching.getActivity('feature/add-auth', 100)
   * ```
   */
  async getActivity(
    idOrSlug: string,
    limit: number = 50,
  ): Promise<{ data: BranchActivity[] | null; error: Error | null }> {
    try {
      const response = await this.fetch.get<{ activity: BranchActivity[] }>(
        `/api/v1/admin/branches/${encodeURIComponent(idOrSlug)}/activity?limit=${limit}`,
      );
      return { data: response.activity, error: null };
    } catch (err) {
      return { data: null, error: err as Error };
    }
  }

  /**
   * Get connection pool statistics for all branches
   *
   * This is useful for monitoring and debugging branch connections.
   *
   * @returns Promise resolving to { data, error } tuple with pool stats
   *
   * @example
   * ```typescript
   * const { data, error } = await client.branching.getPoolStats()
   *
   * if (data) {
   *   for (const pool of data) {
   *     console.log(`${pool.slug}: ${pool.active_connections} active`)
   *   }
   * }
   * ```
   */
  async getPoolStats(): Promise<{
    data: BranchPoolStats[] | null;
    error: Error | null;
  }> {
    try {
      const response = await this.fetch.get<{ pools: BranchPoolStats[] }>(
        "/api/v1/admin/branches/stats/pools",
      );
      return { data: response.pools, error: null };
    } catch (err) {
      return { data: null, error: err as Error };
    }
  }

  /**
   * Check if a branch exists
   *
   * @param idOrSlug - Branch ID (UUID) or slug
   * @returns Promise resolving to true if branch exists, false otherwise
   *
   * @example
   * ```typescript
   * const exists = await client.branching.exists('feature/add-auth')
   *
   * if (!exists) {
   *   await client.branching.create('feature/add-auth')
   * }
   * ```
   */
  async exists(idOrSlug: string): Promise<boolean> {
    const { data, error } = await this.get(idOrSlug);
    return !error && data !== null;
  }

  /**
   * Wait for a branch to be ready
   *
   * Polls the branch status until it reaches 'ready' or an error state.
   *
   * @param idOrSlug - Branch ID (UUID) or slug
   * @param options - Polling options
   * @returns Promise resolving to { data, error } tuple with ready branch
   *
   * @example
   * ```typescript
   * // Create branch and wait for it to be ready
   * const { data: branch } = await client.branching.create('feature/add-auth')
   *
   * const { data: ready, error } = await client.branching.waitForReady(branch!.slug, {
   *   timeout: 60000,     // 60 seconds
   *   pollInterval: 1000  // Check every second
   * })
   *
   * if (ready) {
   *   console.log('Branch is ready!')
   * }
   * ```
   */
  async waitForReady(
    idOrSlug: string,
    options?: {
      /** Timeout in milliseconds (default: 30000) */
      timeout?: number;
      /** Poll interval in milliseconds (default: 1000) */
      pollInterval?: number;
    },
  ): Promise<{ data: Branch | null; error: Error | null }> {
    const timeout = options?.timeout ?? 30000;
    const pollInterval = options?.pollInterval ?? 1000;
    const startTime = Date.now();

    while (Date.now() - startTime < timeout) {
      const { data, error } = await this.get(idOrSlug);

      if (error) {
        return { data: null, error };
      }

      if (!data) {
        return { data: null, error: new Error("Branch not found") };
      }

      if (data.status === "ready") {
        return { data, error: null };
      }

      if (data.status === "error") {
        return {
          data: null,
          error: new Error(data.error_message ?? "Branch creation failed"),
        };
      }

      if (data.status === "deleted" || data.status === "deleting") {
        return {
          data: null,
          error: new Error("Branch was deleted"),
        };
      }

      // Wait before next poll
      await new Promise((resolve) => setTimeout(resolve, pollInterval));
    }

    return {
      data: null,
      error: new Error(
        `Timeout waiting for branch to be ready after ${timeout}ms`,
      ),
    };
  }
}
