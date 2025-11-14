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
export { FluxbaseClient, createClient } from "./client";

// Auth module
export { FluxbaseAuth } from "./auth";

// Database query builder
export { QueryBuilder } from "./query-builder";

// Realtime module
export { FluxbaseRealtime, RealtimeChannel } from "./realtime";

// Storage module
export { FluxbaseStorage, StorageBucket } from "./storage";

// Functions module
export { FluxbaseFunctions } from "./functions";

// Admin module
export { FluxbaseAdmin } from "./admin";

// Management module
export {
  FluxbaseManagement,
  APIKeysManager,
  WebhooksManager,
  InvitationsManager,
} from "./management";

// Settings module
export {
  FluxbaseSettings,
  SystemSettingsManager,
  AppSettingsManager,
  EmailTemplateManager,
} from "./settings";

// DDL module
export { DDLManager } from "./ddl";

// OAuth configuration module
export {
  FluxbaseOAuth,
  OAuthProviderManager,
  AuthSettingsManager,
} from "./oauth";

// Impersonation module
export { ImpersonationManager } from "./impersonation";

// HTTP client (advanced users)
export { FluxbaseFetch } from "./fetch";

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
  RealtimePostgresChangesPayload,
  RealtimeChangePayload, // Deprecated
  RealtimeCallback,
  PostgresChangesConfig,

  // Storage types
  FileObject,
  StorageObject, // Deprecated alias for FileObject
  UploadOptions,
  ListOptions,
  SignedUrlOptions,

  // Functions types
  FunctionInvokeOptions,
  EdgeFunction,
  CreateFunctionRequest,
  UpdateFunctionRequest,
  EdgeFunctionExecution,

  // Admin types
  AdminSetupStatusResponse,
  AdminSetupRequest,
  AdminUser,
  AdminAuthResponse,
  AdminLoginRequest,
  AdminRefreshRequest,
  AdminRefreshResponse,
  AdminMeResponse,
  EnrichedUser,
  ListUsersResponse,
  ListUsersOptions,
  InviteUserRequest,
  InviteUserResponse,
  UpdateUserRoleRequest,
  ResetUserPasswordResponse,
  DeleteUserResponse,

  // Management types - API Keys
  APIKey,
  CreateAPIKeyRequest,
  CreateAPIKeyResponse,
  ListAPIKeysResponse,
  UpdateAPIKeyRequest,
  RevokeAPIKeyResponse,
  DeleteAPIKeyResponse,

  // Management types - Webhooks
  Webhook,
  CreateWebhookRequest,
  UpdateWebhookRequest,
  ListWebhooksResponse,
  TestWebhookResponse,
  WebhookDelivery,
  ListWebhookDeliveriesResponse,
  DeleteWebhookResponse,

  // Management types - Invitations
  Invitation,
  CreateInvitationRequest,
  CreateInvitationResponse,
  ValidateInvitationResponse,
  AcceptInvitationRequest,
  AcceptInvitationResponse,
  ListInvitationsOptions,
  ListInvitationsResponse,
  RevokeInvitationResponse,

  // Settings types - System Settings
  SystemSetting,
  UpdateSystemSettingRequest,
  ListSystemSettingsResponse,

  // Settings types - App Settings
  AuthenticationSettings,
  FeatureSettings,
  EmailSettings,
  SMTPSettings,
  SendGridSettings,
  MailgunSettings,
  SESSettings,
  SecuritySettings,
  AppSettings,
  UpdateAppSettingsRequest,

  // Email Template types
  EmailTemplateType,
  EmailTemplate,
  UpdateEmailTemplateRequest,
  TestEmailTemplateRequest,
  ListEmailTemplatesResponse,

  // DDL types
  CreateColumnRequest,
  CreateSchemaRequest,
  CreateSchemaResponse,
  CreateTableRequest,
  CreateTableResponse,
  DeleteTableResponse,
  Schema,
  ListSchemasResponse,
  Column,
  Table,
  ListTablesResponse,

  // OAuth Provider Configuration types
  OAuthProvider,
  CreateOAuthProviderRequest,
  CreateOAuthProviderResponse,
  UpdateOAuthProviderRequest,
  UpdateOAuthProviderResponse,
  DeleteOAuthProviderResponse,
  ListOAuthProvidersResponse,
  AuthSettings,
  UpdateAuthSettingsRequest,
  UpdateAuthSettingsResponse,

  // Impersonation types
  ImpersonationType,
  ImpersonationTargetUser,
  ImpersonationSession,
  ImpersonateUserRequest,
  ImpersonateAnonRequest,
  ImpersonateServiceRequest,
  StartImpersonationResponse,
  StopImpersonationResponse,
  GetImpersonationResponse,
  ListImpersonationSessionsOptions,
  ListImpersonationSessionsResponse,

  // HTTP types
  FluxbaseError,
  HttpMethod,
  RequestOptions,

  // Supabase-compatible response wrapper types
  DataResponse,
  VoidResponse,
  SupabaseAuthResponse,
  UserResponse,
  SessionResponse,
  SupabaseResponse,
} from "./types";
