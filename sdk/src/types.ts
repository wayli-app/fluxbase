/**
 * Core types for the Fluxbase SDK
 */

/**
 * Client configuration options (Supabase-compatible)
 * These options are passed as the third parameter to createClient()
 */
export interface FluxbaseClientOptions {
  /**
   * Authentication options
   */
  auth?: {
    /**
     * Access token for authentication
     */
    token?: string;

    /**
     * Auto-refresh token when it expires
     * @default true
     */
    autoRefresh?: boolean;

    /**
     * Persist auth state in localStorage
     * @default true
     */
    persist?: boolean;
  };

  /**
   * Global headers to include in all requests
   */
  headers?: Record<string, string>;

  /**
   * Request timeout in milliseconds
   * @default 30000
   */
  timeout?: number;

  /**
   * Enable debug logging
   * @default false
   */
  debug?: boolean;
}

export interface AuthSession {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
  expires_at?: number;
}

export interface User {
  id: string;
  email: string;
  email_verified: boolean;
  role: string;
  metadata?: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

export interface SignInCredentials {
  email: string;
  password: string;
  /** CAPTCHA token for bot protection (optional, required if CAPTCHA is enabled) */
  captchaToken?: string;
}

export interface SignUpCredentials {
  email: string;
  password: string;
  /** CAPTCHA token for bot protection (optional, required if CAPTCHA is enabled) */
  captchaToken?: string;
  options?: {
    /** User metadata to store in raw_user_meta_data (Supabase-compatible) */
    data?: Record<string, unknown>;
  };
}

/**
 * User attributes for updateUser (Supabase-compatible)
 */
export interface UpdateUserAttributes {
  /** New email address */
  email?: string;
  /** New password */
  password?: string;
  /** User metadata (Supabase-compatible) */
  data?: Record<string, unknown>;
  /** Nonce for password update reauthentication */
  nonce?: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

/**
 * MFA Factor (Supabase-compatible)
 */
export interface Factor {
  id: string;
  type: "totp" | "phone";
  status: "verified" | "unverified";
  created_at: string;
  updated_at: string;
  friendly_name?: string;
}

/**
 * TOTP setup details (Supabase-compatible)
 */
export interface TOTPSetup {
  qr_code: string;
  secret: string;
  uri: string;
}

/**
 * MFA enroll response (Supabase-compatible)
 */
export interface TwoFactorSetupResponse {
  id: string;
  type: "totp";
  totp: TOTPSetup;
}

/**
 * MFA enable response - returned when activating 2FA after setup
 */
export interface TwoFactorEnableResponse {
  success: boolean;
  backup_codes: string[];
  message: string;
}

/**
 * MFA login response - returned when verifying 2FA during login
 */
export interface TwoFactorLoginResponse {
  access_token: string;
  refresh_token: string;
  user: User;
  token_type?: string;
  expires_in?: number;
}

/**
 * MFA status response (Supabase-compatible)
 */
export interface TwoFactorStatusResponse {
  all: Factor[];
  totp: Factor[];
}

/**
 * MFA unenroll response (Supabase-compatible)
 */
export interface TwoFactorDisableResponse {
  id: string;
}

export interface TwoFactorVerifyRequest {
  user_id: string;
  code: string;
}

export interface SignInWith2FAResponse {
  requires_2fa: boolean;
  user_id: string;
  message: string;
}

export interface FluxbaseError extends Error {
  status?: number;
  code?: string;
  details?: unknown;
}

export type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD";

export interface RequestOptions {
  method: HttpMethod;
  headers?: Record<string, string>;
  body?: unknown;
  timeout?: number;
}

export interface PostgrestError {
  message: string;
  details?: string;
  hint?: string;
  code?: string;
}

export interface PostgrestResponse<T> {
  data: T | null;
  error: PostgrestError | null;
  count: number | null;
  status: number;
  statusText: string;
}

/**
 * Count type for select queries (Supabase-compatible)
 * - 'exact': Returns the exact count of rows (SELECT COUNT(*))
 * - 'planned': Uses PostgreSQL's query planner estimate (faster, less accurate)
 * - 'estimated': Uses statistics-based estimate (fastest, least accurate)
 */
export type CountType = "exact" | "planned" | "estimated";

/**
 * Options for select queries (Supabase-compatible)
 */
export interface SelectOptions {
  /**
   * Count type to use for the query
   * When specified, the count will be returned in the response
   */
  count?: CountType;
  /**
   * If true, only returns count without fetching data (HEAD request)
   * Useful for getting total count without transferring row data
   */
  head?: boolean;
}

export type FilterOperator =
  | "eq" // equals
  | "neq" // not equals
  | "gt" // greater than
  | "gte" // greater than or equal
  | "lt" // less than
  | "lte" // less than or equal
  | "like" // LIKE operator (case-sensitive)
  | "ilike" // ILIKE operator (case-insensitive)
  | "is" // IS operator (for null checks)
  | "in" // IN operator
  | "cs" // contains (array/JSONB)
  | "cd" // contained by (array/JSONB)
  | "ov" // overlaps (array)
  | "sl" // strictly left of (range)
  | "sr" // strictly right of (range)
  | "nxr" // does not extend to right (range)
  | "nxl" // does not extend to left (range)
  | "adj" // adjacent to (range)
  | "not" // negates another operator
  | "fts" // full text search
  | "plfts" // phrase full text search
  | "wfts" // web full text search
  // PostGIS spatial operators
  | "st_intersects" // geometries intersect
  | "st_contains" // geometry A contains B
  | "st_within" // geometry A is within B
  | "st_dwithin" // geometries within distance
  | "st_distance" // distance between geometries
  | "st_touches" // geometries touch
  | "st_crosses" // geometries cross
  | "st_overlaps" // geometries overlap
  | "between" // inclusive range filter (value >= min AND value <= max)
  | "not.between" // exclusive range filter (value < min OR value > max)
  // pgvector similarity operators
  | "vec_l2" // L2/Euclidean distance <-> (lower = more similar)
  | "vec_cos" // Cosine distance <=> (lower = more similar)
  | "vec_ip"; // Negative inner product <#> (lower = more similar)

export interface QueryFilter {
  column: string;
  operator: FilterOperator;
  value: unknown;
}

export type OrderDirection = "asc" | "desc";

export interface OrderBy {
  column: string;
  direction: OrderDirection;
  nulls?: "first" | "last";
  /** Vector operator for similarity ordering (vec_l2, vec_cos, vec_ip) */
  vectorOp?: "vec_l2" | "vec_cos" | "vec_ip";
  /** Vector value for similarity ordering */
  vectorValue?: number[];
}

/**
 * Options for upsert operations (Supabase-compatible)
 */
export interface UpsertOptions {
  /**
   * Comma-separated columns to use for conflict resolution
   * @example 'email'
   * @example 'user_id,tenant_id'
   */
  onConflict?: string;

  /**
   * If true, duplicate rows are ignored (not upserted)
   * @default false
   */
  ignoreDuplicates?: boolean;

  /**
   * If true, missing columns default to null instead of using existing values
   * @default false
   */
  defaultToNull?: boolean;
}

export interface RealtimeMessage {
  type:
    | "subscribe"
    | "unsubscribe"
    | "heartbeat"
    | "broadcast"
    | "presence"
    | "ack"
    | "error"
    | "postgres_changes"
    | "access_token";
  channel?: string;
  event?: string; // INSERT, UPDATE, DELETE, or *
  schema?: string;
  table?: string;
  filter?: string; // Supabase-compatible filter: column=operator.value
  payload?: unknown;
  error?: string;
  config?: PostgresChangesConfig; // Alternative format for postgres_changes
  presence?: unknown; // Presence state data
  broadcast?: unknown; // Broadcast message data
  messageId?: string; // Message ID for acknowledgments
  status?: string; // Status for acknowledgment messages
  subscription_id?: string; // Subscription ID for unsubscribe
  token?: string; // JWT token for access_token message type
}

export interface PostgresChangesConfig {
  event: "INSERT" | "UPDATE" | "DELETE" | "*";
  schema: string;
  table: string;
  filter?: string; // Optional filter: column=operator.value
}

/**
 * Realtime postgres_changes payload structure
 * Compatible with Supabase realtime payloads
 */
export interface RealtimePostgresChangesPayload<T = unknown> {
  /** Event type (Supabase-compatible field name) */
  eventType: "INSERT" | "UPDATE" | "DELETE" | "*";
  /** Database schema */
  schema: string;
  /** Table name */
  table: string;
  /** Commit timestamp (Supabase-compatible field name) */
  commit_timestamp: string;
  /** New record data (Supabase-compatible field name) */
  new: T;
  /** Old record data (Supabase-compatible field name) */
  old: T;
  /** Error message if any */
  errors: string | null;
}

/**
 * @deprecated Use RealtimePostgresChangesPayload instead
 */
export interface RealtimeChangePayload {
  /** @deprecated Use eventType instead */
  type: "INSERT" | "UPDATE" | "DELETE";
  schema: string;
  table: string;
  /** @deprecated Use 'new' instead */
  new_record?: Record<string, unknown>;
  /** @deprecated Use 'old' instead */
  old_record?: Record<string, unknown>;
  /** @deprecated Use commit_timestamp instead */
  timestamp: string;
}

export type RealtimeCallback = (
  payload: RealtimePostgresChangesPayload,
) => void;

/**
 * Realtime channel configuration options
 */
export interface RealtimeChannelConfig {
  broadcast?: {
    self?: boolean; // Receive own broadcasts (default: false)
    ack?: boolean; // Request acknowledgment (default: false)
    ackTimeout?: number; // Acknowledgment timeout in milliseconds (default: 5000)
  };
  presence?: {
    key?: string; // Custom presence key (default: auto-generated)
  };
}

/**
 * Presence state for a user
 */
export interface PresenceState {
  [key: string]: unknown;
}

/**
 * Realtime presence payload structure
 */
export interface RealtimePresencePayload {
  event: "sync" | "join" | "leave";
  key?: string;
  newPresences?: PresenceState[];
  leftPresences?: PresenceState[];
  currentPresences?: Record<string, PresenceState[]>;
}

/**
 * Presence callback type
 */
export type PresenceCallback = (payload: RealtimePresencePayload) => void;

/**
 * Broadcast message structure
 */
export interface BroadcastMessage {
  type: "broadcast";
  event: string;
  payload: unknown;
}

/**
 * Realtime broadcast payload structure
 */
export interface RealtimeBroadcastPayload {
  event: string;
  payload: unknown;
}

/**
 * Broadcast callback type
 */
export type BroadcastCallback = (payload: RealtimeBroadcastPayload) => void;

/**
 * File object returned by storage operations
 * Compatible with Supabase FileObject structure
 */
export interface FileObject {
  name: string;
  id?: string;
  bucket_id?: string;
  owner?: string;
  created_at?: string;
  updated_at?: string;
  last_accessed_at?: string;
  metadata?: Record<string, unknown>;
}

/**
 * @deprecated Use FileObject instead. This alias is provided for backwards compatibility.
 */
export type StorageObject = FileObject;

/**
 * Upload progress information
 */
export interface UploadProgress {
  /** Number of bytes uploaded so far */
  loaded: number;
  /** Total number of bytes to upload */
  total: number;
  /** Upload percentage (0-100) */
  percentage: number;
}

export interface UploadOptions {
  contentType?: string;
  metadata?: Record<string, string>;
  cacheControl?: string;
  upsert?: boolean;
  /** Optional callback to track upload progress */
  onUploadProgress?: (progress: UploadProgress) => void;
}

/**
 * Options for streaming uploads (memory-efficient for large files)
 */
export interface StreamUploadOptions {
  /** MIME type of the file */
  contentType?: string;
  /** Custom metadata to attach to the file */
  metadata?: Record<string, string>;
  /** Cache-Control header value */
  cacheControl?: string;
  /** If true, overwrite existing file at this path */
  upsert?: boolean;
  /** AbortSignal to cancel the upload */
  signal?: AbortSignal;
  /** Optional callback to track upload progress */
  onUploadProgress?: (progress: UploadProgress) => void;
}

export interface ListOptions {
  prefix?: string;
  limit?: number;
  offset?: number;
}

export interface SignedUrlOptions {
  /** Expiration time in seconds (default: 3600 = 1 hour) */
  expiresIn?: number;
  /** Image transformation options (only applies to images) */
  transform?: TransformOptions;
}

export interface DownloadOptions {
  /** If true, returns a ReadableStream instead of Blob */
  stream?: boolean;
  /**
   * Timeout in milliseconds for the download request.
   * For streaming downloads, this applies to the initial response.
   * Set to 0 or undefined for no timeout (recommended for large files).
   * @default undefined (no timeout for streaming, 30000 for non-streaming)
   */
  timeout?: number;
  /** AbortSignal to cancel the download */
  signal?: AbortSignal;
}

/** Response type for stream downloads, includes file size from Content-Length header */
export interface StreamDownloadData {
  /** The readable stream for the file content */
  stream: ReadableStream<Uint8Array>;
  /** File size in bytes from Content-Length header, or null if unknown */
  size: number | null;
}

/** Options for resumable chunked downloads */
export interface ResumableDownloadOptions {
  /**
   * Chunk size in bytes for each download request.
   * @default 5242880 (5MB)
   */
  chunkSize?: number;
  /**
   * Number of retry attempts per chunk on failure.
   * @default 3
   */
  maxRetries?: number;
  /**
   * Base delay in milliseconds for exponential backoff.
   * @default 1000
   */
  retryDelayMs?: number;
  /**
   * Timeout in milliseconds per chunk request.
   * @default 30000
   */
  chunkTimeout?: number;
  /** AbortSignal to cancel the download */
  signal?: AbortSignal;
  /** Callback for download progress */
  onProgress?: (progress: DownloadProgress) => void;
}

/** Download progress information */
export interface DownloadProgress {
  /** Number of bytes downloaded so far */
  loaded: number;
  /** Total file size in bytes, or null if unknown */
  total: number | null;
  /** Download percentage (0-100), or null if total is unknown */
  percentage: number | null;
  /** Current chunk being downloaded (1-indexed) */
  currentChunk: number;
  /** Total number of chunks, or null if total size unknown */
  totalChunks: number | null;
  /** Transfer rate in bytes per second */
  bytesPerSecond: number;
}

/** Response type for resumable downloads - stream abstracts chunking */
export interface ResumableDownloadData {
  /** The readable stream for the file content (abstracts chunking internally) */
  stream: ReadableStream<Uint8Array>;
  /** File size in bytes from HEAD request, or null if unknown */
  size: number | null;
}

/** Options for resumable chunked uploads */
export interface ResumableUploadOptions {
  /**
   * Chunk size in bytes for each upload request.
   * @default 5242880 (5MB)
   */
  chunkSize?: number;
  /**
   * Number of retry attempts per chunk on failure.
   * @default 3
   */
  maxRetries?: number;
  /**
   * Base delay in milliseconds for exponential backoff.
   * @default 1000
   */
  retryDelayMs?: number;
  /**
   * Timeout in milliseconds per chunk request.
   * @default 60000 (1 minute)
   */
  chunkTimeout?: number;
  /** AbortSignal to cancel the upload */
  signal?: AbortSignal;
  /** Callback for upload progress */
  onProgress?: (progress: ResumableUploadProgress) => void;
  /** MIME type of the file */
  contentType?: string;
  /** Custom metadata to attach to the file */
  metadata?: Record<string, string>;
  /** Cache-Control header value */
  cacheControl?: string;
  /** Existing upload session ID to resume (optional) */
  resumeSessionId?: string;
}

/** Upload progress information for resumable uploads */
export interface ResumableUploadProgress {
  /** Number of bytes uploaded so far */
  loaded: number;
  /** Total file size in bytes */
  total: number;
  /** Upload percentage (0-100) */
  percentage: number;
  /** Current chunk being uploaded (1-indexed) */
  currentChunk: number;
  /** Total number of chunks */
  totalChunks: number;
  /** Transfer rate in bytes per second */
  bytesPerSecond: number;
  /** Upload session ID (for resume capability) */
  sessionId: string;
}

/** Chunked upload session information */
export interface ChunkedUploadSession {
  /** Unique session identifier for resume */
  sessionId: string;
  /** Target bucket */
  bucket: string;
  /** Target file path */
  path: string;
  /** Total file size */
  totalSize: number;
  /** Chunk size used */
  chunkSize: number;
  /** Total number of chunks */
  totalChunks: number;
  /** Array of completed chunk indices (0-indexed) */
  completedChunks: number[];
  /** Session status */
  status: "active" | "completing" | "completed" | "aborted" | "expired";
  /** Session expiration time */
  expiresAt: string;
  /** Session creation time */
  createdAt: string;
}

/** Response from initializing a chunked upload */
export interface InitChunkedUploadResponse {
  session: ChunkedUploadSession;
}

/** Response from uploading a chunk */
export interface UploadChunkResponse {
  /** Chunk index that was uploaded */
  chunkIndex: number;
  /** ETag of the uploaded chunk */
  etag?: string;
  /** Size of the uploaded chunk in bytes */
  size: number;
  /** Updated session info */
  session: ChunkedUploadSession;
}

/** Response from completing a chunked upload */
export interface CompleteChunkedUploadResponse {
  /** Unique identifier for the uploaded file */
  id: string;
  /** File path within the bucket */
  path: string;
  /** Full path including bucket name */
  fullPath: string;
  /** Total file size in bytes */
  size: number;
  /** Content type of the file */
  contentType?: string;
}

// File Sharing Types (RLS)
export interface ShareFileOptions {
  userId: string;
  permission: "read" | "write";
}

export interface FileShare {
  user_id: string;
  permission: "read" | "write";
  created_at: string;
}

// Bucket Settings Types (RLS)
export interface BucketSettings {
  public?: boolean;
  allowed_mime_types?: string[];
  max_file_size?: number;
}

export interface Bucket {
  id: string;
  name: string;
  public: boolean;
  allowed_mime_types: string[];
  max_file_size?: number;
  created_at: string;
  updated_at: string;
}

// Password Reset Types
export interface PasswordResetRequest {
  email: string;
  /** CAPTCHA token for bot protection (optional, required if CAPTCHA is enabled) */
  captcha_token?: string;
}

/**
 * Password reset email sent response (Supabase-compatible)
 * Returns OTP-style response similar to Supabase's AuthOtpResponse
 */
export interface PasswordResetResponse {
  user: null;
  session: null;
  messageId?: string;
}

export interface VerifyResetTokenRequest {
  token: string;
}

/**
 * Verify password reset token response (Fluxbase extension)
 */
export interface VerifyResetTokenResponse {
  valid: boolean;
  message: string;
}

export interface ResetPasswordRequest {
  token: string;
  new_password: string;
}

/**
 * Reset password completion response (Supabase-compatible)
 * Returns user and session after successful password reset
 */
export type ResetPasswordResponse = AuthResponseData;

// Magic Link Types
export interface MagicLinkOptions {
  redirect_to?: string;
  /** CAPTCHA token for bot protection (optional, required if CAPTCHA is enabled) */
  captchaToken?: string;
}

export interface MagicLinkRequest {
  email: string;
  redirect_to?: string;
  captcha_token?: string;
}

/**
 * Magic link sent response (Supabase-compatible)
 * Returns OTP-style response similar to Supabase's AuthOtpResponse
 */
export interface MagicLinkResponse {
  user: null;
  session: null;
  messageId?: string;
}

export interface VerifyMagicLinkRequest {
  token: string;
}

// Anonymous Auth Types
export interface AnonymousSignInResponse extends AuthResponse {
  is_anonymous: boolean;
}

// OAuth Types
export interface OAuthProvider {
  id: string;
  name: string;
  enabled: boolean;
  authorize_url?: string;
}

export interface OAuthProvidersResponse {
  providers: OAuthProvider[];
}

export interface OAuthOptions {
  redirect_to?: string; // Post-login redirect URL (where to go after successful login)
  redirect_uri?: string; // OAuth callback URL (where OAuth provider redirects with code)
  scopes?: string[];
}

export interface OAuthUrlResponse {
  url: string;
  provider: string;
}

// OAuth Logout Types
/**
 * Options for OAuth logout
 */
export interface OAuthLogoutOptions {
  /** URL to redirect to after logout completes */
  redirect_url?: string;
}

/**
 * Response from OAuth logout endpoint
 */
export interface OAuthLogoutResponse {
  /** OAuth provider name */
  provider: string;
  /** Whether local JWT tokens were revoked */
  local_tokens_revoked: boolean;
  /** Whether the token was revoked at the OAuth provider */
  provider_token_revoked: boolean;
  /** Whether the user should be redirected to the provider's logout page */
  requires_redirect?: boolean;
  /** URL to redirect to for OIDC logout (if requires_redirect is true) */
  redirect_url?: string;
  /** Warning message if something failed but logout still proceeded */
  warning?: string;
}

// SAML SSO Types
/**
 * SAML Identity Provider configuration
 */
export interface SAMLProvider {
  /** Unique provider identifier (slug name) */
  id: string;
  /** Display name of the provider */
  name: string;
  /** Whether the provider is enabled */
  enabled: boolean;
  /** Provider's entity ID (used for SP metadata) */
  entity_id: string;
  /** SSO endpoint URL */
  sso_url: string;
  /** Single Logout endpoint URL (optional) */
  slo_url?: string;
}

/**
 * Response containing list of SAML providers
 */
export interface SAMLProvidersResponse {
  providers: SAMLProvider[];
}

/**
 * Options for initiating SAML login
 */
export interface SAMLLoginOptions {
  /** URL to redirect after successful authentication */
  redirectUrl?: string;
}

/**
 * Response containing SAML login URL
 */
export interface SAMLLoginResponse {
  /** URL to redirect user to for SAML authentication */
  url: string;
  /** Provider name */
  provider: string;
}

/**
 * SAML session information
 */
export interface SAMLSession {
  /** Session ID */
  id: string;
  /** User ID */
  user_id: string;
  /** Provider name */
  provider_name: string;
  /** SAML NameID */
  name_id: string;
  /** Session index from IdP */
  session_index?: string;
  /** SAML attributes */
  attributes?: Record<string, string[]>;
  /** Session expiration time */
  expires_at?: string;
  /** Session creation time */
  created_at: string;
}

// OTP (One-Time Password) Types
export type OTPType =
  | "signup"
  | "invite"
  | "magiclink"
  | "recovery"
  | "email_change"
  | "sms"
  | "phone_change"
  | "email";

export interface SignInWithOtpCredentials {
  email?: string;
  phone?: string;
  options?: {
    emailRedirectTo?: string;
    shouldCreateUser?: boolean;
    data?: Record<string, unknown>;
    captchaToken?: string;
  };
}

export interface VerifyOtpParams {
  email?: string;
  phone?: string;
  token: string;
  type: OTPType;
  options?: {
    redirectTo?: string;
    captchaToken?: string;
  };
}

export interface ResendOtpParams {
  type: "signup" | "sms" | "email";
  email?: string;
  phone?: string;
  options?: {
    emailRedirectTo?: string;
    captchaToken?: string;
  };
}

export interface OTPResponse {
  user: null;
  session: null;
  messageId?: string;
}

// Identity Linking Types
export interface UserIdentity {
  id: string;
  user_id: string;
  identity_data?: Record<string, unknown>;
  provider: string;
  created_at: string;
  updated_at: string;
}

export interface UserIdentitiesResponse {
  identities: UserIdentity[];
}

export interface LinkIdentityCredentials {
  provider: string;
}

export interface UnlinkIdentityParams {
  identity: UserIdentity;
}

// Reauthenticate Types
export interface ReauthenticateResponse {
  nonce: string;
}

// ID Token Types
export interface SignInWithIdTokenCredentials {
  provider: "google" | "apple";
  token: string;
  nonce?: string;
  options?: {
    captchaToken?: string;
  };
}

// Admin Authentication Types
export interface AdminSetupStatusResponse {
  needs_setup: boolean;
  has_admin: boolean;
}

export interface AdminSetupRequest {
  email: string;
  password: string;
  name: string;
  setup_token: string;
}

export interface AdminUser {
  id: string;
  email: string;
  name: string;
  role: string;
  email_verified: boolean;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
}

export interface AdminAuthResponse {
  user: AdminUser;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface AdminLoginRequest {
  email: string;
  password: string;
}

export interface AdminRefreshRequest {
  refresh_token: string;
}

export interface AdminRefreshResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: AdminUser;
}

export interface AdminMeResponse {
  user: {
    id: string;
    email: string;
    role: string;
  };
}

// User Management Types
export interface EnrichedUser {
  id: string;
  email: string;
  role?: string;
  created_at: string;
  updated_at?: string;
  email_verified?: boolean;
  last_login_at?: string;
  session_count?: number;
  is_anonymous?: boolean;
  metadata?: Record<string, unknown>;
}

export interface ListUsersResponse {
  users: EnrichedUser[];
  total: number;
}

export interface ListUsersOptions {
  exclude_admins?: boolean;
  search?: string;
  limit?: number;
  type?: "app" | "dashboard";
}

export interface InviteUserRequest {
  email: string;
  role?: string;
  send_email?: boolean;
}

export interface InviteUserResponse {
  user: EnrichedUser;
  invitation_link?: string;
  message: string;
}

export interface UpdateUserRoleRequest {
  role: string;
}

export interface ResetUserPasswordResponse {
  message: string;
}

export interface DeleteUserResponse {
  message: string;
}

// ============================================================================
// Client Keys Management Types
// ============================================================================

export interface ClientKey {
  id: string;
  name: string;
  description?: string;
  key_prefix: string;
  scopes: string[];
  rate_limit_per_minute: number;
  created_at: string;
  updated_at?: string;
  expires_at?: string;
  revoked_at?: string;
  last_used_at?: string;
  user_id: string;
}

export interface CreateClientKeyRequest {
  name: string;
  description?: string;
  scopes: string[];
  rate_limit_per_minute: number;
  expires_at?: string;
}

export interface CreateClientKeyResponse {
  client_key: ClientKey;
  key: string; // Full key - only returned on creation
}

export interface ListClientKeysResponse {
  client_keys: ClientKey[];
  total: number;
}

export interface UpdateClientKeyRequest {
  name?: string;
  description?: string;
  scopes?: string[];
  rate_limit_per_minute?: number;
}

export interface RevokeClientKeyResponse {
  message: string;
}

export interface DeleteClientKeyResponse {
  message: string;
}

// ============================================================================
// client keys Management Types (Deprecated - use Client Keys instead)
// ============================================================================

/** @deprecated Use ClientKey instead */
export type APIKey = ClientKey;

/** @deprecated Use CreateClientKeyRequest instead */
export type CreateAPIKeyRequest = CreateClientKeyRequest;

/** @deprecated Use CreateClientKeyResponse instead */
export type CreateAPIKeyResponse = CreateClientKeyResponse;

/** @deprecated Use ListClientKeysResponse instead */
export type ListAPIKeysResponse = ListClientKeysResponse;

/** @deprecated Use UpdateClientKeyRequest instead */
export type UpdateAPIKeyRequest = UpdateClientKeyRequest;

/** @deprecated Use RevokeClientKeyResponse instead */
export type RevokeAPIKeyResponse = RevokeClientKeyResponse;

/** @deprecated Use DeleteClientKeyResponse instead */
export type DeleteAPIKeyResponse = DeleteClientKeyResponse;

// ============================================================================
// Webhooks Management Types
// ============================================================================

export interface Webhook {
  id: string;
  url: string;
  events: string[];
  secret?: string;
  description?: string;
  is_active: boolean;
  created_at: string;
  updated_at?: string;
  user_id: string;
}

export interface CreateWebhookRequest {
  url: string;
  events: string[];
  description?: string;
  secret?: string;
}

export interface UpdateWebhookRequest {
  url?: string;
  events?: string[];
  description?: string;
  is_active?: boolean;
}

export interface ListWebhooksResponse {
  webhooks: Webhook[];
  total: number;
}

export interface TestWebhookResponse {
  success: boolean;
  status_code?: number;
  response_body?: string;
  error?: string;
}

export interface WebhookDelivery {
  id: string;
  webhook_id: string;
  event: string;
  payload: Record<string, unknown>;
  status_code?: number;
  response_body?: string;
  error?: string;
  created_at: string;
  delivered_at?: string;
}

export interface ListWebhookDeliveriesResponse {
  deliveries: WebhookDelivery[];
}

export interface DeleteWebhookResponse {
  message: string;
}

// ============================================================================
// Invitations Management Types
// ============================================================================

export interface Invitation {
  id: string;
  email: string;
  role: string;
  token?: string; // Only included in certain responses
  invited_by: string;
  accepted_at?: string;
  expires_at: string;
  created_at: string;
  revoked_at?: string;
}

export interface CreateInvitationRequest {
  email: string;
  role: "dashboard_admin" | "dashboard_user";
  expiry_duration?: number; // Duration in seconds
}

export interface CreateInvitationResponse {
  invitation: Invitation;
  invite_link: string;
  email_sent: boolean;
  email_status?: string;
}

export interface ValidateInvitationResponse {
  valid: boolean;
  invitation?: Invitation;
  error?: string;
}

export interface AcceptInvitationRequest {
  password: string;
  name: string;
}

export interface AcceptInvitationResponse {
  user: AdminUser;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface ListInvitationsOptions {
  include_accepted?: boolean;
  include_expired?: boolean;
}

export interface ListInvitationsResponse {
  invitations: Invitation[];
}

export interface RevokeInvitationResponse {
  message: string;
}

// ============================================================================
// System Settings Types
// ============================================================================

/**
 * Override information for a setting controlled by environment variable
 */
export interface SettingOverride {
  is_overridden: boolean;
  env_var: string;
}

/**
 * System setting with key-value storage
 */
export interface SystemSetting {
  id: string;
  key: string;
  value: Record<string, unknown>;
  description?: string;
  /** True if this setting is overridden by an environment variable */
  is_overridden?: boolean;
  /** The environment variable name if overridden */
  override_source?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Request to update a system setting
 */
export interface UpdateSystemSettingRequest {
  value: Record<string, unknown>;
  description?: string;
}

/**
 * Response containing all system settings
 */
export interface ListSystemSettingsResponse {
  settings: SystemSetting[];
}

// ============================================================================
// Custom Settings Types
// ============================================================================

/**
 * Custom setting with flexible key-value storage and role-based editing permissions
 */
export interface CustomSetting {
  id: string;
  key: string;
  value: Record<string, unknown>;
  value_type: "string" | "number" | "boolean" | "json";
  description?: string;
  editable_by: string[];
  metadata?: Record<string, unknown>;
  created_by?: string;
  updated_by?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Request to create a custom setting
 */
export interface CreateCustomSettingRequest {
  key: string;
  value: Record<string, unknown>;
  value_type?: "string" | "number" | "boolean" | "json";
  description?: string;
  editable_by?: string[];
  metadata?: Record<string, unknown>;
}

/**
 * Request to update a custom setting
 */
export interface UpdateCustomSettingRequest {
  value: Record<string, unknown>;
  description?: string;
  editable_by?: string[];
  metadata?: Record<string, unknown>;
}

/**
 * Response containing all custom settings
 */
export interface ListCustomSettingsResponse {
  settings: CustomSetting[];
}

/**
 * Metadata for a secret setting (value is never exposed via API)
 * Secret values can only be accessed server-side (in edge functions, jobs, handlers)
 */
export interface SecretSettingMetadata {
  id: string;
  key: string;
  description?: string;
  user_id?: string;
  created_by?: string;
  updated_by?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Request to create a secret setting
 */
export interface CreateSecretSettingRequest {
  key: string;
  value: string;
  description?: string;
}

/**
 * Request to update a secret setting
 */
export interface UpdateSecretSettingRequest {
  value?: string;
  description?: string;
}

// ============================================================================
// Application Settings Types
// ============================================================================

/**
 * Authentication settings for the application
 */
export interface AuthenticationSettings {
  enable_signup: boolean;
  enable_magic_link: boolean;
  password_min_length: number;
  require_email_verification: boolean;
  password_require_uppercase: boolean;
  password_require_lowercase: boolean;
  password_require_number: boolean;
  password_require_special: boolean;
  session_timeout_minutes: number;
  max_sessions_per_user: number;
}

/**
 * Feature flags for the application
 */
export interface FeatureSettings {
  enable_realtime: boolean;
  enable_storage: boolean;
  enable_functions: boolean;
  enable_ai: boolean;
  enable_jobs: boolean;
  enable_rpc: boolean;
}

/**
 * SMTP email provider configuration
 */
export interface SMTPSettings {
  host: string;
  port: number;
  username: string;
  password: string;
  use_tls: boolean;
}

/**
 * SendGrid email provider configuration
 */
export interface SendGridSettings {
  api_key: string;
}

/**
 * Mailgun email provider configuration
 */
export interface MailgunSettings {
  api_key: string;
  domain: string;
  eu_region: boolean;
}

/**
 * AWS SES email provider configuration
 */
export interface SESSettings {
  access_key_id: string;
  secret_access_key: string;
  region: string;
}

/**
 * Email configuration settings
 */
export interface EmailSettings {
  enabled: boolean;
  provider: "smtp" | "sendgrid" | "mailgun" | "ses";
  from_address?: string;
  from_name?: string;
  reply_to_address?: string;
  smtp?: SMTPSettings;
  sendgrid?: SendGridSettings;
  mailgun?: MailgunSettings;
  ses?: SESSettings;
}

/**
 * Security settings for the application
 */
export interface SecuritySettings {
  enable_global_rate_limit: boolean;
}

/**
 * Indicates which settings are overridden by environment variables (read-only)
 */
export interface SettingOverrides {
  authentication?: Record<string, boolean>;
  features?: Record<string, boolean>;
  email?: Record<string, boolean>;
  security?: Record<string, boolean>;
}

/**
 * Complete application settings structure
 */
export interface AppSettings {
  authentication: AuthenticationSettings;
  features: FeatureSettings;
  email: EmailSettings;
  security: SecuritySettings;
  /** Settings overridden by environment variables (read-only, cannot be modified via API) */
  overrides?: SettingOverrides;
}

/**
 * Request to update application settings
 * All fields are optional for partial updates
 */
export interface UpdateAppSettingsRequest {
  authentication?: Partial<AuthenticationSettings>;
  features?: Partial<FeatureSettings>;
  email?: Partial<EmailSettings>;
  security?: Partial<SecuritySettings>;
}

// ============================================================================
// Email Template Types
// ============================================================================

/**
 * Email template type
 */
export type EmailTemplateType =
  | "magic_link"
  | "verify_email"
  | "reset_password"
  | "invite_user";

/**
 * Email template structure
 */
export interface EmailTemplate {
  id: string;
  template_type: EmailTemplateType;
  subject: string;
  html_body: string;
  text_body?: string;
  is_custom: boolean;
  created_at: string;
  updated_at: string;
}

/**
 * Request to update an email template
 */
export interface UpdateEmailTemplateRequest {
  subject: string;
  html_body: string;
  text_body?: string;
}

/**
 * Request to test an email template
 */
export interface TestEmailTemplateRequest {
  recipient_email: string;
}

/**
 * Response when listing email templates
 */
export interface ListEmailTemplatesResponse {
  templates: EmailTemplate[];
}

// ============================================================================
// Email Provider Settings Types (Admin API)
// ============================================================================

/**
 * Override information for a setting controlled by environment variable
 */
export interface EmailSettingOverride {
  is_overridden: boolean;
  env_var: string;
}

/**
 * Email provider settings response from /api/v1/admin/email/settings
 *
 * This is the flat structure returned by the admin API, which differs from
 * the nested EmailSettings structure used in AppSettings.
 */
export interface EmailProviderSettings {
  enabled: boolean;
  provider: "smtp" | "sendgrid" | "mailgun" | "ses";
  from_address: string;
  from_name: string;

  // SMTP settings
  smtp_host: string;
  smtp_port: number;
  smtp_username: string;
  smtp_password_set: boolean; // true if password is configured (never returns actual value)
  smtp_tls: boolean;

  // SendGrid
  sendgrid_api_key_set: boolean; // true if API key is configured

  // Mailgun
  mailgun_api_key_set: boolean; // true if API key is configured
  mailgun_domain: string;

  // AWS SES
  ses_access_key_set: boolean; // true if access key is configured
  ses_secret_key_set: boolean; // true if secret key is configured
  ses_region: string;

  /** Settings overridden by environment variables */
  _overrides: Record<string, EmailSettingOverride>;
}

/**
 * Request to update email provider settings
 *
 * All fields are optional - only provided fields will be updated.
 * Secret fields (passwords, client keys) are only updated if provided.
 */
export interface UpdateEmailProviderSettingsRequest {
  enabled?: boolean;
  provider?: "smtp" | "sendgrid" | "mailgun" | "ses";
  from_address?: string;
  from_name?: string;

  // SMTP settings
  smtp_host?: string;
  smtp_port?: number;
  smtp_username?: string;
  smtp_password?: string; // Only set if changing
  smtp_tls?: boolean;

  // SendGrid
  sendgrid_api_key?: string; // Only set if changing

  // Mailgun
  mailgun_api_key?: string; // Only set if changing
  mailgun_domain?: string;

  // AWS SES
  ses_access_key?: string; // Only set if changing
  ses_secret_key?: string; // Only set if changing
  ses_region?: string;
}

/**
 * Response from testing email settings
 */
export interface TestEmailSettingsResponse {
  success: boolean;
  message: string;
}

// ============================================================================
// OAuth Provider Configuration Types
// ============================================================================

/**
 * OAuth provider configuration
 */
export interface OAuthProvider {
  id: string;
  provider_name: string;
  display_name: string;
  enabled: boolean;
  client_id: string;
  client_secret?: string; // Only included in certain responses
  redirect_url: string;
  scopes: string[];
  is_custom: boolean;
  authorization_url?: string;
  token_url?: string;
  user_info_url?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Request to create a new OAuth provider
 */
export interface CreateOAuthProviderRequest {
  provider_name: string;
  display_name: string;
  enabled: boolean;
  client_id: string;
  client_secret: string;
  redirect_url: string;
  scopes: string[];
  is_custom: boolean;
  authorization_url?: string;
  token_url?: string;
  user_info_url?: string;
}

/**
 * Response after creating an OAuth provider
 */
export interface CreateOAuthProviderResponse {
  success: boolean;
  id: string;
  provider: string;
  message: string;
  created_at: string;
  updated_at: string;
}

/**
 * Request to update an OAuth provider
 */
export interface UpdateOAuthProviderRequest {
  display_name?: string;
  enabled?: boolean;
  client_id?: string;
  client_secret?: string;
  redirect_url?: string;
  scopes?: string[];
  authorization_url?: string;
  token_url?: string;
  user_info_url?: string;
}

/**
 * Response after updating an OAuth provider
 */
export interface UpdateOAuthProviderResponse {
  success: boolean;
  message: string;
}

/**
 * Response after deleting an OAuth provider
 */
export interface DeleteOAuthProviderResponse {
  success: boolean;
  message: string;
}

/**
 * Response for listing OAuth providers
 */
export interface ListOAuthProvidersResponse {
  providers: OAuthProvider[];
}

/**
 * Authentication settings configuration
 */
export interface AuthSettings {
  enable_signup: boolean;
  require_email_verification: boolean;
  enable_magic_link: boolean;
  password_min_length: number;
  password_require_uppercase: boolean;
  password_require_lowercase: boolean;
  password_require_number: boolean;
  password_require_special: boolean;
  session_timeout_minutes: number;
  max_sessions_per_user: number;
  disable_dashboard_password_login: boolean;
  disable_app_password_login: boolean;
  /** Settings overridden by environment variables (read-only, cannot be modified via API) */
  _overrides?: Record<string, SettingOverride>;
}

/**
 * Request to update authentication settings
 */
export interface UpdateAuthSettingsRequest {
  enable_signup?: boolean;
  require_email_verification?: boolean;
  enable_magic_link?: boolean;
  password_min_length?: number;
  password_require_uppercase?: boolean;
  password_require_lowercase?: boolean;
  password_require_number?: boolean;
  password_require_special?: boolean;
  session_timeout_minutes?: number;
  max_sessions_per_user?: number;
  disable_dashboard_password_login?: boolean;
  disable_app_password_login?: boolean;
}

/**
 * Response after updating authentication settings
 */
export interface UpdateAuthSettingsResponse {
  success: boolean;
  message: string;
}

// ============================================================================
// DDL (Data Definition Language) Types
// ============================================================================

/**
 * Column definition for creating a table
 */
export interface CreateColumnRequest {
  name: string;
  type: string;
  nullable?: boolean;
  primaryKey?: boolean;
  defaultValue?: string;
}

/**
 * Request to create a new database schema
 */
export interface CreateSchemaRequest {
  name: string;
}

/**
 * Response after creating a schema
 */
export interface CreateSchemaResponse {
  message: string;
  schema: string;
}

/**
 * Request to create a new table
 */
export interface CreateTableRequest {
  schema: string;
  name: string;
  columns: CreateColumnRequest[];
}

/**
 * Response after creating a table
 */
export interface CreateTableResponse {
  message: string;
  schema: string;
  table: string;
}

/**
 * Response after deleting a table
 */
export interface DeleteTableResponse {
  message: string;
}

/**
 * Database schema information
 */
export interface Schema {
  name: string;
  owner?: string;
}

/**
 * Response for listing schemas
 */
export interface ListSchemasResponse {
  schemas: Schema[];
}

/**
 * Table column information
 */
export interface Column {
  name: string;
  type: string;
  nullable: boolean;
  default_value?: string;
  is_primary_key?: boolean;
}

/**
 * Database table information
 */
export interface Table {
  schema: string;
  name: string;
  columns?: Column[];
}

/**
 * Response for listing tables
 */
export interface ListTablesResponse {
  tables: Table[];
}

// ============================================================================
// User Impersonation Types
// ============================================================================

/**
 * Impersonation type
 */
export type ImpersonationType = "user" | "anon" | "service";

/**
 * Target user information for impersonation
 */
export interface ImpersonationTargetUser {
  id: string;
  email: string;
  role: string;
}

/**
 * Impersonation session information
 */
export interface ImpersonationSession {
  id: string;
  admin_user_id: string;
  target_user_id: string | null;
  impersonation_type: ImpersonationType;
  target_role: string;
  reason: string;
  started_at: string;
  ended_at: string | null;
  is_active: boolean;
  ip_address: string | null;
  user_agent: string | null;
}

/**
 * Request to start impersonating a specific user
 */
export interface ImpersonateUserRequest {
  target_user_id: string;
  reason: string;
}

/**
 * Request to start impersonating as anonymous user
 */
export interface ImpersonateAnonRequest {
  reason: string;
}

/**
 * Request to start impersonating with service role
 */
export interface ImpersonateServiceRequest {
  reason: string;
}

/**
 * Response after starting impersonation
 */
export interface StartImpersonationResponse {
  session: ImpersonationSession;
  target_user: ImpersonationTargetUser | null;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

/**
 * Response after stopping impersonation
 */
export interface StopImpersonationResponse {
  success: boolean;
  message: string;
}

/**
 * Response for getting current impersonation session
 */
export interface GetImpersonationResponse {
  session: ImpersonationSession | null;
  target_user: ImpersonationTargetUser | null;
}

/**
 * Options for listing impersonation sessions
 */
export interface ListImpersonationSessionsOptions {
  limit?: number;
  offset?: number;
  admin_user_id?: string;
  target_user_id?: string;
  impersonation_type?: ImpersonationType;
  is_active?: boolean;
}

/**
 * Response for listing impersonation sessions
 */
export interface ListImpersonationSessionsResponse {
  sessions: ImpersonationSession[];
  total: number;
}

// ============================================================================
// Image Transform Types
// ============================================================================

/**
 * Fit mode for image transformations
 * - cover: Resize to cover target dimensions, cropping if needed (default)
 * - contain: Resize to fit within target dimensions, letterboxing if needed
 * - fill: Stretch to exactly fill target dimensions
 * - inside: Resize to fit within target, only scale down
 * - outside: Resize to be at least as large as target
 */
export type ImageFitMode = "cover" | "contain" | "fill" | "inside" | "outside";

/**
 * Output format for image transformations
 */
export type ImageFormat = "webp" | "jpg" | "png" | "avif";

/**
 * Options for on-the-fly image transformations
 * Applied to storage downloads via query parameters
 */
export interface TransformOptions {
  /** Target width in pixels (0 or undefined = auto based on height) */
  width?: number;
  /** Target height in pixels (0 or undefined = auto based on width) */
  height?: number;
  /** Output format (defaults to original format) */
  format?: ImageFormat;
  /** Output quality 1-100 (default: 80) */
  quality?: number;
  /** How to fit the image within target dimensions (default: cover) */
  fit?: ImageFitMode;
}

// ============================================================================
// CAPTCHA Types
// ============================================================================

/**
 * CAPTCHA provider types supported by Fluxbase
 * - hcaptcha: Privacy-focused visual challenge
 * - recaptcha_v3: Google's invisible risk-based CAPTCHA
 * - turnstile: Cloudflare's invisible CAPTCHA
 * - cap: Self-hosted proof-of-work CAPTCHA (https://capjs.js.org/)
 */
export type CaptchaProvider = "hcaptcha" | "recaptcha_v3" | "turnstile" | "cap";

/**
 * Public CAPTCHA configuration returned from the server
 * Used by clients to know which CAPTCHA provider to load
 */
export interface CaptchaConfig {
  /** Whether CAPTCHA is enabled */
  enabled: boolean;
  /** CAPTCHA provider name */
  provider?: CaptchaProvider;
  /** Public site key for the CAPTCHA widget (hcaptcha, recaptcha, turnstile) */
  site_key?: string;
  /** Endpoints that require CAPTCHA verification */
  endpoints?: string[];
  /** Cap server URL - only present when provider is 'cap' */
  cap_server_url?: string;
}

/**
 * Public OAuth provider information
 */
export interface OAuthProviderPublic {
  /** Provider identifier (e.g., "google", "github") */
  provider: string;
  /** Display name for UI */
  display_name: string;
  /** Authorization URL to initiate OAuth flow */
  authorize_url: string;
}

/**
 * Public SAML provider information
 */
export interface SAMLProvider {
  /** Provider identifier */
  provider: string;
  /** Display name for UI */
  display_name: string;
}

/**
 * Comprehensive authentication configuration
 * Returns all public auth settings from the server
 */
export interface AuthConfig {
  /** Whether user signup is enabled */
  signup_enabled: boolean;
  /** Whether email verification is required after signup */
  require_email_verification: boolean;
  /** Whether magic link authentication is enabled */
  magic_link_enabled: boolean;
  /** Whether password login is enabled for app users */
  password_login_enabled: boolean;
  /** Whether MFA/2FA is available (always true, users opt-in) */
  mfa_available: boolean;
  /** Minimum password length requirement */
  password_min_length: number;
  /** Whether passwords must contain uppercase letters */
  password_require_uppercase: boolean;
  /** Whether passwords must contain lowercase letters */
  password_require_lowercase: boolean;
  /** Whether passwords must contain numbers */
  password_require_number: boolean;
  /** Whether passwords must contain special characters */
  password_require_special: boolean;
  /** Available OAuth providers for authentication */
  oauth_providers: OAuthProviderPublic[];
  /** Available SAML providers for enterprise SSO */
  saml_providers: SAMLProvider[];
  /** CAPTCHA configuration */
  captcha: CaptchaConfig | null;
}

// ============================================================================
// Auth State Change Types
// ============================================================================

/**
 * Auth state change events
 */
export type AuthChangeEvent =
  | "SIGNED_IN"
  | "SIGNED_OUT"
  | "TOKEN_REFRESHED"
  | "USER_UPDATED"
  | "PASSWORD_RECOVERY"
  | "MFA_CHALLENGE_VERIFIED";

/**
 * Callback for auth state changes
 */
export type AuthStateChangeCallback = (
  event: AuthChangeEvent,
  session: AuthSession | null,
) => void;

/**
 * Subscription object returned by onAuthStateChange
 */
export interface AuthSubscription {
  /**
   * Unsubscribe from auth state changes
   */
  unsubscribe: () => void;
}

/**
 * Options for invoking an edge function
 */
export interface FunctionInvokeOptions {
  /**
   * Request body to send to the function
   */
  body?: unknown;

  /**
   * Custom headers to include in the request
   */
  headers?: Record<string, string>;

  /**
   * HTTP method to use
   * @default 'POST'
   */
  method?: "GET" | "POST" | "PUT" | "DELETE" | "PATCH";

  /**
   * Namespace of the function to invoke
   * If not provided, the first function with the given name is used (alphabetically by namespace)
   */
  namespace?: string;
}

/**
 * Edge function metadata
 */
export interface EdgeFunction {
  id: string;
  name: string;
  namespace: string;
  description?: string;
  code: string;
  version: number;
  enabled: boolean;
  timeout_seconds: number;
  memory_limit_mb: number;
  allow_net: boolean;
  allow_env: boolean;
  allow_read: boolean;
  allow_write: boolean;
  allow_unauthenticated: boolean;
  cron_schedule?: string;
  created_at: string;
  updated_at: string;
  created_by?: string;
}

/**
 * Request to create a new edge function
 */
export interface CreateFunctionRequest {
  name: string;
  description?: string;
  code: string;
  enabled?: boolean;
  timeout_seconds?: number;
  memory_limit_mb?: number;
  allow_net?: boolean;
  allow_env?: boolean;
  allow_read?: boolean;
  allow_write?: boolean;
  allow_unauthenticated?: boolean;
  cron_schedule?: string;
}

/**
 * Request to update an existing edge function
 */
export interface UpdateFunctionRequest {
  description?: string;
  code?: string;
  enabled?: boolean;
  timeout_seconds?: number;
  memory_limit_mb?: number;
  allow_net?: boolean;
  allow_env?: boolean;
  allow_read?: boolean;
  allow_write?: boolean;
  allow_unauthenticated?: boolean;
  cron_schedule?: string;
}

/**
 * Edge function execution record
 */
export interface EdgeFunctionExecution {
  id: string;
  function_id: string;
  trigger_type: string;
  status: "success" | "error";
  status_code?: number;
  duration_ms?: number;
  result?: string;
  logs?: string;
  error_message?: string;
  executed_at: string;
  completed_at?: string;
}

/**
 * Function specification for bulk sync operations
 */
export interface FunctionSpec {
  name: string;
  description?: string;
  code: string;
  /** If true, code is already bundled and server will skip bundling */
  is_pre_bundled?: boolean;
  /** Original source code (for debugging when pre-bundled) */
  original_code?: string;
  /** Source directory for resolving relative imports during bundling (used by syncWithBundling) */
  sourceDir?: string;
  /** Additional paths to search for node_modules during bundling (used by syncWithBundling) */
  nodePaths?: string[];
  enabled?: boolean;
  timeout_seconds?: number;
  memory_limit_mb?: number;
  allow_net?: boolean;
  allow_env?: boolean;
  allow_read?: boolean;
  allow_write?: boolean;
  allow_unauthenticated?: boolean;
  is_public?: boolean;
  cron_schedule?: string;
}

/**
 * Options for syncing functions
 */
export interface SyncFunctionsOptions {
  /** Namespace to sync functions to (defaults to "default") */
  namespace?: string;
  /** Functions to sync */
  functions: FunctionSpec[];
  /** Options for sync operation */
  options?: {
    /** Delete functions in namespace that are not in the sync payload */
    delete_missing?: boolean;
    /** Preview changes without applying them */
    dry_run?: boolean;
  };
}

/**
 * Sync operation error details
 */
export interface SyncError {
  /** Name of the function that failed */
  function: string;
  /** Error message */
  error: string;
  /** Operation that failed */
  action: "create" | "update" | "delete" | "bundle";
}

/**
 * Result of a function sync operation
 */
export interface SyncFunctionsResult {
  /** Status message */
  message: string;
  /** Namespace that was synced */
  namespace: string;
  /** Summary counts */
  summary: {
    created: number;
    updated: number;
    deleted: number;
    unchanged: number;
    errors: number;
  };
  /** Detailed results */
  details: {
    created: string[];
    updated: string[];
    deleted: string[];
    unchanged: string[];
  };
  /** Errors encountered */
  errors: SyncError[];
  /** Whether this was a dry run */
  dry_run: boolean;
}

// ============================================================================
// Background Jobs Types
// ============================================================================

/**
 * Job function metadata
 */
export interface JobFunction {
  id: string;
  name: string;
  namespace: string;
  description?: string;
  code?: string;
  original_code?: string;
  is_bundled: boolean;
  bundle_error?: string;
  enabled: boolean;
  schedule?: string;
  timeout_seconds: number;
  memory_limit_mb: number;
  max_retries: number;
  progress_timeout_seconds: number;
  allow_net: boolean;
  allow_env: boolean;
  allow_read: boolean;
  allow_write: boolean;
  require_role?: string;
  version: number;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Request to create a new job function
 */
export interface CreateJobFunctionRequest {
  name: string;
  namespace?: string;
  description?: string;
  code: string;
  enabled?: boolean;
  schedule?: string;
  timeout_seconds?: number;
  memory_limit_mb?: number;
  max_retries?: number;
  progress_timeout_seconds?: number;
  allow_net?: boolean;
  allow_env?: boolean;
  allow_read?: boolean;
  allow_write?: boolean;
  require_role?: string;
}

/**
 * Request to update an existing job function
 */
export interface UpdateJobFunctionRequest {
  description?: string;
  code?: string;
  enabled?: boolean;
  schedule?: string;
  timeout_seconds?: number;
  memory_limit_mb?: number;
  max_retries?: number;
  progress_timeout_seconds?: number;
  allow_net?: boolean;
  allow_env?: boolean;
  allow_read?: boolean;
  allow_write?: boolean;
  require_role?: string;
}

/**
 * Job execution status
 */
export type JobStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "cancelled"
  | "timeout";

/**
 * Job execution record
 */
export interface Job {
  id: string;
  namespace: string;
  job_function_id?: string;
  job_name: string;
  status: JobStatus;
  payload?: unknown;
  result?: unknown;
  error?: string;
  logs?: string;
  priority: number;
  max_duration_seconds?: number;
  progress_timeout_seconds?: number;
  progress_percent?: number;
  progress_message?: string;
  progress_data?: unknown;
  max_retries: number;
  retry_count: number;
  worker_id?: string;
  created_by?: string;
  user_role?: string;
  user_email?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  scheduled_at?: string;
  last_progress_at?: string;
  /** Estimated completion time (computed, only for running jobs with progress > 0) */
  estimated_completion_at?: string;
  /** Estimated seconds remaining (computed, only for running jobs with progress > 0) */
  estimated_seconds_left?: number;
}

/**
 * User context for submitting jobs on behalf of another user.
 * Only available when using service_role authentication.
 */
export interface OnBehalfOf {
  /** User ID (UUID) to submit the job as */
  user_id: string;
  /** Optional email address of the user */
  user_email?: string;
  /** Optional role of the user (defaults to "authenticated") */
  user_role?: string;
}

/**
 * Request to submit a new job
 */
export interface SubmitJobRequest {
  job_name: string;
  namespace?: string;
  payload?: unknown;
  priority?: number;
  scheduled?: string;
  /**
   * Submit job on behalf of another user.
   * Only available when using service_role authentication.
   * The job will be created with the specified user's identity,
   * allowing them to see the job and its logs via RLS.
   */
  on_behalf_of?: OnBehalfOf;
}

/**
 * Job statistics
 */
export interface JobStats {
  namespace?: string;
  pending: number;
  running: number;
  completed: number;
  failed: number;
  cancelled: number;
  total: number;
}

/**
 * Job worker information
 */
export interface JobWorker {
  id: string;
  hostname: string;
  status: "active" | "idle" | "dead";
  current_jobs: number;
  total_completed: number;
  started_at: string;
  last_heartbeat_at: string;
}

/**
 * Job function specification for sync operations
 */
export interface JobFunctionSpec {
  name: string;
  description?: string;
  code: string;
  /** If true, code is already bundled and server will skip bundling */
  is_pre_bundled?: boolean;
  /** Original source code (for debugging when pre-bundled) */
  original_code?: string;
  /** Source directory for resolving relative imports during bundling (used by syncWithBundling) */
  sourceDir?: string;
  /** Additional paths to search for node_modules during bundling (used by syncWithBundling) */
  nodePaths?: string[];
  enabled?: boolean;
  schedule?: string;
  timeout_seconds?: number;
  memory_limit_mb?: number;
  max_retries?: number;
  progress_timeout_seconds?: number;
  allow_net?: boolean;
  allow_env?: boolean;
  allow_read?: boolean;
  allow_write?: boolean;
  require_role?: string;
}

/**
 * Options for syncing job functions
 */
export interface SyncJobsOptions {
  namespace: string;
  functions?: JobFunctionSpec[];
  options?: {
    delete_missing?: boolean;
    dry_run?: boolean;
  };
}

/**
 * Result of a job sync operation
 */
export interface SyncJobsResult {
  message: string;
  namespace: string;
  summary: {
    created: number;
    updated: number;
    deleted: number;
    unchanged: number;
    errors: number;
  };
  details: {
    created: string[];
    updated: string[];
    deleted: string[];
    unchanged: string[];
  };
  errors: SyncError[];
  dry_run: boolean;
}

// ============================================================================
// Database Migrations Types
// ============================================================================

/**
 * Database migration metadata
 */
export interface Migration {
  id: string;
  namespace: string;
  name: string;
  description?: string;
  up_sql: string;
  down_sql?: string;
  version: number;
  status: "pending" | "applied" | "failed" | "rolled_back";
  created_by?: string;
  applied_by?: string;
  created_at: string;
  updated_at: string;
  applied_at?: string;
  rolled_back_at?: string;
}

/**
 * Request to create a new migration
 */
export interface CreateMigrationRequest {
  namespace?: string;
  name: string;
  description?: string;
  up_sql: string;
  down_sql?: string;
}

/**
 * Request to update a migration (only if pending)
 */
export interface UpdateMigrationRequest {
  description?: string;
  up_sql?: string;
  down_sql?: string;
}

/**
 * Migration execution record (audit log)
 */
export interface MigrationExecution {
  id: string;
  migration_id: string;
  action: "apply" | "rollback";
  status: "success" | "failed";
  duration_ms?: number;
  error_message?: string;
  logs?: string;
  executed_at: string;
  executed_by?: string;
}

/**
 * Request to apply a migration
 */
export interface ApplyMigrationRequest {
  namespace?: string;
}

/**
 * Request to rollback a migration
 */
export interface RollbackMigrationRequest {
  namespace?: string;
}

/**
 * Request to apply pending migrations
 */
export interface ApplyPendingRequest {
  namespace?: string;
}

/**
 * Options for syncing migrations
 */
export interface SyncMigrationsOptions {
  /** Update pending migrations if SQL content changed */
  update_if_changed?: boolean;
  /** Automatically apply new migrations after sync */
  auto_apply?: boolean;
  /** Preview changes without applying them */
  dry_run?: boolean;
}

/**
 * Result of a migration sync operation
 */
export interface SyncMigrationsResult {
  /** Status message */
  message: string;
  /** Namespace that was synced */
  namespace: string;
  /** Summary counts */
  summary: {
    created: number;
    updated: number;
    unchanged: number;
    skipped: number;
    applied: number;
    errors: number;
  };
  /** Detailed results */
  details: {
    created: string[];
    updated: string[];
    unchanged: string[];
    skipped: string[];
    applied: string[];
    errors: string[];
  };
  /** Whether this was a dry run */
  dry_run: boolean;
  /** Warning messages */
  warnings?: string[];
}

// ============================================================================
// AI Chatbot Types
// ============================================================================

/**
 * AI provider type
 */
export type AIProviderType = "openai" | "azure" | "ollama";

/**
 * AI provider configuration
 */
export interface AIProvider {
  id: string;
  name: string;
  display_name: string;
  provider_type: AIProviderType;
  is_default: boolean;
  enabled: boolean;
  config: Record<string, string>;
  /** True if provider was configured via environment variables or fluxbase.yaml */
  from_config?: boolean;
  /** @deprecated Use from_config instead */
  read_only?: boolean;
  created_at: string;
  updated_at: string;
}

/**
 * Request to create an AI provider
 * Note: config values can be strings, numbers, or booleans - they will be converted to strings automatically
 */
export interface CreateAIProviderRequest {
  name: string;
  display_name: string;
  provider_type: AIProviderType;
  is_default?: boolean;
  enabled?: boolean;
  config: Record<string, string | number | boolean>;
}

/**
 * Request to update an AI provider
 * Note: config values can be strings, numbers, or booleans - they will be converted to strings automatically
 */
export interface UpdateAIProviderRequest {
  display_name?: string;
  config?: Record<string, string | number | boolean>;
  enabled?: boolean;
}

/**
 * AI chatbot summary (list view)
 */
export interface AIChatbotSummary {
  id: string;
  name: string;
  namespace: string;
  description?: string;
  enabled: boolean;
  is_public: boolean;
  allowed_tables: string[];
  allowed_operations: string[];
  allowed_schemas: string[];
  version: number;
  source: string;
  created_at: string;
  updated_at: string;
}

/**
 * AI chatbot full details
 */
export interface AIChatbot extends AIChatbotSummary {
  code: string;
  original_code?: string;
  max_tokens: number;
  temperature: number;
  provider_id?: string;
  persist_conversations: boolean;
  conversation_ttl_hours: number;
  max_conversation_turns: number;
  rate_limit_per_minute: number;
  daily_request_limit: number;
  daily_token_budget: number;
  allow_unauthenticated: boolean;
}

/**
 * Chatbot specification for sync operations
 */
export interface ChatbotSpec {
  name: string;
  description?: string;
  code: string;
  original_code?: string;
  is_pre_bundled?: boolean;
  enabled?: boolean;
  allowed_tables?: string[];
  allowed_operations?: string[];
  allowed_schemas?: string[];
  max_tokens?: number;
  temperature?: number;
  persist_conversations?: boolean;
  conversation_ttl_hours?: number;
  max_conversation_turns?: number;
  rate_limit_per_minute?: number;
  daily_request_limit?: number;
  daily_token_budget?: number;
  allow_unauthenticated?: boolean;
  is_public?: boolean;
}

/**
 * Options for syncing chatbots
 */
export interface SyncChatbotsOptions {
  namespace?: string;
  chatbots?: ChatbotSpec[];
  options?: {
    delete_missing?: boolean;
    dry_run?: boolean;
  };
}

/**
 * Result of a chatbot sync operation
 */
export interface SyncChatbotsResult {
  message: string;
  namespace: string;
  summary: {
    created: number;
    updated: number;
    deleted: number;
    unchanged: number;
    errors: number;
  };
  details: {
    created: string[];
    updated: string[];
    deleted: string[];
    unchanged: string[];
  };
  errors: SyncError[];
  dry_run: boolean;
}

/**
 * AI chat message role
 */
export type AIChatMessageRole = "user" | "assistant" | "system" | "tool";

/**
 * AI chat message for WebSocket
 */
export interface AIChatClientMessage {
  type: "start_chat" | "message" | "cancel";
  chatbot?: string;
  namespace?: string;
  conversation_id?: string;
  content?: string;
  impersonate_user_id?: string; // Admin-only: test as this user
}

/**
 * AI chat server message
 */
export interface AIChatServerMessage {
  type:
    | "chat_started"
    | "progress"
    | "content"
    | "query_result"
    | "done"
    | "error"
    | "cancelled";
  conversation_id?: string;
  message_id?: string;
  chatbot?: string;
  step?: string;
  message?: string;
  delta?: string;
  query?: string;
  summary?: string;
  row_count?: number;
  data?: Record<string, unknown>[];
  usage?: AIUsageStats;
  error?: string;
  code?: string;
}

/**
 * AI token usage statistics
 */
export interface AIUsageStats {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens?: number;
}

/**
 * AI conversation summary
 */
export interface AIConversation {
  id: string;
  chatbot_id: string;
  user_id?: string;
  session_id?: string;
  title?: string;
  status: "active" | "archived";
  turn_count: number;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  created_at: string;
  updated_at: string;
  last_message_at: string;
  expires_at?: string;
}

/**
 * AI conversation message
 */
export interface AIConversationMessage {
  id: string;
  conversation_id: string;
  role: AIChatMessageRole;
  content: string;
  tool_call_id?: string;
  tool_name?: string;
  executed_sql?: string;
  sql_result_summary?: string;
  sql_row_count?: number;
  sql_error?: string;
  sql_duration_ms?: number;
  prompt_tokens?: number;
  completion_tokens?: number;
  created_at: string;
  sequence_number: number;
}

// ============================================================================
// AI User Conversation History Types
// ============================================================================

/**
 * User's conversation summary (list view)
 */
export interface AIUserConversationSummary {
  id: string;
  chatbot: string;
  namespace: string;
  title?: string;
  preview: string;
  message_count: number;
  created_at: string;
  updated_at: string;
}

/**
 * Query result data in a user message
 */
export interface AIUserQueryResult {
  query?: string;
  summary: string;
  row_count: number;
  data?: Record<string, unknown>[];
}

/**
 * Token usage stats in a user message
 */
export interface AIUserUsageStats {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens?: number;
}

/**
 * User's message in conversation detail view
 */
export interface AIUserMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  timestamp: string;
  query_results?: AIUserQueryResult[]; // Array of query results for assistant messages
  usage?: AIUserUsageStats;
}

/**
 * User's conversation detail with messages
 */
export interface AIUserConversationDetail {
  id: string;
  chatbot: string;
  namespace: string;
  title?: string;
  created_at: string;
  updated_at: string;
  messages: AIUserMessage[];
}

/**
 * Options for listing user conversations
 */
export interface ListConversationsOptions {
  /** Filter by chatbot name */
  chatbot?: string;
  /** Filter by namespace */
  namespace?: string;
  /** Number of conversations to return (default: 50, max: 100) */
  limit?: number;
  /** Offset for pagination */
  offset?: number;
}

/**
 * Result of listing user conversations
 */
export interface ListConversationsResult {
  conversations: AIUserConversationSummary[];
  total: number;
  has_more: boolean;
}

/**
 * Options for updating a conversation
 */
export interface UpdateConversationOptions {
  /** New title for the conversation */
  title: string;
}

// ============================================================================
// Knowledge Base Types (RAG)
// ============================================================================

/**
 * Knowledge base summary
 */
export interface KnowledgeBaseSummary {
  id: string;
  name: string;
  namespace: string;
  description: string;
  enabled: boolean;
  document_count: number;
  total_chunks: number;
  embedding_model: string;
  created_at: string;
  updated_at: string;
}

/**
 * Knowledge base full details
 */
export interface KnowledgeBase extends KnowledgeBaseSummary {
  embedding_dimensions: number;
  chunk_size: number;
  chunk_overlap: number;
  chunk_strategy: string;
  source: string;
  created_by?: string;
}

/**
 * Request to create a knowledge base
 */
export interface CreateKnowledgeBaseRequest {
  name: string;
  namespace?: string;
  description?: string;
  embedding_model?: string;
  embedding_dimensions?: number;
  chunk_size?: number;
  chunk_overlap?: number;
  chunk_strategy?: string;
}

/**
 * Request to update a knowledge base
 */
export interface UpdateKnowledgeBaseRequest {
  name?: string;
  description?: string;
  embedding_model?: string;
  embedding_dimensions?: number;
  chunk_size?: number;
  chunk_overlap?: number;
  chunk_strategy?: string;
  enabled?: boolean;
}

/**
 * Document status
 */
export type DocumentStatus = "pending" | "processing" | "indexed" | "failed";

/**
 * Document in a knowledge base
 */
export interface KnowledgeBaseDocument {
  id: string;
  knowledge_base_id: string;
  title: string;
  source_url?: string;
  source_type?: string;
  mime_type: string;
  content_hash: string;
  chunk_count: number;
  status: DocumentStatus;
  error_message?: string;
  metadata?: Record<string, string>;
  tags?: string[];
  created_at: string;
  updated_at: string;
}

/**
 * Request to add a document
 */
export interface AddDocumentRequest {
  title?: string;
  content: string;
  source?: string;
  mime_type?: string;
  metadata?: Record<string, string>;
}

/**
 * Response after adding a document
 */
export interface AddDocumentResponse {
  document_id: string;
  status: string;
  message: string;
}

/**
 * Response after uploading a document file
 */
export interface UploadDocumentResponse {
  document_id: string;
  status: string;
  message: string;
  filename: string;
  extracted_length: number;
  mime_type: string;
}

/**
 * Chatbot-knowledge base link
 */
export interface ChatbotKnowledgeBaseLink {
  id: string;
  chatbot_id: string;
  knowledge_base_id: string;
  enabled: boolean;
  max_chunks: number;
  similarity_threshold: number;
  priority: number;
  created_at: string;
}

/**
 * Request to link a knowledge base to a chatbot
 */
export interface LinkKnowledgeBaseRequest {
  knowledge_base_id: string;
  priority?: number;
  max_chunks?: number;
  similarity_threshold?: number;
}

/**
 * Request to update a chatbot-knowledge base link
 */
export interface UpdateChatbotKnowledgeBaseRequest {
  priority?: number;
  max_chunks?: number;
  similarity_threshold?: number;
  enabled?: boolean;
}

/**
 * Search result from knowledge base
 */
export interface KnowledgeBaseSearchResult {
  chunk_id: string;
  document_id: string;
  document_title: string;
  knowledge_base_name?: string;
  content: string;
  similarity: number;
  metadata?: Record<string, unknown>;
}

/**
 * Request to search a knowledge base
 */
export interface SearchKnowledgeBaseRequest {
  query: string;
  max_chunks?: number;
  threshold?: number;
}

/**
 * Response from knowledge base search
 */
export interface SearchKnowledgeBaseResponse {
  results: KnowledgeBaseSearchResult[];
  count: number;
  query: string;
}

// ============================================================================
// Health Check Types
// ============================================================================

/**
 * System health status response
 */
export interface HealthResponse {
  status: string;
  services: {
    database: boolean;
    realtime: boolean;
  };
  timestamp: string;
}

// ============================================================================
// Admin Storage Types
// ============================================================================

/**
 * Storage bucket information (admin API)
 */
export interface AdminBucket {
  id: string;
  name: string;
  public: boolean;
  allowed_mime_types: string[] | null;
  max_file_size: number | null;
  created_at: string;
  updated_at: string;
}

/**
 * Response from listing buckets (admin API)
 */
export interface AdminListBucketsResponse {
  buckets: AdminBucket[];
}

/**
 * Storage object information (admin API)
 */
export interface AdminStorageObject {
  id: string;
  bucket: string;
  path: string;
  mime_type: string;
  size: number;
  metadata: Record<string, unknown> | null;
  owner_id: string | null;
  created_at: string;
  updated_at: string;
}

/**
 * Response from listing objects (admin API)
 */
export interface AdminListObjectsResponse {
  bucket: string;
  objects: AdminStorageObject[] | null;
  prefixes: string[];
  truncated: boolean;
}

/**
 * Response from generating a signed URL
 */
export interface SignedUrlResponse {
  url: string;
  expires_in: number;
}

/**
 * Request to send an email
 */
export interface SendEmailRequest {
  to: string | string[];
  subject: string;
  html?: string;
  text?: string;
}

// ============================================================================
// Fluxbase Response Types (Supabase-compatible)
// ============================================================================

/**
 * Base Fluxbase response type (Supabase-compatible)
 * Returns either `{ data, error: null }` on success or `{ data: null, error }` on failure
 */
export type FluxbaseResponse<T> =
  | { data: T; error: null }
  | { data: null; error: Error };

/**
 * Response type for operations that don't return data (void operations)
 */
export type VoidResponse = { error: Error | null };

/**
 * Weak password information (Supabase-compatible)
 */
export interface WeakPassword {
  reasons: string[];
}

/**
 * Auth response with user and session (Supabase-compatible)
 */
export type AuthResponseData = {
  user: User;
  session: AuthSession | null;
  weakPassword?: WeakPassword;
};

/**
 * Fluxbase auth response
 */
export type FluxbaseAuthResponse = FluxbaseResponse<AuthResponseData>;

/**
 * User response
 */
export type UserResponse = FluxbaseResponse<{ user: User }>;

/**
 * Session response
 */
export type SessionResponse = FluxbaseResponse<{ session: AuthSession }>;

/**
 * Generic data response
 */
export type DataResponse<T> = FluxbaseResponse<T>;

// ============================================================================
// PostGIS / GeoJSON Types
// ============================================================================

/**
 * GeoJSON Position type (longitude, latitude, optional altitude)
 */
export type GeoJSONPosition = [number, number] | [number, number, number];

/**
 * GeoJSON Point geometry
 */
export interface GeoJSONPoint {
  type: "Point";
  coordinates: GeoJSONPosition;
}

/**
 * GeoJSON LineString geometry
 */
export interface GeoJSONLineString {
  type: "LineString";
  coordinates: GeoJSONPosition[];
}

/**
 * GeoJSON Polygon geometry
 */
export interface GeoJSONPolygon {
  type: "Polygon";
  coordinates: GeoJSONPosition[][];
}

/**
 * GeoJSON MultiPoint geometry
 */
export interface GeoJSONMultiPoint {
  type: "MultiPoint";
  coordinates: GeoJSONPosition[];
}

/**
 * GeoJSON MultiLineString geometry
 */
export interface GeoJSONMultiLineString {
  type: "MultiLineString";
  coordinates: GeoJSONPosition[][];
}

/**
 * GeoJSON MultiPolygon geometry
 */
export interface GeoJSONMultiPolygon {
  type: "MultiPolygon";
  coordinates: GeoJSONPosition[][][];
}

/**
 * GeoJSON GeometryCollection
 */
export interface GeoJSONGeometryCollection {
  type: "GeometryCollection";
  geometries: GeoJSONGeometry[];
}

/**
 * Union of all GeoJSON geometry types
 */
export type GeoJSONGeometry =
  | GeoJSONPoint
  | GeoJSONLineString
  | GeoJSONPolygon
  | GeoJSONMultiPoint
  | GeoJSONMultiLineString
  | GeoJSONMultiPolygon
  | GeoJSONGeometryCollection;

/**
 * GeoJSON Feature with optional properties
 */
export interface GeoJSONFeature<P = Record<string, unknown>> {
  type: "Feature";
  geometry: GeoJSONGeometry;
  properties: P | null;
  id?: string | number;
}

/**
 * GeoJSON FeatureCollection
 */
export interface GeoJSONFeatureCollection<P = Record<string, unknown>> {
  type: "FeatureCollection";
  features: Array<GeoJSONFeature<P>>;
}

// ============================================================================
// RPC (Remote Procedure Call) Types
// ============================================================================

/**
 * RPC procedure summary for listings
 */
export interface RPCProcedureSummary {
  id: string;
  name: string;
  namespace: string;
  description?: string;
  allowed_tables: string[];
  allowed_schemas: string[];
  max_execution_time_seconds: number;
  require_role?: string;
  is_public: boolean;
  enabled: boolean;
  version: number;
  source: string;
  created_at: string;
  updated_at: string;
}

/**
 * Full RPC procedure details
 */
export interface RPCProcedure extends RPCProcedureSummary {
  sql_query: string;
  original_code?: string;
  input_schema?: Record<string, string>;
  output_schema?: Record<string, string>;
  created_by?: string;
}

/**
 * RPC execution status
 */
export type RPCExecutionStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "cancelled"
  | "timeout";

/**
 * RPC execution record
 */
export interface RPCExecution {
  id: string;
  procedure_id?: string;
  procedure_name: string;
  namespace: string;
  status: RPCExecutionStatus;
  input_params?: Record<string, unknown>;
  result?: unknown;
  error_message?: string;
  rows_returned?: number;
  duration_ms?: number;
  user_id?: string;
  user_role?: string;
  user_email?: string;
  is_async: boolean;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

/**
 * RPC invocation response
 */
export interface RPCInvokeResponse<T = unknown> {
  execution_id: string;
  status: RPCExecutionStatus;
  result?: T;
  rows_returned?: number;
  duration_ms?: number;
  error?: string;
}

/**
 * Execution log entry (shared by jobs, RPC, and functions)
 */
export interface ExecutionLog {
  /** Unique log entry ID */
  id: number;
  /** ID of the execution (job ID, RPC execution ID, or function execution ID) */
  execution_id: string;
  /** Line number within the execution log */
  line_number: number;
  /** Log level (debug, info, warn, error) */
  level: string;
  /** Log message content */
  message: string;
  /** Timestamp of the log entry */
  timestamp: string;
  /** Additional structured fields */
  fields?: Record<string, unknown>;
}

/**
 * RPC execution log entry
 * @deprecated Use ExecutionLog instead
 */
export type RPCExecutionLog = ExecutionLog;

/**
 * RPC procedure specification for sync operations
 */
export interface RPCProcedureSpec {
  name: string;
  code: string;
  description?: string;
  enabled?: boolean;
}

/**
 * Options for syncing RPC procedures
 */
export interface SyncRPCOptions {
  namespace?: string;
  procedures?: RPCProcedureSpec[];
  options?: {
    delete_missing?: boolean;
    dry_run?: boolean;
  };
}

/**
 * Result of RPC sync operation
 */
export interface SyncRPCResult {
  message: string;
  namespace: string;
  summary: {
    created: number;
    updated: number;
    deleted: number;
    unchanged: number;
    errors: number;
  };
  details: {
    created: string[];
    updated: string[];
    deleted: string[];
    unchanged: string[];
  };
  errors: Array<{
    procedure: string;
    error: string;
  }>;
  dry_run: boolean;
}

/**
 * Options for updating an RPC procedure
 */
export interface UpdateRPCProcedureRequest {
  description?: string;
  enabled?: boolean;
  is_public?: boolean;
  require_role?: string;
  max_execution_time_seconds?: number;
  allowed_tables?: string[];
  allowed_schemas?: string[];
}

/**
 * Filters for listing RPC executions
 */
export interface RPCExecutionFilters {
  namespace?: string;
  procedure?: string;
  status?: RPCExecutionStatus;
  user_id?: string;
  limit?: number;
  offset?: number;
}

// ============================================================================
// Vector Search Types (pgvector)
// ============================================================================

/**
 * Vector distance metric for similarity search
 * - l2: Euclidean distance (L2 norm) - lower is more similar
 * - cosine: Cosine distance - lower is more similar (1 - cosine similarity)
 * - inner_product: Negative inner product - lower is more similar
 */
export type VectorMetric = "l2" | "cosine" | "inner_product";

/**
 * Options for vector similarity ordering
 */
export interface VectorOrderOptions {
  /** The vector to compare against */
  vector: number[];
  /** Distance metric to use */
  metric?: VectorMetric;
}

/**
 * Request for vector embedding generation
 */
export interface EmbedRequest {
  /** Text to embed (single) */
  text?: string;
  /** Multiple texts to embed */
  texts?: string[];
  /** Embedding model to use (defaults to configured model) */
  model?: string;
}

/**
 * Response from vector embedding generation
 */
export interface EmbedResponse {
  /** Generated embeddings (one per input text) */
  embeddings: number[][];
  /** Model used for embedding */
  model: string;
  /** Dimensions of the embeddings */
  dimensions: number;
  /** Token usage information */
  usage?: {
    prompt_tokens: number;
    total_tokens: number;
  };
}

/**
 * Options for vector search via the convenience endpoint
 */
export interface VectorSearchOptions {
  /** Table to search in */
  table: string;
  /** Vector column to search */
  column: string;
  /** Text query to search for (will be auto-embedded) */
  query?: string;
  /** Direct vector input (alternative to text query) */
  vector?: number[];
  /** Distance metric to use */
  metric?: VectorMetric;
  /** Minimum similarity threshold (0-1 for cosine, varies for others) */
  match_threshold?: number;
  /** Maximum number of results */
  match_count?: number;
  /** Columns to select (default: all) */
  select?: string;
  /** Additional filters to apply */
  filters?: QueryFilter[];
}

/**
 * Result from vector search
 */
export interface VectorSearchResult<T = Record<string, unknown>> {
  /** Matched records */
  data: T[];
  /** Distance scores for each result */
  distances: number[];
  /** Embedding model used (if query text was embedded) */
  model?: string;
}

// ============================================================================
// Execution Log Streaming Types
// ============================================================================

/**
 * Log level for execution logs
 */
export type ExecutionLogLevel = "debug" | "info" | "warn" | "error";

/**
 * Execution type for log subscriptions
 */
export type ExecutionType = "function" | "job" | "rpc";

/**
 * Execution log event received from realtime subscription
 */
export interface ExecutionLogEvent {
  /** Unique execution ID */
  execution_id: string;
  /** Type of execution */
  execution_type: ExecutionType;
  /** Line number in the execution log */
  line_number: number;
  /** Log level */
  level: ExecutionLogLevel;
  /** Log message content */
  message: string;
  /** Timestamp of the log entry */
  timestamp: string;
  /** Additional fields */
  fields?: Record<string, unknown>;
}

/**
 * Callback for execution log events
 */
export type ExecutionLogCallback = (log: ExecutionLogEvent) => void;

/**
 * Configuration for execution log subscription
 */
export interface ExecutionLogConfig {
  /** Execution ID to subscribe to */
  execution_id: string;
  /** Type of execution (function, job, rpc) */
  type?: ExecutionType;
}

// ============================================================================
// Database Branching Types
// ============================================================================

/**
 * Branch status
 */
export type BranchStatus =
  | "creating"
  | "ready"
  | "migrating"
  | "error"
  | "deleting"
  | "deleted";

/**
 * Branch type
 */
export type BranchType = "main" | "preview" | "persistent";

/**
 * Data clone mode when creating a branch
 */
export type DataCloneMode = "schema_only" | "full_clone" | "seed_data";

/**
 * Database branch information
 */
export interface Branch {
  /** Unique branch identifier */
  id: string;
  /** Display name of the branch */
  name: string;
  /** URL-safe slug for the branch */
  slug: string;
  /** Actual database name */
  database_name: string;
  /** Current status of the branch */
  status: BranchStatus;
  /** Type of branch */
  type: BranchType;
  /** Parent branch ID (for feature branches) */
  parent_branch_id?: string;
  /** How data was cloned when branch was created */
  data_clone_mode: DataCloneMode;
  /** GitHub PR number if this is a preview branch */
  github_pr_number?: number;
  /** GitHub PR URL */
  github_pr_url?: string;
  /** GitHub repository (owner/repo) */
  github_repo?: string;
  /** Error message if status is 'error' */
  error_message?: string;
  /** User ID who created the branch */
  created_by?: string;
  /** When the branch was created */
  created_at: string;
  /** When the branch was last updated */
  updated_at: string;
  /** When the branch will automatically expire */
  expires_at?: string;
}

/**
 * Options for creating a new branch
 */
export interface CreateBranchOptions {
  /** Parent branch to clone from (defaults to main) */
  parentBranchId?: string;
  /** How to clone data */
  dataCloneMode?: DataCloneMode;
  /** Branch type */
  type?: BranchType;
  /** GitHub PR number (for preview branches) */
  githubPRNumber?: number;
  /** GitHub PR URL */
  githubPRUrl?: string;
  /** GitHub repository (owner/repo) */
  githubRepo?: string;
  /** Duration until branch expires (e.g., "24h", "7d") */
  expiresIn?: string;
}

/**
 * Options for listing branches
 */
export interface ListBranchesOptions {
  /** Filter by branch status */
  status?: BranchStatus;
  /** Filter by branch type */
  type?: BranchType;
  /** Filter by GitHub repository */
  githubRepo?: string;
  /** Only show branches created by the current user */
  mine?: boolean;
  /** Maximum number of branches to return */
  limit?: number;
  /** Offset for pagination */
  offset?: number;
}

/**
 * Response from listing branches
 */
export interface ListBranchesResponse {
  branches: Branch[];
  total: number;
  limit: number;
  offset: number;
}

/**
 * Branch activity log entry
 */
export interface BranchActivity {
  /** Activity ID */
  id: string;
  /** Branch ID */
  branch_id: string;
  /** Action performed */
  action: string;
  /** Activity status */
  status: "success" | "failed" | "pending";
  /** Additional details */
  details?: Record<string, unknown>;
  /** User who performed the action */
  executed_by?: string;
  /** When the activity occurred */
  created_at: string;
}

/**
 * Connection pool statistics
 */
export interface BranchPoolStats {
  /** Branch slug */
  slug: string;
  /** Number of active connections */
  active_connections: number;
  /** Number of idle connections */
  idle_connections: number;
  /** Total connections created */
  total_connections: number;
  /** When the pool was created */
  created_at: string;
}

// ============================================================================
// Deprecated Supabase-compatible type aliases (for backward compatibility)
// ============================================================================

/**
 * @deprecated Use FluxbaseResponse instead
 */
export type SupabaseResponse<T> = FluxbaseResponse<T>;

/**
 * @deprecated Use FluxbaseAuthResponse instead
 */
export type SupabaseAuthResponse = FluxbaseAuthResponse;
