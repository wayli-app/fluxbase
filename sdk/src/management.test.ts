import { describe, it, expect, beforeEach, vi } from 'vitest'
import { FluxbaseManagement, APIKeysManager, WebhooksManager, InvitationsManager } from './management'
import { FluxbaseFetch } from './fetch'
import type {
  APIKey,
  CreateAPIKeyResponse,
  ListAPIKeysResponse,
  Webhook,
  ListWebhooksResponse,
  TestWebhookResponse,
  ListWebhookDeliveriesResponse,
  CreateInvitationResponse,
  ListInvitationsResponse,
  ValidateInvitationResponse,
  AcceptInvitationResponse,
} from './types'

// Mock FluxbaseFetch
vi.mock('./fetch')

describe('APIKeysManager', () => {
  let manager: APIKeysManager
  let mockFetch: any

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    }

    manager = new APIKeysManager(mockFetch as unknown as FluxbaseFetch)
  })

  describe('create()', () => {
    it('should create a new API key', async () => {
      const response: CreateAPIKeyResponse = {
        api_key: {
          id: 'key-123',
          name: 'Production Key',
          description: 'Key for production service',
          key_prefix: 'fb_live_abc',
          scopes: ['read:users', 'write:users'],
          rate_limit_per_minute: 100,
          created_at: '2024-01-26T10:00:00Z',
          user_id: 'user-123',
        },
        key: 'fb_live_abc123def456ghi789jkl',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(response)

      const result = await manager.create({
        name: 'Production Key',
        description: 'Key for production service',
        scopes: ['read:users', 'write:users'],
        rate_limit_per_minute: 100,
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/api-keys', {
        name: 'Production Key',
        description: 'Key for production service',
        scopes: ['read:users', 'write:users'],
        rate_limit_per_minute: 100,
      })

      expect(result.key).toBe('fb_live_abc123def456ghi789jkl')
      expect(result.api_key.name).toBe('Production Key')
    })

    it('should create API key with expiration', async () => {
      const response: CreateAPIKeyResponse = {
        api_key: {
          id: 'key-123',
          name: 'Temporary Key',
          key_prefix: 'fb_test_xyz',
          scopes: ['read:users'],
          rate_limit_per_minute: 50,
          created_at: '2024-01-26T10:00:00Z',
          expires_at: '2025-12-31T23:59:59Z',
          user_id: 'user-123',
        },
        key: 'fb_test_xyz123',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(response)

      const result = await manager.create({
        name: 'Temporary Key',
        scopes: ['read:users'],
        rate_limit_per_minute: 50,
        expires_at: '2025-12-31T23:59:59Z',
      })

      expect(result.api_key.expires_at).toBe('2025-12-31T23:59:59Z')
    })
  })

  describe('list()', () => {
    it('should list all API keys', async () => {
      const response: ListAPIKeysResponse = {
        api_keys: [
          {
            id: 'key-1',
            name: 'Key 1',
            key_prefix: 'fb_live_aaa',
            scopes: ['read:users'],
            rate_limit_per_minute: 100,
            created_at: '2024-01-26T10:00:00Z',
            user_id: 'user-123',
          },
          {
            id: 'key-2',
            name: 'Key 2',
            key_prefix: 'fb_live_bbb',
            scopes: ['read:users', 'write:users'],
            rate_limit_per_minute: 200,
            created_at: '2024-01-26T11:00:00Z',
            user_id: 'user-123',
          },
        ],
        total: 2,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(response)

      const result = await manager.list()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/api-keys')
      expect(result.api_keys).toHaveLength(2)
      expect(result.total).toBe(2)
    })
  })

  describe('get()', () => {
    it('should get a specific API key', async () => {
      const apiKey: APIKey = {
        id: 'key-123',
        name: 'Production Key',
        key_prefix: 'fb_live_abc',
        scopes: ['read:users'],
        rate_limit_per_minute: 100,
        created_at: '2024-01-26T10:00:00Z',
        last_used_at: '2024-01-27T15:30:00Z',
        user_id: 'user-123',
      }

      vi.mocked(mockFetch.get).mockResolvedValue(apiKey)

      const result = await manager.get('key-123')

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/api-keys/key-123')
      expect(result.id).toBe('key-123')
      expect(result.last_used_at).toBe('2024-01-27T15:30:00Z')
    })
  })

  describe('update()', () => {
    it('should update an API key', async () => {
      const updated: APIKey = {
        id: 'key-123',
        name: 'Updated Key Name',
        key_prefix: 'fb_live_abc',
        scopes: ['read:users', 'write:users'],
        rate_limit_per_minute: 200,
        created_at: '2024-01-26T10:00:00Z',
        updated_at: '2024-01-27T10:00:00Z',
        user_id: 'user-123',
      }

      vi.mocked(mockFetch.patch).mockResolvedValue(updated)

      const result = await manager.update('key-123', {
        name: 'Updated Key Name',
        rate_limit_per_minute: 200,
      })

      expect(mockFetch.patch).toHaveBeenCalledWith('/api/v1/api-keys/key-123', {
        name: 'Updated Key Name',
        rate_limit_per_minute: 200,
      })

      expect(result.name).toBe('Updated Key Name')
      expect(result.rate_limit_per_minute).toBe(200)
    })
  })

  describe('revoke()', () => {
    it('should revoke an API key', async () => {
      vi.mocked(mockFetch.post).mockResolvedValue({ message: 'API key revoked successfully' })

      const result = await manager.revoke('key-123')

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/api-keys/key-123/revoke', {})
      expect(result.message).toBe('API key revoked successfully')
    })
  })

  describe('delete()', () => {
    it('should delete an API key', async () => {
      vi.mocked(mockFetch.delete).mockResolvedValue({ message: 'API key deleted successfully' })

      const result = await manager.delete('key-123')

      expect(mockFetch.delete).toHaveBeenCalledWith('/api/v1/api-keys/key-123')
      expect(result.message).toBe('API key deleted successfully')
    })
  })
})

describe('WebhooksManager', () => {
  let manager: WebhooksManager
  let mockFetch: any

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    }

    manager = new WebhooksManager(mockFetch as unknown as FluxbaseFetch)
  })

  describe('create()', () => {
    it('should create a new webhook', async () => {
      const webhook: Webhook = {
        id: 'webhook-123',
        url: 'https://myapp.com/webhook',
        events: ['user.created', 'user.updated'],
        secret: 'secret-123',
        description: 'User events webhook',
        is_active: true,
        created_at: '2024-01-26T10:00:00Z',
        user_id: 'user-123',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(webhook)

      const result = await manager.create({
        url: 'https://myapp.com/webhook',
        events: ['user.created', 'user.updated'],
        secret: 'secret-123',
        description: 'User events webhook',
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/webhooks', {
        url: 'https://myapp.com/webhook',
        events: ['user.created', 'user.updated'],
        secret: 'secret-123',
        description: 'User events webhook',
      })

      expect(result.url).toBe('https://myapp.com/webhook')
      expect(result.events).toEqual(['user.created', 'user.updated'])
    })
  })

  describe('list()', () => {
    it('should list all webhooks', async () => {
      const response: ListWebhooksResponse = {
        webhooks: [
          {
            id: 'webhook-1',
            url: 'https://app1.com/webhook',
            events: ['user.created'],
            is_active: true,
            created_at: '2024-01-26T10:00:00Z',
            user_id: 'user-123',
          },
          {
            id: 'webhook-2',
            url: 'https://app2.com/webhook',
            events: ['user.deleted'],
            is_active: false,
            created_at: '2024-01-26T11:00:00Z',
            user_id: 'user-123',
          },
        ],
        total: 2,
      }

      vi.mocked(mockFetch.get).mockResolvedValue(response)

      const result = await manager.list()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/webhooks')
      expect(result.webhooks).toHaveLength(2)
      expect(result.total).toBe(2)
    })
  })

  describe('get()', () => {
    it('should get a specific webhook', async () => {
      const webhook: Webhook = {
        id: 'webhook-123',
        url: 'https://myapp.com/webhook',
        events: ['user.created'],
        is_active: true,
        created_at: '2024-01-26T10:00:00Z',
        user_id: 'user-123',
      }

      vi.mocked(mockFetch.get).mockResolvedValue(webhook)

      const result = await manager.get('webhook-123')

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/webhooks/webhook-123')
      expect(result.id).toBe('webhook-123')
    })
  })

  describe('update()', () => {
    it('should update a webhook', async () => {
      const updated: Webhook = {
        id: 'webhook-123',
        url: 'https://myapp.com/webhook',
        events: ['user.created', 'user.deleted'],
        is_active: false,
        created_at: '2024-01-26T10:00:00Z',
        updated_at: '2024-01-27T10:00:00Z',
        user_id: 'user-123',
      }

      vi.mocked(mockFetch.patch).mockResolvedValue(updated)

      const result = await manager.update('webhook-123', {
        events: ['user.created', 'user.deleted'],
        is_active: false,
      })

      expect(mockFetch.patch).toHaveBeenCalledWith('/api/v1/webhooks/webhook-123', {
        events: ['user.created', 'user.deleted'],
        is_active: false,
      })

      expect(result.events).toEqual(['user.created', 'user.deleted'])
      expect(result.is_active).toBe(false)
    })
  })

  describe('delete()', () => {
    it('should delete a webhook', async () => {
      vi.mocked(mockFetch.delete).mockResolvedValue({ message: 'Webhook deleted successfully' })

      const result = await manager.delete('webhook-123')

      expect(mockFetch.delete).toHaveBeenCalledWith('/api/v1/webhooks/webhook-123')
      expect(result.message).toBe('Webhook deleted successfully')
    })
  })

  describe('test()', () => {
    it('should test a webhook successfully', async () => {
      const response: TestWebhookResponse = {
        success: true,
        status_code: 200,
        response_body: '{"status":"ok"}',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(response)

      const result = await manager.test('webhook-123')

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/webhooks/webhook-123/test', {})
      expect(result.success).toBe(true)
      expect(result.status_code).toBe(200)
    })

    it('should handle webhook test failure', async () => {
      const response: TestWebhookResponse = {
        success: false,
        error: 'Connection timeout',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(response)

      const result = await manager.test('webhook-123')

      expect(result.success).toBe(false)
      expect(result.error).toBe('Connection timeout')
    })
  })

  describe('listDeliveries()', () => {
    it('should list webhook deliveries', async () => {
      const response: ListWebhookDeliveriesResponse = {
        deliveries: [
          {
            id: 'delivery-1',
            webhook_id: 'webhook-123',
            event: 'user.created',
            payload: { user_id: 'user-456' },
            status_code: 200,
            response_body: '{"status":"ok"}',
            created_at: '2024-01-26T10:00:00Z',
            delivered_at: '2024-01-26T10:00:01Z',
          },
          {
            id: 'delivery-2',
            webhook_id: 'webhook-123',
            event: 'user.updated',
            payload: { user_id: 'user-456' },
            status_code: 500,
            error: 'Internal server error',
            created_at: '2024-01-26T11:00:00Z',
          },
        ],
      }

      vi.mocked(mockFetch.get).mockResolvedValue(response)

      const result = await manager.listDeliveries('webhook-123', 100)

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/webhooks/webhook-123/deliveries?limit=100')
      expect(result.deliveries).toHaveLength(2)
    })

    it('should use default limit', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue({ deliveries: [] })

      await manager.listDeliveries('webhook-123')

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/webhooks/webhook-123/deliveries?limit=50')
    })
  })
})

describe('InvitationsManager', () => {
  let manager: InvitationsManager
  let mockFetch: any

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      delete: vi.fn(),
    }

    manager = new InvitationsManager(mockFetch as unknown as FluxbaseFetch)
  })

  describe('create()', () => {
    it('should create a new invitation', async () => {
      const response: CreateInvitationResponse = {
        invitation: {
          id: 'invite-123',
          email: 'newuser@example.com',
          role: 'dashboard_user',
          invited_by: 'admin-123',
          expires_at: '2024-02-02T10:00:00Z',
          created_at: '2024-01-26T10:00:00Z',
        },
        invite_link: 'https://app.example.com/invite/token-abc123',
        email_sent: false,
        email_status: 'Email notification not configured',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(response)

      const result = await manager.create({
        email: 'newuser@example.com',
        role: 'dashboard_user',
        expiry_duration: 604800, // 7 days
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/admin/invitations', {
        email: 'newuser@example.com',
        role: 'dashboard_user',
        expiry_duration: 604800,
      })

      expect(result.invitation.email).toBe('newuser@example.com')
      expect(result.invite_link).toContain('/invite/')
    })

    it('should create invitation for dashboard_admin', async () => {
      const response: CreateInvitationResponse = {
        invitation: {
          id: 'invite-123',
          email: 'admin@example.com',
          role: 'dashboard_admin',
          invited_by: 'admin-123',
          expires_at: '2024-02-02T10:00:00Z',
          created_at: '2024-01-26T10:00:00Z',
        },
        invite_link: 'https://app.example.com/invite/token-xyz',
        email_sent: true,
        email_status: 'Email sent successfully',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(response)

      const result = await manager.create({
        email: 'admin@example.com',
        role: 'dashboard_admin',
      })

      expect(result.invitation.role).toBe('dashboard_admin')
      expect(result.email_sent).toBe(true)
    })
  })

  describe('list()', () => {
    it('should list pending invitations', async () => {
      const response: ListInvitationsResponse = {
        invitations: [
          {
            id: 'invite-1',
            email: 'user1@example.com',
            role: 'dashboard_user',
            invited_by: 'admin-123',
            expires_at: '2024-02-02T10:00:00Z',
            created_at: '2024-01-26T10:00:00Z',
          },
          {
            id: 'invite-2',
            email: 'user2@example.com',
            role: 'dashboard_user',
            invited_by: 'admin-123',
            expires_at: '2024-02-03T10:00:00Z',
            created_at: '2024-01-26T11:00:00Z',
          },
        ],
      }

      vi.mocked(mockFetch.get).mockResolvedValue(response)

      const result = await manager.list({
        include_accepted: false,
        include_expired: false,
      })

      expect(mockFetch.get).toHaveBeenCalledWith(
        '/api/v1/admin/invitations?include_accepted=false&include_expired=false',
      )
      expect(result.invitations).toHaveLength(2)
    })

    it('should list all invitations including accepted and expired', async () => {
      const response: ListInvitationsResponse = {
        invitations: [
          {
            id: 'invite-1',
            email: 'user1@example.com',
            role: 'dashboard_user',
            invited_by: 'admin-123',
            expires_at: '2024-02-02T10:00:00Z',
            created_at: '2024-01-26T10:00:00Z',
            accepted_at: '2024-01-27T10:00:00Z',
          },
        ],
      }

      vi.mocked(mockFetch.get).mockResolvedValue(response)

      const result = await manager.list({
        include_accepted: true,
        include_expired: true,
      })

      expect(mockFetch.get).toHaveBeenCalledWith(
        '/api/v1/admin/invitations?include_accepted=true&include_expired=true',
      )
      expect(result.invitations[0].accepted_at).toBeDefined()
    })

    it('should list with default options', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue({ invitations: [] })

      await manager.list()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/admin/invitations')
    })
  })

  describe('validate()', () => {
    it('should validate a valid invitation token', async () => {
      const response: ValidateInvitationResponse = {
        valid: true,
        invitation: {
          id: 'invite-123',
          email: 'user@example.com',
          role: 'dashboard_user',
          invited_by: 'admin-123',
          expires_at: '2024-02-02T10:00:00Z',
          created_at: '2024-01-26T10:00:00Z',
        },
      }

      vi.mocked(mockFetch.get).mockResolvedValue(response)

      const result = await manager.validate('token-abc123')

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/invitations/token-abc123/validate')
      expect(result.valid).toBe(true)
      expect(result.invitation?.email).toBe('user@example.com')
    })

    it('should handle invalid invitation token', async () => {
      const response: ValidateInvitationResponse = {
        valid: false,
        error: 'Invitation has expired',
      }

      vi.mocked(mockFetch.get).mockResolvedValue(response)

      const result = await manager.validate('expired-token')

      expect(result.valid).toBe(false)
      expect(result.error).toBe('Invitation has expired')
    })
  })

  describe('accept()', () => {
    it('should accept an invitation and create user', async () => {
      const response: AcceptInvitationResponse = {
        user: {
          id: 'user-123',
          email: 'newuser@example.com',
          name: 'New User',
          role: 'dashboard_user',
          email_verified: true,
          created_at: '2024-01-26T10:00:00Z',
          updated_at: '2024-01-26T10:00:00Z',
        },
        access_token: 'access-token-123',
        refresh_token: 'refresh-token-123',
        expires_in: 900,
      }

      vi.mocked(mockFetch.post).mockResolvedValue(response)

      const result = await manager.accept('token-abc123', {
        password: 'SecurePassword123!',
        name: 'New User',
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/invitations/token-abc123/accept', {
        password: 'SecurePassword123!',
        name: 'New User',
      })

      expect(result.user.email).toBe('newuser@example.com')
      expect(result.access_token).toBe('access-token-123')
    })
  })

  describe('revoke()', () => {
    it('should revoke an invitation', async () => {
      vi.mocked(mockFetch.delete).mockResolvedValue({ message: 'Invitation revoked successfully' })

      const result = await manager.revoke('token-abc123')

      expect(mockFetch.delete).toHaveBeenCalledWith('/api/v1/admin/invitations/token-abc123')
      expect(result.message).toBe('Invitation revoked successfully')
    })
  })
})

describe('FluxbaseManagement', () => {
  let management: FluxbaseManagement
  let mockFetch: any

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    }

    management = new FluxbaseManagement(mockFetch as unknown as FluxbaseFetch)
  })

  it('should initialize all managers', () => {
    expect(management.apiKeys).toBeInstanceOf(APIKeysManager)
    expect(management.webhooks).toBeInstanceOf(WebhooksManager)
    expect(management.invitations).toBeInstanceOf(InvitationsManager)
  })

  it('should share the same fetch client across managers', async () => {
    vi.mocked(mockFetch.get).mockResolvedValue({ api_keys: [], total: 0 })

    await management.apiKeys.list()

    expect(mockFetch.get).toHaveBeenCalled()
  })
})
