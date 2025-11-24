import type { FluxbaseFetch } from "./fetch";
import type {
  AdminAuthResponse,
  AdminLoginRequest,
  AdminMeResponse,
  AdminRefreshRequest,
  AdminRefreshResponse,
  AdminSetupRequest,
  AdminSetupStatusResponse,
  DeleteUserResponse,
  EnrichedUser,
  InviteUserRequest,
  InviteUserResponse,
  ListUsersOptions,
  ListUsersResponse,
  ResetUserPasswordResponse,
  DataResponse,
  VoidResponse,
} from "./types";
import { wrapAsync, wrapAsyncVoid } from "./utils/error-handling";
import { FluxbaseSettings, EmailTemplateManager } from "./settings";
import { DDLManager } from "./ddl";
import { FluxbaseOAuth } from "./oauth";
import { ImpersonationManager } from "./impersonation";
import { FluxbaseManagement } from "./management";
import { FluxbaseAdminFunctions } from "./admin-functions";
import { FluxbaseAdminMigrations } from "./admin-migrations";

/**
 * Admin client for managing Fluxbase instance
 */
export class FluxbaseAdmin {
  private fetch: FluxbaseFetch;
  private adminToken: string | null = null;

  /**
   * Settings manager for system and application settings
   */
  public settings: FluxbaseSettings;

  /**
   * DDL manager for database schema and table operations
   */
  public ddl: DDLManager;

  /**
   * OAuth configuration manager for provider and auth settings
   */
  public oauth: FluxbaseOAuth;

  /**
   * Impersonation manager for user impersonation and audit trail
   */
  public impersonation: ImpersonationManager;

  /**
   * Management namespace for API keys, webhooks, and invitations
   */
  public management: FluxbaseManagement;

  /**
   * Email template manager for customizing authentication and notification emails
   */
  public emailTemplates: EmailTemplateManager;

  /**
   * Functions manager for edge function management (create, update, delete, sync)
   */
  public functions: FluxbaseAdminFunctions;

  /**
   * Migrations manager for database migration operations (create, apply, rollback, sync)
   */
  public migrations: FluxbaseAdminMigrations;

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
    this.settings = new FluxbaseSettings(fetch);
    this.ddl = new DDLManager(fetch);
    this.oauth = new FluxbaseOAuth(fetch);
    this.impersonation = new ImpersonationManager(fetch);
    this.management = new FluxbaseManagement(fetch);
    this.emailTemplates = new EmailTemplateManager(fetch);
    this.functions = new FluxbaseAdminFunctions(fetch);
    this.migrations = new FluxbaseAdminMigrations(fetch);
  }

  /**
   * Set admin authentication token
   */
  setToken(token: string) {
    this.adminToken = token;
    this.fetch.setAuthToken(token);
  }

  /**
   * Get current admin token
   */
  getToken(): string | null {
    return this.adminToken;
  }

  /**
   * Clear admin token
   */
  clearToken() {
    this.adminToken = null;
    this.fetch.setAuthToken(null);
  }

  // ============================================================================
  // Admin Authentication
  // ============================================================================

  /**
   * Check if initial admin setup is needed
   *
   * @returns Setup status indicating if initial setup is required
   *
   * @example
   * ```typescript
   * const status = await admin.getSetupStatus();
   * if (status.needs_setup) {
   *   console.log('Initial setup required');
   * }
   * ```
   */
  async getSetupStatus(): Promise<DataResponse<AdminSetupStatusResponse>> {
    return wrapAsync(async () => {
      return await this.fetch.get<AdminSetupStatusResponse>(
        "/api/v1/admin/setup/status",
      );
    });
  }

  /**
   * Perform initial admin setup
   *
   * Creates the first admin user and completes initial setup.
   * This endpoint can only be called once.
   *
   * @param email - Admin email address
   * @param password - Admin password (minimum 12 characters)
   * @param name - Admin display name
   * @returns Authentication response with tokens
   *
   * @example
   * ```typescript
   * const response = await admin.setup({
   *   email: 'admin@example.com',
   *   password: 'SecurePassword123!',
   *   name: 'Admin User'
   * });
   *
   * // Store tokens
   * localStorage.setItem('admin_token', response.access_token);
   * ```
   */
  async setup(request: AdminSetupRequest): Promise<DataResponse<AdminAuthResponse>> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<AdminAuthResponse>(
        "/api/v1/admin/setup",
        request,
      );
      this.setToken(response.access_token);
      return response;
    });
  }

  /**
   * Admin login
   *
   * Authenticate as an admin user
   *
   * @param email - Admin email
   * @param password - Admin password
   * @returns Authentication response with tokens
   *
   * @example
   * ```typescript
   * const response = await admin.login({
   *   email: 'admin@example.com',
   *   password: 'password123'
   * });
   *
   * // Token is automatically set in the client
   * console.log('Logged in as:', response.user.email);
   * ```
   */
  async login(request: AdminLoginRequest): Promise<DataResponse<AdminAuthResponse>> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<AdminAuthResponse>(
        "/api/v1/admin/login",
        request,
      );
      this.setToken(response.access_token);
      return response;
    });
  }

  /**
   * Refresh admin access token
   *
   * @param refreshToken - Refresh token
   * @returns New access and refresh tokens
   *
   * @example
   * ```typescript
   * const refreshToken = localStorage.getItem('admin_refresh_token');
   * const response = await admin.refreshToken({ refresh_token: refreshToken });
   *
   * // Update stored tokens
   * localStorage.setItem('admin_token', response.access_token);
   * localStorage.setItem('admin_refresh_token', response.refresh_token);
   * ```
   */
  async refreshToken(
    request: AdminRefreshRequest,
  ): Promise<DataResponse<AdminRefreshResponse>> {
    return wrapAsync(async () => {
      const response = await this.fetch.post<AdminRefreshResponse>(
        "/api/v1/admin/refresh",
        request,
      );
      this.setToken(response.access_token);
      return response;
    });
  }

  /**
   * Admin logout
   *
   * Invalidates the current admin session
   *
   * @example
   * ```typescript
   * await admin.logout();
   * localStorage.removeItem('admin_token');
   * ```
   */
  async logout(): Promise<VoidResponse> {
    return wrapAsyncVoid(async () => {
      await this.fetch.post<{ message: string }>("/api/v1/admin/logout", {});
      this.clearToken();
    });
  }

  /**
   * Get current admin user information
   *
   * @returns Current admin user details
   *
   * @example
   * ```typescript
   * const { user } = await admin.me();
   * console.log('Logged in as:', user.email);
   * console.log('Role:', user.role);
   * ```
   */
  async me(): Promise<DataResponse<AdminMeResponse>> {
    return wrapAsync(async () => {
      return await this.fetch.get<AdminMeResponse>("/api/v1/admin/me");
    });
  }

  // ============================================================================
  // User Management
  // ============================================================================

  /**
   * List all users
   *
   * @param options - Filter and pagination options
   * @returns List of users with metadata
   *
   * @example
   * ```typescript
   * // List all users
   * const { users, total } = await admin.listUsers();
   *
   * // List with filters
   * const result = await admin.listUsers({
   *   exclude_admins: true,
   *   search: 'john',
   *   limit: 50,
   *   type: 'app'
   * });
   * ```
   */
  async listUsers(options: ListUsersOptions = {}): Promise<DataResponse<ListUsersResponse>> {
    return wrapAsync(async () => {
      const params = new URLSearchParams();

      if (options.exclude_admins !== undefined) {
        params.append("exclude_admins", String(options.exclude_admins));
      }
      if (options.search) {
        params.append("search", options.search);
      }
      if (options.limit !== undefined) {
        params.append("limit", String(options.limit));
      }
      if (options.type) {
        params.append("type", options.type);
      }

      const queryString = params.toString();
      const url = queryString
        ? `/api/v1/admin/users?${queryString}`
        : "/api/v1/admin/users";

      return await this.fetch.get<ListUsersResponse>(url);
    });
  }

  /**
   * Get a user by ID
   *
   * Fetch a single user's details by their user ID
   *
   * @param userId - User ID to fetch
   * @param type - User type ('app' or 'dashboard')
   * @returns User details with metadata
   *
   * @example
   * ```typescript
   * // Get an app user
   * const user = await admin.getUserById('user-123');
   *
   * // Get a dashboard user
   * const dashboardUser = await admin.getUserById('admin-456', 'dashboard');
   * console.log('User email:', dashboardUser.email);
   * console.log('Last login:', dashboardUser.last_login_at);
   * ```
   */
  async getUserById(
    userId: string,
    type: "app" | "dashboard" = "app",
  ): Promise<DataResponse<EnrichedUser>> {
    return wrapAsync(async () => {
      const url = `/api/v1/admin/users/${userId}?type=${type}`;
      return await this.fetch.get<EnrichedUser>(url);
    });
  }

  /**
   * Invite a new user
   *
   * Creates a new user and optionally sends an invitation email
   *
   * @param request - User invitation details
   * @param type - User type ('app' or 'dashboard')
   * @returns Created user and invitation details
   *
   * @example
   * ```typescript
   * const response = await admin.inviteUser({
   *   email: 'newuser@example.com',
   *   role: 'user',
   *   send_email: true
   * });
   *
   * console.log('User invited:', response.user.email);
   * console.log('Invitation link:', response.invitation_link);
   * ```
   */
  async inviteUser(
    request: InviteUserRequest,
    type: "app" | "dashboard" = "app",
  ): Promise<DataResponse<InviteUserResponse>> {
    return wrapAsync(async () => {
      const url = `/api/v1/admin/users/invite?type=${type}`;
      return await this.fetch.post<InviteUserResponse>(url, request);
    });
  }

  /**
   * Delete a user
   *
   * Permanently deletes a user and all associated data
   *
   * @param userId - User ID to delete
   * @param type - User type ('app' or 'dashboard')
   * @returns Deletion confirmation
   *
   * @example
   * ```typescript
   * await admin.deleteUser('user-uuid');
   * console.log('User deleted');
   * ```
   */
  async deleteUser(
    userId: string,
    type: "app" | "dashboard" = "app",
  ): Promise<DataResponse<DeleteUserResponse>> {
    return wrapAsync(async () => {
      const url = `/api/v1/admin/users/${userId}?type=${type}`;
      return await this.fetch.delete<DeleteUserResponse>(url);
    });
  }

  /**
   * Update user role
   *
   * Changes a user's role
   *
   * @param userId - User ID
   * @param role - New role
   * @param type - User type ('app' or 'dashboard')
   * @returns Updated user
   *
   * @example
   * ```typescript
   * const user = await admin.updateUserRole('user-uuid', 'admin');
   * console.log('User role updated:', user.role);
   * ```
   */
  async updateUserRole(
    userId: string,
    role: string,
    type: "app" | "dashboard" = "app",
  ): Promise<DataResponse<EnrichedUser>> {
    return wrapAsync(async () => {
      const url = `/api/v1/admin/users/${userId}/role?type=${type}`;
      return await this.fetch.patch<EnrichedUser>(url, { role });
    });
  }

  /**
   * Reset user password
   *
   * Generates a new password for the user and optionally sends it via email
   *
   * @param userId - User ID
   * @param type - User type ('app' or 'dashboard')
   * @returns Reset confirmation message
   *
   * @example
   * ```typescript
   * const response = await admin.resetUserPassword('user-uuid');
   * console.log(response.message);
   * ```
   */
  async resetUserPassword(
    userId: string,
    type: "app" | "dashboard" = "app",
  ): Promise<DataResponse<ResetUserPasswordResponse>> {
    return wrapAsync(async () => {
      const url = `/api/v1/admin/users/${userId}/reset-password?type=${type}`;
      return await this.fetch.post<ResetUserPasswordResponse>(url, {});
    });
  }
}
