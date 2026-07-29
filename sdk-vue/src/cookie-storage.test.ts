/**
 * Tests for the Vue/Nuxt cookie-backed StorageAdapter.
 */

import { describe, it, expect, vi } from "vitest";
import { createCookieStorage } from "./cookie-storage";

function createMockEvent() {
  const store = new Map<string, string>();
  return {
    store,
    getCookie: vi.fn((name: string) => store.get(name)),
    setCookie: vi.fn((name: string, value: string) => {
      store.set(name, value);
    }),
    deleteCookie: vi.fn((name: string) => {
      store.delete(name);
    }),
  };
}

describe("createCookieStorage (Vue/Nuxt)", () => {
  it("reads values through the event cookie API", () => {
    const event = createMockEvent();
    event.store.set("fluxbase.auth.session", "session-value");
    const adapter = createCookieStorage(event);

    expect(adapter.getItem("fluxbase.auth.session")).toBe("session-value");
    expect(event.getCookie).toHaveBeenCalledWith("fluxbase.auth.session");
  });

  it("returns null for missing keys", () => {
    const event = createMockEvent();
    const adapter = createCookieStorage(event);

    expect(adapter.getItem("missing")).toBeNull();
  });

  it("writes with httpOnly + sameSite defaults", () => {
    const event = createMockEvent();
    const adapter = createCookieStorage(event);

    adapter.setItem("fluxbase.auth.session", "v");

    expect(event.setCookie).toHaveBeenCalledWith(
      "fluxbase.auth.session",
      "v",
      expect.objectContaining({ path: "/", httpOnly: true, sameSite: "lax" }),
    );
  });

  it("deletes via the event cookie API with the configured path", () => {
    const event = createMockEvent();
    event.store.set("k", "v");
    const adapter = createCookieStorage(event);

    adapter.removeItem("k");

    expect(event.deleteCookie).toHaveBeenCalledWith("k", { path: "/" });
    expect(event.store.has("k")).toBe(false);
  });

  it("round-trips as a StorageAdapter", () => {
    const event = createMockEvent();
    const adapter = createCookieStorage(event);

    adapter.setItem("a", "1");
    expect(adapter.getItem("a")).toBe("1");
    adapter.removeItem("a");
    expect(adapter.getItem("a")).toBeNull();
  });
});
