/**
 * Authentication module for Fluxbase SDK
 */

import type { FluxbaseFetch } from './fetch'
import type {
  AuthResponse,
  AuthSession,
  SignInCredentials,
  SignUpCredentials,
  User,
  TwoFactorSetupResponse,
  TwoFactorEnableResponse,
  TwoFactorStatusResponse,
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
} from './types'

const AUTH_STORAGE_KEY = 'fluxbase.auth.session'

export class FluxbaseAuth {
  private fetch: FluxbaseFetch
  private session: AuthSession | null = null
  private persist: boolean
  private autoRefresh: boolean
  private refreshTimer: ReturnType<typeof setTimeout> | null = null

  constructor(fetch: FluxbaseFetch, autoRefresh = true, persist = true) {
    this.fetch = fetch
    this.persist = persist
    this.autoRefresh = autoRefresh

    // Load session from storage if persisted
    if (this.persist && typeof localStorage !== 'undefined') {
      const stored = localStorage.getItem(AUTH_STORAGE_KEY)
      if (stored) {
        try {
          this.session = JSON.parse(stored)
          if (this.session) {
            this.fetch.setAuthToken(this.session.access_token)
            this.scheduleTokenRefresh()
          }
        } catch {
          // Invalid stored session, ignore
          localStorage.removeItem(AUTH_STORAGE_KEY)
        }
      }
    }
  }

  /**
   * Get the current session
   */
  getSession(): AuthSession | null {
    return this.session
  }

  /**
   * Get the current user
   */
  getUser(): User | null {
    return this.session?.user ?? null
  }

  /**
   * Get the current access token
   */
  getAccessToken(): string | null {
    return this.session?.access_token ?? null
  }

  /**
   * Sign in with email and password
   * Returns AuthSession if successful, or SignInWith2FAResponse if 2FA is required
   */
  async signIn(credentials: SignInCredentials): Promise<AuthSession | SignInWith2FAResponse> {
    const response = await this.fetch.post<AuthResponse | SignInWith2FAResponse>(
      '/api/v1/auth/signin',
      credentials
    )

    // Check if 2FA is required
    if ('requires_2fa' in response && response.requires_2fa) {
      return response as SignInWith2FAResponse
    }

    // Normal sign in without 2FA
    const authResponse = response as AuthResponse
    const session: AuthSession = {
      ...authResponse,
      expires_at: Date.now() + authResponse.expires_in * 1000,
    }

    this.setSession(session)
    return session
  }

  /**
   * Sign up with email and password
   */
  async signUp(credentials: SignUpCredentials): Promise<AuthSession> {
    const response = await this.fetch.post<AuthResponse>('/api/v1/auth/signup', credentials)

    const session: AuthSession = {
      ...response,
      expires_at: Date.now() + response.expires_in * 1000,
    }

    this.setSession(session)
    return session
  }

  /**
   * Sign out the current user
   */
  async signOut(): Promise<void> {
    try {
      await this.fetch.post('/api/v1/auth/signout')
    } finally {
      this.clearSession()
    }
  }

  /**
   * Refresh the access token
   */
  async refreshToken(): Promise<AuthSession> {
    if (!this.session?.refresh_token) {
      throw new Error('No refresh token available')
    }

    const response = await this.fetch.post<AuthResponse>('/api/v1/auth/refresh', {
      refresh_token: this.session.refresh_token,
    })

    const session: AuthSession = {
      ...response,
      expires_at: Date.now() + response.expires_in * 1000,
    }

    this.setSession(session)
    return session
  }

  /**
   * Get the current user from the server
   */
  async getCurrentUser(): Promise<User> {
    if (!this.session) {
      throw new Error('Not authenticated')
    }

    return await this.fetch.get<User>('/api/v1/auth/user')
  }

  /**
   * Update the current user
   */
  async updateUser(data: Partial<Pick<User, 'email' | 'metadata'>>): Promise<User> {
    if (!this.session) {
      throw new Error('Not authenticated')
    }

    const user = await this.fetch.patch<User>('/api/v1/auth/user', data)

    // Update session with new user data
    if (this.session) {
      this.session.user = user
      this.saveSession()
    }

    return user
  }

  /**
   * Set the auth token manually
   */
  setToken(token: string) {
    this.fetch.setAuthToken(token)
  }

  /**
   * Setup 2FA for the current user
   * Returns TOTP secret and QR code URL
   */
  async setup2FA(): Promise<TwoFactorSetupResponse> {
    if (!this.session) {
      throw new Error('Not authenticated')
    }

    return await this.fetch.post<TwoFactorSetupResponse>('/api/v1/auth/2fa/setup')
  }

  /**
   * Enable 2FA after verifying the TOTP code
   * Returns backup codes that should be saved by the user
   */
  async enable2FA(code: string): Promise<TwoFactorEnableResponse> {
    if (!this.session) {
      throw new Error('Not authenticated')
    }

    return await this.fetch.post<TwoFactorEnableResponse>('/api/v1/auth/2fa/enable', { code })
  }

  /**
   * Disable 2FA for the current user
   * Requires password confirmation
   */
  async disable2FA(password: string): Promise<{ success: boolean; message: string }> {
    if (!this.session) {
      throw new Error('Not authenticated')
    }

    return await this.fetch.post<{ success: boolean; message: string }>(
      '/api/v1/auth/2fa/disable',
      { password }
    )
  }

  /**
   * Check 2FA status for the current user
   */
  async get2FAStatus(): Promise<TwoFactorStatusResponse> {
    if (!this.session) {
      throw new Error('Not authenticated')
    }

    return await this.fetch.get<TwoFactorStatusResponse>('/api/v1/auth/2fa/status')
  }

  /**
   * Verify 2FA code during login
   * Call this after signIn returns requires_2fa: true
   */
  async verify2FA(request: TwoFactorVerifyRequest): Promise<AuthSession> {
    const response = await this.fetch.post<AuthResponse>('/api/v1/auth/2fa/verify', request)

    const session: AuthSession = {
      ...response,
      expires_at: Date.now() + response.expires_in * 1000,
    }

    this.setSession(session)
    return session
  }

  /**
   * Send password reset email
   * Sends a password reset link to the provided email address
   * @param email - Email address to send reset link to
   */
  async sendPasswordReset(email: string): Promise<PasswordResetResponse> {
    return await this.fetch.post<PasswordResetResponse>('/api/v1/auth/password/reset', { email })
  }

  /**
   * Verify password reset token
   * Check if a password reset token is valid before allowing password reset
   * @param token - Password reset token to verify
   */
  async verifyResetToken(token: string): Promise<VerifyResetTokenResponse> {
    return await this.fetch.post<VerifyResetTokenResponse>('/api/v1/auth/password/reset/verify', {
      token,
    })
  }

  /**
   * Reset password with token
   * Complete the password reset process with a valid token
   * @param token - Password reset token
   * @param newPassword - New password to set
   */
  async resetPassword(token: string, newPassword: string): Promise<ResetPasswordResponse> {
    return await this.fetch.post<ResetPasswordResponse>('/api/v1/auth/password/reset/confirm', {
      token,
      new_password: newPassword,
    })
  }

  /**
   * Send magic link for passwordless authentication
   * @param email - Email address to send magic link to
   * @param options - Optional configuration for magic link
   */
  async sendMagicLink(email: string, options?: MagicLinkOptions): Promise<MagicLinkResponse> {
    return await this.fetch.post<MagicLinkResponse>('/api/v1/auth/magiclink', {
      email,
      redirect_to: options?.redirect_to,
    })
  }

  /**
   * Verify magic link token and sign in
   * @param token - Magic link token from email
   */
  async verifyMagicLink(token: string): Promise<AuthSession> {
    const response = await this.fetch.post<AuthResponse>('/api/v1/auth/magiclink/verify', {
      token,
    })

    const session: AuthSession = {
      ...response,
      expires_at: Date.now() + response.expires_in * 1000,
    }

    this.setSession(session)
    return session
  }

  /**
   * Sign in anonymously
   * Creates a temporary anonymous user session
   */
  async signInAnonymously(): Promise<AuthSession> {
    const response = await this.fetch.post<AnonymousSignInResponse>('/api/v1/auth/signin/anonymous')

    const session: AuthSession = {
      ...response,
      expires_at: Date.now() + response.expires_in * 1000,
    }

    this.setSession(session)
    return session
  }

  /**
   * Get list of enabled OAuth providers
   */
  async getOAuthProviders(): Promise<OAuthProvidersResponse> {
    return await this.fetch.get<OAuthProvidersResponse>('/api/v1/auth/oauth/providers')
  }

  /**
   * Get OAuth authorization URL for a provider
   * @param provider - OAuth provider name (e.g., 'google', 'github')
   * @param options - Optional OAuth configuration
   */
  async getOAuthUrl(provider: string, options?: OAuthOptions): Promise<OAuthUrlResponse> {
    const params = new URLSearchParams()
    if (options?.redirect_to) {
      params.append('redirect_to', options.redirect_to)
    }
    if (options?.scopes && options.scopes.length > 0) {
      params.append('scopes', options.scopes.join(','))
    }

    const queryString = params.toString()
    const url = queryString
      ? `/api/v1/auth/oauth/${provider}/authorize?${queryString}`
      : `/api/v1/auth/oauth/${provider}/authorize`

    const response = await this.fetch.get<OAuthUrlResponse>(url)
    return response
  }

  /**
   * Exchange OAuth authorization code for session
   * This is typically called in your OAuth callback handler
   * @param code - Authorization code from OAuth callback
   */
  async exchangeCodeForSession(code: string): Promise<AuthSession> {
    const response = await this.fetch.post<AuthResponse>('/api/v1/auth/oauth/callback', { code })

    const session: AuthSession = {
      ...response,
      expires_at: Date.now() + response.expires_in * 1000,
    }

    this.setSession(session)
    return session
  }

  /**
   * Convenience method to initiate OAuth sign-in
   * Redirects the user to the OAuth provider's authorization page
   * @param provider - OAuth provider name (e.g., 'google', 'github')
   * @param options - Optional OAuth configuration
   */
  async signInWithOAuth(provider: string, options?: OAuthOptions): Promise<void> {
    const { url } = await this.getOAuthUrl(provider, options)

    if (typeof window !== 'undefined') {
      window.location.href = url
    } else {
      throw new Error('signInWithOAuth can only be called in a browser environment')
    }
  }

  /**
   * Internal: Set the session and persist it
   */
  private setSession(session: AuthSession) {
    this.session = session
    this.fetch.setAuthToken(session.access_token)
    this.saveSession()
    this.scheduleTokenRefresh()
  }

  /**
   * Internal: Clear the session
   */
  private clearSession() {
    this.session = null
    this.fetch.setAuthToken(null)

    if (this.persist && typeof localStorage !== 'undefined') {
      localStorage.removeItem(AUTH_STORAGE_KEY)
    }

    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer)
      this.refreshTimer = null
    }
  }

  /**
   * Internal: Save session to storage
   */
  private saveSession() {
    if (this.persist && typeof localStorage !== 'undefined' && this.session) {
      localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(this.session))
    }
  }

  /**
   * Internal: Schedule automatic token refresh
   */
  private scheduleTokenRefresh() {
    if (!this.autoRefresh || !this.session?.expires_at) {
      return
    }

    // Clear existing timer
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer)
    }

    // Refresh 1 minute before expiry
    const refreshAt = this.session.expires_at - 60 * 1000
    const delay = refreshAt - Date.now()

    if (delay > 0) {
      this.refreshTimer = setTimeout(() => {
        this.refreshToken().catch((err) => {
          console.error('Failed to refresh token:', err)
          this.clearSession()
        })
      }, delay)
    }
  }
}
