import { create } from 'zustand'
import { getCookie, setCookie, removeCookie } from '@/lib/cookies'
import { setAuthToken as setFluxbaseAuthToken } from '@/lib/fluxbase-client'

const AUTH_COOKIE_NAME = 'fluxbase_admin_token'
const REFRESH_COOKIE_NAME = 'fluxbase_admin_refresh_token'

interface AuthUser {
  accountNo: string
  email: string
  role: string[]
  exp: number
}

interface AuthState {
  auth: {
    user: AuthUser | null
    setUser: (user: AuthUser | null) => void
    accessToken: string
    refreshToken: string
    setAccessToken: (accessToken: string) => void
    setTokens: (accessToken: string, refreshToken: string) => void
    resetAccessToken: () => void
    reset: () => void
  }
}

function parseCookieToken(name: string): string {
  try {
    const raw = getCookie(name)
    return raw ? JSON.parse(raw) : ''
  } catch {
    removeCookie(name)
    return ''
  }
}

export const useAuthStore = create<AuthState>()((set) => {
  const initToken = parseCookieToken(AUTH_COOKIE_NAME)
  const initRefreshToken = parseCookieToken(REFRESH_COOKIE_NAME)

  if (initToken) {
    setFluxbaseAuthToken(initToken)
  }

  return {
    auth: {
      user: null,
      setUser: (user) =>
        set((state) => ({ ...state, auth: { ...state.auth, user } })),
      accessToken: initToken,
      refreshToken: initRefreshToken,
      setAccessToken: (accessToken) =>
        set((state) => {
          setCookie(AUTH_COOKIE_NAME, JSON.stringify(accessToken))
          setFluxbaseAuthToken(accessToken)
          return { ...state, auth: { ...state.auth, accessToken } }
        }),
      setTokens: (accessToken, refreshToken) =>
        set((state) => {
          setCookie(AUTH_COOKIE_NAME, JSON.stringify(accessToken))
          setCookie(REFRESH_COOKIE_NAME, JSON.stringify(refreshToken))
          setFluxbaseAuthToken(accessToken)
          return {
            ...state,
            auth: { ...state.auth, accessToken, refreshToken },
          }
        }),
      resetAccessToken: () =>
        set((state) => {
          removeCookie(AUTH_COOKIE_NAME)
          setFluxbaseAuthToken(null)
          return { ...state, auth: { ...state.auth, accessToken: '' } }
        }),
      reset: () =>
        set((state) => {
          removeCookie(AUTH_COOKIE_NAME)
          removeCookie(REFRESH_COOKIE_NAME)
          setFluxbaseAuthToken(null)
          return {
            ...state,
            auth: {
              ...state.auth,
              user: null,
              accessToken: '',
              refreshToken: '',
            },
          }
        }),
    },
  }
})
