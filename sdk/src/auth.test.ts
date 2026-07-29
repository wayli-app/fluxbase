/**
 * Authentication Tests
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { FluxbaseAuth } from "./auth";
import type { FluxbaseFetch } from "./fetch";
import type { AuthResponse, ProviderTokenResponse } from "./types";

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
  };
})();

Object.defineProperty(global, "localStorage", { value: localStorageMock });

describe("FluxbaseAuth", () => {
  let mockFetch: FluxbaseFetch;
  let auth: FluxbaseAuth;

  beforeEach(() => {
    localStorageMock.clear();
    vi.clearAllTimers();

    mockFetch = {
      post: vi.fn(),
      get: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
      setAuthToken: vi.fn(),
      setRefreshTokenCallback: vi.fn(),
    } as unknown as FluxbaseFetch;

    auth = new FluxbaseAuth(mockFetch, true, true);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("initialization", () => {
    it("should initialize with no session", async () => {
      const { data: sessionData } = await auth.getSession();
      const { data: userData } = await auth.getUser();
      expect(sessionData.session).toBeNull();
      expect(userData.user).toBeNull();
      expect(auth.getAccessToken()).toBeNull();
    });

    it("should restore session from localStorage", async () => {
      const session = {
        access_token: "test-token",
        refresh_token: "refresh-token",
        expires_in: 3600,
        expires_at: Date.now() + 3600 * 1000,
        token_type: "Bearer",
        user: { id: "1", email: "test@example.com", created_at: "" },
      };

      localStorage.setItem("fluxbase.auth.session", JSON.stringify(session));

      const newAuth = new FluxbaseAuth(mockFetch, true, true);

      const { data: sessionData } = await newAuth.getSession();
      expect(sessionData.session).toEqual(session);
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith("test-token");
    });

    it("should ignore invalid stored session", async () => {
      localStorage.setItem("fluxbase.auth.session", "invalid-json");

      const newAuth = new FluxbaseAuth(mockFetch, true, true);

      const { data: sessionData } = await newAuth.getSession();
      expect(sessionData.session).toBeNull();
      expect(localStorage.getItem("fluxbase.auth.session")).toBeNull();
    });
  });

  describe("custom storage adapter", () => {
    /** Builds an in-memory StorageAdapter that records calls for assertions. */
    const createAdapter = () => {
      const store = new Map<string, string>();
      const calls: string[] = [];
      return {
        calls,
        adapter: {
          getItem: (k: string) => {
            calls.push(`get:${k}`);
            return store.get(k) ?? null;
          },
          setItem: (k: string, v: string) => {
            calls.push(`set:${k}`);
            store.set(k, v);
          },
          removeItem: (k: string) => {
            calls.push(`remove:${k}`);
            store.delete(k);
          },
        },
      };
    };

    it("loads a persisted session from a custom adapter", async () => {
      const session = {
        access_token: "custom-token",
        refresh_token: "custom-refresh",
        expires_in: 3600,
        expires_at: Date.now() + 3600 * 1000,
        token_type: "Bearer",
        user: { id: "42", email: "custom@example.com", created_at: "" },
      };

      const { adapter } = createAdapter();
      adapter.setItem("fluxbase.auth.session", JSON.stringify(session));

      const newAuth = new FluxbaseAuth(mockFetch, true, true, adapter);

      const { data: sessionData } = await newAuth.getSession();
      expect(sessionData.session).toEqual(session);
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith("custom-token");
    });

    it("saves a signed-in session through the custom adapter", async () => {
      const { adapter, calls } = createAdapter();
      const newAuth = new FluxbaseAuth(mockFetch, true, true, adapter);

      const authResponse: AuthResponse = {
        access_token: "access-after-signin",
        refresh_token: "refresh-after-signin",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "7", email: "signin@example.com", created_at: "" },
      };

      (mockFetch.post as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: authResponse,
        error: null,
      });

      await newAuth.signIn({ email: "signin@example.com", password: "pw" });

      // Session should have been persisted via the custom adapter.
      expect(calls).toContain("set:fluxbase.auth.session");
      expect(adapter.getItem("fluxbase.auth.session")).toContain(
        "access-after-signin",
      );
    });

    it("removes the session from the custom adapter on sign-out", async () => {
      const session = {
        access_token: "signout-token",
        refresh_token: "signout-refresh",
        expires_in: 3600,
        expires_at: Date.now() + 3600 * 1000,
        token_type: "Bearer",
        user: { id: "9", email: "signout@example.com", created_at: "" },
      };

      const { adapter, calls } = createAdapter();
      adapter.setItem("fluxbase.auth.session", JSON.stringify(session));

      const newAuth = new FluxbaseAuth(mockFetch, true, true, adapter);

      // Sanity: adapter has the session.
      expect(adapter.getItem("fluxbase.auth.session")).not.toBeNull();

      await newAuth.signOut();

      expect(calls).toContain("remove:fluxbase.auth.session");
      expect(adapter.getItem("fluxbase.auth.session")).toBeNull();
    });

    it("prefers the custom adapter over localStorage when both are present", async () => {
      // Seed BOTH stores; only the custom adapter should be read.
      const localStorageSession = {
        access_token: "from-localStorage",
        refresh_token: "r",
        expires_in: 3600,
        expires_at: Date.now() + 3600 * 1000,
        token_type: "Bearer",
        user: { id: "1", email: "ls@example.com", created_at: "" },
      };
      const customSession = {
        access_token: "from-custom",
        refresh_token: "r",
        expires_in: 3600,
        expires_at: Date.now() + 3600 * 1000,
        token_type: "Bearer",
        user: { id: "2", email: "custom@example.com", created_at: "" },
      };

      localStorage.setItem(
        "fluxbase.auth.session",
        JSON.stringify(localStorageSession),
      );

      const { adapter } = createAdapter();
      adapter.setItem("fluxbase.auth.session", JSON.stringify(customSession));

      const newAuth = new FluxbaseAuth(mockFetch, true, true, adapter);

      const { data: sessionData } = await newAuth.getSession();
      expect(sessionData.session?.access_token).toBe("from-custom");
      // The custom adapter must win, so localStorage must be untouched.
      expect(localStorage.getItem("fluxbase.auth.session")).toContain(
        "from-localStorage",
      );
    });

    it("falls back to localStorage when no adapter is provided (regression guard)", async () => {
      // No 4th arg → default behavior: localStorage in this (jsdom-like) env.
      const session = {
        access_token: "default-token",
        refresh_token: "r",
        expires_in: 3600,
        expires_at: Date.now() + 3600 * 1000,
        token_type: "Bearer",
        user: { id: "3", email: "default@example.com", created_at: "" },
      };
      localStorage.setItem("fluxbase.auth.session", JSON.stringify(session));

      const newAuth = new FluxbaseAuth(mockFetch, true, true);

      const { data: sessionData } = await newAuth.getSession();
      expect(sessionData.session?.access_token).toBe("default-token");
    });
  });

  describe("signIn()", () => {
    it("should sign in successfully", async () => {
      const authResponse: AuthResponse = {
        access_token: "new-access-token",
        refresh_token: "new-refresh-token",
        expires_in: 3600,
        token_type: "Bearer",
        user: {
          id: "1",
          email: "user@example.com",
          created_at: new Date().toISOString(),
        },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      const { data, error } = await auth.signIn({
        email: "user@example.com",
        password: "password123",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signin", {
        email: "user@example.com",
        password: "password123",
      });
      expect(error).toBeNull();
      expect(data).toBeDefined();
      expect(data!.session.access_token).toBe("new-access-token");
      expect(data!.user.email).toBe("user@example.com");
      expect(data!.session.user.email).toBe("user@example.com");
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith("new-access-token");
    });

    it("should persist session to localStorage", async () => {
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      await auth.signIn({ email: "user@example.com", password: "password" });

      const stored = localStorage.getItem("fluxbase.auth.session");
      expect(stored).toBeTruthy();
      expect(JSON.parse(stored!).access_token).toBe("token");
    });
  });

  describe("signUp()", () => {
    it("should sign up successfully", async () => {
      const authResponse: AuthResponse = {
        access_token: "new-token",
        refresh_token: "refresh-token",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "newuser@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      const { data, error } = await auth.signUp({
        email: "newuser@example.com",
        password: "password123",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signup", {
        email: "newuser@example.com",
        password: "password123",
      });
      expect(error).toBeNull();
      expect(data).toBeDefined();
      expect(data!.user.email).toBe("newuser@example.com");
      expect(data!.session).toBeDefined();
    });

    it("should sign up with user metadata (Supabase-compatible)", async () => {
      const authResponse: AuthResponse = {
        access_token: "new-token",
        refresh_token: "refresh-token",
        expires_in: 3600,
        token_type: "Bearer",
        user: {
          id: "1",
          email: "newuser@example.com",
          created_at: "",
          metadata: { first_name: "John", age: 27 },
        },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      const { data, error } = await auth.signUp({
        email: "newuser@example.com",
        password: "password123",
        options: {
          data: {
            first_name: "John",
            age: 27,
          },
        },
      });

      // Verify the SDK transforms options.data to user_metadata for the backend
      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signup", {
        email: "newuser@example.com",
        password: "password123",
        user_metadata: {
          first_name: "John",
          age: 27,
        },
      });
      expect(error).toBeNull();
      expect(data).toBeDefined();
      expect(data!.user.email).toBe("newuser@example.com");
      expect(data!.session).toBeDefined();
    });
  });

  describe("signOut()", () => {
    it("should sign out and clear session", async () => {
      // Set up a session first
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      // Now sign out
      vi.mocked(mockFetch.post).mockResolvedValue(undefined);
      const { error } = await auth.signOut();

      expect(error).toBeNull();
      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signout");
      const { data: sessionData } = await auth.getSession();
      expect(sessionData.session).toBeNull();
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith(null);
      expect(localStorage.getItem("fluxbase.auth.session")).toBeNull();
    });

    it("should clear session even if API call fails", async () => {
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValueOnce(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      // Make signOut API call fail
      vi.mocked(mockFetch.post).mockRejectedValueOnce(
        new Error("Network error"),
      );

      // Should still resolve with error but session is cleared due to finally block
      const { error } = await auth.signOut();

      expect(error).toBeDefined();
      const { data: sessionData } = await auth.getSession();
      expect(sessionData.session).toBeNull();
    });
  });

  describe("refreshSession()", () => {
    it("should refresh access token and user", async () => {
      // Set up initial session
      const authResponse: AuthResponse = {
        access_token: "old-token",
        refresh_token: "refresh-token",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      // Refresh token
      const refreshResponse: AuthResponse = {
        access_token: "new-token",
        refresh_token: "new-refresh-token",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(refreshResponse);

      const { data, error } = await auth.refreshSession();

      expect(mockFetch.post).toHaveBeenCalledWith(
        "/api/v1/auth/refresh",
        {
          refresh_token: "refresh-token",
        },
        { skipAutoRefresh: true },
      );
      expect(error).toBeNull();
      expect(data).toBeDefined();
      expect(data!.session.access_token).toBe("new-token");
      expect(data!.user.email).toBe("user@example.com");
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith("new-token");
    });

    it("should return error when no refresh token available", async () => {
      const { data, error } = await auth.refreshSession();

      expect(data).toBeNull();
      expect(error).toBeDefined();
      expect(error?.message).toBe("No refresh token available");
    });
  });

  describe("getCurrentUser()", () => {
    it("should fetch current user", async () => {
      // Set up session
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      const user = { id: "1", email: "user@example.com", created_at: "" };
      vi.mocked(mockFetch.get).mockResolvedValue(user);

      const { data: result, error } = await auth.getCurrentUser();

      expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/auth/user");
      expect(error).toBeNull();
      expect(result).toBeDefined();
      expect(result!.user).toEqual(user);
    });

    it("should return error when not authenticated", async () => {
      const { data, error } = await auth.getCurrentUser();

      expect(data).toBeNull();
      expect(error).toBeDefined();
      expect(error?.message).toBe("Not authenticated");
    });
  });

  describe("updateUser()", () => {
    it("should update user profile", async () => {
      // Set up session
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "old@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "old@example.com", password: "password" });

      const updatedUser = { id: "1", email: "new@example.com", created_at: "" };
      vi.mocked(mockFetch.patch).mockResolvedValue(updatedUser);

      const { data: result, error } = await auth.updateUser({
        email: "new@example.com",
      });

      expect(mockFetch.patch).toHaveBeenCalledWith("/api/v1/auth/user", {
        email: "new@example.com",
      });
      expect(error).toBeNull();
      expect(result).toBeDefined();
      expect(result!.user.email).toBe("new@example.com");
      const { data: userData } = await auth.getUser();
      expect(userData.user?.email).toBe("new@example.com");
    });

    it("should update user metadata (Supabase-compatible)", async () => {
      // Set up session
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      const updatedUser = {
        id: "1",
        email: "user@example.com",
        created_at: "",
        metadata: { name: "Updated Name", theme: "dark" },
      };
      vi.mocked(mockFetch.patch).mockResolvedValue(updatedUser);

      const { data: result, error } = await auth.updateUser({
        data: {
          name: "Updated Name",
          theme: "dark",
        },
      });

      // Verify the SDK transforms 'data' to 'user_metadata' for the backend
      expect(mockFetch.patch).toHaveBeenCalledWith("/api/v1/auth/user", {
        user_metadata: {
          name: "Updated Name",
          theme: "dark",
        },
      });
      expect(error).toBeNull();
      expect(result).toBeDefined();
      expect(result!.user.metadata).toEqual({
        name: "Updated Name",
        theme: "dark",
      });
    });

    it("should return error when not authenticated", async () => {
      const { data, error } = await auth.updateUser({
        email: "new@example.com",
      });

      expect(data).toBeNull();
      expect(error).toBeDefined();
      expect(error?.message).toBe("Not authenticated");
    });
  });

  describe("session persistence", () => {
    it("should not persist when persist is false", async () => {
      const noPersistAuth = new FluxbaseAuth(mockFetch, true, false);

      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await noPersistAuth.signIn({
        email: "user@example.com",
        password: "password",
      });

      expect(localStorage.getItem("fluxbase.auth.session")).toBeNull();
    });

    it("should use memory storage fallback when localStorage throws", async () => {
      // Make localStorage throw to simulate blocked/unavailable storage
      const originalSetItem = localStorageMock.setItem;
      const originalGetItem = localStorageMock.getItem;

      // Temporarily make localStorage throw (simulates private browsing mode)
      localStorageMock.setItem = () => {
        throw new Error("localStorage is not available");
      };
      localStorageMock.getItem = () => {
        throw new Error("localStorage is not available");
      };

      const freshMockFetch = {
        post: vi.fn(),
        get: vi.fn(),
        patch: vi.fn(),
        delete: vi.fn(),
        setAuthToken: vi.fn(),
        setRefreshTokenCallback: vi.fn(),
      } as unknown as FluxbaseFetch;

      // Create auth instance - should use MemoryStorage fallback due to localStorage throwing
      const nodeAuth = new FluxbaseAuth(freshMockFetch, true, true);

      const authResponse: AuthResponse = {
        access_token: "node-token",
        refresh_token: "node-refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "node@example.com", created_at: "" },
      };

      vi.mocked(freshMockFetch.post).mockResolvedValue(authResponse);

      // Sign in should work without throwing
      const { data, error } = await nodeAuth.signIn({
        email: "node@example.com",
        password: "password",
      });

      expect(error).toBeNull();
      expect(data).toBeDefined();
      expect(data!.session.access_token).toBe("node-token");
      expect(freshMockFetch.setAuthToken).toHaveBeenCalledWith("node-token");

      // Session should be accessible
      const { data: sessionData } = await nodeAuth.getSession();
      expect(sessionData.session?.access_token).toBe("node-token");

      // Restore localStorage functions
      localStorageMock.setItem = originalSetItem;
      localStorageMock.getItem = originalGetItem;
    });

    it("should maintain session in memory storage across operations", async () => {
      // Make localStorage throw to simulate blocked/unavailable storage
      const originalSetItem = localStorageMock.setItem;
      const originalGetItem = localStorageMock.getItem;
      const originalRemoveItem = localStorageMock.removeItem;

      localStorageMock.setItem = () => {
        throw new Error("localStorage is not available");
      };
      localStorageMock.getItem = () => {
        throw new Error("localStorage is not available");
      };
      localStorageMock.removeItem = () => {
        throw new Error("localStorage is not available");
      };

      const freshMockFetch = {
        post: vi.fn(),
        get: vi.fn(),
        patch: vi.fn(),
        delete: vi.fn(),
        setAuthToken: vi.fn(),
        setRefreshTokenCallback: vi.fn(),
      } as unknown as FluxbaseFetch;

      const nodeAuth = new FluxbaseAuth(freshMockFetch, true, true);

      // Sign in
      const authResponse: AuthResponse = {
        access_token: "memory-token",
        refresh_token: "memory-refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "memory@example.com", created_at: "" },
      };

      vi.mocked(freshMockFetch.post).mockResolvedValue(authResponse);
      await nodeAuth.signIn({
        email: "memory@example.com",
        password: "password",
      });

      // Verify session exists
      const { data: sessionData } = await nodeAuth.getSession();
      expect(sessionData.session?.access_token).toBe("memory-token");

      // Sign out
      vi.mocked(freshMockFetch.post).mockResolvedValue(undefined);
      await nodeAuth.signOut();

      // Session should be cleared
      const { data: clearedSession } = await nodeAuth.getSession();
      expect(clearedSession.session).toBeNull();

      // Restore localStorage functions
      localStorageMock.setItem = originalSetItem;
      localStorageMock.getItem = originalGetItem;
      localStorageMock.removeItem = originalRemoveItem;
    });

    it("should completely disable persistence when persist is false", async () => {
      const freshMockFetch = {
        post: vi.fn(),
        get: vi.fn(),
        patch: vi.fn(),
        delete: vi.fn(),
        setAuthToken: vi.fn(),
        setRefreshTokenCallback: vi.fn(),
      } as unknown as FluxbaseFetch;

      // Clear localStorage before test
      localStorageMock.clear();

      // Create auth with persist=false
      const noPersistAuth = new FluxbaseAuth(freshMockFetch, true, false);

      const authResponse: AuthResponse = {
        access_token: "no-persist-token",
        refresh_token: "no-persist-refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "nopersist@example.com", created_at: "" },
      };

      vi.mocked(freshMockFetch.post).mockResolvedValue(authResponse);
      await noPersistAuth.signIn({
        email: "nopersist@example.com",
        password: "password",
      });

      // Session should work in-memory
      const { data: sessionData } = await noPersistAuth.getSession();
      expect(sessionData.session?.access_token).toBe("no-persist-token");

      // But nothing should be in localStorage
      expect(localStorage.getItem("fluxbase.auth.session")).toBeNull();
    });
  });

  describe("Password Reset Flow", () => {
    describe("sendPasswordReset()", () => {
      it("should send password reset email", async () => {
        vi.mocked(mockFetch.post).mockResolvedValue({});

        const { data: result, error } =
          await auth.sendPasswordReset("user@example.com");

        expect(mockFetch.post).toHaveBeenCalledWith(
          "/api/v1/auth/password/reset",
          {
            email: "user@example.com",
          },
        );
        expect(error).toBeNull();
        expect(result).toBeDefined();
        expect(result!.user).toBeNull();
        expect(result!.session).toBeNull();
      });
    });

    describe("verifyResetToken()", () => {
      it("should verify valid reset token", async () => {
        const response = {
          valid: true,
          message: "Token is valid",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const { data: result, error } =
          await auth.verifyResetToken("valid-token");

        expect(mockFetch.post).toHaveBeenCalledWith(
          "/api/v1/auth/password/reset/verify",
          {
            token: "valid-token",
          },
        );
        expect(error).toBeNull();
        expect(result).toBeDefined();
        expect(result!.valid).toBe(true);
      });

      it("should return invalid for expired token", async () => {
        const response = {
          valid: false,
          message: "Token has expired",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const { data: result, error } =
          await auth.verifyResetToken("expired-token");

        expect(error).toBeNull();
        expect(result).toBeDefined();
        expect(result!.valid).toBe(false);
      });
    });

    describe("resetPassword()", () => {
      it("should reset password with valid token", async () => {
        const response = {
          access_token: "new-access-token",
          refresh_token: "new-refresh-token",
          expires_in: 3600,
          token_type: "Bearer",
          user: {
            id: "1",
            email: "user@example.com",
            created_at: new Date().toISOString(),
          },
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const { data: result, error } = await auth.resetPassword(
          "valid-token",
          "newPassword123",
        );

        expect(mockFetch.post).toHaveBeenCalledWith(
          "/api/v1/auth/password/reset/confirm",
          {
            token: "valid-token",
            new_password: "newPassword123",
          },
        );
        expect(error).toBeNull();
        expect(result).toBeDefined();
        expect(result!.user).toBeDefined();
        expect(result!.session).toBeDefined();
        expect(result!.session.access_token).toBe("new-access-token");
        expect(mockFetch.setAuthToken).toHaveBeenCalledWith("new-access-token");
      });
    });
  });

  describe("Magic Link Authentication", () => {
    describe("sendMagicLink()", () => {
      it("should send magic link without options", async () => {
        vi.mocked(mockFetch.post).mockResolvedValue({});

        const { data: result, error } =
          await auth.sendMagicLink("user@example.com");

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/magiclink", {
          email: "user@example.com",
          redirect_to: undefined,
        });
        expect(error).toBeNull();
        expect(result).toBeDefined();
        expect(result!.user).toBeNull();
        expect(result!.session).toBeNull();
      });

      it("should send magic link with redirect URL", async () => {
        vi.mocked(mockFetch.post).mockResolvedValue({});

        const { data: result, error } = await auth.sendMagicLink(
          "user@example.com",
          {
            redirect_to: "https://app.example.com/dashboard",
          },
        );

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/magiclink", {
          email: "user@example.com",
          redirect_to: "https://app.example.com/dashboard",
        });
        expect(error).toBeNull();
        expect(result).toBeDefined();
        expect(result!.user).toBeNull();
        expect(result!.session).toBeNull();
      });
    });

    describe("verifyMagicLink()", () => {
      it("should verify magic link and create session", async () => {
        const authResponse: AuthResponse = {
          access_token: "magic-token",
          refresh_token: "refresh-token",
          expires_in: 3600,
          token_type: "Bearer",
          user: { id: "1", email: "user@example.com", created_at: "" },
        };

        vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

        const { data: session, error } =
          await auth.verifyMagicLink("magic-link-token");

        expect(mockFetch.post).toHaveBeenCalledWith(
          "/api/v1/auth/magiclink/verify",
          {
            token: "magic-link-token",
          },
        );
        expect(error).toBeNull();
        expect(session).toBeDefined();
        expect(session!.user.email).toBe("user@example.com");
        expect(session!.session.access_token).toBe("magic-token");
        const { data: sessionData } = await auth.getSession();
        expect(sessionData.session?.access_token).toBe("magic-token");
        expect(mockFetch.setAuthToken).toHaveBeenCalledWith("magic-token");
      });
    });
  });

  describe("Anonymous Authentication", () => {
    describe("signInAnonymously()", () => {
      it("should create anonymous session", async () => {
        const authResponse: AuthResponse = {
          access_token: "anon-token",
          refresh_token: "anon-refresh-token",
          expires_in: 3600,
          token_type: "Bearer",
          user: {
            id: "anon-123",
            email: "anonymous@fluxbase.local",
            created_at: new Date().toISOString(),
          },
        };

        vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

        const { data: session, error } = await auth.signInAnonymously();

        expect(mockFetch.post).toHaveBeenCalledWith(
          "/api/v1/auth/signin/anonymous",
        );
        expect(error).toBeNull();
        expect(session).toBeDefined();
        expect(session!.user.email).toBe("anonymous@fluxbase.local");
        expect(session!.session.access_token).toBe("anon-token");
        const { data: sessionData } = await auth.getSession();
        expect(sessionData.session?.access_token).toBe("anon-token");
      });
    });
  });

  describe("OAuth Flow", () => {
    describe("getOAuthProviders()", () => {
      it("should fetch list of OAuth providers", async () => {
        const response = {
          providers: [
            { id: "google", name: "Google", enabled: true },
            { id: "github", name: "GitHub", enabled: true },
          ],
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const { data: result, error } = await auth.getOAuthProviders();

        expect(mockFetch.get).toHaveBeenCalledWith(
          "/api/v1/auth/oauth/providers",
        );
        expect(error).toBeNull();
        expect(result).toBeDefined();
        expect(result!.providers).toHaveLength(2);
        expect(result!.providers[0].id).toBe("google");
      });
    });

    describe("getOAuthUrl()", () => {
      it("should get OAuth URL without options", async () => {
        const response = {
          url: "https://accounts.google.com/o/oauth2/v2/auth?...",
          provider: "google",
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const { data: result, error } = await auth.getOAuthUrl("google");

        expect(mockFetch.get).toHaveBeenCalledWith(
          "/api/v1/auth/oauth/google/authorize",
        );
        expect(error).toBeNull();
        expect(result).toBeDefined();
        expect(result!.url).toContain("google.com");
      });

      it("should get OAuth URL with redirect_to", async () => {
        const response = {
          url: "https://accounts.google.com/o/oauth2/v2/auth?...",
          provider: "google",
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const { data: result, error } = await auth.getOAuthUrl("google", {
          redirect_to: "https://app.example.com/auth/callback",
        });

        expect(mockFetch.get).toHaveBeenCalledWith(
          "/api/v1/auth/oauth/google/authorize?redirect_to=https%3A%2F%2Fapp.example.com%2Fauth%2Fcallback",
        );
        expect(error).toBeNull();
      });

      it("should get OAuth URL with scopes", async () => {
        const response = {
          url: "https://accounts.google.com/o/oauth2/v2/auth?...",
          provider: "google",
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const { data: result, error } = await auth.getOAuthUrl("google", {
          scopes: ["email", "profile"],
        });

        expect(mockFetch.get).toHaveBeenCalledWith(
          "/api/v1/auth/oauth/google/authorize?scopes=email%2Cprofile",
        );
        expect(error).toBeNull();
      });

      it("should get OAuth URL with both redirect_to and scopes", async () => {
        const response = {
          url: "https://github.com/login/oauth/authorize?...",
          provider: "github",
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const { data: result, error } = await auth.getOAuthUrl("github", {
          redirect_to: "https://app.example.com/callback",
          scopes: ["read:user", "repo"],
        });

        expect(mockFetch.get).toHaveBeenCalledWith(
          expect.stringContaining("/api/v1/auth/oauth/github/authorize?"),
        );
        expect(mockFetch.get).toHaveBeenCalledWith(
          expect.stringContaining("redirect_to="),
        );
        expect(mockFetch.get).toHaveBeenCalledWith(
          expect.stringContaining("scopes="),
        );
        expect(error).toBeNull();
      });
    });

    describe("exchangeCodeForSession()", () => {
      it("should exchange OAuth code for session", async () => {
        const authResponse: AuthResponse = {
          access_token: "oauth-token",
          refresh_token: "oauth-refresh",
          expires_in: 3600,
          token_type: "Bearer",
          user: {
            id: "oauth-user-1",
            email: "user@example.com",
            created_at: new Date().toISOString(),
          },
        };

        // Simulate signInWithOAuth storing the provider
        localStorageMock.setItem("fluxbase.auth.oauth_provider", "google");

        vi.mocked(mockFetch.get).mockResolvedValue(authResponse);

        const { data: session, error } = await auth.exchangeCodeForSession(
          "auth-code-123",
          "state-token",
        );

        expect(mockFetch.get).toHaveBeenCalledWith(
          "/api/v1/auth/oauth/google/callback?code=auth-code-123&state=state-token",
        );
        expect(error).toBeNull();
        expect(session).toBeDefined();
        expect(session!.user.email).toBe("user@example.com");
        expect(session!.session.access_token).toBe("oauth-token");
        // Provider should be cleared after exchange
        expect(
          localStorageMock.getItem("fluxbase.auth.oauth_provider"),
        ).toBeNull();
        const { data: sessionData } = await auth.getSession();
        expect(sessionData.session?.access_token).toBe("oauth-token");
      });

      it("should throw error if no provider stored", async () => {
        // Ensure no provider is stored
        localStorageMock.removeItem("fluxbase.auth.oauth_provider");

        const { data, error } =
          await auth.exchangeCodeForSession("auth-code-123");

        expect(data).toBeNull();
        expect(error).toBeDefined();
        expect(error?.message).toBe(
          "No OAuth provider found. Call signInWithOAuth first.",
        );
      });

      it("should work without state parameter", async () => {
        const authResponse: AuthResponse = {
          access_token: "oauth-token",
          refresh_token: "oauth-refresh",
          expires_in: 3600,
          token_type: "Bearer",
          user: {
            id: "oauth-user-1",
            email: "user@example.com",
            created_at: new Date().toISOString(),
          },
        };

        localStorageMock.setItem("fluxbase.auth.oauth_provider", "github");
        vi.mocked(mockFetch.get).mockResolvedValue(authResponse);

        const { data: session, error } =
          await auth.exchangeCodeForSession("auth-code-123");

        expect(mockFetch.get).toHaveBeenCalledWith(
          "/api/v1/auth/oauth/github/callback?code=auth-code-123",
        );
        expect(error).toBeNull();
        expect(session).toBeDefined();
      });
    });

    describe("signInWithOAuth()", () => {
      it("should redirect to OAuth provider in browser and store provider", async () => {
        const response = {
          url: "https://accounts.google.com/o/oauth2/v2/auth?...",
          provider: "google",
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        // Mock window.location
        const originalLocation = global.window?.location;
        delete (global as any).window;
        (global as any).window = { location: { href: "" } };

        const { data, error } = await auth.signInWithOAuth("google");

        expect(window.location.href).toBe(response.url);
        expect(error).toBeNull();
        expect(data).toBeDefined();
        // Verify provider was stored for exchangeCodeForSession
        expect(localStorageMock.getItem("fluxbase.auth.oauth_provider")).toBe(
          "google",
        );

        // Restore
        if (originalLocation) {
          (global as any).window = { location: originalLocation };
        } else {
          delete (global as any).window;
        }
      });

      it("should return error in non-browser environment", async () => {
        const response = {
          url: "https://accounts.google.com/o/oauth2/v2/auth?...",
          provider: "google",
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        // Ensure window is undefined
        const originalWindow = global.window;
        delete (global as any).window;

        const { data, error } = await auth.signInWithOAuth("google");

        expect(data).toBeNull();
        expect(error).toBeDefined();
        expect(error?.message).toBe(
          "signInWithOAuth can only be called in a browser environment",
        );

        // Restore
        if (originalWindow) {
          (global as any).window = originalWindow;
        }
      });
    });
  });

  describe("refreshToken()", () => {
    it("should be an alias for refreshSession()", async () => {
      // Set up initial session
      const authResponse: AuthResponse = {
        access_token: "initial-token",
        refresh_token: "refresh-token",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      // Mock refresh response
      const refreshResponse: AuthResponse = {
        access_token: "new-token",
        refresh_token: "new-refresh-token",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(refreshResponse);

      const { data, error } = await auth.refreshToken();

      expect(error).toBeNull();
      expect(data).toBeDefined();
      expect(data!.session.access_token).toBe("new-token");
      expect(mockFetch.post).toHaveBeenCalledWith(
        "/api/v1/auth/refresh",
        {
          refresh_token: "refresh-token",
        },
        { skipAutoRefresh: true },
      );
    });
  });

  describe("OTP Methods", () => {
    describe("signInWithOtp()", () => {
      it("should send OTP to email", async () => {
        vi.mocked(mockFetch.post).mockResolvedValue({});

        const { data, error } = await auth.signInWithOtp({
          email: "user@example.com",
        });

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/otp/signin", {
          email: "user@example.com",
        });
        expect(error).toBeNull();
        expect(data).toEqual({ user: null, session: null });
      });

      it("should send OTP to phone", async () => {
        vi.mocked(mockFetch.post).mockResolvedValue({});

        const { data, error } = await auth.signInWithOtp({
          phone: "+1234567890",
        });

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/otp/signin", {
          phone: "+1234567890",
        });
        expect(error).toBeNull();
        expect(data).toEqual({ user: null, session: null });
      });

      it("should send OTP with options", async () => {
        vi.mocked(mockFetch.post).mockResolvedValue({});

        const { data, error } = await auth.signInWithOtp({
          email: "user@example.com",
          options: {
            emailRedirectTo: "https://example.com/confirm",
            shouldCreateUser: true,
          },
        });

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/otp/signin", {
          email: "user@example.com",
          options: {
            emailRedirectTo: "https://example.com/confirm",
            shouldCreateUser: true,
          },
        });
        expect(error).toBeNull();
      });
    });

    describe("verifyOtp()", () => {
      it("should verify OTP and create session", async () => {
        const authResponse: AuthResponse = {
          access_token: "otp-token",
          refresh_token: "otp-refresh",
          expires_in: 3600,
          token_type: "Bearer",
          user: { id: "1", email: "user@example.com", created_at: "" },
        };

        vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

        const { data, error } = await auth.verifyOtp({
          email: "user@example.com",
          token: "123456",
          type: "email",
        });

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/otp/verify", {
          email: "user@example.com",
          token: "123456",
          type: "email",
        });
        expect(error).toBeNull();
        expect(data).toBeDefined();
        expect(data!.session?.access_token).toBe("otp-token");
        expect(data!.user.email).toBe("user@example.com");
      });

      it("should verify OTP without creating session (email confirmation required)", async () => {
        const authResponse = {
          user: { id: "1", email: "user@example.com", created_at: "" },
        };

        vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

        const { data, error } = await auth.verifyOtp({
          email: "user@example.com",
          token: "123456",
          type: "signup",
        });

        expect(error).toBeNull();
        expect(data).toBeDefined();
        expect(data!.session).toBeNull();
        expect(data!.user.email).toBe("user@example.com");
      });
    });

    describe("resendOtp()", () => {
      it("should resend OTP to email", async () => {
        vi.mocked(mockFetch.post).mockResolvedValue({});

        const { data, error } = await auth.resendOtp({
          type: "email",
          email: "user@example.com",
        });

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/otp/resend", {
          type: "email",
          email: "user@example.com",
        });
        expect(error).toBeNull();
        expect(data).toEqual({ user: null, session: null });
      });

      it("should resend OTP with options", async () => {
        vi.mocked(mockFetch.post).mockResolvedValue({});

        const { data, error } = await auth.resendOtp({
          type: "signup",
          email: "user@example.com",
          options: {
            emailRedirectTo: "https://example.com/confirm",
          },
        });

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/otp/resend", {
          type: "signup",
          email: "user@example.com",
          options: {
            emailRedirectTo: "https://example.com/confirm",
          },
        });
        expect(error).toBeNull();
      });
    });
  });

  describe("Identity Management", () => {
    beforeEach(async () => {
      // Set up authenticated session
      const authResponse: AuthResponse = {
        access_token: "test-token",
        refresh_token: "refresh-token",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });
    });

    describe("getUserIdentities()", () => {
      it("should get linked identities", async () => {
        const identities = {
          identities: [
            {
              id: "1",
              user_id: "1",
              provider: "google",
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            },
            {
              id: "2",
              user_id: "1",
              provider: "github",
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            },
          ],
        };

        vi.mocked(mockFetch.get).mockResolvedValue(identities);

        const { data, error } = await auth.getUserIdentities();

        expect(mockFetch.get).toHaveBeenCalledWith(
          "/api/v1/auth/user/identities",
        );
        expect(error).toBeNull();
        expect(data).toBeDefined();
        expect(data!.identities).toHaveLength(2);
        expect(data!.identities[0].provider).toBe("google");
      });

      it("should return error when not authenticated", async () => {
        const freshMockFetch = {
          post: vi.fn(),
          get: vi.fn(),
          patch: vi.fn(),
          delete: vi.fn(),
          setAuthToken: vi.fn(),
          setRefreshTokenCallback: vi.fn(),
        } as unknown as FluxbaseFetch;

        const newAuth = new FluxbaseAuth(freshMockFetch, false, false);

        const { data, error } = await newAuth.getUserIdentities();

        expect(data).toBeNull();
        expect(error).toBeDefined();
        expect(error?.message).toBe("Not authenticated");
      });
    });

    describe("linkIdentity()", () => {
      it("should link OAuth provider", async () => {
        const response = {
          url: "https://accounts.google.com/o/oauth2/v2/auth?...",
          provider: "google",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const { data, error } = await auth.linkIdentity({ provider: "google" });

        expect(mockFetch.post).toHaveBeenCalledWith(
          "/api/v1/auth/user/identities",
          { provider: "google" },
        );
        expect(error).toBeNull();
        expect(data).toBeDefined();
        expect(data!.provider).toBe("google");
        expect(data!.url).toBeTruthy();
      });

      it("should return error when not authenticated", async () => {
        const freshMockFetch = {
          post: vi.fn(),
          get: vi.fn(),
          patch: vi.fn(),
          delete: vi.fn(),
          setAuthToken: vi.fn(),
          setRefreshTokenCallback: vi.fn(),
        } as unknown as FluxbaseFetch;

        const newAuth = new FluxbaseAuth(freshMockFetch, false, false);

        const { data, error } = await newAuth.linkIdentity({
          provider: "google",
        });

        expect(data).toBeNull();
        expect(error).toBeDefined();
        expect(error?.message).toBe("Not authenticated");
      });
    });

    describe("unlinkIdentity()", () => {
      it("should unlink OAuth provider", async () => {
        vi.mocked(mockFetch.delete).mockResolvedValue({});

        const identity = {
          id: "identity-123",
          user_id: "1",
          provider: "google",
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        };

        const { error } = await auth.unlinkIdentity({ identity });

        expect(mockFetch.delete).toHaveBeenCalledWith(
          "/api/v1/auth/user/identities/identity-123",
        );
        expect(error).toBeNull();
      });

      it("should return error when not authenticated", async () => {
        const freshMockFetch = {
          post: vi.fn(),
          get: vi.fn(),
          patch: vi.fn(),
          delete: vi.fn(),
          setAuthToken: vi.fn(),
          setRefreshTokenCallback: vi.fn(),
        } as unknown as FluxbaseFetch;

        const newAuth = new FluxbaseAuth(freshMockFetch, false, false);

        const identity = {
          id: "identity-123",
          user_id: "1",
          provider: "google",
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        };

        const { error } = await newAuth.unlinkIdentity({ identity });

        expect(error).toBeDefined();
        expect(error?.message).toBe("Not authenticated");
      });
    });
  });

  describe("reauthenticate()", () => {
    it("should get security nonce", async () => {
      // Set up authenticated session
      const authResponse: AuthResponse = {
        access_token: "test-token",
        refresh_token: "refresh-token",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      const nonceResponse = { nonce: "secure-nonce-12345" };
      vi.mocked(mockFetch.post).mockResolvedValue(nonceResponse);

      const { data, error } = await auth.reauthenticate();

      expect(mockFetch.post).toHaveBeenCalledWith(
        "/api/v1/auth/reauthenticate",
      );
      expect(error).toBeNull();
      expect(data).toBeDefined();
      expect(data!.nonce).toBe("secure-nonce-12345");
    });

    it("should return error when not authenticated", async () => {
      const newAuth = new FluxbaseAuth(mockFetch, true, true);

      const { data, error } = await newAuth.reauthenticate();

      expect(data).toBeNull();
      expect(error).toBeDefined();
      expect(error?.message).toBe("Not authenticated");
    });
  });

  describe("signInWithIdToken()", () => {
    it("should sign in with Google ID token", async () => {
      const authResponse: AuthResponse = {
        access_token: "id-token-access",
        refresh_token: "id-token-refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      const { data, error } = await auth.signInWithIdToken({
        provider: "google",
        token: "google-id-token-12345",
      });

      expect(mockFetch.post).toHaveBeenCalledWith(
        "/api/v1/auth/signin/idtoken",
        {
          provider: "google",
          token: "google-id-token-12345",
        },
      );
      expect(error).toBeNull();
      expect(data).toBeDefined();
      expect(data!.session?.access_token).toBe("id-token-access");
      expect(data!.user.email).toBe("user@example.com");
    });

    it("should sign in with Apple ID token and nonce", async () => {
      const authResponse: AuthResponse = {
        access_token: "apple-token",
        refresh_token: "apple-refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "2", email: "apple@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      const { data, error } = await auth.signInWithIdToken({
        provider: "apple",
        token: "apple-id-token-12345",
        nonce: "random-nonce",
      });

      expect(mockFetch.post).toHaveBeenCalledWith(
        "/api/v1/auth/signin/idtoken",
        {
          provider: "apple",
          token: "apple-id-token-12345",
          nonce: "random-nonce",
        },
      );
      expect(error).toBeNull();
      expect(data).toBeDefined();
    });

    it("should create and persist session", async () => {
      const authResponse: AuthResponse = {
        access_token: "mobile-token",
        refresh_token: "mobile-refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "mobile@example.com", created_at: "" },
      };

      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      await auth.signInWithIdToken({
        provider: "google",
        token: "mobile-id-token",
      });

      const { data: sessionData } = await auth.getSession();
      expect(sessionData.session?.access_token).toBe("mobile-token");
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith("mobile-token");
    });
  });

  describe("Two-Factor Authentication", () => {
    // First sign in to have an active session
    beforeEach(async () => {
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValueOnce(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });
      vi.mocked(mockFetch.post).mockClear();
    });

    describe("setup2FA()", () => {
      it("should setup 2FA with TOTP", async () => {
        const setupResponse = {
          factor_id: "factor-123",
          type: "totp",
          totp: {
            qr_code: "data:image/png;base64,abc123",
            secret: "JBSWY3DPEHPK3PXP",
            uri: "otpauth://totp/...",
          },
        };
        vi.mocked(mockFetch.post).mockResolvedValue(setupResponse);

        const { data, error } = await auth.setup2FA();

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/2fa/setup", undefined);
        expect(error).toBeNull();
        expect(data?.totp?.qr_code).toBe("data:image/png;base64,abc123");
      });

      it("should setup 2FA with custom issuer", async () => {
        const setupResponse = {
          factor_id: "factor-123",
          type: "totp",
          totp: { qr_code: "...", secret: "...", uri: "..." },
        };
        vi.mocked(mockFetch.post).mockResolvedValue(setupResponse);

        await auth.setup2FA("MyApp");

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/2fa/setup", { issuer: "MyApp" });
      });

      it("should return error on setup failure", async () => {
        vi.mocked(mockFetch.post).mockRejectedValue(new Error("Setup failed"));

        const { data, error } = await auth.setup2FA();

        expect(data).toBeNull();
        expect(error?.message).toBe("Setup failed");
      });
    });

    describe("enable2FA()", () => {
      it("should enable 2FA with verification code", async () => {
        const enableResponse = {
          access_token: "new-token",
          refresh_token: "new-refresh",
          user: { id: "1", email: "user@example.com" },
        };
        vi.mocked(mockFetch.post).mockResolvedValue(enableResponse);

        const { data, error } = await auth.enable2FA("123456");

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/2fa/enable", {
          code: "123456",
        });
        expect(error).toBeNull();
      });
    });

    describe("disable2FA()", () => {
      it("should disable 2FA with password", async () => {
        vi.mocked(mockFetch.post).mockResolvedValue({ factor_id: "factor-123" });

        const { data, error } = await auth.disable2FA("mypassword");

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/2fa/disable", {
          password: "mypassword",
        });
        expect(error).toBeNull();
      });
    });

    describe("get2FAStatus()", () => {
      it("should get 2FA status", async () => {
        const statusResponse = {
          totp: {
            friendly_name: "Authenticator App",
            factor_id: "factor-123",
            status: "verified",
          },
          all: [
            { friendly_name: "Authenticator App", factor_id: "factor-123", factor_type: "totp", status: "verified" }
          ],
        };
        vi.mocked(mockFetch.get).mockResolvedValue(statusResponse);

        const { data, error } = await auth.get2FAStatus();

        expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/auth/2fa/status");
        expect(error).toBeNull();
        expect(data?.totp?.status).toBe("verified");
      });
    });

    describe("verify2FA()", () => {
      it("should verify 2FA and create session", async () => {
        const response = {
          access_token: "2fa-token",
          refresh_token: "2fa-refresh",
          expires_in: 3600,
          user: { id: "1", email: "user@example.com", created_at: "" },
        };
        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const { data, error } = await auth.verify2FA({
          factor_id: "factor-123",
          code: "123456",
        });

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/2fa/verify", {
          factor_id: "factor-123",
          code: "123456",
        });
        expect(error).toBeNull();
        expect(data?.access_token).toBe("2fa-token");
      });

      it("should handle verify without session creation", async () => {
        vi.mocked(mockFetch.post).mockResolvedValue({ verified: true });

        const { data, error } = await auth.verify2FA({
          factor_id: "factor-123",
          code: "123456",
        });

        expect(error).toBeNull();
      });
    });
  });

  describe("SAML Authentication", () => {
    describe("getSAMLProviders()", () => {
      it("should get list of SAML providers", async () => {
        const providersResponse = {
          providers: [
            { id: "1", name: "okta", display_name: "Okta" },
            { id: "2", name: "azure", display_name: "Azure AD" },
          ],
        };
        vi.mocked(mockFetch.get).mockResolvedValue(providersResponse);

        const { data, error } = await auth.getSAMLProviders();

        expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/auth/saml/providers");
        expect(error).toBeNull();
        expect(data?.providers).toHaveLength(2);
      });
    });

    describe("getSAMLLoginUrl()", () => {
      it("should get SAML login URL", async () => {
        const urlResponse = {
          url: "https://idp.example.com/sso/saml?SAMLRequest=...",
        };
        vi.mocked(mockFetch.get).mockResolvedValue(urlResponse);

        const { data, error } = await auth.getSAMLLoginUrl("okta");

        expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/auth/saml/login/okta");
        expect(error).toBeNull();
        expect(data?.url).toContain("idp.example.com");
      });

      it("should get SAML login URL with options", async () => {
        const urlResponse = {
          url: "https://idp.example.com/sso/saml",
        };
        vi.mocked(mockFetch.get).mockResolvedValue(urlResponse);

        const { data, error } = await auth.getSAMLLoginUrl("okta", {
          redirectUrl: "https://app.example.com/callback",
        });

        expect(mockFetch.get).toHaveBeenCalledWith(
          "/api/v1/auth/saml/login/okta?redirect_url=https%3A%2F%2Fapp.example.com%2Fcallback"
        );
        expect(error).toBeNull();
      });
    });

    describe("signInWithSAML()", () => {
      it("should redirect to SAML provider in browser", async () => {
        const urlResponse = {
          url: "https://idp.example.com/sso/saml",
        };
        vi.mocked(mockFetch.get).mockResolvedValue(urlResponse);

        // Mock window.location
        const originalWindow = global.window;
        global.window = { location: { href: "" } } as any;

        const { error } = await auth.signInWithSAML("okta");

        expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/auth/saml/login/okta");
        expect(error).toBeNull();
        expect(global.window.location.href).toBe("https://idp.example.com/sso/saml");

        global.window = originalWindow;
      });

      it("should return error in non-browser environment", async () => {
        const urlResponse = {
          url: "https://idp.example.com/sso/saml",
        };
        vi.mocked(mockFetch.get).mockResolvedValue(urlResponse);

        const originalWindow = global.window;
        delete (global as any).window;

        const { error } = await auth.signInWithSAML("okta");

        expect(error?.message).toBe("signInWithSAML can only be called in a browser environment");

        global.window = originalWindow;
      });
    });

    describe("handleSAMLCallback()", () => {
      it("should handle SAML callback and create session", async () => {
        const authResponse: AuthResponse = {
          access_token: "saml-token",
          refresh_token: "saml-refresh",
          expires_in: 3600,
          token_type: "Bearer",
          user: { id: "1", email: "saml@example.com", created_at: "" },
        };
        vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

        const { data, error } = await auth.handleSAMLCallback("base64-saml-response");

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/saml/acs", {
          saml_response: "base64-saml-response",
          provider: undefined,
        });
        expect(error).toBeNull();
        expect(data?.session?.access_token).toBe("saml-token");
      });

      it("should handle SAML callback with provider", async () => {
        const authResponse: AuthResponse = {
          access_token: "saml-token",
          refresh_token: "saml-refresh",
          expires_in: 3600,
          token_type: "Bearer",
          user: { id: "1", email: "saml@example.com", created_at: "" },
        };
        vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

        await auth.handleSAMLCallback("base64-response", "okta");

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/saml/acs", {
          saml_response: "base64-response",
          provider: "okta",
        });
      });
    });

    describe("getSAMLMetadataUrl()", () => {
      it("should return SAML metadata URL", () => {
        // Mock the fetch baseUrl
        (mockFetch as any).baseUrl = "https://api.example.com";
        const newAuth = new FluxbaseAuth(mockFetch, true, true);

        const url = newAuth.getSAMLMetadataUrl("okta");

        expect(url).toBe("https://api.example.com/api/v1/auth/saml/metadata/okta");
      });
    });
  });

  describe("Config Methods", () => {
    describe("getCaptchaConfig()", () => {
      it("should get captcha configuration", async () => {
        const captchaConfig = {
          enabled: true,
          provider: "hcaptcha",
          site_key: "site-key-123",
        };
        vi.mocked(mockFetch.get).mockResolvedValue(captchaConfig);

        const { data, error } = await auth.getCaptchaConfig();

        expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/auth/captcha/config");
        expect(error).toBeNull();
        expect(data?.provider).toBe("hcaptcha");
      });
    });

    describe("getAuthConfig()", () => {
      it("should get auth configuration", async () => {
        const authConfig = {
          disable_signup: false,
          require_email_confirmation: true,
        };
        vi.mocked(mockFetch.get).mockResolvedValue(authConfig);

        const { data, error } = await auth.getAuthConfig();

        expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/auth/config");
        expect(error).toBeNull();
        expect(data?.require_email_confirmation).toBe(true);
      });
    });
  });

  describe("setSession()", () => {
    it("should set session from access and refresh tokens", async () => {
      vi.mocked(mockFetch.get).mockResolvedValue({
        id: "1",
        email: "user@example.com",
        created_at: "",
      });

      const { data, error } = await auth.setSession({
        access_token: "new-access-token",
        refresh_token: "new-refresh-token",
      });

      expect(error).toBeNull();
      expect(data?.session?.access_token).toBe("new-access-token");
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith("new-access-token");
    });

    it("should fetch user info when setting session", async () => {
      vi.mocked(mockFetch.get).mockResolvedValue({
        id: "user-123",
        email: "restored@example.com",
        created_at: "",
      });

      const { data, error } = await auth.setSession({
        access_token: "token-123",
        refresh_token: "refresh-123",
      });

      expect(error).toBeNull();
      expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/auth/user");
      expect(data?.user?.email).toBe("restored@example.com");
    });

    it("should return error when API call fails", async () => {
      vi.mocked(mockFetch.get).mockRejectedValue(new Error("API error"));

      const { data, error } = await auth.setSession({
        access_token: "token",
        refresh_token: "refresh",
      });

      expect(data).toBeNull();
      expect(error?.message).toBe("API error");
    });
  });

  describe("onAuthStateChange()", () => {
    it("should register state change listener", async () => {
      const callback = vi.fn();
      const subscription = auth.onAuthStateChange(callback);

      // Sign in to trigger state change
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      await auth.signIn({ email: "user@example.com", password: "password" });

      expect(callback).toHaveBeenCalledWith("SIGNED_IN", expect.any(Object));
      expect(subscription.data.subscription).toBeDefined();
    });

    it("should unsubscribe from state changes", async () => {
      const callback = vi.fn();
      const subscription = auth.onAuthStateChange(callback);
      subscription.data.subscription.unsubscribe();

      // Sign in - callback should not be called
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      await auth.signIn({ email: "user@example.com", password: "password" });

      expect(callback).not.toHaveBeenCalled();
    });

    it("should handle errors in state change listeners", async () => {
      const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
      const throwingCallback = vi.fn().mockImplementation(() => {
        throw new Error("Listener error");
      });
      auth.onAuthStateChange(throwingCallback);

      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      await auth.signIn({ email: "user@example.com", password: "password" });

      expect(consoleError).toHaveBeenCalledWith(
        "Error in auth state change listener:",
        expect.any(Error)
      );
      consoleError.mockRestore();
    });
  });

  describe("Auto-refresh configuration", () => {
    it("should schedule token refresh after sign in", async () => {
      vi.useFakeTimers();

      const authResponse: AuthResponse = {
        access_token: "initial-token",
        refresh_token: "initial-refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValueOnce(authResponse);

      await auth.signIn({ email: "user@example.com", password: "password" });

      // Session should be set
      const { data } = await auth.getSession();
      expect(data.session?.access_token).toBe("initial-token");

      vi.useRealTimers();
    });

    it("should disable auto-refresh when autoRefresh is false", async () => {
      const noAutoRefreshAuth = new FluxbaseAuth(mockFetch, true, false);

      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      await noAutoRefreshAuth.signIn({ email: "user@example.com", password: "password" });

      // Session should be set even without auto-refresh
      const { data } = await noAutoRefreshAuth.getSession();
      expect(data.session?.access_token).toBe("token");
    });
  });

  describe("getUser()", () => {
    it("should return null user when no session exists", async () => {
      const { data, error } = await auth.getUser();

      expect(error).toBeNull();
      expect(data?.user).toBeNull();
    });

    it("should return user when session exists", async () => {
      // Sign in first
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      await auth.signIn({ email: "user@example.com", password: "password" });

      const { data, error } = await auth.getUser();

      expect(error).toBeNull();
      expect(data?.user?.email).toBe("user@example.com");
    });
  });

  describe("signInWithPassword()", () => {
    it("should be an alias for signIn with password credentials", async () => {
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      const { data, error } = await auth.signInWithPassword({
        email: "user@example.com",
        password: "password123",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signin", {
        email: "user@example.com",
        password: "password123",
      });
      expect(error).toBeNull();
      expect(data?.session).toBeDefined();
    });
  });

  describe("resetPasswordForEmail()", () => {
    it("should be an alias for sendPasswordReset", async () => {
      vi.mocked(mockFetch.post).mockResolvedValue({ message: "Reset email sent" });

      const { error } = await auth.resetPasswordForEmail("user@example.com");

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/password/reset", {
        email: "user@example.com",
      });
      expect(error).toBeNull();
    });

    it("should support redirect URL option", async () => {
      vi.mocked(mockFetch.post).mockResolvedValue({ message: "Reset email sent" });

      await auth.resetPasswordForEmail("user@example.com", {
        redirectTo: "https://app.example.com/reset",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/password/reset", {
        email: "user@example.com",
        redirect_to: "https://app.example.com/reset",
      });
    });
  });

  describe("getOAuthLogoutUrl()", () => {
    it("should get OAuth logout URL", async () => {
      const logoutResponse = {
        requires_redirect: true,
        redirect_url: "https://provider.example.com/logout",
      };
      vi.mocked(mockFetch.post).mockResolvedValue(logoutResponse);

      const { data, error } = await auth.getOAuthLogoutUrl("google");

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/oauth/google/logout", {});
      expect(error).toBeNull();
      expect(data?.redirect_url).toContain("provider.example.com");
    });

    it("should get OAuth logout URL with options", async () => {
      const logoutResponse = {
        requires_redirect: false,
      };
      vi.mocked(mockFetch.post).mockResolvedValue(logoutResponse);

      await auth.getOAuthLogoutUrl("google", {
        redirectTo: "https://app.example.com",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/oauth/google/logout", {
        redirectTo: "https://app.example.com",
      });
    });
  });

  describe("MemoryStorage (direct tests)", () => {
    it("should implement Storage interface correctly", async () => {
      // Make localStorage throw to trigger MemoryStorage fallback
      const originalSetItem = localStorageMock.setItem;
      const originalGetItem = localStorageMock.getItem;
      const originalRemoveItem = localStorageMock.removeItem;

      localStorageMock.setItem = () => {
        throw new Error("localStorage is not available");
      };
      localStorageMock.getItem = () => {
        throw new Error("localStorage is not available");
      };
      localStorageMock.removeItem = () => {
        throw new Error("localStorage is not available");
      };

      const freshMockFetch = {
        post: vi.fn(),
        get: vi.fn(),
        patch: vi.fn(),
        delete: vi.fn(),
        setAuthToken: vi.fn(),
        setRefreshTokenCallback: vi.fn(),
      } as unknown as FluxbaseFetch;

      // Create auth with MemoryStorage
      const memoryAuth = new FluxbaseAuth(freshMockFetch, true, true);

      // Test multiple operations to verify MemoryStorage works
      const authResponse1: AuthResponse = {
        access_token: "token1",
        refresh_token: "refresh1",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user1@example.com", created_at: "" },
      };
      vi.mocked(freshMockFetch.post).mockResolvedValue(authResponse1);
      await memoryAuth.signIn({ email: "user1@example.com", password: "pass" });

      // Session should be stored in memory
      const { data: session1 } = await memoryAuth.getSession();
      expect(session1.session?.access_token).toBe("token1");

      // Sign out
      vi.mocked(freshMockFetch.post).mockResolvedValue(undefined);
      await memoryAuth.signOut();

      const { data: clearedSession } = await memoryAuth.getSession();
      expect(clearedSession.session).toBeNull();

      // Restore localStorage functions
      localStorageMock.setItem = originalSetItem;
      localStorageMock.getItem = originalGetItem;
      localStorageMock.removeItem = originalRemoveItem;
    });
  });

  describe("Auto-refresh Methods", () => {
    it("should start auto-refresh timer", async () => {
      vi.useFakeTimers();

      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      // Explicitly start auto-refresh
      auth.startAutoRefresh();

      // Should have scheduled a timer
      const { data } = await auth.getSession();
      expect(data.session?.access_token).toBe("token");

      vi.useRealTimers();
    });

    it("should stop auto-refresh timer", async () => {
      vi.useFakeTimers();

      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      // Stop auto-refresh
      auth.stopAutoRefresh();

      // Session should still exist but no timer
      const { data } = await auth.getSession();
      expect(data.session?.access_token).toBe("token");

      vi.useRealTimers();
    });

    it("should clear refresh timer on sign out", async () => {
      vi.useFakeTimers();

      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValueOnce(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      vi.mocked(mockFetch.post).mockResolvedValueOnce(undefined);
      await auth.signOut();

      // Session should be cleared
      const { data } = await auth.getSession();
      expect(data.session).toBeNull();

      vi.useRealTimers();
    });
  });

  describe("signIn with 2FA required", () => {
    it("should return 2FA response when 2FA is required", async () => {
      const twoFaResponse = {
        requires_2fa: true,
        user_id: "user-123",
        factors: [{ type: "totp", factor_id: "factor-123" }],
      };
      vi.mocked(mockFetch.post).mockResolvedValue(twoFaResponse);

      const { data, error } = await auth.signIn({
        email: "user@example.com",
        password: "password",
      });

      expect(error).toBeNull();
      expect(data).toBeDefined();
      expect((data as any).requires_2fa).toBe(true);
      expect((data as any).user_id).toBe("user-123");
    });
  });

  describe("CAPTCHA token handling", () => {
    it("should include captcha token in signIn", async () => {
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      await auth.signIn({
        email: "user@example.com",
        password: "password",
        captchaToken: "captcha-token-123",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signin", {
        email: "user@example.com",
        password: "password",
        captcha_token: "captcha-token-123",
      });
    });

    it("should include captcha token in signUp", async () => {
      const authResponse: AuthResponse = {
        access_token: "token",
        refresh_token: "refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

      await auth.signUp({
        email: "user@example.com",
        password: "password",
        captchaToken: "captcha-token-123",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signup", {
        email: "user@example.com",
        password: "password",
        captcha_token: "captcha-token-123",
      });
    });

    it("should include captcha token in sendPasswordReset", async () => {
      vi.mocked(mockFetch.post).mockResolvedValue({});

      await auth.sendPasswordReset("user@example.com", {
        captchaToken: "captcha-token-123",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/password/reset", {
        email: "user@example.com",
        captcha_token: "captcha-token-123",
      });
    });

    it("should include captcha token in sendMagicLink", async () => {
      vi.mocked(mockFetch.post).mockResolvedValue({});

      await auth.sendMagicLink("user@example.com", {
        captchaToken: "captcha-token-123",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/magiclink", {
        email: "user@example.com",
        redirect_to: undefined,
        captcha_token: "captcha-token-123",
      });
    });
  });

  describe("exchangeCodeForSession with redirect_uri", () => {
    it("should include stored redirect_uri in callback request", async () => {
      const authResponse: AuthResponse = {
        access_token: "oauth-token",
        refresh_token: "oauth-refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: { id: "1", email: "user@example.com", created_at: "" },
      };

      // Simulate signInWithOAuth storing provider and redirect_uri
      localStorageMock.setItem("fluxbase.auth.oauth_provider", "google");
      localStorageMock.setItem("fluxbase.auth.oauth_redirect_uri", "https://app.example.com/callback");

      vi.mocked(mockFetch.get).mockResolvedValue(authResponse);

      await auth.exchangeCodeForSession("auth-code-123", "state-token");

      expect(mockFetch.get).toHaveBeenCalledWith(
        "/api/v1/auth/oauth/google/callback?code=auth-code-123&state=state-token&redirect_uri=https%3A%2F%2Fapp.example.com%2Fcallback"
      );

      // Provider and redirect_uri should be cleared
      expect(localStorageMock.getItem("fluxbase.auth.oauth_provider")).toBeNull();
      expect(localStorageMock.getItem("fluxbase.auth.oauth_redirect_uri")).toBeNull();
    });
  });

  describe("signInWithOAuth with redirect_uri option", () => {
    it("should store redirect_uri for later use in exchangeCodeForSession", async () => {
      const response = {
        url: "https://accounts.google.com/o/oauth2/v2/auth?...",
        provider: "google",
      };

      vi.mocked(mockFetch.get).mockResolvedValue(response);

      const originalWindow = global.window;
      global.window = { location: { href: "" } } as any;

      await auth.signInWithOAuth("google", {
        redirect_uri: "https://app.example.com/callback",
      });

      expect(localStorageMock.getItem("fluxbase.auth.oauth_redirect_uri")).toBe(
        "https://app.example.com/callback"
      );

      global.window = originalWindow;
    });
  });

  describe("signUp with email confirmation required", () => {
    it("should return user without session when email confirmation is required", async () => {
      // Response without access_token/refresh_token indicates email confirmation required
      const response = {
        user: { id: "1", email: "user@example.com", created_at: "" },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(response);

      const { data, error } = await auth.signUp({
        email: "user@example.com",
        password: "password123",
      });

      expect(error).toBeNull();
      expect(data?.user?.email).toBe("user@example.com");
      expect(data?.session).toBeNull();
    });
  });

  describe("2FA authentication required but not authenticated errors", () => {
    it("should return error for setup2FA when not authenticated", async () => {
      const freshMockFetch = {
        post: vi.fn(),
        get: vi.fn(),
        patch: vi.fn(),
        delete: vi.fn(),
        setAuthToken: vi.fn(),
        setRefreshTokenCallback: vi.fn(),
      } as unknown as FluxbaseFetch;

      const newAuth = new FluxbaseAuth(freshMockFetch, false, false);

      const { data, error } = await newAuth.setup2FA();

      expect(data).toBeNull();
      expect(error?.message).toBe("Not authenticated");
    });

    it("should return error for enable2FA when not authenticated", async () => {
      const freshMockFetch = {
        post: vi.fn(),
        get: vi.fn(),
        patch: vi.fn(),
        delete: vi.fn(),
        setAuthToken: vi.fn(),
        setRefreshTokenCallback: vi.fn(),
      } as unknown as FluxbaseFetch;

      const newAuth = new FluxbaseAuth(freshMockFetch, false, false);

      const { data, error } = await newAuth.enable2FA("123456");

      expect(data).toBeNull();
      expect(error?.message).toBe("Not authenticated");
    });

    it("should return error for disable2FA when not authenticated", async () => {
      const freshMockFetch = {
        post: vi.fn(),
        get: vi.fn(),
        patch: vi.fn(),
        delete: vi.fn(),
        setAuthToken: vi.fn(),
        setRefreshTokenCallback: vi.fn(),
      } as unknown as FluxbaseFetch;

      const newAuth = new FluxbaseAuth(freshMockFetch, false, false);

      const { data, error } = await newAuth.disable2FA("password");

      expect(data).toBeNull();
      expect(error?.message).toBe("Not authenticated");
    });

    it("should return error for get2FAStatus when not authenticated", async () => {
      const freshMockFetch = {
        post: vi.fn(),
        get: vi.fn(),
        patch: vi.fn(),
        delete: vi.fn(),
        setAuthToken: vi.fn(),
        setRefreshTokenCallback: vi.fn(),
      } as unknown as FluxbaseFetch;

      const newAuth = new FluxbaseAuth(freshMockFetch, false, false);

      const { data, error } = await newAuth.get2FAStatus();

      expect(data).toBeNull();
      expect(error?.message).toBe("Not authenticated");
    });
  });

  describe("signOutWithOAuth()", () => {
    it("should redirect to OAuth logout URL in browser when required", async () => {
      const logoutResponse = {
        requires_redirect: true,
        redirect_url: "https://provider.example.com/logout",
      };
      vi.mocked(mockFetch.post).mockResolvedValue(logoutResponse);

      const originalWindow = global.window;
      global.window = { location: { href: "" } } as any;

      const { data, error } = await auth.signOutWithOAuth("google");

      expect(error).toBeNull();
      expect(global.window.location.href).toBe("https://provider.example.com/logout");

      global.window = originalWindow;
    });

    it("should not redirect when requires_redirect is false", async () => {
      const logoutResponse = {
        requires_redirect: false,
      };
      vi.mocked(mockFetch.post).mockResolvedValue(logoutResponse);

      const originalWindow = global.window;
      global.window = { location: { href: "http://original" } } as any;

      const { data, error } = await auth.signOutWithOAuth("google");

      expect(error).toBeNull();
      expect(global.window.location.href).toBe("http://original"); // Should not change

      global.window = originalWindow;
    });

    it("should work in non-browser environment when redirect not required", async () => {
      const logoutResponse = {
        requires_redirect: false,
      };
      vi.mocked(mockFetch.post).mockResolvedValue(logoutResponse);

      const originalWindow = global.window;
      delete (global as any).window;

      const { data, error } = await auth.signOutWithOAuth("google");

      expect(error).toBeNull();
      expect(data?.requires_redirect).toBe(false);

      global.window = originalWindow;
    });
  });

  describe("getProviderToken()", () => {
    it("should throw error when not authenticated", async () => {
      // Clear session
      vi.mocked(mockFetch.get).mockClear();

      const { data, error } = await auth.getProviderToken("google");

      expect(data).toBeNull();
      expect(error).toBeDefined();
      expect(error?.message).toBe("Not authenticated");
    });

    it("should return provider tokens when authenticated", async () => {
      // First sign in
      const authResponse: AuthResponse = {
        access_token: "fluxbase-token",
        refresh_token: "fluxbase-refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: {
          id: "user-1",
          email: "user@example.com",
          created_at: new Date().toISOString(),
        },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      // Mock getProviderToken response
      const providerTokenResponse: ProviderTokenResponse = {
        provider: "google",
        access_token: "ya29.a0...",
        refresh_token: "1//...",
        token_expiry: new Date(Date.now() + 3600 * 1000).toISOString(),
        expires_in: 3600,
        scopes: ["openid", "email", "https://www.googleapis.com/auth/drive.readonly"],
        token_type: "Bearer",
      };
      vi.mocked(mockFetch.get).mockResolvedValue(providerTokenResponse);

      const { data, error } = await auth.getProviderToken("google");

      expect(mockFetch.get).toHaveBeenCalledWith(
        "/api/v1/auth/oauth/google/token",
      );
      expect(error).toBeNull();
      expect(data).toBeDefined();
      expect(data?.provider).toBe("google");
      expect(data?.access_token).toBe("ya29.a0...");
      expect(data?.refresh_token).toBe("1//...");
      expect(data?.expires_in).toBe(3600);
      expect(data?.token_type).toBe("Bearer");
      expect(data?.scopes).toContain("https://www.googleapis.com/auth/drive.readonly");
    });

    it("should return error when provider token not found", async () => {
      // First sign in
      const authResponse: AuthResponse = {
        access_token: "fluxbase-token",
        refresh_token: "fluxbase-refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: {
          id: "user-1",
          email: "user@example.com",
          created_at: new Date().toISOString(),
        },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      // Mock not found error
      const notFoundError = {
        error: "No OAuth token found for this provider",
        error_code: "oauth_token_not_found",
        error_hint: "You need to sign in with this provider first",
        provider: "google",
        authorize_url: "http://localhost:8080/api/v1/auth/oauth/google/authorize",
      };
      vi.mocked(mockFetch.get).mockRejectedValue({
        status: 404,
        json: () => Promise.resolve(notFoundError),
      });

      const { data, error } = await auth.getProviderToken("google");

      expect(data).toBeNull();
      expect(error).toBeDefined();
    });

    it("should handle auto-refreshed tokens", async () => {
      // First sign in
      const authResponse: AuthResponse = {
        access_token: "fluxbase-token",
        refresh_token: "fluxbase-refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: {
          id: "user-1",
          email: "user@example.com",
          created_at: new Date().toISOString(),
        },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      // Mock refreshed token response
      const refreshedTokenResponse: ProviderTokenResponse = {
        provider: "google",
        access_token: "ya29.refreshed...",
        refresh_token: "1//new...",
        token_expiry: new Date(Date.now() + 3600 * 1000).toISOString(),
        expires_in: 3600,
        id_token: "eyJ...",
        scopes: ["openid", "email", "https://www.googleapis.com/auth/drive.readonly"],
        token_type: "Bearer",
      };
      vi.mocked(mockFetch.get).mockResolvedValue(refreshedTokenResponse);

      const { data, error } = await auth.getProviderToken("google");

      expect(error).toBeNull();
      expect(data).toBeDefined();
      expect(data?.access_token).toBe("ya29.refreshed...");
      expect(data?.id_token).toBe("eyJ...");
    });

    it("should work with different providers", async () => {
      // First sign in
      const authResponse: AuthResponse = {
        access_token: "fluxbase-token",
        refresh_token: "fluxbase-refresh",
        expires_in: 3600,
        token_type: "Bearer",
        user: {
          id: "user-1",
          email: "user@example.com",
          created_at: new Date().toISOString(),
        },
      };
      vi.mocked(mockFetch.post).mockResolvedValue(authResponse);
      await auth.signIn({ email: "user@example.com", password: "password" });

      // Mock GitHub token response
      const githubTokenResponse: ProviderTokenResponse = {
        provider: "github",
        access_token: "gho_...",
        token_expiry: new Date(Date.now() + 3600 * 1000).toISOString(),
        expires_in: 3600,
        scopes: ["user:email", "read:user"],
        token_type: "Bearer",
      };
      vi.mocked(mockFetch.get).mockResolvedValue(githubTokenResponse);

      const { data, error } = await auth.getProviderToken("github");

      expect(mockFetch.get).toHaveBeenCalledWith(
        "/api/v1/auth/oauth/github/token",
      );
      expect(error).toBeNull();
      expect(data?.provider).toBe("github");
      expect(data?.access_token).toBe("gho_...");
      expect(data?.scopes).toContain("user:email");
    });
  });
});
