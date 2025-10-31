import axios, { type AxiosError, type AxiosInstance } from 'axios'
import { getAccessToken, getRefreshToken, setTokens, clearTokens } from './auth'
import type { AdminUser } from './auth'

// Base URL for the API - can be overridden with environment variable
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// Create axios instance with default config
const api: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 30000, // 30 seconds
})

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const accessToken = getAccessToken()
    if (accessToken) {
      config.headers.Authorization = `Bearer ${accessToken}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor to handle token refresh and errors
let isRefreshing = false
let failedQueue: Array<{
  resolve: (value: unknown) => void
  reject: (reason?: unknown) => void
}> = []

const processQueue = (error: Error | null, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error)
    } else {
      prom.resolve(token)
    }
  })

  failedQueue = []
}

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as typeof error.config & {
      _retry?: boolean
    }

    // Handle 401 Unauthorized - try to refresh token
    if (error.response?.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        // If already refreshing, queue this request
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        })
          .then((token) => {
            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${token}`
            }
            return api(originalRequest)
          })
          .catch((err) => {
            return Promise.reject(err)
          })
      }

      originalRequest._retry = true
      isRefreshing = true

      const refreshToken = getRefreshToken()

      if (!refreshToken) {
        // No refresh token, redirect to login
        clearTokens()
        window.location.href = '/admin/login'
        return Promise.reject(error)
      }

      try {
        // Attempt to refresh the token
        const response = await axios.post(`${API_BASE_URL}/api/v1/admin/refresh`, {
          refresh_token: refreshToken,
        })

        const { access_token, refresh_token: newRefreshToken, user, expires_in } = response.data

        // Update tokens
        setTokens(
          { access_token, refresh_token: newRefreshToken, expires_in },
          user as AdminUser
        )

        // Update the failed request and retry
        if (originalRequest.headers) {
          originalRequest.headers.Authorization = `Bearer ${access_token}`
        }

        processQueue(null, access_token)
        isRefreshing = false

        return api(originalRequest)
      } catch (refreshError) {
        processQueue(refreshError as Error, null)
        isRefreshing = false

        // Refresh failed, logout user
        clearTokens()
        window.location.href = '/admin/login'

        return Promise.reject(refreshError)
      }
    }

    return Promise.reject(error)
  }
)

// API Types
export interface User {
  id: string
  email: string
  email_verified: boolean
  role: string
  metadata: Record<string, unknown> | null
  created_at: string
  updated_at: string
}

export interface SignInRequest {
  email: string
  password: string
}

export interface SignInResponse {
  user: User
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface SignUpRequest {
  email: string
  password: string
  metadata?: Record<string, unknown>
}

export interface SignUpResponse {
  user: User
  access_token: string
  refresh_token: string
  expires_in: number
}

// Auth API methods
export const authApi = {
  signIn: async (data: SignInRequest): Promise<SignInResponse> => {
    const response = await api.post<SignInResponse>('/api/v1/auth/signin', data)
    return response.data
  },

  signUp: async (data: SignUpRequest): Promise<SignUpResponse> => {
    const response = await api.post<SignUpResponse>('/api/v1/auth/signup', data)
    return response.data
  },

  signOut: async (): Promise<void> => {
    await api.post('/api/v1/auth/signout')
  },

  getUser: async (): Promise<User> => {
    const response = await api.get<User>('/api/v1/auth/user')
    return response.data
  },

  updateUser: async (
    data: Partial<Pick<User, 'email' | 'metadata'>>
  ): Promise<User> => {
    const response = await api.patch<User>('/api/v1/auth/user', data)
    return response.data
  },
}

// Database API methods
export interface TableInfo {
  schema: string
  name: string
  columns: Array<{
    name: string
    data_type: string
    is_nullable: boolean
    default_value: string | null
    is_primary_key: boolean
    is_foreign_key: boolean
    is_unique: boolean
    max_length: number | null
    position: number
  }>
  primary_key: string[]
  foreign_keys: unknown
  indexes: unknown
  rls_enabled: boolean
}

export const databaseApi = {
  getSchemas: async (): Promise<string[]> => {
    const response = await api.get<string[]>('/api/v1/admin/schemas')
    return response.data
  },

  getTables: async (): Promise<string[]> => {
    const response = await api.get<TableInfo[]>('/api/v1/admin/tables')
    // Convert table objects to "schema.table" strings
    return response.data.map((t) => `${t.schema}.${t.name}`)
  },

  getTableData: async <T = unknown>(
    table: string,
    params?: {
      limit?: number
      offset?: number
      order?: string
      select?: string
      filter?: Record<string, unknown>
    }
  ): Promise<T[]> => {
    const response = await api.get<T[]>(`/api/rest/${table}`, { params })
    return response.data
  },

  createRecord: async <T = unknown>(
    table: string,
    data: Record<string, unknown>
  ): Promise<T> => {
    const response = await api.post<T>(`/api/rest/${table}`, data)
    return response.data
  },

  updateRecord: async <T = unknown>(
    table: string,
    id: string | number,
    data: Record<string, unknown>
  ): Promise<T> => {
    const response = await api.patch<T>(`/api/rest/${table}/${id}`, data)
    return response.data
  },

  deleteRecord: async (table: string, id: string | number): Promise<void> => {
    await api.delete(`/api/rest/${table}/${id}`)
  },

  getTableSchema: async (
    table: string
  ): Promise<{
    columns: Array<{
      name: string
      type: string
      nullable: boolean
      default: string | null
      primary_key: boolean
    }>
  }> => {
    const response = await api.get(`/api/admin/tables/${table}/schema`)
    return response.data
  },
}

// Health check
export const healthApi = {
  check: async (): Promise<{
    status: string
    services: { database: boolean; realtime: boolean }
    timestamp: string
  }> => {
    const response = await api.get('/health')
    return response.data
  },
}

// User Management API Types
export interface EnrichedUser {
  id: string
  email: string
  email_verified: boolean
  role: string
  provider: 'email' | 'invite_pending' | 'magic_link'
  active_sessions: number
  last_sign_in: string | null
  metadata: Record<string, unknown> | null
  created_at: string
  updated_at: string
}

export interface InviteUserRequest {
  email: string
  role: string
  password?: string // Optional: if provided, use this instead of auto-generating
}

export interface InviteUserResponse {
  user: User
  temporary_password?: string
  email_sent: boolean
  message: string
}

// User Management API methods (admin only)
export const userManagementApi = {
  listUsers: async (): Promise<EnrichedUser[]> => {
    const response = await api.get<EnrichedUser[]>('/api/v1/admin/users')
    return response.data
  },

  inviteUser: async (data: InviteUserRequest): Promise<InviteUserResponse> => {
    const response = await api.post<InviteUserResponse>(
      '/api/v1/admin/users/invite',
      data
    )
    return response.data
  },

  deleteUser: async (userId: string): Promise<{ message: string }> => {
    const response = await api.delete<{ message: string }>(
      `/api/v1/admin/users/${userId}`
    )
    return response.data
  },

  updateUserRole: async (userId: string, role: string): Promise<User> => {
    const response = await api.patch<User>(`/api/v1/admin/users/${userId}/role`, {
      role,
    })
    return response.data
  },

  resetUserPassword: async (userId: string): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>(
      `/api/v1/admin/users/${userId}/reset-password`
    )
    return response.data
  },
}

export default api

// Admin Authentication API
export const adminAuthAPI = {
  // Check if initial setup is needed
  getSetupStatus: async (): Promise<{ needs_setup: boolean; has_admin: boolean }> => {
    const response = await axios.get(`${API_BASE_URL}/api/v1/admin/setup/status`)
    return response.data
  },

  // Initial setup - create first admin user
  initialSetup: async (data: {
    email: string
    password: string
    name: string
  }): Promise<{ user: AdminUser; access_token: string; refresh_token: string; expires_in: number }> => {
    const response = await axios.post(`${API_BASE_URL}/api/v1/admin/setup`, data)
    return response.data
  },

  // Admin login
  login: async (credentials: {
    email: string
    password: string
  }): Promise<{ user: AdminUser; access_token: string; refresh_token: string; expires_in: number }> => {
    const response = await axios.post(`${API_BASE_URL}/api/v1/admin/login`, credentials)
    return response.data
  },

  // Admin logout
  logout: async (): Promise<{ message: string }> => {
    const response = await api.post('/api/v1/admin/logout')
    return response.data
  },

  // Get current admin user
  me: async (): Promise<{ user: AdminUser }> => {
    const response = await api.get('/api/v1/admin/me')
    return response.data
  },
}

// Dashboard user types
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

export interface DashboardSignupRequest {
  email: string
  password: string
  full_name: string
}

export interface DashboardLoginRequest {
  email: string
  password: string
}

export interface DashboardLoginResponse {
  access_token: string
  user: DashboardUser
  requires_2fa?: boolean
  user_id?: string
}

export interface UpdateProfileRequest {
  full_name: string
  avatar_url?: string | null
}

export interface ChangePasswordRequest {
  current_password: string
  new_password: string
}

export interface DeleteAccountRequest {
  password: string
}

export interface Setup2FAResponse {
  secret: string
  qr_url: string
}

export interface Enable2FARequest {
  code: string
}

export interface Enable2FAResponse {
  message: string
  backup_codes: string[]
}

export interface Verify2FARequest {
  user_id: string
  code: string
}

export interface Disable2FARequest {
  password: string
}

// Dashboard Auth API methods
export const dashboardAuthAPI = {
  // Signup for dashboard
  signup: async (data: DashboardSignupRequest): Promise<{ user: DashboardUser; message: string }> => {
    const response = await axios.post(`${API_BASE_URL}/dashboard/auth/signup`, data)
    return response.data
  },

  // Login to dashboard
  login: async (credentials: DashboardLoginRequest): Promise<DashboardLoginResponse> => {
    const response = await axios.post(`${API_BASE_URL}/dashboard/auth/login`, credentials)
    return response.data
  },

  // Get current dashboard user
  me: async (): Promise<DashboardUser> => {
    const response = await api.get('/dashboard/auth/me')
    return response.data
  },

  // Update profile
  updateProfile: async (data: UpdateProfileRequest): Promise<DashboardUser> => {
    const response = await api.put('/dashboard/auth/profile', data)
    return response.data
  },

  // Change password
  changePassword: async (data: ChangePasswordRequest): Promise<{ message: string }> => {
    const response = await api.post('/dashboard/auth/password/change', data)
    return response.data
  },

  // Delete account
  deleteAccount: async (data: DeleteAccountRequest): Promise<{ message: string }> => {
    const response = await api.delete('/dashboard/auth/account', { data })
    return response.data
  },

  // Setup 2FA
  setup2FA: async (): Promise<Setup2FAResponse> => {
    const response = await api.post('/dashboard/auth/2fa/setup')
    return response.data
  },

  // Enable 2FA
  enable2FA: async (data: Enable2FARequest): Promise<Enable2FAResponse> => {
    const response = await api.post('/dashboard/auth/2fa/enable', data)
    return response.data
  },

  // Verify 2FA code during login
  verify2FA: async (data: Verify2FARequest): Promise<DashboardLoginResponse> => {
    const response = await axios.post(`${API_BASE_URL}/dashboard/auth/2fa/verify`, data)
    return response.data
  },

  // Disable 2FA
  disable2FA: async (data: Disable2FARequest): Promise<{ message: string }> => {
    const response = await api.post('/dashboard/auth/2fa/disable', data)
    return response.data
  },
}
