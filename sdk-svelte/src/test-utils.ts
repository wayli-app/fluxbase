/**
 * Test utilities for the Fluxbase Svelte SDK.
 *
 * Mirrors the approach in `sdk-react/src/test-utils.tsx`: build a fully-mocked
 * `FluxbaseClient` and a fresh QueryClient so store functions can be exercised
 * without a live backend.
 */

import { QueryClient } from "@tanstack/svelte-query";
import type { FluxbaseClient } from "@nimbleflux/fluxbase-sdk";
import { vi } from "vitest";

/**
 * Create a mock FluxbaseClient for testing. Any subset of the client can be
 * overridden via the `overrides` parameter (deep-merged one level).
 */
export function createMockClient(
  overrides: Partial<FluxbaseClient> = {},
): FluxbaseClient {
  return {
    auth: {
      getSession: vi.fn().mockResolvedValue({ data: { session: null }, error: null }),
      getUser: vi.fn().mockResolvedValue({ data: { user: null }, error: null }),
      getCurrentUser: vi.fn().mockResolvedValue({ data: { user: null }, error: null }),
      signIn: vi.fn().mockResolvedValue({ data: null, error: null }),
      signUp: vi.fn().mockResolvedValue({ data: null, error: null }),
      signOut: vi.fn().mockResolvedValue(undefined),
      updateUser: vi.fn().mockResolvedValue({ data: null, error: null }),
      onAuthStateChange: vi.fn().mockReturnValue({
        data: { subscription: { unsubscribe: vi.fn() } },
      }),
      getAuthConfig: vi.fn().mockResolvedValue({ data: {}, error: null }),
      getCaptchaConfig: vi.fn().mockResolvedValue({ data: {}, error: null }),
      getSAMLProviders: vi.fn().mockResolvedValue({ data: { providers: [] }, error: null }),
      getSAMLLoginUrl: vi.fn().mockResolvedValue({ data: { url: "" }, error: null }),
      signInWithSAML: vi.fn().mockResolvedValue({ data: null, error: null }),
      handleSAMLCallback: vi.fn().mockResolvedValue({ data: null, error: null }),
      getSAMLMetadataUrl: vi.fn().mockReturnValue("http://localhost/saml/metadata"),
      ...overrides.auth,
    },
    from: vi.fn().mockReturnValue({
      select: vi.fn().mockReturnThis(),
      insert: vi.fn().mockResolvedValue({ data: null, error: null }),
      update: vi.fn().mockResolvedValue({ data: null, error: null }),
      upsert: vi.fn().mockResolvedValue({ data: null, error: null }),
      delete: vi.fn().mockResolvedValue({ data: null, error: null }),
      execute: vi.fn().mockResolvedValue({ data: [], error: null }),
      eq: vi.fn().mockReturnThis(),
    }),
    storage: {
      from: vi.fn().mockReturnValue({
        list: vi.fn().mockResolvedValue({ data: [], error: null }),
        upload: vi.fn().mockResolvedValue({ data: { path: "test.txt" }, error: null }),
        download: vi.fn().mockResolvedValue({ data: new Blob(), error: null }),
        remove: vi.fn().mockResolvedValue({ data: null, error: null }),
        getPublicUrl: vi.fn().mockReturnValue({ data: { publicUrl: "http://localhost/file" } }),
        getTransformUrl: vi.fn().mockReturnValue("http://localhost/transform/file"),
        createSignedUrl: vi.fn().mockResolvedValue({ data: { signedUrl: "http://localhost/signed" }, error: null }),
      }),
      listBuckets: vi.fn().mockResolvedValue({ data: [], error: null }),
      createBucket: vi.fn().mockResolvedValue({ error: null }),
      deleteBucket: vi.fn().mockResolvedValue({ error: null }),
      ...overrides.storage,
    },
    realtime: {
      channel: vi.fn().mockReturnValue({
        on: vi.fn().mockReturnThis(),
        subscribe: vi.fn().mockReturnThis(),
        unsubscribe: vi.fn(),
      }),
      ...overrides.realtime,
    },
    jobs: {
      submit: vi.fn().mockResolvedValue({ data: { id: "job-1", status: "pending" }, error: null }),
      get: vi.fn().mockResolvedValue({ data: { id: "job-1", status: "completed" }, error: null }),
      list: vi.fn().mockResolvedValue({ data: [], error: null }),
      cancel: vi.fn().mockResolvedValue({ data: null, error: null }),
      retry: vi.fn().mockResolvedValue({ data: { id: "job-2", status: "pending" }, error: null }),
      ...overrides.jobs,
    },
    functions: {
      invoke: vi.fn().mockResolvedValue({ data: null, error: null }),
      list: vi.fn().mockResolvedValue({ data: [], error: null }),
      ...overrides.functions,
    },
    branching: {
      list: vi.fn().mockResolvedValue({ data: { branches: [], total: 0, limit: 50, offset: 0 }, error: null }),
      create: vi.fn().mockResolvedValue({ data: { id: "b1", slug: "test", status: "creating" }, error: null }),
      delete: vi.fn().mockResolvedValue({ error: null }),
      reset: vi.fn().mockResolvedValue({ data: null, error: null }),
      ...overrides.branching,
    },
    graphql: {
      execute: vi.fn().mockResolvedValue({ data: null, errors: null }),
      query: vi.fn().mockResolvedValue({ data: null, errors: null }),
      mutation: vi.fn().mockResolvedValue({ data: null, errors: null }),
      introspect: vi.fn().mockResolvedValue({ data: { __schema: {} }, errors: null }),
      ...overrides.graphql,
    },
    rpc: Object.assign(vi.fn().mockResolvedValue({ data: null, error: null }), {
      list: vi.fn().mockResolvedValue({ data: [], error: null }),
      invoke: vi.fn().mockResolvedValue({ data: null, error: null }),
    }),
    vector: {
      embed: vi.fn().mockResolvedValue({ data: null, error: null }),
      search: vi.fn().mockResolvedValue({ data: null, error: null }),
      ...overrides.vector,
    },
    secrets: {
      list: vi.fn().mockResolvedValue([]),
      create: vi.fn().mockResolvedValue({ id: "1", name: "test", scope: "global", version: 1, created_at: "", updated_at: "" }),
      update: vi.fn().mockResolvedValue({ id: "1", name: "test", scope: "global", version: 2, created_at: "", updated_at: "" }),
      delete: vi.fn().mockResolvedValue(undefined),
      ...overrides.secrets,
    },
    admin: {
      me: vi.fn().mockResolvedValue({ data: null, error: null }),
      login: vi.fn().mockResolvedValue({ data: null, error: null }),
      listUsers: vi.fn().mockResolvedValue({ data: { users: [], total: 0 }, error: null }),
      ...overrides.admin,
    },
    getTenantId: vi.fn().mockReturnValue(null),
    ...overrides,
  } as unknown as FluxbaseClient;
}

/**
 * Create a fresh QueryClient for testing (no retries, no GC delay) so tests
 * are deterministic.
 */
export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
}
