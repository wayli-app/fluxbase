/**
 * Authentication module for Fluxbase SDK
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  AuthResponse,
  AuthResponseData,
  AuthSession,
  SignInCredentials,
  SignUpCredentials,
  User,
  TwoFactorSetupResponse,
  TwoFactorEnableResponse,
  TwoFactorStatusResponse,
  TwoFactorDisableResponse,
  TwoFactorVerifyRequest,
  SignInWith2FAResponse,
  PasswordResetResponse,
  VerifyResetTokenResponse,
  ResetPasswordResponse,
  MagicLinkOptions,
  MagicLinkResponse,
  AnonymousSignInResponse,
  OAuthProvidersResponse,
  OAuthOptions,
  OAuthUrlResponse,
  AuthChangeEvent,
  AuthStateChangeCallback,
  AuthSubscription,
  FluxbaseAuthResponse,
  FluxbaseResponse,
  UserResponse,
  DataResponse,
  VoidResponse,
} from "./types";
import { wrapAsync, wrapAsyncVoid } from "./utils/error-handling";

const AUTH_STORAGE_KEY = "fluxbase.auth.session";

export class FluxbaseAuth {
  private fetch: FluxbaseFetch;
  private session: AuthSession | null = null;
  private persist: boolean;
  private autoRefresh: boolean;
  private refreshTimer: ReturnType<typeof setTimeout> | null = null;
  private stateChangeListeners: Set<AuthStateChangeCallback> = new Set();

  constructor(fetch: FluxbaseFetch, autoRefresh = true, persist = true) {
    this.fetch = fetch;
    this.persist = persist;
    this.autoRefresh = autoRefresh;

    // Load session from storage if persisted
    if (this.persist && typeof localStorage !== "undefined") {
      const stored = localStorage.getItem(AUTH_STORAGE_KEY);
      if (stored) {
        try {
          this.session = JSON.parse(stored);
          if (this.session) {
            this.fetch.setAuthToken(this.session.access_token);
            this.scheduleTokenRefresh();
          }
        } catch {
          // Invalid stored session, ignore
          localStorage.removeItem(AUTH_STORAGE_KEY);
        }
      }
    }
  }

  /**
   * Get the current session (Supabase-compatible)
   * Returns the session from the client-side cache without making a network request
   */
  async getSession(): Promise<
    FluxbaseResponse<{ session: AuthSession | null }>
  > {
    return { data: { session: this.session }, error: null };
  }

  /**
   * Get the current user (Supabase-compatible)
   * Returns the user from the client-side session without making a network request
   * For server-side validation, use getCurrentUser() instead
   */
  async getUser(): Promise<FluxbaseResponse<{ user: User | null }>> {
    return { data: { user: this.session?.user ?? null }, error: null };
  }

  /**
   * Get the current access token
   */
  getAccessToken(): string | null {
    return this.session?.access_token ?? null;
  }

  /**
   * Listen to auth state changes (Supabase-compatible)
   * @param callback - Function called when auth state changes
   * @returns Object containing subscription data
   *
   * @example
   * ```typescript
   * const { data: { subscription } } = client.auth.onAuthStateChange((event, session) => {
   *   console.log('Auth event:', event, session)
   * })
   *
   * // Later, to unsubscribe:
   * subscription.unsubscribe()
   * ```
   */
  onAuthStateChange(callback: AuthStateChangeCallback): {
    data: { subscription: AuthSubscription };
  } {
    this.stateChangeListeners.add(callback);

    const subscription: AuthSubscription = {
      unsubscribe: () => {
        this.stateChangeListeners.delete(callback);
      },
    };

    return { data: { subscription } };
  }

  /**
   * Sign in with email and password (Supabase-compatible)
   * Returns { user, session } if successful, or SignInWith2FAResponse if 2FA is required
   */
  async signIn(
    credentials: SignInCredentials,
  ): Promise<FluxbaseResponse<AuthResponseData | SignInWith2FAResponse>> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<
        AuthResponse | SignInWith2FAResponse
      >("/api/v1/auth/signin", credentials);

      // Check if 2FA is required
      if ("requires_2fa" in response && response.requires_2fa) {
        return response as SignInWith2FAResponse;
      }

      // Normal sign in without 2FA
      const authResponse = response as AuthResponse;
      const session: AuthSession = {
        ...authResponse,
        expires_at: Date.now() + authResponse.expires_in * 1000,
      };

      this.setSessionInternal(session);
      return { user: session.user, session };
    });
  }

  /**
   * Sign in with email and password (Supabase-compatible)
   * Alias for signIn() to maintain compatibility with common authentication patterns
   * Returns { user, session } if successful, or SignInWith2FAResponse if 2FA is required
   */
  async signInWithPassword(
    credentials: SignInCredentials,
  ): Promise<FluxbaseResponse<AuthResponseData | SignInWith2FAResponse>> {
    return this.signIn(credentials);
  }

  /**
   * Sign up with email and password (Supabase-compatible)
   * Returns session when email confirmation is disabled
   * Returns null session when email confirmation is required
   */
  async signUp(credentials: SignUpCredentials): Promise<FluxbaseAuthResponse> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<AuthResponse>(
        "/api/v1/auth/signup",
        credentials,
      );

      // Check if session tokens are provided (no email confirmation required)
      if (response.access_token && response.refresh_token) {
        const session: AuthSession = {
          ...response,
          expires_at: Date.now() + response.expires_in * 1000,
        };

        this.setSessionInternal(session);
        return { user: response.user, session };
      }

      // Email confirmation required - return user without session
      return { user: response.user, session: null };
    });
  }

  /**
   * Sign out the current user
   */
  async signOut(): Promise<VoidResponse> {
    return wrapAsyncVoid(async () => {
      try {
        await this.fetch.post("/api/v1/auth/signout");
      } finally {
        this.clearSession();
      }
    });
  }

  /**
   * Refresh the session (Supabase-compatible)
   * Returns a new session with refreshed tokens
   */
  async refreshSession(): Promise<
    FluxbaseResponse<{ user: User; session: AuthSession }>
  > {
    return wrapAsync(async () => {
      if (!this.session?.refresh_token) {
        throw new Error("No refresh token available");
      }

      const response = await this.fetch.post<AuthResponse>(
        "/api/v1/auth/refresh",
        {
          refresh_token: this.session.refresh_token,
        },
      );

      const session: AuthSession = {
        ...response,
        expires_at: Date.now() + response.expires_in * 1000,
      };

      this.setSessionInternal(session, "TOKEN_REFRESHED");
      return { user: session.user, session };
    });
  }

  /**
   * Get the current user from the server
   */
  async getCurrentUser(): Promise<UserResponse> {
    return wrapAsync(async () => {
      if (!this.session) {
        throw new Error("Not authenticated");
      }

      const user = await this.fetch.get<User>("/api/v1/auth/user");
      return { user };
    });
  }

  /**
   * Update the current user
   */
  async updateUser(
    data: Partial<Pick<User, "email" | "metadata">>,
  ): Promise<UserResponse> {
    return wrapAsync(async () => {
      if (!this.session) {
        throw new Error("Not authenticated");
      }

      const user = await this.fetch.patch<User>("/api/v1/auth/user", data);

      // Update session with new user data
      if (this.session) {
        this.session.user = user;
        this.saveSession();
        this.emitAuthChange("USER_UPDATED", this.session);
      }

      return { user };
    });
  }

  /**
   * Set the session manually (Supabase-compatible)
   * Useful for restoring a session from storage or SSR scenarios
   * @param session - Object containing access_token and refresh_token
   * @returns Promise with session data
   */
  async setSession(session: {
    access_token: string;
    refresh_token: string;
  }): Promise<FluxbaseAuthResponse> {
    return wrapAsync(async () => {
      // Create a full auth session from the tokens
      const authSession: AuthSession = {
        access_token: session.access_token,
        refresh_token: session.refresh_token,
        user: null as any, // Will be populated by getCurrentUser
        expires_in: 3600, // Default, will be updated on refresh
        expires_at: Date.now() + 3600 * 1000,
      };

      // Set the token so we can make authenticated requests
      this.fetch.setAuthToken(session.access_token);

      // Fetch the current user to populate the session
      const user = await this.fetch.get<User>("/api/v1/auth/user");
      authSession.user = user;

      // Store the session
      this.setSessionInternal(authSession, "SIGNED_IN");

      return { user, session: authSession };
    });
  }

  /**
   * Setup 2FA for the current user (Supabase-compatible)
   * Enrolls a new MFA factor and returns TOTP details
   * @returns Promise with factor id, type, and TOTP setup details
   */
  async setup2FA(): Promise<DataResponse<TwoFactorSetupResponse>> {
    return wrapAsync(async () => {
      if (!this.session) {
        throw new Error("Not authenticated");
      }

      return await this.fetch.post<TwoFactorSetupResponse>(
        "/api/v1/auth/2fa/setup",
      );
    });
  }

  /**
   * Enable 2FA after verifying the TOTP code (Supabase-compatible)
   * Verifies the TOTP code and returns new tokens with MFA session
   * @param code - TOTP code from authenticator app
   * @returns Promise with access_token, refresh_token, and user
   */
  async enable2FA(
    code: string,
  ): Promise<DataResponse<TwoFactorEnableResponse>> {
    return wrapAsync(async () => {
      if (!this.session) {
        throw new Error("Not authenticated");
      }

      return await this.fetch.post<TwoFactorEnableResponse>(
        "/api/v1/auth/2fa/enable",
        { code },
      );
    });
  }

  /**
   * Disable 2FA for the current user (Supabase-compatible)
   * Unenrolls the MFA factor
   * @param password - User password for confirmation
   * @returns Promise with unenrolled factor id
   */
  async disable2FA(
    password: string,
  ): Promise<DataResponse<TwoFactorDisableResponse>> {
    return wrapAsync(async () => {
      if (!this.session) {
        throw new Error("Not authenticated");
      }

      return await this.fetch.post<TwoFactorDisableResponse>(
        "/api/v1/auth/2fa/disable",
        { password },
      );
    });
  }

  /**
   * Check 2FA status for the current user (Supabase-compatible)
   * Lists all enrolled MFA factors
   * @returns Promise with all factors and TOTP factors
   */
  async get2FAStatus(): Promise<DataResponse<TwoFactorStatusResponse>> {
    return wrapAsync(async () => {
      if (!this.session) {
        throw new Error("Not authenticated");
      }

      return await this.fetch.get<TwoFactorStatusResponse>(
        "/api/v1/auth/2fa/status",
      );
    });
  }

  /**
   * Verify 2FA code during login (Supabase-compatible)
   * Call this after signIn returns requires_2fa: true
   * @param request - User ID and TOTP code
   * @returns Promise with access_token, refresh_token, and user
   */
  async verify2FA(
    request: TwoFactorVerifyRequest,
  ): Promise<DataResponse<TwoFactorEnableResponse>> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<TwoFactorEnableResponse>(
        "/api/v1/auth/2fa/verify",
        request,
      );

      // Create session from the response tokens
      if (response.access_token && response.refresh_token) {
        const session: AuthSession = {
          user: response.user,
          access_token: response.access_token,
          refresh_token: response.refresh_token,
          expires_in: response.expires_in || 3600,
          expires_at: Date.now() + (response.expires_in || 3600) * 1000,
        };

        this.setSessionInternal(session, "MFA_CHALLENGE_VERIFIED");
      }

      return response;
    });
  }

  /**
   * Send password reset email (Supabase-compatible)
   * Sends a password reset link to the provided email address
   * @param email - Email address to send reset link to
   * @returns Promise with OTP-style response
   */
  async sendPasswordReset(
    email: string,
  ): Promise<DataResponse<PasswordResetResponse>> {
    return wrapAsync(async () => {
      await this.fetch.post("/api/v1/auth/password/reset", { email });
      // Return Supabase-compatible OTP response
      return { user: null, session: null };
    });
  }

  /**
   * Supabase-compatible alias for sendPasswordReset()
   * @param email - Email address to send reset link to
   * @param _options - Optional redirect configuration (currently not used)
   * @returns Promise with OTP-style response
   */
  async resetPasswordForEmail(
    email: string,
    _options?: { redirectTo?: string },
  ): Promise<DataResponse<PasswordResetResponse>> {
    return this.sendPasswordReset(email);
  }

  /**
   * Verify password reset token
   * Check if a password reset token is valid before allowing password reset
   * @param token - Password reset token to verify
   */
  async verifyResetToken(
    token: string,
  ): Promise<DataResponse<VerifyResetTokenResponse>> {
    return wrapAsync(async () => {
      return await this.fetch.post<VerifyResetTokenResponse>(
        "/api/v1/auth/password/reset/verify",
        {
          token,
        },
      );
    });
  }

  /**
   * Reset password with token (Supabase-compatible)
   * Complete the password reset process with a valid token
   * @param token - Password reset token
   * @param newPassword - New password to set
   * @returns Promise with user and new session
   */
  async resetPassword(
    token: string,
    newPassword: string,
  ): Promise<DataResponse<ResetPasswordResponse>> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<AuthResponse>(
        "/api/v1/auth/password/reset/confirm",
        {
          token,
          new_password: newPassword,
        },
      );

      const session: AuthSession = {
        ...response,
        expires_at: Date.now() + response.expires_in * 1000,
      };

      this.setSessionInternal(session, "PASSWORD_RECOVERY");
      return { user: session.user, session };
    });
  }

  /**
   * Send magic link for passwordless authentication (Supabase-compatible)
   * @param email - Email address to send magic link to
   * @param options - Optional configuration for magic link
   * @returns Promise with OTP-style response
   */
  async sendMagicLink(
    email: string,
    options?: MagicLinkOptions,
  ): Promise<DataResponse<MagicLinkResponse>> {
    return wrapAsync(async () => {
      await this.fetch.post("/api/v1/auth/magiclink", {
        email,
        redirect_to: options?.redirect_to,
      });
      // Return Supabase-compatible OTP response
      return { user: null, session: null };
    });
  }

  /**
   * Verify magic link token and sign in
   * @param token - Magic link token from email
   */
  async verifyMagicLink(token: string): Promise<FluxbaseAuthResponse> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<AuthResponse>(
        "/api/v1/auth/magiclink/verify",
        {
          token,
        },
      );

      const session: AuthSession = {
        ...response,
        expires_at: Date.now() + response.expires_in * 1000,
      };

      this.setSessionInternal(session);
      return { user: session.user, session };
    });
  }

  /**
   * Sign in anonymously
   * Creates a temporary anonymous user session
   */
  async signInAnonymously(): Promise<FluxbaseAuthResponse> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<AnonymousSignInResponse>(
        "/api/v1/auth/signin/anonymous",
      );

      const session: AuthSession = {
        ...response,
        expires_at: Date.now() + response.expires_in * 1000,
      };

      this.setSessionInternal(session);
      return { user: session.user, session };
    });
  }

  /**
   * Get list of enabled OAuth providers
   */
  async getOAuthProviders(): Promise<DataResponse<OAuthProvidersResponse>> {
    return wrapAsync(async () => {
      return await this.fetch.get<OAuthProvidersResponse>(
        "/api/v1/auth/oauth/providers",
      );
    });
  }

  /**
   * Get OAuth authorization URL for a provider
   * @param provider - OAuth provider name (e.g., 'google', 'github')
   * @param options - Optional OAuth configuration
   */
  async getOAuthUrl(
    provider: string,
    options?: OAuthOptions,
  ): Promise<DataResponse<OAuthUrlResponse>> {
    return wrapAsync(async () => {
      const params = new URLSearchParams();
      if (options?.redirect_to) {
        params.append("redirect_to", options.redirect_to);
      }
      if (options?.scopes && options.scopes.length > 0) {
        params.append("scopes", options.scopes.join(","));
      }

      const queryString = params.toString();
      const url = queryString
        ? `/api/v1/auth/oauth/${provider}/authorize?${queryString}`
        : `/api/v1/auth/oauth/${provider}/authorize`;

      const response = await this.fetch.get<OAuthUrlResponse>(url);
      return response;
    });
  }

  /**
   * Exchange OAuth authorization code for session
   * This is typically called in your OAuth callback handler
   * @param code - Authorization code from OAuth callback
   */
  async exchangeCodeForSession(code: string): Promise<FluxbaseAuthResponse> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<AuthResponse>(
        "/api/v1/auth/oauth/callback",
        { code },
      );

      const session: AuthSession = {
        ...response,
        expires_at: Date.now() + response.expires_in * 1000,
      };

      this.setSessionInternal(session);
      return { user: session.user, session };
    });
  }

  /**
   * Convenience method to initiate OAuth sign-in
   * Redirects the user to the OAuth provider's authorization page
   * @param provider - OAuth provider name (e.g., 'google', 'github')
   * @param options - Optional OAuth configuration
   */
  async signInWithOAuth(
    provider: string,
    options?: OAuthOptions,
  ): Promise<DataResponse<{ provider: string; url: string }>> {
    return wrapAsync(async () => {
      const result = await this.getOAuthUrl(provider, options);

      if (result.error) {
        throw result.error;
      }

      const url = result.data.url;

      if (typeof window !== "undefined") {
        window.location.href = url;
      } else {
        throw new Error(
          "signInWithOAuth can only be called in a browser environment",
        );
      }

      return { provider, url };
    });
  }

  /**
   * Internal: Set the session and persist it
   */
  private setSessionInternal(
    session: AuthSession,
    event: AuthChangeEvent = "SIGNED_IN",
  ) {
    this.session = session;
    this.fetch.setAuthToken(session.access_token);
    this.saveSession();
    this.scheduleTokenRefresh();
    this.emitAuthChange(event, session);
  }

  /**
   * Internal: Clear the session
   */
  private clearSession() {
    this.session = null;
    this.fetch.setAuthToken(null);

    if (this.persist && typeof localStorage !== "undefined") {
      localStorage.removeItem(AUTH_STORAGE_KEY);
    }

    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }

    this.emitAuthChange("SIGNED_OUT", null);
  }

  /**
   * Internal: Save session to storage
   */
  private saveSession() {
    if (this.persist && typeof localStorage !== "undefined" && this.session) {
      localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(this.session));
    }
  }

  /**
   * Internal: Schedule automatic token refresh
   */
  private scheduleTokenRefresh() {
    if (!this.autoRefresh || !this.session?.expires_at) {
      return;
    }

    // Clear existing timer
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
    }

    // Refresh 1 minute before expiry
    const refreshAt = this.session.expires_at - 60 * 1000;
    const delay = refreshAt - Date.now();

    if (delay > 0) {
      this.refreshTimer = setTimeout(async () => {
        const result = await this.refreshSession();
        if (result.error) {
          console.error("Failed to refresh token:", result.error);
          this.clearSession();
        }
      }, delay);
    }
  }

  /**
   * Internal: Emit auth state change event to all listeners
   */
  private emitAuthChange(event: AuthChangeEvent, session: AuthSession | null) {
    this.stateChangeListeners.forEach((callback) => {
      try {
        callback(event, session);
      } catch (error) {
        console.error("Error in auth state change listener:", error);
      }
    });
  }
}
