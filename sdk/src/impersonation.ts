import type { FluxbaseFetch } from './fetch'
import type {
  ImpersonateUserRequest,
  ImpersonateAnonRequest,
  ImpersonateServiceRequest,
  StartImpersonationResponse,
  StopImpersonationResponse,
  GetImpersonationResponse,
  ListImpersonationSessionsOptions,
  ListImpersonationSessionsResponse,
} from './types'

/**
 * Impersonation Manager
 *
 * Manages user impersonation for debugging, testing RLS policies, and customer support.
 * Allows admins to view data as different users, anonymous visitors, or with service role permissions.
 *
 * All impersonation sessions are logged in the audit trail for security and compliance.
 *
 * @example
 * ```typescript
 * const impersonation = client.admin.impersonation
 *
 * // Impersonate a specific user
 * const { session, access_token } = await impersonation.impersonateUser({
 *   target_user_id: 'user-uuid',
 *   reason: 'Support ticket #1234'
 * })
 *
 * // Impersonate anonymous user
 * await impersonation.impersonateAnon({
 *   reason: 'Testing public data access'
 * })
 *
 * // Impersonate with service role
 * await impersonation.impersonateService({
 *   reason: 'Administrative query'
 * })
 *
 * // Stop impersonation
 * await impersonation.stop()
 * ```
 */
export class ImpersonationManager {
  constructor(private fetch: FluxbaseFetch) {}

  /**
   * Impersonate a specific user
   *
   * Start an impersonation session as a specific user. This allows you to see data
   * exactly as that user would see it, respecting all RLS policies and permissions.
   *
   * @param request - Impersonation request with target user ID and reason
   * @returns Promise resolving to impersonation session with access token
   *
   * @example
   * ```typescript
   * const result = await client.admin.impersonation.impersonateUser({
   *   target_user_id: 'user-123',
   *   reason: 'Support ticket #5678 - user reports missing data'
   * })
   *
   * console.log('Impersonating:', result.target_user.email)
   * console.log('Session ID:', result.session.id)
   *
   * // Use the access token for subsequent requests
   * // (typically handled automatically by the SDK)
   * ```
   */
  async impersonateUser(request: ImpersonateUserRequest): Promise<StartImpersonationResponse> {
    return await this.fetch.post<StartImpersonationResponse>('/api/v1/auth/impersonate', request)
  }

  /**
   * Impersonate anonymous user
   *
   * Start an impersonation session as an unauthenticated user. This allows you to see
   * what data is publicly accessible and test RLS policies for anonymous access.
   *
   * @param request - Impersonation request with reason
   * @returns Promise resolving to impersonation session with access token
   *
   * @example
   * ```typescript
   * await client.admin.impersonation.impersonateAnon({
   *   reason: 'Testing public data access for blog posts'
   * })
   *
   * // Now all queries will use anonymous permissions
   * const publicPosts = await client.from('posts').select('*')
   * console.log('Public posts:', publicPosts.length)
   * ```
   */
  async impersonateAnon(request: ImpersonateAnonRequest): Promise<StartImpersonationResponse> {
    return await this.fetch.post<StartImpersonationResponse>('/api/v1/auth/impersonate/anon', request)
  }

  /**
   * Impersonate with service role
   *
   * Start an impersonation session with service-level permissions. This provides elevated
   * access that may bypass RLS policies, useful for administrative operations.
   *
   * @param request - Impersonation request with reason
   * @returns Promise resolving to impersonation session with access token
   *
   * @example
   * ```typescript
   * await client.admin.impersonation.impersonateService({
   *   reason: 'Administrative data cleanup'
   * })
   *
   * // Now all queries will use service role permissions
   * const allRecords = await client.from('sensitive_data').select('*')
   * console.log('All records:', allRecords.length)
   * ```
   */
  async impersonateService(request: ImpersonateServiceRequest): Promise<StartImpersonationResponse> {
    return await this.fetch.post<StartImpersonationResponse>('/api/v1/auth/impersonate/service', request)
  }

  /**
   * Stop impersonation
   *
   * Ends the current impersonation session and returns to admin context.
   * The session is marked as ended in the audit trail.
   *
   * @returns Promise resolving to stop confirmation
   *
   * @example
   * ```typescript
   * await client.admin.impersonation.stop()
   * console.log('Impersonation ended')
   *
   * // Subsequent queries will use admin permissions
   * ```
   */
  async stop(): Promise<StopImpersonationResponse> {
    return await this.fetch.delete<StopImpersonationResponse>('/api/v1/auth/impersonate')
  }

  /**
   * Get current impersonation session
   *
   * Retrieves information about the active impersonation session, if any.
   *
   * @returns Promise resolving to current impersonation session or null
   *
   * @example
   * ```typescript
   * const current = await client.admin.impersonation.getCurrent()
   *
   * if (current.session) {
   *   console.log('Currently impersonating:', current.target_user?.email)
   *   console.log('Reason:', current.session.reason)
   *   console.log('Started:', current.session.started_at)
   * } else {
   *   console.log('No active impersonation')
   * }
   * ```
   */
  async getCurrent(): Promise<GetImpersonationResponse> {
    return await this.fetch.get<GetImpersonationResponse>('/api/v1/auth/impersonate')
  }

  /**
   * List impersonation sessions (audit trail)
   *
   * Retrieves a list of impersonation sessions for audit and compliance purposes.
   * Can be filtered by admin user, target user, type, and active status.
   *
   * @param options - Filter and pagination options
   * @returns Promise resolving to list of impersonation sessions
   *
   * @example
   * ```typescript
   * // List all sessions
   * const { sessions, total } = await client.admin.impersonation.listSessions()
   * console.log(`Total sessions: ${total}`)
   *
   * // List active sessions only
   * const active = await client.admin.impersonation.listSessions({
   *   is_active: true
   * })
   * console.log('Active sessions:', active.sessions.length)
   *
   * // List sessions for a specific admin
   * const adminSessions = await client.admin.impersonation.listSessions({
   *   admin_user_id: 'admin-uuid',
   *   limit: 50
   * })
   *
   * // List user impersonation sessions only
   * const userSessions = await client.admin.impersonation.listSessions({
   *   impersonation_type: 'user',
   *   offset: 0,
   *   limit: 100
   * })
   * ```
   *
   * @example
   * ```typescript
   * // Audit trail: Find who impersonated a specific user
   * const userHistory = await client.admin.impersonation.listSessions({
   *   target_user_id: 'user-uuid'
   * })
   *
   * userHistory.sessions.forEach(session => {
   *   console.log(`Admin ${session.admin_user_id} impersonated user`)
   *   console.log(`Reason: ${session.reason}`)
   *   console.log(`Duration: ${session.started_at} - ${session.ended_at}`)
   * })
   * ```
   */
  async listSessions(options: ListImpersonationSessionsOptions = {}): Promise<ListImpersonationSessionsResponse> {
    const params = new URLSearchParams()

    if (options.limit !== undefined) {
      params.append('limit', String(options.limit))
    }
    if (options.offset !== undefined) {
      params.append('offset', String(options.offset))
    }
    if (options.admin_user_id) {
      params.append('admin_user_id', options.admin_user_id)
    }
    if (options.target_user_id) {
      params.append('target_user_id', options.target_user_id)
    }
    if (options.impersonation_type) {
      params.append('impersonation_type', options.impersonation_type)
    }
    if (options.is_active !== undefined) {
      params.append('is_active', String(options.is_active))
    }

    const queryString = params.toString()
    const url = queryString ? `/api/v1/auth/impersonate/sessions?${queryString}` : '/api/v1/auth/impersonate/sessions'

    return await this.fetch.get<ListImpersonationSessionsResponse>(url)
  }
}
