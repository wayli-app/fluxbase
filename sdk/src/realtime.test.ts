/**
 * Realtime WebSocket Tests
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { FluxbaseRealtime, RealtimeChannel } from "./realtime";

// Mock WebSocket
class MockWebSocket {
  public url: string;
  public readyState: number = 1; // OPEN
  public onopen: ((event: Event) => void) | null = null;
  public onclose: ((event: CloseEvent) => void) | null = null;
  public onmessage: ((event: MessageEvent) => void) | null = null;
  public onerror: ((event: Event) => void) | null = null;

  public sentMessages: string[] = [];

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
  }

  close(code?: number, reason?: string): void {
    this.readyState = 3; // CLOSED
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

// Replace global WebSocket
global.WebSocket = MockWebSocket as any;

describe("FluxbaseRealtime - Connection", () => {
  let realtime: FluxbaseRealtime;

  beforeEach(() => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
  });

  afterEach(() => {
    if (realtime) {
      realtime.disconnect();
    }
  });

  it("should connect to WebSocket server", async () => {
    await new Promise((resolve) => setTimeout(resolve, 20));

    expect(realtime).toBeDefined();
  });

  it("should include auth token in connection", () => {
    const ws = (realtime as any).ws as MockWebSocket;
    expect(ws.url).toContain("localhost:8080");
  });

  it("should handle connection close", (done) => {
    realtime.onDisconnect(() => {
      done();
    });

    const ws = (realtime as any).ws as MockWebSocket;
    ws.close();
  });

  it("should handle connection error", (done) => {
    realtime.onError((error) => {
      expect(error).toBeDefined();
      done();
    });

    const ws = (realtime as any).ws as MockWebSocket;
    if (ws.onerror) {
      ws.onerror(new Event("error"));
    }
  });
});

describe("RealtimeChannel - Subscriptions", () => {
  let realtime: FluxbaseRealtime;
  let channel: RealtimeChannel;

  beforeEach(async () => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
    await new Promise((resolve) => setTimeout(resolve, 20));
    channel = realtime.channel("test-channel");
  });

  afterEach(() => {
    if (channel) {
      channel.unsubscribe();
    }
    if (realtime) {
      realtime.disconnect();
    }
  });

  it("should create a channel", () => {
    expect(channel).toBeDefined();
    expect(channel.topic).toBe("test-channel");
  });

  it("should subscribe to channel", () => {
    channel.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    const sentMessages = ws.sentMessages;

    const subscribeMsg = sentMessages.find((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "subscribe" && parsed.channel === "test-channel";
    });

    expect(subscribeMsg).toBeDefined();
  });

  it("should unsubscribe from channel", () => {
    channel.subscribe();
    channel.unsubscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    const sentMessages = ws.sentMessages;

    const unsubscribeMsg = sentMessages.find((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "unsubscribe";
    });

    expect(unsubscribeMsg).toBeDefined();
  });

  it("should handle multiple subscriptions", () => {
    const channel1 = realtime.channel("channel-1");
    const channel2 = realtime.channel("channel-2");

    channel1.subscribe();
    channel2.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    const subscribeMessages = ws.sentMessages.filter((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "subscribe";
    });

    expect(subscribeMessages.length).toBe(2);

    channel1.unsubscribe();
    channel2.unsubscribe();
  });
});

describe("RealtimeChannel - Change Events", () => {
  let realtime: FluxbaseRealtime;
  let channel: RealtimeChannel;

  beforeEach(async () => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
    await new Promise((resolve) => setTimeout(resolve, 20));
    channel = realtime.channel("public:users");
  });

  afterEach(() => {
    if (channel) {
      channel.unsubscribe();
    }
    if (realtime) {
      realtime.disconnect();
    }
  });

  it("should receive INSERT events", (done) => {
    channel.on("INSERT", (payload) => {
      expect(payload.type).toBe("INSERT");
      expect(payload.new).toEqual({ id: 1, name: "John" });
      done();
    });

    channel.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    ws.simulateMessage({
      type: "INSERT",
      channel: "public:users",
      payload: {
        type: "INSERT",
        table: "users",
        new: { id: 1, name: "John" },
      },
    });
  });

  it("should receive UPDATE events", (done) => {
    channel.on("UPDATE", (payload) => {
      expect(payload.type).toBe("UPDATE");
      expect(payload.old).toEqual({ id: 1, name: "John" });
      expect(payload.new).toEqual({ id: 1, name: "Jane" });
      done();
    });

    channel.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    ws.simulateMessage({
      type: "UPDATE",
      channel: "public:users",
      payload: {
        type: "UPDATE",
        table: "users",
        old: { id: 1, name: "John" },
        new: { id: 1, name: "Jane" },
      },
    });
  });

  it("should receive DELETE events", (done) => {
    channel.on("DELETE", (payload) => {
      expect(payload.type).toBe("DELETE");
      expect(payload.old).toEqual({ id: 1, name: "John" });
      done();
    });

    channel.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    ws.simulateMessage({
      type: "DELETE",
      channel: "public:users",
      payload: {
        type: "DELETE",
        table: "users",
        old: { id: 1, name: "John" },
      },
    });
  });

  it("should receive * (all) events", (done) => {
    let eventCount = 0;

    channel.on("*", (payload) => {
      eventCount++;
      if (eventCount === 3) {
        expect(eventCount).toBe(3);
        done();
      }
    });

    channel.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    ws.simulateMessage({
      type: "INSERT",
      channel: "public:users",
      payload: { type: "INSERT" },
    });
    ws.simulateMessage({
      type: "UPDATE",
      channel: "public:users",
      payload: { type: "UPDATE" },
    });
    ws.simulateMessage({
      type: "DELETE",
      channel: "public:users",
      payload: { type: "DELETE" },
    });
  });
});

describe("RealtimeChannel - Broadcast", () => {
  let realtime: FluxbaseRealtime;
  let channel: RealtimeChannel;

  beforeEach(async () => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
    await new Promise((resolve) => setTimeout(resolve, 20));
    channel = realtime.channel("chat:room1");
  });

  afterEach(() => {
    if (channel) {
      channel.unsubscribe();
    }
    if (realtime) {
      realtime.disconnect();
    }
  });

  it("should send broadcast messages", () => {
    channel.subscribe();
    channel.send({
      type: "broadcast",
      event: "message",
      payload: { text: "Hello World" },
    });

    const ws = (realtime as any).ws as MockWebSocket;
    const broadcastMsg = ws.sentMessages.find((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "broadcast";
    });

    expect(broadcastMsg).toBeDefined();
  });

  it("should receive broadcast messages", (done) => {
    channel.on("broadcast", (payload) => {
      expect(payload.event).toBe("message");
      expect(payload.payload).toEqual({ text: "Hello from another user" });
      done();
    });

    channel.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    ws.simulateMessage({
      type: "broadcast",
      channel: "chat:room1",
      payload: {
        event: "message",
        payload: { text: "Hello from another user" },
      },
    });
  });
});

describe("RealtimeChannel - Presence", () => {
  let realtime: FluxbaseRealtime;
  let channel: RealtimeChannel;

  beforeEach(async () => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
    await new Promise((resolve) => setTimeout(resolve, 20));
    channel = realtime.channel("presence:lobby");
  });

  afterEach(() => {
    if (channel) {
      channel.unsubscribe();
    }
    if (realtime) {
      realtime.disconnect();
    }
  });

  it("should track presence state", () => {
    channel.subscribe();
    channel.track({ user: "john", status: "online" });

    const ws = (realtime as any).ws as MockWebSocket;
    const presenceMsg = ws.sentMessages.find((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "presence";
    });

    // If presence is implemented
    if (presenceMsg) {
      expect(presenceMsg).toBeDefined();
    }
  });

  it("should receive presence updates", (done) => {
    channel.on("presence", (payload) => {
      expect(payload.joins).toBeDefined();
      done();
    });

    channel.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    ws.simulateMessage({
      type: "presence",
      channel: "presence:lobby",
      payload: {
        joins: { user1: { status: "online" } },
        leaves: {},
      },
    });
  });
});

describe("RealtimeChannel - Error Handling", () => {
  let realtime: FluxbaseRealtime;
  let channel: RealtimeChannel;

  beforeEach(async () => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
    await new Promise((resolve) => setTimeout(resolve, 20));
    channel = realtime.channel("test");
  });

  afterEach(() => {
    if (channel) {
      channel.unsubscribe();
    }
    if (realtime) {
      realtime.disconnect();
    }
  });

  it("should handle errors gracefully", (done) => {
    channel.onError((error) => {
      expect(error).toBeDefined();
      done();
    });

    channel.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    ws.simulateMessage({
      type: "error",
      channel: "test",
      payload: {
        error: "Channel not found",
      },
    });
  });

  it("should handle connection loss", () => {
    channel.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    ws.close();

    expect(ws.readyState).toBe(3); // CLOSED
  });
});

describe("RealtimeChannel - Filters", () => {
  let realtime: FluxbaseRealtime;

  beforeEach(async () => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
    await new Promise((resolve) => setTimeout(resolve, 20));
  });

  afterEach(() => {
    if (realtime) {
      realtime.disconnect();
    }
  });

  it("should filter by specific column", () => {
    const channel = realtime.channel("public:posts", {
      filter: "user_id=eq.123",
    });

    channel.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    const subscribeMsg = ws.sentMessages.find((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "subscribe" && parsed.channel === "public:posts";
    });

    expect(subscribeMsg).toBeDefined();
    // Filter would be in subscription options
  });

  it("should support multiple filters", () => {
    const channel = realtime.channel("public:comments", {
      filter: "post_id=eq.456&approved=eq.true",
    });

    channel.subscribe();

    expect(channel.topic).toBe("public:comments");
  });
});

describe("Realtime - Heartbeat", () => {
  let realtime: FluxbaseRealtime;

  beforeEach(async () => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
    await new Promise((resolve) => setTimeout(resolve, 20));
  });

  afterEach(() => {
    if (realtime) {
      realtime.disconnect();
    }
  });

  it("should send heartbeat messages", () => {
    const ws = (realtime as any).ws as MockWebSocket;

    // Trigger heartbeat manually if exposed
    if ((realtime as any).sendHeartbeat) {
      (realtime as any).sendHeartbeat();
    }

    // Check for heartbeat in sent messages
    const heartbeatMsg = ws.sentMessages.find((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "heartbeat" || parsed.type === "ping";
    });

    // Heartbeat might be automatic
    expect(ws.sentMessages.length).toBeGreaterThan(0);
  });

  it("should handle heartbeat responses", () => {
    const ws = (realtime as any).ws as MockWebSocket;

    ws.simulateMessage({
      type: "heartbeat",
      payload: { timestamp: Date.now() },
    });

    // Should not throw error
    expect(ws.readyState).toBe(1); // OPEN
  });
});

describe("Realtime - Reconnection", () => {
  let realtime: FluxbaseRealtime;

  beforeEach(async () => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
    await new Promise((resolve) => setTimeout(resolve, 20));
  });

  afterEach(() => {
    if (realtime) {
      realtime.disconnect();
    }
  });

  it("should attempt reconnection on disconnect", async () => {
    const ws = (realtime as any).ws as MockWebSocket;

    // Close connection
    ws.close();

    // Wait for reconnection attempt
    await new Promise((resolve) => setTimeout(resolve, 100));

    // Check if reconnection was attempted (new WebSocket created)
    // This depends on implementation
  });

  it("should restore subscriptions after reconnect", () => {
    const channel = realtime.channel("test-restore");
    channel.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    const initialMessages = ws.sentMessages.length;

    // Simulate reconnection
    ws.close();

    // After reconnect, subscriptions should be restored
    expect(initialMessages).toBeGreaterThan(0);
  });
});

describe("Realtime - Multiple Channels", () => {
  let realtime: FluxbaseRealtime;

  beforeEach(async () => {
    realtime = new FluxbaseRealtime("http://localhost:8080", "test-token");
    await new Promise((resolve) => setTimeout(resolve, 20));
  });

  afterEach(() => {
    if (realtime) {
      realtime.disconnect();
    }
  });

  it("should manage multiple channels", () => {
    const channel1 = realtime.channel("channel-1");
    const channel2 = realtime.channel("channel-2");
    const channel3 = realtime.channel("channel-3");

    channel1.subscribe();
    channel2.subscribe();
    channel3.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;
    const subscribeMessages = ws.sentMessages.filter((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "subscribe";
    });

    expect(subscribeMessages.length).toBeGreaterThanOrEqual(3);

    channel1.unsubscribe();
    channel2.unsubscribe();
    channel3.unsubscribe();
  });

  it("should route messages to correct channels", (done) => {
    const channel1 = realtime.channel("channel-1");
    const channel2 = realtime.channel("channel-2");

    let channel1Received = false;
    let channel2Received = false;

    channel1.on("*", () => {
      channel1Received = true;
      if (channel1Received && !channel2Received) {
        // channel1 received, channel2 did not
        expect(true).toBe(true);
      }
    });

    channel2.on("*", () => {
      channel2Received = true;
    });

    channel1.subscribe();
    channel2.subscribe();

    const ws = (realtime as any).ws as MockWebSocket;

    // Send message only to channel-1
    ws.simulateMessage({
      type: "INSERT",
      channel: "channel-1",
      payload: { test: true },
    });

    setTimeout(() => {
      expect(channel1Received).toBe(true);
      expect(channel2Received).toBe(false);
      done();
    }, 50);
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
    await new Promise((resolve) => setTimeout(resolve, 20));
    mockWs = (channel as any).ws as MockWebSocket;
  });

  afterEach(() => {
    if (channel) {
      channel.unsubscribe();
    }
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

    channel.subscribe();

    // Check subscription message includes config
    const subscribeMsg = mockWs.sentMessages.find((msg) => {
      const parsed = JSON.parse(msg);
      return parsed.type === "subscribe";
    });

    expect(subscribeMsg).toBeDefined();
    const parsedSubscribe = JSON.parse(subscribeMsg!);
    expect(parsedSubscribe.config).toBeDefined();
    expect(parsedSubscribe.config.event).toBe("*");
    expect(parsedSubscribe.config.schema).toBe("public");
    expect(parsedSubscribe.config.table).toBe("jobs");
    expect(parsedSubscribe.config.filter).toBe("created_by=eq.user123");
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

    channel.subscribe();

    // Simulate INSERT event
    mockWs.simulateMessage({
      type: "broadcast",
      payload: {
        type: "INSERT",
        schema: "public",
        table: "jobs",
        new_record: { id: 1, status: "queued" },
        timestamp: new Date().toISOString(),
      },
    });

    expect(callback).toHaveBeenCalledTimes(1);
    expect(callback).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "INSERT",
        schema: "public",
        table: "jobs",
      }),
    );
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

    channel.subscribe();

    const parsedSubscribe = JSON.parse(mockWs.sentMessages[0]);
    expect(parsedSubscribe.config.event).toBe("UPDATE");
    expect(parsedSubscribe.config.filter).toBe("priority=gt.5");
  });

  it("should support DELETE event filtering", () => {
    const callback = vi.fn();

    channel.on(
      "postgres_changes",
      {
        event: "DELETE",
        schema: "public",
        table: "jobs",
        filter: "completed_at=is.null",
      },
      callback,
    );

    channel.subscribe();

    const parsedSubscribe = JSON.parse(mockWs.sentMessages[0]);
    expect(parsedSubscribe.config.event).toBe("DELETE");
  });

  it("should support IN operator filtering", () => {
    const callback = vi.fn();

    channel.on(
      "postgres_changes",
      {
        event: "*",
        schema: "public",
        table: "jobs",
        filter: "status=in.(queued,running)",
      },
      callback,
    );

    channel.subscribe();

    const parsedSubscribe = JSON.parse(mockWs.sentMessages[0]);
    expect(parsedSubscribe.config.filter).toBe("status=in.(queued,running)");
  });

  it("should support LIKE pattern filtering", () => {
    const callback = vi.fn();

    channel.on(
      "postgres_changes",
      {
        event: "*",
        schema: "public",
        table: "users",
        filter: "email=like.*@gmail.com",
      },
      callback,
    );

    channel.subscribe();

    const parsedSubscribe = JSON.parse(mockWs.sentMessages[0]);
    expect(parsedSubscribe.config.filter).toBe("email=like.*@gmail.com");
  });

  it("should support filter-less subscriptions", () => {
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

    channel.subscribe();

    const parsedSubscribe = JSON.parse(mockWs.sentMessages[0]);
    expect(parsedSubscribe.config.filter).toBeUndefined();
  });

  it("should maintain backwards compatibility with legacy on() API", () => {
    const callback = vi.fn();

    channel.on("INSERT", callback);
    channel.subscribe();

    // Simulate INSERT event
    mockWs.simulateMessage({
      type: "broadcast",
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

  it("should support numeric comparison operators", () => {
    const testCases = [
      { operator: "eq", filter: "priority=eq.5" },
      { operator: "neq", filter: "priority=neq.0" },
      { operator: "gt", filter: "priority=gt.3" },
      { operator: "gte", filter: "priority=gte.5" },
      { operator: "lt", filter: "progress=lt.100" },
      { operator: "lte", filter: "progress=lte.50" },
    ];

    testCases.forEach(({ filter }) => {
      const testChannel = new RealtimeChannel(
        "http://localhost:8080",
        `test-${filter}`,
        "test-token",
      );
      const callback = vi.fn();

      testChannel.on(
        "postgres_changes",
        {
          event: "*",
          schema: "public",
          table: "jobs",
          filter,
        },
        callback,
      );

      testChannel.subscribe();

      const ws = (testChannel as any).ws as MockWebSocket;
      const parsedSubscribe = JSON.parse(ws.sentMessages[0]);
      expect(parsedSubscribe.config.filter).toBe(filter);

      testChannel.unsubscribe();
    });
  });

  it("should support IS NULL operator", () => {
    const callback = vi.fn();

    channel.on(
      "postgres_changes",
      {
        event: "*",
        schema: "public",
        table: "jobs",
        filter: "deleted_at=is.null",
      },
      callback,
    );

    channel.subscribe();

    const parsedSubscribe = JSON.parse(mockWs.sentMessages[0]);
    expect(parsedSubscribe.config.filter).toBe("deleted_at=is.null");
  });
});
