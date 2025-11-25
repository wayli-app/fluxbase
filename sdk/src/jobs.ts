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
import type { Job, SubmitJobRequest } from "./types";

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

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
  }

  /**
   * Submit a new job for execution
   *
   * @param jobName - Name of the job function to execute
   * @param payload - Job input data
   * @param options - Additional options (priority, namespace, scheduled time)
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
   * ```
   */
  async submit(
    jobName: string,
    payload?: any,
    options?: {
      priority?: number;
      namespace?: string;
      scheduled?: string;
    },
  ): Promise<{ data: Job | null; error: Error | null }> {
    try {
      const request: SubmitJobRequest = {
        job_name: jobName,
        payload,
        ...options,
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
  }): Promise<{ data: Job[] | null; error: Error | null }> {
    try {
      const params = new URLSearchParams();
      if (filters?.status) params.append("status", filters.status);
      if (filters?.namespace) params.append("namespace", filters.namespace);
      if (filters?.limit) params.append("limit", filters.limit.toString());
      if (filters?.offset) params.append("offset", filters.offset.toString());

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
}
