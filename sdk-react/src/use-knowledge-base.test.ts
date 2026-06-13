/**
 * Tests for Knowledge Base hooks
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import {
  useKnowledgeBases,
  useKnowledgeBase,
  useKBDocuments,
  useCreateKnowledgeBase,
  useDeleteKnowledgeBase,
  useAddDocument,
  useUploadDocument,
  useDeleteDocument,
  useKBSearch,
  useKBEntities,
  useKnowledgeGraph,
} from "./use-knowledge-base";
import {
  createMockAIClient,
  createWrapper,
} from "./test-utils";

describe("useKnowledgeBases", () => {
  it("should list knowledge bases", async () => {
    const mockKbs = [
      { id: "kb1", name: "Test KB", namespace: "default", description: "", enabled: true, document_count: 0, total_chunks: 0, embedding_model: "text-embedding-3-small", created_at: "", updated_at: "" },
    ];
    const client = createMockAIClient({
      knowledgeBase: {
        list: vi.fn().mockResolvedValue({ data: mockKbs, error: null }),
      } as any,
    });

    const { result } = renderHook(() => useKnowledgeBases(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockKbs);
  });

  it("should handle errors", async () => {
    const client = createMockAIClient({
      knowledgeBase: {
        list: vi.fn().mockResolvedValue({ data: null, error: new Error("Failed") }),
      } as any,
    });

    const { result } = renderHook(() => useKnowledgeBases(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useKnowledgeBase", () => {
  it("should get a single knowledge base", async () => {
    const mockKb = {
      id: "kb1", name: "Test KB", namespace: "default", description: "Test",
      enabled: true, document_count: 5, total_chunks: 50,
      embedding_model: "text-embedding-3-small", embedding_dimensions: 1536,
      chunk_size: 1000, chunk_overlap: 200, chunk_strategy: "recursive",
      source: "", created_at: "", updated_at: "",
    };
    const client = createMockAIClient({
      knowledgeBase: {
        get: vi.fn().mockResolvedValue({ data: mockKb, error: null }),
      } as any,
    });

    const { result } = renderHook(() => useKnowledgeBase("kb1"), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockKb);
  });

  it("should not fetch when kbId is null", () => {
    const client = createMockAIClient();
    const { result } = renderHook(() => useKnowledgeBase(null), {
      wrapper: createWrapper(client),
    });
    expect(result.current.data).toBeUndefined();
  });
});

describe("useCreateKnowledgeBase", () => {
  it("should create a knowledge base", async () => {
    const createFn = vi.fn().mockResolvedValue({
      data: { id: "kb1", name: "New" },
      error: null,
    });
    const client = createMockAIClient({
      knowledgeBase: { create: createFn } as any,
    });

    const { result } = renderHook(() => useCreateKnowledgeBase(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: "New" });
    });

    expect(createFn).toHaveBeenCalledWith({ name: "New" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useDeleteKnowledgeBase", () => {
  it("should delete a knowledge base", async () => {
    const deleteFn = vi.fn().mockResolvedValue({ data: true, error: null });
    const client = createMockAIClient({
      knowledgeBase: { delete: deleteFn } as any,
    });

    const { result } = renderHook(() => useDeleteKnowledgeBase(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync("kb1");
    });

    expect(deleteFn).toHaveBeenCalledWith("kb1");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useKBDocuments", () => {
  it("should list documents in a knowledge base", async () => {
    const mockDocs = [
      { id: "doc1", knowledge_base_id: "kb1", title: "Doc 1", mime_type: "text/plain", content_hash: "abc", chunk_count: 5, status: "indexed", created_at: "", updated_at: "" },
    ];
    const client = createMockAIClient({
      knowledgeBase: {
        listDocuments: vi.fn().mockResolvedValue({ data: mockDocs, error: null }),
      } as any,
    });

    const { result } = renderHook(() => useKBDocuments("kb1"), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockDocs);
  });

  it("should not fetch when kbId is null", () => {
    const client = createMockAIClient();
    const { result } = renderHook(() => useKBDocuments(null), {
      wrapper: createWrapper(client),
    });
    expect(result.current.isLoading).toBe(false);
  });
});

describe("useAddDocument", () => {
  it("should add a document", async () => {
    const addFn = vi.fn().mockResolvedValue({
      data: { document_id: "doc1", status: "processing", message: "Added" },
      error: null,
    });
    const client = createMockAIClient({
      knowledgeBase: { addDocument: addFn } as any,
    });

    const { result } = renderHook(() => useAddDocument(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({
        kbId: "kb1",
        request: { title: "Test", content: "Hello world" },
      });
    });

    expect(addFn).toHaveBeenCalledWith("kb1", { title: "Test", content: "Hello world" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useUploadDocument", () => {
  it("should upload a file document", async () => {
    const uploadFn = vi.fn().mockResolvedValue({
      data: { document_id: "doc1", status: "processing", message: "Uploaded", filename: "test.pdf", extracted_length: 1000, mime_type: "application/pdf" },
      error: null,
    });
    const client = createMockAIClient({
      knowledgeBase: { uploadDocument: uploadFn } as any,
    });

    const { result } = renderHook(() => useUploadDocument(), {
      wrapper: createWrapper(client),
    });

    const file = new Blob(["content"], { type: "application/pdf" });

    await act(async () => {
      await result.current.mutateAsync({
        kbId: "kb1",
        file,
        filename: "test.pdf",
      });
    });

    expect(uploadFn).toHaveBeenCalledWith("kb1", file, "test.pdf", undefined);
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useDeleteDocument", () => {
  it("should delete a document", async () => {
    const deleteFn = vi.fn().mockResolvedValue({ data: true, error: null });
    const client = createMockAIClient({
      knowledgeBase: { deleteDocument: deleteFn } as any,
    });

    const { result } = renderHook(() => useDeleteDocument(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync({ kbId: "kb1", docId: "doc1" });
    });

    expect(deleteFn).toHaveBeenCalledWith("kb1", "doc1");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useKBSearch", () => {
  it("should search a knowledge base", async () => {
    const mockResults = {
      results: [{ chunk_id: "c1", document_id: "d1", document_title: "Doc 1", content: "test", similarity: 0.95 }],
      count: 1,
      query: "test",
    };
    const searchFn = vi.fn().mockResolvedValue({ data: mockResults, error: null });
    const client = createMockAIClient({
      knowledgeBase: { search: searchFn } as any,
    });

    const { result } = renderHook(
      () => useKBSearch("kb1", { query: "test" }),
      { wrapper: createWrapper(client) },
    );

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockResults);
    expect(searchFn).toHaveBeenCalledWith("kb1", { query: "test" });
  });

  it("should not search when request is null", () => {
    const client = createMockAIClient();
    const { result } = renderHook(() => useKBSearch("kb1", null), {
      wrapper: createWrapper(client),
    });
    expect(result.current.data).toBeUndefined();
  });
});

describe("useKBEntities", () => {
  it("should list entities", async () => {
    const mockEntities = [
      { id: "e1", knowledge_base_id: "kb1", entity_type: "person", name: "John", canonical_name: "John", aliases: [], metadata: {}, created_at: "", updated_at: "" },
    ];
    const client = createMockAIClient({
      knowledgeBase: {
        listEntities: vi.fn().mockResolvedValue({ data: mockEntities, error: null }),
      } as any,
    });

    const { result } = renderHook(() => useKBEntities("kb1"), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockEntities);
  });
});

describe("useKnowledgeGraph", () => {
  it("should get the knowledge graph", async () => {
    const mockGraph = {
      entities: [],
      relationships: [],
      entity_count: 0,
      relationship_count: 0,
    };
    const client = createMockAIClient({
      knowledgeBase: {
        getKnowledgeGraph: vi.fn().mockResolvedValue({ data: mockGraph, error: null }),
      } as any,
    });

    const { result } = renderHook(() => useKnowledgeGraph("kb1"), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockGraph);
  });
});
