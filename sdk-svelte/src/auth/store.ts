/**
 * Authentication stores for the Fluxbase Svelte SDK.
 *
 * Each function returns TanStack Svelte Query stores (read with `$`) and is
 * designed to be called during component initialization. The functions accept
 * an explicit client + queryClient so they are fully testable without Svelte
 * context; the context-bound wrappers in `index.ts` pass them in for you.
 */

import { createQuery, createMutation } from "@tanstack/svelte-query";
import type { QueryClient } from "@tanstack/svelte-query";
import type {
  FluxbaseClient,
  SignInCredentials,
  SignUpCredentials,
  User,
  AuthSession,
} from "@nimbleflux/fluxbase-sdk";

export interface StoreDeps {
  client: FluxbaseClient;
  queryClient: QueryClient;
}

/**
 * Reactive session store. Reads the cached session without a network call.
 *
 * @example
 * ```svelte
 * <script lang="ts">
 *   import { session } from '@nimbleflux/fluxbase-sdk-svelte'
 *   const $session = session()
 * </script>
 * {#if $session.data}Signed in{/if}
 * ```
 */
export function createSessionStore({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: ["fluxbase", "auth", "session"],
      queryFn: async () => {
        const { data } = await client.auth.getSession();
        return data?.session ?? null;
      },
      staleTime: 1000 * 60 * 5, // 5 minutes
    },
    queryClient,
  );
}

/**
 * Reactive current user store. Calls `getCurrentUser()` (validated server-side) when a
 * session exists.
 */
export function createUserStore({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: ["fluxbase", "auth", "user"],
      queryFn: async () => {
        const { data } = await client.auth.getSession();
        if (!data?.session) return null;
        try {
          const result = await client.auth.getCurrentUser();
          return result.data?.user ?? null;
        } catch {
          return null;
        }
      },
      staleTime: 1000 * 60 * 5,
    },
    queryClient,
  );
}

/** Sign-in mutation. On success it warms the session + user caches. */
export function createSignInMutation({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (credentials: SignInCredentials) => {
        return await client.auth.signIn(credentials);
      },
      onSuccess: (response: any) => {
        const session =
          response?.data?.session ??
          (response?.access_token
            ? (response as AuthSession)
            : null);
        if (session) {
          queryClient.setQueryData(["fluxbase", "auth", "session"], session);
          if ((session as any).user) {
            queryClient.setQueryData(["fluxbase", "auth", "user"], (session as any).user as User);
          }
        }
      },
    },
    queryClient,
  );
}

/** Sign-up mutation. */
export function createSignUpMutation({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (credentials: SignUpCredentials) => {
        return await client.auth.signUp(credentials);
      },
    },
    queryClient,
  );
}

/** Sign-out mutation. Clears session + user caches on success. */
export function createSignOutMutation({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async () => {
        await client.auth.signOut();
      },
      onSuccess: () => {
        queryClient.setQueryData(["fluxbase", "auth", "session"], null);
        queryClient.setQueryData(["fluxbase", "auth", "user"], null);
        queryClient.invalidateQueries({ queryKey: ["fluxbase"] });
      },
    },
    queryClient,
  );
}

/** Update-user mutation. Refreshes the user cache on success. */
export function createUpdateUserMutation({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (attrs: Parameters<FluxbaseClient["auth"]["updateUser"]>[0]) => {
        return await client.auth.updateUser(attrs);
      },
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ["fluxbase", "auth", "user"] });
      },
    },
    queryClient,
  );
}
