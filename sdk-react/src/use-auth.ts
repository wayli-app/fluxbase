/**
 * Authentication hooks for Fluxbase SDK
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type {
  SignInCredentials,
  SignUpCredentials,
  User,
  AuthSession,
} from "@fluxbase/sdk";

/**
 * Hook to get the current user
 */
export function useUser() {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "auth", "user"],
    queryFn: async () => {
      const { data } = await client.auth.getSession();
      if (!data.session) {
        return null;
      }

      try {
        const result = await client.auth.getCurrentUser();
        return result.data?.user ?? null;
      } catch {
        return null;
      }
    },
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook to get the current session
 */
export function useSession() {
  const client = useFluxbaseClient();

  return useQuery<AuthSession | null>({
    queryKey: ["fluxbase", "auth", "session"],
    queryFn: async () => {
      const { data } = await client.auth.getSession();
      return data.session;
    },
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

/**
 * Hook for signing in
 */
export function useSignIn() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (credentials: SignInCredentials) => {
      return await client.auth.signIn(credentials);
    },
    onSuccess: (session) => {
      queryClient.setQueryData(["fluxbase", "auth", "session"], session);
      // Only set user if this is a complete auth session (not 2FA required)
      if ("user" in session) {
        queryClient.setQueryData(["fluxbase", "auth", "user"], session.user);
      }
    },
  });
}

/**
 * Hook for signing up
 */
export function useSignUp() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (credentials: SignUpCredentials) => {
      return await client.auth.signUp(credentials);
    },
    onSuccess: (response) => {
      if (response.data) {
        queryClient.setQueryData(
          ["fluxbase", "auth", "session"],
          response.data.session,
        );
        queryClient.setQueryData(
          ["fluxbase", "auth", "user"],
          response.data.user,
        );
      }
    },
  });
}

/**
 * Hook for signing out
 */
export function useSignOut() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      await client.auth.signOut();
    },
    onSuccess: () => {
      queryClient.setQueryData(["fluxbase", "auth", "session"], null);
      queryClient.setQueryData(["fluxbase", "auth", "user"], null);
      queryClient.invalidateQueries({ queryKey: ["fluxbase"] });
    },
  });
}

/**
 * Hook for updating the current user
 */
export function useUpdateUser() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: Partial<Pick<User, "email" | "metadata">>) => {
      return await client.auth.updateUser(data);
    },
    onSuccess: (user) => {
      queryClient.setQueryData(["fluxbase", "auth", "user"], user);
    },
  });
}

/**
 * Combined auth hook with all auth state and methods
 */
export function useAuth() {
  const { data: user, isLoading: isLoadingUser } = useUser();
  const { data: session, isLoading: isLoadingSession } = useSession();
  const signIn = useSignIn();
  const signUp = useSignUp();
  const signOut = useSignOut();
  const updateUser = useUpdateUser();

  return {
    user,
    session,
    isLoading: isLoadingUser || isLoadingSession,
    isAuthenticated: !!session,
    signIn: signIn.mutateAsync,
    signUp: signUp.mutateAsync,
    signOut: signOut.mutateAsync,
    updateUser: updateUser.mutateAsync,
    isSigningIn: signIn.isPending,
    isSigningUp: signUp.isPending,
    isSigningOut: signOut.isPending,
    isUpdating: updateUser.isPending,
  };
}
