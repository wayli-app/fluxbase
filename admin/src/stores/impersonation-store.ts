import { create } from 'zustand'

export type ImpersonationType = 'user' | 'anon' | 'service'

export interface ImpersonationSession {
  id: string
  admin_user_id: string
  target_user_id?: string
  impersonation_type: ImpersonationType
  target_role?: string
  reason: string
  started_at: string
  ip_address?: string
  user_agent?: string
  is_active: boolean
}

export interface ImpersonatedUser {
  id: string
  email: string
  role?: string
}

interface ImpersonationState {
  isImpersonating: boolean
  impersonationType: ImpersonationType | null
  impersonationToken: string | null
  impersonationRefreshToken: string | null
  impersonatedUser: ImpersonatedUser | null
  session: ImpersonationSession | null

  // Actions
  startImpersonation: (
    token: string,
    refreshToken: string,
    user: ImpersonatedUser,
    session: ImpersonationSession,
    type: ImpersonationType
  ) => void
  stopImpersonation: () => void
  updateSession: (session: ImpersonationSession) => void
}

const STORAGE_KEYS = {
  TOKEN: 'fluxbase_impersonation_token',
  REFRESH_TOKEN: 'fluxbase_impersonation_refresh_token',
  USER: 'fluxbase_impersonated_user',
  SESSION: 'fluxbase_impersonation_session',
  TYPE: 'fluxbase_impersonation_type',
}

// Load initial state from localStorage
const loadFromStorage = () => {
  try {
    const token = localStorage.getItem(STORAGE_KEYS.TOKEN)
    const refreshToken = localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN)
    const userStr = localStorage.getItem(STORAGE_KEYS.USER)
    const sessionStr = localStorage.getItem(STORAGE_KEYS.SESSION)
    const typeStr = localStorage.getItem(STORAGE_KEYS.TYPE)

    if (token && userStr && sessionStr && typeStr) {
      return {
        isImpersonating: true,
        impersonationType: typeStr as ImpersonationType,
        impersonationToken: token,
        impersonationRefreshToken: refreshToken,
        impersonatedUser: JSON.parse(userStr),
        session: JSON.parse(sessionStr),
      }
    }
  } catch (error) {
    console.error('Failed to load impersonation state from storage:', error)
  }

  return {
    isImpersonating: false,
    impersonationType: null,
    impersonationToken: null,
    impersonationRefreshToken: null,
    impersonatedUser: null,
    session: null,
  }
}

export const useImpersonationStore = create<ImpersonationState>((set) => ({
  ...loadFromStorage(),

  startImpersonation: (token, refreshToken, user, session, type) => {
    // Save to localStorage
    localStorage.setItem(STORAGE_KEYS.TOKEN, token)
    if (refreshToken) {
      localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, refreshToken)
    }
    localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify(user))
    localStorage.setItem(STORAGE_KEYS.SESSION, JSON.stringify(session))
    localStorage.setItem(STORAGE_KEYS.TYPE, type)

    set({
      isImpersonating: true,
      impersonationType: type,
      impersonationToken: token,
      impersonationRefreshToken: refreshToken,
      impersonatedUser: user,
      session,
    })
  },

  stopImpersonation: () => {
    // Clear from localStorage
    localStorage.removeItem(STORAGE_KEYS.TOKEN)
    localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN)
    localStorage.removeItem(STORAGE_KEYS.USER)
    localStorage.removeItem(STORAGE_KEYS.SESSION)
    localStorage.removeItem(STORAGE_KEYS.TYPE)

    set({
      isImpersonating: false,
      impersonationType: null,
      impersonationToken: null,
      impersonationRefreshToken: null,
      impersonatedUser: null,
      session: null,
    })
  },

  updateSession: (session) => {
    localStorage.setItem(STORAGE_KEYS.SESSION, JSON.stringify(session))
    set({ session })
  },
}))
