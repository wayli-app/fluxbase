import type { FluxbaseFetch } from './fetch'
import type {
  SystemSetting,
  UpdateSystemSettingRequest,
  ListSystemSettingsResponse,
  AppSettings,
  UpdateAppSettingsRequest,
} from './types'

/**
 * System Settings Manager
 *
 * Manages low-level system settings with key-value storage.
 * For application-level settings, use AppSettingsManager instead.
 *
 * @example
 * ```typescript
 * const settings = client.admin.settings.system
 *
 * // List all system settings
 * const { settings } = await settings.list()
 *
 * // Get specific setting
 * const setting = await settings.get('app.auth.enable_signup')
 *
 * // Update setting
 * await settings.update('app.auth.enable_signup', {
 *   value: { value: true },
 *   description: 'Enable user signup'
 * })
 *
 * // Delete setting
 * await settings.delete('app.auth.enable_signup')
 * ```
 */
export class SystemSettingsManager {
  constructor(private fetch: FluxbaseFetch) {}

  /**
   * List all system settings
   *
   * @returns Promise resolving to ListSystemSettingsResponse
   *
   * @example
   * ```typescript
   * const response = await client.admin.settings.system.list()
   * console.log(response.settings)
   * ```
   */
  async list(): Promise<ListSystemSettingsResponse> {
    const settings = await this.fetch.get<SystemSetting[]>('/api/v1/admin/system/settings')
    return { settings: Array.isArray(settings) ? settings : [] }
  }

  /**
   * Get a specific system setting by key
   *
   * @param key - Setting key (e.g., 'app.auth.enable_signup')
   * @returns Promise resolving to SystemSetting
   *
   * @example
   * ```typescript
   * const setting = await client.admin.settings.system.get('app.auth.enable_signup')
   * console.log(setting.value)
   * ```
   */
  async get(key: string): Promise<SystemSetting> {
    return await this.fetch.get<SystemSetting>(`/api/v1/admin/system/settings/${key}`)
  }

  /**
   * Update or create a system setting
   *
   * @param key - Setting key
   * @param request - Update request with value and optional description
   * @returns Promise resolving to SystemSetting
   *
   * @example
   * ```typescript
   * const updated = await client.admin.settings.system.update('app.auth.enable_signup', {
   *   value: { value: true },
   *   description: 'Enable user signup'
   * })
   * ```
   */
  async update(key: string, request: UpdateSystemSettingRequest): Promise<SystemSetting> {
    return await this.fetch.put<SystemSetting>(`/api/v1/admin/system/settings/${key}`, request)
  }

  /**
   * Delete a system setting
   *
   * @param key - Setting key to delete
   * @returns Promise<void>
   *
   * @example
   * ```typescript
   * await client.admin.settings.system.delete('app.auth.enable_signup')
   * ```
   */
  async delete(key: string): Promise<void> {
    await this.fetch.delete(`/api/v1/admin/system/settings/${key}`)
  }
}

/**
 * Application Settings Manager
 *
 * Manages high-level application settings with a structured API.
 * Provides type-safe access to authentication, features, email, and security settings.
 *
 * @example
 * ```typescript
 * const settings = client.admin.settings.app
 *
 * // Get all app settings
 * const appSettings = await settings.get()
 * console.log(appSettings.authentication.enable_signup)
 *
 * // Update specific settings
 * const updated = await settings.update({
 *   authentication: {
 *     enable_signup: true,
 *     password_min_length: 12
 *   }
 * })
 *
 * // Reset to defaults
 * await settings.reset()
 * ```
 */
export class AppSettingsManager {
  constructor(private fetch: FluxbaseFetch) {}

  /**
   * Get all application settings
   *
   * Returns structured settings for authentication, features, email, and security.
   *
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * const settings = await client.admin.settings.app.get()
   *
   * console.log('Signup enabled:', settings.authentication.enable_signup)
   * console.log('Realtime enabled:', settings.features.enable_realtime)
   * console.log('Email provider:', settings.email.provider)
   * ```
   */
  async get(): Promise<AppSettings> {
    return await this.fetch.get<AppSettings>('/api/v1/admin/app/settings')
  }

  /**
   * Update application settings
   *
   * Supports partial updates - only provide the fields you want to change.
   *
   * @param request - Settings to update (partial update supported)
   * @returns Promise resolving to AppSettings - Updated settings
   *
   * @example
   * ```typescript
   * // Update authentication settings
   * const updated = await client.admin.settings.app.update({
   *   authentication: {
   *     enable_signup: true,
   *     password_min_length: 12
   *   }
   * })
   *
   * // Update multiple categories
   * await client.admin.settings.app.update({
   *   authentication: { enable_signup: false },
   *   features: { enable_realtime: true },
   *   security: { enable_global_rate_limit: true }
   * })
   * ```
   */
  async update(request: UpdateAppSettingsRequest): Promise<AppSettings> {
    return await this.fetch.put<AppSettings>('/api/v1/admin/app/settings', request)
  }

  /**
   * Reset all application settings to defaults
   *
   * This will delete all custom settings and return to default values.
   *
   * @returns Promise resolving to AppSettings - Default settings
   *
   * @example
   * ```typescript
   * const defaults = await client.admin.settings.app.reset()
   * console.log('Settings reset to defaults:', defaults)
   * ```
   */
  async reset(): Promise<AppSettings> {
    return await this.fetch.post<AppSettings>('/api/v1/admin/app/settings/reset', {})
  }

  /**
   * Enable user signup
   *
   * Convenience method to enable user registration.
   *
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.enableSignup()
   * ```
   */
  async enableSignup(): Promise<AppSettings> {
    return await this.update({
      authentication: { enable_signup: true },
    })
  }

  /**
   * Disable user signup
   *
   * Convenience method to disable user registration.
   *
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.disableSignup()
   * ```
   */
  async disableSignup(): Promise<AppSettings> {
    return await this.update({
      authentication: { enable_signup: false },
    })
  }

  /**
   * Update password minimum length
   *
   * Convenience method to set password requirements.
   *
   * @param length - Minimum password length (8-128 characters)
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.setPasswordMinLength(12)
   * ```
   */
  async setPasswordMinLength(length: number): Promise<AppSettings> {
    if (length < 8 || length > 128) {
      throw new Error('Password minimum length must be between 8 and 128 characters')
    }

    return await this.update({
      authentication: { password_min_length: length },
    })
  }

  /**
   * Enable or disable a feature
   *
   * Convenience method to toggle feature flags.
   *
   * @param feature - Feature name ('realtime' | 'storage' | 'functions')
   * @param enabled - Whether to enable or disable the feature
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * // Enable realtime
   * await client.admin.settings.app.setFeature('realtime', true)
   *
   * // Disable storage
   * await client.admin.settings.app.setFeature('storage', false)
   * ```
   */
  async setFeature(feature: 'realtime' | 'storage' | 'functions', enabled: boolean): Promise<AppSettings> {
    const featureKey =
      feature === 'realtime'
        ? 'enable_realtime'
        : feature === 'storage'
          ? 'enable_storage'
          : 'enable_functions'

    return await this.update({
      features: { [featureKey]: enabled },
    })
  }

  /**
   * Enable or disable global rate limiting
   *
   * Convenience method to toggle global rate limiting.
   *
   * @param enabled - Whether to enable rate limiting
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.setRateLimiting(true)
   * ```
   */
  async setRateLimiting(enabled: boolean): Promise<AppSettings> {
    return await this.update({
      security: { enable_global_rate_limit: enabled },
    })
  }
}

/**
 * Settings Manager
 *
 * Provides access to both system-level and application-level settings.
 *
 * @example
 * ```typescript
 * const settings = client.admin.settings
 *
 * // Access system settings
 * const systemSettings = await settings.system.list()
 *
 * // Access app settings
 * const appSettings = await settings.app.get()
 * ```
 */
export class FluxbaseSettings {
  public system: SystemSettingsManager
  public app: AppSettingsManager

  constructor(fetch: FluxbaseFetch) {
    this.system = new SystemSettingsManager(fetch)
    this.app = new AppSettingsManager(fetch)
  }
}
