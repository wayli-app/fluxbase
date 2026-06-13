/**
 * Test utilities for Fluxbase React SDK
 */

import React, { ReactElement } from "react";
import { render, RenderOptions, RenderResult } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FluxbaseProvider } from "./context";
import type { FluxbaseClient } from "@nimbleflux/fluxbase-sdk";
import { vi } from "vitest";

/**
 * Create a mock FluxbaseClient for testing
 */
export function createMockClient(
  overrides: Partial<FluxbaseClient> = {},
): FluxbaseClient {
  return {
    auth: {
      getSession: vi.fn().mockResolvedValue({ data: null, error: null }),
      getCurrentUser: vi.fn().mockResolvedValue({ data: null, error: null }),
      signIn: vi.fn().mockResolvedValue({ user: null, session: null }),
      signUp: vi.fn().mockResolvedValue({ data: null, error: null }),
      signOut: vi.fn().mockResolvedValue(undefined),
      updateUser: vi
        .fn()
        .mockResolvedValue({ id: "1", email: "test@example.com" }),
      getAuthConfig: vi.fn().mockResolvedValue({ data: {}, error: null }),
      getCaptchaConfig: vi.fn().mockResolvedValue({ data: {}, error: null }),
      getSAMLProviders: vi
        .fn()
        .mockResolvedValue({ data: { providers: [] }, error: null }),
      getSAMLLoginUrl: vi
        .fn()
        .mockResolvedValue({ data: { url: "" }, error: null }),
      signInWithSAML: vi.fn().mockResolvedValue({ data: null, error: null }),
      handleSAMLCallback: vi
        .fn()
        .mockResolvedValue({ data: null, error: null }),
      getSAMLMetadataUrl: vi
        .fn()
        .mockReturnValue("http://localhost/saml/metadata"),
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
        upload: vi
          .fn()
          .mockResolvedValue({ data: { path: "test.txt" }, error: null }),
        download: vi.fn().mockResolvedValue({ data: new Blob(), error: null }),
        remove: vi.fn().mockResolvedValue({ data: null, error: null }),
        getPublicUrl: vi
          .fn()
          .mockReturnValue({ data: { publicUrl: "http://localhost/file" } }),
        getTransformUrl: vi
          .fn()
          .mockReturnValue("http://localhost/transform/file"),
        createSignedUrl: vi
          .fn()
          .mockResolvedValue({
            data: { signedUrl: "http://localhost/signed" },
            error: null,
          }),
        move: vi
          .fn()
          .mockResolvedValue({ data: { path: "new.txt" }, error: null }),
        copy: vi
          .fn()
          .mockResolvedValue({ data: { path: "copy.txt" }, error: null }),
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
      getLogs: vi.fn().mockResolvedValue({ data: [], error: null }),
      ...overrides.jobs,
    },
    functions: {
      invoke: vi.fn().mockResolvedValue({ data: null, error: null }),
      list: vi.fn().mockResolvedValue({ data: [], error: null }),
      get: vi.fn().mockResolvedValue({ data: null, error: null }),
      ...overrides.functions,
    },
    branching: {
      list: vi.fn().mockResolvedValue({ data: { branches: [], total: 0, limit: 50, offset: 0 }, error: null }),
      get: vi.fn().mockResolvedValue({ data: null, error: null }),
      create: vi.fn().mockResolvedValue({ data: { id: "b1", slug: "test-branch", status: "creating" }, error: null }),
      delete: vi.fn().mockResolvedValue({ error: null }),
      reset: vi.fn().mockResolvedValue({ data: { id: "b1", slug: "test-branch", status: "creating" }, error: null }),
      getActivity: vi.fn().mockResolvedValue({ data: [], error: null }),
      getPoolStats: vi.fn().mockResolvedValue({ data: [], error: null }),
      exists: vi.fn().mockResolvedValue(false),
      waitForReady: vi.fn().mockResolvedValue({ data: null, error: null }),
      ...overrides.branching,
    },
    graphql: {
      execute: vi.fn().mockResolvedValue({ data: null, errors: null }),
      query: vi.fn().mockResolvedValue({ data: null, errors: null }),
      mutation: vi.fn().mockResolvedValue({ data: null, errors: null }),
      introspect: vi
        .fn()
        .mockResolvedValue({ data: { __schema: {} }, errors: null }),
      ...overrides.graphql,
    },
    admin: {
      me: vi.fn().mockResolvedValue({ data: null, error: null }),
      login: vi.fn().mockResolvedValue({ data: null, error: null }),
      listUsers: vi
        .fn()
        .mockResolvedValue({ data: { users: [], total: 0 }, error: null }),
      inviteUser: vi.fn().mockResolvedValue({ data: null, error: null }),
      updateUserRole: vi.fn().mockResolvedValue({ data: null, error: null }),
      deleteUser: vi.fn().mockResolvedValue({ data: null, error: null }),
      resetUserPassword: vi
        .fn()
        .mockResolvedValue({
          data: { message: "Password reset" },
          error: null,
        }),
      settings: {
        app: {
          get: vi.fn().mockResolvedValue({}),
          update: vi.fn().mockResolvedValue({}),
        },
        system: {
          list: vi.fn().mockResolvedValue({ settings: [] }),
          update: vi.fn().mockResolvedValue({}),
          delete: vi.fn().mockResolvedValue({}),
        },
      },
      management: {
        clientKeys: {
          list: vi.fn().mockResolvedValue({ client_keys: [] }),
          create: vi.fn().mockResolvedValue({ key: "new-key", client_key: {} }),
          update: vi.fn().mockResolvedValue({}),
          revoke: vi.fn().mockResolvedValue({}),
          delete: vi.fn().mockResolvedValue({}),
        },
        webhooks: {
          list: vi.fn().mockResolvedValue({ webhooks: [] }),
          create: vi.fn().mockResolvedValue({}),
          update: vi.fn().mockResolvedValue({}),
          delete: vi.fn().mockResolvedValue({}),
          test: vi.fn().mockResolvedValue({}),
        },
      },
      serviceKeys: {
        list: vi.fn().mockResolvedValue({ data: [], error: null }),
        get: vi.fn().mockResolvedValue({ data: null, error: null }),
        create: vi.fn().mockResolvedValue({ data: null, error: null }),
        update: vi.fn().mockResolvedValue({ data: null, error: null }),
        delete: vi.fn().mockResolvedValue({ error: null }),
        disable: vi.fn().mockResolvedValue({ error: null }),
        enable: vi.fn().mockResolvedValue({ error: null }),
        revoke: vi.fn().mockResolvedValue({ error: null }),
        deprecate: vi.fn().mockResolvedValue({ data: null, error: null }),
        rotate: vi.fn().mockResolvedValue({ data: null, error: null }),
        getRevocationHistory: vi.fn().mockResolvedValue({ data: null, error: null }),
      },
      migrations: {
        list: vi.fn().mockResolvedValue({ data: [], error: null }),
        apply: vi
          .fn()
          .mockResolvedValue({ data: { message: "Migration applied" }, error: null }),
        rollback: vi
          .fn()
          .mockResolvedValue({ data: { message: "Migration rolled back" }, error: null }),
        sync: vi.fn().mockResolvedValue({ data: null, error: null }),
      },
      impersonation: {
        impersonateUser: vi
          .fn()
          .mockResolvedValue({
            session: null,
            target_user: null,
            access_token: "",
            refresh_token: "",
            expires_in: 0,
          }),
        impersonateAnon: vi
          .fn()
          .mockResolvedValue({
            session: null,
            target_user: null,
            access_token: "",
            refresh_token: "",
            expires_in: 0,
          }),
        stop: vi
          .fn()
          .mockResolvedValue({ success: true, message: "Impersonation stopped" }),
        getCurrent: vi.fn().mockResolvedValue({ session: null, target_user: null }),
        listSessions: vi.fn().mockResolvedValue({ sessions: [], total: 0 }),
      },
      ddl: {
        listSchemas: vi.fn().mockResolvedValue({ schemas: [] }),
        createSchema: vi
          .fn()
          .mockResolvedValue({ message: "Schema created", schema: "" }),
        listTables: vi.fn().mockResolvedValue({ tables: [] }),
        deleteTable: vi
          .fn()
          .mockResolvedValue({ message: "Table deleted" }),
      },
      ...overrides.admin,
    },
    rpc: Object.assign(
      vi.fn().mockResolvedValue({ data: null, error: null }),
      {
        list: vi.fn().mockResolvedValue({ data: [], error: null }),
        invoke: vi.fn().mockResolvedValue({ data: null, error: null }),
        getStatus: vi.fn().mockResolvedValue({ data: null, error: null }),
        getLogs: vi.fn().mockResolvedValue({ data: [], error: null }),
        waitForCompletion: vi.fn().mockResolvedValue({ data: null, error: null }),
      },
    ),
    secrets: {
      list: vi.fn().mockResolvedValue([]),
      create: vi.fn().mockResolvedValue({ id: "1", name: "test", scope: "global", version: 1, created_at: "", updated_at: "" }),
      get: vi.fn().mockResolvedValue({ id: "1", name: "test", scope: "global", version: 1, created_at: "", updated_at: "" }),
      update: vi.fn().mockResolvedValue({ id: "1", name: "test", scope: "global", version: 2, created_at: "", updated_at: "" }),
      delete: vi.fn().mockResolvedValue(undefined),
      getVersions: vi.fn().mockResolvedValue([]),
      rollback: vi.fn().mockResolvedValue({ id: "1", name: "test", scope: "global", version: 3, created_at: "", updated_at: "" }),
      stats: vi.fn().mockResolvedValue({ total: 0, expiring_soon: 0, expired: 0 }),
      getById: vi.fn().mockResolvedValue({ id: "1", name: "test", scope: "global", version: 1, created_at: "", updated_at: "" }),
      updateById: vi.fn().mockResolvedValue({ id: "1", name: "test", scope: "global", version: 2, created_at: "", updated_at: "" }),
      deleteById: vi.fn().mockResolvedValue(undefined),
      getVersionsById: vi.fn().mockResolvedValue([]),
      rollbackById: vi.fn().mockResolvedValue({ id: "1", name: "test", scope: "global", version: 3, created_at: "", updated_at: "" }),
      ...overrides.secrets,
    },
    vector: {
      embed: vi.fn().mockResolvedValue({ data: null, error: null }),
      search: vi.fn().mockResolvedValue({ data: null, error: null }),
      ...overrides.vector,
    },
    ...overrides,
  } as unknown as FluxbaseClient;
}

/**
 * Create a mock client with AI/Knowledge Base support
 */
export function createMockAIClient(
  overrides: Partial<FluxbaseClient> = {},
): FluxbaseClient {
  const base = createMockClient(overrides);
  return {
    ...base,
    ai: {
      listChatbots: vi.fn().mockResolvedValue({ data: [], error: null }),
      getChatbot: vi.fn().mockResolvedValue({ data: null, error: null }),
      lookupChatbot: vi.fn().mockResolvedValue({ data: null, error: null }),
      createChat: vi.fn().mockReturnValue({
        connect: vi.fn().mockResolvedValue(undefined),
        disconnect: vi.fn(),
        isConnected: vi.fn().mockReturnValue(true),
        startChat: vi.fn().mockResolvedValue("conv-1"),
        sendMessage: vi.fn().mockResolvedValue(undefined),
        cancel: vi.fn(),
        getAccumulatedContent: vi.fn().mockReturnValue(""),
      }),
      listConversations: vi
        .fn()
        .mockResolvedValue({ data: { conversations: [], total: 0, has_more: false }, error: null }),
      getConversation: vi.fn().mockResolvedValue({ data: null, error: null }),
      deleteConversation: vi.fn().mockResolvedValue({ error: null }),
      updateConversation: vi.fn().mockResolvedValue({ data: null, error: null }),
      ...overrides.ai,
    },
    knowledgeBase: {
      list: vi.fn().mockResolvedValue({ data: [], error: null }),
      get: vi.fn().mockResolvedValue({ data: null, error: null }),
      create: vi.fn().mockResolvedValue({ data: null, error: null }),
      update: vi.fn().mockResolvedValue({ data: null, error: null }),
      delete: vi.fn().mockResolvedValue({ data: true, error: null }),
      listDocuments: vi.fn().mockResolvedValue({ data: [], error: null }),
      getDocument: vi.fn().mockResolvedValue({ data: null, error: null }),
      addDocument: vi.fn().mockResolvedValue({ data: null, error: null }),
      uploadDocument: vi.fn().mockResolvedValue({ data: null, error: null }),
      updateDocument: vi.fn().mockResolvedValue({ data: null, error: null }),
      deleteDocument: vi.fn().mockResolvedValue({ data: true, error: null }),
      deleteDocumentsByFilter: vi.fn().mockResolvedValue({ data: null, error: null }),
      search: vi.fn().mockResolvedValue({ data: null, error: null }),
      listEntities: vi.fn().mockResolvedValue({ data: [], error: null }),
      searchEntities: vi.fn().mockResolvedValue({ data: [], error: null }),
      getEntityRelationships: vi.fn().mockResolvedValue({ data: [], error: null }),
      getKnowledgeGraph: vi.fn().mockResolvedValue({ data: null, error: null }),
      ...overrides.knowledgeBase,
    },
    vector: {
      embed: vi.fn().mockResolvedValue({ data: null, error: null }),
      search: vi.fn().mockResolvedValue({ data: null, error: null }),
      ...overrides.vector,
    },
  } as unknown as FluxbaseClient;
}

/**
 * Create a fresh QueryClient for testing
 */
export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

interface WrapperProps {
  children: React.ReactNode;
}

/**
 * Create a wrapper component with all providers
 */
export function createWrapper(
  client: FluxbaseClient,
  queryClient?: QueryClient,
) {
  const qc = queryClient || createTestQueryClient();

  return function Wrapper({ children }: WrapperProps) {
    return (
      <QueryClientProvider client={qc}>
        <FluxbaseProvider client={client}>{children}</FluxbaseProvider>
      </QueryClientProvider>
    );
  };
}

/**
 * Custom render function that includes all providers
 */
export function renderWithProviders(
  ui: ReactElement,
  options?: Omit<RenderOptions, "wrapper"> & {
    client?: FluxbaseClient;
    queryClient?: QueryClient;
  },
): RenderResult & { client: FluxbaseClient; queryClient: QueryClient } {
  const {
    client = createMockClient(),
    queryClient,
    ...renderOptions
  } = options || {};
  const wrapper = createWrapper(client, queryClient);

  return {
    ...render(ui, { wrapper, ...renderOptions }),
    client,
    queryClient: queryClient || createTestQueryClient(),
  };
}

// Re-export testing library utilities
export * from "@testing-library/react";
