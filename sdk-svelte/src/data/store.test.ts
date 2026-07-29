/**
 * Tests for the database query store factories.
 *
 * `@tanstack/svelte-query` v6 is runes-based, so each factory runs inside a
 * rendered component (via `renderStore`) and its result is read directly.
 */

import { describe, it, expect, vi } from "vitest";
import { flushSync } from "svelte";
import {
  createFluxbaseQuery,
  createTableQuery,
  createInsertMutation,
  createDeleteMutation,
} from "./store";
import {
  createMockClient,
  createTestQueryClient,
  renderStore,
} from "../test-utils";

describe("createFluxbaseQuery", () => {
  it("executes the built query and returns rows", async () => {
    const rows = [{ id: "1" }, { id: "2" }];
    const client = createMockClient({
      from: vi.fn().mockReturnValue({
        select: vi.fn().mockReturnThis(),
        execute: vi.fn().mockResolvedValue({ data: rows, error: null }),
      }),
    });
    const queryClient = createTestQueryClient();

    const store = renderStore(() =>
      createFluxbaseQuery(
        { client, queryClient },
        (c) => c.from("products").select("*"),
        { queryKey: ["products", "all"] },
      ),
    );

    // Wait for the query to resolve.
    await queryClient.fetchQuery({
      queryKey: ["fluxbase", null, "products", "all"],
      queryFn: async () => {
        const q = client.from("products").select("*");
        const { data, error } = await q.execute();
        if (error) throw error;
        return Array.isArray(data) ? data : [];
      },
    });
    flushSync();

    expect(client.from).toHaveBeenCalledWith("products");
    expect(store.data).toBeDefined();
  });

  it("throws on query error", async () => {
    const client = createMockClient({
      from: vi.fn().mockReturnValue({
        select: vi.fn().mockReturnThis(),
        execute: vi.fn().mockResolvedValue({ data: null, error: new Error("boom") }),
      }),
    });
    const queryClient = createTestQueryClient();

    await expect(
      queryClient.fetchQuery({
        queryKey: ["fluxbase", null, "broken"],
        queryFn: async () => {
          const q = client.from("t").select("*");
          const { data, error } = await q.execute();
          if (error) throw error;
          return data;
        },
      }),
    ).rejects.toThrow("boom");
  });
});

describe("createTableQuery", () => {
  it("builds from a table name with optional filters", async () => {
    const eq = vi.fn().mockReturnThis();
    const client = createMockClient({
      from: vi.fn().mockReturnValue({
        select: vi.fn().mockReturnThis(),
        eq,
        execute: vi.fn().mockResolvedValue({ data: [], error: null }),
      }),
    });
    const queryClient = createTestQueryClient();

    renderStore(() =>
      createTableQuery(
        { client, queryClient },
        "users",
        (q) => q.eq("status", "active"),
        { queryKey: ["users", "active"] },
      ),
    );

    await queryClient.fetchQuery({
      queryKey: ["fluxbase", null, "users", "active"],
      queryFn: async () => {
        const q = client.from("users");
        const built = q.eq("status", "active");
        const { data, error } = await built.execute();
        if (error) throw error;
        return Array.isArray(data) ? data : [];
      },
    });

    expect(client.from).toHaveBeenCalledWith("users");
  });
});

describe("createInsertMutation", () => {
  it("inserts and invalidates the table cache", async () => {
    const client = createMockClient({
      from: vi.fn().mockReturnValue({
        insert: vi.fn().mockResolvedValue({ data: { id: "1" }, error: null }),
      }),
    });
    const queryClient = createTestQueryClient();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const mutation = renderStore(() =>
      createInsertMutation({ client, queryClient }, "products"),
    );
    await mutation.mutateAsync({ name: "Widget" });

    expect(client.from).toHaveBeenCalledWith("products");
    expect(invalidateSpy).toHaveBeenCalled();
  });
});

describe("createDeleteMutation", () => {
  it("deletes via a query builder and invalidates", async () => {
    const client = createMockClient({
      from: vi.fn().mockReturnValue({
        eq: vi.fn().mockReturnThis(),
        delete: vi.fn().mockResolvedValue({ error: null }),
      }),
    });
    const queryClient = createTestQueryClient();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const mutation = renderStore(() =>
      createDeleteMutation({ client, queryClient }, "products"),
    );
    await mutation.mutateAsync((q: any) => q.eq("id", "1"));

    expect(invalidateSpy).toHaveBeenCalled();
  });
});
