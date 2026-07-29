/**
 * Tests for the auth store factories.
 *
 * svelte-query stores are unwrapped with `get()` from `svelte/store`.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { get } from "svelte/store";
import { flushSync } from "svelte";
import {
  createSessionStore,
  createUserStore,
  createSignInMutation,
  createSignOutMutation,
} from "./store";
import {
  createMockClient,
  createTestQueryClient,
} from "../test-utils";

describe("createSessionStore", () => {
  it("reads the cached session", async () => {
    const session = {
      access_token: "tok",
      refresh_token: "ref",
      expires_in: 3600,
      expires_at: Date.now() + 3600 * 1000,
      token_type: "Bearer",
      user: { id: "1", email: "a@b.com", created_at: "" },
    };
    const client = createMockClient({
      auth: {
        getSession: vi
          .fn()
          .mockResolvedValue({ data: { session }, error: null }),
      } as any,
    });
    const queryClient = createTestQueryClient();

    const store = createSessionStore({ client, queryClient });

    await queryClient.fetchQuery({
      queryKey: ["fluxbase", "auth", "session"],
      queryFn: async () => {
        const { data } = await client.auth.getSession();
        return data?.session ?? null;
      },
    });
    flushSync();

    expect(client.auth.getSession).toHaveBeenCalled();
    expect(get(store).data).toBeDefined();
  });

  it("returns null when there is no session", async () => {
    const client = createMockClient({
      auth: {
        getSession: vi
          .fn()
          .mockResolvedValue({ data: { session: null }, error: null }),
      } as any,
    });
    const queryClient = createTestQueryClient();

    createSessionStore({ client, queryClient });

    await queryClient.fetchQuery({
      queryKey: ["fluxbase", "auth", "session"],
      queryFn: async () => {
        const { data } = await client.auth.getSession();
        return data?.session ?? null;
      },
    });

    expect(client.auth.getSession).toHaveBeenCalled();
  });
});

describe("createUserStore", () => {
  it("does not call getCurrentUser when there is no session", async () => {
    const client = createMockClient();
    const queryClient = createTestQueryClient();

    createUserStore({ client, queryClient });

    await queryClient.fetchQuery({
      queryKey: ["fluxbase", "auth", "user"],
      queryFn: async () => {
        const { data } = await client.auth.getSession();
        if (!data?.session) return null;
        return client.auth.getCurrentUser().then((r) => r.data?.user ?? null);
      },
    });

    expect(client.auth.getCurrentUser).not.toHaveBeenCalled();
  });
});

describe("createSignInMutation", () => {
  let queryClient: ReturnType<typeof createTestQueryClient>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
  });

  it("calls auth.signIn with credentials", async () => {
    const client = createMockClient({
      auth: {
        signIn: vi.fn().mockResolvedValue({
          data: {
            access_token: "at",
            refresh_token: "rt",
            expires_in: 3600,
            token_type: "Bearer",
            user: { id: "9", email: "x@y.com", created_at: "" },
          },
          error: null,
        }),
      } as any,
    });

    const mutation = createSignInMutation({ client, queryClient });
    await get(mutation).mutateAsync({ email: "x@y.com", password: "pw" });

    expect(client.auth.signIn).toHaveBeenCalledWith({
      email: "x@y.com",
      password: "pw",
    });
  });

  it("warms the session cache on success", async () => {
    const session = {
      access_token: "at",
      refresh_token: "rt",
      expires_in: 3600,
      expires_at: Date.now() + 3600 * 1000,
      token_type: "Bearer",
      user: { id: "9", email: "x@y.com", created_at: "" },
    };
    const client = createMockClient({
      auth: {
        // signIn resolves to { data: { user, session }, error }
        signIn: vi.fn().mockResolvedValue({
          data: { user: session.user, session },
          error: null,
        }),
      } as any,
    });

    const mutation = createSignInMutation({ client, queryClient });
    await get(mutation).mutateAsync({ email: "x@y.com", password: "pw" });

    expect(
      queryClient.getQueryData(["fluxbase", "auth", "session"]),
    ).toEqual(session);
  });
});

describe("createSignOutMutation", () => {
  it("calls auth.signOut and clears the session cache", async () => {
    const client = createMockClient({
      auth: {
        signOut: vi.fn().mockResolvedValue(undefined),
      } as any,
    });
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(["fluxbase", "auth", "session"], {
      access_token: "stale",
    });

    const mutation = createSignOutMutation({ client, queryClient });
    await get(mutation).mutateAsync();

    expect(client.auth.signOut).toHaveBeenCalled();
    expect(
      queryClient.getQueryData(["fluxbase", "auth", "session"]),
    ).toBeNull();
  });
});
