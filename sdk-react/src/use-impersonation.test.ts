/**
 * Tests for impersonation hooks
 */

import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import {
  useImpersonationSessions,
  useCurrentImpersonation,
  useImpersonateUser,
  useStopImpersonation,
} from "./use-impersonation";
import { createMockClient, createWrapper } from "./test-utils";

describe("useImpersonationSessions", () => {
  it("should list impersonation sessions", async () => {
    const mockSessions = {
      sessions: [
        {
          id: "s1",
          admin_user_id: "admin-1",
          target_user_id: "user-1",
          impersonation_type: "user" as const,
          target_role: "user",
          reason: "Support",
          started_at: "",
          ended_at: null,
          is_active: true,
          ip_address: null,
          user_agent: null,
        },
      ],
      total: 1,
    };
    const client = createMockClient({
      admin: {
        impersonation: {
          listSessions: vi.fn().mockResolvedValue(mockSessions),
        },
      } as any,
    });

    const { result } = renderHook(() => useImpersonationSessions(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockSessions);
  });

  it("should pass options to listSessions", async () => {
    const listSessions = vi.fn().mockResolvedValue({ sessions: [], total: 0 });
    const client = createMockClient({
      admin: { impersonation: { listSessions } } as any,
    });

    renderHook(() => useImpersonationSessions({ is_active: true }), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => {
      expect(listSessions).toHaveBeenCalledWith({ is_active: true });
    });
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      admin: {
        impersonation: {
          listSessions: vi.fn().mockRejectedValue(new Error("Failed")),
        },
      } as any,
    });

    const { result } = renderHook(() => useImpersonationSessions(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useCurrentImpersonation", () => {
  it("should get current impersonation session", async () => {
    const mockCurrent = {
      session: {
        id: "s1",
        admin_user_id: "admin-1",
        target_user_id: "user-1",
        impersonation_type: "user" as const,
        target_role: "user",
        reason: "Support",
        started_at: "",
        ended_at: null,
        is_active: true,
        ip_address: null,
        user_agent: null,
      },
      target_user: { id: "user-1", email: "test@example.com", role: "user" },
    };
    const client = createMockClient({
      admin: {
        impersonation: {
          getCurrent: vi.fn().mockResolvedValue(mockCurrent),
        },
      } as any,
    });

    const { result } = renderHook(() => useCurrentImpersonation(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockCurrent);
  });

  it("should return null session when not impersonating", async () => {
    const client = createMockClient({
      admin: {
        impersonation: {
          getCurrent: vi.fn().mockResolvedValue({
            session: null,
            target_user: null,
          }),
        },
      } as any,
    });

    const { result } = renderHook(() => useCurrentImpersonation(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data?.session).toBeNull();
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      admin: {
        impersonation: {
          getCurrent: vi.fn().mockRejectedValue(new Error("Failed")),
        },
      } as any,
    });

    const { result } = renderHook(() => useCurrentImpersonation(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useImpersonateUser", () => {
  it("should impersonate a user", async () => {
    const mockResponse = {
      session: {
        id: "s1",
        admin_user_id: "admin-1",
        target_user_id: "user-1",
        impersonation_type: "user" as const,
        target_role: "user",
        reason: "Support",
        started_at: "",
        ended_at: null,
        is_active: true,
        ip_address: null,
        user_agent: null,
      },
      target_user: { id: "user-1", email: "test@example.com", role: "user" },
      access_token: "token-123",
      refresh_token: "refresh-123",
      expires_in: 3600,
    };
    const impersonateUser = vi.fn().mockResolvedValue(mockResponse);
    const client = createMockClient({
      admin: { impersonation: { impersonateUser } } as any,
    });

    const { result } = renderHook(() => useImpersonateUser(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({
        target_user_id: "user-1",
        reason: "Support ticket #1234",
      });
    });

    expect(impersonateUser).toHaveBeenCalledWith({
      target_user_id: "user-1",
      reason: "Support ticket #1234",
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockResponse);
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      admin: {
        impersonation: {
          impersonateUser: vi.fn().mockRejectedValue(new Error("Not allowed")),
        },
      } as any,
    });

    const { result } = renderHook(() => useImpersonateUser(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({
          target_user_id: "user-1",
          reason: "test",
        });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useStopImpersonation", () => {
  it("should stop impersonation", async () => {
    const mockResponse = { success: true, message: "Impersonation stopped" };
    const stop = vi.fn().mockResolvedValue(mockResponse);
    const client = createMockClient({
      admin: { impersonation: { stop } } as any,
    });

    const { result } = renderHook(() => useStopImpersonation(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync();
    });

    expect(stop).toHaveBeenCalled();
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockResponse);
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      admin: {
        impersonation: {
          stop: vi.fn().mockRejectedValue(new Error("No active session")),
        },
      } as any,
    });

    const { result } = renderHook(() => useStopImpersonation(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync();
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
