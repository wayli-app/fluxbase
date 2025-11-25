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
import { FluxbaseAdmin } from "./admin";
import { FluxbaseManagement } from "./management";
import { SettingsClient } from "./settings";
import { QueryBuilder } from "./query-builder";
import type { FluxbaseClientOptions } from "./types";

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
   * Call a PostgreSQL function (Remote Procedure Call)
   *
   * @param functionName - The name of the PostgreSQL function to call
   * @param params - Optional parameters to pass to the function
   * @returns Promise containing the function result or error
   *
   * @example
   * ```typescript
   * // Call a function without parameters
   * const { data, error } = await client.rpc('get_total_users')
   *
   * // Call a function with parameters
   * const { data, error } = await client.rpc('calculate_discount', {
   *   product_id: 123,
   *   coupon_code: 'SAVE20'
   * })
   * ```
   *
   * @category Database
   */
  async rpc<T = any>(
    functionName: string,
    params?: Record<string, unknown>,
  ): Promise<{ data: T | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<T>(
        `/api/v1/rpc/${functionName}`,
        params || {},
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
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

/**
 * Create a new Fluxbase client instance (Supabase-compatible)
 *
 * This function signature is identical to Supabase's createClient, making migration seamless.
 *
 * @param fluxbaseUrl - The URL of your Fluxbase instance
 * @param fluxbaseKey - The anon key (JWT token with "anon" role). Generate using: `./scripts/generate-keys.sh` (option 3)
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
  fluxbaseUrl: string,
  fluxbaseKey: string,
  options?: FluxbaseClientOptions,
): FluxbaseClient<Database, SchemaName> {
  return new FluxbaseClient<Database, SchemaName>(
    fluxbaseUrl,
    fluxbaseKey,
    options,
  );
}
