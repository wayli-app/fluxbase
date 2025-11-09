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

// Password Reset Types
export interface PasswordResetRequest {
  email: string
}

export interface PasswordResetResponse {
  message: string
}

export interface VerifyResetTokenRequest {
  token: string
}

export interface VerifyResetTokenResponse {
  valid: boolean
  message: string
}

export interface ResetPasswordRequest {
  token: string
  new_password: string
}

export interface ResetPasswordResponse {
  message: string
}

// Magic Link Types
export interface MagicLinkOptions {
  redirect_to?: string
}

export interface MagicLinkRequest {
  email: string
  redirect_to?: string
}

export interface MagicLinkResponse {
  message: string
}

export interface VerifyMagicLinkRequest {
  token: string
}

// Anonymous Auth Types
export interface AnonymousSignInResponse extends AuthResponse {
  is_anonymous: boolean
}

// OAuth Types
export interface OAuthProvider {
  id: string
  name: string
  enabled: boolean
  authorize_url?: string
}

export interface OAuthProvidersResponse {
  providers: OAuthProvider[]
}

export interface OAuthOptions {
  redirect_to?: string
  scopes?: string[]
}

export interface OAuthUrlResponse {
  url: string
  provider: string
}

// Admin Authentication Types
export interface AdminSetupStatusResponse {
  needs_setup: boolean
  has_admin: boolean
}

export interface AdminSetupRequest {
  email: string
  password: string
  name: string
}

export interface AdminUser {
  id: string
  email: string
  name: string
  role: string
  email_verified: boolean
  created_at: string
  updated_at: string
  last_login_at?: string
}

export interface AdminAuthResponse {
  user: AdminUser
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface AdminLoginRequest {
  email: string
  password: string
}

export interface AdminRefreshRequest {
  refresh_token: string
}

export interface AdminRefreshResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: AdminUser
}

export interface AdminMeResponse {
  user: {
    id: string
    email: string
    role: string
  }
}

// User Management Types
export interface EnrichedUser {
  id: string
  email: string
  role?: string
  created_at: string
  updated_at?: string
  email_verified?: boolean
  last_login_at?: string
  session_count?: number
  is_anonymous?: boolean
  metadata?: Record<string, any>
}

export interface ListUsersResponse {
  users: EnrichedUser[]
  total: number
}

export interface ListUsersOptions {
  exclude_admins?: boolean
  search?: string
  limit?: number
  type?: 'app' | 'dashboard'
}

export interface InviteUserRequest {
  email: string
  role?: string
  send_email?: boolean
}

export interface InviteUserResponse {
  user: EnrichedUser
  invitation_link?: string
  message: string
}

export interface UpdateUserRoleRequest {
  role: string
}

export interface ResetUserPasswordResponse {
  message: string
}

export interface DeleteUserResponse {
  message: string
}

// ============================================================================
// API Keys Management Types
// ============================================================================

export interface APIKey {
  id: string
  name: string
  description?: string
  key_prefix: string
  scopes: string[]
  rate_limit_per_minute: number
  created_at: string
  updated_at?: string
  expires_at?: string
  revoked_at?: string
  last_used_at?: string
  user_id: string
}

export interface CreateAPIKeyRequest {
  name: string
  description?: string
  scopes: string[]
  rate_limit_per_minute: number
  expires_at?: string
}

export interface CreateAPIKeyResponse {
  api_key: APIKey
  key: string // Full key - only returned on creation
}

export interface ListAPIKeysResponse {
  api_keys: APIKey[]
  total: number
}

export interface UpdateAPIKeyRequest {
  name?: string
  description?: string
  scopes?: string[]
  rate_limit_per_minute?: number
}

export interface RevokeAPIKeyResponse {
  message: string
}

export interface DeleteAPIKeyResponse {
  message: string
}

// ============================================================================
// Webhooks Management Types
// ============================================================================

export interface Webhook {
  id: string
  url: string
  events: string[]
  secret?: string
  description?: string
  is_active: boolean
  created_at: string
  updated_at?: string
  user_id: string
}

export interface CreateWebhookRequest {
  url: string
  events: string[]
  description?: string
  secret?: string
}

export interface UpdateWebhookRequest {
  url?: string
  events?: string[]
  description?: string
  is_active?: boolean
}

export interface ListWebhooksResponse {
  webhooks: Webhook[]
  total: number
}

export interface TestWebhookResponse {
  success: boolean
  status_code?: number
  response_body?: string
  error?: string
}

export interface WebhookDelivery {
  id: string
  webhook_id: string
  event: string
  payload: Record<string, any>
  status_code?: number
  response_body?: string
  error?: string
  created_at: string
  delivered_at?: string
}

export interface ListWebhookDeliveriesResponse {
  deliveries: WebhookDelivery[]
}

export interface DeleteWebhookResponse {
  message: string
}

// ============================================================================
// Invitations Management Types
// ============================================================================

export interface Invitation {
  id: string
  email: string
  role: string
  token?: string // Only included in certain responses
  invited_by: string
  accepted_at?: string
  expires_at: string
  created_at: string
  revoked_at?: string
}

export interface CreateInvitationRequest {
  email: string
  role: 'dashboard_admin' | 'dashboard_user'
  expiry_duration?: number // Duration in seconds
}

export interface CreateInvitationResponse {
  invitation: Invitation
  invite_link: string
  email_sent: boolean
  email_status?: string
}

export interface ValidateInvitationResponse {
  valid: boolean
  invitation?: Invitation
  error?: string
}

export interface AcceptInvitationRequest {
  password: string
  name: string
}

export interface AcceptInvitationResponse {
  user: AdminUser
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface ListInvitationsOptions {
  include_accepted?: boolean
  include_expired?: boolean
}

export interface ListInvitationsResponse {
  invitations: Invitation[]
}

export interface RevokeInvitationResponse {
  message: string
}

// ============================================================================
// System Settings Types
// ============================================================================

/**
 * System setting with key-value storage
 */
export interface SystemSetting {
  id: string
  key: string
  value: Record<string, unknown>
  description?: string
  created_at: string
  updated_at: string
}

/**
 * Request to update a system setting
 */
export interface UpdateSystemSettingRequest {
  value: Record<string, unknown>
  description?: string
}

/**
 * Response containing all system settings
 */
export interface ListSystemSettingsResponse {
  settings: SystemSetting[]
}

// ============================================================================
// Application Settings Types
// ============================================================================

/**
 * Authentication settings for the application
 */
export interface AuthenticationSettings {
  enable_signup: boolean
  enable_magic_link: boolean
  password_min_length: number
  require_email_verification: boolean
}

/**
 * Feature flags for the application
 */
export interface FeatureSettings {
  enable_realtime: boolean
  enable_storage: boolean
  enable_functions: boolean
}

/**
 * Email configuration settings
 */
export interface EmailSettings {
  enabled: boolean
  provider: string
}

/**
 * Security settings for the application
 */
export interface SecuritySettings {
  enable_global_rate_limit: boolean
}

/**
 * Complete application settings structure
 */
export interface AppSettings {
  authentication: AuthenticationSettings
  features: FeatureSettings
  email: EmailSettings
  security: SecuritySettings
}

/**
 * Request to update application settings
 * All fields are optional for partial updates
 */
export interface UpdateAppSettingsRequest {
  authentication?: Partial<AuthenticationSettings>
  features?: Partial<FeatureSettings>
  email?: Partial<EmailSettings>
  security?: Partial<SecuritySettings>
}

// ============================================================================
// OAuth Provider Configuration Types
// ============================================================================

/**
 * OAuth provider configuration
 */
export interface OAuthProvider {
  id: string
  provider_name: string
  display_name: string
  enabled: boolean
  client_id: string
  client_secret?: string // Only included in certain responses
  redirect_url: string
  scopes: string[]
  is_custom: boolean
  authorization_url?: string
  token_url?: string
  user_info_url?: string
  created_at: string
  updated_at: string
}

/**
 * Request to create a new OAuth provider
 */
export interface CreateOAuthProviderRequest {
  provider_name: string
  display_name: string
  enabled: boolean
  client_id: string
  client_secret: string
  redirect_url: string
  scopes: string[]
  is_custom: boolean
  authorization_url?: string
  token_url?: string
  user_info_url?: string
}

/**
 * Response after creating an OAuth provider
 */
export interface CreateOAuthProviderResponse {
  success: boolean
  id: string
  provider: string
  message: string
  created_at: string
  updated_at: string
}

/**
 * Request to update an OAuth provider
 */
export interface UpdateOAuthProviderRequest {
  display_name?: string
  enabled?: boolean
  client_id?: string
  client_secret?: string
  redirect_url?: string
  scopes?: string[]
  authorization_url?: string
  token_url?: string
  user_info_url?: string
}

/**
 * Response after updating an OAuth provider
 */
export interface UpdateOAuthProviderResponse {
  success: boolean
  message: string
}

/**
 * Response after deleting an OAuth provider
 */
export interface DeleteOAuthProviderResponse {
  success: boolean
  message: string
}

/**
 * Response for listing OAuth providers
 */
export interface ListOAuthProvidersResponse {
  providers: OAuthProvider[]
}

/**
 * Authentication settings configuration
 */
export interface AuthSettings {
  enable_signup: boolean
  require_email_verification: boolean
  enable_magic_link: boolean
  password_min_length: number
  password_require_uppercase: boolean
  password_require_lowercase: boolean
  password_require_number: boolean
  password_require_special: boolean
  session_timeout_minutes: number
  max_sessions_per_user: number
}

/**
 * Request to update authentication settings
 */
export interface UpdateAuthSettingsRequest {
  enable_signup?: boolean
  require_email_verification?: boolean
  enable_magic_link?: boolean
  password_min_length?: number
  password_require_uppercase?: boolean
  password_require_lowercase?: boolean
  password_require_number?: boolean
  password_require_special?: boolean
  session_timeout_minutes?: number
  max_sessions_per_user?: number
}

/**
 * Response after updating authentication settings
 */
export interface UpdateAuthSettingsResponse {
  success: boolean
  message: string
}

// ============================================================================
// DDL (Data Definition Language) Types
// ============================================================================

/**
 * Column definition for creating a table
 */
export interface CreateColumnRequest {
  name: string
  type: string
  nullable?: boolean
  primaryKey?: boolean
  defaultValue?: string
}

/**
 * Request to create a new database schema
 */
export interface CreateSchemaRequest {
  name: string
}

/**
 * Response after creating a schema
 */
export interface CreateSchemaResponse {
  message: string
  schema: string
}

/**
 * Request to create a new table
 */
export interface CreateTableRequest {
  schema: string
  name: string
  columns: CreateColumnRequest[]
}

/**
 * Response after creating a table
 */
export interface CreateTableResponse {
  message: string
  schema: string
  table: string
}

/**
 * Response after deleting a table
 */
export interface DeleteTableResponse {
  message: string
}

/**
 * Database schema information
 */
export interface Schema {
  name: string
  owner?: string
}

/**
 * Response for listing schemas
 */
export interface ListSchemasResponse {
  schemas: Schema[]
}

/**
 * Table column information
 */
export interface Column {
  name: string
  type: string
  nullable: boolean
  default_value?: string
  is_primary_key?: boolean
}

/**
 * Database table information
 */
export interface Table {
  schema: string
  name: string
  columns?: Column[]
}

/**
 * Response for listing tables
 */
export interface ListTablesResponse {
  tables: Table[]
}

// ============================================================================
// User Impersonation Types
// ============================================================================

/**
 * Impersonation type
 */
export type ImpersonationType = 'user' | 'anon' | 'service'

/**
 * Target user information for impersonation
 */
export interface ImpersonationTargetUser {
  id: string
  email: string
  role: string
}

/**
 * Impersonation session information
 */
export interface ImpersonationSession {
  id: string
  admin_user_id: string
  target_user_id: string | null
  impersonation_type: ImpersonationType
  target_role: string
  reason: string
  started_at: string
  ended_at: string | null
  is_active: boolean
  ip_address: string | null
  user_agent: string | null
}

/**
 * Request to start impersonating a specific user
 */
export interface ImpersonateUserRequest {
  target_user_id: string
  reason: string
}

/**
 * Request to start impersonating as anonymous user
 */
export interface ImpersonateAnonRequest {
  reason: string
}

/**
 * Request to start impersonating with service role
 */
export interface ImpersonateServiceRequest {
  reason: string
}

/**
 * Response after starting impersonation
 */
export interface StartImpersonationResponse {
  session: ImpersonationSession
  target_user: ImpersonationTargetUser | null
  access_token: string
  refresh_token: string
  expires_in: number
}

/**
 * Response after stopping impersonation
 */
export interface StopImpersonationResponse {
  success: boolean
  message: string
}

/**
 * Response for getting current impersonation session
 */
export interface GetImpersonationResponse {
  session: ImpersonationSession | null
  target_user: ImpersonationTargetUser | null
}

/**
 * Options for listing impersonation sessions
 */
export interface ListImpersonationSessionsOptions {
  limit?: number
  offset?: number
  admin_user_id?: string
  target_user_id?: string
  impersonation_type?: ImpersonationType
  is_active?: boolean
}

/**
 * Response for listing impersonation sessions
 */
export interface ListImpersonationSessionsResponse {
  sessions: ImpersonationSession[]
  total: number
}

// ============================================================================
// Auth State Change Types
// ============================================================================

/**
 * Auth state change events
 */
export type AuthChangeEvent =
  | 'SIGNED_IN'
  | 'SIGNED_OUT'
  | 'TOKEN_REFRESHED'
  | 'USER_UPDATED'
  | 'PASSWORD_RECOVERY'
  | 'MFA_CHALLENGE_VERIFIED'

/**
 * Callback for auth state changes
 */
export type AuthStateChangeCallback = (event: AuthChangeEvent, session: AuthSession | null) => void

/**
 * Subscription object returned by onAuthStateChange
 */
export interface AuthSubscription {
  /**
   * Unsubscribe from auth state changes
   */
  unsubscribe: () => void
}
