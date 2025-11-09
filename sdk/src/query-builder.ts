/**
 * PostgreSQL query builder for Fluxbase SDK
 * Inspired by Supabase's PostgREST client
 */

import type { FluxbaseFetch } from './fetch'
import type { FilterOperator, OrderBy, PostgrestResponse } from './types'

export class QueryBuilder<T = unknown> implements PromiseLike<PostgrestResponse<T>> {
  private fetch: FluxbaseFetch
  private table: string
  private selectQuery: string = '*'
  private filters: Array<{ column: string; operator: FilterOperator; value: unknown }> = []
  private orderBys: OrderBy[] = []
  private limitValue?: number
  private offsetValue?: number
  private singleRow: boolean = false
  private groupByColumns?: string[]

  constructor(fetch: FluxbaseFetch, table: string) {
    this.fetch = fetch
    this.table = table
  }

  /**
   * Select columns to return
   * @example select('*')
   * @example select('id, name, email')
   * @example select('id, name, posts(title, content)')
   */
  select(columns: string = '*'): this {
    this.selectQuery = columns
    return this
  }

  /**
   * Insert a single row or multiple rows
   */
  async insert(data: Partial<T> | Array<Partial<T>>): Promise<PostgrestResponse<T>> {
    const body = Array.isArray(data) ? data : data
    const response = await this.fetch.post<T>(`/api/v1/tables/${this.table}`, body)

    return {
      data: response,
      error: null,
      count: Array.isArray(data) ? data.length : 1,
      status: 201,
      statusText: 'Created',
    }
  }

  /**
   * Upsert (insert or update) rows
   */
  async upsert(data: Partial<T> | Array<Partial<T>>): Promise<PostgrestResponse<T>> {
    const body = Array.isArray(data) ? data : data
    const response = await this.fetch.post<T>(`/api/v1/tables/${this.table}`, body, {
      headers: {
        Prefer: 'resolution=merge-duplicates',
      },
    })

    return {
      data: response,
      error: null,
      count: Array.isArray(data) ? data.length : 1,
      status: 201,
      statusText: 'Created',
    }
  }

  /**
   * Update rows matching the filters
   */
  async update(data: Partial<T>): Promise<PostgrestResponse<T>> {
    const queryString = this.buildQueryString()
    const path = `/api/v1/tables/${this.table}${queryString}`
    const response = await this.fetch.patch<T>(path, data)

    return {
      data: response,
      error: null,
      count: null,
      status: 200,
      statusText: 'OK',
    }
  }

  /**
   * Delete rows matching the filters
   */
  async delete(): Promise<PostgrestResponse<null>> {
    const queryString = this.buildQueryString()
    const path = `/api/v1/tables/${this.table}${queryString}`
    await this.fetch.delete(path)

    return {
      data: null,
      error: null,
      count: null,
      status: 204,
      statusText: 'No Content',
    }
  }

  /**
   * Equal to
   */
  eq(column: string, value: unknown): this {
    this.filters.push({ column, operator: 'eq', value })
    return this
  }

  /**
   * Not equal to
   */
  neq(column: string, value: unknown): this {
    this.filters.push({ column, operator: 'neq', value })
    return this
  }

  /**
   * Greater than
   */
  gt(column: string, value: unknown): this {
    this.filters.push({ column, operator: 'gt', value })
    return this
  }

  /**
   * Greater than or equal to
   */
  gte(column: string, value: unknown): this {
    this.filters.push({ column, operator: 'gte', value })
    return this
  }

  /**
   * Less than
   */
  lt(column: string, value: unknown): this {
    this.filters.push({ column, operator: 'lt', value })
    return this
  }

  /**
   * Less than or equal to
   */
  lte(column: string, value: unknown): this {
    this.filters.push({ column, operator: 'lte', value })
    return this
  }

  /**
   * Pattern matching (case-sensitive)
   */
  like(column: string, pattern: string): this {
    this.filters.push({ column, operator: 'like', value: pattern })
    return this
  }

  /**
   * Pattern matching (case-insensitive)
   */
  ilike(column: string, pattern: string): this {
    this.filters.push({ column, operator: 'ilike', value: pattern })
    return this
  }

  /**
   * Check if value is null or not null
   */
  is(column: string, value: null | boolean): this {
    this.filters.push({ column, operator: 'is', value })
    return this
  }

  /**
   * Check if value is in array
   */
  in(column: string, values: unknown[]): this {
    this.filters.push({ column, operator: 'in', value: values })
    return this
  }

  /**
   * Contains (for arrays and JSONB)
   */
  contains(column: string, value: unknown): this {
    this.filters.push({ column, operator: 'cs', value })
    return this
  }

  /**
   * Full-text search
   */
  textSearch(column: string, query: string): this {
    this.filters.push({ column, operator: 'fts', value: query })
    return this
  }

  /**
   * Order results
   */
  order(column: string, options?: { ascending?: boolean; nullsFirst?: boolean }): this {
    this.orderBys.push({
      column,
      direction: options?.ascending === false ? 'desc' : 'asc',
      nulls: options?.nullsFirst ? 'first' : 'last',
    })
    return this
  }

  /**
   * Limit number of rows returned
   */
  limit(count: number): this {
    this.limitValue = count
    return this
  }

  /**
   * Skip rows
   */
  offset(count: number): this {
    this.offsetValue = count
    return this
  }

  /**
   * Return a single row (adds limit(1))
   */
  single(): this {
    this.singleRow = true
    this.limitValue = 1
    return this
  }

  /**
   * Range selection (pagination)
   */
  range(from: number, to: number): this {
    this.offsetValue = from
    this.limitValue = to - from + 1
    return this
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
    this.groupByColumns = Array.isArray(columns) ? columns : [columns]
    return this
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
  count(column: string = '*'): this {
    this.selectQuery = `count(${column})`
    return this
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
    this.selectQuery = `sum(${column})`
    return this
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
    this.selectQuery = `avg(${column})`
    return this
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
    this.selectQuery = `min(${column})`
    return this
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
    this.selectQuery = `max(${column})`
    return this
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
   * ]).execute()
   * ```
   *
   * @category Batch Operations
   */
  async insertMany(rows: Array<Partial<T>>): Promise<PostgrestResponse<T>> {
    return this.insert(rows)
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
   *   .execute()
   *
   * // Mark all pending orders as processing
   * const { data } = await client.from('orders')
   *   .eq('status', 'pending')
   *   .updateMany({ status: 'processing' })
   *   .execute()
   * ```
   *
   * @category Batch Operations
   */
  async updateMany(data: Partial<T>): Promise<PostgrestResponse<T>> {
    return this.update(data)
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
   *   .execute()
   *
   * // Delete old logs
   * await client.from('logs')
   *   .lt('created_at', '2024-01-01')
   *   .deleteMany()
   *   .execute()
   * ```
   *
   * @category Batch Operations
   */
  async deleteMany(): Promise<PostgrestResponse<null>> {
    return this.delete()
  }

  /**
   * Execute the query and return results
   */
  async execute(): Promise<PostgrestResponse<T>> {
    const queryString = this.buildQueryString()
    const path = `/api/v1/tables/${this.table}${queryString}`

    try {
      const data = await this.fetch.get<T | T[]>(path)

      // Handle single row response
      if (this.singleRow) {
        if (Array.isArray(data) && data.length === 0) {
          return {
            data: null,
            error: { message: 'No rows found', code: 'PGRST116' },
            count: 0,
            status: 404,
            statusText: 'Not Found',
          }
        }
        const singleData = Array.isArray(data) ? data[0] : data
        return {
          data: singleData as T,
          error: null,
          count: 1,
          status: 200,
          statusText: 'OK',
        }
      }

      return {
        data: data as T,
        error: null,
        count: Array.isArray(data) ? data.length : null,
        status: 200,
        statusText: 'OK',
      }
    } catch (err) {
      const error = err as Error
      return {
        data: null,
        error: {
          message: error.message,
          code: 'PGRST000',
        },
        count: null,
        status: 500,
        statusText: 'Internal Server Error',
      }
    }
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
    onfulfilled?: ((value: PostgrestResponse<T>) => TResult1 | PromiseLike<TResult1>) | null,
    onrejected?: ((reason: any) => TResult2 | PromiseLike<TResult2>) | null,
  ): PromiseLike<TResult1 | TResult2> {
    return this.execute().then(onfulfilled, onrejected)
  }

  /**
   * Build the query string from filters, ordering, etc.
   */
  private buildQueryString(): string {
    const params = new URLSearchParams()

    // Select
    if (this.selectQuery && this.selectQuery !== '*') {
      params.append('select', this.selectQuery)
    }

    // Filters
    for (const filter of this.filters) {
      params.append(filter.column, `${filter.operator}.${this.formatValue(filter.value)}`)
    }

    // Group By
    if (this.groupByColumns && this.groupByColumns.length > 0) {
      params.append('group_by', this.groupByColumns.join(','))
    }

    // Order
    if (this.orderBys.length > 0) {
      const orderStr = this.orderBys
        .map((o) => `${o.column}.${o.direction}${o.nulls ? `.nulls${o.nulls}` : ''}`)
        .join(',')
      params.append('order', orderStr)
    }

    // Limit
    if (this.limitValue !== undefined) {
      params.append('limit', String(this.limitValue))
    }

    // Offset
    if (this.offsetValue !== undefined) {
      params.append('offset', String(this.offsetValue))
    }

    const queryString = params.toString()
    return queryString ? `?${queryString}` : ''
  }

  /**
   * Format a value for the query string
   */
  private formatValue(value: unknown): string {
    if (value === null) {
      return 'null'
    }
    if (typeof value === 'boolean') {
      return value ? 'true' : 'false'
    }
    if (Array.isArray(value)) {
      return `(${value.map((v) => this.formatValue(v)).join(',')})`
    }
    if (typeof value === 'object') {
      return JSON.stringify(value)
    }
    return String(value)
  }
}
