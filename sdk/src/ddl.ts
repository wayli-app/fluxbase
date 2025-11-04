import type { FluxbaseFetch } from './fetch'
import type {
  CreateSchemaRequest,
  CreateSchemaResponse,
  CreateTableRequest,
  CreateTableResponse,
  DeleteTableResponse,
  ListSchemasResponse,
  ListTablesResponse,
  CreateColumnRequest,
} from './types'

/**
 * DDL (Data Definition Language) Manager
 *
 * Provides methods for managing database schemas and tables programmatically.
 * This includes creating schemas, creating tables with custom columns, listing
 * schemas and tables, and deleting tables.
 *
 * @example
 * ```typescript
 * const ddl = client.admin.ddl
 *
 * // Create a new schema
 * await ddl.createSchema('analytics')
 *
 * // Create a table with columns
 * await ddl.createTable('analytics', 'events', [
 *   { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
 *   { name: 'user_id', type: 'UUID', nullable: false },
 *   { name: 'event_name', type: 'TEXT', nullable: false },
 *   { name: 'event_data', type: 'JSONB' },
 *   { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
 * ])
 *
 * // List all schemas
 * const { schemas } = await ddl.listSchemas()
 *
 * // List all tables in a schema
 * const { tables } = await ddl.listTables('analytics')
 *
 * // Delete a table
 * await ddl.deleteTable('analytics', 'events')
 * ```
 */
export class DDLManager {
  constructor(private fetch: FluxbaseFetch) {}

  /**
   * Create a new database schema
   *
   * Creates a new schema in the database. Schemas are used to organize tables
   * into logical groups and provide namespace isolation.
   *
   * @param name - Schema name (must be valid PostgreSQL identifier)
   * @returns Promise resolving to CreateSchemaResponse
   *
   * @example
   * ```typescript
   * // Create a schema for analytics data
   * const result = await client.admin.ddl.createSchema('analytics')
   * console.log(result.message) // "Schema created successfully"
   * console.log(result.schema)  // "analytics"
   * ```
   */
  async createSchema(name: string): Promise<CreateSchemaResponse> {
    const request: CreateSchemaRequest = { name }
    return await this.fetch.post<CreateSchemaResponse>('/api/v1/admin/ddl/schemas', request)
  }

  /**
   * List all database schemas
   *
   * Retrieves a list of all schemas in the database. This includes both
   * system schemas (like 'public', 'pg_catalog') and user-created schemas.
   *
   * @returns Promise resolving to ListSchemasResponse
   *
   * @example
   * ```typescript
   * const { schemas } = await client.admin.ddl.listSchemas()
   *
   * schemas.forEach(schema => {
   *   console.log(`Schema: ${schema.name}, Owner: ${schema.owner}`)
   * })
   * ```
   */
  async listSchemas(): Promise<ListSchemasResponse> {
    return await this.fetch.get<ListSchemasResponse>('/api/v1/admin/ddl/schemas')
  }

  /**
   * Create a new table in a schema
   *
   * Creates a new table with the specified columns. Supports various column
   * options including primary keys, nullability, data types, and default values.
   *
   * @param schema - Schema name where the table will be created
   * @param name - Table name (must be valid PostgreSQL identifier)
   * @param columns - Array of column definitions
   * @returns Promise resolving to CreateTableResponse
   *
   * @example
   * ```typescript
   * // Create a users table
   * await client.admin.ddl.createTable('public', 'users', [
   *   {
   *     name: 'id',
   *     type: 'UUID',
   *     primaryKey: true,
   *     defaultValue: 'gen_random_uuid()'
   *   },
   *   {
   *     name: 'email',
   *     type: 'TEXT',
   *     nullable: false
   *   },
   *   {
   *     name: 'name',
   *     type: 'TEXT'
   *   },
   *   {
   *     name: 'created_at',
   *     type: 'TIMESTAMPTZ',
   *     nullable: false,
   *     defaultValue: 'NOW()'
   *   }
   * ])
   * ```
   *
   * @example
   * ```typescript
   * // Create a products table with JSONB metadata
   * await client.admin.ddl.createTable('public', 'products', [
   *   { name: 'id', type: 'SERIAL', primaryKey: true },
   *   { name: 'name', type: 'TEXT', nullable: false },
   *   { name: 'price', type: 'DECIMAL(10,2)', nullable: false },
   *   { name: 'metadata', type: 'JSONB' },
   *   { name: 'in_stock', type: 'BOOLEAN', defaultValue: 'true' }
   * ])
   * ```
   */
  async createTable(schema: string, name: string, columns: CreateColumnRequest[]): Promise<CreateTableResponse> {
    const request: CreateTableRequest = { schema, name, columns }
    return await this.fetch.post<CreateTableResponse>('/api/v1/admin/ddl/tables', request)
  }

  /**
   * List all tables in the database or a specific schema
   *
   * Retrieves a list of all tables. If a schema is specified, only tables
   * from that schema are returned. Otherwise, all tables from all schemas
   * are returned.
   *
   * @param schema - Optional schema name to filter tables
   * @returns Promise resolving to ListTablesResponse
   *
   * @example
   * ```typescript
   * // List all tables in the public schema
   * const { tables } = await client.admin.ddl.listTables('public')
   *
   * tables.forEach(table => {
   *   console.log(`Table: ${table.schema}.${table.name}`)
   *   table.columns?.forEach(col => {
   *     console.log(`  - ${col.name}: ${col.type}`)
   *   })
   * })
   * ```
   *
   * @example
   * ```typescript
   * // List all tables across all schemas
   * const { tables } = await client.admin.ddl.listTables()
   *
   * const tablesBySchema = tables.reduce((acc, table) => {
   *   if (!acc[table.schema]) acc[table.schema] = []
   *   acc[table.schema].push(table.name)
   *   return acc
   * }, {} as Record<string, string[]>)
   *
   * console.log(tablesBySchema)
   * ```
   */
  async listTables(schema?: string): Promise<ListTablesResponse> {
    const params = schema ? `?schema=${encodeURIComponent(schema)}` : ''
    return await this.fetch.get<ListTablesResponse>(`/api/v1/admin/ddl/tables${params}`)
  }

  /**
   * Delete a table from a schema
   *
   * Permanently deletes a table and all its data. This operation cannot be undone.
   *
   * @param schema - Schema name containing the table
   * @param name - Table name to delete
   * @returns Promise resolving to DeleteTableResponse
   *
   * @example
   * ```typescript
   * // Delete a table
   * const result = await client.admin.ddl.deleteTable('public', 'old_data')
   * console.log(result.message) // "Table deleted successfully"
   * ```
   *
   * @example
   * ```typescript
   * // Safe deletion with confirmation
   * const confirm = await askUser('Are you sure you want to delete this table?')
   * if (confirm) {
   *   await client.admin.ddl.deleteTable('analytics', 'events')
   *   console.log('Table deleted')
   * }
   * ```
   */
  async deleteTable(schema: string, name: string): Promise<DeleteTableResponse> {
    return await this.fetch.delete<DeleteTableResponse>(
      `/api/v1/admin/ddl/tables/${encodeURIComponent(schema)}/${encodeURIComponent(name)}`
    )
  }
}
