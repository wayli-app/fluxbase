import type { FluxbaseFetch } from './fetch'
import type {
  // API Keys
  APIKey,
  CreateAPIKeyRequest,
  CreateAPIKeyResponse,
  DeleteAPIKeyResponse,
  ListAPIKeysResponse,
  RevokeAPIKeyResponse,
  UpdateAPIKeyRequest,
  // Webhooks
  CreateWebhookRequest,
  DeleteWebhookResponse,
  ListWebhookDeliveriesResponse,
  ListWebhooksResponse,
  TestWebhookResponse,
  UpdateWebhookRequest,
  Webhook,
  // Invitations
  AcceptInvitationRequest,
  AcceptInvitationResponse,
  CreateInvitationRequest,
  CreateInvitationResponse,
  ListInvitationsOptions,
  ListInvitationsResponse,
  RevokeInvitationResponse,
  ValidateInvitationResponse,
} from './types'

/**
 * API Keys management client
 *
 * Provides methods for managing API keys for service-to-service authentication.
 * API keys allow external services to authenticate without user credentials.
 *
 * @example
 * ```typescript
 * const client = createClient({ url: 'http://localhost:8080' })
 * await client.auth.login({ email: 'user@example.com', password: 'password' })
 *
 * // Create an API key
 * const { api_key, key } = await client.management.apiKeys.create({
 *   name: 'Production Service',
 *   scopes: ['read:users', 'write:users'],
 *   rate_limit_per_minute: 100
 * })
 *
 * // List API keys
 * const { api_keys } = await client.management.apiKeys.list()
 * ```
 *
 * @category Management
 */
export class APIKeysManager {
  private fetch: FluxbaseFetch

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch
  }

  /**
   * Create a new API key
   *
   * @param request - API key configuration
   * @returns Created API key with the full key value (only shown once)
   *
   * @example
   * ```typescript
   * const { api_key, key } = await client.management.apiKeys.create({
   *   name: 'Production Service',
   *   description: 'API key for production service',
   *   scopes: ['read:users', 'write:users'],
   *   rate_limit_per_minute: 100,
   *   expires_at: '2025-12-31T23:59:59Z'
   * })
   *
   * // Store the key securely - it won't be shown again
   * console.log('API Key:', key)
   * ```
   */
  async create(request: CreateAPIKeyRequest): Promise<CreateAPIKeyResponse> {
    return await this.fetch.post<CreateAPIKeyResponse>('/api/v1/api-keys', request)
  }

  /**
   * List all API keys for the authenticated user
   *
   * @returns List of API keys (without full key values)
   *
   * @example
   * ```typescript
   * const { api_keys, total } = await client.management.apiKeys.list()
   *
   * api_keys.forEach(key => {
   *   console.log(`${key.name}: ${key.key_prefix}... (expires: ${key.expires_at})`)
   * })
   * ```
   */
  async list(): Promise<ListAPIKeysResponse> {
    return await this.fetch.get<ListAPIKeysResponse>('/api/v1/api-keys')
  }

  /**
   * Get a specific API key by ID
   *
   * @param keyId - API key ID
   * @returns API key details
   *
   * @example
   * ```typescript
   * const apiKey = await client.management.apiKeys.get('key-uuid')
   * console.log('Last used:', apiKey.last_used_at)
   * ```
   */
  async get(keyId: string): Promise<APIKey> {
    return await this.fetch.get<APIKey>(`/api/v1/api-keys/${keyId}`)
  }

  /**
   * Update an API key
   *
   * @param keyId - API key ID
   * @param updates - Fields to update
   * @returns Updated API key
   *
   * @example
   * ```typescript
   * const updated = await client.management.apiKeys.update('key-uuid', {
   *   name: 'Updated Name',
   *   rate_limit_per_minute: 200
   * })
   * ```
   */
  async update(keyId: string, updates: UpdateAPIKeyRequest): Promise<APIKey> {
    return await this.fetch.patch<APIKey>(`/api/v1/api-keys/${keyId}`, updates)
  }

  /**
   * Revoke an API key
   *
   * Revoked keys can no longer be used but remain in the system for audit purposes.
   *
   * @param keyId - API key ID
   * @returns Revocation confirmation
   *
   * @example
   * ```typescript
   * await client.management.apiKeys.revoke('key-uuid')
   * console.log('API key revoked')
   * ```
   */
  async revoke(keyId: string): Promise<RevokeAPIKeyResponse> {
    return await this.fetch.post<RevokeAPIKeyResponse>(`/api/v1/api-keys/${keyId}/revoke`, {})
  }

  /**
   * Delete an API key
   *
   * Permanently removes the API key from the system.
   *
   * @param keyId - API key ID
   * @returns Deletion confirmation
   *
   * @example
   * ```typescript
   * await client.management.apiKeys.delete('key-uuid')
   * console.log('API key deleted')
   * ```
   */
  async delete(keyId: string): Promise<DeleteAPIKeyResponse> {
    return await this.fetch.delete<DeleteAPIKeyResponse>(`/api/v1/api-keys/${keyId}`)
  }
}

/**
 * Webhooks management client
 *
 * Provides methods for managing webhooks to receive real-time event notifications.
 * Webhooks allow your application to be notified when events occur in Fluxbase.
 *
 * @example
 * ```typescript
 * const client = createClient({ url: 'http://localhost:8080' })
 * await client.auth.login({ email: 'user@example.com', password: 'password' })
 *
 * // Create a webhook
 * const webhook = await client.management.webhooks.create({
 *   url: 'https://myapp.com/webhook',
 *   events: ['user.created', 'user.updated'],
 *   secret: 'my-webhook-secret'
 * })
 *
 * // Test the webhook
 * const result = await client.management.webhooks.test(webhook.id)
 * ```
 *
 * @category Management
 */
export class WebhooksManager {
  private fetch: FluxbaseFetch

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch
  }

  /**
   * Create a new webhook
   *
   * @param request - Webhook configuration
   * @returns Created webhook
   *
   * @example
   * ```typescript
   * const webhook = await client.management.webhooks.create({
   *   url: 'https://myapp.com/webhook',
   *   events: ['user.created', 'user.updated', 'user.deleted'],
   *   description: 'User events webhook',
   *   secret: 'my-webhook-secret'
   * })
   * ```
   */
  async create(request: CreateWebhookRequest): Promise<Webhook> {
    return await this.fetch.post<Webhook>('/api/v1/webhooks', request)
  }

  /**
   * List all webhooks for the authenticated user
   *
   * @returns List of webhooks
   *
   * @example
   * ```typescript
   * const { webhooks, total } = await client.management.webhooks.list()
   *
   * webhooks.forEach(webhook => {
   *   console.log(`${webhook.url}: ${webhook.is_active ? 'active' : 'inactive'}`)
   * })
   * ```
   */
  async list(): Promise<ListWebhooksResponse> {
    return await this.fetch.get<ListWebhooksResponse>('/api/v1/webhooks')
  }

  /**
   * Get a specific webhook by ID
   *
   * @param webhookId - Webhook ID
   * @returns Webhook details
   *
   * @example
   * ```typescript
   * const webhook = await client.management.webhooks.get('webhook-uuid')
   * console.log('Events:', webhook.events)
   * ```
   */
  async get(webhookId: string): Promise<Webhook> {
    return await this.fetch.get<Webhook>(`/api/v1/webhooks/${webhookId}`)
  }

  /**
   * Update a webhook
   *
   * @param webhookId - Webhook ID
   * @param updates - Fields to update
   * @returns Updated webhook
   *
   * @example
   * ```typescript
   * const updated = await client.management.webhooks.update('webhook-uuid', {
   *   events: ['user.created', 'user.deleted'],
   *   is_active: false
   * })
   * ```
   */
  async update(webhookId: string, updates: UpdateWebhookRequest): Promise<Webhook> {
    return await this.fetch.patch<Webhook>(`/api/v1/webhooks/${webhookId}`, updates)
  }

  /**
   * Delete a webhook
   *
   * @param webhookId - Webhook ID
   * @returns Deletion confirmation
   *
   * @example
   * ```typescript
   * await client.management.webhooks.delete('webhook-uuid')
   * console.log('Webhook deleted')
   * ```
   */
  async delete(webhookId: string): Promise<DeleteWebhookResponse> {
    return await this.fetch.delete<DeleteWebhookResponse>(`/api/v1/webhooks/${webhookId}`)
  }

  /**
   * Test a webhook by sending a test payload
   *
   * @param webhookId - Webhook ID
   * @returns Test result with status and response
   *
   * @example
   * ```typescript
   * const result = await client.management.webhooks.test('webhook-uuid')
   *
   * if (result.success) {
   *   console.log('Webhook test successful')
   * } else {
   *   console.error('Webhook test failed:', result.error)
   * }
   * ```
   */
  async test(webhookId: string): Promise<TestWebhookResponse> {
    return await this.fetch.post<TestWebhookResponse>(`/api/v1/webhooks/${webhookId}/test`, {})
  }

  /**
   * List webhook delivery history
   *
   * @param webhookId - Webhook ID
   * @param limit - Maximum number of deliveries to return (default: 50)
   * @returns List of webhook deliveries
   *
   * @example
   * ```typescript
   * const { deliveries } = await client.management.webhooks.listDeliveries('webhook-uuid', 100)
   *
   * deliveries.forEach(delivery => {
   *   console.log(`Event: ${delivery.event}, Status: ${delivery.status_code}`)
   * })
   * ```
   */
  async listDeliveries(webhookId: string, limit: number = 50): Promise<ListWebhookDeliveriesResponse> {
    return await this.fetch.get<ListWebhookDeliveriesResponse>(
      `/api/v1/webhooks/${webhookId}/deliveries?limit=${limit}`,
    )
  }
}

/**
 * Invitations management client
 *
 * Provides methods for creating and managing user invitations.
 * Invitations allow admins to invite new users to join the dashboard.
 *
 * @example
 * ```typescript
 * const client = createClient({ url: 'http://localhost:8080' })
 * await client.admin.login({ email: 'admin@example.com', password: 'password' })
 *
 * // Create an invitation
 * const invitation = await client.management.invitations.create({
 *   email: 'newuser@example.com',
 *   role: 'dashboard_user'
 * })
 *
 * console.log('Invite link:', invitation.invite_link)
 * ```
 *
 * @category Management
 */
export class InvitationsManager {
  private fetch: FluxbaseFetch

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch
  }

  /**
   * Create a new invitation (admin only)
   *
   * @param request - Invitation details
   * @returns Created invitation with invite link
   *
   * @example
   * ```typescript
   * const invitation = await client.management.invitations.create({
   *   email: 'newuser@example.com',
   *   role: 'dashboard_user',
   *   expiry_duration: 604800 // 7 days in seconds
   * })
   *
   * // Share the invite link
   * console.log('Send this link to the user:', invitation.invite_link)
   * ```
   */
  async create(request: CreateInvitationRequest): Promise<CreateInvitationResponse> {
    return await this.fetch.post<CreateInvitationResponse>('/api/v1/admin/invitations', request)
  }

  /**
   * List all invitations (admin only)
   *
   * @param options - Filter options
   * @returns List of invitations
   *
   * @example
   * ```typescript
   * // List pending invitations only
   * const { invitations } = await client.management.invitations.list({
   *   include_accepted: false,
   *   include_expired: false
   * })
   *
   * // List all invitations including accepted and expired
   * const all = await client.management.invitations.list({
   *   include_accepted: true,
   *   include_expired: true
   * })
   * ```
   */
  async list(options: ListInvitationsOptions = {}): Promise<ListInvitationsResponse> {
    const params = new URLSearchParams()

    if (options.include_accepted !== undefined) {
      params.append('include_accepted', String(options.include_accepted))
    }
    if (options.include_expired !== undefined) {
      params.append('include_expired', String(options.include_expired))
    }

    const queryString = params.toString()
    const url = queryString ? `/api/v1/admin/invitations?${queryString}` : '/api/v1/admin/invitations'

    return await this.fetch.get<ListInvitationsResponse>(url)
  }

  /**
   * Validate an invitation token (public endpoint)
   *
   * @param token - Invitation token
   * @returns Validation result with invitation details
   *
   * @example
   * ```typescript
   * const result = await client.management.invitations.validate('invitation-token')
   *
   * if (result.valid) {
   *   console.log('Valid invitation for:', result.invitation?.email)
   * } else {
   *   console.error('Invalid:', result.error)
   * }
   * ```
   */
  async validate(token: string): Promise<ValidateInvitationResponse> {
    return await this.fetch.get<ValidateInvitationResponse>(`/api/v1/invitations/${token}/validate`)
  }

  /**
   * Accept an invitation and create a new user (public endpoint)
   *
   * @param token - Invitation token
   * @param request - User details (password and name)
   * @returns Created user with authentication tokens
   *
   * @example
   * ```typescript
   * const response = await client.management.invitations.accept('invitation-token', {
   *   password: 'SecurePassword123!',
   *   name: 'John Doe'
   * })
   *
   * // Store tokens
   * localStorage.setItem('access_token', response.access_token)
   * console.log('Welcome:', response.user.name)
   * ```
   */
  async accept(token: string, request: AcceptInvitationRequest): Promise<AcceptInvitationResponse> {
    return await this.fetch.post<AcceptInvitationResponse>(`/api/v1/invitations/${token}/accept`, request)
  }

  /**
   * Revoke an invitation (admin only)
   *
   * @param token - Invitation token
   * @returns Revocation confirmation
   *
   * @example
   * ```typescript
   * await client.management.invitations.revoke('invitation-token')
   * console.log('Invitation revoked')
   * ```
   */
  async revoke(token: string): Promise<RevokeInvitationResponse> {
    return await this.fetch.delete<RevokeInvitationResponse>(`/api/v1/admin/invitations/${token}`)
  }
}

/**
 * Management client for API keys, webhooks, and invitations
 *
 * @category Management
 */
export class FluxbaseManagement {
  /** API Keys management */
  public apiKeys: APIKeysManager

  /** Webhooks management */
  public webhooks: WebhooksManager

  /** Invitations management */
  public invitations: InvitationsManager

  constructor(fetch: FluxbaseFetch) {
    this.apiKeys = new APIKeysManager(fetch)
    this.webhooks = new WebhooksManager(fetch)
    this.invitations = new InvitationsManager(fetch)
  }
}
