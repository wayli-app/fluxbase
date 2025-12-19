/**
 * Schema-scoped query builder for accessing tables in non-public schemas.
 *
 * @example
 * ```typescript
 * // Query the logging.entries table
 * const { data } = await client
 *   .schema('logging')
 *   .from('entries')
 *   .select('*')
 *   .execute();
 * ```
 */

import type { FluxbaseFetch } from "./fetch";
import { QueryBuilder } from "./query-builder";

export class SchemaQueryBuilder {
  constructor(
    private fetch: FluxbaseFetch,
    private schemaName: string,
  ) {}

  /**
   * Create a query builder for a table in this schema
   *
   * @param table - The table name (without schema prefix)
   * @returns A query builder instance for constructing and executing queries
   */
  from<T = any>(table: string): QueryBuilder<T> {
    return new QueryBuilder<T>(this.fetch, table, this.schemaName);
  }
}
