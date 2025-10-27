/**
 * Main Fluxbase client for interacting with the Fluxbase backend.
 *
 * This client provides access to all Fluxbase features including:
 * - Database operations via PostgREST-compatible API
 * - Authentication and user management
 * - Real-time subscriptions via WebSockets
 * - File storage and management
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
 * // Subscribe to realtime changes
 * client.realtime.subscribe('users', (payload) => {
 *   console.log('Change:', payload)
 * })
 * ```
 *
 * @category Client
 */

import { FluxbaseFetch } from './fetch'
import { FluxbaseAuth } from './auth'
import { FluxbaseRealtime } from './realtime'
import { FluxbaseStorage } from './storage'
import { QueryBuilder } from './query-builder'
import type { FluxbaseClientOptions } from './types'

/**
 * Main Fluxbase client class
 * @category Client
 */
export class FluxbaseClient {
  /** Internal HTTP client for making requests */
  private fetch: FluxbaseFetch

  /** Authentication module for user management */
  public auth: FluxbaseAuth

  /** Realtime module for WebSocket subscriptions */
  public realtime: FluxbaseRealtime

  /** Storage module for file operations */
  public storage: FluxbaseStorage

  /**
   * Create a new Fluxbase client instance
   * @param options - Client configuration options
   */
  constructor(options: FluxbaseClientOptions) {
    // Initialize HTTP client
    this.fetch = new FluxbaseFetch(options.url, {
      headers: options.headers,
      timeout: options.timeout,
      debug: options.debug,
    })

    // Initialize auth module
    this.auth = new FluxbaseAuth(
      this.fetch,
      options.auth?.autoRefresh ?? true,
      options.auth?.persist ?? true
    )

    // Set auth token if provided
    if (options.auth?.token) {
      this.fetch.setAuthToken(options.auth.token)
    }

    // Initialize realtime module
    this.realtime = new FluxbaseRealtime(options.url, options.auth?.token || null)

    // Initialize storage module
    this.storage = new FluxbaseStorage(this.fetch)

    // Subscribe to auth changes to update realtime token
    this.setupAuthSync()
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
    return new QueryBuilder<T>(this.fetch, table)
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
    params?: Record<string, unknown>
  ): Promise<{ data: T | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<T>(`/api/v1/rpc/${functionName}`, params || {})
      return { data, error: null }
    } catch (error) {
      return { data: null, error: error as Error }
    }
  }

  /**
   * Sync auth state with realtime connections
   * @internal
   */
  private setupAuthSync() {
    // When auth token changes, update realtime
    const originalSetAuthToken = this.fetch.setAuthToken.bind(this.fetch)
    this.fetch.setAuthToken = (token: string | null) => {
      originalSetAuthToken(token)
      this.realtime.setToken(token)
    }
  }

  /**
   * Get the current authentication token
   *
   * @returns The current JWT access token, or null if not authenticated
   *
   * @category Authentication
   */
  getAuthToken(): string | null {
    return this.auth.getAccessToken()
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
    this.fetch.setAuthToken(token)
    this.realtime.setToken(token)
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
    return this.fetch
  }
}

/**
 * Create a new Fluxbase client instance
 *
 * This is the recommended way to initialize the Fluxbase SDK.
 *
 * @param options - Client configuration options
 * @returns A configured Fluxbase client instance
 *
 * @example
 * ```typescript
 * import { createClient } from '@fluxbase/sdk'
 *
 * const client = createClient({
 *   url: 'http://localhost:8080',
 *   auth: {
 *     token: 'your-jwt-token',
 *     autoRefresh: true,
 *     persist: true
 *   },
 *   timeout: 30000,
 *   debug: false
 * })
 * ```
 *
 * @category Client
 */
export function createClient(options: FluxbaseClientOptions): FluxbaseClient {
  return new FluxbaseClient(options)
}
