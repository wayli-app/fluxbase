/**
 * Tests for Functions hooks
 */

import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { useInvokeFunction, useFunctions } from "./use-functions";
import { createMockClient, createWrapper } from "./test-utils";

describe("useInvokeFunction", () => {
  it("should invoke a function", async () => {
    const invoke = vi.fn().mockResolvedValue({
      data: { result: "ok" },
      error: null,
    });
    const client = createMockClient({
      functions: { invoke } as any,
    });

    const { result } = renderHook(() => useInvokeFunction(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: "hello", payload: { name: "World" } });
    });

    expect(invoke).toHaveBeenCalledWith("hello", { body: { name: "World" }, method: undefined });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      functions: {
        invoke: vi.fn().mockResolvedValue({ data: null, error: new Error("Failed") }),
      } as any,
    });

    const { result } = renderHook(() => useInvokeFunction(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({ name: "bad" });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useFunctions", () => {
  it("should list functions", async () => {
    const mockFunctions = [
      { name: "hello", version: "1", namespace: "default" },
      { name: "world", version: "2", namespace: "default" },
    ];
    const client = createMockClient({
      functions: {
        list: vi.fn().mockResolvedValue({ data: mockFunctions, error: null }),
      } as any,
    });

    const { result } = renderHook(() => useFunctions(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockFunctions);
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      functions: {
        list: vi.fn().mockResolvedValue({ data: null, error: new Error("Failed") }),
      } as any,
    });

    const { result } = renderHook(() => useFunctions(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});
