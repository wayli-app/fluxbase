import { describe, it, expect, beforeEach, vi } from 'vitest'
import { SystemSettingsManager, AppSettingsManager, FluxbaseSettings, SettingsClient } from './settings'
import type { FluxbaseFetch } from './fetch'
import type {
  SystemSetting,
  AppSettings,
  CustomSetting,
} from './types'

describe('SystemSettingsManager', () => {
  let manager: SystemSettingsManager
  let mockFetch: any

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    }
    manager = new SystemSettingsManager(mockFetch as unknown as FluxbaseFetch)
  })

  describe('list', () => {
    it('should list all system settings', async () => {
      const mockSettings: SystemSetting[] = [
        {
          id: 'setting-1',
          key: 'app.auth.enable_signup',
          value: { value: true },
          description: 'Enable user signup',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
        {
          id: 'setting-2',
          key: 'app.realtime.enabled',
          value: { value: true },
          description: 'Enable realtime features',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ]

      vi.mocked(mockFetch.get).mockResolvedValue(mockSettings)

      const result = await manager.list()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/admin/system/settings')
      expect(result.settings).toEqual(mockSettings)
      expect(result.settings).toHaveLength(2)
    })

    it('should handle empty settings array', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      const result = await manager.list()

      expect(result.settings).toEqual([])
    })
  })

  describe('get', () => {
    it('should get a specific setting by key', async () => {
      const mockSetting: SystemSetting = {
        id: 'setting-1',
        key: 'app.auth.enable_signup',
        value: { value: true },
        description: 'Enable user signup',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockSetting)

      const result = await manager.get('app.auth.enable_signup')

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/admin/system/settings/app.auth.enable_signup')
      expect(result).toEqual(mockSetting)
      expect(result.key).toBe('app.auth.enable_signup')
    })

    it('should handle not found error', async () => {
      vi.mocked(mockFetch.get).mockRejectedValue(new Error('Setting not found'))

      await expect(manager.get('nonexistent.key')).rejects.toThrow('Setting not found')
    })
  })

  describe('update', () => {
    it('should update a setting', async () => {
      const mockSetting: SystemSetting = {
        id: 'setting-1',
        key: 'app.auth.enable_signup',
        value: { value: false },
        description: 'Enable user signup',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-02T00:00:00Z',
      }

      vi.mocked(mockFetch.put).mockResolvedValue(mockSetting)

      const result = await manager.update('app.auth.enable_signup', {
        value: { value: false },
        description: 'Enable user signup',
      })

      expect(mockFetch.put).toHaveBeenCalledWith('/api/v1/admin/system/settings/app.auth.enable_signup', {
        value: { value: false },
        description: 'Enable user signup',
      })
      expect(result.value.value).toBe(false)
    })

    it('should create a new setting if it does not exist', async () => {
      const mockSetting: SystemSetting = {
        id: 'setting-new',
        key: 'app.new.setting',
        value: { value: 'test' },
        description: 'New setting',
        created_at: '2024-01-02T00:00:00Z',
        updated_at: '2024-01-02T00:00:00Z',
      }

      vi.mocked(mockFetch.put).mockResolvedValue(mockSetting)

      const result = await manager.update('app.new.setting', {
        value: { value: 'test' },
        description: 'New setting',
      })

      expect(result.key).toBe('app.new.setting')
    })
  })

  describe('delete', () => {
    it('should delete a setting', async () => {
      vi.mocked(mockFetch.delete).mockResolvedValue(undefined)

      await manager.delete('app.auth.enable_signup')

      expect(mockFetch.delete).toHaveBeenCalledWith('/api/v1/admin/system/settings/app.auth.enable_signup')
    })

    it('should handle delete errors', async () => {
      vi.mocked(mockFetch.delete).mockRejectedValue(new Error('Setting not found'))

      await expect(manager.delete('nonexistent.key')).rejects.toThrow('Setting not found')
    })
  })
})

describe('AppSettingsManager', () => {
  let manager: AppSettingsManager
  let mockFetch: any

  const mockAppSettings: AppSettings = {
    authentication: {
      enable_signup: true,
      enable_magic_link: true,
      password_min_length: 8,
      require_email_verification: false,
    },
    features: {
      enable_realtime: true,
      enable_storage: true,
      enable_functions: true,
    },
    email: {
      enabled: false,
      provider: 'smtp',
    },
    security: {
      enable_global_rate_limit: false,
    },
  }

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    }
    manager = new AppSettingsManager(mockFetch as unknown as FluxbaseFetch)
  })

  describe('get', () => {
    it('should get all app settings', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue(mockAppSettings)

      const result = await manager.get()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/admin/app/settings')
      expect(result).toEqual(mockAppSettings)
      expect(result.authentication.enable_signup).toBe(true)
      expect(result.features.enable_realtime).toBe(true)
    })
  })

  describe('update', () => {
    it('should update authentication settings', async () => {
      const updatedSettings = {
        ...mockAppSettings,
        authentication: {
          ...mockAppSettings.authentication,
          enable_signup: false,
          password_min_length: 12,
        },
      }

      vi.mocked(mockFetch.put).mockResolvedValue(updatedSettings)

      const result = await manager.update({
        authentication: {
          enable_signup: false,
          password_min_length: 12,
        },
      })

      expect(mockFetch.put).toHaveBeenCalledWith('/api/v1/admin/app/settings', {
        authentication: {
          enable_signup: false,
          password_min_length: 12,
        },
      })
      expect(result.authentication.enable_signup).toBe(false)
      expect(result.authentication.password_min_length).toBe(12)
    })

    it('should update multiple categories at once', async () => {
      const updatedSettings = {
        ...mockAppSettings,
        authentication: { ...mockAppSettings.authentication, enable_signup: false },
        features: { ...mockAppSettings.features, enable_realtime: false },
        security: { ...mockAppSettings.security, enable_global_rate_limit: true },
      }

      vi.mocked(mockFetch.put).mockResolvedValue(updatedSettings)

      const result = await manager.update({
        authentication: { enable_signup: false },
        features: { enable_realtime: false },
        security: { enable_global_rate_limit: true },
      })

      expect(result.authentication.enable_signup).toBe(false)
      expect(result.features.enable_realtime).toBe(false)
      expect(result.security.enable_global_rate_limit).toBe(true)
    })
  })

  describe('reset', () => {
    it('should reset all settings to defaults', async () => {
      vi.mocked(mockFetch.post).mockResolvedValue(mockAppSettings)

      const result = await manager.reset()

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/admin/app/settings/reset', {})
      expect(result).toEqual(mockAppSettings)
    })
  })

  describe('convenience methods', () => {
    it('should enable signup', async () => {
      const updatedSettings = {
        ...mockAppSettings,
        authentication: { ...mockAppSettings.authentication, enable_signup: true },
      }

      vi.mocked(mockFetch.put).mockResolvedValue(updatedSettings)

      const result = await manager.enableSignup()

      expect(mockFetch.put).toHaveBeenCalledWith('/api/v1/admin/app/settings', {
        authentication: { enable_signup: true },
      })
      expect(result.authentication.enable_signup).toBe(true)
    })

    it('should disable signup', async () => {
      const updatedSettings = {
        ...mockAppSettings,
        authentication: { ...mockAppSettings.authentication, enable_signup: false },
      }

      vi.mocked(mockFetch.put).mockResolvedValue(updatedSettings)

      const result = await manager.disableSignup()

      expect(mockFetch.put).toHaveBeenCalledWith('/api/v1/admin/app/settings', {
        authentication: { enable_signup: false },
      })
      expect(result.authentication.enable_signup).toBe(false)
    })

    it('should set password min length', async () => {
      const updatedSettings = {
        ...mockAppSettings,
        authentication: { ...mockAppSettings.authentication, password_min_length: 16 },
      }

      vi.mocked(mockFetch.put).mockResolvedValue(updatedSettings)

      const result = await manager.setPasswordMinLength(16)

      expect(mockFetch.put).toHaveBeenCalledWith('/api/v1/admin/app/settings', {
        authentication: { password_min_length: 16 },
      })
      expect(result.authentication.password_min_length).toBe(16)
    })

    it('should reject invalid password length', async () => {
      await expect(manager.setPasswordMinLength(7)).rejects.toThrow('Password minimum length must be between 8 and 128')
      await expect(manager.setPasswordMinLength(129)).rejects.toThrow('Password minimum length must be between 8 and 128')
    })

    it('should enable feature', async () => {
      const updatedSettings = {
        ...mockAppSettings,
        features: { ...mockAppSettings.features, enable_realtime: true },
      }

      vi.mocked(mockFetch.put).mockResolvedValue(updatedSettings)

      const result = await manager.setFeature('realtime', true)

      expect(mockFetch.put).toHaveBeenCalledWith('/api/v1/admin/app/settings', {
        features: { enable_realtime: true },
      })
      expect(result.features.enable_realtime).toBe(true)
    })

    it('should disable feature', async () => {
      const updatedSettings = {
        ...mockAppSettings,
        features: { ...mockAppSettings.features, enable_storage: false },
      }

      vi.mocked(mockFetch.put).mockResolvedValue(updatedSettings)

      const result = await manager.setFeature('storage', false)

      expect(mockFetch.put).toHaveBeenCalledWith('/api/v1/admin/app/settings', {
        features: { enable_storage: false },
      })
      expect(result.features.enable_storage).toBe(false)
    })

    it('should enable rate limiting', async () => {
      const updatedSettings = {
        ...mockAppSettings,
        security: { ...mockAppSettings.security, enable_global_rate_limit: true },
      }

      vi.mocked(mockFetch.put).mockResolvedValue(updatedSettings)

      const result = await manager.setRateLimiting(true)

      expect(mockFetch.put).toHaveBeenCalledWith('/api/v1/admin/app/settings', {
        security: { enable_global_rate_limit: true },
      })
      expect(result.security.enable_global_rate_limit).toBe(true)
    })
  })
})

describe('AppSettingsManager - Custom Settings', () => {
  let manager: AppSettingsManager
  let mockFetch: any

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    }
    manager = new AppSettingsManager(mockFetch as unknown as FluxbaseFetch)
  })

  describe('getSetting', () => {
    it('should get setting value without metadata', async () => {
      const mockSetting: CustomSetting = {
        id: 'setting-1',
        key: 'features.beta_enabled',
        value: { enabled: true },
        value_type: 'json',
        description: 'Beta feature toggle',
        editable_by: ['dashboard_admin'],
        metadata: {},
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockSetting)

      const result = await manager.getSetting('features.beta_enabled')

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/admin/settings/custom/features.beta_enabled')
      expect(result).toEqual({ enabled: true })
      expect(result).not.toHaveProperty('id')
    })

    it('should handle errors', async () => {
      vi.mocked(mockFetch.get).mockRejectedValue(new Error('Setting not found'))

      await expect(manager.getSetting('nonexistent')).rejects.toThrow('Setting not found')
    })
  })

  describe('getSettings', () => {
    it('should get multiple setting values', async () => {
      const mockSettings: CustomSetting[] = [
        {
          id: 'setting-1',
          key: 'features.beta_enabled',
          value: { enabled: true },
          value_type: 'json',
          description: 'Beta toggle',
          editable_by: ['dashboard_admin'],
          metadata: {},
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
        {
          id: 'setting-2',
          key: 'features.dark_mode',
          value: { enabled: false },
          value_type: 'json',
          description: 'Dark mode toggle',
          editable_by: ['dashboard_admin'],
          metadata: {},
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ]

      vi.mocked(mockFetch.post).mockResolvedValue(mockSettings)

      const result = await manager.getSettings(['features.beta_enabled', 'features.dark_mode'])

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/settings/batch', {
        keys: ['features.beta_enabled', 'features.dark_mode'],
      })
      expect(result).toEqual({
        'features.beta_enabled': { enabled: true },
        'features.dark_mode': { enabled: false },
      })
    })

    it('should handle empty keys array', async () => {
      vi.mocked(mockFetch.post).mockResolvedValue([])

      const result = await manager.getSettings([])

      expect(result).toEqual({})
    })
  })

  describe('setSetting', () => {
    it('should create new setting if not found', async () => {
      const mockSetting: CustomSetting = {
        id: 'setting-1',
        key: 'billing.tiers',
        value: { free: 1000, pro: 10000 },
        value_type: 'json',
        description: 'API quotas',
        editable_by: ['dashboard_admin'],
        metadata: {},
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }

      vi.mocked(mockFetch.put).mockRejectedValue({ status: 404, message: 'not found' })
      vi.mocked(mockFetch.post).mockResolvedValue(mockSetting)

      const result = await manager.setSetting('billing.tiers', { free: 1000, pro: 10000 }, {
        description: 'API quotas'
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/admin/settings/custom', {
        key: 'billing.tiers',
        value: { free: 1000, pro: 10000 },
        value_type: 'json',
        description: 'API quotas',
        is_public: false,
        is_secret: false,
      })
      expect(result).toEqual(mockSetting)
    })
  })
})

describe('SettingsClient', () => {
  let client: SettingsClient
  let mockFetch: any

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    }
    client = new SettingsClient(mockFetch as unknown as FluxbaseFetch)
  })

  describe('get', () => {
    it('should get public setting value', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue({ value: { enabled: true } })

      const result = await client.get('features.beta_enabled')

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/settings/features.beta_enabled')
      expect(result).toEqual({ enabled: true })
    })

    it('should handle settings with special characters in key', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue({ value: '1.0.0' })

      const result = await client.get('public.app_version')

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/settings/public.app_version')
      expect(result).toBe('1.0.0')
    })

    it('should throw error for unauthorized access', async () => {
      vi.mocked(mockFetch.get).mockRejectedValue(new Error('Forbidden'))

      await expect(client.get('internal.secret')).rejects.toThrow('Forbidden')
    })

    it('should throw error for non-existent setting', async () => {
      vi.mocked(mockFetch.get).mockRejectedValue(new Error('Not found'))

      await expect(client.get('nonexistent.key')).rejects.toThrow('Not found')
    })
  })

  describe('getMany', () => {
    it('should get multiple public settings', async () => {
      vi.mocked(mockFetch.post).mockResolvedValue([
        { key: 'features.beta_enabled', value: { enabled: true } },
        { key: 'features.dark_mode', value: { enabled: false } },
        { key: 'public.app_version', value: '1.0.0' },
      ])

      const result = await client.getMany([
        'features.beta_enabled',
        'features.dark_mode',
        'public.app_version',
      ])

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/settings/batch', {
        keys: ['features.beta_enabled', 'features.dark_mode', 'public.app_version'],
      })
      expect(result).toEqual({
        'features.beta_enabled': { enabled: true },
        'features.dark_mode': { enabled: false },
        'public.app_version': '1.0.0',
      })
    })

    it('should filter unauthorized settings (RLS)', async () => {
      // Backend filters out settings user can't access
      vi.mocked(mockFetch.post).mockResolvedValue([
        { key: 'features.beta_enabled', value: { enabled: true } },
        { key: 'features.dark_mode', value: { enabled: false } },
        // 'internal.secret' is omitted by RLS
      ])

      const result = await client.getMany([
        'features.beta_enabled',
        'features.dark_mode',
        'internal.secret',
      ])

      expect(result).toEqual({
        'features.beta_enabled': { enabled: true },
        'features.dark_mode': { enabled: false },
      })
      expect(result).not.toHaveProperty('internal.secret')
    })

    it('should handle empty keys array', async () => {
      vi.mocked(mockFetch.post).mockResolvedValue([])

      const result = await client.getMany([])

      expect(result).toEqual({})
    })
  })
})

describe('FluxbaseSettings', () => {
  it('should initialize both managers', () => {
    const mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    } as unknown as FluxbaseFetch

    const settings = new FluxbaseSettings(mockFetch)

    expect(settings.system).toBeInstanceOf(SystemSettingsManager)
    expect(settings.app).toBeInstanceOf(AppSettingsManager)
  })
})
