import { describe, it, expect, beforeEach, vi } from 'vitest'
import { OAuthProviderManager, AuthSettingsManager, FluxbaseOAuth } from './oauth'
import type { FluxbaseFetch } from './fetch'
import type {
  OAuthProvider,
  CreateOAuthProviderResponse,
  UpdateOAuthProviderResponse,
  DeleteOAuthProviderResponse,
  AuthSettings,
  UpdateAuthSettingsResponse,
} from './types'

describe('OAuthProviderManager', () => {
  let manager: OAuthProviderManager
  let mockFetch: any

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    }
    manager = new OAuthProviderManager(mockFetch as unknown as FluxbaseFetch)
  })

  describe('listProviders', () => {
    it('should list all OAuth providers', async () => {
      const mockProviders: OAuthProvider[] = [
        {
          id: 'provider-1',
          provider_name: 'github',
          display_name: 'GitHub',
          enabled: true,
          client_id: 'github-client-id',
          redirect_url: 'https://app.com/callback',
          scopes: ['user:email', 'read:user'],
          is_custom: false,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
        {
          id: 'provider-2',
          provider_name: 'google',
          display_name: 'Google',
          enabled: true,
          client_id: 'google-client-id',
          redirect_url: 'https://app.com/callback',
          scopes: ['openid', 'email', 'profile'],
          is_custom: false,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ]

      vi.mocked(mockFetch.get).mockResolvedValue(mockProviders)

      const result = await manager.listProviders()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/admin/oauth/providers')
      expect(result).toEqual(mockProviders)
      expect(result).toHaveLength(2)
    })

    it('should handle empty providers list', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      const result = await manager.listProviders()

      expect(result).toEqual([])
    })
  })

  describe('getProvider', () => {
    it('should get a specific OAuth provider', async () => {
      const mockProvider: OAuthProvider = {
        id: 'provider-1',
        provider_name: 'github',
        display_name: 'GitHub',
        enabled: true,
        client_id: 'github-client-id',
        redirect_url: 'https://app.com/callback',
        scopes: ['user:email', 'read:user'],
        is_custom: false,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockProvider)

      const result = await manager.getProvider('provider-1')

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/admin/oauth/providers/provider-1')
      expect(result).toEqual(mockProvider)
      expect(result.provider_name).toBe('github')
    })

    it('should handle provider not found', async () => {
      vi.mocked(mockFetch.get).mockRejectedValue(new Error('Provider not found'))

      await expect(manager.getProvider('nonexistent')).rejects.toThrow('Provider not found')
    })
  })

  describe('createProvider', () => {
    it('should create a new built-in OAuth provider', async () => {
      const mockResponse: CreateOAuthProviderResponse = {
        success: true,
        id: 'provider-new',
        provider: 'github',
        message: "OAuth provider 'GitHub' created successfully",
        created_at: '2024-01-02T00:00:00Z',
        updated_at: '2024-01-02T00:00:00Z',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      const result = await manager.createProvider({
        provider_name: 'github',
        display_name: 'GitHub',
        enabled: true,
        client_id: 'github-client-id',
        client_secret: 'github-client-secret',
        redirect_url: 'https://app.com/callback',
        scopes: ['user:email', 'read:user'],
        is_custom: false,
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/admin/oauth/providers', {
        provider_name: 'github',
        display_name: 'GitHub',
        enabled: true,
        client_id: 'github-client-id',
        client_secret: 'github-client-secret',
        redirect_url: 'https://app.com/callback',
        scopes: ['user:email', 'read:user'],
        is_custom: false,
      })
      expect(result.success).toBe(true)
      expect(result.id).toBe('provider-new')
    })

    it('should create a custom OAuth provider', async () => {
      const mockResponse: CreateOAuthProviderResponse = {
        success: true,
        id: 'provider-custom',
        provider: 'custom_sso',
        message: "OAuth provider 'Custom SSO' created successfully",
        created_at: '2024-01-02T00:00:00Z',
        updated_at: '2024-01-02T00:00:00Z',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      await manager.createProvider({
        provider_name: 'custom_sso',
        display_name: 'Custom SSO',
        enabled: true,
        client_id: 'custom-client-id',
        client_secret: 'custom-client-secret',
        redirect_url: 'https://app.com/callback',
        scopes: ['openid', 'profile', 'email'],
        is_custom: true,
        authorization_url: 'https://sso.example.com/oauth/authorize',
        token_url: 'https://sso.example.com/oauth/token',
        user_info_url: 'https://sso.example.com/oauth/userinfo',
      })

      expect(mockFetch.post).toHaveBeenCalledWith(
        '/api/v1/admin/oauth/providers',
        expect.objectContaining({
          is_custom: true,
          authorization_url: 'https://sso.example.com/oauth/authorize',
        })
      )
    })

    it('should handle provider creation errors', async () => {
      vi.mocked(mockFetch.post).mockRejectedValue(new Error('Provider already exists'))

      await expect(
        manager.createProvider({
          provider_name: 'github',
          display_name: 'GitHub',
          enabled: true,
          client_id: 'id',
          client_secret: 'secret',
          redirect_url: 'url',
          scopes: [],
          is_custom: false,
        })
      ).rejects.toThrow('Provider already exists')
    })
  })

  describe('updateProvider', () => {
    it('should update a provider', async () => {
      const mockResponse: UpdateOAuthProviderResponse = {
        success: true,
        message: "OAuth provider 'GitHub' updated successfully",
      }

      vi.mocked(mockFetch.put).mockResolvedValue(mockResponse)

      const result = await manager.updateProvider('provider-1', {
        enabled: false,
        scopes: ['user:email'],
      })

      expect(mockFetch.put).toHaveBeenCalledWith('/api/v1/admin/oauth/providers/provider-1', {
        enabled: false,
        scopes: ['user:email'],
      })
      expect(result.success).toBe(true)
    })

    it('should update multiple fields', async () => {
      const mockResponse: UpdateOAuthProviderResponse = {
        success: true,
        message: "OAuth provider 'GitHub' updated successfully",
      }

      vi.mocked(mockFetch.put).mockResolvedValue(mockResponse)

      await manager.updateProvider('provider-1', {
        display_name: 'GitHub OAuth',
        enabled: true,
        redirect_url: 'https://new-domain.com/callback',
        scopes: ['user:email', 'read:user', 'read:org'],
      })

      expect(mockFetch.put).toHaveBeenCalledWith(
        '/api/v1/admin/oauth/providers/provider-1',
        expect.objectContaining({
          display_name: 'GitHub OAuth',
          enabled: true,
        })
      )
    })

    it('should handle update errors', async () => {
      vi.mocked(mockFetch.put).mockRejectedValue(new Error('Provider not found'))

      await expect(manager.updateProvider('nonexistent', { enabled: false })).rejects.toThrow(
        'Provider not found'
      )
    })
  })

  describe('deleteProvider', () => {
    it('should delete a provider', async () => {
      const mockResponse: DeleteOAuthProviderResponse = {
        success: true,
        message: "OAuth provider 'GitHub' deleted successfully",
      }

      vi.mocked(mockFetch.delete).mockResolvedValue(mockResponse)

      const result = await manager.deleteProvider('provider-1')

      expect(mockFetch.delete).toHaveBeenCalledWith('/api/v1/admin/oauth/providers/provider-1')
      expect(result.success).toBe(true)
    })

    it('should handle delete errors', async () => {
      vi.mocked(mockFetch.delete).mockRejectedValue(new Error('Provider not found'))

      await expect(manager.deleteProvider('nonexistent')).rejects.toThrow('Provider not found')
    })
  })

  describe('convenience methods', () => {
    it('should enable a provider', async () => {
      const mockResponse: UpdateOAuthProviderResponse = {
        success: true,
        message: "OAuth provider 'GitHub' updated successfully",
      }

      vi.mocked(mockFetch.put).mockResolvedValue(mockResponse)

      await manager.enableProvider('provider-1')

      expect(mockFetch.put).toHaveBeenCalledWith('/api/v1/admin/oauth/providers/provider-1', {
        enabled: true,
      })
    })

    it('should disable a provider', async () => {
      const mockResponse: UpdateOAuthProviderResponse = {
        success: true,
        message: "OAuth provider 'GitHub' updated successfully",
      }

      vi.mocked(mockFetch.put).mockResolvedValue(mockResponse)

      await manager.disableProvider('provider-1')

      expect(mockFetch.put).toHaveBeenCalledWith('/api/v1/admin/oauth/providers/provider-1', {
        enabled: false,
      })
    })
  })
})

describe('AuthSettingsManager', () => {
  let manager: AuthSettingsManager
  let mockFetch: any

  const mockSettings: AuthSettings = {
    enable_signup: true,
    require_email_verification: true,
    enable_magic_link: true,
    password_min_length: 12,
    password_require_uppercase: true,
    password_require_lowercase: true,
    password_require_number: true,
    password_require_special: true,
    session_timeout_minutes: 120,
    max_sessions_per_user: 5,
  }

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    }
    manager = new AuthSettingsManager(mockFetch as unknown as FluxbaseFetch)
  })

  describe('get', () => {
    it('should get authentication settings', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue(mockSettings)

      const result = await manager.get()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/admin/auth/settings')
      expect(result).toEqual(mockSettings)
      expect(result.password_min_length).toBe(12)
    })
  })

  describe('update', () => {
    it('should update password requirements', async () => {
      const mockResponse: UpdateAuthSettingsResponse = {
        success: true,
        message: 'Authentication settings updated successfully',
      }

      vi.mocked(mockFetch.put).mockResolvedValue(mockResponse)

      const result = await manager.update({
        password_min_length: 16,
        password_require_uppercase: true,
        password_require_number: true,
        password_require_special: true,
      })

      expect(mockFetch.put).toHaveBeenCalledWith('/api/v1/admin/auth/settings', {
        password_min_length: 16,
        password_require_uppercase: true,
        password_require_number: true,
        password_require_special: true,
      })
      expect(result.success).toBe(true)
    })

    it('should update session settings', async () => {
      const mockResponse: UpdateAuthSettingsResponse = {
        success: true,
        message: 'Authentication settings updated successfully',
      }

      vi.mocked(mockFetch.put).mockResolvedValue(mockResponse)

      await manager.update({
        session_timeout_minutes: 240,
        max_sessions_per_user: 10,
      })

      expect(mockFetch.put).toHaveBeenCalledWith(
        '/api/v1/admin/auth/settings',
        expect.objectContaining({
          session_timeout_minutes: 240,
          max_sessions_per_user: 10,
        })
      )
    })

    it('should update signup and verification settings', async () => {
      const mockResponse: UpdateAuthSettingsResponse = {
        success: true,
        message: 'Authentication settings updated successfully',
      }

      vi.mocked(mockFetch.put).mockResolvedValue(mockResponse)

      await manager.update({
        enable_signup: false,
        require_email_verification: false,
        enable_magic_link: false,
      })

      expect(mockFetch.put).toHaveBeenCalledWith(
        '/api/v1/admin/auth/settings',
        expect.objectContaining({
          enable_signup: false,
          require_email_verification: false,
          enable_magic_link: false,
        })
      )
    })

    it('should handle update errors', async () => {
      vi.mocked(mockFetch.put).mockRejectedValue(new Error('Update failed'))

      await expect(manager.update({ password_min_length: 16 })).rejects.toThrow('Update failed')
    })
  })
})

describe('FluxbaseOAuth', () => {
  it('should initialize both managers', () => {
    const mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    } as unknown as FluxbaseFetch

    const oauth = new FluxbaseOAuth(mockFetch)

    expect(oauth.providers).toBeInstanceOf(OAuthProviderManager)
    expect(oauth.authSettings).toBeInstanceOf(AuthSettingsManager)
  })
})
