/**
 * Tests for the feature store factories (functions, jobs, branches, rpc).
 *
 * svelte-query stores are unwrapped with `get()` from `svelte/store`.
 */

import { describe, it, expect, vi } from "vitest";
import { get } from "svelte/store";
import {
  createInvokeFunction,
  createSubmitJob,
  createBranches,
  createCreateBranch,
  createInvokeRPC,
} from "./features";
import {
  createMockClient,
  createTestQueryClient,
} from "../test-utils";

describe("createInvokeFunction", () => {
  it("invokes the named function", async () => {
    const client = createMockClient({
      functions: {
        invoke: vi.fn().mockResolvedValue({ data: { ok: true }, error: null }),
      } as any,
    });
    const queryClient = createTestQueryClient();

    const mutation = createInvokeFunction({ client, queryClient });
    await get(mutation).mutateAsync({ name: "hello", options: { body: { a: 1 } } });

    expect(client.functions.invoke).toHaveBeenCalledWith("hello", {
      body: { a: 1 },
    });
  });

  it("throws on error", async () => {
    const client = createMockClient({
      functions: {
        invoke: vi
          .fn()
          .mockResolvedValue({ data: null, error: new Error("fn failed") }),
      } as any,
    });
    const queryClient = createTestQueryClient();

    const mutation = createInvokeFunction({ client, queryClient });
    await expect(get(mutation).mutateAsync({ name: "bad" })).rejects.toThrow(
      "fn failed",
    );
  });
});

describe("createSubmitJob", () => {
  it("submits and refreshes the jobs list", async () => {
    const client = createMockClient({
      jobs: {
        submit: vi.fn().mockResolvedValue({
          data: { id: "j1", status: "pending" },
          error: null,
        }),
      } as any,
    });
    const queryClient = createTestQueryClient();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const mutation = createSubmitJob({ client, queryClient });
    await get(mutation).mutateAsync({ function: "work" } as any);

    expect(client.jobs.submit).toHaveBeenCalled();
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["fluxbase", "jobs"],
    });
  });
});

describe("createBranches", () => {
  it("lists branches reactively", async () => {
    const data = { branches: [], total: 0, limit: 50, offset: 0 };
    const client = createMockClient({
      branching: {
        list: vi.fn().mockResolvedValue({ data, error: null }),
      } as any,
    });
    const queryClient = createTestQueryClient();

    const result = await queryClient.fetchQuery({
      queryKey: ["fluxbase", "branches"],
      queryFn: async () => {
        const { data, error } = await client.branching.list();
        if (error) throw error;
        return data;
      },
    });

    expect(result).toEqual(data);
  });
});

describe("createCreateBranch", () => {
  it("creates a branch and invalidates the list", async () => {
    const client = createMockClient({
      branching: {
        create: vi.fn().mockResolvedValue({
          data: { id: "b1", slug: "x", status: "creating" },
          error: null,
        }),
      } as any,
    });
    const queryClient = createTestQueryClient();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const mutation = createCreateBranch({ client, queryClient });
    await get(mutation).mutateAsync({ slug: "x" } as any);

    expect(client.branching.create).toHaveBeenCalled();
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["fluxbase", "branches"],
    });
  });
});

describe("createInvokeRPC", () => {
  it("invokes an RPC with params", async () => {
    const invokeMock = vi
      .fn()
      .mockResolvedValue({ data: { result: 42 }, error: null });
    const client = createMockClient({
      rpc: Object.assign(vi.fn(), {
        list: vi.fn().mockResolvedValue({ data: [], error: null }),
        invoke: invokeMock,
      }) as any,
    });
    const queryClient = createTestQueryClient();

    const mutation = createInvokeRPC({ client, queryClient });
    await get(mutation).mutateAsync({ name: "calc", params: { x: 1 } });

    expect(invokeMock).toHaveBeenCalledWith("calc", { x: 1 }, undefined);
  });
});
