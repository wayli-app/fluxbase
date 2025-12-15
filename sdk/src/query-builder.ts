/**
 * PostgreSQL query builder for Fluxbase SDK
 * Inspired by Supabase's PostgREST client
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  CountType,
  FilterOperator,
  OrderBy,
  PostgrestResponse,
  SelectOptions,
  UpsertOptions,
} from "./types";

// Threshold for switching to POST-based queries (4KB)
const URL_LENGTH_THRESHOLD = 4096;

export class QueryBuilder<T = unknown>
  implements PromiseLike<PostgrestResponse<T>>
{
  private fetch: FluxbaseFetch;
  private table: string;
  private schema?: string;
  private selectQuery: string = "*";
  private filters: Array<{
    column: string;
    operator: FilterOperator;
    value: unknown;
  }> = [];
  private orFilters: string[] = [];
  private andFilters: string[] = [];
  private betweenFilters: Array<{
    column: string;
    min: unknown;
    max: unknown;
    negated: boolean;
  }> = [];
  private orderBys: OrderBy[] = [];
  private limitValue?: number;
  private offsetValue?: number;
  private singleRow: boolean = false;
  private maybeSingleRow: boolean = false;
  private groupByColumns?: string[];
  private operationType: "select" | "insert" | "update" | "delete" = "select";
  private countType?: CountType;
  private headOnly: boolean = false;
  private isCountAggregation: boolean = false;
  private insertData?: Partial<T> | Array<Partial<T>>;
  private updateData?: Partial<T>;

  constructor(fetch: FluxbaseFetch, table: string, schema?: string) {
    this.fetch = fetch;
    this.table = table;
    this.schema = schema;
  }

  /**
   * Build the API path for this table, including schema if specified
   */
  private buildTablePath(): string {
    return this.schema
      ? `/api/v1/tables/${this.schema}/${this.table}`
      : `/api/v1/tables/${this.table}`;
  }

  /**
   * Select columns to return
   * @example select('*')
   * @example select('id, name, email')
   * @example select('id, name, posts(title, content)')
   * @example select('*', { count: 'exact' }) // Get exact count
   * @example select('*', { count: 'exact', head: true }) // Get count only (no data)
   */
  select(columns: string = "*", options?: SelectOptions): this {
    this.selectQuery = columns;
    if (options?.count) {
      this.countType = options.count;
    }
    if (options?.head) {
      this.headOnly = true;
    }
    return this;
  }

  /**
   * Insert a single row or multiple rows
   */
  insert(data: Partial<T> | Array<Partial<T>>): this {
    this.operationType = "insert";
    this.insertData = data;
    return this;
  }

  /**
   * Upsert (insert or update) rows (Supabase-compatible)
   * @param data - Row(s) to upsert
   * @param options - Upsert options (onConflict, ignoreDuplicates, defaultToNull)
   */
  async upsert(
    data: Partial<T> | Array<Partial<T>>,
    options?: UpsertOptions,
  ): Promise<PostgrestResponse<T>> {
    const body = Array.isArray(data) ? data : data;

    // Build Prefer header based on options
    const preferValues: string[] = [];

    if (options?.ignoreDuplicates) {
      preferValues.push("resolution=ignore-duplicates");
    } else {
      preferValues.push("resolution=merge-duplicates");
    }

    if (options?.defaultToNull) {
      preferValues.push("missing=default");
    }

    const headers: Record<string, string> = {
      Prefer: preferValues.join(","),
    };

    // Add onConflict as query parameter if specified
    let path = this.buildTablePath();
    if (options?.onConflict) {
      path += `?on_conflict=${encodeURIComponent(options.onConflict)}`;
    }

    const response = await this.fetch.post<T>(path, body, { headers });

    return {
      data: response,
      error: null,
      count: Array.isArray(data) ? data.length : 1,
      status: 201,
      statusText: "Created",
    };
  }

  /**
   * Update rows matching the filters
   */
  update(data: Partial<T>): this {
    this.operationType = "update";
    this.updateData = data;
    return this;
  }

  /**
   * Delete rows matching the filters
   */
  delete(): this {
    this.operationType = "delete";
    return this;
  }

  /**
   * Equal to
   */
  eq(column: string, value: unknown): this {
    this.filters.push({ column, operator: "eq", value });
    return this;
  }

  /**
   * Not equal to
   */
  neq(column: string, value: unknown): this {
    this.filters.push({ column, operator: "neq", value });
    return this;
  }

  /**
   * Greater than
   */
  gt(column: string, value: unknown): this {
    this.filters.push({ column, operator: "gt", value });
    return this;
  }

  /**
   * Greater than or equal to
   */
  gte(column: string, value: unknown): this {
    this.filters.push({ column, operator: "gte", value });
    return this;
  }

  /**
   * Less than
   */
  lt(column: string, value: unknown): this {
    this.filters.push({ column, operator: "lt", value });
    return this;
  }

  /**
   * Less than or equal to
   */
  lte(column: string, value: unknown): this {
    this.filters.push({ column, operator: "lte", value });
    return this;
  }

  /**
   * Pattern matching (case-sensitive)
   */
  like(column: string, pattern: string): this {
    this.filters.push({ column, operator: "like", value: pattern });
    return this;
  }

  /**
   * Pattern matching (case-insensitive)
   */
  ilike(column: string, pattern: string): this {
    this.filters.push({ column, operator: "ilike", value: pattern });
    return this;
  }

  /**
   * Check if value is null or not null
   */
  is(column: string, value: null | boolean): this {
    this.filters.push({ column, operator: "is", value });
    return this;
  }

  /**
   * Check if value is in array
   */
  in(column: string, values: unknown[]): this {
    this.filters.push({ column, operator: "in", value: values });
    return this;
  }

  /**
   * Contains (for arrays and JSONB)
   */
  contains(column: string, value: unknown): this {
    this.filters.push({ column, operator: "cs", value });
    return this;
  }

  /**
   * Full-text search
   */
  textSearch(column: string, query: string): this {
    this.filters.push({ column, operator: "fts", value: query });
    return this;
  }

  /**
   * Negate a filter condition (Supabase-compatible)
   * @example not('status', 'eq', 'deleted')
   * @example not('completed_at', 'is', null)
   */
  not(column: string, operator: FilterOperator, value: unknown): this {
    this.filters.push({
      column,
      operator: "not" as FilterOperator,
      value: `${operator}.${this.formatValue(value)}`,
    });
    return this;
  }

  /**
   * Apply OR logic to filters (Supabase-compatible)
   * @example or('status.eq.active,status.eq.pending')
   * @example or('id.eq.2,name.eq.Han')
   */
  or(filters: string): this {
    this.orFilters.push(filters);
    return this;
  }

  /**
   * Apply AND logic to filters (Supabase-compatible)
   * Groups multiple conditions that must all be true
   * @example and('status.eq.active,verified.eq.true')
   * @example and('age.gte.18,age.lte.65')
   */
  and(filters: string): this {
    this.andFilters.push(filters);
    return this;
  }

  /**
   * Match multiple columns with exact values (Supabase-compatible)
   * Shorthand for multiple .eq() calls
   * @example match({ id: 1, status: 'active', role: 'admin' })
   */
  match(conditions: Record<string, unknown>): this {
    for (const [column, value] of Object.entries(conditions)) {
      this.eq(column, value);
    }
    return this;
  }

  /**
   * Generic filter method using PostgREST syntax (Supabase-compatible)
   * @example filter('name', 'in', '("Han","Yoda")')
   * @example filter('age', 'gte', '18')
   * @example filter('recorded_at', 'between', ['2024-01-01', '2024-01-10'])
   * @example filter('recorded_at', 'not.between', ['2024-01-01', '2024-01-10'])
   */
  filter(column: string, operator: FilterOperator, value: unknown): this {
    // Handle special compound operators
    if (operator === "between" || operator === "not.between") {
      const [min, max] = this.validateBetweenValue(value, operator);
      this.betweenFilters.push({
        column,
        min,
        max,
        negated: operator === "not.between",
      });
      return this;
    }

    this.filters.push({ column, operator, value });
    return this;
  }

  /**
   * Check if column is contained by value (Supabase-compatible)
   * For arrays and JSONB
   * @example containedBy('tags', '["news","update"]')
   */
  containedBy(column: string, value: unknown): this {
    this.filters.push({ column, operator: "cd", value });
    return this;
  }

  /**
   * Check if arrays have common elements (Supabase-compatible)
   * @example overlaps('tags', '["news","sports"]')
   */
  overlaps(column: string, value: unknown): this {
    this.filters.push({ column, operator: "ov", value });
    return this;
  }

  /**
   * Filter column value within an inclusive range (BETWEEN)
   * Generates: AND (column >= min AND column <= max)
   *
   * @param column - Column name to filter
   * @param min - Minimum value (inclusive)
   * @param max - Maximum value (inclusive)
   * @example between('recorded_at', '2024-01-01', '2024-01-10')
   * @example between('price', 10, 100)
   */
  between(column: string, min: unknown, max: unknown): this {
    return this.filter(column, "between", [min, max]);
  }

  /**
   * Filter column value outside an inclusive range (NOT BETWEEN)
   * Generates: OR (column < min OR column > max)
   * Multiple notBetween calls on the same column AND together
   *
   * @param column - Column name to filter
   * @param min - Minimum value of excluded range
   * @param max - Maximum value of excluded range
   * @example notBetween('recorded_at', '2024-01-01', '2024-01-10')
   * @example notBetween('price', 0, 10) // Excludes items priced 0-10
   */
  notBetween(column: string, min: unknown, max: unknown): this {
    return this.filter(column, "not.between", [min, max]);
  }

  // PostGIS Spatial Query Methods

  /**
   * Check if geometries intersect (PostGIS ST_Intersects)
   * @param column - Column containing geometry/geography data
   * @param geojson - GeoJSON object to test intersection with
   * @example intersects('location', { type: 'Point', coordinates: [-122.4, 37.8] })
   */
  intersects(column: string, geojson: unknown): this {
    this.filters.push({
      column,
      operator: "st_intersects" as FilterOperator,
      value: geojson,
    });
    return this;
  }

  /**
   * Check if geometry A contains geometry B (PostGIS ST_Contains)
   * @param column - Column containing geometry/geography data
   * @param geojson - GeoJSON object to test containment
   * @example contains('region', { type: 'Point', coordinates: [-122.4, 37.8] })
   */
  stContains(column: string, geojson: unknown): this {
    this.filters.push({
      column,
      operator: "st_contains" as FilterOperator,
      value: geojson,
    });
    return this;
  }

  /**
   * Check if geometry A is within geometry B (PostGIS ST_Within)
   * @param column - Column containing geometry/geography data
   * @param geojson - GeoJSON object to test containment within
   * @example within('point', { type: 'Polygon', coordinates: [[...]] })
   */
  within(column: string, geojson: unknown): this {
    this.filters.push({
      column,
      operator: "st_within" as FilterOperator,
      value: geojson,
    });
    return this;
  }

  /**
   * Check if geometries touch (PostGIS ST_Touches)
   * @param column - Column containing geometry/geography data
   * @param geojson - GeoJSON object to test touching
   * @example touches('boundary', { type: 'LineString', coordinates: [[...]] })
   */
  touches(column: string, geojson: unknown): this {
    this.filters.push({
      column,
      operator: "st_touches" as FilterOperator,
      value: geojson,
    });
    return this;
  }

  /**
   * Check if geometries cross (PostGIS ST_Crosses)
   * @param column - Column containing geometry/geography data
   * @param geojson - GeoJSON object to test crossing
   * @example crosses('road', { type: 'LineString', coordinates: [[...]] })
   */
  crosses(column: string, geojson: unknown): this {
    this.filters.push({
      column,
      operator: "st_crosses" as FilterOperator,
      value: geojson,
    });
    return this;
  }

  /**
   * Check if geometries spatially overlap (PostGIS ST_Overlaps)
   * @param column - Column containing geometry/geography data
   * @param geojson - GeoJSON object to test overlap
   * @example stOverlaps('area', { type: 'Polygon', coordinates: [[...]] })
   */
  stOverlaps(column: string, geojson: unknown): this {
    this.filters.push({
      column,
      operator: "st_overlaps" as FilterOperator,
      value: geojson,
    });
    return this;
  }

  /**
   * Order results
   */
  order(
    column: string,
    options?: { ascending?: boolean; nullsFirst?: boolean },
  ): this {
    this.orderBys.push({
      column,
      direction: options?.ascending === false ? "desc" : "asc",
      nulls: options?.nullsFirst ? "first" : "last",
    });
    return this;
  }

  /**
   * Limit number of rows returned
   */
  limit(count: number): this {
    this.limitValue = count;
    return this;
  }

  /**
   * Skip rows
   */
  offset(count: number): this {
    this.offsetValue = count;
    return this;
  }

  /**
   * Return a single row (adds limit(1))
   * Errors if no rows found
   */
  single(): this {
    this.singleRow = true;
    this.limitValue = 1;
    return this;
  }

  /**
   * Return a single row or null (adds limit(1))
   * Does not error if no rows found (Supabase-compatible)
   * @example
   * ```typescript
   * // Returns null instead of erroring when no row exists
   * const { data, error } = await client
   *   .from('users')
   *   .select('*')
   *   .eq('id', 999)
   *   .maybeSingle()
   * // data will be null if no row found
   * ```
   */
  maybeSingle(): this {
    this.maybeSingleRow = true;
    this.limitValue = 1;
    return this;
  }

  /**
   * Range selection (pagination)
   */
  range(from: number, to: number): this {
    this.offsetValue = from;
    this.limitValue = to - from + 1;
    return this;
  }

  /**
   * Group results by one or more columns (for use with aggregations)
   *
   * @param columns - Column name(s) to group by
   * @returns Query builder for chaining
   *
   * @example
   * ```typescript
   * // Group by single column
   * const { data } = await client.from('orders')
   *   .count('*')
   *   .groupBy('status')
   *   .execute()
   *
   * // Group by multiple columns
   * const { data } = await client.from('sales')
   *   .sum('amount')
   *   .groupBy(['region', 'product_category'])
   *   .execute()
   * ```
   *
   * @category Aggregation
   */
  groupBy(columns: string | string[]): this {
    this.groupByColumns = Array.isArray(columns) ? columns : [columns];
    return this;
  }

  /**
   * Count rows or a specific column
   *
   * @param column - Column to count (default: '*' for row count)
   * @returns Query builder for chaining
   *
   * @example
   * ```typescript
   * // Count all rows
   * const { data } = await client.from('users').count().execute()
   * // Returns: { count: 150 }
   *
   * // Count non-null values in a column
   * const { data } = await client.from('orders').count('completed_at').execute()
   *
   * // Count with grouping
   * const { data } = await client.from('products')
   *   .count('*')
   *   .groupBy('category')
   *   .execute()
   * // Returns: [{ category: 'electronics', count: 45 }, { category: 'books', count: 23 }]
   * ```
   *
   * @category Aggregation
   */
  count(column: string = "*"): this {
    this.selectQuery = `count(${column})`;
    this.isCountAggregation = true;
    return this;
  }

  /**
   * Calculate the sum of a numeric column
   *
   * @param column - Column to sum
   * @returns Query builder for chaining
   *
   * @example
   * ```typescript
   * // Sum all prices
   * const { data } = await client.from('products').sum('price').execute()
   * // Returns: { sum_price: 15420.50 }
   *
   * // Sum by category
   * const { data } = await client.from('orders')
   *   .sum('total')
   *   .groupBy('status')
   *   .execute()
   * // Returns: [{ status: 'completed', sum_total: 12500 }, { status: 'pending', sum_total: 3200 }]
   * ```
   *
   * @category Aggregation
   */
  sum(column: string): this {
    this.selectQuery = `sum(${column})`;
    return this;
  }

  /**
   * Calculate the average of a numeric column
   *
   * @param column - Column to average
   * @returns Query builder for chaining
   *
   * @example
   * ```typescript
   * // Average price
   * const { data } = await client.from('products').avg('price').execute()
   * // Returns: { avg_price: 129.99 }
   *
   * // Average by category
   * const { data } = await client.from('products')
   *   .avg('price')
   *   .groupBy('category')
   *   .execute()
   * ```
   *
   * @category Aggregation
   */
  avg(column: string): this {
    this.selectQuery = `avg(${column})`;
    return this;
  }

  /**
   * Find the minimum value in a column
   *
   * @param column - Column to find minimum value
   * @returns Query builder for chaining
   *
   * @example
   * ```typescript
   * // Find lowest price
   * const { data } = await client.from('products').min('price').execute()
   * // Returns: { min_price: 9.99 }
   *
   * // Find earliest date
   * const { data } = await client.from('orders').min('created_at').execute()
   * ```
   *
   * @category Aggregation
   */
  min(column: string): this {
    this.selectQuery = `min(${column})`;
    return this;
  }

  /**
   * Find the maximum value in a column
   *
   * @param column - Column to find maximum value
   * @returns Query builder for chaining
   *
   * @example
   * ```typescript
   * // Find highest price
   * const { data } = await client.from('products').max('price').execute()
   * // Returns: { max_price: 1999.99 }
   *
   * // Find most recent order
   * const { data } = await client.from('orders').max('created_at').execute()
   * ```
   *
   * @category Aggregation
   */
  max(column: string): this {
    this.selectQuery = `max(${column})`;
    return this;
  }

  /**
   * Insert multiple rows in a single request (batch insert)
   *
   * This is a convenience method that explicitly shows intent for batch operations.
   * Internally calls `insert()` with an array.
   *
   * @param rows - Array of row objects to insert
   * @returns Promise with the inserted rows
   *
   * @example
   * ```typescript
   * // Insert multiple users at once
   * const { data } = await client.from('users').insertMany([
   *   { name: 'Alice', email: 'alice@example.com' },
   *   { name: 'Bob', email: 'bob@example.com' },
   *   { name: 'Charlie', email: 'charlie@example.com' }
   * ])
   * ```
   *
   * @category Batch Operations
   */
  async insertMany(rows: Array<Partial<T>>): Promise<PostgrestResponse<T>> {
    return this.insert(rows).execute();
  }

  /**
   * Update multiple rows matching the filters (batch update)
   *
   * Updates all rows that match the current query filters.
   * This is a convenience method that explicitly shows intent for batch operations.
   *
   * @param data - Data to update matching rows with
   * @returns Promise with the updated rows
   *
   * @example
   * ```typescript
   * // Apply discount to all electronics
   * const { data } = await client.from('products')
   *   .eq('category', 'electronics')
   *   .updateMany({ discount: 10, updated_at: new Date() })
   *
   * // Mark all pending orders as processing
   * const { data } = await client.from('orders')
   *   .eq('status', 'pending')
   *   .updateMany({ status: 'processing' })
   * ```
   *
   * @category Batch Operations
   */
  async updateMany(data: Partial<T>): Promise<PostgrestResponse<T>> {
    return this.update(data).execute();
  }

  /**
   * Delete multiple rows matching the filters (batch delete)
   *
   * Deletes all rows that match the current query filters.
   * This is a convenience method that explicitly shows intent for batch operations.
   *
   * @returns Promise confirming deletion
   *
   * @example
   * ```typescript
   * // Delete all inactive users
   * await client.from('users')
   *   .eq('active', false)
   *   .deleteMany()
   *
   * // Delete old logs
   * await client.from('logs')
   *   .lt('created_at', '2024-01-01')
   *   .deleteMany()
   * ```
   *
   * @category Batch Operations
   */
  async deleteMany(): Promise<PostgrestResponse<null>> {
    return this.delete().execute() as Promise<PostgrestResponse<null>>;
  }

  /**
   * Execute the query and return results
   */
  async execute(): Promise<PostgrestResponse<T>> {
    try {
      // Handle INSERT operation
      if (this.operationType === "insert") {
        if (!this.insertData) {
          throw new Error("Insert data is required for insert operation");
        }
        const body = Array.isArray(this.insertData)
          ? this.insertData
          : this.insertData;
        const response = await this.fetch.post<T>(this.buildTablePath(), body);

        return {
          data: response,
          error: null,
          count: Array.isArray(this.insertData) ? this.insertData.length : 1,
          status: 201,
          statusText: "Created",
        };
      }

      // Handle UPDATE operation
      if (this.operationType === "update") {
        if (!this.updateData) {
          throw new Error("Update data is required for update operation");
        }
        const queryString = this.buildQueryString();
        const path = `${this.buildTablePath()}${queryString}`;
        const response = await this.fetch.patch<T>(path, this.updateData);

        return {
          data: response,
          error: null,
          count: null,
          status: 200,
          statusText: "OK",
        };
      }

      // Handle DELETE operation
      if (this.operationType === "delete") {
        const queryString = this.buildQueryString();
        const path = `${this.buildTablePath()}${queryString}`;
        await this.fetch.delete(path);

        return {
          data: null,
          error: null,
          count: null,
          status: 204,
          statusText: "No Content",
        } as PostgrestResponse<T>;
      }

      // Handle SELECT operation (default)
      // Check if we should use POST-based query (for complex filters exceeding URL limits)
      if (this.shouldUsePostQuery()) {
        return this.executePostQuery();
      }

      const queryString = this.buildQueryString();
      const path = `${this.buildTablePath()}${queryString}`;

      // When count is requested, use getWithHeaders to access Content-Range header
      if (this.countType) {
        const response = await this.fetch.getWithHeaders<T | T[]>(path);
        const serverCount = this.parseContentRangeCount(response.headers);
        const data = response.data;

        // Handle head-only request (only return count, no data)
        if (this.headOnly) {
          return {
            data: null,
            error: null,
            count: serverCount,
            status: response.status,
            statusText: "OK",
          };
        }

        // Handle single row response with count
        if (this.singleRow) {
          if (Array.isArray(data) && data.length === 0) {
            return {
              data: null,
              error: { message: "No rows found", code: "PGRST116" },
              count: serverCount ?? 0,
              status: 404,
              statusText: "Not Found",
            };
          }
          const singleData = Array.isArray(data) ? data[0] : data;
          return {
            data: singleData as T,
            error: null,
            count: serverCount ?? 1,
            status: 200,
            statusText: "OK",
          };
        }

        // Handle maybeSingle row response with count
        if (this.maybeSingleRow) {
          if (Array.isArray(data) && data.length === 0) {
            return {
              data: null,
              error: null,
              count: serverCount ?? 0,
              status: 200,
              statusText: "OK",
            };
          }
          const singleData = Array.isArray(data) ? data[0] : data;
          return {
            data: singleData as T,
            error: null,
            count: serverCount ?? 1,
            status: 200,
            statusText: "OK",
          };
        }

        // Normal response with server count
        return {
          data: data as T,
          error: null,
          count: serverCount ?? (Array.isArray(data) ? data.length : null),
          status: 200,
          statusText: "OK",
        };
      }

      // Standard path without count - use regular get
      const data = await this.fetch.get<T | T[]>(path);

      // Handle count aggregation response - extract count from data[0].count
      // Skip if using groupBy (return full array instead)
      if (this.isCountAggregation && !this.groupByColumns) {
        if (Array.isArray(data) && data.length === 1) {
          const countData = data[0] as { count?: number };
          if (countData && typeof countData.count === "number") {
            return {
              data: data as T,
              error: null,
              count: countData.count,
              status: 200,
              statusText: "OK",
            };
          }
        }
        // Handle empty result for count aggregation
        if (Array.isArray(data) && data.length === 0) {
          return {
            data: data as T,
            error: null,
            count: 0,
            status: 200,
            statusText: "OK",
          };
        }
      }

      // Handle single row response
      if (this.singleRow) {
        if (Array.isArray(data) && data.length === 0) {
          return {
            data: null,
            error: { message: "No rows found", code: "PGRST116" },
            count: 0,
            status: 404,
            statusText: "Not Found",
          };
        }
        const singleData = Array.isArray(data) ? data[0] : data;
        return {
          data: singleData as T,
          error: null,
          count: 1,
          status: 200,
          statusText: "OK",
        };
      }

      // Handle maybeSingle row response (returns null instead of error when no rows found)
      if (this.maybeSingleRow) {
        if (Array.isArray(data) && data.length === 0) {
          return {
            data: null,
            error: null,
            count: 0,
            status: 200,
            statusText: "OK",
          };
        }
        const singleData = Array.isArray(data) ? data[0] : data;
        return {
          data: singleData as T,
          error: null,
          count: 1,
          status: 200,
          statusText: "OK",
        };
      }

      return {
        data: data as T,
        error: null,
        count: Array.isArray(data) ? data.length : null,
        status: 200,
        statusText: "OK",
      };
    } catch (err) {
      const error = err as Error;
      return {
        data: null,
        error: {
          message: error.message,
          code: "PGRST000",
        },
        count: null,
        status: 500,
        statusText: "Internal Server Error",
      };
    }
  }

  /**
   * Execute the query and throw an error if one occurs (Supabase-compatible)
   * Returns the data directly instead of { data, error } wrapper
   *
   * @throws {Error} If the query fails or returns an error
   * @example
   * ```typescript
   * // Throws error instead of returning { data, error }
   * try {
   *   const user = await client
   *     .from('users')
   *     .select('*')
   *     .eq('id', 1)
   *     .single()
   *     .throwOnError()
   * } catch (error) {
   *   console.error('Query failed:', error)
   * }
   * ```
   */
  async throwOnError(): Promise<T> {
    const response = await this.execute();

    if (response.error) {
      const error = new Error(response.error.message);
      // Preserve error code if available
      if (response.error.code) {
        (error as any).code = response.error.code;
      }
      throw error;
    }

    return response.data as T;
  }

  /**
   * Make QueryBuilder awaitable (implements PromiseLike)
   * This allows using `await client.from('table').select()` without calling `.execute()`
   *
   * @example
   * ```typescript
   * // Without .execute() (new way)
   * const { data } = await client.from('users').select('*')
   *
   * // With .execute() (old way, still supported)
   * const { data } = await client.from('users').select('*').execute()
   * ```
   */
  then<TResult1 = PostgrestResponse<T>, TResult2 = never>(
    onfulfilled?:
      | ((value: PostgrestResponse<T>) => TResult1 | PromiseLike<TResult1>)
      | null,
    onrejected?: ((reason: any) => TResult2 | PromiseLike<TResult2>) | null,
  ): PromiseLike<TResult1 | TResult2> {
    return this.execute().then(onfulfilled, onrejected);
  }

  /**
   * Build the query string from filters, ordering, etc.
   */
  private buildQueryString(): string {
    const params = new URLSearchParams();

    // Select
    if (this.selectQuery && this.selectQuery !== "*") {
      params.append("select", this.selectQuery);
    }

    // Filters
    for (const filter of this.filters) {
      params.append(
        filter.column,
        `${filter.operator}.${this.formatValue(filter.value)}`,
      );
    }

    // Collect all OR expressions that need to be ANDed together
    // This includes explicit orFilters and negated betweenFilters
    const orExpressions: string[] = [];

    // OR Filters (explicit .or() calls)
    for (const orFilter of this.orFilters) {
      orExpressions.push(`or(${orFilter})`);
    }

    // Between Filters - collect negated ones as OR expressions
    for (const bf of this.betweenFilters) {
      const minFormatted = this.formatValue(bf.min);
      const maxFormatted = this.formatValue(bf.max);

      if (bf.negated) {
        // not.between: or(column.lt.min,column.gt.max)
        // This generates: WHERE column < min OR column > max
        orExpressions.push(
          `or(${bf.column}.lt.${minFormatted},${bf.column}.gt.${maxFormatted})`,
        );
      } else {
        // between: column=gte.min&column=lte.max (two separate filters ANDed)
        // This generates: WHERE column >= min AND column <= max
        params.append(bf.column, `gte.${minFormatted}`);
        params.append(bf.column, `lte.${maxFormatted}`);
      }
    }

    // Combine OR expressions efficiently
    if (orExpressions.length === 1) {
      // Single OR - use simple or= format (strip outer "or(" wrapper)
      const expr = orExpressions[0]!;
      const inner = expr.replace(/^or\(/, "(");
      params.append("or", inner);
    } else if (orExpressions.length > 1) {
      // Multiple ORs - combine into single and= parameter with nested or() expressions
      // This significantly reduces URL length for complex filter sets
      params.append("and", `(${orExpressions.join(",")})`);
    }

    // AND Filters (explicit .and() calls)
    for (const andFilter of this.andFilters) {
      params.append("and", `(${andFilter})`);
    }

    // Group By
    if (this.groupByColumns && this.groupByColumns.length > 0) {
      params.append("group_by", this.groupByColumns.join(","));
    }

    // Order
    if (this.orderBys.length > 0) {
      const orderStr = this.orderBys
        .map(
          (o) =>
            `${o.column}.${o.direction}${o.nulls ? `.nulls${o.nulls}` : ""}`,
        )
        .join(",");
      params.append("order", orderStr);
    }

    // Limit
    if (this.limitValue !== undefined) {
      params.append("limit", String(this.limitValue));
    }

    // Offset
    if (this.offsetValue !== undefined) {
      params.append("offset", String(this.offsetValue));
    }

    // Count - request server to return total count in Content-Range header
    if (this.countType) {
      params.append("count", this.countType);
    }

    const queryString = params.toString();
    return queryString ? `?${queryString}` : "";
  }

  /**
   * Format a value for the query string
   */
  private formatValue(value: unknown): string {
    if (value === null) {
      return "null";
    }
    if (typeof value === "boolean") {
      return value ? "true" : "false";
    }
    if (Array.isArray(value)) {
      return `(${value.map((v) => this.formatValue(v)).join(",")})`;
    }
    if (typeof value === "object") {
      return JSON.stringify(value);
    }
    return String(value);
  }

  /**
   * Validate between filter value - must be array of exactly 2 elements
   * @throws Error if value is invalid
   */
  private validateBetweenValue(
    value: unknown,
    operator: string,
  ): [unknown, unknown] {
    if (!Array.isArray(value)) {
      throw new Error(
        `Invalid value for '${operator}' operator: expected array of [min, max], got ${typeof value}`,
      );
    }
    if (value.length !== 2) {
      throw new Error(
        `Invalid value for '${operator}' operator: expected array with exactly 2 elements [min, max], got ${value.length} elements`,
      );
    }
    const [min, max] = value;
    if (min === null || min === undefined) {
      throw new Error(
        `Invalid value for '${operator}' operator: min value cannot be null or undefined`,
      );
    }
    if (max === null || max === undefined) {
      throw new Error(
        `Invalid value for '${operator}' operator: max value cannot be null or undefined`,
      );
    }
    return [min, max];
  }

  /**
   * Parse the Content-Range header to extract the total count
   * Header format: "0-999/50000" or "* /50000" (when no rows returned)
   */
  private parseContentRangeCount(headers: Headers): number | null {
    const contentRange = headers.get("Content-Range");
    if (!contentRange) {
      return null;
    }
    // Match the total count after the slash: "0-999/50000" -> "50000"
    const match = contentRange.match(/\/(\d+)$/);
    if (match && match[1]) {
      return parseInt(match[1], 10);
    }
    return null;
  }

  /**
   * Check if the query should use POST-based query endpoint
   * Returns true if the query string would exceed the URL length threshold
   */
  private shouldUsePostQuery(): boolean {
    const queryString = this.buildQueryString();
    return queryString.length > URL_LENGTH_THRESHOLD;
  }

  /**
   * Execute a SELECT query using the POST /query endpoint
   * Used when query parameters would exceed URL length limits
   */
  private async executePostQuery(): Promise<PostgrestResponse<T>> {
    const body = this.buildQueryBody();
    const path = `${this.buildTablePath()}/query`;

    // When count is requested, use postWithHeaders to access Content-Range header
    if (this.countType) {
      const response = await this.fetch.postWithHeaders<T | T[]>(path, body);
      const serverCount = this.parseContentRangeCount(response.headers);
      const data = response.data;

      // Handle head-only request (only return count, no data)
      if (this.headOnly) {
        return {
          data: null,
          error: null,
          count: serverCount,
          status: response.status,
          statusText: "OK",
        };
      }

      // Handle single row response with count
      if (this.singleRow) {
        if (Array.isArray(data) && data.length === 0) {
          return {
            data: null,
            error: { message: "No rows found", code: "PGRST116" },
            count: serverCount ?? 0,
            status: 404,
            statusText: "Not Found",
          };
        }
        const singleData = Array.isArray(data) ? data[0] : data;
        return {
          data: singleData as T,
          error: null,
          count: serverCount ?? 1,
          status: 200,
          statusText: "OK",
        };
      }

      // Handle maybeSingle row response with count
      if (this.maybeSingleRow) {
        if (Array.isArray(data) && data.length === 0) {
          return {
            data: null,
            error: null,
            count: serverCount ?? 0,
            status: 200,
            statusText: "OK",
          };
        }
        const singleData = Array.isArray(data) ? data[0] : data;
        return {
          data: singleData as T,
          error: null,
          count: serverCount ?? 1,
          status: 200,
          statusText: "OK",
        };
      }

      // Normal response with server count
      return {
        data: data as T,
        error: null,
        count: serverCount ?? (Array.isArray(data) ? data.length : null),
        status: 200,
        statusText: "OK",
      };
    }

    // Standard path without count - use regular post
    const data = await this.fetch.post<T | T[]>(path, body);

    // Handle single row response
    if (this.singleRow) {
      if (Array.isArray(data) && data.length === 0) {
        return {
          data: null,
          error: { message: "No rows found", code: "PGRST116" },
          count: 0,
          status: 404,
          statusText: "Not Found",
        };
      }
      const singleData = Array.isArray(data) ? data[0] : data;
      return {
        data: singleData as T,
        error: null,
        count: 1,
        status: 200,
        statusText: "OK",
      };
    }

    // Handle maybeSingle row response
    if (this.maybeSingleRow) {
      if (Array.isArray(data) && data.length === 0) {
        return {
          data: null,
          error: null,
          count: 0,
          status: 200,
          statusText: "OK",
        };
      }
      const singleData = Array.isArray(data) ? data[0] : data;
      return {
        data: singleData as T,
        error: null,
        count: 1,
        status: 200,
        statusText: "OK",
      };
    }

    // Normal response
    return {
      data: data as T,
      error: null,
      count: Array.isArray(data) ? data.length : null,
      status: 200,
      statusText: "OK",
    };
  }

  /**
   * Build the request body for POST-based queries
   * Used when query parameters would exceed URL length limits
   */
  private buildQueryBody(): PostQueryRequestBody {
    return {
      select: this.selectQuery !== "*" ? this.selectQuery : undefined,
      filters: this.filters.map((f) => ({
        column: f.column,
        operator: f.operator,
        value: f.value,
      })),
      orFilters: this.orFilters,
      andFilters: this.andFilters,
      betweenFilters: this.betweenFilters.map((bf) => ({
        column: bf.column,
        min: bf.min,
        max: bf.max,
        negated: bf.negated,
      })),
      order: this.orderBys.map((o) => ({
        column: o.column,
        direction: o.direction,
        nulls: o.nulls,
      })),
      limit: this.limitValue,
      offset: this.offsetValue,
      count: this.countType,
      groupBy: this.groupByColumns,
    };
  }
}

/**
 * Request body structure for POST-based queries
 * Mirrors the backend PostQueryRequest structure
 */
interface PostQueryRequestBody {
  select?: string;
  filters?: Array<{
    column: string;
    operator: string;
    value: unknown;
  }>;
  orFilters?: string[];
  andFilters?: string[];
  betweenFilters?: Array<{
    column: string;
    min: unknown;
    max: unknown;
    negated: boolean;
  }>;
  order?: Array<{
    column: string;
    direction: string;
    nulls?: string;
  }>;
  limit?: number;
  offset?: number;
  count?: string;
  groupBy?: string[];
}
