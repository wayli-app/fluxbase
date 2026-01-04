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
export { SchemaQueryBuilder } from "./schema-query-builder";

// Realtime module
export {
  FluxbaseRealtime,
  RealtimeChannel,
  ExecutionLogsChannel,
} from "./realtime";

// Storage module
export { FluxbaseStorage, StorageBucket } from "./storage";

// Functions module
export { FluxbaseFunctions } from "./functions";

// Jobs module
export { FluxbaseJobs } from "./jobs";

// Admin Functions module
export { FluxbaseAdminFunctions } from "./admin-functions";

// Admin Jobs module
export { FluxbaseAdminJobs } from "./admin-jobs";

// Shared bundling module (for both functions and jobs)
export {
  bundleCode,
  loadImportMap,
  denoExternalPlugin,
  type BundleOptions,
  type BundleResult,
} from "./bundling";

// Admin AI module
export { FluxbaseAdminAI } from "./admin-ai";

// Branching module
export { FluxbaseBranching } from "./branching";

// RPC module
export { FluxbaseRPC } from "./rpc";

// Admin RPC module
export { FluxbaseAdminRPC } from "./admin-rpc";

// AI module
export { FluxbaseAI, FluxbaseAIChat } from "./ai";
export type { AIChatOptions, AIChatEvent, AIChatEventType } from "./ai";

// Vector search module
export { FluxbaseVector } from "./vector";

// GraphQL module
export { FluxbaseGraphQL } from "./graphql";
export type {
  GraphQLResponse,
  GraphQLError,
  GraphQLErrorLocation,
  GraphQLRequestOptions,
  IntrospectionSchema,
  IntrospectionType,
  IntrospectionField,
  IntrospectionInputValue,
  IntrospectionTypeRef,
  IntrospectionEnumValue,
  IntrospectionDirective,
} from "./graphql";

// Admin Migrations module
export { FluxbaseAdminMigrations } from "./admin-migrations";

// Admin Storage module
export { FluxbaseAdminStorage } from "./admin-storage";

// Admin module
export { FluxbaseAdmin } from "./admin";

// Management module
export {
  FluxbaseManagement,
  ClientKeysManager,
  APIKeysManager, // Deprecated alias
  WebhooksManager,
  InvitationsManager,
} from "./management";

// Settings module
export {
  FluxbaseSettings,
  SystemSettingsManager,
  AppSettingsManager,
  EmailTemplateManager,
  EmailSettingsManager,
  SettingsClient,
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

// Type guards
export {
  isFluxbaseError,
  isFluxbaseSuccess,
  isAuthError,
  isAuthSuccess,
  hasPostgrestError,
  isPostgrestSuccess,
  isObject,
  isArray,
  isString,
  isNumber,
  isBoolean,
  assertType,
} from "./type-guards";

// Types
export type {
  // Client options
  FluxbaseClientOptions,

  // Auth types
  AuthSession,
  User,
  SignInCredentials,
  SignUpCredentials,
  UpdateUserAttributes,
  AuthResponse,
  TwoFactorSetupResponse,
  TwoFactorEnableResponse,
  TwoFactorStatusResponse,
  TwoFactorVerifyRequest,
  SignInWith2FAResponse,
  CaptchaConfig,
  CaptchaProvider,

  // SAML SSO types
  SAMLProvider,
  SAMLProvidersResponse,
  SAMLLoginOptions,
  SAMLLoginResponse,
  SAMLSession,

  // Auth configuration types
  AuthConfig,
  OAuthProviderPublic,

  // Database types
  PostgrestResponse,
  PostgrestError,
  FilterOperator,
  QueryFilter,
  OrderBy,
  OrderDirection,
  UpsertOptions,

  // Realtime types
  RealtimeMessage,
  RealtimePostgresChangesPayload,
  RealtimeChangePayload, // Deprecated
  RealtimeCallback,
  PostgresChangesConfig,
  RealtimeChannelConfig,
  PresenceState,
  RealtimePresencePayload,
  PresenceCallback,
  BroadcastMessage,
  RealtimeBroadcastPayload,
  BroadcastCallback,

  // Execution Log types
  ExecutionLog,
  ExecutionLogEvent,
  ExecutionLogCallback,
  ExecutionLogLevel,
  ExecutionType,
  ExecutionLogConfig,

  // Storage types
  FileObject,
  StorageObject, // Deprecated alias for FileObject
  UploadOptions,
  UploadProgress,
  StreamUploadOptions,
  ListOptions,
  SignedUrlOptions,
  DownloadOptions,
  StreamDownloadData,
  ResumableDownloadOptions,
  DownloadProgress,
  ResumableDownloadData,
  ResumableUploadOptions,
  ResumableUploadProgress,
  ChunkedUploadSession,

  // Image Transform types
  TransformOptions,
  ImageFitMode,
  ImageFormat,

  // Functions types
  FunctionInvokeOptions,
  EdgeFunction,
  CreateFunctionRequest,
  UpdateFunctionRequest,
  EdgeFunctionExecution,
  SyncFunctionsOptions,
  SyncFunctionsResult,
  FunctionSpec,
  SyncError,

  // Migrations types
  Migration,
  CreateMigrationRequest,
  UpdateMigrationRequest,
  MigrationExecution,
  ApplyMigrationRequest,
  RollbackMigrationRequest,
  ApplyPendingRequest,
  SyncMigrationsOptions,
  SyncMigrationsResult,

  // Health check types
  HealthResponse,

  // Admin storage types
  AdminBucket,
  AdminListBucketsResponse,
  AdminStorageObject,
  AdminListObjectsResponse,
  SignedUrlResponse,

  // Email types
  SendEmailRequest,

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

  // Management types - Client Keys
  ClientKey,
  CreateClientKeyRequest,
  CreateClientKeyResponse,
  ListClientKeysResponse,
  UpdateClientKeyRequest,
  RevokeClientKeyResponse,
  DeleteClientKeyResponse,

  // Management types - Client Keys (Deprecated aliases)
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

  // Email Provider Settings types (Admin API)
  EmailSettingOverride,
  EmailProviderSettings,
  UpdateEmailProviderSettingsRequest,
  TestEmailSettingsResponse,

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

  // OAuth Logout types
  OAuthLogoutOptions,
  OAuthLogoutResponse,
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

  // AI types
  AIProviderType,
  AIProvider,
  CreateAIProviderRequest,
  UpdateAIProviderRequest,
  AIChatbotSummary,
  AIChatbot,
  ChatbotSpec,
  SyncChatbotsOptions,
  SyncChatbotsResult,
  AIChatMessageRole,
  AIChatClientMessage,
  AIChatServerMessage,
  AIUsageStats,
  AIConversation,
  AIConversationMessage,

  // AI User Conversation History types
  AIUserConversationSummary,
  AIUserConversationDetail,
  AIUserMessage,
  AIUserQueryResult,
  AIUserUsageStats,
  ListConversationsOptions,
  ListConversationsResult,
  UpdateConversationOptions,

  // RPC types
  RPCProcedureSummary,
  RPCProcedure,
  RPCExecutionStatus,
  RPCExecution,
  RPCInvokeResponse,
  RPCExecutionLog,
  RPCProcedureSpec,
  SyncRPCOptions,
  SyncRPCResult,
  UpdateRPCProcedureRequest,
  RPCExecutionFilters,

  // HTTP types
  FluxbaseError,
  HttpMethod,
  RequestOptions,

  // Fluxbase response wrapper types
  FluxbaseResponse,
  FluxbaseAuthResponse,
  AuthResponseData,
  WeakPassword,
  DataResponse,
  VoidResponse,
  UserResponse,
  SessionResponse,

  // Vector search types
  VectorMetric,
  VectorOrderOptions,
  EmbedRequest,
  EmbedResponse,
  VectorSearchOptions,
  VectorSearchResult,

  // Branching types
  BranchStatus,
  BranchType,
  DataCloneMode,
  Branch,
  CreateBranchOptions,
  ListBranchesOptions,
  ListBranchesResponse,
  BranchActivity,
  BranchPoolStats,

  // Deprecated Supabase-compatible aliases
  SupabaseResponse,
  SupabaseAuthResponse,
} from "./types";
