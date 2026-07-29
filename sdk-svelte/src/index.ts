/**
 * Fluxbase Svelte SDK
 *
 * Svelte stores and SvelteKit SSR helpers built on TanStack Svelte Query.
 *
 * @example
 * ```svelte
 * <!-- +layout.svelte -->
 * <script lang="ts">
 *   import { setFluxbaseClient } from '@nimbleflux/fluxbase-sdk-svelte'
 *   import { createClient } from '@nimbleflux/fluxbase-sdk'
 *
 *   const client = createClient({ url: 'http://localhost:8080' })
 *   setFluxbaseClient(client)
 * </script>
 * <slot />
 * ```
 *
 * @example
 * ```svelte
 * <!-- +page.svelte -->
 * <script lang="ts">
 *   import { session, table, signIn } from '@nimbleflux/fluxbase-sdk-svelte'
 *
 *   const $session = session()
 *   const products = table('products', (q) => q.eq('active', true), { queryKey: ['products','active'] })
 * </script>
 *
 * {#if !$session.data}
 *   <form on:submit|preventDefault={() => signIn.mutate({ email, password })}> ... </form>
 * {:else}
 *   {#each $products.data ?? [] as p}{p.name}{/each}
 * {/if}
 * ```
 */

// Context + provider
export {
  setFluxbaseClient,
  getClient,
  getQueryClient,
} from "./context";

// SSR cookie-backed storage adapter (uses the core SDK's injectable seam)
export {
  createCookieStorage,
  type CookieStorageOptions,
  type SvelteKitCookies,
} from "./auth/cookie-storage";

// Re-export the StoreDeps type for advanced consumers
export type { StoreDeps } from "./auth/store";

// ---------------------------------------------------------------------------
// Auth stores (context-bound convenience wrappers)
// ---------------------------------------------------------------------------

import { getClient, getQueryClient } from "./context";
import {
  createSessionStore,
  createUserStore,
  createSignInMutation,
  createSignUpMutation,
  createSignOutMutation,
  createUpdateUserMutation,
} from "./auth/store";

const deps = () => ({ client: getClient(), queryClient: getQueryClient() });

/** Reactive auth session. Read with `$session()`. */
export function session() {
  return createSessionStore(deps());
}
/** Reactive current user. Read with `$user()`. */
export function user() {
  return createUserStore(deps());
}
/** Sign-in mutation. */
export function signIn() {
  return createSignInMutation(deps());
}
/** Sign-up mutation. */
export function signUp() {
  return createSignUpMutation(deps());
}
/** Sign-out mutation. */
export function signOut() {
  return createSignOutMutation(deps());
}
/** Update-user mutation. */
export function updateUser() {
  return createUpdateUserMutation(deps());
}

// ---------------------------------------------------------------------------
// Data stores
// ---------------------------------------------------------------------------

import {
  createFluxbaseQuery,
  createTableQuery,
  createInsertMutation,
  createUpdateMutation,
  createUpsertMutation,
  createDeleteMutation,
  type FluxbaseQueryOptions,
} from "./data/store";

/** Reactive Fluxbase query. Read with `$fluxbaseQuery(...)`. */
export function fluxbaseQuery<T = any>(
  buildQuery: Parameters<typeof createFluxbaseQuery<T>>[1],
  options?: FluxbaseQueryOptions<T>,
) {
  return createFluxbaseQuery<T>(deps(), buildQuery, options);
}
/** Reactive table read. Read with `$table(...)`. */
export function table<T = any>(
  tableName: string,
  buildQuery?: Parameters<typeof createTableQuery<T>>[2],
  options?: FluxbaseQueryOptions<T>,
) {
  return createTableQuery<T>(deps(), tableName, buildQuery, options);
}
/** Insert mutation. */
export function insert<T = any>(tableName: string) {
  return createInsertMutation<T>(deps(), tableName);
}
/** Update mutation. */
export function update<T = any>(tableName: string) {
  return createUpdateMutation<T>(deps(), tableName);
}
/** Upsert mutation. */
export function upsert<T = any>(tableName: string) {
  return createUpsertMutation<T>(deps(), tableName);
}
/** Delete mutation. */
export function remove<T = any>(tableName: string) {
  return createDeleteMutation<T>(deps(), tableName);
}
export type { FluxbaseQueryOptions };

// ---------------------------------------------------------------------------
// Realtime stores
// ---------------------------------------------------------------------------

import {
  createRealtimeChannel,
  createTableInserts,
  createTableUpdates,
  createTableDeletes,
} from "./data/realtime";

export function realtimeChannel<T = any>(
  channelName: string,
  event: "INSERT" | "UPDATE" | "DELETE" | "*",
) {
  return createRealtimeChannel<T>(deps(), channelName, event);
}
export function tableInserts<T = any>(tableName: string) {
  return createTableInserts<T>(deps(), tableName);
}
export function tableUpdates<T = any>(tableName: string) {
  return createTableUpdates<T>(deps(), tableName);
}
export function tableDeletes<T = any>(tableName: string) {
  return createTableDeletes<T>(deps(), tableName);
}

// ---------------------------------------------------------------------------
// Storage stores
// ---------------------------------------------------------------------------

import {
  createStorageList,
  createStorageBuckets,
  createStorageUpload,
  createStorageDownload,
  createStorageDelete,
  storagePublicUrl,
} from "./data/storage";

export function storageList(bucket: string, options?: { queryKey?: unknown[] }) {
  return createStorageList(deps(), bucket, options);
}
export function storageBuckets() {
  return createStorageBuckets(deps());
}
export function storageUpload(bucket: string) {
  return createStorageUpload(deps(), bucket);
}
export function storageDownload(bucket: string) {
  return createStorageDownload(deps(), bucket);
}
export function storageRemove(bucket: string) {
  return createStorageDelete(deps(), bucket);
}
export { storagePublicUrl };

// ---------------------------------------------------------------------------
// Feature stores (functions, jobs, branches, rpc, vector, secrets, graphql,
// saml, captcha, auth-config, admin)
// ---------------------------------------------------------------------------

import {
  createInvokeFunction,
  createFunctions,
  createJobs,
  createJobStatus,
  createSubmitJob,
  createCancelJob,
  createBranches,
  createCreateBranch,
  createDeleteBranch,
  createRPCList,
  createInvokeRPC,
  createVectorEmbed,
  createVectorSearch,
  createSecrets,
  createCreateSecret,
  createUpdateSecret,
  createDeleteSecret,
  createGraphQLQuery,
  createGraphQLMutation,
  createSAMLProviders,
  createSignInWithSAML,
  createCaptchaConfig,
  createAuthConfig,
  createUsers,
  createWebhooks,
  createAppSettings,
  createSystemSettings,
} from "./data/features";

// Functions
export function invokeFunction() {
  return createInvokeFunction(deps());
}
export function functions() {
  return createFunctions(deps());
}
// Jobs
export function jobs() {
  return createJobs(deps());
}
export function jobStatus(jobId: string) {
  return createJobStatus(deps(), jobId);
}
export function submitJob() {
  return createSubmitJob(deps());
}
export function cancelJob() {
  return createCancelJob(deps());
}
// Branches
export function branches() {
  return createBranches(deps());
}
export function createBranch() {
  return createCreateBranch(deps());
}
export function deleteBranch() {
  return createDeleteBranch(deps());
}
// RPC
export function rpcList() {
  return createRPCList(deps());
}
export function invokeRPC() {
  return createInvokeRPC(deps());
}
// Vector
export function vectorEmbed() {
  return createVectorEmbed(deps());
}
export function vectorSearch() {
  return createVectorSearch(deps());
}
// Secrets
export function secrets() {
  return createSecrets(deps());
}
export function createSecret() {
  return createCreateSecret(deps());
}
export function updateSecret() {
  return createUpdateSecret(deps());
}
export function deleteSecret() {
  return createDeleteSecret(deps());
}
// GraphQL
export function graphqlQuery(query: string, variables?: Record<string, unknown>) {
  return createGraphQLQuery(deps(), query, variables);
}
export function graphqlMutation() {
  return createGraphQLMutation(deps());
}
// SAML
export function samlProviders() {
  return createSAMLProviders(deps());
}
export function signInWithSAML() {
  return createSignInWithSAML(deps());
}
// Captcha & auth config
export function captchaConfig() {
  return createCaptchaConfig(deps());
}
export function authConfig() {
  return createAuthConfig(deps());
}
// Admin
export function users() {
  return createUsers(deps());
}
export function webhooks() {
  return createWebhooks(deps());
}
export function appSettings() {
  return createAppSettings(deps());
}
export function systemSettings() {
  return createSystemSettings(deps());
}

// ---------------------------------------------------------------------------
// Re-export commonly used types from the core SDK
// ---------------------------------------------------------------------------

export type {
  FluxbaseClient,
  QueryBuilder,
  AuthSession,
  User,
  SignInCredentials,
  SignUpCredentials,
  StorageAdapter,
  FluxbaseResponse,
  // Types referenced by inferred return types of jobs/branches stores
  Job,
  Branch,
  ListBranchesResponse,
} from "@nimbleflux/fluxbase-sdk";
