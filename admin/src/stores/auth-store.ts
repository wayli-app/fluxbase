import { create } from 'zustand'
import { getCookie, setCookie, removeCookie } from '@/lib/cookies'
import { setAuthToken as setFluxbaseAuthToken } from '@/lib/fluxbase-client'

const AUTH_COOKIE_NAME = 'fluxbase_admin_token'

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
    setAccessToken: (accessToken: string) => void
    resetAccessToken: () => void
    reset: () => void
  }
}

export const useAuthStore = create<AuthState>()((set) => {
  const cookieState = getCookie(AUTH_COOKIE_NAME)
  const initToken = cookieState ? JSON.parse(cookieState) : ''

  // Initialize Fluxbase client with the stored token
  if (initToken) {
    setFluxbaseAuthToken(initToken)
  }

  return {
    auth: {
      user: null,
      setUser: (user) =>
        set((state) => ({ ...state, auth: { ...state.auth, user } })),
      accessToken: initToken,
      setAccessToken: (accessToken) =>
        set((state) => {
          setCookie(AUTH_COOKIE_NAME, JSON.stringify(accessToken))
          setFluxbaseAuthToken(accessToken) // Sync with Fluxbase client
          return { ...state, auth: { ...state.auth, accessToken } }
        }),
      resetAccessToken: () =>
        set((state) => {
          removeCookie(AUTH_COOKIE_NAME)
          setFluxbaseAuthToken(null) // Clear Fluxbase client token
          return { ...state, auth: { ...state.auth, accessToken: '' } }
        }),
      reset: () =>
        set((state) => {
          removeCookie(AUTH_COOKIE_NAME)
          setFluxbaseAuthToken(null) // Clear Fluxbase client token
          return {
            ...state,
            auth: { ...state.auth, user: null, accessToken: '' },
          }
        }),
    },
  }
})
