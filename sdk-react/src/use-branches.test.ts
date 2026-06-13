/**
 * Tests for Branches hooks
 */

import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import {
  useBranches,
  useCreateBranch,
  useDeleteBranch,
  useResetBranch,
} from "./use-branches";
import { createMockClient, createWrapper } from "./test-utils";

describe("useBranches", () => {
  it("should list branches", async () => {
    const mockData = {
      branches: [
        { id: "b1", slug: "main", status: "ready", type: "main" },
        { id: "b2", slug: "feature", status: "ready", type: "preview" },
      ],
      total: 2,
      limit: 50,
      offset: 0,
    };
    const client = createMockClient({
      branching: {
        list: vi.fn().mockResolvedValue({ data: mockData, error: null }),
      } as any,
    });

    const { result } = renderHook(() => useBranches(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockData);
  });

  it("should pass options to list", async () => {
    const list = vi.fn().mockResolvedValue({
      data: { branches: [], total: 0, limit: 50, offset: 0 },
      error: null,
    });
    const client = createMockClient({
      branching: { list } as any,
    });

    renderHook(() => useBranches({ status: "ready", limit: 10 }), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => {
      expect(list).toHaveBeenCalledWith({ status: "ready", limit: 10 });
    });
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      branching: {
        list: vi.fn().mockResolvedValue({ data: null, error: new Error("Failed") }),
      } as any,
    });

    const { result } = renderHook(() => useBranches(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useCreateBranch", () => {
  it("should create a branch and invalidate queries", async () => {
    const create = vi.fn().mockResolvedValue({
      data: { id: "b2", slug: "feature-x", status: "creating" },
      error: null,
    });
    const client = createMockClient({
      branching: { create } as any,
    });

    const { result } = renderHook(() => useCreateBranch(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: "feature/x" });
    });

    expect(create).toHaveBeenCalledWith("feature/x", {});
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("should pass options to create", async () => {
    const create = vi.fn().mockResolvedValue({
      data: { id: "b2", slug: "feature-x", status: "creating" },
      error: null,
    });
    const client = createMockClient({
      branching: { create } as any,
    });

    const { result } = renderHook(() => useCreateBranch(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({
        name: "feature/x",
        dataCloneMode: "schema_only",
        expiresIn: "7d",
      });
    });

    expect(create).toHaveBeenCalledWith("feature/x", {
      dataCloneMode: "schema_only",
      expiresIn: "7d",
    });
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      branching: {
        create: vi.fn().mockResolvedValue({ data: null, error: new Error("Failed") }),
      } as any,
    });

    const { result } = renderHook(() => useCreateBranch(), {
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

describe("useDeleteBranch", () => {
  it("should delete a branch and invalidate queries", async () => {
    const del = vi.fn().mockResolvedValue({ error: null });
    const client = createMockClient({
      branching: { delete: del } as any,
    });

    const { result } = renderHook(() => useDeleteBranch(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync("feature/x");
    });

    expect(del).toHaveBeenCalledWith("feature/x");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      branching: {
        delete: vi.fn().mockResolvedValue({ error: new Error("Not found") }),
      } as any,
    });

    const { result } = renderHook(() => useDeleteBranch(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync("feature/x");
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useResetBranch", () => {
  it("should reset a branch and invalidate queries", async () => {
    const reset = vi.fn().mockResolvedValue({
      data: { id: "b2", slug: "feature-x", status: "creating" },
      error: null,
    });
    const client = createMockClient({
      branching: { reset } as any,
    });

    const { result } = renderHook(() => useResetBranch(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync("feature/x");
    });

    expect(reset).toHaveBeenCalledWith("feature/x");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      branching: {
        reset: vi.fn().mockResolvedValue({ data: null, error: new Error("Failed") }),
      } as any,
    });

    const { result } = renderHook(() => useResetBranch(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync("feature/x");
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
