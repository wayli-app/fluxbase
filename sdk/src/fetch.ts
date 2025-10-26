/**
 * HTTP client for making requests to the Fluxbase API
 */

import type { FluxbaseError, HttpMethod } from './types'

export interface FetchOptions {
  method: HttpMethod
  headers?: Record<string, string>
  body?: unknown
  timeout?: number
}

export class FluxbaseFetch {
  private baseUrl: string
  private defaultHeaders: Record<string, string>
  private timeout: number
  private debug: boolean

  constructor(
    baseUrl: string,
    options: {
      headers?: Record<string, string>
      timeout?: number
      debug?: boolean
    } = {}
  ) {
    this.baseUrl = baseUrl.replace(/\/$/, '') // Remove trailing slash
    this.defaultHeaders = {
      'Content-Type': 'application/json',
      ...options.headers,
    }
    this.timeout = options.timeout ?? 30000
    this.debug = options.debug ?? false
  }

  /**
   * Update the authorization header
   */
  setAuthToken(token: string | null) {
    if (token) {
      this.defaultHeaders['Authorization'] = `Bearer ${token}`
    } else {
      delete this.defaultHeaders['Authorization']
    }
  }

  /**
   * Make an HTTP request
   */
  async request<T = unknown>(path: string, options: FetchOptions): Promise<T> {
    const url = `${this.baseUrl}${path}`
    const headers = { ...this.defaultHeaders, ...options.headers }

    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), options.timeout ?? this.timeout)

    if (this.debug) {
      console.log(`[Fluxbase SDK] ${options.method} ${url}`, options.body)
    }

    try {
      const response = await fetch(url, {
        method: options.method,
        headers,
        body: options.body ? JSON.stringify(options.body) : undefined,
        signal: controller.signal,
      })

      clearTimeout(timeoutId)

      // Parse response
      const contentType = response.headers.get('content-type')
      let data: unknown

      if (contentType?.includes('application/json')) {
        data = await response.json()
      } else {
        data = await response.text()
      }

      if (this.debug) {
        console.log(`[Fluxbase SDK] Response:`, response.status, data)
      }

      // Handle errors
      if (!response.ok) {
        const error = new Error(
          typeof data === 'object' && data && 'error' in data
            ? String(data.error)
            : response.statusText
        ) as FluxbaseError

        error.status = response.status
        error.details = data

        throw error
      }

      return data as T
    } catch (err) {
      clearTimeout(timeoutId)

      if (err instanceof Error) {
        if (err.name === 'AbortError') {
          const timeoutError = new Error('Request timeout') as FluxbaseError
          timeoutError.status = 408
          throw timeoutError
        }

        throw err
      }

      throw new Error('Unknown error occurred')
    }
  }

  /**
   * GET request
   */
  async get<T = unknown>(path: string, options: Omit<FetchOptions, 'method'> = {}): Promise<T> {
    return this.request<T>(path, { ...options, method: 'GET' })
  }

  /**
   * POST request
   */
  async post<T = unknown>(path: string, body?: unknown, options: Omit<FetchOptions, 'method' | 'body'> = {}): Promise<T> {
    return this.request<T>(path, { ...options, method: 'POST', body })
  }

  /**
   * PATCH request
   */
  async patch<T = unknown>(path: string, body?: unknown, options: Omit<FetchOptions, 'method' | 'body'> = {}): Promise<T> {
    return this.request<T>(path, { ...options, method: 'PATCH', body })
  }

  /**
   * DELETE request
   */
  async delete<T = unknown>(path: string, options: Omit<FetchOptions, 'method'> = {}): Promise<T> {
    return this.request<T>(path, { ...options, method: 'DELETE' })
  }

  /**
   * HEAD request
   */
  async head(path: string, options: Omit<FetchOptions, 'method'> = {}): Promise<Headers> {
    const url = `${this.baseUrl}${path}`
    const headers = { ...this.defaultHeaders, ...options.headers }

    const response = await fetch(url, {
      method: 'HEAD',
      headers,
    })

    return response.headers
  }
}
