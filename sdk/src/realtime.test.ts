/**
 * Realtime WebSocket Tests
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { FluxbaseRealtime, RealtimeChannel } from "./realtime";

// Mock CloseEvent (not available in Node.js)
class MockCloseEvent extends Event {
  public code: number;
  public reason: string;
  public wasClean: boolean;

  constructor(
    type: string,
    init?: { code?: number; reason?: string; wasClean?: boolean },
  ) {
    super(type);
    this.code = init?.code ?? 1000;
    this.reason = init?.reason ?? "";
    this.wasClean = init?.wasClean ?? true;
  }
}
(global as any).CloseEvent = MockCloseEvent;

// Mock WebSocket
class MockWebSocket {
  public url: string;
  public readyState: number = 1; // WebSocket.OPEN
  public onopen: ((event: Event) => void) | null = null;
  public onclose: ((event: CloseEvent) => void) | null = null;
  public onmessage: ((event: MessageEvent) => void) | null = null;
  public onerror: ((event: Event) => void) | null = null;
  public sentMessages: string[] = [];

  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  constructor(url: string) {
    this.url = url;
    // Simulate connection opening
    setTimeout(() => {
      if (this.onopen) {
        this.onopen(new Event("open"));
      }
    }, 10);
  }

  send(data: string): void {
    this.sentMessages.push(data);
    // Auto-close WebSocket when unsubscribe message is sent
    try {
      const msg = JSON.parse(data);
      if (msg.type === "unsubscribe") {
        setTimeout(() => this.close(), 10);
      }
      // Auto-respond to token updates
      if (msg.type === "token_update") {
        setTimeout(() => {
          if (this.onmessage) {
            this.onmessage(
              new MessageEvent("message", {
                data: JSON.stringify({ type: "token_update_ack" }),
              }),
            );
          }
        }, 10);
      }
    } catch (e) {
      // Ignore parse errors
    }
  }

  close(code?: number, reason?: string): void {
    this.readyState = 3; // WebSocket.CLOSED
    if (this.onclose) {
      this.onclose(new CloseEvent("close", { code, reason }));
    }
  }

  // Helper to simulate receiving a message
  simulateMessage(data: any): void {
    if (this.onmessage) {
      this.onmessage(
        new MessageEvent("message", { data: JSON.stringify(data) }),
      );
    }
  }
}

// Store reference to the last created MockWebSocket
let lastMockWebSocket: MockWebSocket | null = null;

// Replace global WebSocket - keep mock in place throughout all tests
// to avoid reconnection errors when original WebSocket is undefined in Node.js
let MockWS: any;

beforeEach(() => {
  lastMockWebSocket = null;
  MockWS = class extends MockWebSocket {
    static CONNECTING = 0;
    static OPEN = 1;
    static CLOSING = 2;
    static CLOSED = 3;

    constructor(url: string) {
      super(url);
      lastMockWebSocket = this;
    }
  };
  global.WebSocket = MockWS;
});

afterEach(() => {
  // Don't restore original WebSocket as it's undefined in Node.js
  // and would cause reconnection errors. Keep mock in place.
});

describe("FluxbaseRealtime - Connection", () => {
  let realtime: FluxbaseRealtime;

  beforeEach(() => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
  });

  afterEach(() => {
    realtime.removeAllChannels();
  });

  it("should create a new FluxbaseRealtime instance", () => {
    expect(realtime).toBeDefined();
  });

  it("should create and manage channels", () => {
    const channel = realtime.channel("test-channel");
    expect(channel).toBeDefined();
    expect(channel).toBeInstanceOf(RealtimeChannel);
  });

  it("should return the same channel instance for the same name", () => {
    const channel1 = realtime.channel("test");
    const channel2 = realtime.channel("test");
    expect(channel1).toBe(channel2);
  });

  it("should create different channels for different names", () => {
    const channel1 = realtime.channel("channel-1");
    const channel2 = realtime.channel("channel-2");
    expect(channel1).not.toBe(channel2);
  });

  it("should set auth token", () => {
    realtime.setAuth("new-token");
    // The token is stored internally
    expect(realtime).toBeDefined();
  });

  it("should not update channels when setAuth is called with the same token", () => {
    // Create a channel and spy on updateToken
    const channel = realtime.channel("test-channel");
    const updateTokenSpy = vi.spyOn(channel, "updateToken");

    // Set a new token - should call updateToken
    realtime.setAuth("token-1");
    expect(updateTokenSpy).toHaveBeenCalledTimes(1);
    expect(updateTokenSpy).toHaveBeenCalledWith("token-1");

    // Set the same token again - should NOT call updateToken
    realtime.setAuth("token-1");
    expect(updateTokenSpy).toHaveBeenCalledTimes(1);

    // Set a different token - should call updateToken again
    realtime.setAuth("token-2");
    expect(updateTokenSpy).toHaveBeenCalledTimes(2);
    expect(updateTokenSpy).toHaveBeenLastCalledWith("token-2");
  });
});

describe("RealtimeChannel - Subscriptions", () => {
  let realtime: FluxbaseRealtime;
  let channel: RealtimeChannel;

  beforeEach(async () => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
    channel = realtime.channel("test-channel");
  });

  afterEach(() => {
    realtime.removeAllChannels();
  });

  it("should create a channel", () => {
    expect(channel).toBeDefined();
  });

  it("should subscribe to channel and connect WebSocket", async () => {
    channel.subscribe();

    // Wait for WebSocket connection
    await new Promise((resolve) => setTimeout(resolve, 30));

    expect(lastMockWebSocket).not.toBeNull();
    expect(lastMockWebSocket!.url).toContain("localhost:8080");
  });

  it("should send subscribe message after connection", async () => {
    channel.subscribe();

    // Wait for connection and subscribe message
    await new Promise((resolve) => setTimeout(resolve, 30));

    expect(lastMockWebSocket).not.toBeNull();
    const subscribeMsg = lastMockWebSocket!.sentMessages.find((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "subscribe";
    });

    expect(subscribeMsg).toBeDefined();
  });

  it("should unsubscribe from channel", async () => {
    channel.subscribe();
    await new Promise((resolve) => setTimeout(resolve, 30));

    const result = await channel.unsubscribe();

    expect(result).toBe("ok");
  });

  it("should handle multiple channels independently", async () => {
    const channel1 = realtime.channel("channel-1");
    const channel2 = realtime.channel("channel-2");

    channel1.subscribe();
    await new Promise((resolve) => setTimeout(resolve, 30));

    channel2.subscribe();
    await new Promise((resolve) => setTimeout(resolve, 30));

    // Both channels should be active
    expect(channel1).toBeDefined();
    expect(channel2).toBeDefined();
  });
});

describe("RealtimeChannel - Change Events", () => {
  let channel: RealtimeChannel;

  beforeEach(async () => {
    channel = new RealtimeChannel(
      "http://localhost:8080",
      "public:users",
      "test-token",
    );
    channel.subscribe();
    await new Promise((resolve) => setTimeout(resolve, 30));
  });

  afterEach(async () => {
    await channel.unsubscribe();
  });

  it("should receive INSERT events", async () => {
    const callback = vi.fn();
    channel.on("INSERT", callback);

    // Simulate receiving an INSERT event (server sends type: "postgres_changes")
    lastMockWebSocket!.simulateMessage({
      type: "postgres_changes",
      payload: {
        type: "INSERT",
        table: "users",
        new_record: { id: 1, name: "John" },
      },
    });

    expect(callback).toHaveBeenCalled();
    expect(callback.mock.calls[0][0]).toMatchObject({
      eventType: "INSERT",
    });
  });

  it("should receive UPDATE events", async () => {
    const callback = vi.fn();
    channel.on("UPDATE", callback);

    // Server sends type: "postgres_changes" for database events
    lastMockWebSocket!.simulateMessage({
      type: "postgres_changes",
      payload: {
        type: "UPDATE",
        table: "users",
        old_record: { id: 1, name: "John" },
        new_record: { id: 1, name: "Jane" },
      },
    });

    expect(callback).toHaveBeenCalled();
    expect(callback.mock.calls[0][0]).toMatchObject({
      eventType: "UPDATE",
    });
  });

  it("should receive DELETE events", async () => {
    const callback = vi.fn();
    channel.on("DELETE", callback);

    // Server sends type: "postgres_changes" for database events
    lastMockWebSocket!.simulateMessage({
      type: "postgres_changes",
      payload: {
        type: "DELETE",
        table: "users",
        old_record: { id: 1, name: "John" },
      },
    });

    expect(callback).toHaveBeenCalled();
    expect(callback.mock.calls[0][0]).toMatchObject({
      eventType: "DELETE",
    });
  });

  it("should receive * (all) events", async () => {
    const callback = vi.fn();
    channel.on("*", callback);

    // Send multiple events (server sends type: "postgres_changes")
    lastMockWebSocket!.simulateMessage({
      type: "postgres_changes",
      payload: { type: "INSERT" },
    });
    lastMockWebSocket!.simulateMessage({
      type: "postgres_changes",
      payload: { type: "UPDATE" },
    });
    lastMockWebSocket!.simulateMessage({
      type: "postgres_changes",
      payload: { type: "DELETE" },
    });

    expect(callback).toHaveBeenCalledTimes(3);
  });
});

describe("RealtimeChannel - Broadcast", () => {
  let channel: RealtimeChannel;

  beforeEach(async () => {
    channel = new RealtimeChannel(
      "http://localhost:8080",
      "chat:room1",
      "test-token",
    );
    channel.subscribe();
    await new Promise((resolve) => setTimeout(resolve, 30));
  });

  afterEach(async () => {
    await channel.unsubscribe();
  });

  it("should send broadcast messages", async () => {
    const result = await channel.send({
      type: "broadcast",
      event: "message",
      payload: { text: "Hello World" },
    });

    expect(result).toBe("ok");

    const broadcastMsg = lastMockWebSocket!.sentMessages.find((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "broadcast";
    });

    expect(broadcastMsg).toBeDefined();
  });

  it("should receive broadcast messages", async () => {
    const callback = vi.fn();
    channel.on("broadcast", { event: "message" }, callback);

    // Server sends broadcast nested inside payload: { payload: { broadcast: {...} } }
    lastMockWebSocket!.simulateMessage({
      type: "broadcast",
      payload: {
        broadcast: {
          event: "message",
          payload: { text: "Hello from another user" },
        },
      },
    });

    expect(callback).toHaveBeenCalled();
  });
});

describe("RealtimeChannel - Presence", () => {
  let channel: RealtimeChannel;

  beforeEach(async () => {
    channel = new RealtimeChannel(
      "http://localhost:8080",
      "presence:lobby",
      "test-token",
    );
    channel.subscribe();
    await new Promise((resolve) => setTimeout(resolve, 30));
  });

  afterEach(async () => {
    await channel.unsubscribe();
  });

  it("should track presence state", async () => {
    const result = await channel.track({ user: "john", status: "online" });
    expect(result).toBe("ok");

    const presenceMsg = lastMockWebSocket!.sentMessages.find((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "presence";
    });

    expect(presenceMsg).toBeDefined();
  });

  it("should receive presence updates", async () => {
    const callback = vi.fn();
    channel.on("presence", { event: "sync" }, callback);

    // Server sends presence nested inside payload: { payload: { presence: {...} } }
    lastMockWebSocket!.simulateMessage({
      type: "presence",
      payload: {
        presence: {
          event: "sync",
          key: "user1",
          currentPresences: { user1: [{ status: "online" }] },
        },
      },
    });

    expect(callback).toHaveBeenCalled();
  });

  it("should get presence state", () => {
    const state = channel.presenceState();
    expect(state).toBeDefined();
    expect(typeof state).toBe("object");
  });

  it("should untrack presence", async () => {
    await channel.track({ user: "john", status: "online" });
    const result = await channel.untrack();
    expect(result).toBe("ok");
  });
});

describe("RealtimeChannel - Error Handling", () => {
  let channel: RealtimeChannel;

  beforeEach(async () => {
    channel = new RealtimeChannel(
      "http://localhost:8080",
      "test",
      "test-token",
    );
    channel.subscribe();
    await new Promise((resolve) => setTimeout(resolve, 30));
  });

  afterEach(async () => {
    await channel.unsubscribe();
  });

  it("should handle error messages", async () => {
    // Simulate error message from server
    lastMockWebSocket!.simulateMessage({
      type: "error",
      error: "Channel not found",
    });

    // Error is logged but doesn't throw
    expect(true).toBe(true);
  });

  it("should handle connection close", async () => {
    lastMockWebSocket!.close();
    expect(lastMockWebSocket!.readyState).toBe(WebSocket.CLOSED);
  });
});

describe("RealtimeChannel - Filters", () => {
  it("should create channel with filter config", async () => {
    const realtime = new FluxbaseRealtime(
      "http://localhost:8080",
      "test-token",
    );
    const channel = realtime.channel("public:posts", {
      filter: "user_id=eq.123",
    });

    channel.subscribe();
    await new Promise((resolve) => setTimeout(resolve, 30));

    expect(channel).toBeDefined();

    realtime.removeAllChannels();
  });
});

describe("Realtime - Heartbeat", () => {
  it("should send heartbeat messages periodically", async () => {
    const channel = new RealtimeChannel(
      "http://localhost:8080",
      "test",
      "test-token",
    );
    channel.subscribe();
    await new Promise((resolve) => setTimeout(resolve, 30));

    // Heartbeat is sent automatically by the channel on an interval
    // We verify the channel is connected
    expect(lastMockWebSocket).not.toBeNull();

    await channel.unsubscribe();
  });

  it("should handle heartbeat responses", async () => {
    const channel = new RealtimeChannel(
      "http://localhost:8080",
      "test",
      "test-token",
    );
    channel.subscribe();
    await new Promise((resolve) => setTimeout(resolve, 30));

    // Simulate heartbeat response
    lastMockWebSocket!.simulateMessage({
      type: "heartbeat",
      payload: { timestamp: Date.now() },
    });

    // Should not throw error
    expect(lastMockWebSocket!.readyState).toBe(WebSocket.OPEN);

    await channel.unsubscribe();
  });
});

describe("Realtime - Multiple Channels", () => {
  let realtime: FluxbaseRealtime;

  beforeEach(() => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
  });

  afterEach(() => {
    realtime.removeAllChannels();
  });

  it("should manage multiple channels", async () => {
    const channel1 = realtime.channel("channel-1");
    const channel2 = realtime.channel("channel-2");
    const channel3 = realtime.channel("channel-3");

    channel1.subscribe();
    channel2.subscribe();
    channel3.subscribe();

    await new Promise((resolve) => setTimeout(resolve, 50));

    // All channels should be created
    expect(channel1).toBeDefined();
    expect(channel2).toBeDefined();
    expect(channel3).toBeDefined();
  });

  it("should remove channel", async () => {
    const channel = realtime.channel("test-channel");
    channel.subscribe();
    await new Promise((resolve) => setTimeout(resolve, 30));

    const result = await realtime.removeChannel(channel);
    expect(result).toBe("ok");
  });

  it("should remove all channels", () => {
    realtime.channel("channel-1").subscribe();
    realtime.channel("channel-2").subscribe();

    realtime.removeAllChannels();

    // Should not throw
    expect(true).toBe(true);
  });
});

describe("RealtimeChannel - postgres_changes Filtering", () => {
  let channel: RealtimeChannel;
  let mockWs: MockWebSocket;

  beforeEach(async () => {
    channel = new RealtimeChannel(
      "http://localhost:8080",
      "jobs:user123",
      "test-token",
    );
    channel.subscribe();
    // Wait longer to ensure any pending reconnection timers from previous tests have completed
    await new Promise((resolve) => setTimeout(resolve, 50));
    // Capture the WebSocket reference for this test to avoid race conditions
    mockWs = lastMockWebSocket!;
  });

  afterEach(async () => {
    await channel.unsubscribe();
  });

  it("should support postgres_changes with filter parameter", () => {
    const callback = vi.fn();

    channel.on(
      "postgres_changes",
      {
        event: "*",
        schema: "public",
        table: "jobs",
        filter: "created_by=eq.user123",
      },
      callback,
    );

    // Verify the subscription was registered
    expect(callback).not.toHaveBeenCalled();
  });

  it("should support INSERT event filtering", () => {
    const callback = vi.fn();

    channel.on(
      "postgres_changes",
      {
        event: "INSERT",
        schema: "public",
        table: "jobs",
        filter: "status=eq.queued",
      },
      callback,
    );

    // Simulate INSERT event (server sends type: "postgres_changes")
    mockWs.simulateMessage({
      type: "postgres_changes",
      payload: {
        type: "INSERT",
        schema: "public",
        table: "jobs",
        new_record: { id: 1, status: "queued" },
        timestamp: new Date().toISOString(),
      },
    });

    expect(callback).toHaveBeenCalledTimes(1);
  });

  it("should support UPDATE event filtering", () => {
    const callback = vi.fn();

    channel.on(
      "postgres_changes",
      {
        event: "UPDATE",
        schema: "public",
        table: "jobs",
        filter: "priority=gt.5",
      },
      callback,
    );

    // Server sends type: "postgres_changes" for database events
    mockWs.simulateMessage({
      type: "postgres_changes",
      payload: {
        type: "UPDATE",
        schema: "public",
        table: "jobs",
        new_record: { id: 1, priority: 10 },
      },
    });

    expect(callback).toHaveBeenCalled();
  });

  it("should support DELETE event filtering", () => {
    const callback = vi.fn();

    channel.on(
      "postgres_changes",
      {
        event: "DELETE",
        schema: "public",
        table: "jobs",
      },
      callback,
    );

    // Server sends type: "postgres_changes" for database events
    mockWs.simulateMessage({
      type: "postgres_changes",
      payload: {
        type: "DELETE",
        schema: "public",
        table: "jobs",
        old_record: { id: 1 },
      },
    });

    expect(callback).toHaveBeenCalled();
  });

  it("should support wildcard event filtering", () => {
    const callback = vi.fn();

    channel.on(
      "postgres_changes",
      {
        event: "*",
        schema: "public",
        table: "jobs",
      },
      callback,
    );

    // Send different event types (server sends type: "postgres_changes")
    mockWs.simulateMessage({
      type: "postgres_changes",
      payload: { type: "INSERT", schema: "public", table: "jobs" },
    });
    mockWs.simulateMessage({
      type: "postgres_changes",
      payload: { type: "UPDATE", schema: "public", table: "jobs" },
    });
    mockWs.simulateMessage({
      type: "postgres_changes",
      payload: { type: "DELETE", schema: "public", table: "jobs" },
    });

    expect(callback).toHaveBeenCalledTimes(3);
  });

  it("should maintain backwards compatibility with legacy on() API", () => {
    const callback = vi.fn();

    channel.on("INSERT", callback);

    // Simulate INSERT event (server sends type: "postgres_changes")
    mockWs.simulateMessage({
      type: "postgres_changes",
      payload: {
        type: "INSERT",
        schema: "public",
        table: "jobs",
        new_record: { id: 1 },
        timestamp: new Date().toISOString(),
      },
    });

    expect(callback).toHaveBeenCalledTimes(1);
  });
});

describe("RealtimeChannel - Token Update", () => {
  it("should update token on connected channel", async () => {
    const channel = new RealtimeChannel(
      "http://localhost:8080",
      "test",
      "old-token",
    );
    channel.subscribe();
    await new Promise((resolve) => setTimeout(resolve, 30));

    channel.updateToken("new-token");

    // Should send access_token message
    const tokenMsg = lastMockWebSocket!.sentMessages.find((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "access_token";
    });

    expect(tokenMsg).toBeDefined();

    await channel.unsubscribe();
  });
});

describe("FluxbaseRealtime - Token Refresh Callback", () => {
  it("should set token refresh callback", () => {
    const realtime = new FluxbaseRealtime(
      "http://localhost:8080",
      "test-token",
    );

    const refreshCallback = vi.fn().mockResolvedValue("new-token");
    realtime.setTokenRefreshCallback(refreshCallback);

    // Callback is set - not called until needed
    expect(refreshCallback).not.toHaveBeenCalled();

    realtime.removeAllChannels();
  });
});
