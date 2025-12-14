/**
 * Main Fluxbase client for interacting with the Fluxbase backend.
 *
 * This client provides access to all Fluxbase features including:
 * - Database operations via PostgREST-compatible API
 * - Authentication and user management
 * - Real-time subscriptions via WebSockets
 * - File storage and management
 * - Edge functions for serverless compute
 * - PostgreSQL function calls (RPC)
 *
 * @example
 * ```typescript
 * import { createClient } from '@fluxbase/sdk'
 *
 * const client = createClient({
 *   url: 'http://localhost:8080',
 *   auth: {
 *     token: 'your-jwt-token',
 *     autoRefresh: true
 *   }
 * })
 *
 * // Query database
 * const { data } = await client.from('users').select('*').execute()
 *
 * // Invoke edge function
 * const { data, error } = await client.functions.invoke('hello-world', {
 *   body: { name: 'Alice' }
 * })
 *
 * // Subscribe to realtime changes
 * client.realtime.subscribe('users', (payload) => {
 *   console.log('Change:', payload)
 * })
 * ```
 *
 * @category Client
 */

import { FluxbaseFetch } from "./fetch";
import { FluxbaseAuth } from "./auth";
import { FluxbaseRealtime } from "./realtime";
import { FluxbaseStorage } from "./storage";
import { FluxbaseFunctions } from "./functions";
import { FluxbaseJobs } from "./jobs";
import { FluxbaseRPC } from "./rpc";
import { FluxbaseAdmin } from "./admin";
import { FluxbaseManagement } from "./management";
import { SettingsClient } from "./settings";
import { FluxbaseAI } from "./ai";
import { QueryBuilder } from "./query-builder";
import { SchemaQueryBuilder } from "./schema-query-builder";
import type { FluxbaseClientOptions } from "./types";

/**
 * Callable RPC type - can be called directly (Supabase-compatible) or access methods
 * @category RPC
 */
export type CallableRPC = {
  /**
   * Call a PostgreSQL function (RPC) - Supabase compatible
   * Uses 'default' namespace
   *
   * @param fn - Function name
   * @param params - Function parameters
   * @returns Promise with data or error
   *
   * @example
   * ```typescript
   * const { data, error } = await client.rpc('get_user_orders', { user_id: '123' })
   * ```
   */
  <T = any>(
    fn: string,
    params?: Record<string, unknown>,
  ): Promise<{ data: T | null; error: Error | null }>;
} & FluxbaseRPC;

/**
 * Main Fluxbase client class
 * @category Client
 */
export class FluxbaseClient<
  Database = any,
  _SchemaName extends string & keyof Database = any,
> {
  /** Internal HTTP client for making requests */
  private fetch: FluxbaseFetch;

  /** Authentication module for user management */
  public auth: FluxbaseAuth;

  /** Realtime module for WebSocket subscriptions */
  public realtime: FluxbaseRealtime;

  /** Storage module for file operations */
  public storage: FluxbaseStorage;

  /** Functions module for invoking and managing edge functions */
  public functions: FluxbaseFunctions;

  /** Jobs module for submitting and monitoring background jobs */
  public jobs: FluxbaseJobs;

  /** Admin module for instance management (requires admin authentication) */
  public admin: FluxbaseAdmin;

  /** Management module for API keys, webhooks, and invitations */
  public management: FluxbaseManagement;

  /** Settings module for reading public application settings (respects RLS policies) */
  public settings: SettingsClient;

  /** AI module for chatbots and conversation history */
  public ai: FluxbaseAI;

  /**
   * RPC module for calling PostgreSQL functions - Supabase compatible
   *
   * Can be called directly (Supabase-style) or access methods like invoke(), list(), getStatus()
   *
   * @example
   * ```typescript
   * // Supabase-style direct call (uses 'default' namespace)
   * const { data, error } = await client.rpc('get_user_orders', { user_id: '123' })
   *
   * // With full options
   * const { data, error } = await client.rpc.invoke('get_user_orders', { user_id: '123' }, {
   *   namespace: 'custom',
   *   async: true
   * })
   *
   * // List available procedures
   * const { data: procedures } = await client.rpc.list()
   * ```
   *
   * @category RPC
   */
  public rpc: CallableRPC;

  /**
   * Create a new Fluxbase client instance
   *
   * @param fluxbaseUrl - The URL of your Fluxbase instance
   * @param fluxbaseKey - The anon key (JWT token with "anon" role). Generate using scripts/generate-keys.sh
   * @param options - Additional client configuration options
   *
   * @example
   * ```typescript
   * const client = new FluxbaseClient(
   *   'http://localhost:8080',
   *   'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...',  // Anon JWT token
   *   { timeout: 30000 }
   * )
   * ```
   */
  constructor(
    protected fluxbaseUrl: string,
    protected fluxbaseKey: string,
    options?: FluxbaseClientOptions,
  ) {
    // Prepare headers with anon key
    const headers = {
      apikey: fluxbaseKey,
      Authorization: `Bearer ${fluxbaseKey}`,
      ...options?.headers,
    };

    // Initialize HTTP client
    this.fetch = new FluxbaseFetch(fluxbaseUrl, {
      headers,
      timeout: options?.timeout,
      debug: options?.debug,
    });

    // Store anon key for auth restoration (when user signs out, restore anon key auth)
    this.fetch.setAnonKey(fluxbaseKey);

    // Initialize auth module
    this.auth = new FluxbaseAuth(
      this.fetch,
      options?.auth?.autoRefresh ?? true,
      options?.auth?.persist ?? true,
    );

    // Set auth token if provided
    if (options?.auth?.token) {
      this.fetch.setAuthToken(options.auth.token);
    }

    // Initialize realtime module
    this.realtime = new FluxbaseRealtime(
      fluxbaseUrl,
      options?.auth?.token || null,
    );

    // Initialize storage module
    this.storage = new FluxbaseStorage(this.fetch);

    // Initialize functions module
    this.functions = new FluxbaseFunctions(this.fetch);

    // Initialize jobs module
    this.jobs = new FluxbaseJobs(this.fetch);

    // Initialize admin module
    this.admin = new FluxbaseAdmin(this.fetch);

    // Initialize management module
    this.management = new FluxbaseManagement(this.fetch);

    // Initialize settings module (public read-only access with RLS)
    this.settings = new SettingsClient(this.fetch);

    // Initialize AI module
    // Convert HTTP URL to WebSocket URL (http(s) -> ws(s))
    const wsProtocol = fluxbaseUrl.startsWith("https") ? "wss" : "ws";
    const wsBaseUrl = fluxbaseUrl.replace(/^https?:/, wsProtocol + ":");
    this.ai = new FluxbaseAI(this.fetch, wsBaseUrl);

    // Initialize RPC module with callable wrapper (Supabase-compatible)
    const rpcInstance = new FluxbaseRPC(this.fetch);

    // Create callable function that wraps invoke() for Supabase-style calls
    const rpcCallable = async <T = any>(
      fn: string,
      params?: Record<string, unknown>,
    ): Promise<{ data: T | null; error: Error | null }> => {
      const result = await rpcInstance.invoke<T>(fn, params);
      return {
        data: (result.data?.result as T) ?? null,
        error: result.error,
      };
    };

    // Attach FluxbaseRPC methods to callable function
    Object.assign(rpcCallable, {
      invoke: rpcInstance.invoke.bind(rpcInstance),
      list: rpcInstance.list.bind(rpcInstance),
      getStatus: rpcInstance.getStatus.bind(rpcInstance),
      getLogs: rpcInstance.getLogs.bind(rpcInstance),
      waitForCompletion: rpcInstance.waitForCompletion.bind(rpcInstance),
    });

    this.rpc = rpcCallable as CallableRPC;

    // Subscribe to auth changes to update realtime token
    this.setupAuthSync();
  }

  /**
   * Create a query builder for a database table
   *
   * @param table - The table name (can include schema, e.g., 'public.users')
   * @returns A query builder instance for constructing and executing queries
   *
   * @example
   * ```typescript
   * // Simple select
   * const { data } = await client.from('users').select('*').execute()
   *
   * // With filters
   * const { data } = await client.from('products')
   *   .select('id, name, price')
   *   .gt('price', 100)
   *   .eq('category', 'electronics')
   *   .execute()
   *
   * // Insert
   * await client.from('users').insert({ name: 'John', email: 'john@example.com' }).execute()
   * ```
   *
   * @category Database
   */
  from<T = any>(table: string): QueryBuilder<T> {
    return new QueryBuilder<T>(this.fetch, table);
  }

  /**
   * Access a specific database schema
   *
   * Use this to query tables in non-public schemas.
   *
   * @param schemaName - The schema name (e.g., 'jobs', 'analytics')
   * @returns A schema query builder for constructing queries on that schema
   *
   * @example
   * ```typescript
   * // Query the jobs.execution_logs table
   * const { data } = await client
   *   .schema('jobs')
   *   .from('execution_logs')
   *   .select('*')
   *   .eq('job_id', jobId)
   *   .execute()
   *
   * // Insert into a custom schema table
   * await client
   *   .schema('analytics')
   *   .from('events')
   *   .insert({ event_type: 'click', data: {} })
   *   .execute()
   * ```
   *
   * @category Database
   */
  schema(schemaName: string): SchemaQueryBuilder {
    return new SchemaQueryBuilder(this.fetch, schemaName);
  }

  /**
   * Sync auth state with realtime connections
   * @internal
   */
  private setupAuthSync() {
    // When auth token changes, update realtime
    const originalSetAuthToken = this.fetch.setAuthToken.bind(this.fetch);
    this.fetch.setAuthToken = (token: string | null) => {
      originalSetAuthToken(token);
      this.realtime.setAuth(token);
    };

    // Set up token refresh callback for realtime connections
    // This is called when WebSocket connections detect an expired token
    this.realtime.setTokenRefreshCallback(async () => {
      const result = await this.auth.refreshSession();
      if (result.error || !result.data?.session) {
        console.error(
          "[Fluxbase] Failed to refresh token for realtime:",
          result.error,
        );
        return null;
      }
      return result.data.session.access_token;
    });
  }

  /**
   * Get the current authentication token
   *
   * @returns The current JWT access token, or null if not authenticated
   *
   * @category Authentication
   */
  getAuthToken(): string | null {
    return this.auth.getAccessToken();
  }

  /**
   * Set a new authentication token
   *
   * This updates both the HTTP client and realtime connection with the new token.
   *
   * @param token - The JWT access token to set, or null to clear authentication
   *
   * @category Authentication
   */
  setAuthToken(token: string | null) {
    this.fetch.setAuthToken(token);
    this.realtime.setAuth(token);
  }

  /**
   * Create or get a realtime channel (Supabase-compatible)
   *
   * @param name - Channel name
   * @param config - Optional channel configuration
   * @returns RealtimeChannel instance
   *
   * @example
   * ```typescript
   * const channel = client.channel('room-1', {
   *   broadcast: { self: true },
   *   presence: { key: 'user-123' }
   * })
   *   .on('broadcast', { event: 'message' }, (payload) => {
   *     console.log('Message:', payload)
   *   })
   *   .subscribe()
   * ```
   *
   * @category Realtime
   */
  channel(name: string, config?: import("./types").RealtimeChannelConfig) {
    return this.realtime.channel(name, config);
  }

  /**
   * Remove a realtime channel (Supabase-compatible)
   *
   * @param channel - The channel to remove
   * @returns Promise resolving to status
   *
   * @example
   * ```typescript
   * const channel = client.channel('room-1')
   * await client.removeChannel(channel)
   * ```
   *
   * @category Realtime
   */
  removeChannel(channel: import("./realtime").RealtimeChannel) {
    return this.realtime.removeChannel(channel);
  }

  /**
   * Get the internal HTTP client
   *
   * Use this for advanced scenarios like making custom API calls or admin operations.
   *
   * @returns The internal FluxbaseFetch instance
   *
   * @example
   * ```typescript
   * // Make a custom API call
   * const data = await client.http.get('/api/custom-endpoint')
   * ```
   *
   * @category Advanced
   */
  get http(): FluxbaseFetch {
    return this.fetch;
  }
}

// Deno type declaration for env access
declare const Deno:
  | {
      env: {
        get(name: string): string | undefined;
      };
    }
  | undefined;

/**
 * Get environment variable (works in Node.js, Deno, and browser with globalThis)
 * @internal
 */
function getEnvVar(name: string): string | undefined {
  // Node.js
  if (typeof process !== "undefined" && process.env) {
    return process.env[name];
  }
  // Deno
  if (typeof Deno !== "undefined" && Deno?.env) {
    return Deno.env.get(name);
  }
  return undefined;
}

/**
 * Create a new Fluxbase client instance (Supabase-compatible)
 *
 * This function signature is identical to Supabase's createClient, making migration seamless.
 *
 * When called without arguments (or with undefined values), the function will attempt to
 * read from environment variables:
 * - `FLUXBASE_URL` - The URL of your Fluxbase instance
 * - `FLUXBASE_ANON_KEY` or `FLUXBASE_JOB_TOKEN` or `FLUXBASE_SERVICE_TOKEN` - The API key/token
 *
 * This is useful in:
 * - Server-side environments where env vars are set
 * - Fluxbase job functions where tokens are automatically provided
 * - Edge functions with configured environment
 *
 * @param fluxbaseUrl - The URL of your Fluxbase instance (optional if FLUXBASE_URL env var is set)
 * @param fluxbaseKey - The anon key or JWT token (optional if env var is set)
 * @param options - Optional client configuration
 * @returns A configured Fluxbase client instance with full TypeScript support
 *
 * @example
 * ```typescript
 * import { createClient } from '@fluxbase/sdk'
 *
 * // Initialize with anon key (identical to Supabase)
 * const client = createClient(
 *   'http://localhost:8080',
 *   'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'  // Anon JWT token
 * )
 *
 * // With additional options
 * const client = createClient(
 *   'http://localhost:8080',
 *   'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...',
 *   { timeout: 30000, debug: true }
 * )
 *
 * // In a Fluxbase job function (reads from env vars automatically)
 * const client = createClient()
 *
 * // With TypeScript database types
 * const client = createClient<Database>(
 *   'http://localhost:8080',
 *   'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'
 * )
 * ```
 *
 * @category Client
 */
export function createClient<
  Database = any,
  SchemaName extends string & keyof Database = any,
>(
  fluxbaseUrl?: string,
  fluxbaseKey?: string,
  options?: FluxbaseClientOptions,
): FluxbaseClient<Database, SchemaName> {
  // Resolve URL from argument or environment variable
  const url =
    fluxbaseUrl ||
    getEnvVar("FLUXBASE_URL") ||
    getEnvVar("NEXT_PUBLIC_FLUXBASE_URL") ||
    getEnvVar("VITE_FLUXBASE_URL");

  // Resolve key from argument or environment variables (try multiple common names)
  const key =
    fluxbaseKey ||
    getEnvVar("FLUXBASE_ANON_KEY") ||
    getEnvVar("FLUXBASE_SERVICE_TOKEN") ||
    getEnvVar("FLUXBASE_JOB_TOKEN") ||
    getEnvVar("NEXT_PUBLIC_FLUXBASE_ANON_KEY") ||
    getEnvVar("VITE_FLUXBASE_ANON_KEY");

  if (!url) {
    throw new Error(
      "Fluxbase URL is required. Pass it as the first argument or set FLUXBASE_URL environment variable.",
    );
  }

  if (!key) {
    throw new Error(
      "Fluxbase key is required. Pass it as the second argument or set FLUXBASE_ANON_KEY environment variable.",
    );
  }

  return new FluxbaseClient<Database, SchemaName>(url, key, options);
}
