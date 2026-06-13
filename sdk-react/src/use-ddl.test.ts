/**
 * Tests for DDL hooks
 */

import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import {
  useSchemas,
  useTables,
  useCreateSchema,
  useDeleteTable,
} from "./use-ddl";
import { createMockClient, createWrapper } from "./test-utils";

describe("useSchemas", () => {
  it("should list schemas", async () => {
    const mockSchemas = {
      schemas: [
        { name: "public", owner: "postgres" },
        { name: "analytics", owner: "postgres" },
      ],
    };
    const client = createMockClient({
      admin: {
        ddl: {
          listSchemas: vi.fn().mockResolvedValue(mockSchemas),
        },
      } as any,
    });

    const { result } = renderHook(() => useSchemas(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockSchemas);
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      admin: {
        ddl: {
          listSchemas: vi.fn().mockRejectedValue(new Error("Failed")),
        },
      } as any,
    });

    const { result } = renderHook(() => useSchemas(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useTables", () => {
  it("should list all tables when no schema provided", async () => {
    const mockTables = {
      tables: [
        { schema: "public", name: "users" },
        { schema: "analytics", name: "events" },
      ],
    };
    const listTables = vi.fn().mockResolvedValue(mockTables);
    const client = createMockClient({
      admin: { ddl: { listTables } } as any,
    });

    const { result } = renderHook(() => useTables(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockTables);
    expect(listTables).toHaveBeenCalledWith(undefined);
  });

  it("should list tables filtered by schema", async () => {
    const mockTables = {
      tables: [{ schema: "public", name: "users" }],
    };
    const listTables = vi.fn().mockResolvedValue(mockTables);
    const client = createMockClient({
      admin: { ddl: { listTables } } as any,
    });

    const { result } = renderHook(() => useTables("public"), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(listTables).toHaveBeenCalledWith("public");
    expect(result.current.data).toEqual(mockTables);
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      admin: {
        ddl: {
          listTables: vi.fn().mockRejectedValue(new Error("Failed")),
        },
      } as any,
    });

    const { result } = renderHook(() => useTables(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useCreateSchema", () => {
  it("should create a schema", async () => {
    const mockResponse = { message: "Schema created", schema: "analytics" };
    const createSchema = vi.fn().mockResolvedValue(mockResponse);
    const client = createMockClient({
      admin: { ddl: { createSchema } } as any,
    });

    const { result } = renderHook(() => useCreateSchema(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync("analytics");
    });

    expect(createSchema).toHaveBeenCalledWith("analytics");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockResponse);
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      admin: {
        ddl: {
          createSchema: vi.fn().mockRejectedValue(new Error("Schema exists")),
        },
      } as any,
    });

    const { result } = renderHook(() => useCreateSchema(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync("analytics");
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useDeleteTable", () => {
  it("should delete a table", async () => {
    const mockResponse = { message: "Table deleted" };
    const deleteTable = vi.fn().mockResolvedValue(mockResponse);
    const client = createMockClient({
      admin: { ddl: { deleteTable } } as any,
    });

    const { result } = renderHook(() => useDeleteTable(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ schema: "public", name: "old_data" });
    });

    expect(deleteTable).toHaveBeenCalledWith("public", "old_data");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockResponse);
  });

  it("should handle errors", async () => {
    const client = createMockClient({
      admin: {
        ddl: {
          deleteTable: vi.fn().mockRejectedValue(new Error("Table not found")),
        },
      } as any,
    });

    const { result } = renderHook(() => useDeleteTable(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({ schema: "public", name: "missing" });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
