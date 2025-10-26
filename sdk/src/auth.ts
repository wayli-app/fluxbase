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
   */
  async signIn(credentials: SignInCredentials): Promise<AuthSession> {
    const response = await this.fetch.post<AuthResponse>('/api/auth/signin', credentials)

    const session: AuthSession = {
      ...response,
      expires_at: Date.now() + response.expires_in * 1000,
    }

    this.setSession(session)
    return session
  }

  /**
   * Sign up with email and password
   */
  async signUp(credentials: SignUpCredentials): Promise<AuthSession> {
    const response = await this.fetch.post<AuthResponse>('/api/auth/signup', credentials)

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
      await this.fetch.post('/api/auth/signout')
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

    const response = await this.fetch.post<AuthResponse>('/api/auth/refresh', {
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

    return await this.fetch.get<User>('/api/auth/user')
  }

  /**
   * Update the current user
   */
  async updateUser(data: Partial<Pick<User, 'email' | 'metadata'>>): Promise<User> {
    if (!this.session) {
      throw new Error('Not authenticated')
    }

    const user = await this.fetch.patch<User>('/api/auth/user', data)

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
