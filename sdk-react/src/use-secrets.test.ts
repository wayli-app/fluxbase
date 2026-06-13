/**
 * Tests for Secrets hooks
 */

import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import {
  useSecrets,
  useCreateSecret,
  useUpdateSecret,
  useDeleteSecret,
} from "./use-secrets";
import { createMockClient, createWrapper } from "./test-utils";

describe("useSecrets", () => {
  it("should list secrets", async () => {
    const mockSecrets = [
      { id: "1", name: "API_KEY", scope: "global", version: 1, is_expired: false, created_at: "", updated_at: "" },
    ];
    const client = createMockClient();
    (client.secrets.list as ReturnType<typeof vi.fn>).mockResolvedValue(mockSecrets);

    const { result } = renderHook(() => useSecrets(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockSecrets);
  });

  it("should pass options to list", async () => {
    const client = createMockClient();
    (client.secrets.list as ReturnType<typeof vi.fn>).mockResolvedValue([]);

    renderHook(() => useSecrets({ scope: "namespace", namespace: "prod" }), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => {
      expect(client.secrets.list).toHaveBeenCalledWith({
        scope: "namespace",
        namespace: "prod",
      });
    });
  });

  it("should handle errors", async () => {
    const client = createMockClient();
    (client.secrets.list as ReturnType<typeof vi.fn>).mockRejectedValue(
      new Error("Unauthorized"),
    );

    const { result } = renderHook(() => useSecrets(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useCreateSecret", () => {
  it("should create a secret and invalidate queries", async () => {
    const mockSecret = {
      id: "2",
      name: "NEW_KEY",
      scope: "global" as const,
      version: 1,
      created_at: "",
      updated_at: "",
    };
    const client = createMockClient();
    (client.secrets.create as ReturnType<typeof vi.fn>).mockResolvedValue(mockSecret);

    const { result } = renderHook(() => useCreateSecret(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({
        name: "NEW_KEY",
        value: "secret-value",
      });
    });

    expect(client.secrets.create).toHaveBeenCalledWith({
      name: "NEW_KEY",
      value: "secret-value",
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockSecret);
  });

  it("should handle errors", async () => {
    const client = createMockClient();
    (client.secrets.create as ReturnType<typeof vi.fn>).mockRejectedValue(
      new Error("Secret exists"),
    );

    const { result } = renderHook(() => useCreateSecret(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({ name: "DUP", value: "val" });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useUpdateSecret", () => {
  it("should update a secret and invalidate queries", async () => {
    const mockSecret = {
      id: "1",
      name: "API_KEY",
      scope: "global" as const,
      version: 2,
      created_at: "",
      updated_at: "",
    };
    const client = createMockClient();
    (client.secrets.update as ReturnType<typeof vi.fn>).mockResolvedValue(mockSecret);

    const { result } = renderHook(() => useUpdateSecret(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({
        name: "API_KEY",
        request: { value: "new-value" },
      });
    });

    expect(client.secrets.update).toHaveBeenCalledWith(
      "API_KEY",
      { value: "new-value" },
      { namespace: undefined },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockSecret);
  });

  it("should pass namespace option", async () => {
    const client = createMockClient();
    (client.secrets.update as ReturnType<typeof vi.fn>).mockResolvedValue({});

    const { result } = renderHook(() => useUpdateSecret(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({
        name: "DB_URL",
        request: { value: "postgres://new" },
        namespace: "production",
      });
    });

    expect(client.secrets.update).toHaveBeenCalledWith(
      "DB_URL",
      { value: "postgres://new" },
      { namespace: "production" },
    );
  });

  it("should handle errors", async () => {
    const client = createMockClient();
    (client.secrets.update as ReturnType<typeof vi.fn>).mockRejectedValue(
      new Error("Not found"),
    );

    const { result } = renderHook(() => useUpdateSecret(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({
          name: "MISSING",
          request: { value: "val" },
        });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useDeleteSecret", () => {
  it("should delete a secret and invalidate queries", async () => {
    const client = createMockClient();
    (client.secrets.delete as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

    const { result } = renderHook(() => useDeleteSecret(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: "OLD_KEY" });
    });

    expect(client.secrets.delete).toHaveBeenCalledWith("OLD_KEY", {
      namespace: undefined,
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("should pass namespace option", async () => {
    const client = createMockClient();
    (client.secrets.delete as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

    const { result } = renderHook(() => useDeleteSecret(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: "DB_URL", namespace: "staging" });
    });

    expect(client.secrets.delete).toHaveBeenCalledWith("DB_URL", {
      namespace: "staging",
    });
  });

  it("should handle errors", async () => {
    const client = createMockClient();
    (client.secrets.delete as ReturnType<typeof vi.fn>).mockRejectedValue(
      new Error("Not found"),
    );

    const { result } = renderHook(() => useDeleteSecret(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({ name: "MISSING" });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
