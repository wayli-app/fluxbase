/**
 * Tests for the Next.js cookie-backed StorageAdapter.
 */

import { describe, it, expect, vi } from "vitest";
import { createCookieStorage } from "./cookie-storage";

function createMockCookies() {
  const store = new Map<string, string>();
  return {
    store,
    get: vi.fn((name: string) =>
      store.has(name) ? { value: store.get(name)! } : undefined,
    ),
    set: vi.fn((name: string, value: string) => {
      store.set(name, value);
    }),
    delete: vi.fn((name: string) => {
      store.delete(name);
    }),
  };
}

describe("createCookieStorage (Next.js)", () => {
  it("reads values through the Next cookies API", () => {
    const cookies = createMockCookies();
    cookies.store.set("fluxbase.auth.session", "session-value");
    const adapter = createCookieStorage(cookies);

    expect(adapter.getItem("fluxbase.auth.session")).toBe("session-value");
    expect(cookies.get).toHaveBeenCalledWith("fluxbase.auth.session");
  });

  it("returns null for missing keys", () => {
    const cookies = createMockCookies();
    const adapter = createCookieStorage(cookies);

    expect(adapter.getItem("missing")).toBeNull();
  });

  it("writes with httpOnly + sameSite defaults", () => {
    const cookies = createMockCookies();
    const adapter = createCookieStorage(cookies);

    adapter.setItem("fluxbase.auth.session", "v");

    expect(cookies.set).toHaveBeenCalledWith(
      "fluxbase.auth.session",
      "v",
      expect.objectContaining({ path: "/", httpOnly: true, sameSite: "lax" }),
    );
  });

  it("deletes via the cookies API", () => {
    const cookies = createMockCookies();
    cookies.store.set("k", "v");
    const adapter = createCookieStorage(cookies);

    adapter.removeItem("k");

    expect(cookies.delete).toHaveBeenCalledWith("k");
    expect(cookies.store.has("k")).toBe(false);
  });

  it("round-trips as a StorageAdapter", () => {
    const cookies = createMockCookies();
    const adapter = createCookieStorage(cookies);

    adapter.setItem("a", "1");
    expect(adapter.getItem("a")).toBe("1");
    adapter.removeItem("a");
    expect(adapter.getItem("a")).toBeNull();
  });
});
