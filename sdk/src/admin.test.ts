import { describe, it, expect, beforeEach, vi } from "vitest";
import { FluxbaseAdmin } from "./admin";
import { FluxbaseFetch } from "./fetch";
import { FluxbaseSettings } from "./settings";
import { DDLManager } from "./ddl";
import { FluxbaseOAuth } from "./oauth";
import { ImpersonationManager } from "./impersonation";
import type {
  AdminAuthResponse,
  AdminSetupStatusResponse,
  AdminMeResponse,
  ListUsersResponse,
  InviteUserResponse,
  EnrichedUser,
  DeleteUserResponse,
  ResetUserPasswordResponse,
} from "./types";

// Mock FluxbaseFetch
vi.mock("./fetch");

describe("FluxbaseAdmin", () => {
  let admin: FluxbaseAdmin;
  let mockFetch: any;

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
      setAuthToken: vi.fn(),
    };

    admin = new FluxbaseAdmin(mockFetch as unknown as FluxbaseFetch);
  });

  describe("Manager Initialization", () => {
    it("should initialize all managers", () => {
      expect(admin.settings).toBeInstanceOf(FluxbaseSettings);
      expect(admin.ddl).toBeInstanceOf(DDLManager);
      expect(admin.oauth).toBeInstanceOf(FluxbaseOAuth);
      expect(admin.impersonation).toBeInstanceOf(ImpersonationManager);
    });
  });

  describe("Token Management", () => {
    it("should set admin token", () => {
      const token = "admin-token-123";
      admin.setToken(token);

      expect(admin.getToken()).toBe(token);
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith(token);
    });

    it("should clear admin token", () => {
      admin.setToken("admin-token-123");
      admin.clearToken();

      expect(admin.getToken()).toBeNull();
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith(null);
    });
  });

  describe("Admin Authentication", () => {
    describe("getSetupStatus()", () => {
      it("should check setup status", async () => {
        const response: AdminSetupStatusResponse = {
          needs_setup: true,
          has_admin: false,
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const result = await admin.getSetupStatus();

        expect(mockFetch.get).toHaveBeenCalledWith(
          "/api/v1/admin/setup/status",
        );
        expect(result.needs_setup).toBe(true);
        expect(result.has_admin).toBe(false);
      });
    });

    describe("setup()", () => {
      it("should perform initial setup", async () => {
        const response: AdminAuthResponse = {
          user: {
            id: "admin-id",
            email: "admin@example.com",
            name: "Admin User",
            role: "dashboard_admin",
            email_verified: true,
            created_at: "2024-01-26T10:00:00Z",
            updated_at: "2024-01-26T10:00:00Z",
          },
          access_token: "access-token",
          refresh_token: "refresh-token",
          expires_in: 900,
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const result = await admin.setup({
          email: "admin@example.com",
          password: "SecurePassword123!",
          name: "Admin User",
          setup_token: "test-setup-token",
        });

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/admin/setup", {
          email: "admin@example.com",
          password: "SecurePassword123!",
          name: "Admin User",
          setup_token: "test-setup-token",
        });

        expect(result.user.email).toBe("admin@example.com");
        expect(result.access_token).toBe("access-token");
        expect(admin.getToken()).toBe("access-token");
      });
    });

    describe("login()", () => {
      it("should login admin user", async () => {
        const response: AdminAuthResponse = {
          user: {
            id: "admin-id",
            email: "admin@example.com",
            name: "Admin User",
            role: "dashboard_admin",
            email_verified: true,
            created_at: "2024-01-26T10:00:00Z",
            updated_at: "2024-01-26T10:00:00Z",
          },
          access_token: "access-token",
          refresh_token: "refresh-token",
          expires_in: 900,
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const result = await admin.login({
          email: "admin@example.com",
          password: "password123",
        });

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/admin/login", {
          email: "admin@example.com",
          password: "password123",
        });

        expect(result.user.email).toBe("admin@example.com");
        expect(admin.getToken()).toBe("access-token");
      });
    });

    describe("refreshToken()", () => {
      it("should refresh admin token", async () => {
        const response = {
          access_token: "new-access-token",
          refresh_token: "new-refresh-token",
          expires_in: 900,
          user: {
            id: "admin-id",
            email: "admin@example.com",
            name: "Admin User",
            role: "dashboard_admin",
            email_verified: true,
            created_at: "2024-01-26T10:00:00Z",
            updated_at: "2024-01-26T10:00:00Z",
          },
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const result = await admin.refreshToken({
          refresh_token: "old-refresh-token",
        });

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/admin/refresh", {
          refresh_token: "old-refresh-token",
        });

        expect(result.access_token).toBe("new-access-token");
        expect(admin.getToken()).toBe("new-access-token");
      });
    });

    describe("logout()", () => {
      it("should logout admin user", async () => {
        admin.setToken("admin-token");
        vi.mocked(mockFetch.post).mockResolvedValue({
          message: "Logged out successfully",
        });

        await admin.logout();

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/admin/logout", {});
        expect(admin.getToken()).toBeNull();
      });
    });

    describe("me()", () => {
      it("should get current admin user", async () => {
        const response: AdminMeResponse = {
          user: {
            id: "admin-id",
            email: "admin@example.com",
            role: "admin",
          },
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const result = await admin.me();

        expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/admin/me");
        expect(result.user.email).toBe("admin@example.com");
        expect(result.user.role).toBe("admin");
      });
    });
  });

  describe("User Management", () => {
    describe("listUsers()", () => {
      it("should list all users", async () => {
        const response: ListUsersResponse = {
          users: [
            {
              id: "user-1",
              email: "user1@example.com",
              role: "user",
              created_at: "2024-01-26T10:00:00Z",
              email_verified: true,
            },
            {
              id: "user-2",
              email: "user2@example.com",
              role: "user",
              created_at: "2024-01-26T11:00:00Z",
              email_verified: false,
            },
          ],
          total: 2,
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const result = await admin.listUsers();

        expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/admin/users");
        expect(result.users).toHaveLength(2);
        expect(result.total).toBe(2);
      });

      it("should list users with filters", async () => {
        const response: ListUsersResponse = {
          users: [
            {
              id: "user-1",
              email: "john@example.com",
              role: "user",
              created_at: "2024-01-26T10:00:00Z",
            },
          ],
          total: 1,
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const result = await admin.listUsers({
          exclude_admins: true,
          search: "john",
          limit: 10,
          type: "app",
        });

        expect(mockFetch.get).toHaveBeenCalledWith(
          "/api/v1/admin/users?exclude_admins=true&search=john&limit=10&type=app",
        );
        expect(result.users).toHaveLength(1);
      });

      it("should handle empty filters", async () => {
        const response: ListUsersResponse = { users: [], total: 0 };
        vi.mocked(mockFetch.get).mockResolvedValue(response);

        await admin.listUsers({});

        expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/admin/users");
      });
    });

    describe("inviteUser()", () => {
      it("should invite a new user", async () => {
        const response: InviteUserResponse = {
          user: {
            id: "new-user-id",
            email: "newuser@example.com",
            role: "user",
            created_at: "2024-01-26T12:00:00Z",
            email_verified: false,
          },
          invitation_link: "https://app.example.com/invite?token=abc123",
          message: "User invited successfully",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const result = await admin.inviteUser({
          email: "newuser@example.com",
          role: "user",
          send_email: true,
        });

        expect(mockFetch.post).toHaveBeenCalledWith(
          "/api/v1/admin/users/invite?type=app",
          {
            email: "newuser@example.com",
            role: "user",
            send_email: true,
          },
        );

        expect(result.user.email).toBe("newuser@example.com");
        expect(result.invitation_link).toBeDefined();
      });

      it("should invite dashboard user", async () => {
        const response: InviteUserResponse = {
          user: {
            id: "new-user-id",
            email: "dashboarduser@example.com",
            role: "dashboard_user",
            created_at: "2024-01-26T12:00:00Z",
          },
          message: "User invited successfully",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        await admin.inviteUser(
          {
            email: "dashboarduser@example.com",
            role: "dashboard_user",
          },
          "dashboard",
        );

        expect(mockFetch.post).toHaveBeenCalledWith(
          "/api/v1/admin/users/invite?type=dashboard",
          {
            email: "dashboarduser@example.com",
            role: "dashboard_user",
          },
        );
      });
    });

    describe("deleteUser()", () => {
      it("should delete a user", async () => {
        const response: DeleteUserResponse = {
          message: "User deleted successfully",
        };

        vi.mocked(mockFetch.delete).mockResolvedValue(response);

        const result = await admin.deleteUser("user-123");

        expect(mockFetch.delete).toHaveBeenCalledWith(
          "/api/v1/admin/users/user-123?type=app",
        );
        expect(result.message).toBe("User deleted successfully");
      });

      it("should delete dashboard user", async () => {
        const response: DeleteUserResponse = {
          message: "User deleted successfully",
        };

        vi.mocked(mockFetch.delete).mockResolvedValue(response);

        await admin.deleteUser("user-123", "dashboard");

        expect(mockFetch.delete).toHaveBeenCalledWith(
          "/api/v1/admin/users/user-123?type=dashboard",
        );
      });
    });

    describe("updateUserRole()", () => {
      it("should update user role", async () => {
        const user: EnrichedUser = {
          id: "user-123",
          email: "user@example.com",
          role: "admin",
          created_at: "2024-01-26T10:00:00Z",
          updated_at: "2024-01-26T12:00:00Z",
        };

        vi.mocked(mockFetch.patch).mockResolvedValue(user);

        const result = await admin.updateUserRole("user-123", "admin");

        expect(mockFetch.patch).toHaveBeenCalledWith(
          "/api/v1/admin/users/user-123/role?type=app",
          {
            role: "admin",
          },
        );
        expect(result.role).toBe("admin");
      });

      it("should update dashboard user role", async () => {
        const user: EnrichedUser = {
          id: "user-123",
          email: "user@example.com",
          role: "dashboard_admin",
          created_at: "2024-01-26T10:00:00Z",
        };

        vi.mocked(mockFetch.patch).mockResolvedValue(user);

        await admin.updateUserRole("user-123", "dashboard_admin", "dashboard");

        expect(mockFetch.patch).toHaveBeenCalledWith(
          "/api/v1/admin/users/user-123/role?type=dashboard",
          {
            role: "dashboard_admin",
          },
        );
      });
    });

    describe("resetUserPassword()", () => {
      it("should reset user password", async () => {
        const response: ResetUserPasswordResponse = {
          message: "Password reset email sent",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const result = await admin.resetUserPassword("user-123");

        expect(mockFetch.post).toHaveBeenCalledWith(
          "/api/v1/admin/users/user-123/reset-password?type=app",
          {},
        );
        expect(result.message).toBe("Password reset email sent");
      });

      it("should reset dashboard user password", async () => {
        const response: ResetUserPasswordResponse = {
          message: "Password reset successfully",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        await admin.resetUserPassword("user-123", "dashboard");

        expect(mockFetch.post).toHaveBeenCalledWith(
          "/api/v1/admin/users/user-123/reset-password?type=dashboard",
          {},
        );
      });
    });
  });

  describe("Integration Scenarios", () => {
    it("should handle complete admin workflow", async () => {
      // 1. Check setup status
      vi.mocked(mockFetch.get).mockResolvedValueOnce({
        needs_setup: true,
        has_admin: false,
      });

      const status = await admin.getSetupStatus();
      expect(status.needs_setup).toBe(true);

      // 2. Perform setup
      vi.mocked(mockFetch.post).mockResolvedValueOnce({
        user: {
          id: "admin-id",
          email: "admin@example.com",
          name: "Admin",
          role: "dashboard_admin",
          email_verified: true,
          created_at: "2024-01-26T10:00:00Z",
          updated_at: "2024-01-26T10:00:00Z",
        },
        access_token: "token-123",
        refresh_token: "refresh-123",
        expires_in: 900,
      });

      await admin.setup({
        email: "admin@example.com",
        password: "SecurePassword123!",
        name: "Admin",
      });

      expect(admin.getToken()).toBe("token-123");

      // 3. Get current user
      vi.mocked(mockFetch.get).mockResolvedValueOnce({
        user: {
          id: "admin-id",
          email: "admin@example.com",
          role: "admin",
        },
      });

      const me = await admin.me();
      expect(me.user.email).toBe("admin@example.com");

      // 4. List users
      vi.mocked(mockFetch.get).mockResolvedValueOnce({
        users: [
          {
            id: "user-1",
            email: "user1@example.com",
            role: "user",
            created_at: "2024-01-26T10:00:00Z",
          },
        ],
        total: 1,
      });

      const users = await admin.listUsers();
      expect(users.total).toBe(1);

      // 5. Logout
      vi.mocked(mockFetch.post).mockResolvedValueOnce({
        message: "Logged out successfully",
      });

      await admin.logout();
      expect(admin.getToken()).toBeNull();
    });
  });
});
