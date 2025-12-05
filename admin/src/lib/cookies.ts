/**
 * Cookie utility functions using manual document.cookie approach
 * Replaces js-cookie dependency for better consistency
 *
 * Security notes:
 * - SameSite=Strict prevents CSRF attacks by not sending cookies on cross-site requests
 * - Secure ensures cookies are only sent over HTTPS (disabled in development)
 * - Note: httpOnly cannot be set via JavaScript - for maximum security,
 *   sensitive tokens should be set by the server as httpOnly cookies
 */

const DEFAULT_MAX_AGE = 60 * 60 * 24 * 7 // 7 days

/**
 * Check if running in secure context (HTTPS or localhost)
 */
const isSecureContext = (): boolean => {
  if (typeof window === 'undefined') return false
  return (
    window.location.protocol === 'https:' ||
    window.location.hostname === 'localhost' ||
    window.location.hostname === '127.0.0.1'
  )
}

/**
 * Get a cookie value by name
 */
export function getCookie(name: string): string | undefined {
  if (typeof document === 'undefined') return undefined

  const value = `; ${document.cookie}`
  const parts = value.split(`; ${name}=`)
  if (parts.length === 2) {
    const cookieValue = parts.pop()?.split(';').shift()
    return cookieValue
  }
  return undefined
}

/**
 * Set a cookie with name, value, and optional max age
 * Includes security attributes:
 * - SameSite=Strict: Prevents CSRF by only sending cookie for same-site requests
 * - Secure: Only sent over HTTPS (except in local development)
 */
export function setCookie(
  name: string,
  value: string,
  maxAge: number = DEFAULT_MAX_AGE
): void {
  if (typeof document === 'undefined') return

  // Build cookie string with security attributes
  let cookieString = `${name}=${value}; path=/; max-age=${maxAge}; SameSite=Strict`

  // Add Secure flag in production (HTTPS) but allow HTTP in development
  if (isSecureContext() && window.location.protocol === 'https:') {
    cookieString += '; Secure'
  }

  document.cookie = cookieString
}

/**
 * Remove a cookie by setting its max age to 0
 */
export function removeCookie(name: string): void {
  if (typeof document === 'undefined') return

  document.cookie = `${name}=; path=/; max-age=0; SameSite=Strict`
}
