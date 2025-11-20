/**
 * Authentication Tests
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { FluxbaseAuth } from "./auth";
import type { FluxbaseFetch } from "./fetch";
import type { AuthResponse } from "./types";

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

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/refresh", {
        refresh_token: "refresh-token",
      });
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

        vi.mocked(mockFetch.post).mockResolvedValue(authResponse);

        const { data: session, error } =
          await auth.exchangeCodeForSession("auth-code-123");

        expect(mockFetch.post).toHaveBeenCalledWith(
          "/api/v1/auth/oauth/callback",
          {
            code: "auth-code-123",
          },
        );
        expect(error).toBeNull();
        expect(session).toBeDefined();
        expect(session!.user.email).toBe("user@example.com");
        expect(session!.session.access_token).toBe("oauth-token");
        const { data: sessionData } = await auth.getSession();
        expect(sessionData.session?.access_token).toBe("oauth-token");
      });
    });

    describe("signInWithOAuth()", () => {
      it("should redirect to OAuth provider in browser", async () => {
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
      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/refresh", {
        refresh_token: "refresh-token",
      });
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
});
