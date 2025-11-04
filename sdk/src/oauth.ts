import type { FluxbaseFetch } from './fetch'
import type {
  OAuthProvider,
  CreateOAuthProviderRequest,
  CreateOAuthProviderResponse,
  UpdateOAuthProviderRequest,
  UpdateOAuthProviderResponse,
  DeleteOAuthProviderResponse,
  AuthSettings,
  UpdateAuthSettingsRequest,
  UpdateAuthSettingsResponse,
} from './types'

/**
 * OAuth Provider Manager
 *
 * Manages OAuth provider configurations for third-party authentication.
 * Supports both built-in providers (Google, GitHub, etc.) and custom OAuth2 providers.
 *
 * @example
 * ```typescript
 * const oauth = client.admin.oauth
 *
 * // List all OAuth providers
 * const { providers } = await oauth.listProviders()
 *
 * // Create a new provider
 * await oauth.createProvider({
 *   provider_name: 'github',
 *   display_name: 'GitHub',
 *   enabled: true,
 *   client_id: 'your-client-id',
 *   client_secret: 'your-client-secret',
 *   redirect_url: 'https://yourapp.com/auth/callback',
 *   scopes: ['user:email', 'read:user'],
 *   is_custom: false
 * })
 *
 * // Update a provider
 * await oauth.updateProvider('provider-id', {
 *   enabled: false
 * })
 *
 * // Delete a provider
 * await oauth.deleteProvider('provider-id')
 * ```
 */
export class OAuthProviderManager {
  constructor(private fetch: FluxbaseFetch) {}

  /**
   * List all OAuth providers
   *
   * Retrieves all configured OAuth providers including both enabled and disabled providers.
   * Note: Client secrets are not included in the response for security reasons.
   *
   * @returns Promise resolving to ListOAuthProvidersResponse
   *
   * @example
   * ```typescript
   * const { providers } = await client.admin.oauth.listProviders()
   *
   * providers.forEach(provider => {
   *   console.log(`${provider.display_name}: ${provider.enabled ? 'enabled' : 'disabled'}`)
   * })
   * ```
   */
  async listProviders(): Promise<OAuthProvider[]> {
    const providers = await this.fetch.get<OAuthProvider[]>('/api/v1/admin/oauth/providers')
    return Array.isArray(providers) ? providers : []
  }

  /**
   * Get a specific OAuth provider by ID
   *
   * Retrieves detailed configuration for a single OAuth provider.
   * Note: Client secret is not included in the response.
   *
   * @param providerId - Provider ID (UUID)
   * @returns Promise resolving to OAuthProvider
   *
   * @example
   * ```typescript
   * const provider = await client.admin.oauth.getProvider('provider-uuid')
   *
   * console.log('Provider:', provider.display_name)
   * console.log('Scopes:', provider.scopes.join(', '))
   * console.log('Redirect URL:', provider.redirect_url)
   * ```
   */
  async getProvider(providerId: string): Promise<OAuthProvider> {
    return await this.fetch.get<OAuthProvider>(`/api/v1/admin/oauth/providers/${providerId}`)
  }

  /**
   * Create a new OAuth provider
   *
   * Creates a new OAuth provider configuration. For built-in providers (Google, GitHub, etc.),
   * set `is_custom` to false. For custom OAuth2 providers, set `is_custom` to true and provide
   * the authorization, token, and user info URLs.
   *
   * @param request - OAuth provider configuration
   * @returns Promise resolving to CreateOAuthProviderResponse
   *
   * @example
   * ```typescript
   * // Create GitHub provider
   * const result = await client.admin.oauth.createProvider({
   *   provider_name: 'github',
   *   display_name: 'GitHub',
   *   enabled: true,
   *   client_id: process.env.GITHUB_CLIENT_ID,
   *   client_secret: process.env.GITHUB_CLIENT_SECRET,
   *   redirect_url: 'https://yourapp.com/auth/callback',
   *   scopes: ['user:email', 'read:user'],
   *   is_custom: false
   * })
   *
   * console.log('Provider created:', result.id)
   * ```
   *
   * @example
   * ```typescript
   * // Create custom OAuth2 provider
   * await client.admin.oauth.createProvider({
   *   provider_name: 'custom_sso',
   *   display_name: 'Custom SSO',
   *   enabled: true,
   *   client_id: 'client-id',
   *   client_secret: 'client-secret',
   *   redirect_url: 'https://yourapp.com/auth/callback',
   *   scopes: ['openid', 'profile', 'email'],
   *   is_custom: true,
   *   authorization_url: 'https://sso.example.com/oauth/authorize',
   *   token_url: 'https://sso.example.com/oauth/token',
   *   user_info_url: 'https://sso.example.com/oauth/userinfo'
   * })
   * ```
   */
  async createProvider(request: CreateOAuthProviderRequest): Promise<CreateOAuthProviderResponse> {
    return await this.fetch.post<CreateOAuthProviderResponse>('/api/v1/admin/oauth/providers', request)
  }

  /**
   * Update an existing OAuth provider
   *
   * Updates an OAuth provider configuration. All fields are optional - only provided fields
   * will be updated. To update the client secret, provide a non-empty value.
   *
   * @param providerId - Provider ID (UUID)
   * @param request - Fields to update
   * @returns Promise resolving to UpdateOAuthProviderResponse
   *
   * @example
   * ```typescript
   * // Disable a provider
   * await client.admin.oauth.updateProvider('provider-id', {
   *   enabled: false
   * })
   * ```
   *
   * @example
   * ```typescript
   * // Update scopes and redirect URL
   * await client.admin.oauth.updateProvider('provider-id', {
   *   scopes: ['user:email', 'read:user', 'read:org'],
   *   redirect_url: 'https://newdomain.com/auth/callback'
   * })
   * ```
   *
   * @example
   * ```typescript
   * // Rotate client secret
   * await client.admin.oauth.updateProvider('provider-id', {
   *   client_id: 'new-client-id',
   *   client_secret: 'new-client-secret'
   * })
   * ```
   */
  async updateProvider(
    providerId: string,
    request: UpdateOAuthProviderRequest
  ): Promise<UpdateOAuthProviderResponse> {
    return await this.fetch.put<UpdateOAuthProviderResponse>(
      `/api/v1/admin/oauth/providers/${providerId}`,
      request
    )
  }

  /**
   * Delete an OAuth provider
   *
   * Permanently deletes an OAuth provider configuration. This will prevent users from
   * authenticating with this provider.
   *
   * @param providerId - Provider ID (UUID) to delete
   * @returns Promise resolving to DeleteOAuthProviderResponse
   *
   * @example
   * ```typescript
   * await client.admin.oauth.deleteProvider('provider-id')
   * console.log('Provider deleted')
   * ```
   *
   * @example
   * ```typescript
   * // Safe deletion with confirmation
   * const provider = await client.admin.oauth.getProvider('provider-id')
   * const confirmed = await confirm(`Delete ${provider.display_name}?`)
   *
   * if (confirmed) {
   *   await client.admin.oauth.deleteProvider('provider-id')
   * }
   * ```
   */
  async deleteProvider(providerId: string): Promise<DeleteOAuthProviderResponse> {
    return await this.fetch.delete<DeleteOAuthProviderResponse>(
      `/api/v1/admin/oauth/providers/${providerId}`
    )
  }

  /**
   * Enable an OAuth provider
   *
   * Convenience method to enable a provider.
   *
   * @param providerId - Provider ID (UUID)
   * @returns Promise resolving to UpdateOAuthProviderResponse
   *
   * @example
   * ```typescript
   * await client.admin.oauth.enableProvider('provider-id')
   * ```
   */
  async enableProvider(providerId: string): Promise<UpdateOAuthProviderResponse> {
    return await this.updateProvider(providerId, { enabled: true })
  }

  /**
   * Disable an OAuth provider
   *
   * Convenience method to disable a provider.
   *
   * @param providerId - Provider ID (UUID)
   * @returns Promise resolving to UpdateOAuthProviderResponse
   *
   * @example
   * ```typescript
   * await client.admin.oauth.disableProvider('provider-id')
   * ```
   */
  async disableProvider(providerId: string): Promise<UpdateOAuthProviderResponse> {
    return await this.updateProvider(providerId, { enabled: false })
  }
}

/**
 * Authentication Settings Manager
 *
 * Manages global authentication settings including password requirements, session timeouts,
 * and signup configuration.
 *
 * @example
 * ```typescript
 * const authSettings = client.admin.authSettings
 *
 * // Get current settings
 * const settings = await authSettings.get()
 *
 * // Update settings
 * await authSettings.update({
 *   password_min_length: 12,
 *   password_require_uppercase: true,
 *   session_timeout_minutes: 120
 * })
 * ```
 */
export class AuthSettingsManager {
  constructor(private fetch: FluxbaseFetch) {}

  /**
   * Get current authentication settings
   *
   * Retrieves all authentication configuration settings.
   *
   * @returns Promise resolving to AuthSettings
   *
   * @example
   * ```typescript
   * const settings = await client.admin.authSettings.get()
   *
   * console.log('Password min length:', settings.password_min_length)
   * console.log('Signup enabled:', settings.enable_signup)
   * console.log('Session timeout:', settings.session_timeout_minutes, 'minutes')
   * ```
   */
  async get(): Promise<AuthSettings> {
    return await this.fetch.get<AuthSettings>('/api/v1/admin/auth/settings')
  }

  /**
   * Update authentication settings
   *
   * Updates one or more authentication settings. All fields are optional - only provided
   * fields will be updated.
   *
   * @param request - Settings to update
   * @returns Promise resolving to UpdateAuthSettingsResponse
   *
   * @example
   * ```typescript
   * // Strengthen password requirements
   * await client.admin.authSettings.update({
   *   password_min_length: 16,
   *   password_require_uppercase: true,
   *   password_require_lowercase: true,
   *   password_require_number: true,
   *   password_require_special: true
   * })
   * ```
   *
   * @example
   * ```typescript
   * // Extend session timeout
   * await client.admin.authSettings.update({
   *   session_timeout_minutes: 240,
   *   max_sessions_per_user: 10
   * })
   * ```
   *
   * @example
   * ```typescript
   * // Disable email verification during development
   * await client.admin.authSettings.update({
   *   require_email_verification: false
   * })
   * ```
   */
  async update(request: UpdateAuthSettingsRequest): Promise<UpdateAuthSettingsResponse> {
    return await this.fetch.put<UpdateAuthSettingsResponse>('/api/v1/admin/auth/settings', request)
  }
}

/**
 * OAuth Configuration Manager
 *
 * Root manager providing access to OAuth provider and authentication settings management.
 *
 * @example
 * ```typescript
 * const oauth = client.admin.oauth
 *
 * // Manage OAuth providers
 * const providers = await oauth.providers.listProviders()
 *
 * // Manage auth settings
 * const settings = await oauth.authSettings.get()
 * ```
 */
export class FluxbaseOAuth {
  public providers: OAuthProviderManager
  public authSettings: AuthSettingsManager

  constructor(fetch: FluxbaseFetch) {
    this.providers = new OAuthProviderManager(fetch)
    this.authSettings = new AuthSettingsManager(fetch)
  }
}
