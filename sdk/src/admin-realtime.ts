import type { FluxbaseFetch } from "./fetch";
import type {
  EnableRealtimeRequest,
  EnableRealtimeResponse,
  RealtimeTableStatus,
  ListRealtimeTablesResponse,
  UpdateRealtimeConfigRequest,
} from "./types";

/**
 * Realtime Admin Manager
 *
 * Provides methods for enabling and managing realtime subscriptions on database tables.
 * When enabled, changes to a table (INSERT, UPDATE, DELETE) are automatically broadcast
 * to WebSocket subscribers.
 *
 * @example
 * ```typescript
 * const realtime = client.admin.realtime
 *
 * // Enable realtime on a table
 * await realtime.enableRealtime('products')
 *
 * // Enable with options
 * await realtime.enableRealtime('orders', {
 *   events: ['INSERT', 'UPDATE'],
 *   exclude: ['internal_notes', 'raw_data']
 * })
 *
 * // List all realtime-enabled tables
 * const { tables } = await realtime.listTables()
 *
 * // Check status of a specific table
 * const status = await realtime.getStatus('public', 'products')
 *
 * // Disable realtime
 * await realtime.disableRealtime('public', 'products')
 * ```
 */
export class FluxbaseAdminRealtime {
  constructor(private fetch: FluxbaseFetch) {}

  /**
   * Enable realtime on a table
   *
   * Creates the necessary database triggers to broadcast changes to WebSocket subscribers.
   * Also sets REPLICA IDENTITY FULL to include old values in UPDATE/DELETE events.
   *
   * @param table - Table name to enable realtime on
   * @param options - Optional configuration
   * @returns Promise resolving to EnableRealtimeResponse
   *
   * @example
   * ```typescript
   * // Enable realtime on products table (all events)
   * await client.admin.realtime.enableRealtime('products')
   *
   * // Enable on a specific schema
   * await client.admin.realtime.enableRealtime('orders', {
   *   schema: 'sales'
   * })
   *
   * // Enable specific events only
   * await client.admin.realtime.enableRealtime('audit_log', {
   *   events: ['INSERT'] // Only broadcast inserts
   * })
   *
   * // Exclude large columns from notifications
   * await client.admin.realtime.enableRealtime('posts', {
   *   exclude: ['content', 'raw_html'] // Skip these in payload
   * })
   * ```
   */
  async enableRealtime(
    table: string,
    options?: {
      schema?: string;
      events?: ("INSERT" | "UPDATE" | "DELETE")[];
      exclude?: string[];
    },
  ): Promise<EnableRealtimeResponse> {
    const request: EnableRealtimeRequest = {
      schema: options?.schema ?? "public",
      table,
      events: options?.events,
      exclude: options?.exclude,
    };
    return await this.fetch.post<EnableRealtimeResponse>(
      "/api/v1/admin/realtime/tables",
      request,
    );
  }

  /**
   * Disable realtime on a table
   *
   * Removes the realtime trigger from a table. Existing subscribers will stop
   * receiving updates for this table.
   *
   * @param schema - Schema name
   * @param table - Table name
   * @returns Promise resolving to success message
   *
   * @example
   * ```typescript
   * await client.admin.realtime.disableRealtime('public', 'products')
   * console.log('Realtime disabled')
   * ```
   */
  async disableRealtime(
    schema: string,
    table: string,
  ): Promise<{ success: boolean; message: string }> {
    return await this.fetch.delete<{ success: boolean; message: string }>(
      `/api/v1/admin/realtime/tables/${encodeURIComponent(schema)}/${encodeURIComponent(table)}`,
    );
  }

  /**
   * List all realtime-enabled tables
   *
   * Returns a list of all tables that have realtime enabled, along with their
   * configuration (events, excluded columns, etc.).
   *
   * @param options - Optional filter options
   * @returns Promise resolving to ListRealtimeTablesResponse
   *
   * @example
   * ```typescript
   * // List all enabled tables
   * const { tables, count } = await client.admin.realtime.listTables()
   * console.log(`${count} tables have realtime enabled`)
   *
   * tables.forEach(t => {
   *   console.log(`${t.schema}.${t.table}: ${t.events.join(', ')}`)
   * })
   *
   * // Include disabled tables
   * const all = await client.admin.realtime.listTables({ includeDisabled: true })
   * ```
   */
  async listTables(options?: {
    includeDisabled?: boolean;
  }): Promise<ListRealtimeTablesResponse> {
    const params = options?.includeDisabled ? "?enabled=false" : "";
    return await this.fetch.get<ListRealtimeTablesResponse>(
      `/api/v1/admin/realtime/tables${params}`,
    );
  }

  /**
   * Get realtime status for a specific table
   *
   * Returns the realtime configuration for a table, including whether it's enabled,
   * which events are tracked, and which columns are excluded.
   *
   * @param schema - Schema name
   * @param table - Table name
   * @returns Promise resolving to RealtimeTableStatus
   *
   * @example
   * ```typescript
   * const status = await client.admin.realtime.getStatus('public', 'products')
   *
   * if (status.realtime_enabled) {
   *   console.log('Events:', status.events.join(', '))
   *   console.log('Excluded:', status.excluded_columns?.join(', ') || 'none')
   * } else {
   *   console.log('Realtime not enabled')
   * }
   * ```
   */
  async getStatus(schema: string, table: string): Promise<RealtimeTableStatus> {
    return await this.fetch.get<RealtimeTableStatus>(
      `/api/v1/admin/realtime/tables/${encodeURIComponent(schema)}/${encodeURIComponent(table)}`,
    );
  }

  /**
   * Update realtime configuration for a table
   *
   * Modifies the events or excluded columns for a realtime-enabled table
   * without recreating the trigger.
   *
   * @param schema - Schema name
   * @param table - Table name
   * @param config - New configuration
   * @returns Promise resolving to success message
   *
   * @example
   * ```typescript
   * // Change which events are tracked
   * await client.admin.realtime.updateConfig('public', 'products', {
   *   events: ['INSERT', 'UPDATE'] // Stop tracking deletes
   * })
   *
   * // Update excluded columns
   * await client.admin.realtime.updateConfig('public', 'posts', {
   *   exclude: ['raw_content', 'search_vector']
   * })
   *
   * // Clear excluded columns
   * await client.admin.realtime.updateConfig('public', 'posts', {
   *   exclude: [] // Include all columns again
   * })
   * ```
   */
  async updateConfig(
    schema: string,
    table: string,
    config: UpdateRealtimeConfigRequest,
  ): Promise<{ success: boolean; message: string }> {
    return await this.fetch.patch<{ success: boolean; message: string }>(
      `/api/v1/admin/realtime/tables/${encodeURIComponent(schema)}/${encodeURIComponent(table)}`,
      config,
    );
  }
}
