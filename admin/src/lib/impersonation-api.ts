import { apiClient } from './api'
import type {
  ImpersonationSession,
  ImpersonatedUser,
} from '../stores/impersonation-store'

export interface StartImpersonationRequest {
  target_user_id?: string
  reason: string
}

export interface StartImpersonationResponse {
  session: ImpersonationSession
  target_user: ImpersonatedUser
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface ListUsersResponse {
  users: Array<{
    id: string
    email: string
    created_at: string
    role: string
  }>
  total: number
}

export const impersonationApi = {
  /**
   * Start impersonating a specific user
   */
  async startUserImpersonation(
    targetUserId: string,
    reason: string
  ): Promise<StartImpersonationResponse> {
    const response = await apiClient.post<StartImpersonationResponse>(
      '/api/v1/auth/impersonate',
      {
        target_user_id: targetUserId,
        reason,
      }
    )
    return response.data
  },

  /**
   * Start impersonating as anonymous user (anon key)
   */
  async startAnonImpersonation(
    reason: string
  ): Promise<StartImpersonationResponse> {
    const response = await apiClient.post<StartImpersonationResponse>(
      '/api/v1/auth/impersonate/anon',
      { reason }
    )
    return response.data
  },

  /**
   * Start impersonating with service role
   */
  async startServiceImpersonation(
    reason: string
  ): Promise<StartImpersonationResponse> {
    const response = await apiClient.post<StartImpersonationResponse>(
      '/api/v1/auth/impersonate/service',
      { reason }
    )
    return response.data
  },

  /**
   * Stop the active impersonation session
   */
  async stopImpersonation(): Promise<void> {
    await apiClient.delete('/api/v1/auth/impersonate')
  },

  /**
   * Get the currently active impersonation session
   */
  async getActiveSession(): Promise<{
    session: ImpersonationSession
    target_user: ImpersonatedUser
  } | null> {
    try {
      const response = await apiClient.get('/api/v1/auth/impersonate')
      return response.data
    } catch (error: unknown) {
      if (error && typeof error === 'object' && 'response' in error) {
        const axiosError = error as { response?: { status?: number } }
        if (axiosError.response?.status === 404) {
          return null
        }
      }
      throw error
    }
  },

  /**
   * List all impersonation sessions (audit trail)
   */
  async listSessions(
    limit = 50,
    offset = 0
  ): Promise<{
    sessions: ImpersonationSession[]
    total: number
  }> {
    const response = await apiClient.get('/api/v1/auth/impersonate/sessions', {
      params: { limit, offset },
    })
    return response.data
  },

  /**
   * List users available for impersonation (non-admin users)
   */
  async listUsers(
    search?: string,
    limit = 20
  ): Promise<ListUsersResponse> {
    const response = await apiClient.get<ListUsersResponse>(
      '/api/v1/admin/users',
      {
        params: {
          exclude_admins: true,
          search,
          limit,
        },
      }
    )
    return response.data
  },
}
