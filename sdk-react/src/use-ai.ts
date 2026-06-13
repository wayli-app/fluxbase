/**
 * AI hooks for Fluxbase React SDK
 * Provides hooks for chatbot chat, conversation history, and knowledge base search
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type {
  ListConversationsOptions,
  AIChatEvent,
} from "@nimbleflux/fluxbase-sdk";

/**
 * Hook to list all public/enabled chatbots
 */
export function useChatbots(namespace?: string) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "ai", "chatbots", namespace],
    queryFn: async () => {
      const { data, error } = await client.ai.listChatbots(namespace);
      if (error) throw error;
      return data || [];
    },
  });
}

/**
 * Hook to list the current user's conversations
 */
export function useConversations(options?: ListConversationsOptions) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "ai", "conversations", options],
    queryFn: async () => {
      const { data, error } = await client.ai.listConversations(options);
      if (error) throw error;
      return data;
    },
  });
}

/**
 * Hook to get a conversation with full message history
 */
export function useConversation(conversationId: string | null) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "ai", "conversation", conversationId],
    queryFn: async () => {
      if (!conversationId) return null;
      const { data, error } = await client.ai.getConversation(conversationId);
      if (error) throw error;
      return data;
    },
    enabled: !!conversationId,
  });
}

/**
 * Hook to delete a conversation
 */
export function useDeleteConversation() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (conversationId: string) => {
      const { error } = await client.ai.deleteConversation(conversationId);
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["fluxbase", "ai", "conversations"] });
    },
  });
}

interface AIChatState {
  messages: ChatMessage[];
  isStreaming: boolean;
  isConnected: boolean;
  error: Error | null;
}

interface ChatMessage {
  role: "user" | "assistant";
  content: string;
  queryResults?: Array<{
    query: string;
    summary: string;
    rowCount: number;
    data?: Record<string, unknown>[];
  }>;
  progress?: string[];
}

export type { ChatMessage };

/**
 * Hook for AI chatbot streaming chat
 *
 * @example
 * ```tsx
 * const { messages, sendMessage, isStreaming, error } = useAIChat({
 *   chatbot: 'my-chatbot',
 *   onQueryResult: (result) => console.log('SQL:', result.query),
 * })
 * ```
 */
export function useAIChat(options: {
  chatbot: string;
  namespace?: string;
  onError?: (error: Error) => void;
  onQueryResult?: (result: {
    query: string;
    summary: string;
    rowCount: number;
    data?: Record<string, unknown>[];
  }) => void;
}) {
  const client = useFluxbaseClient();
  const chatRef = useRef<ReturnType<typeof client.ai.createChat> | null>(null);
  const conversationIdRef = useRef<string | null>(null);
  const [state, setState] = useState<AIChatState>({
    messages: [],
    isStreaming: false,
    isConnected: false,
    error: null,
  });
  const assistantBufferRef = useRef<string>("");
  const assistantProgressRef = useRef<string[]>([]);

  // Connect on mount
  useEffect(() => {
    const chat = client.ai.createChat({
      onEvent: (event: AIChatEvent) => {
        handleEvent(event);
      },
      onContent: (delta: string) => {
        assistantBufferRef.current += delta;
      },
      onProgress: (_step: string, message: string) => {
        assistantProgressRef.current = [
          ...assistantProgressRef.current,
          message,
        ];
      },
      onQueryResult: (query, summary, rowCount, data) => {
        options.onQueryResult?.({ query, summary, rowCount, data });
      },
      onDone: () => {
        setState((prev) => ({
          ...prev,
          messages: [
            ...prev.messages,
            {
              role: "assistant" as const,
              content: assistantBufferRef.current,
              progress: assistantProgressRef.current.length > 0 ? [...assistantProgressRef.current] : undefined,
            },
          ],
          isStreaming: false,
        }));
        assistantBufferRef.current = "";
        assistantProgressRef.current = [];
      },
      onError: (error: string) => {
        const err = new Error(error);
        setState((prev) => ({ ...prev, isStreaming: false, error: err }));
        options.onError?.(err);
      },
    });

    chatRef.current = chat;

    chat.connect().then(() => {
      setState((prev) => ({ ...prev, isConnected: true }));
    }).catch((err: Error) => {
      setState((prev) => ({ ...prev, error: err }));
    });

    return () => {
      chat.disconnect();
      chatRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [options.chatbot, options.namespace]);

  // Internal event handler (not used directly since we set callbacks)
  const handleEvent = (_event: AIChatEvent) => {
    // Events are handled via individual callbacks above
  };

  const sendMessage = useCallback(async (content: string) => {
    const chat = chatRef.current;
    if (!chat || !chat.isConnected()) {
      setState((prev) => ({ ...prev, error: new Error("Not connected") }));
      return;
    }

    // Add user message immediately
    setState((prev) => ({
      ...prev,
      messages: [...prev.messages, { role: "user", content }],
      isStreaming: true,
      error: null,
    }));

    // Start chat if needed, then send
    if (!conversationIdRef.current) {
      conversationIdRef.current = await chat.startChat(
        options.chatbot,
        options.namespace,
      );
    }

    await chat.sendMessage(conversationIdRef.current, content);
  }, [options.chatbot, options.namespace]);

  const cancel = useCallback(() => {
    const chat = chatRef.current;
    if (chat && conversationIdRef.current) {
      chat.cancel(conversationIdRef.current);
    }
    setState((prev) => ({ ...prev, isStreaming: false }));
  }, []);

  const reset = useCallback(() => {
    conversationIdRef.current = null;
    assistantBufferRef.current = "";
    assistantProgressRef.current = [];
    setState({
      messages: [],
      isStreaming: false,
      isConnected: state.isConnected,
      error: null,
    });
  }, [state.isConnected]);

  return {
    messages: state.messages,
    isStreaming: state.isStreaming,
    isConnected: state.isConnected,
    error: state.error,
    sendMessage,
    cancel,
    reset,
  };
}
