/**
 * Fluxbase TypeScript SDK
 *
 * @example
 * ```typescript
 * import { createClient } from '@nimbleflux/fluxbase-sdk'
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

// Knowledge Base module
export { FluxbaseKnowledgeBase } from "./knowledge-base";

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

// Tenant module
export { FluxbaseTenant } from "./tenant";

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

// Secrets module
export { SecretsManager } from "./secrets";

// DDL module
export { DDLManager } from "./ddl";

// Realtime Admin module
export { FluxbaseAdminRealtime } from "./admin-realtime";
export { ServiceKeysManager } from "./admin-service-keys";

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

  // Settings types - User Settings (non-encrypted, with system fallback)
  UserSetting,
  UserSettingWithSource,
  CreateUserSettingRequest,

  // Email Template types
  EmailTemplateType,
  EmailTemplate,
  UpdateEmailTemplateRequest,
  TestEmailTemplateRequest,
  ListEmailTemplatesResponse,

  // Email Provider Settings types (Admin API)
  EmailSettingOverride,
  EmailProviderSettings,
  TenantEmailProviderSettings,
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

  // Realtime Admin types
  EnableRealtimeRequest,
  EnableRealtimeResponse,
  RealtimeTableStatus,
  ListRealtimeTablesResponse,
  UpdateRealtimeConfigRequest,

  // OAuth Provider Configuration types
  OAuthProvider,
  OAuthProviderInfo,
  CreateOAuthProviderRequest,
  CreateOAuthProviderResponse,
  UpdateOAuthProviderRequest,
  UpdateOAuthProviderResponse,
  DeleteOAuthProviderResponse,
  ListOAuthProvidersResponse,

  // OAuth Logout types
  OAuthLogoutOptions,
  OAuthLogoutResponse,

  // Provider Token types
  ProviderTokenResponse,
  ProviderTokenNotFoundError,
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
  AIChatbotLookupResponse,
  ChatbotSpec,
  // Tool integrations
  IntegrationType,
  IntegrationProvider,
  ToolIntegration,
  CreateToolIntegrationRequest,
  UpdateToolIntegrationRequest,
  TestToolIntegrationResult,
  SyncChatbotsOptions,
  SyncChatbotsResult,
  AIChatMessageRole,
  AIChatClientMessage,
  AIChatServerMessage,
  AIUsageStats,
  AIMatchedIntentRule,
  AIDailyQuotaSnapshot,
  AIQuota,
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

  // Table export types
  TableColumn,
  TableForeignKey,
  TableIndex,
  TableDetails,

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

  // Knowledge Base types
  KnowledgeBaseSummary,
  KnowledgeBase,
  CreateKnowledgeBaseRequest,
  UpdateKnowledgeBaseRequest,
  DocumentStatus,
  KnowledgeBaseDocument,
  AddDocumentRequest,
  AddDocumentResponse,
  UploadDocumentResponse,
  UpdateDocumentRequest,
  DeleteDocumentsByFilterRequest,
  DeleteDocumentsByFilterResponse,
  KnowledgeBaseSearchResult,
  SearchKnowledgeBaseRequest,
  SearchKnowledgeBaseResponse,
  ChatbotKnowledgeBaseLink,
  LinkKnowledgeBaseRequest,
  UpdateChatbotKnowledgeBaseRequest,
  EntityType as AIEntityType,
  Entity as AIEntity,
  EntityRelationship as AIEntityRelationship,
  KnowledgeGraphData,

  // Multi-tenancy types
  Tenant,
  TenantStatus,
  TenantAdminAssignment,
  TenantAdminWithUser,
  TenantWithRole,
  CreateTenantOptions,
  UpdateTenantOptions,
  AssignAdminOptions,

  // Service Key types
  ServiceKey,
  ServiceKeyWithKey,
  CreateServiceKeyRequest,
  UpdateServiceKeyRequest,
  RevokeServiceKeyRequest,
  DeprecateServiceKeyRequest,
} from "./types";

// Secrets types (defined in secrets module)
export type {
  Secret,
  SecretSummary,
  SecretVersion,
  SecretStats,
  CreateSecretRequest,
  UpdateSecretRequest,
  ListSecretsOptions,
  SecretByNameOptions,
} from "./secrets";
