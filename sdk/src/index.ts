/**
 * Fluxbase TypeScript SDK
 *
 * @example
 * ```typescript
 * import { createClient } from '@fluxbase/sdk'
 *
 * const client = createClient({
 *   url: 'http://localhost:8080',
 *   auth: {
 *     token: 'your-token',
 *     autoRefresh: true,
 *     persist: true
 *   }
 * })
 *
 * // Authentication
 * const { user } = await client.auth.signIn({
 *   email: 'user@example.com',
 *   password: 'password'
 * })
 *
 * // Database
 * const { data, error } = await client
 *   .from('products')
 *   .select('*')
 *   .eq('category', 'electronics')
 *   .execute()
 *
 * // Realtime
 * client.channel('table:public.products')
 *   .on('INSERT', (payload) => console.log('New product:', payload))
 *   .subscribe()
 *
 * // Storage
 * await client.storage
 *   .from('avatars')
 *   .upload('user-123.png', file)
 * ```
 */

// Main client
export { FluxbaseClient, createClient } from './client'

// Auth module
export { FluxbaseAuth } from './auth'

// Database query builder
export { QueryBuilder } from './query-builder'

// Realtime module
export { FluxbaseRealtime, RealtimeChannel } from './realtime'

// Storage module
export { FluxbaseStorage, StorageBucket } from './storage'

// HTTP client (advanced users)
export { FluxbaseFetch } from './fetch'

// Types
export type {
  // Client options
  FluxbaseClientOptions,

  // Auth types
  AuthSession,
  User,
  SignInCredentials,
  SignUpCredentials,
  AuthResponse,
  TwoFactorSetupResponse,
  TwoFactorEnableResponse,
  TwoFactorStatusResponse,
  TwoFactorVerifyRequest,
  SignInWith2FAResponse,

  // Database types
  PostgrestResponse,
  PostgrestError,
  FilterOperator,
  QueryFilter,
  OrderBy,
  OrderDirection,

  // Realtime types
  RealtimeMessage,
  RealtimeChangePayload,
  RealtimeCallback,

  // Storage types
  StorageObject,
  UploadOptions,
  ListOptions,
  SignedUrlOptions,

  // HTTP types
  FluxbaseError,
  HttpMethod,
  RequestOptions,
} from './types'
