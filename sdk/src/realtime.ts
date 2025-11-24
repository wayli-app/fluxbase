/**
 * Realtime subscriptions using WebSockets
 */

import type {
  RealtimeCallback,
  RealtimePostgresChangesPayload,
  RealtimeMessage,
  PostgresChangesConfig,
  RealtimeChannelConfig,
  PresenceState,
  RealtimePresencePayload,
  PresenceCallback,
  BroadcastMessage,
  RealtimeBroadcastPayload,
  BroadcastCallback,
} from "./types";

export class RealtimeChannel {
  private ws: WebSocket | null = null;
  private url: string;
  private token: string | null;
  private channelName: string;
  private callbacks: Map<string, Set<RealtimeCallback>> = new Map();
  private presenceCallbacks: Map<string, Set<PresenceCallback>> = new Map();
  private broadcastCallbacks: Map<string, Set<BroadcastCallback>> = new Map();
  private subscriptionConfig: PostgresChangesConfig | null = null;
  private subscriptionId: string | null = null;
  private _presenceState: Record<string, PresenceState[]> = {};
  private myPresenceKey: string | null = null;
  private config: RealtimeChannelConfig;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private reconnectDelay = 1000;
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null;
  private pendingAcks: Map<
    string,
    { resolve: (value: string) => void; reject: (reason: any) => void; timeout: ReturnType<typeof setTimeout> }
  > = new Map();
  private messageIdCounter = 0;

  constructor(
    url: string,
    channelName: string,
    token: string | null = null,
    config: RealtimeChannelConfig = {}
  ) {
    this.url = url;
    this.channelName = channelName;
    this.token = token;
    this.config = config;
  }

  /**
   * Listen to postgres_changes with optional row-level filtering
   *
   * @param event - 'postgres_changes'
   * @param config - Configuration including optional filter
   * @param callback - Function to call when changes occur
   * @returns This channel for chaining
   *
   * @example
   * ```typescript
   * channel.on('postgres_changes', {
   *   event: '*',
   *   schema: 'public',
   *   table: 'jobs',
   *   filter: 'created_by=eq.user123'
   * }, (payload) => {
   *   console.log('Job updated:', payload)
   * })
   * ```
   */
  on(
    event: "postgres_changes",
    config: PostgresChangesConfig,
    callback: RealtimeCallback,
  ): this;

  /**
   * Listen to a specific event type (backwards compatibility)
   *
   * @param event - The event type (INSERT, UPDATE, DELETE, or '*' for all)
   * @param callback - The callback function
   * @returns This channel for chaining
   *
   * @example
   * ```typescript
   * channel.on('INSERT', (payload) => {
   *   console.log('New record inserted:', payload.new_record)
   * })
   * ```
   */
  on(
    event: "INSERT" | "UPDATE" | "DELETE" | "*",
    callback: RealtimeCallback,
  ): this;

  /**
   * Listen to broadcast messages
   *
   * @param event - 'broadcast'
   * @param config - Configuration with event name
   * @param callback - Function to call when broadcast received
   * @returns This channel for chaining
   *
   * @example
   * ```typescript
   * channel.on('broadcast', { event: 'cursor-pos' }, (payload) => {
   *   console.log('Cursor moved:', payload)
   * })
   * ```
   */
  on(
    event: "broadcast",
    config: { event: string },
    callback: BroadcastCallback,
  ): this;

  /**
   * Listen to presence events
   *
   * @param event - 'presence'
   * @param config - Configuration with event type (sync, join, leave)
   * @param callback - Function to call when presence changes
   * @returns This channel for chaining
   *
   * @example
   * ```typescript
   * channel.on('presence', { event: 'sync' }, (payload) => {
   *   console.log('Presence synced:', payload)
   * })
   * ```
   */
  on(
    event: "presence",
    config: { event: "sync" | "join" | "leave" },
    callback: PresenceCallback,
  ): this;

  // Implementation
  on(
    event:
      | "postgres_changes"
      | "INSERT"
      | "UPDATE"
      | "DELETE"
      | "*"
      | "broadcast"
      | "presence",
    configOrCallback:
      | PostgresChangesConfig
      | RealtimeCallback
      | { event: string }
      | { event: "sync" | "join" | "leave" },
    callback?: RealtimeCallback | BroadcastCallback | PresenceCallback,
  ): this {
    if (
      event === "postgres_changes" &&
      typeof configOrCallback !== "function"
    ) {
      // on('postgres_changes', config, callback)
      const config = configOrCallback as PostgresChangesConfig;
      this.subscriptionConfig = config;
      const actualCallback = callback!;

      const eventType = config.event;
      if (!this.callbacks.has(eventType)) {
        this.callbacks.set(eventType, new Set());
      }
      this.callbacks.get(eventType)!.add(actualCallback as RealtimeCallback);
    } else if (event === "broadcast" && typeof configOrCallback !== "function") {
      // on('broadcast', { event }, callback)
      const config = configOrCallback as { event: string };
      const actualCallback = callback as BroadcastCallback;

      if (!this.broadcastCallbacks.has(config.event)) {
        this.broadcastCallbacks.set(config.event, new Set());
      }
      this.broadcastCallbacks.get(config.event)!.add(actualCallback);
    } else if (event === "presence" && typeof configOrCallback !== "function") {
      // on('presence', { event }, callback)
      const config = configOrCallback as { event: "sync" | "join" | "leave" };
      const actualCallback = callback as PresenceCallback;

      if (!this.presenceCallbacks.has(config.event)) {
        this.presenceCallbacks.set(config.event, new Set());
      }
      this.presenceCallbacks.get(config.event)!.add(actualCallback);
    } else {
      // on('INSERT'|'UPDATE'|'DELETE'|'*', callback)
      const actualEvent = event as "INSERT" | "UPDATE" | "DELETE" | "*";
      const actualCallback = configOrCallback as RealtimeCallback;

      if (!this.callbacks.has(actualEvent)) {
        this.callbacks.set(actualEvent, new Set());
      }
      this.callbacks.get(actualEvent)!.add(actualCallback);
    }

    return this;
  }

  /**
   * Remove a callback
   */
  off(
    event: "INSERT" | "UPDATE" | "DELETE" | "*",
    callback: RealtimeCallback,
  ): this {
    const callbacks = this.callbacks.get(event);
    if (callbacks) {
      callbacks.delete(callback);
    }
    return this;
  }

  /**
   * Subscribe to the channel
   * @param callback - Optional status callback (Supabase-compatible)
   * @param _timeout - Optional timeout in milliseconds (currently unused)
   */
  subscribe(
    callback?: (
      status: "SUBSCRIBED" | "CHANNEL_ERROR" | "TIMED_OUT" | "CLOSED",
      err?: Error,
    ) => void,
    _timeout?: number,
  ): this {
    this.connect();

    // Call callback with SUBSCRIBED status after connection
    if (callback) {
      // Wait for connection to open
      const checkConnection = () => {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
          callback("SUBSCRIBED");
        } else if (this.ws && this.ws.readyState === WebSocket.CLOSED) {
          callback("CHANNEL_ERROR", new Error("Failed to connect"));
        } else {
          setTimeout(checkConnection, 100);
        }
      };
      setTimeout(checkConnection, 100);
    }

    return this;
  }

  /**
   * Unsubscribe from the channel
   * @param timeout - Optional timeout in milliseconds
   * @returns Promise resolving to status string (Supabase-compatible)
   */
  async unsubscribe(timeout?: number): Promise<"ok" | "timed out" | "error"> {
    return new Promise((resolve) => {
      if (this.ws) {
        this.sendMessage({
          type: "unsubscribe",
          channel: this.channelName,
          subscription_id: this.subscriptionId || undefined,
        });

        // Wait for disconnect
        const startTime = Date.now();
        const maxWait = timeout || 5000;

        const checkDisconnect = () => {
          if (!this.ws || this.ws.readyState === WebSocket.CLOSED) {
            this.disconnect();
            resolve("ok");
          } else if (Date.now() - startTime > maxWait) {
            this.disconnect();
            resolve("timed out");
          } else {
            setTimeout(checkDisconnect, 100);
          }
        };

        setTimeout(checkDisconnect, 100);
      } else {
        resolve("ok");
      }
    });
  }

  /**
   * Send a broadcast message to all subscribers on this channel
   *
   * @param message - Broadcast message with type, event, and payload
   * @returns Promise resolving to status
   *
   * @example
   * ```typescript
   * await channel.send({
   *   type: 'broadcast',
   *   event: 'cursor-pos',
   *   payload: { x: 100, y: 200 }
   * })
   * ```
   */
  async send(message: BroadcastMessage): Promise<"ok" | "error"> {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return "error";
    }

    try {
      // Generate message ID if acknowledgment is requested
      const messageId = this.config.broadcast?.ack
        ? `msg_${Date.now()}_${++this.messageIdCounter}`
        : undefined;

      this.ws.send(
        JSON.stringify({
          type: "broadcast",
          channel: this.channelName,
          event: message.event,
          payload: message.payload,
          ...(messageId && { messageId }),
        })
      );

      // Handle acknowledgment if configured
      if (this.config.broadcast?.ack && messageId) {
        // Wait for ack response with timeout (default 5 seconds)
        const ackTimeout = this.config.broadcast.ackTimeout || 5000;
        return await new Promise<"ok" | "error">((resolve, reject) => {
          const timeout = setTimeout(() => {
            this.pendingAcks.delete(messageId);
            reject(new Error("Acknowledgment timeout"));
          }, ackTimeout);

          this.pendingAcks.set(messageId, {
            resolve: (value) => {
              clearTimeout(timeout);
              this.pendingAcks.delete(messageId);
              resolve(value as "ok" | "error");
            },
            reject: (reason) => {
              clearTimeout(timeout);
              this.pendingAcks.delete(messageId);
              reject(reason);
            },
            timeout,
          });
        }).catch((error) => {
          console.error("[Fluxbase Realtime] Acknowledgment error:", error);
          return "error" as const;
        });
      }

      return "ok";
    } catch (error) {
      console.error("[Fluxbase Realtime] Failed to send broadcast:", error);
      return "error";
    }
  }

  /**
   * Track user presence on this channel
   *
   * @param state - Presence state to track
   * @returns Promise resolving to status
   *
   * @example
   * ```typescript
   * await channel.track({
   *   user_id: 123,
   *   status: 'online'
   * })
   * ```
   */
  async track(state: PresenceState): Promise<"ok" | "error"> {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return "error";
    }

    try {
      // Generate presence key if not set
      if (!this.myPresenceKey) {
        this.myPresenceKey =
          this.config.presence?.key || `presence-${Math.random().toString(36).substr(2, 9)}`;
      }

      this.ws.send(
        JSON.stringify({
          type: "presence",
          channel: this.channelName,
          event: "track",
          payload: {
            key: this.myPresenceKey,
            state,
          },
        })
      );

      return "ok";
    } catch (error) {
      console.error("[Fluxbase Realtime] Failed to track presence:", error);
      return "error";
    }
  }

  /**
   * Stop tracking presence on this channel
   *
   * @returns Promise resolving to status
   *
   * @example
   * ```typescript
   * await channel.untrack()
   * ```
   */
  async untrack(): Promise<"ok" | "error"> {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return "error";
    }

    if (!this.myPresenceKey) {
      return "ok"; // Already not tracking
    }

    try {
      this.ws.send(
        JSON.stringify({
          type: "presence",
          channel: this.channelName,
          event: "untrack",
          payload: {
            key: this.myPresenceKey,
          },
        })
      );

      this.myPresenceKey = null;
      return "ok";
    } catch (error) {
      console.error("[Fluxbase Realtime] Failed to untrack presence:", error);
      return "error";
    }
  }

  /**
   * Get current presence state for all users on this channel
   *
   * @returns Current presence state
   *
   * @example
   * ```typescript
   * const state = channel.presenceState()
   * console.log('Online users:', Object.keys(state).length)
   * ```
   */
  presenceState(): Record<string, PresenceState[]> {
    return { ...this._presenceState };
  }

  /**
   * Internal: Connect to WebSocket
   */
  private connect() {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return;
    }

    // Build WebSocket URL
    const wsUrl = new URL(this.url);
    wsUrl.protocol = wsUrl.protocol === "https:" ? "wss:" : "ws:";
    wsUrl.pathname = "/realtime";

    if (this.token) {
      wsUrl.searchParams.set("token", this.token);
    }

    this.ws = new WebSocket(wsUrl.toString());

    this.ws.onopen = () => {
      console.log("[Fluxbase Realtime] Connected");
      this.reconnectAttempts = 0;

      // Subscribe to channel with optional config
      const subscribeMessage: RealtimeMessage = {
        type: "subscribe",
        channel: this.channelName,
      };

      // Add subscription config if using new postgres_changes API
      if (this.subscriptionConfig) {
        subscribeMessage.config = this.subscriptionConfig;
      }

      this.sendMessage(subscribeMessage);

      // Start heartbeat
      this.startHeartbeat();
    };

    this.ws.onmessage = (event) => {
      try {
        const message: RealtimeMessage = JSON.parse(event.data);
        this.handleMessage(message);
      } catch (err) {
        console.error("[Fluxbase Realtime] Failed to parse message:", err);
      }
    };

    this.ws.onerror = (error) => {
      console.error("[Fluxbase Realtime] WebSocket error:", error);
    };

    this.ws.onclose = () => {
      console.log("[Fluxbase Realtime] Disconnected");
      this.stopHeartbeat();
      this.attemptReconnect();
    };
  }

  /**
   * Internal: Disconnect WebSocket
   */
  private disconnect() {
    this.stopHeartbeat();

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  /**
   * Internal: Send a message
   */
  private sendMessage(message: RealtimeMessage) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  /**
   * Internal: Handle incoming message
   */
  private handleMessage(message: RealtimeMessage) {
    switch (message.type) {
      case "heartbeat":
        // Echo heartbeat back
        this.ws?.send(JSON.stringify({ type: "heartbeat" }));
        break;

      case "broadcast":
        if (message.broadcast) {
          this.handleBroadcastMessage(message.broadcast);
        } else if (message.payload) {
          // Legacy postgres_changes format
          this.handlePostgresChanges(message.payload);
        }
        break;

      case "presence":
        if (message.presence) {
          this.handlePresenceMessage(message.presence);
        }
        break;

      case "ack":
        // Handle broadcast acknowledgment
        if (message.messageId && this.pendingAcks.has(message.messageId)) {
          const ackHandler = this.pendingAcks.get(message.messageId);
          if (ackHandler) {
            ackHandler.resolve(message.status || "ok");
          }
        } else {
          // Store subscription_id from subscription acknowledgment
          if (message.payload && typeof message.payload === 'object' && 'subscription_id' in message.payload) {
            this.subscriptionId = (message.payload as { subscription_id: string }).subscription_id;
            console.log("[Fluxbase Realtime] Subscription ID received:", this.subscriptionId);
          } else {
            // Log other acknowledgments
            console.log("[Fluxbase Realtime] Acknowledged:", message);
          }
        }
        break;

      case "error":
        console.error("[Fluxbase Realtime] Error:", message.error);
        break;
    }
  }

  /**
   * Internal: Handle broadcast message
   */
  private handleBroadcastMessage(message: any) {
    const event = message.event;
    const payload: RealtimeBroadcastPayload = {
      event,
      payload: message.payload,
    };

    // Filter self-messages if configured
    if (!this.config.broadcast?.self && message.self) {
      return;
    }

    // Call event-specific callbacks
    const callbacks = this.broadcastCallbacks.get(event);
    if (callbacks) {
      callbacks.forEach((callback) => callback(payload));
    }

    // Call wildcard callbacks
    const wildcardCallbacks = this.broadcastCallbacks.get("*");
    if (wildcardCallbacks) {
      wildcardCallbacks.forEach((callback) => callback(payload));
    }
  }

  /**
   * Internal: Handle presence message
   */
  private handlePresenceMessage(message: any) {
    const event = message.event as "sync" | "join" | "leave";
    const payload: RealtimePresencePayload = {
      event,
      key: message.key,
      newPresences: message.newPresences,
      leftPresences: message.leftPresences,
      currentPresences: message.currentPresences || this._presenceState,
    };

    // Update local presence state
    if (message.currentPresences) {
      this._presenceState = message.currentPresences;
    }

    // Call event-specific callbacks
    const callbacks = this.presenceCallbacks.get(event);
    if (callbacks) {
      callbacks.forEach((callback) => callback(payload));
    }
  }

  /**
   * Internal: Handle postgres_changes message
   */
  private handlePostgresChanges(payload: any) {
    // Convert to Supabase-compatible format
    const supabasePayload: RealtimePostgresChangesPayload = {
      eventType: payload.type || payload.eventType,
      schema: payload.schema,
      table: payload.table,
      commit_timestamp:
        payload.timestamp ||
        payload.commit_timestamp ||
        new Date().toISOString(),
      new: payload.new_record || payload.new || {},
      old: payload.old_record || payload.old || {},
      errors: payload.errors || null,
    };

    // Call event-specific callbacks
    const callbacks = this.callbacks.get(supabasePayload.eventType);
    if (callbacks) {
      callbacks.forEach((callback) => callback(supabasePayload));
    }

    // Call wildcard callbacks
    const wildcardCallbacks = this.callbacks.get("*");
    if (wildcardCallbacks) {
      wildcardCallbacks.forEach((callback) => callback(supabasePayload));
    }
  }

  /**
   * Internal: Start heartbeat interval
   */
  private startHeartbeat() {
    this.heartbeatInterval = setInterval(() => {
      this.sendMessage({ type: "heartbeat" });
    }, 30000); // 30 seconds
  }

  /**
   * Internal: Stop heartbeat interval
   */
  private stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  /**
   * Internal: Attempt to reconnect
   */
  private attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error("[Fluxbase Realtime] Max reconnect attempts reached");
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    console.log(
      `[Fluxbase Realtime] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`,
    );

    setTimeout(() => {
      this.connect();
    }, delay);
  }
}

export class FluxbaseRealtime {
  private url: string;
  private token: string | null;
  private channels: Map<string, RealtimeChannel> = new Map();

  constructor(url: string, token: string | null = null) {
    this.url = url;
    this.token = token;
  }

  /**
   * Create or get a channel with optional configuration
   *
   * @param channelName - Channel name (e.g., 'table:public.products')
   * @param config - Optional channel configuration
   * @returns RealtimeChannel instance
   *
   * @example
   * ```typescript
   * const channel = realtime.channel('room-1', {
   *   broadcast: { self: true, ack: true },
   *   presence: { key: 'user-123' }
   * })
   * ```
   */
  channel(
    channelName: string,
    config?: RealtimeChannelConfig
  ): RealtimeChannel {
    // Create a unique key based on name and config
    const configKey = config ? JSON.stringify(config) : "";
    const key = `${channelName}:${configKey}`;

    if (this.channels.has(key)) {
      return this.channels.get(key)!;
    }

    const channel = new RealtimeChannel(
      this.url,
      channelName,
      this.token,
      config
    );
    this.channels.set(key, channel);
    return channel;
  }

  /**
   * Remove a specific channel
   *
   * @param channel - The channel to remove
   * @returns Promise resolving to status
   *
   * @example
   * ```typescript
   * const channel = realtime.channel('room-1')
   * await realtime.removeChannel(channel)
   * ```
   */
  async removeChannel(
    channel: RealtimeChannel
  ): Promise<"ok" | "error"> {
    // Unsubscribe the channel
    await channel.unsubscribe();

    // Remove from channels map
    for (const [key, ch] of this.channels.entries()) {
      if (ch === channel) {
        this.channels.delete(key);
        return "ok";
      }
    }

    return "error";
  }

  /**
   * Remove all channels
   */
  removeAllChannels() {
    this.channels.forEach((channel) => channel.unsubscribe());
    this.channels.clear();
  }

  /**
   * Update auth token for all channels
   * @param token - The new auth token
   */
  setAuth(token: string | null) {
    this.token = token;
    // Note: Existing channels won't be updated, only new ones
    // For existing channels to update, they need to reconnect
  }
}
