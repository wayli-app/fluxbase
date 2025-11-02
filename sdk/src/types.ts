/**
 * Core types for the Fluxbase SDK
 */

export interface FluxbaseClientOptions {
  /**
   * Base URL of your Fluxbase instance
   * @example 'https://api.myapp.com'
   * @example 'http://localhost:8080'
   */
  url: string

  /**
   * Authentication options
   */
  auth?: {
    /**
     * Access token for authentication
     */
    token?: string

    /**
     * Auto-refresh token when it expires
     * @default true
     */
    autoRefresh?: boolean

    /**
     * Persist auth state in localStorage
     * @default true
     */
    persist?: boolean
  }

  /**
   * Global headers to include in all requests
   */
  headers?: Record<string, string>

  /**
   * Request timeout in milliseconds
   * @default 30000
   */
  timeout?: number

  /**
   * Enable debug logging
   * @default false
   */
  debug?: boolean
}

export interface AuthSession {
  user: User
  access_token: string
  refresh_token: string
  expires_in: number
  expires_at?: number
}

export interface User {
  id: string
  email: string
  email_verified: boolean
  role: string
  metadata?: Record<string, unknown> | null
  created_at: string
  updated_at: string
}

export interface SignInCredentials {
  email: string
  password: string
}

export interface SignUpCredentials {
  email: string
  password: string
  metadata?: Record<string, unknown>
}

export interface AuthResponse {
  user: User
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface TwoFactorSetupResponse {
  secret: string
  qr_code_url: string
  message: string
}

export interface TwoFactorEnableResponse {
  success: boolean
  backup_codes: string[]
  message: string
}

export interface TwoFactorStatusResponse {
  totp_enabled: boolean
}

export interface TwoFactorVerifyRequest {
  user_id: string
  code: string
}

export interface SignInWith2FAResponse {
  requires_2fa: boolean
  user_id: string
  message: string
}

export interface FluxbaseError extends Error {
  status?: number
  code?: string
  details?: unknown
}

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD'

export interface RequestOptions {
  method: HttpMethod
  headers?: Record<string, string>
  body?: unknown
  timeout?: number
}

export interface PostgrestError {
  message: string
  details?: string
  hint?: string
  code?: string
}

export interface PostgrestResponse<T> {
  data: T | null
  error: PostgrestError | null
  count: number | null
  status: number
  statusText: string
}

export type FilterOperator =
  | 'eq'
  | 'neq'
  | 'gt'
  | 'gte'
  | 'lt'
  | 'lte'
  | 'like'
  | 'ilike'
  | 'is'
  | 'in'
  | 'cs'
  | 'cd'
  | 'ov'
  | 'sl'
  | 'sr'
  | 'nxr'
  | 'nxl'
  | 'fts'
  | 'plfts'
  | 'wfts'

export interface QueryFilter {
  column: string
  operator: FilterOperator
  value: unknown
}

export type OrderDirection = 'asc' | 'desc'

export interface OrderBy {
  column: string
  direction: OrderDirection
  nulls?: 'first' | 'last'
}

export interface RealtimeMessage {
  type: 'subscribe' | 'unsubscribe' | 'heartbeat' | 'broadcast' | 'ack' | 'error'
  channel?: string
  payload?: unknown
  error?: string
}

export interface RealtimeChangePayload {
  type: 'INSERT' | 'UPDATE' | 'DELETE'
  schema: string
  table: string
  new_record?: Record<string, unknown>
  old_record?: Record<string, unknown>
  timestamp: string
}

export type RealtimeCallback = (payload: RealtimeChangePayload) => void

export interface StorageObject {
  key: string
  bucket: string
  size: number
  content_type: string
  last_modified: string
  etag?: string
  metadata?: Record<string, string>
}

export interface UploadOptions {
  contentType?: string
  metadata?: Record<string, string>
  cacheControl?: string
  upsert?: boolean
}

export interface ListOptions {
  prefix?: string
  limit?: number
  offset?: number
}

export interface SignedUrlOptions {
  expiresIn?: number // seconds
}
