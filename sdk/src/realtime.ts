/**
 * Realtime subscriptions using WebSockets
 */

import type {
  RealtimeCallback,
  RealtimeChangePayload,
  RealtimeMessage,
  PostgresChangesConfig,
} from './types'

export class RealtimeChannel {
  private ws: WebSocket | null = null
  private url: string
  private token: string | null
  private channelName: string
  private callbacks: Map<string, Set<RealtimeCallback>> = new Map()
  private subscriptionConfig: PostgresChangesConfig | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 10
  private reconnectDelay = 1000
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null

  constructor(url: string, channelName: string, token: string | null = null) {
    this.url = url
    this.channelName = channelName
    this.token = token
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
  on(event: 'postgres_changes', config: PostgresChangesConfig, callback: RealtimeCallback): this

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
  on(event: 'INSERT' | 'UPDATE' | 'DELETE' | '*', callback: RealtimeCallback): this

  // Implementation
  on(
    event: 'postgres_changes' | 'INSERT' | 'UPDATE' | 'DELETE' | '*',
    configOrCallback: PostgresChangesConfig | RealtimeCallback,
    callback?: RealtimeCallback
  ): this {
    if (event === 'postgres_changes' && typeof configOrCallback !== 'function') {
      // New API: on('postgres_changes', config, callback)
      const config = configOrCallback as PostgresChangesConfig
      this.subscriptionConfig = config
      const actualCallback = callback!

      // Store callback with event type
      const eventType = config.event
      if (!this.callbacks.has(eventType)) {
        this.callbacks.set(eventType, new Set())
      }
      this.callbacks.get(eventType)!.add(actualCallback)
    } else {
      // Old API: on('INSERT', callback)
      const actualEvent = event as 'INSERT' | 'UPDATE' | 'DELETE' | '*'
      const actualCallback = configOrCallback as RealtimeCallback

      if (!this.callbacks.has(actualEvent)) {
        this.callbacks.set(actualEvent, new Set())
      }
      this.callbacks.get(actualEvent)!.add(actualCallback)
    }

    return this
  }

  /**
   * Remove a callback
   */
  off(event: 'INSERT' | 'UPDATE' | 'DELETE' | '*', callback: RealtimeCallback): this {
    const callbacks = this.callbacks.get(event)
    if (callbacks) {
      callbacks.delete(callback)
    }
    return this
  }

  /**
   * Subscribe to the channel
   */
  subscribe(): this {
    this.connect()
    return this
  }

  /**
   * Unsubscribe from the channel
   */
  unsubscribe(): void {
    if (this.ws) {
      this.send({
        type: 'unsubscribe',
        channel: this.channelName,
      })
      this.disconnect()
    }
  }

  /**
   * Internal: Connect to WebSocket
   */
  private connect() {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return
    }

    // Build WebSocket URL
    const wsUrl = new URL(this.url)
    wsUrl.protocol = wsUrl.protocol === 'https:' ? 'wss:' : 'ws:'
    wsUrl.pathname = '/realtime'

    if (this.token) {
      wsUrl.searchParams.set('token', this.token)
    }

    this.ws = new WebSocket(wsUrl.toString())

    this.ws.onopen = () => {
      console.log('[Fluxbase Realtime] Connected')
      this.reconnectAttempts = 0

      // Subscribe to channel with optional config
      const subscribeMessage: RealtimeMessage = {
        type: 'subscribe',
        channel: this.channelName,
      }

      // Add subscription config if using new postgres_changes API
      if (this.subscriptionConfig) {
        subscribeMessage.config = this.subscriptionConfig
      }

      this.send(subscribeMessage)

      // Start heartbeat
      this.startHeartbeat()
    }

    this.ws.onmessage = (event) => {
      try {
        const message: RealtimeMessage = JSON.parse(event.data)
        this.handleMessage(message)
      } catch (err) {
        console.error('[Fluxbase Realtime] Failed to parse message:', err)
      }
    }

    this.ws.onerror = (error) => {
      console.error('[Fluxbase Realtime] WebSocket error:', error)
    }

    this.ws.onclose = () => {
      console.log('[Fluxbase Realtime] Disconnected')
      this.stopHeartbeat()
      this.attemptReconnect()
    }
  }

  /**
   * Internal: Disconnect WebSocket
   */
  private disconnect() {
    this.stopHeartbeat()

    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  /**
   * Internal: Send a message
   */
  private send(message: RealtimeMessage) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
    }
  }

  /**
   * Internal: Handle incoming message
   */
  private handleMessage(message: RealtimeMessage) {
    switch (message.type) {
      case 'heartbeat':
        // Echo heartbeat back
        this.send({ type: 'heartbeat' })
        break

      case 'broadcast':
        if (message.payload) {
          this.handleBroadcast(message.payload as RealtimeChangePayload)
        }
        break

      case 'ack':
        console.log('[Fluxbase Realtime] Subscription acknowledged')
        break

      case 'error':
        console.error('[Fluxbase Realtime] Error:', message.error)
        break
    }
  }

  /**
   * Internal: Handle broadcast message
   */
  private handleBroadcast(payload: RealtimeChangePayload) {
    // Call event-specific callbacks
    const callbacks = this.callbacks.get(payload.type)
    if (callbacks) {
      callbacks.forEach((callback) => callback(payload))
    }

    // Call wildcard callbacks
    const wildcardCallbacks = this.callbacks.get('*')
    if (wildcardCallbacks) {
      wildcardCallbacks.forEach((callback) => callback(payload))
    }
  }

  /**
   * Internal: Start heartbeat interval
   */
  private startHeartbeat() {
    this.heartbeatInterval = setInterval(() => {
      this.send({ type: 'heartbeat' })
    }, 30000) // 30 seconds
  }

  /**
   * Internal: Stop heartbeat interval
   */
  private stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval)
      this.heartbeatInterval = null
    }
  }

  /**
   * Internal: Attempt to reconnect
   */
  private attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('[Fluxbase Realtime] Max reconnect attempts reached')
      return
    }

    this.reconnectAttempts++
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)

    console.log(`[Fluxbase Realtime] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`)

    setTimeout(() => {
      this.connect()
    }, delay)
  }
}

export class FluxbaseRealtime {
  private url: string
  private token: string | null
  private channels: Map<string, RealtimeChannel> = new Map()

  constructor(url: string, token: string | null = null) {
    this.url = url
    this.token = token
  }

  /**
   * Create or get a channel
   * @param channelName - Channel name (e.g., 'table:public.products')
   */
  channel(channelName: string): RealtimeChannel {
    if (this.channels.has(channelName)) {
      return this.channels.get(channelName)!
    }

    const channel = new RealtimeChannel(this.url, channelName, this.token)
    this.channels.set(channelName, channel)
    return channel
  }

  /**
   * Remove all channels
   */
  removeAllChannels() {
    this.channels.forEach((channel) => channel.unsubscribe())
    this.channels.clear()
  }

  /**
   * Update auth token for all channels
   */
  setToken(token: string | null) {
    this.token = token
    // Note: Existing channels won't be updated, only new ones
    // For existing channels to update, they need to reconnect
  }
}
