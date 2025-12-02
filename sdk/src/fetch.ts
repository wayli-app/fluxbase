/**
 * HTTP client for making requests to the Fluxbase API
 */

import type { FluxbaseError, HttpMethod } from './types'

export interface FetchOptions {
  method: HttpMethod
  headers?: Record<string, string>
  body?: unknown
  timeout?: number
  /** Skip automatic token refresh on 401 (used for auth endpoints) */
  skipAutoRefresh?: boolean
}

/**
 * Response with headers included (for count queries)
 */
export interface FetchResponseWithHeaders<T> {
  data: T
  headers: Headers
  status: number
}

/** Callback type for automatic token refresh on 401 errors */
export type RefreshTokenCallback = () => Promise<boolean>

export class FluxbaseFetch {
  private baseUrl: string
  private defaultHeaders: Record<string, string>
  private timeout: number
  private debug: boolean
  private refreshTokenCallback: RefreshTokenCallback | null = null
  private isRefreshing = false
  private refreshPromise: Promise<boolean> | null = null
  private anonKey: string | null = null

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
   * Register a callback to refresh the token when a 401 error occurs
   * The callback should return true if refresh was successful, false otherwise
   */
  setRefreshTokenCallback(callback: RefreshTokenCallback | null) {
    this.refreshTokenCallback = callback
  }

  /**
   * Set the anon key for fallback authentication
   * When setAuthToken(null) is called, the Authorization header will be
   * restored to use this anon key instead of being deleted
   */
  setAnonKey(key: string) {
    this.anonKey = key
  }

  /**
   * Update the authorization header
   * When token is null, restores to anon key if available
   */
  setAuthToken(token: string | null) {
    if (token) {
      this.defaultHeaders['Authorization'] = `Bearer ${token}`
    } else if (this.anonKey) {
      // Restore anon key auth instead of deleting header
      this.defaultHeaders['Authorization'] = `Bearer ${this.anonKey}`
    } else {
      delete this.defaultHeaders['Authorization']
    }
  }

  /**
   * Make an HTTP request
   */
  async request<T = unknown>(path: string, options: FetchOptions): Promise<T> {
    return this.requestInternal<T>(path, options, false)
  }

  /**
   * Internal request implementation with retry capability
   */
  private async requestInternal<T = unknown>(
    path: string,
    options: FetchOptions,
    isRetry: boolean
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`
    const headers = { ...this.defaultHeaders, ...options.headers }

    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), options.timeout ?? this.timeout)

    if (this.debug) {
      console.log(`[Fluxbase SDK] ${options.method} ${url}`, options.body)
    }

    try {
      // Determine if body is FormData (needs special handling for multipart uploads)
      // Use constructor.name check for cross-runtime compatibility (Deno, Node, Browser)
      // instanceof can fail across different realms/contexts in bundled IIFE code
      const isFormData = options.body &&
        (options.body.constructor?.name === 'FormData' || options.body instanceof FormData);

      // For FormData, omit Content-Type to let runtime set multipart/form-data with boundary
      const requestHeaders = isFormData
        ? Object.fromEntries(
            Object.entries(headers).filter(([key]) => key.toLowerCase() !== 'content-type')
          )
        : headers;

      const response = await fetch(url, {
        method: options.method,
        headers: requestHeaders,
        body: isFormData ? (options.body as FormData) : (options.body ? JSON.stringify(options.body) : undefined),
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

      // Handle 401 errors with automatic token refresh
      if (
        response.status === 401 &&
        !isRetry &&
        !options.skipAutoRefresh &&
        this.refreshTokenCallback
      ) {
        const refreshSuccess = await this.handleTokenRefresh()
        if (refreshSuccess) {
          // Retry the request with the new token
          return this.requestInternal<T>(path, options, true)
        }
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
   * Handle token refresh with deduplication
   * Multiple concurrent requests that fail with 401 will share the same refresh operation
   */
  private async handleTokenRefresh(): Promise<boolean> {
    // If already refreshing, wait for the existing refresh to complete
    if (this.isRefreshing && this.refreshPromise) {
      return this.refreshPromise
    }

    this.isRefreshing = true
    this.refreshPromise = this.executeRefresh()

    try {
      return await this.refreshPromise
    } finally {
      this.isRefreshing = false
      this.refreshPromise = null
    }
  }

  /**
   * Execute the actual token refresh
   */
  private async executeRefresh(): Promise<boolean> {
    if (!this.refreshTokenCallback) {
      return false
    }

    try {
      return await this.refreshTokenCallback()
    } catch (error) {
      if (this.debug) {
        console.error('[Fluxbase SDK] Token refresh failed:', error)
      }
      return false
    }
  }

  /**
   * GET request
   */
  async get<T = unknown>(path: string, options: Omit<FetchOptions, 'method'> = {}): Promise<T> {
    return this.request<T>(path, { ...options, method: 'GET' })
  }

  /**
   * GET request that returns response with headers (for count queries)
   */
  async getWithHeaders<T = unknown>(path: string, options: Omit<FetchOptions, 'method'> = {}): Promise<FetchResponseWithHeaders<T>> {
    return this.requestWithHeaders<T>(path, { ...options, method: 'GET' })
  }

  /**
   * Make an HTTP request and return response with headers
   */
  async requestWithHeaders<T = unknown>(path: string, options: FetchOptions): Promise<FetchResponseWithHeaders<T>> {
    return this.requestWithHeadersInternal<T>(path, options, false)
  }

  /**
   * Internal request implementation that returns response with headers
   */
  private async requestWithHeadersInternal<T = unknown>(
    path: string,
    options: FetchOptions,
    isRetry: boolean
  ): Promise<FetchResponseWithHeaders<T>> {
    const url = `${this.baseUrl}${path}`
    const headers = { ...this.defaultHeaders, ...options.headers }

    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), options.timeout ?? this.timeout)

    if (this.debug) {
      console.log(`[Fluxbase SDK] ${options.method} ${url}`, options.body)
    }

    try {
      // Determine if body is FormData (needs special handling for multipart uploads)
      // Use constructor.name check for cross-runtime compatibility (Deno, Node, Browser)
      // instanceof can fail across different realms/contexts in bundled IIFE code
      const isFormData = options.body &&
        (options.body.constructor?.name === 'FormData' || options.body instanceof FormData);

      // For FormData, omit Content-Type to let runtime set multipart/form-data with boundary
      const requestHeaders = isFormData
        ? Object.fromEntries(
            Object.entries(headers).filter(([key]) => key.toLowerCase() !== 'content-type')
          )
        : headers;

      const response = await fetch(url, {
        method: options.method,
        headers: requestHeaders,
        body: isFormData ? (options.body as FormData) : (options.body ? JSON.stringify(options.body) : undefined),
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

      // Handle 401 errors with automatic token refresh
      if (
        response.status === 401 &&
        !isRetry &&
        !options.skipAutoRefresh &&
        this.refreshTokenCallback
      ) {
        const refreshSuccess = await this.handleTokenRefresh()
        if (refreshSuccess) {
          // Retry the request with the new token
          return this.requestWithHeadersInternal<T>(path, options, true)
        }
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

      return {
        data: data as T,
        headers: response.headers,
        status: response.status,
      }
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
   * POST request
   */
  async post<T = unknown>(path: string, body?: unknown, options: Omit<FetchOptions, 'method' | 'body'> = {}): Promise<T> {
    return this.request<T>(path, { ...options, method: 'POST', body })
  }

  /**
   * PUT request
   */
  async put<T = unknown>(path: string, body?: unknown, options: Omit<FetchOptions, 'method' | 'body'> = {}): Promise<T> {
    return this.request<T>(path, { ...options, method: 'PUT', body })
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
