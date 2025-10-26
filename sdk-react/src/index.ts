/**
 * Fluxbase React Hooks
 *
 * @example
 * ```tsx
 * import { createClient } from '@fluxbase/sdk'
 * import { FluxbaseProvider, useAuth, useTable } from '@fluxbase/sdk-react'
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
export { FluxbaseProvider, useFluxbaseClient } from './context'

// Auth hooks
export {
  useAuth,
  useUser,
  useSession,
  useSignIn,
  useSignUp,
  useSignOut,
  useUpdateUser,
} from './use-auth'

// Database query hooks
export {
  useFluxbaseQuery,
  useTable,
  useInsert,
  useUpdate,
  useUpsert,
  useDelete,
} from './use-query'

// Realtime hooks
export {
  useRealtime,
  useTableSubscription,
  useTableInserts,
  useTableUpdates,
  useTableDeletes,
} from './use-realtime'

// Storage hooks
export {
  useStorageList,
  useStorageUpload,
  useStorageDownload,
  useStorageDelete,
  useStoragePublicUrl,
  useStorageSignedUrl,
  useStorageMove,
  useStorageCopy,
  useStorageBuckets,
  useCreateBucket,
  useDeleteBucket,
} from './use-storage'

// RPC hooks
export {
  useRPC,
  useRPCMutation,
  useRPCBatch,
} from './use-rpc'

// Re-export types from SDK
export type {
  FluxbaseClient,
  AuthSession,
  User,
  SignInCredentials,
  SignUpCredentials,
  PostgrestResponse,
  RealtimeChangePayload,
  StorageObject,
} from '@fluxbase/sdk'
