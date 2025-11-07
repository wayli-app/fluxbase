// Authentication utilities for admin UI

const ACCESS_TOKEN_KEY = 'fluxbase_admin_access_token'
const REFRESH_TOKEN_KEY = 'fluxbase_admin_refresh_token'
const USER_KEY = 'fluxbase_admin_user'

export interface AdminUser {
  id: string
  email: string
  role: string
  email_verified: boolean
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

// DashboardUser type (from dashboard authentication)
export interface DashboardUser {
  id: string
  email: string
  email_verified: boolean
  full_name: string | null
  avatar_url: string | null
  totp_enabled: boolean
  is_active: boolean
  is_locked: boolean
  last_login_at: string | null
  created_at: string
  updated_at: string
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  expires_in: number
}

// Get access token from localStorage
export function getAccessToken(): string | null {
  if (typeof window === 'undefined') return null
  // Check both new key and legacy key for backwards compatibility
  return localStorage.getItem(ACCESS_TOKEN_KEY) || localStorage.getItem('access_token')
}

// Get refresh token from localStorage
export function getRefreshToken(): string | null {
  if (typeof window === 'undefined') return null
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}

// Get stored user from localStorage (can be either AdminUser or DashboardUser)
export function getStoredUser(): AdminUser | DashboardUser | null {
  if (typeof window === 'undefined') return null
  // Check both new key and legacy key for backwards compatibility
  const userJson = localStorage.getItem(USER_KEY) || localStorage.getItem('user')
  if (!userJson) return null
  try {
    return JSON.parse(userJson)
  } catch {
    return null
  }
}

// Store tokens and user in localStorage
export function setTokens(tokens: TokenPair, user: AdminUser): void {
  if (typeof window === 'undefined') return
  localStorage.setItem(ACCESS_TOKEN_KEY, tokens.access_token)
  localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token)
  localStorage.setItem(USER_KEY, JSON.stringify(user))
}

// Clear all auth data from localStorage
export function clearTokens(): void {
  if (typeof window === 'undefined') return
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
  // Also clear legacy keys
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  localStorage.removeItem('user')
}

// Check if user is authenticated (has valid token)
export function isAuthenticated(): boolean {
  return !!getAccessToken()
}

// Logout helper - clears tokens and redirects to login
export function logout(): void {
  clearTokens()
  window.location.href = '/admin/login'
}
