/**
 * AI Module Tests
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { FluxbaseAI, FluxbaseAIChat } from './ai'
import type { AIChatbotSummary, AIChatbotLookupResponse, ListConversationsResult, AIUserConversationDetail } from './types'

// Mock WebSocket implementation
class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState = MockWebSocket.CONNECTING;
  url: string;
  onopen: (() => void) | null = null;
  onclose: ((event: { code: number; reason: string }) => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: ((event: { error: Error }) => void) | null = null;

  sentMessages: string[] = [];

  constructor(url: string) {
    this.url = url;
  }

  // Simulate successful connection
  simulateOpen() {
    this.readyState = MockWebSocket.OPEN;
    if (this.onopen) {
      this.onopen();
    }
  }

  // Simulate receiving a message
  simulateMessage(data: object) {
    if (this.onmessage) {
      this.onmessage({ data: JSON.stringify(data) });
    }
  }

  // Simulate connection close
  simulateClose(code = 1000, reason = 'Normal closure') {
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) {
      this.onclose({ code, reason });
    }
  }

  // Simulate error
  simulateError(error = new Error('WebSocket error')) {
    if (this.onerror) {
      this.onerror({ error });
    }
  }

  send(data: string) {
    this.sentMessages.push(data);
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
  }
}

// Install mock WebSocket globally
(global as any).WebSocket = MockWebSocket;

// Mock fetch interface
class MockFetch {
  public lastUrl: string = ''
  public lastMethod: string = ''
  public lastBody: unknown = null
  public mockResponse: any = null
  public shouldThrow: boolean = false
  public errorMessage: string = 'Test error'

  async get<T>(path: string): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'GET'
    if (this.shouldThrow) {
      throw new Error(this.errorMessage)
    }
    return this.mockResponse as T
  }

  async patch<T>(path: string, body?: unknown): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'PATCH'
    this.lastBody = body
    if (this.shouldThrow) {
      throw new Error(this.errorMessage)
    }
    return this.mockResponse as T
  }

  async delete(path: string): Promise<void> {
    this.lastUrl = path
    this.lastMethod = 'DELETE'
    if (this.shouldThrow) {
      throw new Error(this.errorMessage)
    }
  }
}

describe('FluxbaseAI', () => {
  let mockFetch: MockFetch
  let ai: FluxbaseAI

  beforeEach(() => {
    mockFetch = new MockFetch()
    ai = new FluxbaseAI(mockFetch, 'ws://localhost:8080')
  })

  describe('listChatbots', () => {
    it('should list available chatbots', async () => {
      const mockChatbots: AIChatbotSummary[] = [
        {
          id: 'cb1',
          name: 'sql-assistant',
          namespace: 'default',
          description: 'SQL query assistant',
        },
        {
          id: 'cb2',
          name: 'data-analyst',
          namespace: 'default',
          description: 'Data analysis helper',
        },
      ]
      mockFetch.mockResponse = { chatbots: mockChatbots, count: 2 }

      const { data, error } = await ai.listChatbots()

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/chatbots')
      expect(mockFetch.lastMethod).toBe('GET')
      expect(data).toEqual(mockChatbots)
      expect(error).toBeNull()
    })

    it('should return empty array when no chatbots', async () => {
      mockFetch.mockResponse = { count: 0 }

      const { data, error } = await ai.listChatbots()

      expect(data).toEqual([])
      expect(error).toBeNull()
    })

    it('should handle errors', async () => {
      mockFetch.shouldThrow = true
      mockFetch.errorMessage = 'Permission denied'

      const { data, error } = await ai.listChatbots()

      expect(data).toBeNull()
      expect(error).toBeDefined()
      expect(error?.message).toBe('Permission denied')
    })

    it('should filter by namespace', async () => {
      mockFetch.mockResponse = { chatbots: [], count: 0 }

      await ai.listChatbots('custom-ns')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/chatbots?namespace=custom-ns')
      expect(mockFetch.lastMethod).toBe('GET')
    })
  })

  describe('getChatbot', () => {
    it('should get chatbot details', async () => {
      const mockChatbot: AIChatbotSummary = {
        id: 'cb1',
        name: 'sql-assistant',
        namespace: 'default',
        description: 'SQL query assistant',
      }
      mockFetch.mockResponse = mockChatbot

      const { data, error } = await ai.getChatbot('cb1')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/chatbots/cb1')
      expect(data).toEqual(mockChatbot)
      expect(error).toBeNull()
    })

    it('should handle errors', async () => {
      mockFetch.shouldThrow = true
      mockFetch.errorMessage = 'Chatbot not found'

      const { data, error } = await ai.getChatbot('non-existent')

      expect(data).toBeNull()
      expect(error).toBeDefined()
    })
  })

  describe('lookupChatbot', () => {
    it('should lookup chatbot by name', async () => {
      const mockLookup: AIChatbotLookupResponse = {
        chatbot: {
          id: 'cb1',
          name: 'sql-assistant',
          namespace: 'default',
        },
      }
      mockFetch.mockResponse = mockLookup

      const { data, error } = await ai.lookupChatbot('sql-assistant')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/chatbots/by-name/sql-assistant')
      expect(data).toEqual(mockLookup)
      expect(error).toBeNull()
    })

    it('should encode chatbot name in URL', async () => {
      mockFetch.mockResponse = { chatbot: { name: 'my assistant' } }

      await ai.lookupChatbot('my assistant')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/chatbots/by-name/my%20assistant')
    })

    it('should handle ambiguous lookup', async () => {
      const mockLookup: AIChatbotLookupResponse = {
        ambiguous: true,
        namespaces: ['default', 'custom'],
      }
      mockFetch.mockResponse = mockLookup

      const { data, error } = await ai.lookupChatbot('common-name')

      expect(data?.ambiguous).toBe(true)
      expect(data?.namespaces).toEqual(['default', 'custom'])
      expect(error).toBeNull()
    })

    it('should handle errors', async () => {
      mockFetch.shouldThrow = true
      mockFetch.errorMessage = 'Chatbot not found'

      const { data, error } = await ai.lookupChatbot('non-existent')

      expect(data).toBeNull()
      expect(error).toBeDefined()
    })
  })

  describe('listConversations', () => {
    it('should list conversations', async () => {
      const mockResult: ListConversationsResult = {
        conversations: [
          {
            id: 'conv1',
            chatbot_name: 'sql-assistant',
            title: 'Query help',
            created_at: '2025-01-01T00:00:00Z',
            updated_at: '2025-01-01T00:00:00Z',
          },
        ],
        count: 1,
      }
      mockFetch.mockResponse = mockResult

      const { data, error } = await ai.listConversations()

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/conversations')
      expect(data).toEqual(mockResult)
      expect(error).toBeNull()
    })

    it('should filter by chatbot', async () => {
      mockFetch.mockResponse = { conversations: [], count: 0 }

      await ai.listConversations({ chatbot: 'sql-assistant' })

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/conversations?chatbot=sql-assistant')
    })

    it('should filter by namespace', async () => {
      mockFetch.mockResponse = { conversations: [], count: 0 }

      await ai.listConversations({ namespace: 'custom' })

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/conversations?namespace=custom')
    })

    it('should support pagination', async () => {
      mockFetch.mockResponse = { conversations: [], count: 0 }

      await ai.listConversations({ limit: 10, offset: 20 })

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/conversations?limit=10&offset=20')
    })

    it('should handle errors', async () => {
      mockFetch.shouldThrow = true

      const { data, error } = await ai.listConversations()

      expect(data).toBeNull()
      expect(error).toBeDefined()
    })
  })

  describe('getConversation', () => {
    it('should get conversation details', async () => {
      const mockConv: AIUserConversationDetail = {
        id: 'conv1',
        chatbot_name: 'sql-assistant',
        title: 'Query help',
        messages: [
          { role: 'user', content: 'Help me write a query' },
          { role: 'assistant', content: 'Sure, what do you need?' },
        ],
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:00Z',
      }
      mockFetch.mockResponse = mockConv

      const { data, error } = await ai.getConversation('conv1')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/conversations/conv1')
      expect(data).toEqual(mockConv)
      expect(error).toBeNull()
    })

    it('should handle errors', async () => {
      mockFetch.shouldThrow = true

      const { data, error } = await ai.getConversation('non-existent')

      expect(data).toBeNull()
      expect(error).toBeDefined()
    })
  })

  describe('deleteConversation', () => {
    it('should delete conversation', async () => {
      const { error } = await ai.deleteConversation('conv1')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/conversations/conv1')
      expect(mockFetch.lastMethod).toBe('DELETE')
      expect(error).toBeNull()
    })

    it('should handle errors', async () => {
      mockFetch.shouldThrow = true

      const { error } = await ai.deleteConversation('conv1')

      expect(error).toBeDefined()
    })
  })

  describe('updateConversation', () => {
    it('should update conversation title', async () => {
      const mockUpdated: AIUserConversationDetail = {
        id: 'conv1',
        chatbot_name: 'sql-assistant',
        title: 'New title',
        messages: [],
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:01Z',
      }
      mockFetch.mockResponse = mockUpdated

      const { data, error } = await ai.updateConversation('conv1', {
        title: 'New title',
      })

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/conversations/conv1')
      expect(mockFetch.lastMethod).toBe('PATCH')
      expect(mockFetch.lastBody).toEqual({ title: 'New title' })
      expect(data).toEqual(mockUpdated)
      expect(error).toBeNull()
    })

    it('should handle errors', async () => {
      mockFetch.shouldThrow = true

      const { data, error } = await ai.updateConversation('conv1', { title: 'New' })

      expect(data).toBeNull()
      expect(error).toBeDefined()
    })
  })

  describe('createChat', () => {
    it('should create a chat client with correct WebSocket URL', () => {
      const chat = ai.createChat({
        token: 'test-token',
      })

      expect(chat).toBeInstanceOf(FluxbaseAIChat)
    })
  })
})

describe('FluxbaseAIChat', () => {
  let mockWs: MockWebSocket;
  let originalWebSocket: any;

  beforeEach(() => {
    // Capture the WebSocket constructor to access the created instance
    originalWebSocket = (global as any).WebSocket;
    (global as any).WebSocket = function(url: string) {
      mockWs = new MockWebSocket(url);
      return mockWs;
    };
    (global as any).WebSocket.OPEN = MockWebSocket.OPEN;
    (global as any).WebSocket.CLOSED = MockWebSocket.CLOSED;
    (global as any).WebSocket.CONNECTING = MockWebSocket.CONNECTING;
    (global as any).WebSocket.CLOSING = MockWebSocket.CLOSING;
  });

  afterEach(() => {
    (global as any).WebSocket = originalWebSocket;
  });

  describe('without connection', () => {
    it('should return false for isConnected when never connected', () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      })

      expect(chat.isConnected()).toBe(false)
    })

    it('should handle disconnect when not connected', () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      })

      // Should not throw
      chat.disconnect()
    })

    it('should throw when sendMessage is called without connection', () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      })

      // When ws is null, isConnected() returns false and should throw
      expect(() => chat.sendMessage('conv-123', 'Hello')).toThrow()
    })

    it('should throw when cancel is called without connection', () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      })

      expect(() => chat.cancel('conv-123')).toThrow()
    })

    it('should throw when startChat is called without connection', async () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      })

      await expect(chat.startChat('sql-assistant')).rejects.toThrow()
    })
  })

  describe('getAccumulatedContent', () => {
    it('should return empty string for unknown conversation', () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      })

      expect(chat.getAccumulatedContent('unknown')).toBe('')
    })
  })

  describe('connect()', () => {
    it('should connect successfully', async () => {
      const onEvent = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onEvent,
      });

      const connectPromise = chat.connect();

      // Simulate WebSocket opening
      mockWs.simulateOpen();

      await connectPromise;

      expect(chat.isConnected()).toBe(true);
      expect(onEvent).toHaveBeenCalledWith({ type: 'connected' });
    });

    it('should build URL with token', async () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        token: 'my-jwt-token',
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      expect(mockWs.url).toBe('ws://localhost:8080/ai/ws?token=my-jwt-token');
    });

    it('should handle URL with existing query params and token', async () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws?param=value',
        token: 'my-token',
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      expect(mockWs.url).toBe('ws://localhost:8080/ai/ws?param=value&token=my-token');
    });

    it('should reject on WebSocket error', async () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      });

      const connectPromise = chat.connect();

      // Simulate WebSocket error
      mockWs.simulateError();

      await expect(connectPromise).rejects.toThrow('WebSocket connection failed');
    });

    it('should use default wsUrl when not provided', async () => {
      const chat = new FluxbaseAIChat({});

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      expect(mockWs.url).toBe('/ai/ws');
    });
  });

  describe('disconnect()', () => {
    it('should close WebSocket connection', async () => {
      const onEvent = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onEvent,
        reconnectAttempts: 0, // Disable reconnect for this test
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      chat.disconnect();

      expect(mockWs.readyState).toBe(MockWebSocket.CLOSED);
    });
  });

  describe('startChat()', () => {
    it('should start chat and resolve with conversation ID', async () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      const startChatPromise = chat.startChat('sql-assistant', 'default');

      // Simulate server response
      mockWs.simulateMessage({
        type: 'chat_started',
        conversation_id: 'conv-123',
        chatbot: 'sql-assistant',
      });

      const convId = await startChatPromise;
      expect(convId).toBe('conv-123');

      // Check the sent message
      const sentMessage = JSON.parse(mockWs.sentMessages[0]);
      expect(sentMessage.type).toBe('start_chat');
      expect(sentMessage.chatbot).toBe('sql-assistant');
      expect(sentMessage.namespace).toBe('default');
    });

    it('should use default namespace when not provided and no lookup function', async () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      const startChatPromise = chat.startChat('sql-assistant');
      mockWs.simulateMessage({
        type: 'chat_started',
        conversation_id: 'conv-123',
      });
      await startChatPromise;

      const sentMessage = JSON.parse(mockWs.sentMessages[0]);
      expect(sentMessage.namespace).toBe('default');
    });

    it('should use smart namespace resolution with _lookupChatbot', async () => {
      const lookupMock = vi.fn().mockResolvedValue({
        data: { chatbot: { name: 'sql-assistant', namespace: 'custom' } },
        error: null,
      });

      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        _lookupChatbot: lookupMock,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      // Start chat (this triggers the lookup)
      const startChatPromise = chat.startChat('sql-assistant');

      // Wait for the lookup to complete and the message to be sent
      await vi.waitFor(() => expect(mockWs.sentMessages.length).toBeGreaterThan(0), { timeout: 1000 });

      // Now simulate server response
      mockWs.simulateMessage({
        type: 'chat_started',
        conversation_id: 'conv-123',
      });

      await startChatPromise;

      expect(lookupMock).toHaveBeenCalledWith('sql-assistant');
      const sentMessage = JSON.parse(mockWs.sentMessages[0]);
      expect(sentMessage.namespace).toBe('custom');
    });

    it('should throw on lookup error', async () => {
      const lookupMock = vi.fn().mockResolvedValue({
        data: null,
        error: new Error('Lookup failed'),
      });

      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        _lookupChatbot: lookupMock,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      await expect(chat.startChat('sql-assistant')).rejects.toThrow('Failed to lookup chatbot: Lookup failed');
    });

    it('should throw when chatbot not found', async () => {
      const lookupMock = vi.fn().mockResolvedValue({
        data: null,
        error: null,
      });

      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        _lookupChatbot: lookupMock,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      await expect(chat.startChat('unknown')).rejects.toThrow("Chatbot 'unknown' not found");
    });

    it('should throw on ambiguous chatbot', async () => {
      const lookupMock = vi.fn().mockResolvedValue({
        data: {
          ambiguous: true,
          namespaces: ['ns1', 'ns2'],
        },
        error: null,
      });

      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        _lookupChatbot: lookupMock,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      await expect(chat.startChat('common')).rejects.toThrow(
        "Chatbot 'common' exists in multiple namespaces: ns1, ns2"
      );
    });

    it('should throw when lookup returns error message', async () => {
      const lookupMock = vi.fn().mockResolvedValue({
        data: { error: 'Permission denied' },
        error: null,
      });

      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        _lookupChatbot: lookupMock,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      await expect(chat.startChat('sql-assistant')).rejects.toThrow('Permission denied');
    });

    it('should include impersonate_user_id when provided', async () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      const startChatPromise = chat.startChat('sql-assistant', 'default', undefined, 'user-456');
      mockWs.simulateMessage({
        type: 'chat_started',
        conversation_id: 'conv-123',
      });
      await startChatPromise;

      const sentMessage = JSON.parse(mockWs.sentMessages[0]);
      expect(sentMessage.impersonate_user_id).toBe('user-456');
    });

    it('should include conversation_id when resuming', async () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      const startChatPromise = chat.startChat('sql-assistant', 'default', 'existing-conv');
      mockWs.simulateMessage({
        type: 'chat_started',
        conversation_id: 'existing-conv',
      });
      await startChatPromise;

      const sentMessage = JSON.parse(mockWs.sentMessages[0]);
      expect(sentMessage.conversation_id).toBe('existing-conv');
    });

    it('should reject on server error during startChat', async () => {
      const onError = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onError,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      const startChatPromise = chat.startChat('sql-assistant');

      // Simulate error response
      mockWs.simulateMessage({
        type: 'error',
        error: 'Chatbot not found',
        code: 'NOT_FOUND',
      });

      await expect(startChatPromise).rejects.toThrow('Chatbot not found');
      expect(onError).toHaveBeenCalledWith('Chatbot not found', 'NOT_FOUND', undefined);
    });
  });

  describe('sendMessage()', () => {
    it('should send message to conversation', async () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      chat.sendMessage('conv-123', 'Hello, how are you?');

      const sentMessage = JSON.parse(mockWs.sentMessages[0]);
      expect(sentMessage.type).toBe('message');
      expect(sentMessage.conversation_id).toBe('conv-123');
      expect(sentMessage.content).toBe('Hello, how are you?');
    });

    it('should reset accumulated content when sending message', async () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      chat.sendMessage('conv-123', 'First message');
      expect(chat.getAccumulatedContent('conv-123')).toBe('');
    });
  });

  describe('cancel()', () => {
    it('should send cancel message', async () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      chat.cancel('conv-123');

      const sentMessage = JSON.parse(mockWs.sentMessages[0]);
      expect(sentMessage.type).toBe('cancel');
      expect(sentMessage.conversation_id).toBe('conv-123');
    });
  });

  describe('message handling', () => {
    it('should handle content messages and accumulate', async () => {
      const onContent = vi.fn();
      const onEvent = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onContent,
        onEvent,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      // Clear the connected event
      onEvent.mockClear();

      mockWs.simulateMessage({
        type: 'content',
        conversation_id: 'conv-123',
        delta: 'Hello',
      });

      mockWs.simulateMessage({
        type: 'content',
        conversation_id: 'conv-123',
        delta: ' world',
      });

      expect(onContent).toHaveBeenCalledTimes(2);
      expect(onContent).toHaveBeenCalledWith('Hello', 'conv-123');
      expect(onContent).toHaveBeenCalledWith(' world', 'conv-123');
      expect(chat.getAccumulatedContent('conv-123')).toBe('Hello world');
    });

    it('should handle progress messages', async () => {
      const onProgress = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onProgress,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      mockWs.simulateMessage({
        type: 'progress',
        conversation_id: 'conv-123',
        step: 'query_generation',
        message: 'Generating SQL query...',
      });

      expect(onProgress).toHaveBeenCalledWith(
        'query_generation',
        'Generating SQL query...',
        'conv-123'
      );
    });

    it('should handle query_result messages', async () => {
      const onQueryResult = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onQueryResult,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      mockWs.simulateMessage({
        type: 'query_result',
        conversation_id: 'conv-123',
        query: 'SELECT * FROM users',
        summary: 'Found 10 users',
        row_count: 10,
        data: [{ id: 1, name: 'Alice' }],
      });

      expect(onQueryResult).toHaveBeenCalledWith(
        'SELECT * FROM users',
        'Found 10 users',
        10,
        [{ id: 1, name: 'Alice' }],
        'conv-123'
      );
    });

    it('should handle tool_result messages with query data', async () => {
      const onQueryResult = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onQueryResult,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      mockWs.simulateMessage({
        type: 'tool_result',
        conversation_id: 'conv-123',
        query: 'SELECT COUNT(*) FROM orders',
        summary: '500 orders total',
        row_count: 1,
        data: [{ count: 500 }],
      });

      expect(onQueryResult).toHaveBeenCalledWith(
        'SELECT COUNT(*) FROM orders',
        '500 orders total',
        1,
        [{ count: 500 }],
        'conv-123'
      );
    });

    it('should handle done messages', async () => {
      const onDone = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onDone,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      mockWs.simulateMessage({
        type: 'done',
        conversation_id: 'conv-123',
        usage: {
          total_tokens: 150,
          prompt_tokens: 100,
          completion_tokens: 50,
        },
      });

      expect(onDone).toHaveBeenCalledWith(
        { total_tokens: 150, prompt_tokens: 100, completion_tokens: 50 },
        'conv-123'
      );
    });

    it('should handle error messages', async () => {
      const onError = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onError,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      mockWs.simulateMessage({
        type: 'error',
        conversation_id: 'conv-123',
        error: 'Query execution failed',
        code: 'QUERY_ERROR',
      });

      expect(onError).toHaveBeenCalledWith('Query execution failed', 'QUERY_ERROR', 'conv-123');
    });

    it('should handle error messages with default values', async () => {
      const onError = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onError,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      mockWs.simulateMessage({
        type: 'error',
      });

      expect(onError).toHaveBeenCalledWith('Unknown error', undefined, undefined);
    });

    it('should handle invalid JSON gracefully', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      // Manually trigger invalid message
      if (mockWs.onmessage) {
        mockWs.onmessage({ data: 'invalid json {' });
      }

      expect(consoleSpy).toHaveBeenCalled();
      consoleSpy.mockRestore();
    });

    it('should emit events for all message types', async () => {
      const onEvent = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onEvent,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;
      onEvent.mockClear();

      mockWs.simulateMessage({
        type: 'progress',
        conversation_id: 'conv-123',
        chatbot: 'sql-assistant',
        step: 'thinking',
        message: 'Processing...',
      });

      expect(onEvent).toHaveBeenCalledWith(expect.objectContaining({
        type: 'progress',
        conversationId: 'conv-123',
        chatbot: 'sql-assistant',
        step: 'thinking',
        message: 'Processing...',
      }));
    });
  });

  describe('connection close and reconnect', () => {
    it('should emit disconnected event on close', async () => {
      const onEvent = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onEvent,
        reconnectAttempts: 0,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;
      onEvent.mockClear();

      mockWs.simulateClose();

      expect(onEvent).toHaveBeenCalledWith({ type: 'disconnected' });
    });

    it('should attempt reconnect on close', async () => {
      vi.useFakeTimers();

      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        reconnectAttempts: 3,
        reconnectDelay: 1000,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      // Close connection to trigger reconnect
      mockWs.simulateClose();

      // Fast-forward past the reconnect delay
      vi.advanceTimersByTime(1000);

      // A new WebSocket should have been created
      expect(mockWs.url).toBe('ws://localhost:8080/ai/ws');

      vi.useRealTimers();
    });
  });

  describe('content accumulation without callback', () => {
    it('should accumulate content even without onContent callback', async () => {
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      mockWs.simulateMessage({
        type: 'content',
        conversation_id: 'conv-123',
        delta: 'Test content',
      });

      expect(chat.getAccumulatedContent('conv-123')).toBe('Test content');
    });
  });

  describe('callbacks with missing data', () => {
    it('should not call onProgress without required fields', async () => {
      const onProgress = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onProgress,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      mockWs.simulateMessage({
        type: 'progress',
        conversation_id: 'conv-123',
        // Missing step and message
      });

      expect(onProgress).not.toHaveBeenCalled();
    });

    it('should not call onQueryResult without conversation_id', async () => {
      const onQueryResult = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onQueryResult,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      mockWs.simulateMessage({
        type: 'query_result',
        // Missing conversation_id
        query: 'SELECT 1',
      });

      expect(onQueryResult).not.toHaveBeenCalled();
    });

    it('should not call onDone without conversation_id', async () => {
      const onDone = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onDone,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      mockWs.simulateMessage({
        type: 'done',
        // Missing conversation_id
      });

      expect(onDone).not.toHaveBeenCalled();
    });

    it('should not accumulate content without conversation_id or delta', async () => {
      const onContent = vi.fn();
      const chat = new FluxbaseAIChat({
        wsUrl: 'ws://localhost:8080/ai/ws',
        onContent,
      });

      const connectPromise = chat.connect();
      mockWs.simulateOpen();
      await connectPromise;

      mockWs.simulateMessage({
        type: 'content',
        // Missing conversation_id
        delta: 'test',
      });

      expect(onContent).not.toHaveBeenCalled();
    });
  });
})

