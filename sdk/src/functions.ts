/**
 * Edge Functions module for Fluxbase SDK
 * Compatible with Supabase Functions API
 *
 * @example
 * ```typescript
 * // Invoke a function
 * const { data, error } = await client.functions.invoke('hello-world', {
 *   body: { name: 'Alice' }
 * })
 *
 * // With custom headers
 * const { data, error } = await client.functions.invoke('api-call', {
 *   body: { query: 'data' },
 *   headers: { 'X-Custom-Header': 'value' },
 *   method: 'POST'
 * })
 * ```
 */

import type { FluxbaseFetch } from './fetch'
import type {
  FunctionInvokeOptions,
  EdgeFunction,
  CreateFunctionRequest,
  UpdateFunctionRequest,
  EdgeFunctionExecution,
} from './types'

/**
 * Edge Functions client for invoking and managing serverless functions
 * API-compatible with Supabase Functions
 *
 * @category Functions
 */
export class FluxbaseFunctions {
  private fetch: FluxbaseFetch

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch
  }

  /**
   * Invoke an edge function
   *
   * This method is fully compatible with Supabase's functions.invoke() API.
   *
   * @param functionName - The name of the function to invoke
   * @param options - Invocation options including body, headers, and HTTP method
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * // Simple invocation
   * const { data, error } = await client.functions.invoke('hello', {
   *   body: { name: 'World' }
   * })
   *
   * // With GET method
   * const { data, error } = await client.functions.invoke('get-data', {
   *   method: 'GET'
   * })
   *
   * // With custom headers
   * const { data, error } = await client.functions.invoke('api-proxy', {
   *   body: { query: 'search' },
   *   headers: { 'Authorization': 'Bearer token' },
   *   method: 'POST'
   * })
   * ```
   */
  async invoke<T = any>(
    functionName: string,
    options?: FunctionInvokeOptions
  ): Promise<{ data: T | null; error: Error | null }> {
    try {
      const method = options?.method || 'POST'
      const headers = options?.headers || {}
      const body = options?.body

      // Use the Fluxbase backend endpoint
      const endpoint = `/api/v1/functions/${functionName}/invoke`

      let response: T

      // Route to appropriate HTTP method
      switch (method) {
        case 'GET':
          response = await this.fetch.get<T>(endpoint, { headers })
          break
        case 'DELETE':
          response = await this.fetch.delete<T>(endpoint, { headers })
          break
        case 'PUT':
          response = await this.fetch.put<T>(endpoint, body, { headers })
          break
        case 'PATCH':
          response = await this.fetch.patch<T>(endpoint, body, { headers })
          break
        case 'POST':
        default:
          response = await this.fetch.post<T>(endpoint, body, { headers })
          break
      }

      return { data: response, error: null }
    } catch (error) {
      return { data: null, error: error as Error }
    }
  }

  /**
   * Create a new edge function
   *
   * @param request - Function configuration and code
   * @returns Promise resolving to { data, error } tuple with created function metadata
   *
   * @example
   * ```typescript
   * const { data, error } = await client.functions.create({
   *   name: 'my-function',
   *   code: 'export default async function handler(req) { return { hello: "world" } }',
   *   enabled: true
   * })
   * ```
   */
  async create(
    request: CreateFunctionRequest
  ): Promise<{ data: EdgeFunction | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<EdgeFunction>('/api/v1/functions', request)
      return { data, error: null }
    } catch (error) {
      return { data: null, error: error as Error }
    }
  }

  /**
   * List all edge functions
   *
   * @returns Promise resolving to { data, error } tuple with array of functions
   *
   * @example
   * ```typescript
   * const { data, error } = await client.functions.list()
   * if (data) {
   *   console.log('Functions:', data.map(f => f.name))
   * }
   * ```
   */
  async list(): Promise<{ data: EdgeFunction[] | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<EdgeFunction[]>('/api/v1/functions')
      return { data, error: null }
    } catch (error) {
      return { data: null, error: error as Error }
    }
  }

  /**
   * Get details of a specific edge function
   *
   * @param name - Function name
   * @returns Promise resolving to { data, error } tuple with function metadata
   *
   * @example
   * ```typescript
   * const { data, error } = await client.functions.get('my-function')
   * if (data) {
   *   console.log('Function version:', data.version)
   * }
   * ```
   */
  async get(name: string): Promise<{ data: EdgeFunction | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<EdgeFunction>(`/api/v1/functions/${name}`)
      return { data, error: null }
    } catch (error) {
      return { data: null, error: error as Error }
    }
  }

  /**
   * Update an existing edge function
   *
   * @param name - Function name
   * @param updates - Fields to update
   * @returns Promise resolving to { data, error } tuple with updated function metadata
   *
   * @example
   * ```typescript
   * const { data, error } = await client.functions.update('my-function', {
   *   enabled: false,
   *   description: 'Updated description'
   * })
   * ```
   */
  async update(
    name: string,
    updates: UpdateFunctionRequest
  ): Promise<{ data: EdgeFunction | null; error: Error | null }> {
    try {
      const data = await this.fetch.put<EdgeFunction>(`/api/v1/functions/${name}`, updates)
      return { data, error: null }
    } catch (error) {
      return { data: null, error: error as Error }
    }
  }

  /**
   * Delete an edge function
   *
   * @param name - Function name
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { data, error } = await client.functions.delete('my-function')
   * ```
   */
  async delete(name: string): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.delete(`/api/v1/functions/${name}`)
      return { data: null, error: null }
    } catch (error) {
      return { data: null, error: error as Error }
    }
  }

  /**
   * Get execution history for an edge function
   *
   * @param name - Function name
   * @param limit - Maximum number of executions to return (optional)
   * @returns Promise resolving to { data, error } tuple with execution records
   *
   * @example
   * ```typescript
   * const { data, error } = await client.functions.getExecutions('my-function', 10)
   * if (data) {
   *   data.forEach(exec => {
   *     console.log(`${exec.executed_at}: ${exec.status} (${exec.duration_ms}ms)`)
   *   })
   * }
   * ```
   */
  async getExecutions(
    name: string,
    limit?: number
  ): Promise<{ data: EdgeFunctionExecution[] | null; error: Error | null }> {
    try {
      const params = limit ? `?limit=${limit}` : ''
      const data = await this.fetch.get<EdgeFunctionExecution[]>(
        `/api/v1/functions/${name}/executions${params}`
      )
      return { data, error: null }
    } catch (error) {
      return { data: null, error: error as Error }
    }
  }
}
