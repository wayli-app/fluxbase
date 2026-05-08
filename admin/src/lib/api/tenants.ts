import { api } from "./client";

export interface Tenant {
  id: string;
  slug: string;
  name: string;
  is_default: boolean;
  metadata?: Record<string, unknown>;
  user_count: number;
  admin_count: number;
  created_at: string;
  updated_at?: string;
  deleted_at?: string;
}

export interface TenantWithRole extends Tenant {
  my_role?: "tenant_admin" | "tenant_member";
}

export interface TenantMembership {
  id: string;
  tenant_id: string;
  user_id: string;
  role: "tenant_admin" | "tenant_member";
  created_at: string;
  updated_at?: string;
  email?: string;
  user_role?: string;
}

export interface CreateTenantRequest {
  slug: string;
  name: string;
  metadata?: Record<string, unknown>;
  db_mode?: "auto" | "existing";
  db_name?: string;
  auto_generate_keys?: boolean;
  admin_email?: string;
  admin_user_id?: string;
  send_keys_to_email?: boolean;
}

export interface CreateTenantResponse {
  tenant: Tenant;
  anon_key?: string;
  service_key?: string;
  invitation_sent: boolean;
  invitation_email?: string;
}

export interface UpdateTenantRequest {
  name?: string;
  metadata?: Record<string, unknown>;
}

export interface AddMemberRequest {
  user_id: string;
  role: "tenant_admin" | "tenant_member";
}

export interface UpdateMemberRequest {
  role: "tenant_admin" | "tenant_member";
}

export const tenantsApi = {
  list: async (): Promise<Tenant[]> => {
    const response = await api.get<Tenant[]>("/api/v1/admin/tenants");
    return response.data || [];
  },

  listMine: async (): Promise<TenantWithRole[]> => {
    const response = await api.get<TenantWithRole[]>(
      "/api/v1/admin/tenants/mine",
    );
    return response.data || [];
  },

  get: async (id: string): Promise<Tenant> => {
    const response = await api.get<Tenant>(`/api/v1/admin/tenants/${id}`);
    return response.data;
  },

  create: async (data: CreateTenantRequest): Promise<CreateTenantResponse> => {
    const response = await api.post<CreateTenantResponse>(
      "/api/v1/admin/tenants",
      data,
    );
    return response.data;
  },

  update: async (id: string, data: UpdateTenantRequest): Promise<Tenant> => {
    const response = await api.patch<Tenant>(
      `/api/v1/admin/tenants/${id}`,
      data,
    );
    return response.data;
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/v1/admin/tenants/${id}`);
  },

  repair: async (id: string): Promise<void> => {
    await api.post(`/api/v1/admin/tenants/${id}/repair`);
  },

  listMembers: async (tenantId: string): Promise<TenantMembership[]> => {
    const response = await api.get<TenantMembership[]>(
      `/api/v1/admin/tenants/${tenantId}/members`,
    );
    return response.data || [];
  },

  addMember: async (
    tenantId: string,
    data: AddMemberRequest,
  ): Promise<TenantMembership> => {
    const response = await api.post<TenantMembership>(
      `/api/v1/admin/tenants/${tenantId}/members`,
      data,
    );
    return response.data;
  },

  updateMemberRole: async (
    tenantId: string,
    userId: string,
    data: UpdateMemberRequest,
  ): Promise<void> => {
    await api.patch(
      `/api/v1/admin/tenants/${tenantId}/members/${userId}`,
      data,
    );
  },

  removeMember: async (tenantId: string, userId: string): Promise<void> => {
    await api.delete(`/api/v1/admin/tenants/${tenantId}/members/${userId}`);
  },
};
