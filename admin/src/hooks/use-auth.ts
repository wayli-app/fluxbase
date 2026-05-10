import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import type { SignUpCredentials } from "@nimbleflux/fluxbase-sdk";
import { useFluxbaseClient } from "@nimbleflux/fluxbase-sdk-react";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/get-error-message";
import { useAuthStore } from "@/stores/auth-store";
import { useImpersonationStore } from "@/stores/impersonation-store";
import { dashboardAuthAPI, type DashboardLoginRequest } from "@/lib/api";
import { syncAuthToken } from "@/lib/fluxbase-client";

export function useAuth() {
  const { auth } = useAuthStore();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const client = useFluxbaseClient();

  // Fetch current user data from dashboard auth endpoint
  const { data: dashboardUser, isLoading: isLoadingUser } = useQuery({
    queryKey: ["auth", "user"],
    queryFn: async () => {
      return await dashboardAuthAPI.me();
    },
    enabled: !!auth.accessToken,
    retry: false,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });

  // Use dashboard user (with role) or fall back to Zustand store
  const user = dashboardUser || auth.user;

  // Sign in mutation - uses dashboard authentication (platform.users)
  const signInMutation = useMutation({
    mutationFn: async (data: DashboardLoginRequest) => {
      return await dashboardAuthAPI.login(data);
    },
    onSuccess: (response) => {
      // Clear any stale impersonation state from previous session
      useImpersonationStore.getState().stopImpersonation();

      // Check if 2FA is required
      if (response.requires_2fa) {
        // Handle 2FA flow - don't store tokens yet
        toast.info("Two-factor authentication required");
        // Navigate to 2FA verification page with user_id
        navigate({ to: "/login/otp" });
        return;
      }

      const { access_token, refresh_token, expires_in, user } = response;

      auth.setTokens(access_token, refresh_token);

      auth.setUser({
        accountNo: user.id,
        email: user.email,
        role: [user.role || "tenant_admin"],
        exp: Date.now() + expires_in * 1000,
      });

      syncAuthToken();

      // Invalidate and refetch user query to get full user data with role
      queryClient.invalidateQueries({ queryKey: ["auth", "user"] });

      toast.success(`Welcome back, ${user.email}!`);
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Sign up mutation
  const signUpMutation = useMutation({
    mutationFn: async (data: SignUpCredentials) => {
      return await client.auth.signUp(data);
    },
    onSuccess: (response) => {
      if (!response.data) {
        toast.error("Invalid response from server");
        return;
      }

      const { session, user } = response.data;

      if (!session) {
        toast.error("No session returned from server");
        return;
      }

      auth.setTokens(session.access_token, session.refresh_token);

      auth.setUser({
        accountNo: user.id,
        email: user.email,
        role: [user.role],
        exp: Date.now() + session.expires_in * 1000,
      });

      // Invalidate and refetch user query
      queryClient.invalidateQueries({ queryKey: ["auth", "user"] });

      toast.success(`Account created successfully! Welcome, ${user.email}!`);
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to create account");
    },
  });

  // Sign out mutation
  const signOutMutation = useMutation({
    mutationFn: async () => {
      await client.auth.signOut();
    },
    onSuccess: () => {
      auth.reset();
      queryClient.clear();
      navigate({ to: "/login", replace: true });
      toast.success("Signed out successfully");
    },
    onError: (error: Error) => {
      auth.reset();
      queryClient.clear();
      navigate({ to: "/login", replace: true });
      toast.error(error.message || "Failed to sign out");
    },
  });

  return {
    user,
    isAuthenticated: !!auth.accessToken,
    isLoading: isLoadingUser,
    signIn: signInMutation.mutate,
    signUp: signUpMutation.mutate,
    signOut: signOutMutation.mutate,
    isSigningIn: signInMutation.isPending,
    isSigningUp: signUpMutation.isPending,
    isSigningOut: signOutMutation.isPending,
  };
}
