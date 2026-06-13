/**
 * Fluxbase React Hooks
 *
 * @example
 * ```tsx
 * import { createClient } from '@nimbleflux/fluxbase-sdk'
 * import { FluxbaseProvider, useAuth, useTable } from '@nimbleflux/fluxbase-sdk-react'
 * import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
 *
 * const client = createClient({ url: 'http://localhost:8080' })
 * const queryClient = new QueryClient()
 *
 * function App() {
 *   return (
 *     <QueryClientProvider client={queryClient}>
 *       <FluxbaseProvider client={client}>
 *         <MyComponent />
 *       </FluxbaseProvider>
 *     </QueryClientProvider>
 *   )
 * }
 *
 * function MyComponent() {
 *   const { user, signIn, signOut } = useAuth()
 *   const { data: products } = useTable('products', (q) => q.select('*').eq('active', true))
 *
 *   return <div>...</div>
 * }
 * ```
 */

// Context and provider
export { FluxbaseProvider, useFluxbaseClient } from "./context";

// Auth hooks
export {
  useAuth,
  useUser,
  useSession,
  useSignIn,
  useSignUp,
  useSignOut,
  useUpdateUser,
} from "./use-auth";

// CAPTCHA hooks
export {
  useCaptchaConfig,
  useCaptcha,
  isCaptchaRequiredForEndpoint,
  type CaptchaState,
} from "./use-captcha";

// Auth configuration hooks
export { useAuthConfig } from "./use-auth-config";

// SAML SSO hooks
export {
  useSAMLProviders,
  useGetSAMLLoginUrl,
  useSignInWithSAML,
  useHandleSAMLCallback,
  useSAMLMetadataUrl,
} from "./use-saml";

// GraphQL hooks
export {
  useGraphQLQuery,
  useGraphQLMutation,
  useGraphQLIntrospection,
  useGraphQL,
  type UseGraphQLQueryOptions,
  type UseGraphQLMutationOptions,
} from "./use-graphql";

// Database query hooks
export {
  useFluxbaseQuery,
  useTable,
  useInsert,
  useUpdate,
  useUpsert,
  useDelete,
} from "./use-query";

// Realtime hooks
export {
  useRealtime,
  useTableSubscription,
  useTableInserts,
  useTableUpdates,
  useTableDeletes,
} from "./use-realtime";

// Storage hooks
export {
  useStorageList,
  useStorageUpload,
  useStorageUploadWithProgress,
  useStorageDownload,
  useStorageDelete,
  useStoragePublicUrl,
  useStorageTransformUrl,
  useStorageSignedUrl,
  useStorageSignedUrlWithOptions,
  useStorageBuckets,
  useCreateBucket,
  useDeleteBucket,
} from "./use-storage";

// Admin hooks
export { useAdminAuth } from "./use-admin-auth";
export { useUsers } from "./use-users";
export { useClientKeys, useAPIKeys } from "./use-client-keys";
export {
  useWebhooks,
  useAppSettings,
  useSystemSettings,
} from "./use-admin-hooks";

// Multi-tenancy hooks
export {
  useTenants,
  useTenant,
  type UseTenantsOptions,
  type UseTenantsReturn,
  type UseTenantOptions,
  type UseTenantReturn,
} from "./use-tenant";

// Table export hooks
export {
  useTableDetails,
  type UseTableDetailsOptions,
  type UseTableDetailsReturn,
} from "./use-table-export";

// Function hooks
export { useInvokeFunction } from "./use-functions";

// Job hooks
export { useSubmitJob, useJobStatus } from "./use-jobs";

// Branching hooks
export { useBranches } from "./use-branches";

// RPC hooks
export { useRPCList, useInvokeRPC } from "./use-rpc";

// Vector hooks
export { useVectorEmbed, useVectorSearch } from "./use-vector";

// Secrets hooks
export { useSecrets, useCreateSecret, useUpdateSecret, useDeleteSecret } from "./use-secrets";

// Service Keys hooks
export {
  useServiceKeys,
  useCreateServiceKey,
  useRotateServiceKey,
  useRevokeServiceKey,
} from "./use-service-keys";

// AI hooks
export {
  useChatbots,
  useConversations,
  useConversation,
  useDeleteConversation,
  useAIChat,
  type ChatMessage,
} from "./use-ai";

// Knowledge Base hooks
export {
  useKnowledgeBases,
  useKnowledgeBase,
  useCreateKnowledgeBase,
  useUpdateKnowledgeBase,
  useDeleteKnowledgeBase,
  useKBDocuments,
  useAddDocument,
  useUploadDocument,
  useDeleteDocument,
  useKBSearch,
  useKBEntities,
  useKnowledgeGraph,
} from "./use-knowledge-base";

// Re-export types from SDK
export type {
  FluxbaseClient,
  AuthSession,
  User,
  SignInCredentials,
  SignUpCredentials,
  PostgrestResponse,
  RealtimeChangePayload,
  // @deprecated backward compat aliases
  StorageObject,
  AdminUser,
  EnrichedUser,
  ClientKey,
  APIKey, // Deprecated alias
  Webhook,
  AppSettings,
  SystemSetting,
  CaptchaConfig,
  CaptchaProvider,
  TransformOptions,
  ImageFitMode,
  ImageFormat,
  SignedUrlOptions,
  SAMLProvider,
  SAMLProvidersResponse,
  SAMLLoginOptions,
  SAMLLoginResponse,
  SAMLSession,
  GraphQLResponse,
  GraphQLError,
  GraphQLErrorLocation,
  GraphQLRequestOptions,
  // Multi-tenancy types
  Tenant,
  TenantWithRole,
  CreateTenantOptions,
  UpdateTenantOptions,
} from "@nimbleflux/fluxbase-sdk";
