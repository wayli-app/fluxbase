/**
 * Tests for AI hooks
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import {
  useChatbots,
  useConversations,
  useConversation,
  useDeleteConversation,
} from "./use-ai";
import {
  createMockAIClient,
  createWrapper,
} from "./test-utils";
import type { FluxbaseClient } from "@nimbleflux/fluxbase-sdk";

describe("useChatbots", () => {
  it("should list chatbots", async () => {
    const mockChatbots = [
      { id: "1", name: "Bot1", namespace: "default", enabled: true, version: "1", source: "filesystem" },
    ];
    const client = createMockAIClient({
      ai: {
        listChatbots: vi.fn().mockResolvedValue({ data: mockChatbots, error: null }),
      } as any,
    });

    const { result } = renderHook(() => useChatbots(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockChatbots);
  });

  it("should filter by namespace", async () => {
    const listChatbots = vi
      .fn()
      .mockResolvedValue({ data: [], error: null });
    const client = createMockAIClient({
      ai: { listChatbots } as any,
    });

    renderHook(() => useChatbots("my-ns"), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => {
      expect(listChatbots).toHaveBeenCalledWith("my-ns");
    });
  });

  it("should handle errors", async () => {
    const client = createMockAIClient({
      ai: {
        listChatbots: vi
          .fn()
          .mockResolvedValue({ data: null, error: new Error("Failed") }),
      } as any,
    });

    const { result } = renderHook(() => useChatbots(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});

describe("useConversations", () => {
  it("should list conversations", async () => {
    const mockData = {
      conversations: [
        { id: "c1", chatbot: "bot1", namespace: "default", title: "Test", created_at: "", updated_at: "" },
      ],
      total: 1,
      has_more: false,
    };
    const client = createMockAIClient({
      ai: {
        listConversations: vi
          .fn()
          .mockResolvedValue({ data: mockData, error: null }),
      } as any,
    });

    const { result } = renderHook(() => useConversations(), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockData);
  });

  it("should pass options to listConversations", async () => {
    const listConversations = vi
      .fn()
      .mockResolvedValue({ data: { conversations: [], total: 0, has_more: false }, error: null });
    const client = createMockAIClient({
      ai: { listConversations } as any,
    });

    renderHook(
      () => useConversations({ chatbot: "my-bot", limit: 10 }),
      { wrapper: createWrapper(client) },
    );

    await waitFor(() => {
      expect(listConversations).toHaveBeenCalledWith({
        chatbot: "my-bot",
        limit: 10,
      });
    });
  });
});

describe("useConversation", () => {
  it("should get a conversation by ID", async () => {
    const mockConv = { id: "c1", chatbot: "bot1", namespace: "default", created_at: "", updated_at: "", messages: [] };
    const client = createMockAIClient({
      ai: {
        getConversation: vi
          .fn()
          .mockResolvedValue({ data: mockConv, error: null }),
      } as any,
    });

    const { result } = renderHook(() => useConversation("c1"), {
      wrapper: createWrapper(client),
    });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(mockConv);
  });

  it("should not fetch when conversationId is null", () => {
    const client = createMockAIClient();
    const { result } = renderHook(() => useConversation(null), {
      wrapper: createWrapper(client),
    });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.data).toBeUndefined();
  });
});

describe("useDeleteConversation", () => {
  it("should delete a conversation and invalidate queries", async () => {
    const deleteConversation = vi
      .fn()
      .mockResolvedValue({ error: null });
    const client = createMockAIClient({
      ai: { deleteConversation } as any,
    });

    const { result } = renderHook(() => useDeleteConversation(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      await result.current.mutateAsync("conv-1");
    });

    expect(deleteConversation).toHaveBeenCalledWith("conv-1");
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("should handle errors", async () => {
    const client = createMockAIClient({
      ai: {
        deleteConversation: vi
          .fn()
          .mockResolvedValue({ error: new Error("Not found") }),
      } as any,
    });

    const { result } = renderHook(() => useDeleteConversation(), {
      wrapper: createWrapper(client),
    });

    await act(async () => {
      try {
        await result.current.mutateAsync("conv-1");
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
