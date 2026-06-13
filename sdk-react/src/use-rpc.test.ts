/**
 * Tests for RPC hooks
 */

import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { useRPCList, useInvokeRPC } from "./use-rpc";
import { createMockClient, createWrapper } from "./test-utils";
import type { FluxbaseClient } from "@nimbleflux/fluxbase-sdk";

describe("useRPCList", () => {
  it("should list RPC procedures", async () => {
    const mockProcedures = [
      { id: "1", name: "get-users", namespace: "default", enabled: true, version: 1, source: "filesystem", is_public: true, allowed_tables: [], allowed_schemas: [], max_execution_time_seconds: 30, created_at: "", updated_at: "" },
    ];
    const client = createMockClient({
      rpc: Object.assign(vi.fn(), {
        list: vi.fn().mockResolvedValue({ data: mockProcedures, error: null }),
      }) as any,
    });

    const { result } = renderHook(() => useRPCList(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockProcedures);
  });

  it("should filter by namespace", async () => {
    const list = vi
      .fn()
      .mockResolvedValue({ data: [], error: null });
    const client = createMockClient({
      rpc: Object.assign(vi.fn(), { list }) as any,
    });

    renderHook(() => useRPCList("custom-ns"), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => {
      expect(list).toHaveBeenCalledWith("custom-ns");
    });
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      rpc: Object.assign(vi.fn(), {
        list: vi
          .fn()
          .mockResolvedValue({ data: null, error: new Error("Failed") }),
      }) as any,
    });

    const { result } = renderHook(() => useRPCList(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useInvokeRPC", () => {
  it("should invoke an RPC procedure", async () => {
    const mockResponse = {
      execution_id: "exec-1",
      status: "completed" as const,
      result: { count: 42 },
    };
    const client = createMockClient({
      rpc: Object.assign(vi.fn(), {
        invoke: vi.fn().mockResolvedValue({ data: mockResponse, error: null }),
      }) as any,
    });

    const { result } = renderHook(() => useInvokeRPC(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({
        name: "get-count",
        payload: { table: "users" },
      });
    });

    expect((client.rpc as any).invoke).toHaveBeenCalledWith(
      "get-count",
      { table: "users" },
      undefined,
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockResponse);
  });

  it("should pass options", async () => {
    const mockResponse = { execution_id: "exec-2", status: "running" as const };
    const client = createMockClient({
      rpc: Object.assign(vi.fn(), {
        invoke: vi.fn().mockResolvedValue({ data: mockResponse, error: null }),
      }) as any,
    });

    const { result } = renderHook(() => useInvokeRPC(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({
        name: "long-task",
        options: { async: true, namespace: "prod" },
      });
    });

    expect((client.rpc as any).invoke).toHaveBeenCalledWith(
      "long-task",
      undefined,
      { async: true, namespace: "prod" },
    );
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      rpc: Object.assign(vi.fn(), {
        invoke: vi
          .fn()
          .mockResolvedValue({ data: null, error: new Error("Not found") }),
      }) as any,
    });

    const { result } = renderHook(() => useInvokeRPC(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({ name: "bad-proc" });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
