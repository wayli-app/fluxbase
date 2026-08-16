/**
 * Jobs module for Fluxbase SDK
 * Client-facing API for submitting and monitoring background jobs
 *
 * @example
 * ```typescript
 * // Submit a job
 * const { data, error } = await client.jobs.submit('process-data', {
 *   items: [1, 2, 3]
 * })
 *
 * // Get job status
 * const { data: job, error } = await client.jobs.get(data.id)
 * console.log('Job status:', job.status)
 *
 * // Cancel a running job
 * await client.jobs.cancel(data.id)
 * ```
 */

import type { FluxbaseFetch } from "./fetch";
import type { ExecutionLog, Job, OnBehalfOf, SubmitJobRequest, User } from "./types";

/**
 * Jobs client for submitting and monitoring background jobs
 *
 * For admin operations (create job functions, manage workers, view all jobs),
 * use client.admin.jobs
 *
 * @category Jobs
 */
export class FluxbaseJobs {
  private fetch: FluxbaseFetch;
  private cachedRole: { userId: string; role: string; expiresAt: number } | null = null;
  private readonly ROLE_CACHE_TTL = 5 * 60 * 1000; // 5 minutes
  private isServiceRole: boolean;

  constructor(fetch: FluxbaseFetch, isServiceRole = false) {
    this.fetch = fetch;
    this.isServiceRole = isServiceRole;
  }

  /**
   * Get the current user's role from user_profiles table
   * This is used to auto-populate the onBehalfOf parameter when submitting jobs
   *
   * @returns Promise resolving to the user's role or null if not found
   */
  private async getCurrentUserRole(userId: string): Promise<string | null> {
    try {
      // Check cache
      const now = Date.now();
      if (
        this.cachedRole &&
        this.cachedRole.userId === userId &&
        this.cachedRole.expiresAt > now
      ) {
        return this.cachedRole.role;
      }

      // Fetch user role from user_profiles table
      // Note: This assumes a standard user_profiles table with role column
      const profileResult = await this.fetch.get<Array<{ role: string }>>(
        `/api/v1/tables/user_profiles?id=eq.${userId}&select=role`
      );

      if (!profileResult || !profileResult[0]) {
        return null;
      }

      const role = profileResult[0].role;

      // Update cache
      if (role) {
        this.cachedRole = {
          userId,
          role,
          expiresAt: now + this.ROLE_CACHE_TTL
        };
      }

      return role;
    } catch (error) {
      console.warn('[FluxbaseJobs] Failed to fetch user role:', error);
      return null;
    }
  }

  /**
   * Submit a new job for execution
   *
   * @param jobName - Name of the job function to execute
   * @param payload - Job input data
   * @param options - Additional options (priority, namespace, scheduled time, onBehalfOf)
   * @returns Promise resolving to { data, error } tuple with submitted job details
   *
   * @example
   * ```typescript
   * // Submit a simple job
   * const { data, error } = await client.jobs.submit('send-email', {
   *   to: 'user@example.com',
   *   subject: 'Hello',
   *   body: 'Welcome!'
   * })
   *
   * if (data) {
   *   console.log('Job submitted:', data.id)
   *   console.log('Status:', data.status)
   * }
   *
   * // Submit with priority
   * const { data } = await client.jobs.submit('high-priority-task', payload, {
   *   priority: 10
   * })
   *
   * // Schedule for later
   * const { data } = await client.jobs.submit('scheduled-task', payload, {
   *   scheduled: '2025-01-01T00:00:00Z'
   * })
   *
   * // Submit on behalf of a user (service_role only)
   * const { data } = await serviceClient.jobs.submit('user-task', payload, {
   *   onBehalfOf: {
   *     user_id: 'user-uuid',
   *     user_email: 'user@example.com'
   *   }
   * })
   * ```
   */
  async submit(
    jobName: string,
    payload?: unknown,
    options?: {
      priority?: number;
      namespace?: string;
      scheduled?: string;
      /**
       * Submit job on behalf of another user (service_role only).
       * The job will be created with the specified user's identity,
       * allowing them to see the job and its logs via RLS.
       *
       * If not provided, the current user's identity and role from user_profiles
       * will be automatically included.
       */
      onBehalfOf?: OnBehalfOf;
    },
  ): Promise<{ data: Job | null; error: Error | null }> {
    try {
      let finalOnBehalfOf = options?.onBehalfOf;

      // Only auto-populate onBehalfOf if this is a service role client
      // Regular authenticated users cannot use on_behalf_of (backend rejects it)
      if (this.isServiceRole && !finalOnBehalfOf) {
        // Get current user info to submit on their behalf
        const userResponse = await this.fetch.get<{ user: User }>("/api/v1/auth/user");
        if (userResponse?.user) {
          const user = userResponse.user;

          // Fetch user's role from user_profiles
          const role = await this.getCurrentUserRole(user.id);

          // Populate onBehalfOf with current user's identity and role
          finalOnBehalfOf = {
            user_id: user.id,
            user_email: user.email,
            user_role: role || undefined // Include role if found
          };
        }
      }

      const request: SubmitJobRequest = {
        job_name: jobName,
        payload,
        priority: options?.priority,
        namespace: options?.namespace,
        scheduled: options?.scheduled,
        on_behalf_of: finalOnBehalfOf,
      };

      const data = await this.fetch.post<Job>("/api/v1/jobs/submit", request);
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get status and details of a specific job
   *
   * @param jobId - Job ID
   * @returns Promise resolving to { data, error } tuple with job details
   *
   * @example
   * ```typescript
   * const { data: job, error } = await client.jobs.get('550e8400-e29b-41d4-a716-446655440000')
   *
   * if (job) {
   *   console.log('Status:', job.status)
   *   console.log('Progress:', job.progress_percent + '%')
   *   console.log('Result:', job.result)
   *   console.log('Logs:', job.logs)
   * }
   * ```
   */
  async get(jobId: string): Promise<{ data: Job | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<Job>(`/api/v1/jobs/${jobId}`);
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * List jobs submitted by the current user
   *
   * @param filters - Optional filters (status, namespace, limit, offset)
   * @returns Promise resolving to { data, error } tuple with array of jobs
   *
   * @example
   * ```typescript
   * // List all your jobs
   * const { data: jobs, error } = await client.jobs.list()
   *
   * // Filter by status
   * const { data: running } = await client.jobs.list({
   *   status: 'running'
   * })
   *
   * // Paginate
   * const { data: page } = await client.jobs.list({
   *   limit: 20,
   *   offset: 40
   * })
   * ```
   */
  async list(filters?: {
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
        `/api/v1/jobs${queryString ? `?${queryString}` : ""}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Cancel a pending or running job
   *
   * @param jobId - Job ID to cancel
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { error } = await client.jobs.cancel('550e8400-e29b-41d4-a716-446655440000')
   *
   * if (!error) {
   *   console.log('Job cancelled successfully')
   * }
   * ```
   */
  async cancel(jobId: string): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.post(`/api/v1/jobs/${jobId}/cancel`, {});
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Retry a failed job
   *
   * Creates a new job execution with the same parameters
   *
   * @param jobId - Job ID to retry
   * @returns Promise resolving to { data, error } tuple with new job
   *
   * @example
   * ```typescript
   * const { data: newJob, error } = await client.jobs.retry('550e8400-e29b-41d4-a716-446655440000')
   *
   * if (newJob) {
   *   console.log('Job retried, new ID:', newJob.id)
   * }
   * ```
   */
  async retry(jobId: string): Promise<{ data: Job | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<Job>(`/api/v1/jobs/${jobId}/retry`, {});
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get execution logs for a job
   *
   * Returns logs for the specified job. Only returns logs for jobs
   * owned by the authenticated user (unless using service_role).
   *
   * @param jobId - Job ID
   * @param afterLine - Optional line number to get logs after (for polling/streaming)
   * @returns Promise resolving to { data, error } tuple with execution logs
   *
   * @example
   * ```typescript
   * // Get all logs for a job
   * const { data: logs, error } = await client.jobs.getLogs('550e8400-e29b-41d4-a716-446655440000')
   *
   * if (logs) {
   *   for (const log of logs) {
   *     console.log(`[${log.level}] ${log.message}`)
   *   }
   * }
   *
   * // Backfill + stream pattern
   * const { data: logs } = await client.jobs.getLogs(jobId)
   * let lastLine = Math.max(...(logs?.map(l => l.line_number) ?? []), 0)
   *
   * const channel = client.realtime
   *   .executionLogs(jobId, 'job')
   *   .onLog((log) => {
   *     if (log.line_number > lastLine) {
   *       displayLog(log)
   *       lastLine = log.line_number
   *     }
   *   })
   *   .subscribe()
   * ```
   */
  async getLogs(
    jobId: string,
    afterLine?: number,
  ): Promise<{ data: ExecutionLog[] | null; error: Error | null }> {
    try {
      const params = afterLine !== undefined ? `?after_line=${afterLine}` : "";
      const response = await this.fetch.get<{ logs: ExecutionLog[]; count: number }>(
        `/api/v1/jobs/${jobId}/logs${params}`,
      );
      return { data: response.logs || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }
}
