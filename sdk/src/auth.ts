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
  UpdateUserAttributes,
  User,
  TwoFactorSetupResponse,
  TwoFactorEnableResponse,
  TwoFactorLoginResponse,
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
  OAuthLogoutOptions,
  OAuthLogoutResponse,
  AuthChangeEvent,
  AuthStateChangeCallback,
  AuthSubscription,
  FluxbaseAuthResponse,
  FluxbaseResponse,
  UserResponse,
  DataResponse,
  VoidResponse,
  SignInWithOtpCredentials,
  VerifyOtpParams,
  ResendOtpParams,
  OTPResponse,
  UserIdentitiesResponse,
  LinkIdentityCredentials,
  UnlinkIdentityParams,
  ReauthenticateResponse,
  SignInWithIdTokenCredentials,
  CaptchaConfig,
  SAMLProvidersResponse,
  SAMLLoginOptions,
  SAMLLoginResponse,
} from "./types";
import { wrapAsync, wrapAsyncVoid } from "./utils/error-handling";

const AUTH_STORAGE_KEY = "fluxbase.auth.session";
const OAUTH_PROVIDER_KEY = "fluxbase.auth.oauth_provider";

// Auto-refresh configuration constants
const AUTO_REFRESH_TICK_THRESHOLD = 10; // seconds before expiry to trigger refresh
const AUTO_REFRESH_TICK_MINIMUM = 1000; // minimum delay in ms (1 second)
const MAX_REFRESH_RETRIES = 3; // number of retry attempts before signing out

/**
 * In-memory storage adapter for Node.js/SSR environments
 * where localStorage is not available
 */
class MemoryStorage implements Storage {
  private store = new Map<string, string>();

  get length(): number {
    return this.store.size;
  }

  clear(): void {
    this.store.clear();
  }

  getItem(key: string): string | null {
    return this.store.get(key) ?? null;
  }

  setItem(key: string, value: string): void {
    this.store.set(key, value);
  }

  removeItem(key: string): void {
    this.store.delete(key);
  }

  key(index: number): string | null {
    return [...this.store.keys()][index] ?? null;
  }
}

/**
 * Check if localStorage is available and working
 */
function isLocalStorageAvailable(): boolean {
  try {
    if (typeof localStorage === "undefined") {
      return false;
    }
    // Test that localStorage actually works (some browsers block it)
    const testKey = "__fluxbase_storage_test__";
    localStorage.setItem(testKey, "test");
    localStorage.removeItem(testKey);
    return true;
  } catch {
    return false;
  }
}

export class FluxbaseAuth {
  private fetch: FluxbaseFetch;
  private session: AuthSession | null = null;
  private persist: boolean;
  private autoRefresh: boolean;
  private refreshTimer: ReturnType<typeof setTimeout> | null = null;
  private stateChangeListeners: Set<AuthStateChangeCallback> = new Set();
  private storage: Storage | null = null;

  constructor(fetch: FluxbaseFetch, autoRefresh = true, persist = true) {
    this.fetch = fetch;
    this.persist = persist;
    this.autoRefresh = autoRefresh;

    // Register refresh callback for automatic 401 handling
    this.fetch.setRefreshTokenCallback(async () => {
      const result = await this.refreshSession();
      return !result.error;
    });

    // Initialize storage based on persist option and environment
    if (this.persist) {
      if (isLocalStorageAvailable()) {
        this.storage = localStorage;
      } else {
        // Node.js/SSR fallback - use in-memory storage
        this.storage = new MemoryStorage();
      }
    }

    // Load session from storage if persisted
    if (this.storage) {
      const stored = this.storage.getItem(AUTH_STORAGE_KEY);
      if (stored) {
        try {
          this.session = JSON.parse(stored);
          if (this.session) {
            this.fetch.setAuthToken(this.session.access_token);
            // Schedule auto-refresh if enabled (only runs in browser environments)
            this.scheduleTokenRefresh();
          }
        } catch {
          // Invalid stored session, ignore
          this.storage.removeItem(AUTH_STORAGE_KEY);
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
   * Start the automatic token refresh timer
   * This is called automatically when autoRefresh is enabled and a session exists
   * Only works in browser environments
   */
  startAutoRefresh(): void {
    this.scheduleTokenRefresh();
  }

  /**
   * Stop the automatic token refresh timer
   * Call this when you want to disable auto-refresh without signing out
   */
  stopAutoRefresh(): void {
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }
  }

  /**
   * Sign in with email and password (Supabase-compatible)
   * Returns { user, session } if successful, or SignInWith2FAResponse if 2FA is required
   */
  async signIn(
    credentials: SignInCredentials,
  ): Promise<FluxbaseResponse<AuthResponseData | SignInWith2FAResponse>> {
    return wrapAsync(async () => {
      // Build request body with proper field names for backend
      const requestBody: any = {
        email: credentials.email,
        password: credentials.password,
      };

      // Include CAPTCHA token if provided (transform camelCase to snake_case)
      if (credentials.captchaToken) {
        requestBody.captcha_token = credentials.captchaToken;
      }

      const response = await this.fetch.post<
        AuthResponse | SignInWith2FAResponse
      >("/api/v1/auth/signin", requestBody);

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
      // Transform Supabase-style options.data to backend's user_metadata format
      const requestBody: any = {
        email: credentials.email,
        password: credentials.password,
      };

      // Map options.data to user_metadata for the backend
      if (credentials.options?.data) {
        requestBody.user_metadata = credentials.options.data;
      }

      // Include CAPTCHA token if provided
      if (credentials.captchaToken) {
        requestBody.captcha_token = credentials.captchaToken;
      }

      const response = await this.fetch.post<AuthResponse>(
        "/api/v1/auth/signup",
        requestBody,
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
   * Get CAPTCHA configuration from the server
   * Use this to determine which CAPTCHA provider to load and configure
   * @returns Promise with CAPTCHA configuration (provider, site key, enabled endpoints)
   */
  async getCaptchaConfig(): Promise<DataResponse<CaptchaConfig>> {
    return wrapAsync(async () => {
      return await this.fetch.get<CaptchaConfig>("/api/v1/auth/captcha/config");
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
        { skipAutoRefresh: true }, // Prevent infinite loop on 401
      );

      const session: AuthSession = {
        ...response,
        user: response.user ?? this.session.user,
        expires_at: Date.now() + response.expires_in * 1000,
      };

      this.setSessionInternal(session, "TOKEN_REFRESHED");
      return { user: session.user, session };
    });
  }

  /**
   * Refresh the session (Supabase-compatible alias)
   * Alias for refreshSession() to maintain compatibility with Supabase naming
   * Returns a new session with refreshed tokens
   */
  async refreshToken(): Promise<
    FluxbaseResponse<{ user: User; session: AuthSession }>
  > {
    return this.refreshSession();
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
   * Update the current user (Supabase-compatible)
   * @param attributes - User attributes to update (email, password, data for metadata)
   */
  async updateUser(attributes: UpdateUserAttributes): Promise<UserResponse> {
    return wrapAsync(async () => {
      if (!this.session) {
        throw new Error("Not authenticated");
      }

      // Transform Supabase-style 'data' to backend's 'user_metadata' format
      const requestBody: any = {};

      if (attributes.email) {
        requestBody.email = attributes.email;
      }

      if (attributes.password) {
        requestBody.password = attributes.password;
      }

      if (attributes.data) {
        requestBody.user_metadata = attributes.data;
      }

      if (attributes.nonce) {
        requestBody.nonce = attributes.nonce;
      }

      const user = await this.fetch.patch<User>(
        "/api/v1/auth/user",
        requestBody,
      );

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
   * @param issuer - Optional custom issuer name for the QR code (e.g., "MyApp"). If not provided, uses server default.
   * @returns Promise with factor id, type, and TOTP setup details
   */
  async setup2FA(
    issuer?: string,
  ): Promise<DataResponse<TwoFactorSetupResponse>> {
    return wrapAsync(async () => {
      if (!this.session) {
        throw new Error("Not authenticated");
      }

      return await this.fetch.post<TwoFactorSetupResponse>(
        "/api/v1/auth/2fa/setup",
        issuer ? { issuer } : undefined,
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
  ): Promise<DataResponse<TwoFactorLoginResponse>> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<TwoFactorLoginResponse>(
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
   * @param options - Optional configuration including redirect URL and CAPTCHA token
   * @returns Promise with OTP-style response
   */
  async sendPasswordReset(
    email: string,
    options?: { redirectTo?: string; captchaToken?: string },
  ): Promise<DataResponse<PasswordResetResponse>> {
    return wrapAsync(async () => {
      const requestBody: any = { email };

      // Include redirect URL if provided
      if (options?.redirectTo) {
        requestBody.redirect_to = options.redirectTo;
      }

      // Include CAPTCHA token if provided
      if (options?.captchaToken) {
        requestBody.captcha_token = options.captchaToken;
      }

      await this.fetch.post("/api/v1/auth/password/reset", requestBody);
      // Return Supabase-compatible OTP response
      return { user: null, session: null };
    });
  }

  /**
   * Supabase-compatible alias for sendPasswordReset()
   * @param email - Email address to send reset link to
   * @param options - Optional redirect and CAPTCHA configuration
   * @returns Promise with OTP-style response
   */
  async resetPasswordForEmail(
    email: string,
    options?: { redirectTo?: string; captchaToken?: string },
  ): Promise<DataResponse<PasswordResetResponse>> {
    return this.sendPasswordReset(email, {
      redirectTo: options?.redirectTo,
      captchaToken: options?.captchaToken,
    });
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
      const requestBody: any = {
        email,
        redirect_to: options?.redirect_to,
      };

      // Include CAPTCHA token if provided
      if (options?.captchaToken) {
        requestBody.captcha_token = options.captchaToken;
      }

      await this.fetch.post("/api/v1/auth/magiclink", requestBody);
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
      if (options?.redirect_uri) {
        params.append("redirect_uri", options.redirect_uri);
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
   * @param state - State parameter from OAuth callback (for CSRF protection)
   */
  async exchangeCodeForSession(
    code: string,
    state?: string,
  ): Promise<FluxbaseAuthResponse> {
    return wrapAsync(async () => {
      const provider = this.storage?.getItem(OAUTH_PROVIDER_KEY);
      if (!provider) {
        throw new Error("No OAuth provider found. Call signInWithOAuth first.");
      }

      // Build query string with code and optional state
      const params = new URLSearchParams({ code });
      if (state) {
        params.append("state", state);
      }

      const response = await this.fetch.get<AuthResponse>(
        `/api/v1/auth/oauth/${provider}/callback?${params.toString()}`,
      );

      // Clear stored provider after successful exchange
      this.storage?.removeItem(OAUTH_PROVIDER_KEY);

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
        // Store the provider for use in exchangeCodeForSession
        this.storage?.setItem(OAUTH_PROVIDER_KEY, provider);
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
   * Get OAuth logout URL for a provider
   * Use this to get the logout URL without automatically redirecting
   * @param provider - OAuth provider name (e.g., 'google', 'github')
   * @param options - Optional logout configuration
   * @returns Promise with OAuth logout response including redirect URL if applicable
   *
   * @example
   * ```typescript
   * const { data, error } = await client.auth.getOAuthLogoutUrl('google')
   * if (!error && data.redirect_url) {
   *   // Redirect user to complete logout at provider
   *   window.location.href = data.redirect_url
   * }
   * ```
   */
  async getOAuthLogoutUrl(
    provider: string,
    options?: OAuthLogoutOptions,
  ): Promise<DataResponse<OAuthLogoutResponse>> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<OAuthLogoutResponse>(
        `/api/v1/auth/oauth/${provider}/logout`,
        options || {},
      );

      // Clear local session
      this.clearSession();

      return response;
    });
  }

  /**
   * Sign out with OAuth provider logout
   * Revokes tokens at the OAuth provider and optionally redirects for OIDC logout
   * @param provider - OAuth provider name (e.g., 'google', 'github')
   * @param options - Optional logout configuration
   * @returns Promise with OAuth logout response
   *
   * @example
   * ```typescript
   * // This will revoke tokens and redirect to provider's logout page if supported
   * await client.auth.signOutWithOAuth('google', {
   *   redirect_url: 'https://myapp.com/logged-out'
   * })
   * ```
   */
  async signOutWithOAuth(
    provider: string,
    options?: OAuthLogoutOptions,
  ): Promise<DataResponse<OAuthLogoutResponse>> {
    return wrapAsync(async () => {
      const result = await this.getOAuthLogoutUrl(provider, options);

      if (result.error) {
        throw result.error;
      }

      // If redirect is needed and we're in a browser, redirect
      if (
        result.data.requires_redirect &&
        result.data.redirect_url &&
        typeof window !== "undefined"
      ) {
        window.location.href = result.data.redirect_url;
      }

      return result.data;
    });
  }

  /**
   * Sign in with OTP (One-Time Password) - Supabase-compatible
   * Sends a one-time password via email or SMS for passwordless authentication
   * @param credentials - Email or phone number and optional configuration
   * @returns Promise with OTP-style response
   */
  async signInWithOtp(
    credentials: SignInWithOtpCredentials,
  ): Promise<DataResponse<OTPResponse>> {
    return wrapAsync(async () => {
      await this.fetch.post("/api/v1/auth/otp/signin", credentials);
      // Return Supabase-compatible OTP response
      return { user: null, session: null };
    });
  }

  /**
   * Verify OTP (One-Time Password) - Supabase-compatible
   * Verify OTP tokens for various authentication flows
   * @param params - OTP verification parameters including token and type
   * @returns Promise with user and session if successful
   */
  async verifyOtp(params: VerifyOtpParams): Promise<FluxbaseAuthResponse> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<AuthResponse>(
        "/api/v1/auth/otp/verify",
        params,
      );

      // Check if session tokens are provided
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
   * Resend OTP (One-Time Password) - Supabase-compatible
   * Resend OTP code when user doesn't receive it
   * @param params - Resend parameters including type and email/phone
   * @returns Promise with OTP-style response
   */
  async resendOtp(params: ResendOtpParams): Promise<DataResponse<OTPResponse>> {
    return wrapAsync(async () => {
      await this.fetch.post("/api/v1/auth/otp/resend", params);
      // Return Supabase-compatible OTP response
      return { user: null, session: null };
    });
  }

  /**
   * Get user identities (linked OAuth providers) - Supabase-compatible
   * Lists all OAuth identities linked to the current user
   * @returns Promise with list of user identities
   */
  async getUserIdentities(): Promise<DataResponse<UserIdentitiesResponse>> {
    return wrapAsync(async () => {
      if (!this.session) {
        throw new Error("Not authenticated");
      }

      return await this.fetch.get<UserIdentitiesResponse>(
        "/api/v1/auth/user/identities",
      );
    });
  }

  /**
   * Link an OAuth identity to current user - Supabase-compatible
   * Links an additional OAuth provider to the existing account
   * @param credentials - Provider to link
   * @returns Promise with OAuth URL to complete linking
   */
  async linkIdentity(
    credentials: LinkIdentityCredentials,
  ): Promise<DataResponse<OAuthUrlResponse>> {
    return wrapAsync(async () => {
      if (!this.session) {
        throw new Error("Not authenticated");
      }

      return await this.fetch.post<OAuthUrlResponse>(
        "/api/v1/auth/user/identities",
        credentials,
      );
    });
  }

  /**
   * Unlink an OAuth identity from current user - Supabase-compatible
   * Removes a linked OAuth provider from the account
   * @param params - Identity to unlink
   * @returns Promise with void response
   */
  async unlinkIdentity(params: UnlinkIdentityParams): Promise<VoidResponse> {
    return wrapAsyncVoid(async () => {
      if (!this.session) {
        throw new Error("Not authenticated");
      }

      await this.fetch.delete(
        `/api/v1/auth/user/identities/${params.identity.id}`,
      );
    });
  }

  /**
   * Reauthenticate to get security nonce - Supabase-compatible
   * Get a security nonce for sensitive operations (password change, etc.)
   * @returns Promise with nonce for reauthentication
   */
  async reauthenticate(): Promise<DataResponse<ReauthenticateResponse>> {
    return wrapAsync(async () => {
      if (!this.session) {
        throw new Error("Not authenticated");
      }

      return await this.fetch.post<ReauthenticateResponse>(
        "/api/v1/auth/reauthenticate",
      );
    });
  }

  /**
   * Sign in with ID token (for native mobile apps) - Supabase-compatible
   * Authenticate using native mobile app ID tokens (Google, Apple)
   * @param credentials - Provider, ID token, and optional nonce
   * @returns Promise with user and session
   */
  async signInWithIdToken(
    credentials: SignInWithIdTokenCredentials,
  ): Promise<FluxbaseAuthResponse> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<AuthResponse>(
        "/api/v1/auth/signin/idtoken",
        credentials,
      );

      const session: AuthSession = {
        ...response,
        expires_at: Date.now() + response.expires_in * 1000,
      };

      this.setSessionInternal(session);
      return { user: session.user, session };
    });
  }

  // ==========================================================================
  // SAML SSO Methods
  // ==========================================================================

  /**
   * Get list of available SAML SSO providers
   * @returns Promise with list of configured SAML providers
   *
   * @example
   * ```typescript
   * const { data, error } = await client.auth.getSAMLProviders()
   * if (!error) {
   *   console.log('Available providers:', data.providers)
   * }
   * ```
   */
  async getSAMLProviders(): Promise<DataResponse<SAMLProvidersResponse>> {
    return wrapAsync(async () => {
      return await this.fetch.get<SAMLProvidersResponse>(
        "/api/v1/auth/saml/providers",
      );
    });
  }

  /**
   * Get SAML login URL for a specific provider
   * Use this to redirect the user to the IdP for authentication
   * @param provider - SAML provider name/ID
   * @param options - Optional login configuration
   * @returns Promise with SAML login URL
   *
   * @example
   * ```typescript
   * const { data, error } = await client.auth.getSAMLLoginUrl('okta')
   * if (!error) {
   *   window.location.href = data.url
   * }
   * ```
   */
  async getSAMLLoginUrl(
    provider: string,
    options?: SAMLLoginOptions,
  ): Promise<DataResponse<SAMLLoginResponse>> {
    return wrapAsync(async () => {
      const params = new URLSearchParams();
      if (options?.redirectUrl) {
        params.append("redirect_url", options.redirectUrl);
      }

      const queryString = params.toString();
      const url = queryString
        ? `/api/v1/auth/saml/login/${provider}?${queryString}`
        : `/api/v1/auth/saml/login/${provider}`;

      const response = await this.fetch.get<SAMLLoginResponse>(url);
      return response;
    });
  }

  /**
   * Initiate SAML login and redirect to IdP
   * This is a convenience method that redirects the user to the SAML IdP
   * @param provider - SAML provider name/ID
   * @param options - Optional login configuration
   * @returns Promise with provider and URL (browser will redirect)
   *
   * @example
   * ```typescript
   * // In browser, this will redirect to the SAML IdP
   * await client.auth.signInWithSAML('okta')
   * ```
   */
  async signInWithSAML(
    provider: string,
    options?: SAMLLoginOptions,
  ): Promise<DataResponse<{ provider: string; url: string }>> {
    return wrapAsync(async () => {
      const result = await this.getSAMLLoginUrl(provider, options);

      if (result.error) {
        throw result.error;
      }

      const url = result.data.url;

      if (typeof window !== "undefined") {
        window.location.href = url;
      } else {
        throw new Error(
          "signInWithSAML can only be called in a browser environment",
        );
      }

      return { provider, url };
    });
  }

  /**
   * Handle SAML callback after IdP authentication
   * Call this from your SAML callback page to complete authentication
   * @param samlResponse - Base64-encoded SAML response from the ACS endpoint
   * @param provider - SAML provider name (optional, extracted from RelayState)
   * @returns Promise with user and session
   *
   * @example
   * ```typescript
   * // In your SAML callback page
   * const urlParams = new URLSearchParams(window.location.search)
   * const samlResponse = urlParams.get('SAMLResponse')
   *
   * if (samlResponse) {
   *   const { data, error } = await client.auth.handleSAMLCallback(samlResponse)
   *   if (!error) {
   *     console.log('Logged in:', data.user)
   *   }
   * }
   * ```
   */
  async handleSAMLCallback(
    samlResponse: string,
    provider?: string,
  ): Promise<FluxbaseAuthResponse> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<AuthResponse>(
        "/api/v1/auth/saml/acs",
        {
          saml_response: samlResponse,
          provider,
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
   * Get SAML Service Provider metadata for a specific provider configuration
   * Use this when configuring your IdP to download the SP metadata XML
   * @param provider - SAML provider name/ID
   * @returns Promise with SP metadata URL
   *
   * @example
   * ```typescript
   * const metadataUrl = client.auth.getSAMLMetadataUrl('okta')
   * // Share this URL with your IdP administrator
   * ```
   */
  getSAMLMetadataUrl(provider: string): string {
    const baseUrl = this.fetch["baseUrl"];
    return `${baseUrl}/api/v1/auth/saml/metadata/${provider}`;
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

    if (this.storage) {
      this.storage.removeItem(AUTH_STORAGE_KEY);
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
    if (this.storage && this.session) {
      this.storage.setItem(AUTH_STORAGE_KEY, JSON.stringify(this.session));
    }
  }

  /**
   * Internal: Schedule automatic token refresh
   * Only runs in browser environments when autoRefresh is enabled
   */
  private scheduleTokenRefresh() {
    // Only auto-refresh in browser environments
    if (!this.autoRefresh || typeof window === "undefined") {
      return;
    }

    if (!this.session?.expires_at) {
      return;
    }

    // Clear existing timer
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }

    // Calculate time until expiry (expires_at is in ms)
    const expiresAt = this.session.expires_at;
    const now = Date.now();
    const timeUntilExpiry = expiresAt - now;

    // Refresh 10 seconds before expiry, minimum 1 second delay
    const refreshIn = Math.max(
      timeUntilExpiry - AUTO_REFRESH_TICK_THRESHOLD * 1000,
      AUTO_REFRESH_TICK_MINIMUM,
    );

    this.refreshTimer = setTimeout(() => {
      this.attemptRefresh();
    }, refreshIn);
  }

  /**
   * Internal: Attempt to refresh the token with retry logic
   * Uses exponential backoff: 1s, 2s, 4s delays between retries
   */
  private async attemptRefresh(retries = MAX_REFRESH_RETRIES): Promise<void> {
    try {
      const result = await this.refreshSession();
      if (result.error) {
        throw result.error;
      }
      // Success - scheduleTokenRefresh is called within setSessionInternal
      // via refreshSession -> setSessionInternal -> scheduleTokenRefresh
    } catch (error) {
      if (retries > 0) {
        // Exponential backoff: 1s, 2s, 4s (Math.pow(2, MAX_REFRESH_RETRIES - retries) * 1000)
        const delay = Math.pow(2, MAX_REFRESH_RETRIES - retries) * 1000;
        console.warn(
          `Token refresh failed, retrying in ${delay / 1000}s (${retries} attempts remaining)`,
          error,
        );
        this.refreshTimer = setTimeout(() => {
          this.attemptRefresh(retries - 1);
        }, delay);
      } else {
        // All retries exhausted - sign out
        console.error(
          "Token refresh failed after all retries, signing out",
          error,
        );
        this.clearSession();
      }
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
