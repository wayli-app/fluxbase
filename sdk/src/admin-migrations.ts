/**
 * Admin Migrations module for managing database migrations
 * Provides API-based migration management without filesystem coupling
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  Migration,
  CreateMigrationRequest,
  UpdateMigrationRequest,
  MigrationExecution,
  SyncMigrationsOptions,
  SyncMigrationsResult,
} from "./types";

/**
 * Admin Migrations manager for database migration operations
 * Provides create, update, delete, apply, rollback, and smart sync operations
 *
 * @category Admin
 */
export class FluxbaseAdminMigrations {
  private fetch: FluxbaseFetch;
  private localMigrations: Map<string, CreateMigrationRequest> = new Map();

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
  }

  /**
   * Register a migration locally for smart sync
   *
   * Call this method to register migrations in your application code.
   * When you call sync(), only new or changed migrations will be sent to the server.
   *
   * @param migration - Migration definition
   * @returns { error } tuple (always succeeds unless validation fails)
   *
   * @example
   * ```typescript
   * // In your app initialization
   * const { error: err1 } = client.admin.migrations.register({
   *   name: '001_create_users_table',
   *   namespace: 'myapp',
   *   up_sql: 'CREATE TABLE app.users (...)',
   *   down_sql: 'DROP TABLE app.users',
   *   description: 'Initial users table'
   * })
   *
   * const { error: err2 } = client.admin.migrations.register({
   *   name: '002_add_posts_table',
   *   namespace: 'myapp',
   *   up_sql: 'CREATE TABLE app.posts (...)',
   *   down_sql: 'DROP TABLE app.posts'
   * })
   *
   * // Sync all registered migrations
   * await client.admin.migrations.sync()
   * ```
   */
  register(migration: CreateMigrationRequest): { error: Error | null } {
    try {
      // Basic validation
      if (!migration.name || !migration.up_sql) {
        return {
          error: new Error("Migration name and up_sql are required"),
        };
      }

      const key = `${migration.namespace || "default"}:${migration.name}`;
      this.localMigrations.set(key, migration);

      return { error: null };
    } catch (error) {
      return { error: error as Error };
    }
  }

  /**
   * Trigger schema refresh to update the REST API cache
   * Note: Server no longer restarts - cache is invalidated instantly
   *
   * @private
   */
  private async triggerSchemaRefresh(): Promise<void> {
    const response = await this.fetch.post<{
      message: string;
      tables: number;
      views: number;
    }>("/api/v1/admin/schema/refresh", {});
    console.log(
      `Schema cache refreshed: ${response.tables} tables, ${response.views} views`
    );
  }

  /**
   * Smart sync all registered migrations
   *
   * Automatically determines which migrations need to be created or updated by:
   * 1. Fetching existing migrations from the server
   * 2. Comparing content hashes to detect changes
   * 3. Only sending new or changed migrations
   *
   * After successful sync, can optionally auto-apply new migrations and refresh
   * the server's schema cache.
   *
   * @param options - Sync options
   * @returns Promise resolving to { data, error } tuple with sync results
   *
   * @example
   * ```typescript
   * // Basic sync (idempotent - safe to call on every app startup)
   * const { data, error } = await client.admin.migrations.sync()
   * if (data) {
   *   console.log(`Created: ${data.summary.created}, Updated: ${data.summary.updated}`)
   * }
   *
   * // Sync with auto-apply (applies new migrations automatically)
   * const { data, error } = await client.admin.migrations.sync({
   *   auto_apply: true
   * })
   *
   * // Dry run to preview changes without applying
   * const { data, error } = await client.admin.migrations.sync({
   *   dry_run: true
   * })
   * ```
   */
  async sync(
    options: Partial<SyncMigrationsOptions> = {}
  ): Promise<{ data: SyncMigrationsResult | null; error: Error | null }> {
    try {
      // Group migrations by namespace
      const byNamespace = new Map<string, CreateMigrationRequest[]>();

      for (const migration of this.localMigrations.values()) {
        const ns = migration.namespace || "default";
        if (!byNamespace.has(ns)) {
          byNamespace.set(ns, []);
        }
        byNamespace.get(ns)!.push(migration);
      }

      // Sync each namespace
      const results: SyncMigrationsResult[] = [];
      const errors: Error[] = [];

      for (const [namespace, migrations] of byNamespace) {
        try {
          const result = await this.fetch.post<SyncMigrationsResult>(
            "/api/v1/admin/migrations/sync",
            {
              namespace,
              migrations: migrations.map((m) => ({
                name: m.name,
                description: m.description,
                up_sql: m.up_sql,
                down_sql: m.down_sql,
              })),
              options: {
                update_if_changed: options.update_if_changed ?? true,
                auto_apply: options.auto_apply ?? false,
                dry_run: options.dry_run ?? false,
              },
            }
          );
          results.push(result);
        } catch (error) {
          // If sync failed with errors (422), extract the sync result from error.details
          const err = error as any;
          if (err.status === 422 && err.details) {
            // Server returned sync results with errors - include them
            results.push(err.details as SyncMigrationsResult);
            errors.push(err);
          } else {
            // Other errors (network, auth, etc.) - propagate them
            throw error;
          }
        }
      }

      // Combine results
      const combined: SyncMigrationsResult = {
        message: results.map((r) => r.message).join("; "),
        namespace: Array.from(byNamespace.keys()).join(", "),
        summary: {
          created: results.reduce((sum, r) => sum + r.summary.created, 0),
          updated: results.reduce((sum, r) => sum + r.summary.updated, 0),
          unchanged: results.reduce((sum, r) => sum + r.summary.unchanged, 0),
          skipped: results.reduce((sum, r) => sum + r.summary.skipped, 0),
          applied: results.reduce((sum, r) => sum + r.summary.applied, 0),
          errors: results.reduce((sum, r) => sum + r.summary.errors, 0),
        },
        details: {
          created: results.flatMap((r) => r.details.created),
          updated: results.flatMap((r) => r.details.updated),
          unchanged: results.flatMap((r) => r.details.unchanged),
          skipped: results.flatMap((r) => r.details.skipped),
          applied: results.flatMap((r) => r.details.applied),
          errors: results.flatMap((r) => r.details.errors),
        },
        dry_run: options.dry_run ?? false,
        warnings: results.flatMap((r) => r.warnings || []),
      };

      // Note: Schema cache is automatically invalidated by the server after migrations are applied.
      // We call triggerSchemaRefresh as a safeguard to ensure cache is up-to-date.
      // This is now instant (no server restart required).
      const migrationsAppliedSuccessfully =
        combined.summary.applied > 0 && combined.summary.errors === 0;
      if (!combined.dry_run && migrationsAppliedSuccessfully) {
        try {
          await this.triggerSchemaRefresh();
        } catch (refreshError) {
          // Log warning but don't fail the sync operation
          // Cache is already invalidated server-side, so this is just a safeguard
          console.warn("Schema refresh warning:", refreshError);
        }
      }

      // If there were errors during sync, return error with full details
      if (errors.length > 0 || combined.summary.errors > 0) {
        const error = new Error(combined.message) as any;
        error.syncResult = combined;
        error.details = combined.details.errors;
        return { data: combined, error };
      }

      return { data: combined, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Create a new migration
   *
   * @param request - Migration configuration
   * @returns Promise resolving to { data, error } tuple with created migration
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.migrations.create({
   *   namespace: 'myapp',
   *   name: '001_create_users',
   *   up_sql: 'CREATE TABLE app.users (id UUID PRIMARY KEY, email TEXT)',
   *   down_sql: 'DROP TABLE app.users',
   *   description: 'Create users table'
   * })
   * ```
   */
  async create(
    request: CreateMigrationRequest
  ): Promise<{ data: Migration | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<Migration>(
        "/api/v1/admin/migrations",
        request
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * List migrations in a namespace
   *
   * @param namespace - Migration namespace (default: 'default')
   * @param status - Filter by status: 'pending', 'applied', 'failed', 'rolled_back'
   * @returns Promise resolving to { data, error } tuple with migrations array
   *
   * @example
   * ```typescript
   * // List all migrations
   * const { data, error } = await client.admin.migrations.list('myapp')
   *
   * // List only pending migrations
   * const { data, error } = await client.admin.migrations.list('myapp', 'pending')
   * ```
   */
  async list(
    namespace: string = "default",
    status?: "pending" | "applied" | "failed" | "rolled_back"
  ): Promise<{ data: Migration[] | null; error: Error | null }> {
    try {
      const params = new URLSearchParams({ namespace });
      if (status) params.append("status", status);

      const data = await this.fetch.get<Migration[]>(
        `/api/v1/admin/migrations?${params.toString()}`
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get details of a specific migration
   *
   * @param name - Migration name
   * @param namespace - Migration namespace (default: 'default')
   * @returns Promise resolving to { data, error } tuple with migration details
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.migrations.get('001_create_users', 'myapp')
   * ```
   */
  async get(
    name: string,
    namespace: string = "default"
  ): Promise<{ data: Migration | null; error: Error | null }> {
    try {
      const params = new URLSearchParams({ namespace });
      const data = await this.fetch.get<Migration>(
        `/api/v1/admin/migrations/${name}?${params.toString()}`
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Update a migration (only if status is pending)
   *
   * @param name - Migration name
   * @param updates - Fields to update
   * @param namespace - Migration namespace (default: 'default')
   * @returns Promise resolving to { data, error } tuple with updated migration
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.migrations.update(
   *   '001_create_users',
   *   { description: 'Updated description' },
   *   'myapp'
   * )
   * ```
   */
  async update(
    name: string,
    updates: UpdateMigrationRequest,
    namespace: string = "default"
  ): Promise<{ data: Migration | null; error: Error | null }> {
    try {
      const params = new URLSearchParams({ namespace });
      const data = await this.fetch.put<Migration>(
        `/api/v1/admin/migrations/${name}?${params.toString()}`,
        updates
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Delete a migration (only if status is pending)
   *
   * @param name - Migration name
   * @param namespace - Migration namespace (default: 'default')
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.migrations.delete('001_create_users', 'myapp')
   * ```
   */
  async delete(
    name: string,
    namespace: string = "default"
  ): Promise<{ data: null; error: Error | null }> {
    try {
      const params = new URLSearchParams({ namespace });
      await this.fetch.delete(
        `/api/v1/admin/migrations/${name}?${params.toString()}`
      );
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Apply a specific migration
   *
   * @param name - Migration name
   * @param namespace - Migration namespace (default: 'default')
   * @returns Promise resolving to { data, error } tuple with result message
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.migrations.apply('001_create_users', 'myapp')
   * if (data) {
   *   console.log(data.message) // "Migration applied successfully"
   * }
   * ```
   */
  async apply(
    name: string,
    namespace: string = "default"
  ): Promise<{ data: { message: string } | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<{ message: string }>(
        `/api/v1/admin/migrations/${name}/apply`,
        { namespace }
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Rollback a specific migration
   *
   * @param name - Migration name
   * @param namespace - Migration namespace (default: 'default')
   * @returns Promise resolving to { data, error } tuple with result message
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.migrations.rollback('001_create_users', 'myapp')
   * ```
   */
  async rollback(
    name: string,
    namespace: string = "default"
  ): Promise<{ data: { message: string } | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<{ message: string }>(
        `/api/v1/admin/migrations/${name}/rollback`,
        { namespace }
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Apply all pending migrations in order
   *
   * @param namespace - Migration namespace (default: 'default')
   * @returns Promise resolving to { data, error } tuple with applied/failed counts
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.migrations.applyPending('myapp')
   * if (data) {
   *   console.log(`Applied: ${data.applied.length}, Failed: ${data.failed.length}`)
   * }
   * ```
   */
  async applyPending(namespace: string = "default"): Promise<{
    data: { message: string; applied: string[]; failed: string[] } | null;
    error: Error | null;
  }> {
    try {
      const data = await this.fetch.post<{
        message: string;
        applied: string[];
        failed: string[];
      }>("/api/v1/admin/migrations/apply-pending", { namespace });
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get execution history for a migration
   *
   * @param name - Migration name
   * @param namespace - Migration namespace (default: 'default')
   * @param limit - Maximum number of executions to return (default: 50, max: 100)
   * @returns Promise resolving to { data, error } tuple with execution records
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.migrations.getExecutions(
   *   '001_create_users',
   *   'myapp',
   *   10
   * )
   * if (data) {
   *   data.forEach(exec => {
   *     console.log(`${exec.executed_at}: ${exec.action} - ${exec.status}`)
   *   })
   * }
   * ```
   */
  async getExecutions(
    name: string,
    namespace: string = "default",
    limit: number = 50
  ): Promise<{ data: MigrationExecution[] | null; error: Error | null }> {
    try {
      const params = new URLSearchParams({
        namespace,
        limit: limit.toString(),
      });
      const data = await this.fetch.get<MigrationExecution[]>(
        `/api/v1/admin/migrations/${name}/executions?${params.toString()}`
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }
}
