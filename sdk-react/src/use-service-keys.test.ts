/**
 * Tests for Service Keys hooks
 */

import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import {
  useServiceKeys,
  useCreateServiceKey,
  useRotateServiceKey,
  useRevokeServiceKey,
} from "./use-service-keys";
import { createMockClient, createWrapper } from "./test-utils";

describe("useServiceKeys", () => {
  it("should list service keys", async () => {
    const mockKeys = [
      {
        id: "1",
        name: "Production Key",
        key_type: "service" as const,
        scopes: ["*"],
        enabled: true,
        key_prefix: "fb_prod_abcdef",
        created_at: "",
      },
    ];
    const client = createMockClient();
    (
      client.admin.serviceKeys.list as ReturnType<typeof vi.fn>
    ).mockResolvedValue({ data: mockKeys, error: null });

    const { result } = renderHook(() => useServiceKeys(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockKeys);
  });

  it("should handle errors", async () => {
    const client = createMockClient();
    (
      client.admin.serviceKeys.list as ReturnType<typeof vi.fn>
    ).mockResolvedValue({ data: null, error: new Error("Unauthorized") });

    const { result } = renderHook(() => useServiceKeys(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useCreateServiceKey", () => {
  it("should create a service key and invalidate queries", async () => {
    const mockKey = {
      id: "2",
      name: "New Key",
      key_type: "service" as const,
      scopes: ["*"],
      enabled: true,
      key_prefix: "fb_new_123456",
      key: "fb_new_123456789",
      created_at: "",
    };
    const client = createMockClient();
    (
      client.admin.serviceKeys.create as ReturnType<typeof vi.fn>
    ).mockResolvedValue({ data: mockKey, error: null });

    const { result } = renderHook(() => useCreateServiceKey(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({
        name: "New Key",
        key_type: "service",
        scopes: ["*"],
      });
    });

    expect(client.admin.serviceKeys.create).toHaveBeenCalledWith({
      name: "New Key",
      key_type: "service",
      scopes: ["*"],
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockKey);
  });

  it("should handle errors", async () => {
    const client = createMockClient();
    (
      client.admin.serviceKeys.create as ReturnType<typeof vi.fn>
    ).mockResolvedValue({ data: null, error: new Error("Limit reached") });

    const { result } = renderHook(() => useCreateServiceKey(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({
          name: "Excess",
          key_type: "service",
        });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useRotateServiceKey", () => {
  it("should rotate a service key and invalidate queries", async () => {
    const mockKey = {
      id: "3",
      name: "Rotated Key",
      key_type: "service" as const,
      scopes: ["*"],
      enabled: true,
      key_prefix: "fb_rot_newpass",
      key: "fb_rot_newpassword",
      created_at: "",
      deprecated_at: "",
      grace_period_ends_at: "",
    };
    const client = createMockClient();
    (
      client.admin.serviceKeys.rotate as ReturnType<typeof vi.fn>
    ).mockResolvedValue({ data: mockKey, error: null });

    const { result } = renderHook(() => useRotateServiceKey(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync("old-key-id");
    });

    expect(client.admin.serviceKeys.rotate).toHaveBeenCalledWith("old-key-id");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockKey);
  });

  it("should handle errors", async () => {
    const client = createMockClient();
    (
      client.admin.serviceKeys.rotate as ReturnType<typeof vi.fn>
    ).mockResolvedValue({ data: null, error: new Error("Key not found") });

    const { result } = renderHook(() => useRotateServiceKey(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync("missing-id");
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useRevokeServiceKey", () => {
  it("should revoke a service key and invalidate queries", async () => {
    const client = createMockClient();
    (
      client.admin.serviceKeys.revoke as ReturnType<typeof vi.fn>
    ).mockResolvedValue({ error: null });

    const { result } = renderHook(() => useRevokeServiceKey(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ id: "compromised-id" });
    });

    expect(client.admin.serviceKeys.revoke).toHaveBeenCalledWith(
      "compromised-id",
      undefined,
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("should pass revocation reason", async () => {
    const client = createMockClient();
    (
      client.admin.serviceKeys.revoke as ReturnType<typeof vi.fn>
    ).mockResolvedValue({ error: null });

    const { result } = renderHook(() => useRevokeServiceKey(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({
        id: "key-id",
        request: { reason: "Key was compromised" },
      });
    });

    expect(client.admin.serviceKeys.revoke).toHaveBeenCalledWith("key-id", {
      reason: "Key was compromised",
    });
  });

  it("should handle errors", async () => {
    const client = createMockClient();
    (
      client.admin.serviceKeys.revoke as ReturnType<typeof vi.fn>
    ).mockResolvedValue({ error: new Error("Already revoked") });

    const { result } = renderHook(() => useRevokeServiceKey(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({ id: "bad-id" });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
