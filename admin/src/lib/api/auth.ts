import axios from "axios";
import { api } from "./client";
import { getAccessToken, type AdminUser } from "../auth";

const API_BASE_URL =
  window.__FLUXBASE_CONFIG__?.publicBaseURL ||
  import.meta.env.VITE_API_URL ||
  "";

export interface User {
  id: string;
  email: string;
  email_verified: boolean;
  role: string;
  metadata: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

export interface SignInRequest {
  email: string;
  password: string;
}

export interface SignInResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface SignUpRequest {
  email: string;
  password: string;
  metadata?: Record<string, unknown>;
}

export interface SignUpResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export const authApi = {
  signIn: async (data: SignInRequest): Promise<SignInResponse> => {
    const response = await api.post<SignInResponse>(
      "/api/v1/auth/signin",
      data,
    );
    return response.data;
  },

  signUp: async (data: SignUpRequest): Promise<SignUpResponse> => {
    const response = await api.post<SignUpResponse>(
      "/api/v1/auth/signup",
      data,
    );
    return response.data;
  },

  signOut: async (): Promise<void> => {
    await api.post("/api/v1/auth/signout");
  },

  getUser: async (): Promise<User> => {
    const response = await api.get<User>("/api/v1/auth/user");
    return response.data;
  },

  updateUser: async (
    data: Partial<Pick<User, "email" | "metadata">>,
  ): Promise<User> => {
    const response = await api.patch<User>("/api/v1/auth/user", data);
    return response.data;
  },

  requestPasswordReset: async (email: string): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>(
      "/api/v1/auth/password/reset",
      { email },
    );
    return response.data;
  },

  resetPassword: async (
    token: string,
    newPassword: string,
  ): Promise<{ message: string }> => {
    const response = await api.post<{ message: string }>(
      "/api/v1/auth/password/reset/confirm",
      {
        token,
        new_password: newPassword,
      },
    );
    return response.data;
  },

  verifyResetToken: async (
    token: string,
  ): Promise<{ valid: boolean; message?: string }> => {
    try {
      const response = await api.post<{ message: string }>(
        "/api/v1/auth/password/reset/verify",
        { token },
      );
      return { valid: true, message: response.data.message };
    } catch {
      return { valid: false, message: "Invalid or expired token" };
    }
  },
};

export const healthApi = {
  check: async (): Promise<{
    status: string;
    services?: { database: boolean; realtime: boolean };
    timestamp: string;
  }> => {
    const response = await api.get("/health");
    return response.data;
  },
};

export const adminAuthAPI = {
  getSetupStatus: async (): Promise<{
    needs_setup: boolean;
    has_admin: boolean;
  }> => {
    const response = await axios.get(
      `${API_BASE_URL}/api/v1/admin/setup/status`,
    );
    return response.data;
  },

  initialSetup: async (data: {
    email: string;
    password: string;
    name: string;
    setup_token?: string;
  }): Promise<{
    user: AdminUser;
    access_token: string;
    refresh_token: string;
    expires_in: number;
  }> => {
    const setupAxios = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        "Content-Type": "application/json",
      },
      timeout: 30000,
    });

    setupAxios.interceptors.request.use((config) => {
      return config;
    });

    setupAxios.interceptors.response.use(
      (response) => {
        return response;
      },
      (error) => {
        return Promise.reject(error);
      },
    );

    const response = await setupAxios.post("/api/v1/admin/setup", data);
    return response.data;
  },

  login: async (credentials: {
    email: string;
    password: string;
  }): Promise<{
    user: AdminUser;
    access_token: string;
    refresh_token: string;
    expires_in: number;
  }> => {
    const response = await axios.post(
      `${API_BASE_URL}/api/v1/admin/login`,
      credentials,
    );
    return response.data;
  },

  logout: async (): Promise<{ message: string }> => {
    const response = await api.post("/api/v1/admin/logout");
    return response.data;
  },

  me: async (): Promise<{ user: AdminUser }> => {
    const response = await api.get("/api/v1/admin/me");
    return response.data;
  },
};

export interface DashboardUser {
  id: string;
  email: string;
  email_verified: boolean;
  full_name: string | null;
  avatar_url: string | null;
  totp_enabled: boolean;
  is_active: boolean;
  is_locked: boolean;
  last_login_at: string | null;
  created_at: string;
  updated_at: string;
  role?: string;
}

export interface DashboardSignupRequest {
  email: string;
  password: string;
  full_name: string;
}

export interface DashboardLoginRequest {
  email: string;
  password: string;
}

export interface DashboardLoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: DashboardUser;
  requires_2fa?: boolean;
  user_id?: string;
}

export interface UpdateProfileRequest {
  full_name: string;
  avatar_url?: string | null;
}

export interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
}

export interface DeleteAccountRequest {
  password: string;
}

export interface Setup2FAResponse {
  secret: string;
  qr_url: string;
}

export interface Enable2FARequest {
  code: string;
}

export interface Enable2FAResponse {
  message: string;
  backup_codes: string[];
}

export interface Verify2FARequest {
  user_id: string;
  code: string;
}

export interface Disable2FARequest {
  password: string;
}

const getDashboardAuthHeaders = (): Record<string, string> => {
  const token = getAccessToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
};

// Note: Uses raw axios instead of shared `api` instance because /dashboard/auth/me
// uses a different auth path (service key or admin JWT, not user JWT).
// Tenant/branch headers are intentionally NOT included.
export const dashboardAuthAPI = {
  signup: async (
    data: DashboardSignupRequest,
  ): Promise<{ user: DashboardUser; message: string }> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/signup`,
      data,
    );
    return response.data;
  },

  login: async (
    credentials: DashboardLoginRequest,
  ): Promise<DashboardLoginResponse> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/login`,
      credentials,
    );
    return response.data;
  },

  me: async (): Promise<DashboardUser> => {
    const response = await axios.get(`${API_BASE_URL}/dashboard/auth/me`, {
      headers: getDashboardAuthHeaders(),
    });
    return response.data;
  },

  updateProfile: async (data: UpdateProfileRequest): Promise<DashboardUser> => {
    const response = await axios.put(
      `${API_BASE_URL}/dashboard/auth/profile`,
      data,
      { headers: getDashboardAuthHeaders() },
    );
    return response.data;
  },

  changePassword: async (
    data: ChangePasswordRequest,
  ): Promise<{ message: string }> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/password/change`,
      data,
      { headers: getDashboardAuthHeaders() },
    );
    return response.data;
  },

  deleteAccount: async (
    data: DeleteAccountRequest,
  ): Promise<{ message: string }> => {
    const response = await axios.delete(
      `${API_BASE_URL}/dashboard/auth/account`,
      { data, headers: getDashboardAuthHeaders() },
    );
    return response.data;
  },

  setup2FA: async (): Promise<Setup2FAResponse> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/2fa/setup`,
      {},
      { headers: getDashboardAuthHeaders() },
    );
    return response.data;
  },

  enable2FA: async (data: Enable2FARequest): Promise<Enable2FAResponse> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/2fa/enable`,
      data,
      { headers: getDashboardAuthHeaders() },
    );
    return response.data;
  },

  verify2FA: async (
    data: Verify2FARequest,
  ): Promise<DashboardLoginResponse> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/2fa/verify`,
      data,
    );
    return response.data;
  },

  disable2FA: async (data: Disable2FARequest): Promise<{ message: string }> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/2fa/disable`,
      data,
      { headers: getDashboardAuthHeaders() },
    );
    return response.data;
  },

  requestPasswordReset: async (email: string): Promise<{ message: string }> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/password/reset`,
      { email },
    );
    return response.data;
  },

  verifyResetToken: async (
    token: string,
  ): Promise<{ valid: boolean; message?: string }> => {
    try {
      const response = await axios.post<{ valid: boolean; message: string }>(
        `${API_BASE_URL}/dashboard/auth/password/reset/verify`,
        { token },
      );
      return response.data;
    } catch {
      return { valid: false, message: "Invalid or expired token" };
    }
  },

  resetPassword: async (
    token: string,
    newPassword: string,
  ): Promise<{ message: string }> => {
    const response = await axios.post(
      `${API_BASE_URL}/dashboard/auth/password/reset/confirm`,
      {
        token,
        new_password: newPassword,
      },
    );
    return response.data;
  },

  getSSOProviders: async (): Promise<{
    providers: SSOProvider[];
    password_login_disabled: boolean;
  }> => {
    const response = await axios.get(
      `${API_BASE_URL}/dashboard/auth/sso/providers`,
    );
    return response.data;
  },
};

export interface SSOProvider {
  id: string;
  name: string;
  type: "oauth" | "saml";
  provider?: string;
}

export interface OAuthProviderConfig {
  id: string;
  provider_name: string;
  display_name: string;
  enabled: boolean;
  client_id: string;
  client_secret?: string;
  has_secret: boolean;
  redirect_url: string;
  scopes: string[];
  is_custom: boolean;
  authorization_url?: string;
  token_url?: string;
  user_info_url?: string;
  allow_dashboard_login: boolean;
  allow_app_login: boolean;
  required_claims?: Record<string, string[]>;
  denied_claims?: Record<string, string[]>;
  source?: "database" | "config";
  tenant_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateOAuthProviderRequest {
  provider_name: string;
  display_name: string;
  enabled: boolean;
  client_id: string;
  client_secret: string;
  redirect_url: string;
  scopes: string[];
  is_custom: boolean;
  authorization_url?: string;
  token_url?: string;
  user_info_url?: string;
  allow_dashboard_login?: boolean;
  allow_app_login?: boolean;
}

export interface UpdateOAuthProviderRequest {
  display_name?: string;
  enabled?: boolean;
  client_id?: string;
  client_secret?: string;
  redirect_url?: string;
  scopes?: string[];
  authorization_url?: string;
  token_url?: string;
  user_info_url?: string;
  allow_dashboard_login?: boolean;
  allow_app_login?: boolean;
}

export interface AuthSettings {
  enable_signup: boolean;
  require_email_verification: boolean;
  enable_magic_link: boolean;
  password_min_length: number;
  password_require_uppercase: boolean;
  password_require_lowercase: boolean;
  password_require_number: boolean;
  password_require_special: boolean;
  session_timeout_minutes: number;
  max_sessions_per_user: number;
  disable_dashboard_password_login: boolean;
  disable_app_password_login: boolean;
}

export const oauthProviderApi = {
  list: async (): Promise<OAuthProviderConfig[]> => {
    const response = await api.get<OAuthProviderConfig[]>(
      "/api/v1/admin/oauth/providers",
    );
    return response.data;
  },

  get: async (id: string): Promise<OAuthProviderConfig> => {
    const response = await api.get<OAuthProviderConfig>(
      `/api/v1/admin/oauth/providers/${id}`,
    );
    return response.data;
  },

  create: async (
    data: CreateOAuthProviderRequest,
  ): Promise<{
    success: boolean;
    id: string;
    provider: string;
    message: string;
  }> => {
    const response = await api.post("/api/v1/admin/oauth/providers", data);
    return response.data;
  },

  update: async (
    id: string,
    data: UpdateOAuthProviderRequest,
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.put(`/api/v1/admin/oauth/providers/${id}`, data);
    return response.data;
  },

  delete: async (
    id: string,
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.delete(`/api/v1/admin/oauth/providers/${id}`);
    return response.data;
  },
};

export const authSettingsApi = {
  get: async (): Promise<AuthSettings> => {
    const response = await api.get<AuthSettings>("/api/v1/admin/auth/settings");
    return response.data;
  },

  update: async (
    data: AuthSettings,
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.put("/api/v1/admin/auth/settings", data);
    return response.data;
  },
};

export interface SAMLProviderConfig {
  id: string;
  name: string;
  display_name: string;
  enabled: boolean;
  entity_id: string;
  acs_url: string;
  idp_metadata_url?: string;
  idp_metadata_xml?: string;
  idp_entity_id?: string;
  idp_sso_url?: string;
  attribute_mapping: Record<string, string>;
  auto_create_users: boolean;
  default_role: string;
  allow_dashboard_login: boolean;
  allow_app_login: boolean;
  allow_idp_initiated: boolean;
  allowed_redirect_hosts: string[];
  required_groups?: string[];
  required_groups_all?: string[];
  denied_groups?: string[];
  group_attribute?: string;
  source: "database" | "config";
  tenant_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateSAMLProviderRequest {
  name: string;
  display_name?: string;
  enabled: boolean;
  idp_metadata_url?: string;
  idp_metadata_xml?: string;
  attribute_mapping?: Record<string, string>;
  auto_create_users?: boolean;
  default_role?: string;
  allow_dashboard_login?: boolean;
  allow_app_login?: boolean;
  allow_idp_initiated?: boolean;
  allowed_redirect_hosts?: string[];
  required_groups?: string[];
  required_groups_all?: string[];
  denied_groups?: string[];
  group_attribute?: string;
}

export interface UpdateSAMLProviderRequest {
  display_name?: string;
  enabled?: boolean;
  idp_metadata_url?: string;
  idp_metadata_xml?: string;
  attribute_mapping?: Record<string, string>;
  auto_create_users?: boolean;
  default_role?: string;
  allow_dashboard_login?: boolean;
  allow_app_login?: boolean;
  allow_idp_initiated?: boolean;
  allowed_redirect_hosts?: string[];
  required_groups?: string[];
  required_groups_all?: string[];
  denied_groups?: string[];
  group_attribute?: string;
}

export interface ValidateMetadataResponse {
  valid: boolean;
  entity_id?: string;
  sso_url?: string;
  slo_url?: string;
  certificate?: string;
  error?: string;
}

export const samlProviderApi = {
  list: async (): Promise<SAMLProviderConfig[]> => {
    const response = await api.get<SAMLProviderConfig[]>(
      "/api/v1/admin/saml/providers",
    );
    return response.data;
  },

  get: async (id: string): Promise<SAMLProviderConfig> => {
    const response = await api.get<SAMLProviderConfig>(
      `/api/v1/admin/saml/providers/${id}`,
    );
    return response.data;
  },

  create: async (
    data: CreateSAMLProviderRequest,
  ): Promise<{
    success: boolean;
    id: string;
    provider: string;
    entity_id: string;
    acs_url: string;
    message: string;
  }> => {
    const response = await api.post("/api/v1/admin/saml/providers", data);
    return response.data;
  },

  update: async (
    id: string,
    data: UpdateSAMLProviderRequest,
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.put(`/api/v1/admin/saml/providers/${id}`, data);
    return response.data;
  },

  delete: async (
    id: string,
  ): Promise<{ success: boolean; message: string }> => {
    const response = await api.delete(`/api/v1/admin/saml/providers/${id}`);
    return response.data;
  },

  validateMetadata: async (
    metadataUrl?: string,
    metadataXml?: string,
  ): Promise<ValidateMetadataResponse> => {
    const response = await api.post<ValidateMetadataResponse>(
      "/api/v1/admin/saml/validate-metadata",
      {
        metadata_url: metadataUrl,
        metadata_xml: metadataXml,
      },
    );
    return response.data;
  },

  uploadMetadata: async (
    file: File,
  ): Promise<ValidateMetadataResponse & { metadata?: string }> => {
    const formData = new FormData();
    formData.append("metadata", file);
    const response = await api.post(
      "/api/v1/admin/saml/upload-metadata",
      formData,
      {
        headers: {
          "Content-Type": "multipart/form-data",
        },
      },
    );
    return response.data;
  },
};
