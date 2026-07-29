/**
 * Tests for the cookie-backed StorageAdapter.
 *
 * These are pure unit tests — no Svelte component context required — because
 * createCookieStorage just wraps a cookie API object.
 */

import { describe, it, expect, vi } from "vitest";
import { createCookieStorage } from "./cookie-storage";

/** Minimal in-memory mock of SvelteKit's `Cookies` API. */
function createMockCookies() {
  const store = new Map<string, { value: string; opts: any }>();
  return {
    store,
    get: vi.fn((name: string) => store.get(name)?.value),
    set: vi.fn((name: string, value: string, opts?: any) => {
      store.set(name, { value, opts });
    }),
    delete: vi.fn((name: string, opts?: any) => {
      store.delete(name);
    }),
  };
}

describe("createCookieStorage", () => {
  it("reads values through the cookie API", () => {
    const cookies = createMockCookies();
    cookies.set("fluxbase.auth.session", "session-value");
    const adapter = createCookieStorage(cookies);

    expect(adapter.getItem("fluxbase.auth.session")).toBe("session-value");
    expect(cookies.get).toHaveBeenCalledWith("fluxbase.auth.session");
  });

  it("returns null for missing keys", () => {
    const cookies = createMockCookies();
    const adapter = createCookieStorage(cookies);

    expect(adapter.getItem("missing")).toBeNull();
  });

  it("writes values with httpOnly + sameSite defaults", () => {
    const cookies = createMockCookies();
    const adapter = createCookieStorage(cookies);

    adapter.setItem("fluxbase.auth.session", "new-value");

    expect(cookies.set).toHaveBeenCalledWith(
      "fluxbase.auth.session",
      "new-value",
      expect.objectContaining({
        path: "/",
        httpOnly: true,
        sameSite: "lax",
      }),
    );
    expect(cookies.store.get("fluxbase.auth.session")?.value).toBe("new-value");
  });

  it("deletes values via the cookie API with the configured path", () => {
    const cookies = createMockCookies();
    cookies.set("fluxbase.auth.session", "value");
    const adapter = createCookieStorage(cookies);

    adapter.removeItem("fluxbase.auth.session");

    expect(cookies.delete).toHaveBeenCalledWith("fluxbase.auth.session", {
      path: "/",
    });
    expect(cookies.store.has("fluxbase.auth.session")).toBe(false);
  });

  it("honors custom cookie options", () => {
    const cookies = createMockCookies();
    const adapter = createCookieStorage(cookies, {
      path: "/app",
      httpOnly: false,
      sameSite: "strict",
      maxAge: 3600,
    });

    adapter.setItem("k", "v");

    expect(cookies.set).toHaveBeenCalledWith(
      "k",
      "v",
      expect.objectContaining({
        path: "/app",
        httpOnly: false,
        sameSite: "strict",
        maxAge: 3600,
      }),
    );
  });

  it("is a valid StorageAdapter (get/set/remove round-trip)", () => {
    const cookies = createMockCookies();
    const adapter = createCookieStorage(cookies);

    adapter.setItem("a", "1");
    expect(adapter.getItem("a")).toBe("1");
    adapter.removeItem("a");
    expect(adapter.getItem("a")).toBeNull();
  });
});
