/**
 * Tests for migration hooks
 */

import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import {
  useMigrations,
  useApplyMigration,
  useRollbackMigration,
  useSyncMigrations,
} from "./use-migrations";
import { createMockClient, createWrapper } from "./test-utils";

describe("useMigrations", () => {
  it("should list migrations", async () => {
    const mockMigrations = [
      {
        id: "1",
        namespace: "default",
        name: "001_init",
        up_sql: "CREATE TABLE test();",
        version: 1,
        status: "applied" as const,
        created_at: "",
        updated_at: "",
      },
    ];
    const client = createMockClient({
      admin: {
        migrations: {
          list: vi.fn().mockResolvedValue({ data: mockMigrations, error: null }),
        },
      } as any,
    });

    const { result } = renderHook(() => useMigrations(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockMigrations);
  });

  it("should pass namespace to list", async () => {
    const list = vi.fn().mockResolvedValue({ data: [], error: null });
    const client = createMockClient({
      admin: { migrations: { list } } as any,
    });

    renderHook(() => useMigrations("myapp"), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => {
      expect(list).toHaveBeenCalledWith("myapp");
    });
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      admin: {
        migrations: {
          list: vi
            .fn()
            .mockResolvedValue({ data: null, error: new Error("Failed") }),
        },
      } as any,
    });

    const { result } = renderHook(() => useMigrations(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useApplyMigration", () => {
  it("should apply a migration", async () => {
    const apply = vi
      .fn()
      .mockResolvedValue({ data: { message: "Applied" }, error: null });
    const client = createMockClient({
      admin: { migrations: { apply } } as any,
    });

    const { result } = renderHook(() => useApplyMigration(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: "001_init" });
    });

    expect(apply).toHaveBeenCalledWith("001_init", undefined);
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("should pass namespace to apply", async () => {
    const apply = vi
      .fn()
      .mockResolvedValue({ data: { message: "Applied" }, error: null });
    const client = createMockClient({
      admin: { migrations: { apply } } as any,
    });

    const { result } = renderHook(() => useApplyMigration(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: "001_init", namespace: "myapp" });
    });

    expect(apply).toHaveBeenCalledWith("001_init", "myapp");
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      admin: {
        migrations: {
          apply: vi
            .fn()
            .mockResolvedValue({ data: null, error: new Error("Failed") }),
        },
      } as any,
    });

    const { result } = renderHook(() => useApplyMigration(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({ name: "001_init" });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useRollbackMigration", () => {
  it("should rollback a migration", async () => {
    const rollback = vi
      .fn()
      .mockResolvedValue({ data: { message: "Rolled back" }, error: null });
    const client = createMockClient({
      admin: { migrations: { rollback } } as any,
    });

    const { result } = renderHook(() => useRollbackMigration(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: "001_init" });
    });

    expect(rollback).toHaveBeenCalledWith("001_init", undefined);
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      admin: {
        migrations: {
          rollback: vi
            .fn()
            .mockResolvedValue({ data: null, error: new Error("Failed") }),
        },
      } as any,
    });

    const { result } = renderHook(() => useRollbackMigration(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({ name: "001_init" });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useSyncMigrations", () => {
  it("should sync migrations", async () => {
    const mockResult = {
      message: "Synced",
      namespace: "default",
      summary: {
        created: 1,
        updated: 0,
        unchanged: 0,
        skipped: 0,
        applied: 0,
        errors: 0,
      },
      details: {
        created: ["001_init"],
        updated: [],
        unchanged: [],
        skipped: [],
        applied: [],
        errors: [],
      },
      dry_run: false,
    };
    const sync = vi.fn().mockResolvedValue({ data: mockResult, error: null });
    const client = createMockClient({
      admin: { migrations: { sync } } as any,
    });

    const { result } = renderHook(() => useSyncMigrations(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync();
    });

    expect(sync).toHaveBeenCalledWith(undefined);
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockResult);
  });

  it("should pass options to sync", async () => {
    const sync = vi
      .fn()
      .mockResolvedValue({ data: null, error: null });
    const client = createMockClient({
      admin: { migrations: { sync } } as any,
    });

    const { result } = renderHook(() => useSyncMigrations(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ auto_apply: true });
    });

    expect(sync).toHaveBeenCalledWith({ auto_apply: true });
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      admin: {
        migrations: {
          sync: vi
            .fn()
            .mockResolvedValue({ data: null, error: new Error("Sync failed") }),
        },
      } as any,
    });

    const { result } = renderHook(() => useSyncMigrations(), {
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
