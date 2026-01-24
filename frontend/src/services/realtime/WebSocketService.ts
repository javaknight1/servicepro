/**
 * WebSocket Service - Core WebSocket connection and messaging
 */

import {
  ConnectionConfig,
  ConnectionState,
  ConnectionMetrics,
  WebSocketMessage,
  WebSocketEventType,
  WebSocketEventHandler,
  MessageType,
  MessageHandler,
  Channel,
  ChannelSubscription,
  QueuedMessage,
  AckMessage,
  PingMessage,
  PongMessage,
  SubscribeMessage,
  UnsubscribeMessage,
  DEFAULT_CONNECTION_CONFIG,
  generateMessageId,
  createMessage,
  parseCloseCode,
  isRecoverableError,
  calculateBackoff,
} from '../../types/realtime';

// ============================================
// WebSocket Service Class
// ============================================

export class WebSocketServiceImpl {
  private socket: WebSocket | null = null;
  private config: Required<ConnectionConfig>;
  private state: ConnectionState;
  private metrics: ConnectionMetrics;

  // Event handlers
  private eventHandlers: Map<WebSocketEventType, Set<WebSocketEventHandler>> =
    new Map();
  private messageHandlers: Map<MessageType, Set<MessageHandler>> = new Map();

  // Subscriptions
  private subscriptions: Map<string, ChannelSubscription[]> = new Map();
  private channels: Map<string, Channel> = new Map();

  // Message queue and pending responses
  private messageQueue: QueuedMessage[] = [];
  private pendingResponses: Map<
    string,
    {
      resolve: (value: WebSocketMessage) => void;
      reject: (error: Error) => void;
      timeout: NodeJS.Timeout;
    }
  > = new Map();

  // Timers
  private heartbeatTimer: NodeJS.Timeout | null = null;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private connectionTimeoutTimer: NodeJS.Timeout | null = null;

  // State tracking
  private reconnectAttempt = 0;
  private isManualDisconnect = false;
  private lastPingTime = 0;

  constructor(config: ConnectionConfig) {
    this.config = { ...DEFAULT_CONNECTION_CONFIG, ...config };
    this.state = this.createInitialState();
    this.metrics = this.createInitialMetrics();

    // Bind methods
    this.handleOpen = this.handleOpen.bind(this);
    this.handleClose = this.handleClose.bind(this);
    this.handleError = this.handleError.bind(this);
    this.handleMessage = this.handleMessage.bind(this);
  }

  // ============================================
  // Connection Management
  // ============================================

  async connect(): Promise<void> {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.log('Already connected');
      return;
    }

    if (this.socket?.readyState === WebSocket.CONNECTING) {
      this.log('Connection in progress');
      return;
    }

    this.isManualDisconnect = false;
    this.updateState({ status: 'connecting' });

    return new Promise((resolve, reject) => {
      try {
        const url = this.buildUrl();
        this.log(`Connecting to ${url}`);

        this.socket = new WebSocket(url, this.config.protocols);

        // Set up connection timeout
        this.connectionTimeoutTimer = setTimeout(() => {
          if (this.socket?.readyState !== WebSocket.OPEN) {
            this.socket?.close();
            reject(new Error('Connection timeout'));
          }
        }, this.config.connectionTimeout);

        // Set up event handlers
        this.socket.onopen = (event) => {
          this.handleOpen(event);
          resolve();
        };

        this.socket.onclose = this.handleClose;
        this.socket.onerror = (event) => {
          this.handleError(event as Event);
          if (this.state.status === 'connecting') {
            reject(new Error('Connection failed'));
          }
        };
        this.socket.onmessage = this.handleMessage;

        this.metrics.totalConnections++;
      } catch (error) {
        this.metrics.failedConnections++;
        this.updateState({
          status: 'failed',
          lastError: error as Error,
        });
        reject(error);
      }
    });
  }

  disconnect(code: number = 1000, reason: string = 'Client disconnect'): void {
    this.log(`Disconnecting: ${reason}`);
    this.isManualDisconnect = true;
    this.clearTimers();

    if (this.socket) {
      this.socket.close(code, reason);
      this.socket = null;
    }

    this.updateState({
      status: 'disconnected',
      disconnectedAt: Date.now(),
    });
  }

  async reconnect(): Promise<void> {
    if (this.isManualDisconnect) {
      this.log('Manual disconnect - not reconnecting');
      return;
    }

    if (!this.config.autoReconnect) {
      this.log('Auto-reconnect disabled');
      return;
    }

    if (
      this.config.maxReconnectAttempts > 0 &&
      this.reconnectAttempt >= this.config.maxReconnectAttempts
    ) {
      this.log('Max reconnect attempts reached');
      this.updateState({ status: 'failed' });
      this.emitEvent('reconnect_failed', { attempts: this.reconnectAttempt });
      return;
    }

    this.reconnectAttempt++;
    this.metrics.totalReconnects++;
    this.updateState({
      status: 'reconnecting',
      reconnectAttempts: this.reconnectAttempt,
    });

    const delay = calculateBackoff(
      this.reconnectAttempt - 1,
      this.config.reconnectDelay,
      this.config.maxReconnectDelay,
      this.config.reconnectDelayMultiplier
    );

    this.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempt})`);

    return new Promise((resolve, reject) => {
      this.reconnectTimer = setTimeout(async () => {
        try {
          await this.connect();
          this.emitEvent('reconnect', { attempt: this.reconnectAttempt });
          resolve();
        } catch (error) {
          reject(error);
          // Will trigger another reconnect via handleClose
        }
      }, delay);
    });
  }

  getConnectionState(): ConnectionState {
    return { ...this.state };
  }

  isConnected(): boolean {
    return this.socket?.readyState === WebSocket.OPEN;
  }

  // ============================================
  // Messaging
  // ============================================

  async send(message: WebSocketMessage): Promise<void> {
    if (!message.id) {
      message.id = generateMessageId();
    }
    if (!message.timestamp) {
      message.timestamp = Date.now();
    }

    if (!this.isConnected()) {
      this.queueMessage(message);
      return;
    }

    try {
      const data = JSON.stringify(message);
      this.socket!.send(data);
      this.metrics.messagesSent++;
      this.metrics.bytesSent += data.length;
      this.log(`Sent: ${message.type}`, message);
    } catch (error) {
      this.log(`Send error: ${error}`);
      this.queueMessage(message);
      throw error;
    }
  }

  async sendAndWait<T extends WebSocketMessage>(
    message: WebSocketMessage,
    timeout: number = 10000
  ): Promise<T> {
    return new Promise((resolve, reject) => {
      const timeoutHandle = setTimeout(() => {
        this.pendingResponses.delete(message.id);
        reject(new Error(`Response timeout for message ${message.id}`));
      }, timeout);

      this.pendingResponses.set(message.id, {
        resolve: resolve as (value: WebSocketMessage) => void,
        reject,
        timeout: timeoutHandle,
      });

      this.send(message).catch((error) => {
        clearTimeout(timeoutHandle);
        this.pendingResponses.delete(message.id);
        reject(error);
      });
    });
  }

  // ============================================
  // Subscriptions
  // ============================================

  subscribe(subscription: ChannelSubscription): () => void {
    const { channel } = subscription;

    // Store subscription
    if (!this.subscriptions.has(channel)) {
      this.subscriptions.set(channel, []);
    }
    this.subscriptions.get(channel)!.push(subscription);

    // Create channel if needed
    if (!this.channels.has(channel)) {
      this.channels.set(channel, {
        name: channel,
        subscribed: false,
        subscribedAt: null,
        filters: subscription.filters,
        messageCount: 0,
        lastMessageAt: null,
      });
    }

    // Send subscribe message if connected
    if (this.isConnected() && !this.channels.get(channel)!.subscribed) {
      this.sendSubscribe(channel, subscription.filters);
    }

    // Return unsubscribe function
    return () => {
      const subs = this.subscriptions.get(channel);
      if (subs) {
        const index = subs.indexOf(subscription);
        if (index > -1) {
          subs.splice(index, 1);
        }
        if (subs.length === 0) {
          this.unsubscribe(channel);
        }
      }
    };
  }

  unsubscribe(channel: string): void {
    this.subscriptions.delete(channel);
    this.channels.delete(channel);

    if (this.isConnected()) {
      const message = createMessage<UnsubscribeMessage>('unsubscribe', {
        channel,
      });
      this.send(message).catch((error) => {
        this.log(`Failed to send unsubscribe: ${error}`);
      });
    }
  }

  getSubscriptions(): Channel[] {
    return Array.from(this.channels.values());
  }

  // ============================================
  // Events
  // ============================================

  on(event: WebSocketEventType, handler: WebSocketEventHandler): () => void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, new Set());
    }
    this.eventHandlers.get(event)!.add(handler);

    return () => this.off(event, handler);
  }

  off(event: WebSocketEventType, handler: WebSocketEventHandler): void {
    this.eventHandlers.get(event)?.delete(handler);
  }

  // ============================================
  // Message Handlers
  // ============================================

  registerHandler<T extends MessageType>(
    type: T,
    handler: MessageHandler
  ): () => void {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, new Set());
    }
    this.messageHandlers.get(type)!.add(handler);

    return () => this.unregisterHandler(type, handler);
  }

  unregisterHandler(type: MessageType, handler?: MessageHandler): void {
    if (handler) {
      this.messageHandlers.get(type)?.delete(handler);
    } else {
      this.messageHandlers.delete(type);
    }
  }

  // ============================================
  // Metrics
  // ============================================

  getMetrics(): ConnectionMetrics {
    return {
      ...this.metrics,
      uptime: this.state.connectedAt ? Date.now() - this.state.connectedAt : 0,
    };
  }

  resetMetrics(): void {
    this.metrics = this.createInitialMetrics();
  }

  // ============================================
  // Utilities
  // ============================================

  async ping(): Promise<number> {
    if (!this.isConnected()) {
      throw new Error('Not connected');
    }

    const pingMessage = createMessage<PingMessage>('ping', {});
    this.lastPingTime = Date.now();

    const response = await this.sendAndWait<PongMessage>(pingMessage, 5000);
    const latency = response.latency || Date.now() - this.lastPingTime;

    this.updateState({ latency });
    this.metrics.averageLatency = this.metrics.averageLatency
      ? (this.metrics.averageLatency + latency) / 2
      : latency;

    this.emitEvent('latency', { latency });
    return latency;
  }

  setToken(token: string): void {
    this.config.token = token;
    // If connected, may need to re-authenticate
    if (this.isConnected()) {
      this.log('Token updated - reconnecting to apply');
      this.disconnect();
      this.reconnect();
    }
  }

  // ============================================
  // Private Methods - Event Handlers
  // ============================================

  private handleOpen(_event: Event): void {
    this.clearTimer('connectionTimeout');
    this.reconnectAttempt = 0;
    this.metrics.successfulConnections++;

    this.updateState({
      status: 'connected',
      connectedAt: Date.now(),
      disconnectedAt: null,
      reconnectAttempts: 0,
      lastError: null,
    });

    this.log('Connected');
    this.emitEvent('open', {});

    // Start heartbeat
    this.startHeartbeat();

    // Resubscribe to channels
    this.resubscribeAll();

    // Flush message queue
    this.flushMessageQueue();
  }

  private handleClose(event: CloseEvent): void {
    this.clearTimers();
    this.metrics.totalDisconnects++;

    const reason = parseCloseCode(event.code);
    this.log(`Disconnected: ${reason} (${event.code})`);

    this.updateState({
      status: 'disconnected',
      disconnectedAt: Date.now(),
    });

    // Update channel states
    this.channels.forEach((channel) => {
      channel.subscribed = false;
    });

    this.emitEvent('close', { code: event.code, reason });

    // Auto-reconnect if applicable
    if (
      !this.isManualDisconnect &&
      this.config.autoReconnect &&
      isRecoverableError(event.code)
    ) {
      this.reconnect().catch((error) => {
        this.log(`Reconnect failed: ${error}`);
      });
    }
  }

  private handleError(event: Event): void {
    this.metrics.errors++;
    const error = new Error('WebSocket error');
    this.updateState({ lastError: error });
    this.log('Error occurred', event);
    this.emitEvent('error', { error });
  }

  private handleMessage(event: MessageEvent): void {
    try {
      const message: WebSocketMessage = JSON.parse(event.data);
      this.metrics.messagesReceived++;
      this.metrics.bytesReceived += event.data.length;

      this.log(`Received: ${message.type}`, message);

      // Handle pending responses
      if (message.type === 'ack' || message.type === 'pong') {
        const pending = this.pendingResponses.get(
          (message as AckMessage).messageId || message.id
        );
        if (pending) {
          clearTimeout(pending.timeout);
          this.pendingResponses.delete(
            (message as AckMessage).messageId || message.id
          );
          pending.resolve(message);
          return;
        }
      }

      // Handle pong specifically for latency
      if (message.type === 'pong' && this.lastPingTime) {
        (message as PongMessage).latency = Date.now() - this.lastPingTime;
      }

      // Emit to registered handlers
      this.messageHandlers.get(message.type)?.forEach((handler) => {
        try {
          handler(message);
        } catch (error) {
          this.log(`Handler error for ${message.type}: ${error}`);
        }
      });

      // Emit to channel subscribers
      if ('channel' in message && message.channel) {
        const channel = this.channels.get(message.channel);
        if (channel) {
          channel.messageCount++;
          channel.lastMessageAt = Date.now();
        }

        this.subscriptions.get(message.channel)?.forEach((sub) => {
          try {
            sub.onMessage(message);
          } catch (error) {
            sub.onError?.(error as Error);
          }
        });
      }

      // Emit general message event
      this.emitEvent('message', { message });
    } catch (error) {
      this.log(`Failed to parse message: ${error}`);
      this.metrics.errors++;
    }
  }

  // ============================================
  // Private Methods - Helpers
  // ============================================

  private buildUrl(): string {
    let url = this.config.url;

    // Add token as query parameter if provided
    if (this.config.token) {
      const separator = url.includes('?') ? '&' : '?';
      url += `${separator}token=${encodeURIComponent(this.config.token)}`;
    }

    return url;
  }

  private updateState(updates: Partial<ConnectionState>): void {
    this.state = { ...this.state, ...updates };
  }

  private emitEvent(type: WebSocketEventType, data: unknown): void {
    const event = { type, timestamp: Date.now(), data };
    this.eventHandlers.get(type)?.forEach((handler) => {
      try {
        handler(event);
      } catch (error) {
        this.log(`Event handler error: ${error}`);
      }
    });
  }

  private startHeartbeat(): void {
    if (this.config.heartbeatInterval <= 0) return;

    this.heartbeatTimer = setInterval(() => {
      this.ping().catch((error) => {
        this.log(`Heartbeat failed: ${error}`);
      });
    }, this.config.heartbeatInterval);
  }

  private sendSubscribe(
    channel: string,
    filters?: Record<string, unknown>
  ): void {
    const message = createMessage<SubscribeMessage>('subscribe', {
      channel,
      filters,
    });

    this.send(message)
      .then(() => {
        const ch = this.channels.get(channel);
        if (ch) {
          ch.subscribed = true;
          ch.subscribedAt = Date.now();
        }
      })
      .catch((error) => {
        this.log(`Failed to subscribe to ${channel}: ${error}`);
      });
  }

  private resubscribeAll(): void {
    this.channels.forEach((channel, name) => {
      if (!channel.subscribed) {
        this.sendSubscribe(name, channel.filters);
      }
    });
  }

  private queueMessage(message: WebSocketMessage, priority: number = 1): void {
    if (this.messageQueue.length >= this.config.messageQueueSize) {
      // Remove lowest priority message
      this.messageQueue.sort((a, b) => b.priority - a.priority);
      this.messageQueue.pop();
    }

    this.messageQueue.push({
      id: message.id,
      message,
      priority,
      retries: 0,
      maxRetries: 3,
      createdAt: Date.now(),
    });

    this.log(
      `Message queued: ${message.type} (queue size: ${this.messageQueue.length})`
    );
  }

  private async flushMessageQueue(): Promise<void> {
    if (this.messageQueue.length === 0) return;

    this.log(`Flushing message queue (${this.messageQueue.length} messages)`);

    // Sort by priority and timestamp
    this.messageQueue.sort((a, b) => {
      if (a.priority !== b.priority) return b.priority - a.priority;
      return a.createdAt - b.createdAt;
    });

    const queue = [...this.messageQueue];
    this.messageQueue = [];

    for (const item of queue) {
      try {
        await this.send(item.message);
        item.sentAt = Date.now();
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
      } catch (_error) {
        if (item.retries < item.maxRetries) {
          item.retries++;
          this.messageQueue.push(item);
        } else {
          this.log(
            `Message dropped after ${item.maxRetries} retries: ${item.message.type}`
          );
        }
      }
    }
  }

  private clearTimers(): void {
    this.clearTimer('heartbeat');
    this.clearTimer('reconnect');
    this.clearTimer('connectionTimeout');
  }

  private clearTimer(
    type: 'heartbeat' | 'reconnect' | 'connectionTimeout'
  ): void {
    const timerMap = {
      heartbeat: this.heartbeatTimer,
      reconnect: this.reconnectTimer,
      connectionTimeout: this.connectionTimeoutTimer,
    };

    const timer = timerMap[type];
    if (timer) {
      clearTimeout(timer);
      clearInterval(timer);
      if (type === 'heartbeat') this.heartbeatTimer = null;
      if (type === 'reconnect') this.reconnectTimer = null;
      if (type === 'connectionTimeout') this.connectionTimeoutTimer = null;
    }
  }

  private createInitialState(): ConnectionState {
    return {
      status: 'disconnected',
      connectedAt: null,
      disconnectedAt: null,
      reconnectAttempts: 0,
      lastError: null,
      latency: null,
    };
  }

  private createInitialMetrics(): ConnectionMetrics {
    return {
      totalConnections: 0,
      successfulConnections: 0,
      failedConnections: 0,
      totalReconnects: 0,
      totalDisconnects: 0,
      averageConnectionDuration: 0,
      averageLatency: 0,
      messagesSent: 0,
      messagesReceived: 0,
      bytesReceived: 0,
      bytesSent: 0,
      errors: 0,
      uptime: 0,
    };
  }

  private log(message: string, data?: unknown): void {
    if (this.config.debug) {
      // eslint-disable-next-line no-console
      console.log(`[WebSocket] ${message}`, data || '');
    }
  }
}

// ============================================
// Singleton Factory
// ============================================

let instance: WebSocketServiceImpl | null = null;

export const createWebSocketService = (
  config: ConnectionConfig
): WebSocketServiceImpl => {
  if (instance) {
    instance.disconnect();
  }
  instance = new WebSocketServiceImpl(config);
  return instance;
};

export const getWebSocketService = (): WebSocketServiceImpl | null => {
  return instance;
};

export default WebSocketServiceImpl;
