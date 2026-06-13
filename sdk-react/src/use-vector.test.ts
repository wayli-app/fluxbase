/**
 * Tests for Vector hooks
 */

import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { useVectorEmbed, useVectorSearch } from "./use-vector";
import { createMockClient, createWrapper } from "./test-utils";

describe("useVectorEmbed", () => {
  it("should embed text", async () => {
    const mockResponse = {
      embeddings: [[0.1, 0.2, 0.3]],
      model: "text-embedding-3-small",
      dimensions: 3,
    };
    const client = createMockClient();
    (client.vector.embed as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: mockResponse,
      error: null,
    });

    const { result } = renderHook(() => useVectorEmbed(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ text: "Hello world" });
    });

    expect(client.vector.embed).toHaveBeenCalledWith({ text: "Hello world" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockResponse);
  });

  it("should embed multiple texts", async () => {
    const mockResponse = {
      embeddings: [[0.1], [0.2]],
      model: "text-embedding-3-small",
      dimensions: 1,
    };
    const client = createMockClient();
    (client.vector.embed as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: mockResponse,
      error: null,
    });

    const { result } = renderHook(() => useVectorEmbed(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({
        texts: ["Hello", "World"],
        model: "text-embedding-3-large",
      });
    });

    expect(client.vector.embed).toHaveBeenCalledWith({
      texts: ["Hello", "World"],
      model: "text-embedding-3-large",
    });
  });

  it("should handle errors", async () => {
    const client = createMockClient();
    (client.vector.embed as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: null,
      error: new Error("Embedding failed"),
    });

    const { result } = renderHook(() => useVectorEmbed(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({ text: "test" });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useVectorSearch", () => {
  it("should search with text query", async () => {
    const mockResult = {
      data: [{ id: "1", content: "Match 1" }],
      distances: [0.1],
      model: "text-embedding-3-small",
    };
    const client = createMockClient();
    (client.vector.search as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: mockResult,
      error: null,
    });

    const { result } = renderHook(() => useVectorSearch(), {
      wrapper: createWrapper(client),
    });

    const searchOpts = {
      table: "documents",
      column: "embedding",
      query: "How to use TypeScript?",
      match_count: 10,
    };

    await act(async () => {
      await result.current.mutateAsync(searchOpts);
    });

    expect(client.vector.search).toHaveBeenCalledWith(searchOpts);
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockResult);
  });

  it("should search with pre-computed vector", async () => {
    const mockResult = {
      data: [{ id: "2", content: "Vector match" }],
      distances: [0.05],
    };
    const client = createMockClient();
    (client.vector.search as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: mockResult,
      error: null,
    });

    const { result } = renderHook(() => useVectorSearch(), {
      wrapper: createWrapper(client),
    });

    const searchOpts = {
      table: "documents",
      column: "embedding",
      vector: [0.1, 0.2, 0.3],
      metric: "cosine" as const,
      match_count: 5,
    };

    await act(async () => {
      await result.current.mutateAsync(searchOpts);
    });

    expect(client.vector.search).toHaveBeenCalledWith(searchOpts);
  });

  it("should handle errors", async () => {
    const client = createMockClient();
    (client.vector.search as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: null,
      error: new Error("Search failed"),
    });

    const { result } = renderHook(() => useVectorSearch(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync({
          table: "documents",
          column: "embedding",
          query: "test",
        });
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
