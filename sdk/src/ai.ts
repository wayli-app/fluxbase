/**
 * AI Chat module for interacting with AI chatbots
 * Provides WebSocket-based chat functionality with streaming support
 */

import type {
  AIChatbotSummary,
  AIChatClientMessage,
  AIChatServerMessage,
  AIUsageStats,
  AIUserConversationDetail,
  ListConversationsOptions,
  ListConversationsResult,
  UpdateConversationOptions,
} from "./types";

/**
 * Event types for chat callbacks
 */
export type AIChatEventType =
  | "connected"
  | "chat_started"
  | "progress"
  | "content"
  | "query_result"
  | "done"
  | "error"
  | "cancelled"
  | "disconnected";

/**
 * Chat event data
 */
export interface AIChatEvent {
  type: AIChatEventType;
  conversationId?: string;
  chatbot?: string;
  step?: string;
  message?: string;
  delta?: string;
  query?: string;
  summary?: string;
  rowCount?: number;
  data?: Record<string, any>[];
  usage?: AIUsageStats;
  error?: string;
  code?: string;
}

/**
 * Chat connection options
 */
export interface AIChatOptions {
  /** WebSocket URL (defaults to ws://host/ai/ws) */
  wsUrl?: string;
  /** JWT token for authentication */
  token?: string;
  /** Callback for all events */
  onEvent?: (event: AIChatEvent) => void;
  /** Callback for content chunks (streaming) */
  onContent?: (delta: string, conversationId: string) => void;
  /** Callback for progress updates */
  onProgress?: (step: string, message: string, conversationId: string) => void;
  /** Callback for query results */
  onQueryResult?: (
    query: string,
    summary: string,
    rowCount: number,
    data: Record<string, any>[],
    conversationId: string,
  ) => void;
  /** Callback when message is complete */
  onDone?: (usage: AIUsageStats | undefined, conversationId: string) => void;
  /** Callback for errors */
  onError?: (error: string, code: string | undefined, conversationId: string | undefined) => void;
  /** Reconnect attempts (0 = no reconnect) */
  reconnectAttempts?: number;
  /** Reconnect delay in ms */
  reconnectDelay?: number;
}

/**
 * AI Chat client for WebSocket-based chat with AI chatbots
 *
 * @example
 * ```typescript
 * const chat = new FluxbaseAIChat({
 *   wsUrl: 'ws://localhost:8080/ai/ws',
 *   token: 'my-jwt-token',
 *   onContent: (delta, convId) => {
 *     process.stdout.write(delta)
 *   },
 *   onProgress: (step, message) => {
 *     console.log(`[${step}] ${message}`)
 *   },
 *   onQueryResult: (query, summary, rowCount, data) => {
 *     console.log(`Query: ${query}`)
 *     console.log(`Result: ${summary} (${rowCount} rows)`)
 *   },
 *   onDone: (usage) => {
 *     console.log(`\nTokens: ${usage?.total_tokens}`)
 *   },
 *   onError: (error, code) => {
 *     console.error(`Error: ${error} (${code})`)
 *   },
 * })
 *
 * await chat.connect()
 * const convId = await chat.startChat('sql-assistant')
 * await chat.sendMessage(convId, 'Show me the top 10 users by order count')
 * ```
 */
export class FluxbaseAIChat {
  private ws: WebSocket | null = null;
  private options: AIChatOptions;
  private reconnectCount = 0;
  private pendingStartResolve: ((convId: string) => void) | null = null;
  private pendingStartReject: ((error: Error) => void) | null = null;
  private accumulatedContent: Map<string, string> = new Map();

  constructor(options: AIChatOptions) {
    this.options = {
      reconnectAttempts: 3,
      reconnectDelay: 1000,
      ...options,
    };
  }

  /**
   * Connect to the AI chat WebSocket
   *
   * @returns Promise that resolves when connected
   */
  async connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const url = this.buildWsUrl();

      try {
        this.ws = new WebSocket(url);

        this.ws.onopen = () => {
          this.reconnectCount = 0;
          this.emitEvent({ type: "connected" });
          resolve();
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };

        this.ws.onclose = (event) => {
          this.emitEvent({ type: "disconnected" });
          this.handleClose(event);
        };

        this.ws.onerror = () => {
          reject(new Error("WebSocket connection failed"));
        };
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * Disconnect from the AI chat WebSocket
   */
  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  /**
   * Start a new chat session with a chatbot
   *
   * @param chatbot - Chatbot name
   * @param namespace - Optional namespace (defaults to 'default')
   * @param conversationId - Optional conversation ID to resume
   * @param impersonateUserId - Optional user ID to impersonate (admin only)
   * @returns Promise resolving to conversation ID
   */
  async startChat(
    chatbot: string,
    namespace?: string,
    conversationId?: string,
    impersonateUserId?: string,
  ): Promise<string> {
    return new Promise((resolve, reject) => {
      if (!this.isConnected()) {
        reject(new Error("Not connected to AI chat"));
        return;
      }

      this.pendingStartResolve = resolve;
      this.pendingStartReject = reject;

      const message: AIChatClientMessage = {
        type: "start_chat",
        chatbot,
        namespace: namespace || "default",
        conversation_id: conversationId,
        impersonate_user_id: impersonateUserId,
      };

      this.ws!.send(JSON.stringify(message));
    });
  }

  /**
   * Send a message in a conversation
   *
   * @param conversationId - Conversation ID
   * @param content - Message content
   */
  sendMessage(conversationId: string, content: string): void {
    if (!this.isConnected()) {
      throw new Error("Not connected to AI chat");
    }

    // Reset accumulated content for this conversation
    this.accumulatedContent.set(conversationId, "");

    const message: AIChatClientMessage = {
      type: "message",
      conversation_id: conversationId,
      content,
    };

    this.ws!.send(JSON.stringify(message));
  }

  /**
   * Cancel an ongoing message generation
   *
   * @param conversationId - Conversation ID
   */
  cancel(conversationId: string): void {
    if (!this.isConnected()) {
      throw new Error("Not connected to AI chat");
    }

    const message: AIChatClientMessage = {
      type: "cancel",
      conversation_id: conversationId,
    };

    this.ws!.send(JSON.stringify(message));
  }

  /**
   * Get the full accumulated response content for a conversation
   *
   * @param conversationId - Conversation ID
   * @returns Accumulated content string
   */
  getAccumulatedContent(conversationId: string): string {
    return this.accumulatedContent.get(conversationId) || "";
  }

  private buildWsUrl(): string {
    let url = this.options.wsUrl || "/ai/ws";

    // Add token if provided
    if (this.options.token) {
      const separator = url.includes("?") ? "&" : "?";
      url = `${url}${separator}token=${encodeURIComponent(this.options.token)}`;
    }

    return url;
  }

  private handleMessage(data: string): void {
    try {
      const message: AIChatServerMessage = JSON.parse(data);
      const event = this.serverMessageToEvent(message);

      // Handle special cases
      switch (message.type) {
        case "chat_started":
          if (this.pendingStartResolve && message.conversation_id) {
            this.pendingStartResolve(message.conversation_id);
            this.pendingStartResolve = null;
            this.pendingStartReject = null;
          }
          break;

        case "content":
          if (message.conversation_id && message.delta) {
            const current = this.accumulatedContent.get(message.conversation_id) || "";
            this.accumulatedContent.set(message.conversation_id, current + message.delta);
            this.options.onContent?.(message.delta, message.conversation_id);
          }
          break;

        case "progress":
          if (message.step && message.message && message.conversation_id) {
            this.options.onProgress?.(message.step, message.message, message.conversation_id);
          }
          break;

        case "query_result":
          if (message.conversation_id) {
            this.options.onQueryResult?.(
              message.query || "",
              message.summary || "",
              message.row_count || 0,
              message.data || [],
              message.conversation_id,
            );
          }
          break;

        case "done":
          if (message.conversation_id) {
            this.options.onDone?.(message.usage, message.conversation_id);
          }
          break;

        case "error":
          if (this.pendingStartReject) {
            this.pendingStartReject(new Error(message.error || "Unknown error"));
            this.pendingStartResolve = null;
            this.pendingStartReject = null;
          }
          this.options.onError?.(message.error || "Unknown error", message.code, message.conversation_id);
          break;
      }

      // Always emit the general event
      this.emitEvent(event);
    } catch (error) {
      console.error("Failed to parse AI chat message:", error);
    }
  }

  private serverMessageToEvent(message: AIChatServerMessage): AIChatEvent {
    return {
      type: message.type as AIChatEventType,
      conversationId: message.conversation_id,
      chatbot: message.chatbot,
      step: message.step,
      message: message.message,
      delta: message.delta,
      query: message.query,
      summary: message.summary,
      rowCount: message.row_count,
      data: message.data,
      usage: message.usage,
      error: message.error,
      code: message.code,
    };
  }

  private emitEvent(event: AIChatEvent): void {
    this.options.onEvent?.(event);
  }

  private handleClose(_event: CloseEvent): void {
    // Attempt reconnect if configured
    if (
      this.options.reconnectAttempts &&
      this.reconnectCount < this.options.reconnectAttempts
    ) {
      this.reconnectCount++;
      setTimeout(() => {
        this.connect().catch(() => {
          // Reconnect failed, will try again if attempts remain
        });
      }, this.options.reconnectDelay);
    }
  }
}

/**
 * Fluxbase AI client for listing chatbots and managing conversations
 *
 * @example
 * ```typescript
 * const ai = new FluxbaseAI(fetchClient, 'ws://localhost:8080')
 *
 * // List available chatbots
 * const { data, error } = await ai.listChatbots()
 *
 * // Create a chat connection
 * const chat = ai.createChat({
 *   token: 'my-jwt-token',
 *   onContent: (delta) => process.stdout.write(delta),
 * })
 *
 * await chat.connect()
 * const convId = await chat.startChat('sql-assistant')
 * chat.sendMessage(convId, 'Show me recent orders')
 * ```
 */
export class FluxbaseAI {
  private fetch: {
    get: <T>(path: string) => Promise<T>;
    patch: <T>(path: string, body?: unknown) => Promise<T>;
    delete: (path: string) => Promise<void>;
  };
  private wsBaseUrl: string;

  constructor(
    fetch: {
      get: <T>(path: string) => Promise<T>;
      patch: <T>(path: string, body?: unknown) => Promise<T>;
      delete: (path: string) => Promise<void>;
    },
    wsBaseUrl: string,
  ) {
    this.fetch = fetch;
    this.wsBaseUrl = wsBaseUrl;
  }

  /**
   * List available chatbots (public, enabled)
   *
   * @returns Promise resolving to { data, error } tuple with array of chatbot summaries
   */
  async listChatbots(): Promise<{
    data: AIChatbotSummary[] | null;
    error: Error | null;
  }> {
    try {
      const response = await this.fetch.get<{ chatbots: AIChatbotSummary[]; count: number }>(
        "/api/v1/ai/chatbots",
      );
      return { data: response.chatbots || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get details of a specific chatbot
   *
   * @param id - Chatbot ID
   * @returns Promise resolving to { data, error } tuple with chatbot details
   */
  async getChatbot(
    id: string,
  ): Promise<{ data: AIChatbotSummary | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<AIChatbotSummary>(`/api/v1/ai/chatbots/${id}`);
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Create a new AI chat connection
   *
   * @param options - Chat connection options
   * @returns FluxbaseAIChat instance
   */
  createChat(options: Omit<AIChatOptions, "wsUrl">): FluxbaseAIChat {
    return new FluxbaseAIChat({
      ...options,
      wsUrl: `${this.wsBaseUrl}/ai/ws`,
    });
  }

  /**
   * List the authenticated user's conversations
   *
   * @param options - Optional filters and pagination
   * @returns Promise resolving to { data, error } tuple with conversations
   *
   * @example
   * ```typescript
   * // List all conversations
   * const { data, error } = await ai.listConversations()
   *
   * // Filter by chatbot
   * const { data, error } = await ai.listConversations({ chatbot: 'sql-assistant' })
   *
   * // With pagination
   * const { data, error } = await ai.listConversations({ limit: 20, offset: 0 })
   * ```
   */
  async listConversations(
    options?: ListConversationsOptions,
  ): Promise<{ data: ListConversationsResult | null; error: Error | null }> {
    try {
      const params = new URLSearchParams();
      if (options?.chatbot) params.set("chatbot", options.chatbot);
      if (options?.namespace) params.set("namespace", options.namespace);
      if (options?.limit !== undefined) params.set("limit", String(options.limit));
      if (options?.offset !== undefined) params.set("offset", String(options.offset));

      const queryString = params.toString();
      const path = `/api/v1/ai/conversations${queryString ? `?${queryString}` : ""}`;

      const response = await this.fetch.get<ListConversationsResult>(path);
      return { data: response, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get a single conversation with all messages
   *
   * @param id - Conversation ID
   * @returns Promise resolving to { data, error } tuple with conversation detail
   *
   * @example
   * ```typescript
   * const { data, error } = await ai.getConversation('conv-uuid-123')
   * if (data) {
   *   console.log(`Title: ${data.title}`)
   *   console.log(`Messages: ${data.messages.length}`)
   * }
   * ```
   */
  async getConversation(
    id: string,
  ): Promise<{ data: AIUserConversationDetail | null; error: Error | null }> {
    try {
      const response = await this.fetch.get<AIUserConversationDetail>(
        `/api/v1/ai/conversations/${id}`,
      );
      return { data: response, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Delete a conversation
   *
   * @param id - Conversation ID
   * @returns Promise resolving to { error } (null on success)
   *
   * @example
   * ```typescript
   * const { error } = await ai.deleteConversation('conv-uuid-123')
   * if (!error) {
   *   console.log('Conversation deleted')
   * }
   * ```
   */
  async deleteConversation(id: string): Promise<{ error: Error | null }> {
    try {
      await this.fetch.delete(`/api/v1/ai/conversations/${id}`);
      return { error: null };
    } catch (error) {
      return { error: error as Error };
    }
  }

  /**
   * Update a conversation (currently supports title update only)
   *
   * @param id - Conversation ID
   * @param updates - Fields to update
   * @returns Promise resolving to { data, error } tuple with updated conversation
   *
   * @example
   * ```typescript
   * const { data, error } = await ai.updateConversation('conv-uuid-123', {
   *   title: 'My custom conversation title'
   * })
   * ```
   */
  async updateConversation(
    id: string,
    updates: UpdateConversationOptions,
  ): Promise<{ data: AIUserConversationDetail | null; error: Error | null }> {
    try {
      const response = await this.fetch.patch<AIUserConversationDetail>(
        `/api/v1/ai/conversations/${id}`,
        updates,
      );
      return { data: response, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }
}
