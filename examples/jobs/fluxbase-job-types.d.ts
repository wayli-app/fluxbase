/**
 * Type definitions for Fluxbase job runtime environment
 * Use this for TypeScript autocompletion when writing job functions
 */

/**
 * Fluxbase SDK Client (imported from @fluxbase/sdk)
 * This is the same client used in frontend/backend applications
 */
interface FluxbaseClient {
  /**
   * Query builder for database operations
   * @param table - Table name (e.g., "app.my_table")
   */
  from(table: string): FluxbaseQueryBuilder;

  /**
   * Call a PostgreSQL function
   * @param fn - Function name
   * @param params - Function parameters
   */
  rpc(fn: string, params?: Record<string, any>): Promise<DatabaseResponse<any>>;

  /**
   * Access the storage API
   */
  storage: FluxbaseStorage;

  /**
   * Access the jobs API
   */
  jobs: FluxbaseJobsClient;

  /**
   * Access the auth API (limited in job context)
   */
  auth: FluxbaseAuth;
}

/**
 * Job utilities object passed to handler
 */
interface JobUtils {
  /**
   * Report job progress (0-100)
   * Updates are stored in the database and can be monitored by clients
   *
   * @param percent - Progress percentage (0-100)
   * @param message - Optional progress message
   * @param data - Optional progress data (any JSON-serializable object)
   */
  reportProgress(percent: number, message?: string, data?: any): void;

  /**
   * Check if the job was cancelled by user
   * Call this periodically in long-running jobs to allow graceful cancellation
   */
  checkCancellation(): boolean;

  /**
   * Get the current job execution context
   * Includes job metadata, user information, and payload
   */
  getJobContext(): JobContext;

  /**
   * Get the job payload (convenience method)
   */
  getJobPayload(): any;
}

/**
 * Job execution context
 */
interface JobContext {
  /** Job ID (UUID) */
  job_id: string;

  /** Job function name */
  job_name: string;

  /** Namespace (usually "default") */
  namespace: string;

  /** Current retry count (0 for first attempt) */
  retry_count: number;

  /** Job payload (input data) */
  payload: any;

  /** User context (null if submitted without authentication) */
  user: UserContext | null;
}

/**
 * User context information
 * Available when job was submitted by an authenticated user
 */
interface UserContext {
  /** User ID (UUID) */
  id: string;

  /** User email address */
  email: string;

  /** User role (e.g., "authenticated", "admin") */
  role: string;
}

/**
 * Fluxbase database query builder (Supabase-compatible)
 */
interface FluxbaseQueryBuilder {
  /** Select columns */
  select(
    columns?: string,
    options?: { count?: "exact" | "planned" | "estimated"; head?: boolean }
  ): this;

  /** Insert rows */
  insert(data: any | any[], options?: { onConflict?: string }): this;

  /** Upsert rows */
  upsert(data: any | any[], options?: { onConflict?: string }): this;

  /** Update rows */
  update(data: any): this;

  /** Delete rows */
  delete(): this;

  /** Filter by equality */
  eq(column: string, value: any): this;

  /** Filter by inequality */
  neq(column: string, value: any): this;

  /** Filter greater than */
  gt(column: string, value: any): this;

  /** Filter greater than or equal */
  gte(column: string, value: any): this;

  /** Filter less than */
  lt(column: string, value: any): this;

  /** Filter less than or equal */
  lte(column: string, value: any): this;

  /** Filter by pattern matching */
  like(column: string, pattern: string): this;

  /** Filter by case-insensitive pattern matching */
  ilike(column: string, pattern: string): this;

  /** Filter by inclusion in array */
  in(column: string, values: any[]): this;

  /** Filter by null */
  is(column: string, value: null | boolean): this;

  /** Filter by not null */
  not(column: string, operator: string, value: any): this;

  /** Filter with OR */
  or(filters: string): this;

  /** Order results */
  order(column: string, options?: { ascending?: boolean; nullsFirst?: boolean }): this;

  /** Limit results */
  limit(count: number): this;

  /** Offset results */
  offset(count: number): this;

  /** Range pagination */
  range(from: number, to: number): this;

  /** Get single row */
  single(): Promise<DatabaseResponse<any>>;

  /** Get maybe one row */
  maybeSingle(): Promise<DatabaseResponse<any | null>>;

  /** Execute query and return results */
  then<T>(
    onfulfilled?: (value: DatabaseResponse<T>) => any,
    onrejected?: (reason: any) => any
  ): Promise<any>;
}

/**
 * Database response wrapper
 */
interface DatabaseResponse<T> {
  /** Query result data */
  data: T | null;

  /** Error if query failed */
  error: DatabaseError | null;

  /** Row count (if requested) */
  count?: number;
}

/**
 * Database error
 */
interface DatabaseError {
  /** Error message */
  message: string;

  /** Error code */
  code?: string;

  /** Error details */
  details?: string;

  /** Error hint */
  hint?: string;
}

/**
 * Fluxbase storage client (Supabase-compatible)
 */
interface FluxbaseStorage {
  /**
   * Access a storage bucket
   * @param bucket - Bucket name
   */
  from(bucket: string): FluxbaseStorageBucket;

  /**
   * Create a new bucket (admin only)
   */
  createBucket(
    bucket: string,
    options?: { public?: boolean; fileSizeLimit?: number }
  ): Promise<StorageResponse<{ name: string }>>;

  /** List all buckets */
  listBuckets(): Promise<StorageResponse<StorageBucket[]>>;

  /** Get bucket metadata */
  getBucket(bucket: string): Promise<StorageResponse<StorageBucket>>;

  /** Delete a bucket */
  deleteBucket(bucket: string): Promise<StorageResponse<{ message: string }>>;
}

/**
 * Storage bucket operations
 */
interface FluxbaseStorageBucket {
  /** Upload a file */
  upload(
    path: string,
    data: Uint8Array | ArrayBuffer | File | Blob,
    options?: { contentType?: string; cacheControl?: string; upsert?: boolean }
  ): Promise<StorageResponse<{ path: string }>>;

  /** Download a file */
  download(path: string): Promise<StorageResponse<Blob>>;

  /** List files in bucket */
  list(
    path?: string,
    options?: { limit?: number; offset?: number; sortBy?: { column: string; order: "asc" | "desc" } }
  ): Promise<StorageResponse<StorageFile[]>>;

  /** Delete files */
  remove(paths: string[]): Promise<StorageResponse<{ message: string }>>;

  /** Move a file */
  move(fromPath: string, toPath: string): Promise<StorageResponse<{ message: string }>>;

  /** Copy a file */
  copy(fromPath: string, toPath: string): Promise<StorageResponse<{ message: string }>>;

  /** Create signed URL for file access */
  createSignedUrl(path: string, expiresIn: number): Promise<StorageResponse<{ signedUrl: string }>>;

  /** Get public URL for a file (bucket must be public) */
  getPublicUrl(path: string): { data: { publicUrl: string } };
}

/**
 * Storage response wrapper
 */
interface StorageResponse<T> {
  data: T | null;
  error: StorageError | null;
}

/**
 * Storage error
 */
interface StorageError {
  message: string;
  code?: string;
}

/**
 * Storage bucket metadata
 */
interface StorageBucket {
  id: string;
  name: string;
  public: boolean;
  file_size_limit?: number;
  created_at: string;
  updated_at: string;
}

/**
 * Storage file metadata
 */
interface StorageFile {
  name: string;
  id: string;
  size: number;
  mimetype: string;
  last_modified: string;
  created_at: string;
  updated_at: string;
}

/**
 * Jobs client for submitting follow-up jobs
 */
interface FluxbaseJobsClient {
  /** Submit a new job */
  submit(
    jobName: string,
    payload?: Record<string, any>,
    options?: { namespace?: string; priority?: number; scheduledAt?: Date }
  ): Promise<{ data: { id: string } | null; error: any }>;

  /** Get job status */
  get(jobId: string): Promise<{ data: JobRecord | null; error: any }>;

  /** List jobs */
  list(options?: {
    status?: "pending" | "running" | "completed" | "failed" | "cancelled";
    jobName?: string;
    limit?: number;
  }): Promise<{ data: JobRecord[] | null; error: any }>;

  /** Cancel a job */
  cancel(jobId: string): Promise<{ data: any; error: any }>;
}

/**
 * Job record from database
 */
interface JobRecord {
  id: string;
  job_name: string;
  namespace: string;
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  payload: any;
  result: any;
  progress: { percent: number; message?: string; data?: any } | null;
  created_at: string;
  started_at: string | null;
  completed_at: string | null;
}

/**
 * Auth client (limited in job context)
 */
interface FluxbaseAuth {
  /** Get current session (from job token) */
  getSession(): Promise<{ data: { session: any } | null; error: any }>;

  /** Get current user */
  getUser(): Promise<{ data: { user: any } | null; error: any }>;
}

/**
 * Job handler function signature
 *
 * @param req - HTTP Request object (for compatibility)
 * @param fluxbase - SDK client with user's RLS context (can only access what user can access)
 * @param fluxbaseService - SDK client with service role (bypasses RLS, full access)
 * @param job - Job utilities (reportProgress, getJobContext, checkCancellation)
 */
export type JobHandler = (
  req: Request,
  fluxbase: FluxbaseClient | null,
  fluxbaseService: FluxbaseClient | null,
  job: JobUtils
) => Promise<any> | any;

/**
 * Example job function:
 *
 * ```typescript
 * export async function handler(
 *   req: Request,
 *   fluxbase: FluxbaseClient,
 *   fluxbaseService: FluxbaseClient,
 *   job: JobUtils
 * ) {
 *   const context = job.getJobContext();
 *   console.log('User:', context.user?.email);
 *
 *   // Report progress
 *   job.reportProgress(10, 'Starting...');
 *
 *   // Access user's data (RLS enforced)
 *   const { data: myData } = await fluxbase.from('tracker_data').select('*');
 *
 *   // Access all data (service role, bypasses RLS)
 *   const { data: allData } = await fluxbaseService.from('tracker_data').select('count');
 *
 *   // Upload file to storage
 *   const blob = new Blob(['result'], { type: 'text/plain' });
 *   await fluxbase.storage.from('exports').upload('result.txt', blob);
 *
 *   // Submit follow-up job
 *   await fluxbase.jobs.submit('process-next', { batch: 2 });
 *
 *   job.reportProgress(100, 'Complete');
 *
 *   return { success: true, count: myData?.length };
 * }
 * ```
 */

/**
 * Deno global APIs available in job runtime
 */
declare const Deno: {
  /** Environment variables (FLUXBASE_* only) */
  env: {
    get(key: string): string | undefined;
    set(key: string, value: string): void;
    delete(key: string): void;
    has(key: string): boolean;
  };

  /** Read a file (if allow_read is enabled) */
  readFile(path: string): Promise<Uint8Array>;

  /** Read a text file (if allow_read is enabled) */
  readTextFile(path: string): Promise<string>;

  /** Write a file (if allow_write is enabled) */
  writeFile(path: string, data: Uint8Array): Promise<void>;

  /** Write a text file (if allow_write is enabled) */
  writeTextFile(path: string, data: string): Promise<void>;

  /** Make HTTP requests (if allow_net is enabled) */
  fetch: typeof fetch;
};
