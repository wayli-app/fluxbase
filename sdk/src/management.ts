import type { FluxbaseFetch } from './fetch'
import type {
  // Client Keys
  ClientKey,
  CreateClientKeyRequest,
  CreateClientKeyResponse,
  DeleteClientKeyResponse,
  ListClientKeysResponse,
  RevokeClientKeyResponse,
  UpdateClientKeyRequest,
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
 * Client Keys management client
 *
 * Provides methods for managing client keys for service-to-service authentication.
 * Client keys allow external services to authenticate without user credentials.
 *
 * @example
 * ```typescript
 * const client = createClient({ url: 'http://localhost:8080' })
 * await client.auth.login({ email: 'user@example.com', password: 'password' })
 *
 * // Create a client key
 * const { client_key, key } = await client.management.clientKeys.create({
 *   name: 'Production Service',
 *   scopes: ['read:users', 'write:users'],
 *   rate_limit_per_minute: 100
 * })
 *
 * // List client keys
 * const { client_keys } = await client.management.clientKeys.list()
 * ```
 *
 * @category Management
 */
export class ClientKeysManager {
  private fetch: FluxbaseFetch

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch
  }

  /**
   * Create a new client key
   *
   * @param request - Client key configuration
   * @returns Created client key with the full key value (only shown once)
   *
   * @example
   * ```typescript
   * const { client_key, key } = await client.management.clientKeys.create({
   *   name: 'Production Service',
   *   description: 'Client key for production service',
   *   scopes: ['read:users', 'write:users'],
   *   rate_limit_per_minute: 100,
   *   expires_at: '2025-12-31T23:59:59Z'
   * })
   *
   * // Store the key securely - it won't be shown again
   * console.log('Client Key:', key)
   * ```
   */
  async create(request: CreateClientKeyRequest): Promise<CreateClientKeyResponse> {
    return await this.fetch.post<CreateClientKeyResponse>('/api/v1/client-keys', request)
  }

  /**
   * List all client keys for the authenticated user
   *
   * @returns List of client keys (without full key values)
   *
   * @example
   * ```typescript
   * const { client_keys, total } = await client.management.clientKeys.list()
   *
   * client_keys.forEach(key => {
   *   console.log(`${key.name}: ${key.key_prefix}... (expires: ${key.expires_at})`)
   * })
   * ```
   */
  async list(): Promise<ListClientKeysResponse> {
    return await this.fetch.get<ListClientKeysResponse>('/api/v1/client-keys')
  }

  /**
   * Get a specific client key by ID
   *
   * @param keyId - Client key ID
   * @returns Client key details
   *
   * @example
   * ```typescript
   * const clientKey = await client.management.clientKeys.get('key-uuid')
   * console.log('Last used:', clientKey.last_used_at)
   * ```
   */
  async get(keyId: string): Promise<ClientKey> {
    return await this.fetch.get<ClientKey>(`/api/v1/client-keys/${keyId}`)
  }

  /**
   * Update a client key
   *
   * @param keyId - Client key ID
   * @param updates - Fields to update
   * @returns Updated client key
   *
   * @example
   * ```typescript
   * const updated = await client.management.clientKeys.update('key-uuid', {
   *   name: 'Updated Name',
   *   rate_limit_per_minute: 200
   * })
   * ```
   */
  async update(keyId: string, updates: UpdateClientKeyRequest): Promise<ClientKey> {
    return await this.fetch.patch<ClientKey>(`/api/v1/client-keys/${keyId}`, updates)
  }

  /**
   * Revoke a client key
   *
   * Revoked keys can no longer be used but remain in the system for audit purposes.
   *
   * @param keyId - Client key ID
   * @returns Revocation confirmation
   *
   * @example
   * ```typescript
   * await client.management.clientKeys.revoke('key-uuid')
   * console.log('Client key revoked')
   * ```
   */
  async revoke(keyId: string): Promise<RevokeClientKeyResponse> {
    return await this.fetch.post<RevokeClientKeyResponse>(`/api/v1/client-keys/${keyId}/revoke`, {})
  }

  /**
   * Delete a client key
   *
   * Permanently removes the client key from the system.
   *
   * @param keyId - Client key ID
   * @returns Deletion confirmation
   *
   * @example
   * ```typescript
   * await client.management.clientKeys.delete('key-uuid')
   * console.log('Client key deleted')
   * ```
   */
  async delete(keyId: string): Promise<DeleteClientKeyResponse> {
    return await this.fetch.delete<DeleteClientKeyResponse>(`/api/v1/client-keys/${keyId}`)
  }
}

/**
 * @deprecated Use ClientKeysManager instead
 */
export const APIKeysManager = ClientKeysManager

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
 * Management client for client keys, webhooks, and invitations
 *
 * @category Management
 */
export class FluxbaseManagement {
  /** Client Keys management */
  public clientKeys: ClientKeysManager

  /** @deprecated Use clientKeys instead */
  public apiKeys: ClientKeysManager

  /** Webhooks management */
  public webhooks: WebhooksManager

  /** Invitations management */
  public invitations: InvitationsManager

  constructor(fetch: FluxbaseFetch) {
    this.clientKeys = new ClientKeysManager(fetch)
    this.apiKeys = this.clientKeys // Backwards compatibility alias
    this.webhooks = new WebhooksManager(fetch)
    this.invitations = new InvitationsManager(fetch)
  }
}
