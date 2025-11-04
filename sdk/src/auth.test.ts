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
      setAuthToken: vi.fn(),
    } as unknown as FluxbaseFetch;

    auth = new FluxbaseAuth(mockFetch, true, true);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("initialization", () => {
    it("should initialize with no session", () => {
      expect(auth.getSession()).toBeNull();
      expect(auth.getUser()).toBeNull();
      expect(auth.getAccessToken()).toBeNull();
    });

    it("should restore session from localStorage", () => {
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

      expect(newAuth.getSession()).toEqual(session);
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith("test-token");
    });

    it("should ignore invalid stored session", () => {
      localStorage.setItem("fluxbase.auth.session", "invalid-json");

      const newAuth = new FluxbaseAuth(mockFetch, true, true);

      expect(newAuth.getSession()).toBeNull();
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

      const session = await auth.signIn({
        email: "user@example.com",
        password: "password123",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signin", {
        email: "user@example.com",
        password: "password123",
      });
      expect(session.access_token).toBe("new-access-token");
      expect(session.user.email).toBe("user@example.com");
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

      const session = await auth.signUp({
        email: "newuser@example.com",
        password: "password123",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signup", {
        email: "newuser@example.com",
        password: "password123",
      });
      expect(session.user.email).toBe("newuser@example.com");
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
      await auth.signOut();

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signout");
      expect(auth.getSession()).toBeNull();
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
      vi.mocked(mockFetch.post).mockRejectedValueOnce(new Error("Network error"));

      // Should still resolve but session is cleared due to finally block
      try {
        await auth.signOut();
      } catch (error) {
        // Expected to throw, but session should still be cleared
      }

      expect(auth.getSession()).toBeNull();
    });
  });

  describe("refreshToken()", () => {
    it("should refresh access token", async () => {
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

      const session = await auth.refreshToken();

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/refresh", {
        refresh_token: "refresh-token",
      });
      expect(session.access_token).toBe("new-token");
      expect(mockFetch.setAuthToken).toHaveBeenCalledWith("new-token");
    });

    it("should throw error when no refresh token available", async () => {
      await expect(auth.refreshToken()).rejects.toThrow(
        "No refresh token available"
      );
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

      const result = await auth.getCurrentUser();

      expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/auth/user");
      expect(result).toEqual(user);
    });

    it("should throw error when not authenticated", async () => {
      await expect(auth.getCurrentUser()).rejects.toThrow("Not authenticated");
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

      const result = await auth.updateUser({ email: "new@example.com" });

      expect(mockFetch.patch).toHaveBeenCalledWith("/api/v1/auth/user", {
        email: "new@example.com",
      });
      expect(result.email).toBe("new@example.com");
      expect(auth.getUser()?.email).toBe("new@example.com");
    });

    it("should throw error when not authenticated", async () => {
      await expect(
        auth.updateUser({ email: "new@example.com" })
      ).rejects.toThrow("Not authenticated");
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
        const response = {
          message: "If an account with that email exists, a password reset link has been sent",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const result = await auth.sendPasswordReset("user@example.com");

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/password/reset", {
          email: "user@example.com",
        });
        expect(result.message).toBe(response.message);
      });
    });

    describe("verifyResetToken()", () => {
      it("should verify valid reset token", async () => {
        const response = {
          valid: true,
          message: "Token is valid",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const result = await auth.verifyResetToken("valid-token");

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/password/reset/verify", {
          token: "valid-token",
        });
        expect(result.valid).toBe(true);
      });

      it("should return invalid for expired token", async () => {
        const response = {
          valid: false,
          message: "Token has expired",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const result = await auth.verifyResetToken("expired-token");

        expect(result.valid).toBe(false);
      });
    });

    describe("resetPassword()", () => {
      it("should reset password with valid token", async () => {
        const response = {
          message: "Password has been successfully reset",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const result = await auth.resetPassword("valid-token", "newPassword123");

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/password/reset/confirm", {
          token: "valid-token",
          new_password: "newPassword123",
        });
        expect(result.message).toBe(response.message);
      });
    });
  });

  describe("Magic Link Authentication", () => {
    describe("sendMagicLink()", () => {
      it("should send magic link without options", async () => {
        const response = {
          message: "Magic link sent to your email",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const result = await auth.sendMagicLink("user@example.com");

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/magiclink", {
          email: "user@example.com",
          redirect_to: undefined,
        });
        expect(result.message).toBe(response.message);
      });

      it("should send magic link with redirect URL", async () => {
        const response = {
          message: "Magic link sent to your email",
        };

        vi.mocked(mockFetch.post).mockResolvedValue(response);

        const result = await auth.sendMagicLink("user@example.com", {
          redirect_to: "https://app.example.com/dashboard",
        });

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/magiclink", {
          email: "user@example.com",
          redirect_to: "https://app.example.com/dashboard",
        });
        expect(result.message).toBe(response.message);
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

        const session = await auth.verifyMagicLink("magic-link-token");

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/magiclink/verify", {
          token: "magic-link-token",
        });
        expect(session.access_token).toBe("magic-token");
        expect(auth.getSession()).toEqual(session);
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

        const session = await auth.signInAnonymously();

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signin/anonymous");
        expect(session.access_token).toBe("anon-token");
        expect(auth.getSession()).toEqual(session);
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

        const result = await auth.getOAuthProviders();

        expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/auth/oauth/providers");
        expect(result.providers).toHaveLength(2);
        expect(result.providers[0].id).toBe("google");
      });
    });

    describe("getOAuthUrl()", () => {
      it("should get OAuth URL without options", async () => {
        const response = {
          url: "https://accounts.google.com/o/oauth2/v2/auth?...",
          provider: "google",
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const result = await auth.getOAuthUrl("google");

        expect(mockFetch.get).toHaveBeenCalledWith("/api/v1/auth/oauth/google/authorize");
        expect(result.url).toContain("google.com");
      });

      it("should get OAuth URL with redirect_to", async () => {
        const response = {
          url: "https://accounts.google.com/o/oauth2/v2/auth?...",
          provider: "google",
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const result = await auth.getOAuthUrl("google", {
          redirect_to: "https://app.example.com/auth/callback",
        });

        expect(mockFetch.get).toHaveBeenCalledWith(
          "/api/v1/auth/oauth/google/authorize?redirect_to=https%3A%2F%2Fapp.example.com%2Fauth%2Fcallback"
        );
      });

      it("should get OAuth URL with scopes", async () => {
        const response = {
          url: "https://accounts.google.com/o/oauth2/v2/auth?...",
          provider: "google",
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const result = await auth.getOAuthUrl("google", {
          scopes: ["email", "profile"],
        });

        expect(mockFetch.get).toHaveBeenCalledWith(
          "/api/v1/auth/oauth/google/authorize?scopes=email%2Cprofile"
        );
      });

      it("should get OAuth URL with both redirect_to and scopes", async () => {
        const response = {
          url: "https://github.com/login/oauth/authorize?...",
          provider: "github",
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        const result = await auth.getOAuthUrl("github", {
          redirect_to: "https://app.example.com/callback",
          scopes: ["read:user", "repo"],
        });

        expect(mockFetch.get).toHaveBeenCalledWith(
          expect.stringContaining("/api/v1/auth/oauth/github/authorize?")
        );
        expect(mockFetch.get).toHaveBeenCalledWith(
          expect.stringContaining("redirect_to=")
        );
        expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining("scopes="));
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

        const session = await auth.exchangeCodeForSession("auth-code-123");

        expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/oauth/callback", {
          code: "auth-code-123",
        });
        expect(session.access_token).toBe("oauth-token");
        expect(auth.getSession()).toEqual(session);
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

        await auth.signInWithOAuth("google");

        expect(window.location.href).toBe(response.url);

        // Restore
        if (originalLocation) {
          (global as any).window = { location: originalLocation };
        } else {
          delete (global as any).window;
        }
      });

      it("should throw error in non-browser environment", async () => {
        const response = {
          url: "https://accounts.google.com/o/oauth2/v2/auth?...",
          provider: "google",
        };

        vi.mocked(mockFetch.get).mockResolvedValue(response);

        // Ensure window is undefined
        const originalWindow = global.window;
        delete (global as any).window;

        await expect(auth.signInWithOAuth("google")).rejects.toThrow(
          "signInWithOAuth can only be called in a browser environment"
        );

        // Restore
        if (originalWindow) {
          (global as any).window = originalWindow;
        }
      });
    });
  });
});
