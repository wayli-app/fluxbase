/**
 * Type definitions for Fluxbase job runtime environment
 * Use this for TypeScript autocompletion when writing job functions
 */

/**
 * Global Fluxbase API available in job runtime
 */
declare const Fluxbase: {
  /**
   * Get the current job execution context
   * Includes job metadata, user information, and payload
   */
  getJobContext(): JobContext;

  /**
   * Report job progress (0-100)
   * Updates are stored in the database and can be monitored by clients
   *
   * @param percent - Progress percentage (0-100)
   * @param message - Optional progress message
   * @param data - Optional progress data (any JSON-serializable object)
   */
  reportProgress(
    percent: number,
    message?: string,
    data?: any
  ): Promise<void>;

  /**
   * Access the Fluxbase database with Supabase-compatible API
   * All queries automatically use the job's user context for RLS
   */
  database(): FluxbaseDatabase;

  /**
   * Access the Fluxbase storage API
   * Supabase-compatible storage client
   */
  storage(): FluxbaseStorage;
};

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

  /** User role (e.g., "user", "admin", "dashboard_admin") */
  role: string;
}

/**
 * Fluxbase database client (Supabase-compatible)
 */
interface FluxbaseDatabase {
  /**
   * Query builder for database operations
   * @param table - Table name (e.g., "app.my_table")
   */
  from(table: string): FluxbaseQueryBuilder;
}

/**
 * Database query builder
 */
interface FluxbaseQueryBuilder {
  /** Select columns */
  select(
    columns?: string,
    options?: { count?: "exact" | "planned" | "estimated"; head?: boolean }
  ): this;

  /** Insert rows */
  insert(data: any | any[]): this;

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
  is(column: string, value: null): this;

  /** Order results */
  order(column: string, options?: { ascending?: boolean }): this;

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
   * @param bucket - Bucket name
   * @param options - Bucket options
   */
  createBucket(
    bucket: string,
    options?: { public?: boolean; fileSizeLimit?: number }
  ): Promise<StorageResponse<{ name: string }>>;

  /**
   * List all buckets
   */
  listBuckets(): Promise<StorageResponse<StorageBucket[]>>;

  /**
   * Get bucket metadata
   * @param bucket - Bucket name
   */
  getBucket(bucket: string): Promise<StorageResponse<StorageBucket>>;

  /**
   * Delete a bucket
   * @param bucket - Bucket name
   */
  deleteBucket(bucket: string): Promise<StorageResponse<{ message: string }>>;
}

/**
 * Storage bucket operations
 */
interface FluxbaseStorageBucket {
  /**
   * Upload a file
   * @param path - File path in bucket
   * @param data - File data (Uint8Array, ArrayBuffer, File, or Blob)
   * @param options - Upload options
   */
  upload(
    path: string,
    data: Uint8Array | ArrayBuffer | File | Blob,
    options?: {
      contentType?: string;
      cacheControl?: string;
      upsert?: boolean;
    }
  ): Promise<StorageResponse<{ path: string }>>;

  /**
   * Download a file
   * @param path - File path in bucket
   */
  download(path: string): Promise<StorageResponse<Blob>>;

  /**
   * List files in bucket
   * @param path - Directory path (optional)
   * @param options - List options
   */
  list(
    path?: string,
    options?: {
      limit?: number;
      offset?: number;
      sortBy?: { column: string; order: "asc" | "desc" };
    }
  ): Promise<StorageResponse<StorageFile[]>>;

  /**
   * Delete files
   * @param paths - Array of file paths to delete
   */
  remove(paths: string[]): Promise<StorageResponse<{ message: string }>>;

  /**
   * Move a file
   * @param fromPath - Source path
   * @param toPath - Destination path
   */
  move(fromPath: string, toPath: string): Promise<StorageResponse<{ message: string }>>;

  /**
   * Copy a file
   * @param fromPath - Source path
   * @param toPath - Destination path
   */
  copy(fromPath: string, toPath: string): Promise<StorageResponse<{ message: string }>>;

  /**
   * Create signed URL for file access
   * @param path - File path
   * @param expiresIn - Expiration time in seconds
   */
  createSignedUrl(
    path: string,
    expiresIn: number
  ): Promise<StorageResponse<{ signedUrl: string }>>;

  /**
   * Get public URL for a file (bucket must be public)
   * @param path - File path
   */
  getPublicUrl(path: string): { data: { publicUrl: string } };
}

/**
 * Storage response wrapper
 */
interface StorageResponse<T> {
  /** Response data */
  data: T | null;

  /** Error if operation failed */
  error: StorageError | null;
}

/**
 * Storage error
 */
interface StorageError {
  /** Error message */
  message: string;

  /** Error code */
  code?: string;
}

/**
 * Storage bucket metadata
 */
interface StorageBucket {
  /** Bucket ID */
  id: string;

  /** Bucket name */
  name: string;

  /** Is bucket public */
  public: boolean;

  /** File size limit in bytes */
  file_size_limit?: number;

  /** Created timestamp */
  created_at: string;

  /** Updated timestamp */
  updated_at: string;
}

/**
 * Storage file metadata
 */
interface StorageFile {
  /** File name */
  name: string;

  /** File ID */
  id: string;

  /** File size in bytes */
  size: number;

  /** MIME type */
  mimetype: string;

  /** Last modified timestamp */
  last_modified: string;

  /** Created timestamp */
  created_at: string;

  /** Updated timestamp */
  updated_at: string;
}

/**
 * Job handler function signature
 * Export a function named "handler" as the job entry point
 */
export type JobHandler = (req: any) => Promise<any> | any;

/**
 * Example job function:
 *
 * ```typescript
 * export async function handler(req: any) {
 *   const context = Fluxbase.getJobContext()
 *
 *   console.log('User:', context.user?.email)
 *   console.log('Payload:', context.payload)
 *
 *   await Fluxbase.reportProgress(50, 'Processing...')
 *
 *   const { data, error } = await Fluxbase.database()
 *     .from('app.my_table')
 *     .select('*')
 *
 *   if (error) throw new Error(error.message)
 *
 *   await Fluxbase.reportProgress(100, 'Complete')
 *
 *   return { success: true, count: data?.length }
 * }
 * ```
 */

/**
 * Deno global APIs available in job runtime
 */
declare const Deno: {
  /**
   * Environment variables (FLUXBASE_* only)
   */
  env: {
    get(key: string): string | undefined;
    set(key: string, value: string): void;
    delete(key: string): void;
    has(key: string): boolean;
  };

  /**
   * Read a file (if allow_read is enabled)
   */
  readFile(path: string): Promise<Uint8Array>;

  /**
   * Read a text file (if allow_read is enabled)
   */
  readTextFile(path: string): Promise<string>;

  /**
   * Write a file (if allow_write is enabled)
   */
  writeFile(path: string, data: Uint8Array): Promise<void>;

  /**
   * Write a text file (if allow_write is enabled)
   */
  writeTextFile(path: string, data: string): Promise<void>;

  /**
   * Make HTTP requests (if allow_net is enabled)
   */
  fetch: typeof fetch;
};
