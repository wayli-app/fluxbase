import { describe, it, expect, beforeEach, vi } from 'vitest'
import { ImpersonationManager } from './impersonation'
import type { FluxbaseFetch } from './fetch'
import type {
  StartImpersonationResponse,
  StopImpersonationResponse,
  GetImpersonationResponse,
  ListImpersonationSessionsResponse,
  ImpersonationSession,
  ImpersonationTargetUser,
} from './types'

describe('ImpersonationManager', () => {
  let manager: ImpersonationManager
  let mockFetch: any

  const mockTargetUser: ImpersonationTargetUser = {
    id: 'user-123',
    email: 'user@example.com',
    role: 'user',
  }

  const mockSession: ImpersonationSession = {
    id: 'session-123',
    admin_user_id: 'admin-456',
    target_user_id: 'user-123',
    impersonation_type: 'user',
    target_role: 'user',
    reason: 'Support ticket #1234',
    started_at: '2024-01-15T10:30:00Z',
    ended_at: null,
    is_active: true,
    ip_address: '192.168.1.1',
    user_agent: 'Mozilla/5.0...',
  }

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    }
    manager = new ImpersonationManager(mockFetch as unknown as FluxbaseFetch)
  })

  describe('impersonateUser', () => {
    it('should start impersonating a specific user', async () => {
      const mockResponse: StartImpersonationResponse = {
        session: mockSession,
        target_user: mockTargetUser,
        access_token: 'eyJhbGc...',
        refresh_token: 'eyJhbGc...',
        expires_in: 900,
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      const result = await manager.impersonateUser({
        target_user_id: 'user-123',
        reason: 'Support ticket #1234',
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/auth/impersonate', {
        target_user_id: 'user-123',
        reason: 'Support ticket #1234',
      })
      expect(result.session.id).toBe('session-123')
      expect(result.target_user?.email).toBe('user@example.com')
      expect(result.access_token).toBeDefined()
      expect(result.expires_in).toBe(900)
    })

    it('should include reason in the request', async () => {
      const mockResponse: StartImpersonationResponse = {
        session: mockSession,
        target_user: mockTargetUser,
        access_token: 'token',
        refresh_token: 'refresh',
        expires_in: 900,
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      await manager.impersonateUser({
        target_user_id: 'user-123',
        reason: 'Debugging data access issue',
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/auth/impersonate', {
        target_user_id: 'user-123',
        reason: 'Debugging data access issue',
      })
    })

    it('should handle errors when user not found', async () => {
      vi.mocked(mockFetch.post).mockRejectedValue(new Error('User not found'))

      await expect(
        manager.impersonateUser({
          target_user_id: 'nonexistent',
          reason: 'Testing',
        })
      ).rejects.toThrow('User not found')
    })

    it('should handle errors when self-impersonation', async () => {
      vi.mocked(mockFetch.post).mockRejectedValue(new Error('Cannot impersonate yourself'))

      await expect(
        manager.impersonateUser({
          target_user_id: 'admin-456',
          reason: 'Testing',
        })
      ).rejects.toThrow('Cannot impersonate yourself')
    })
  })

  describe('impersonateAnon', () => {
    it('should start impersonating anonymous user', async () => {
      const anonSession: ImpersonationSession = {
        ...mockSession,
        target_user_id: null,
        impersonation_type: 'anon',
        target_role: 'anon',
        reason: 'Testing public data access',
      }

      const mockResponse: StartImpersonationResponse = {
        session: anonSession,
        target_user: null,
        access_token: 'anon_token',
        refresh_token: 'anon_refresh',
        expires_in: 900,
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      const result = await manager.impersonateAnon({
        reason: 'Testing public data access',
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/auth/impersonate/anon', {
        reason: 'Testing public data access',
      })
      expect(result.session.impersonation_type).toBe('anon')
      expect(result.session.target_role).toBe('anon')
      expect(result.target_user).toBeNull()
    })

    it('should require a reason', async () => {
      const mockResponse: StartImpersonationResponse = {
        session: mockSession,
        target_user: null,
        access_token: 'token',
        refresh_token: 'refresh',
        expires_in: 900,
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      await manager.impersonateAnon({
        reason: 'Required reason text',
      })

      expect(mockFetch.post).toHaveBeenCalledWith(
        '/api/v1/auth/impersonate/anon',
        expect.objectContaining({
          reason: 'Required reason text',
        })
      )
    })
  })

  describe('impersonateService', () => {
    it('should start impersonating with service role', async () => {
      const serviceSession: ImpersonationSession = {
        ...mockSession,
        target_user_id: null,
        impersonation_type: 'service',
        target_role: 'service',
        reason: 'Administrative query',
      }

      const mockResponse: StartImpersonationResponse = {
        session: serviceSession,
        target_user: null,
        access_token: 'service_token',
        refresh_token: 'service_refresh',
        expires_in: 900,
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      const result = await manager.impersonateService({
        reason: 'Administrative query',
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/auth/impersonate/service', {
        reason: 'Administrative query',
      })
      expect(result.session.impersonation_type).toBe('service')
      expect(result.session.target_role).toBe('service')
      expect(result.target_user).toBeNull()
    })

    it('should provide elevated permissions', async () => {
      const mockResponse: StartImpersonationResponse = {
        session: {
          ...mockSession,
          impersonation_type: 'service',
          target_role: 'service',
        },
        target_user: null,
        access_token: 'service_token',
        refresh_token: 'refresh',
        expires_in: 900,
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      const result = await manager.impersonateService({
        reason: 'Admin operations',
      })

      expect(result.session.target_role).toBe('service')
    })
  })

  describe('stop', () => {
    it('should stop impersonation session', async () => {
      const mockResponse: StopImpersonationResponse = {
        success: true,
        message: 'Impersonation session ended',
      }

      vi.mocked(mockFetch.delete).mockResolvedValue(mockResponse)

      const result = await manager.stop()

      expect(mockFetch.delete).toHaveBeenCalledWith('/api/v1/auth/impersonate')
      expect(result.success).toBe(true)
      expect(result.message).toBe('Impersonation session ended')
    })

    it('should handle errors when no active session', async () => {
      vi.mocked(mockFetch.delete).mockRejectedValue(new Error('No active impersonation session'))

      await expect(manager.stop()).rejects.toThrow('No active impersonation session')
    })
  })

  describe('getCurrent', () => {
    it('should get current impersonation session', async () => {
      const mockResponse: GetImpersonationResponse = {
        session: mockSession,
        target_user: mockTargetUser,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      const result = await manager.getCurrent()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/auth/impersonate')
      expect(result.session?.id).toBe('session-123')
      expect(result.target_user?.email).toBe('user@example.com')
    })

    it('should return null when no active session', async () => {
      const mockResponse: GetImpersonationResponse = {
        session: null,
        target_user: null,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      const result = await manager.getCurrent()

      expect(result.session).toBeNull()
      expect(result.target_user).toBeNull()
    })

    it('should include session metadata', async () => {
      const mockResponse: GetImpersonationResponse = {
        session: mockSession,
        target_user: mockTargetUser,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      const result = await manager.getCurrent()

      expect(result.session?.reason).toBe('Support ticket #1234')
      expect(result.session?.started_at).toBeDefined()
      expect(result.session?.is_active).toBe(true)
    })
  })

  describe('listSessions', () => {
    it('should list all impersonation sessions', async () => {
      const mockSessions: ImpersonationSession[] = [
        mockSession,
        {
          ...mockSession,
          id: 'session-456',
          target_user_id: 'user-789',
          reason: 'Different reason',
          ended_at: '2024-01-15T11:00:00Z',
          is_active: false,
        },
      ]

      const mockResponse: ListImpersonationSessionsResponse = {
        sessions: mockSessions,
        total: 2,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      const result = await manager.listSessions()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/auth/impersonate/sessions')
      expect(result.sessions).toHaveLength(2)
      expect(result.total).toBe(2)
    })

    it('should support pagination with limit and offset', async () => {
      const mockResponse: ListImpersonationSessionsResponse = {
        sessions: [mockSession],
        total: 100,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      await manager.listSessions({
        limit: 50,
        offset: 25,
      })

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/auth/impersonate/sessions?limit=50&offset=25')
    })

    it('should filter by admin user ID', async () => {
      const mockResponse: ListImpersonationSessionsResponse = {
        sessions: [mockSession],
        total: 1,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      await manager.listSessions({
        admin_user_id: 'admin-456',
      })

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/auth/impersonate/sessions?admin_user_id=admin-456')
    })

    it('should filter by target user ID', async () => {
      const mockResponse: ListImpersonationSessionsResponse = {
        sessions: [mockSession],
        total: 1,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      await manager.listSessions({
        target_user_id: 'user-123',
      })

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/auth/impersonate/sessions?target_user_id=user-123')
    })

    it('should filter by impersonation type', async () => {
      const mockResponse: ListImpersonationSessionsResponse = {
        sessions: [mockSession],
        total: 1,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      await manager.listSessions({
        impersonation_type: 'user',
      })

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/auth/impersonate/sessions?impersonation_type=user')
    })

    it('should filter by active status', async () => {
      const mockResponse: ListImpersonationSessionsResponse = {
        sessions: [mockSession],
        total: 1,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      await manager.listSessions({
        is_active: true,
      })

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/auth/impersonate/sessions?is_active=true')
    })

    it('should support multiple filters', async () => {
      const mockResponse: ListImpersonationSessionsResponse = {
        sessions: [],
        total: 0,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      await manager.listSessions({
        admin_user_id: 'admin-456',
        impersonation_type: 'user',
        is_active: true,
        limit: 25,
        offset: 0,
      })

      expect(mockFetch.get).toHaveBeenCalledWith(
        '/api/v1/auth/impersonate/sessions?limit=25&offset=0&admin_user_id=admin-456&impersonation_type=user&is_active=true'
      )
    })

    it('should handle empty sessions list', async () => {
      const mockResponse: ListImpersonationSessionsResponse = {
        sessions: [],
        total: 0,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      const result = await manager.listSessions()

      expect(result.sessions).toEqual([])
      expect(result.total).toBe(0)
    })
  })
})
