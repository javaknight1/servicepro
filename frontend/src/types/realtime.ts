/**
 * Real-time WebSocket Types and Interfaces
 */

// ============================================
// Connection Types
// ============================================

export type ConnectionStatus =
  | 'connecting'
  | 'connected'
  | 'disconnected'
  | 'reconnecting'
  | 'failed';

export type CloseReason =
  | 'normal'
  | 'going_away'
  | 'protocol_error'
  | 'unsupported_data'
  | 'no_status'
  | 'abnormal'
  | 'invalid_payload'
  | 'policy_violation'
  | 'message_too_big'
  | 'internal_error'
  | 'service_restart'
  | 'try_again_later'
  | 'unknown';

export interface ConnectionState {
  status: ConnectionStatus;
  connectedAt: number | null;
  disconnectedAt: number | null;
  reconnectAttempts: number;
  lastError: Error | null;
  latency: number | null;
}

export interface ConnectionConfig {
  /** WebSocket URL */
  url: string;
  /** Authentication token */
  token?: string;
  /** Protocols to use */
  protocols?: string | string[];
  /** Auto-reconnect on disconnect */
  autoReconnect?: boolean;
  /** Maximum reconnection attempts (0 = unlimited) */
  maxReconnectAttempts?: number;
  /** Initial reconnect delay in ms */
  reconnectDelay?: number;
  /** Max reconnect delay in ms */
  maxReconnectDelay?: number;
  /** Reconnect delay multiplier for exponential backoff */
  reconnectDelayMultiplier?: number;
  /** Heartbeat interval in ms (0 = disabled) */
  heartbeatInterval?: number;
  /** Connection timeout in ms */
  connectionTimeout?: number;
  /** Message queue size limit */
  messageQueueSize?: number;
  /** Enable debug logging */
  debug?: boolean;
}

// ============================================
// Message Types
// ============================================

export type MessageType =
  | 'ping'
  | 'pong'
  | 'subscribe'
  | 'unsubscribe'
  | 'publish'
  | 'ack'
  | 'error'
  | 'notification'
  | 'update'
  | 'delete'
  | 'create'
  | 'batch'
  | 'sync'
  | 'presence';

export type EntityType =
  | 'job'
  | 'customer'
  | 'invoice'
  | 'quote'
  | 'payment'
  | 'schedule'
  | 'user'
  | 'notification'
  | 'kpi';

export interface BaseMessage {
  id: string;
  type: MessageType;
  timestamp: number;
}

export interface PingMessage extends BaseMessage {
  type: 'ping';
}

export interface PongMessage extends BaseMessage {
  type: 'pong';
  latency?: number;
}

export interface SubscribeMessage extends BaseMessage {
  type: 'subscribe';
  channel: string;
  filters?: Record<string, unknown>;
}

export interface UnsubscribeMessage extends BaseMessage {
  type: 'unsubscribe';
  channel: string;
}

export interface PublishMessage extends BaseMessage {
  type: 'publish';
  channel: string;
  data: unknown;
}

export interface AckMessage extends BaseMessage {
  type: 'ack';
  messageId: string;
  success: boolean;
  error?: string;
}

export interface ErrorMessage extends BaseMessage {
  type: 'error';
  code: string;
  message: string;
  details?: unknown;
}

export interface NotificationMessage extends BaseMessage {
  type: 'notification';
  title: string;
  body: string;
  level: 'info' | 'success' | 'warning' | 'error';
  action?: {
    label: string;
    url: string;
  };
}

export interface EntityUpdateMessage<T = unknown> extends BaseMessage {
  type: 'update' | 'create' | 'delete';
  entity: EntityType;
  entityId: string;
  data: T;
  userId?: string;
  version?: number;
}

export interface BatchMessage extends BaseMessage {
  type: 'batch';
  messages: WebSocketMessage[];
}

export interface SyncMessage extends BaseMessage {
  type: 'sync';
  entity: EntityType;
  since: number;
  data: unknown[];
  hasMore: boolean;
  cursor?: string;
}

export interface PresenceMessage extends BaseMessage {
  type: 'presence';
  userId: string;
  status: 'online' | 'away' | 'offline';
  lastSeen?: number;
}

export type WebSocketMessage =
  | PingMessage
  | PongMessage
  | SubscribeMessage
  | UnsubscribeMessage
  | PublishMessage
  | AckMessage
  | ErrorMessage
  | NotificationMessage
  | EntityUpdateMessage
  | BatchMessage
  | SyncMessage
  | PresenceMessage;

// ============================================
// Channel Types
// ============================================

export interface Channel {
  name: string;
  subscribed: boolean;
  subscribedAt: number | null;
  filters?: Record<string, unknown>;
  messageCount: number;
  lastMessageAt: number | null;
}

export interface ChannelSubscription {
  channel: string;
  filters?: Record<string, unknown>;
  onMessage: (message: WebSocketMessage) => void;
  onError?: (error: Error) => void;
}

// ============================================
// Handler Types
// ============================================

export type MessageHandler<T extends WebSocketMessage = WebSocketMessage> = (
  message: T
) => void | Promise<void>;

export type ConnectionHandler = (state: ConnectionState) => void;

export type ErrorHandler = (error: Error, context?: string) => void;

export interface MessageHandlerMap {
  ping?: MessageHandler<PingMessage>;
  pong?: MessageHandler<PongMessage>;
  subscribe?: MessageHandler<SubscribeMessage>;
  unsubscribe?: MessageHandler<UnsubscribeMessage>;
  publish?: MessageHandler<PublishMessage>;
  ack?: MessageHandler<AckMessage>;
  error?: MessageHandler<ErrorMessage>;
  notification?: MessageHandler<NotificationMessage>;
  update?: MessageHandler<EntityUpdateMessage>;
  create?: MessageHandler<EntityUpdateMessage>;
  delete?: MessageHandler<EntityUpdateMessage>;
  batch?: MessageHandler<BatchMessage>;
  sync?: MessageHandler<SyncMessage>;
  presence?: MessageHandler<PresenceMessage>;
}

// ============================================
// Event Types
// ============================================

export type WebSocketEventType =
  | 'open'
  | 'close'
  | 'error'
  | 'message'
  | 'reconnect'
  | 'reconnect_failed'
  | 'latency';

export interface WebSocketEvent {
  type: WebSocketEventType;
  timestamp: number;
  data?: unknown;
}

export type WebSocketEventHandler = (event: WebSocketEvent) => void;

// ============================================
// Queue Types
// ============================================

export interface QueuedMessage {
  id: string;
  message: WebSocketMessage;
  priority: number;
  retries: number;
  maxRetries: number;
  createdAt: number;
  sentAt?: number;
}

export interface MessageQueueConfig {
  maxSize: number;
  maxRetries: number;
  retryDelay: number;
  priorityLevels: number;
}

// ============================================
// Metrics Types
// ============================================

export interface ConnectionMetrics {
  totalConnections: number;
  successfulConnections: number;
  failedConnections: number;
  totalReconnects: number;
  totalDisconnects: number;
  averageConnectionDuration: number;
  averageLatency: number;
  messagesSent: number;
  messagesReceived: number;
  bytesReceived: number;
  bytesSent: number;
  errors: number;
  lastError?: Error;
  uptime: number;
}

// ============================================
// Service Interface
// ============================================

export interface WebSocketService {
  // Connection
  connect(): Promise<void>;
  disconnect(code?: number, reason?: string): void;
  reconnect(): Promise<void>;
  getConnectionState(): ConnectionState;
  isConnected(): boolean;

  // Messaging
  send(message: WebSocketMessage): Promise<void>;
  sendAndWait<T extends WebSocketMessage>(
    message: WebSocketMessage,
    timeout?: number
  ): Promise<T>;

  // Subscriptions
  subscribe(subscription: ChannelSubscription): () => void;
  unsubscribe(channel: string): void;
  getSubscriptions(): Channel[];

  // Events
  on(event: WebSocketEventType, handler: WebSocketEventHandler): () => void;
  off(event: WebSocketEventType, handler: WebSocketEventHandler): void;

  // Handlers
  registerHandler<T extends MessageType>(
    type: T,
    handler: MessageHandler
  ): () => void;
  unregisterHandler(type: MessageType): void;

  // Metrics
  getMetrics(): ConnectionMetrics;
  resetMetrics(): void;

  // Utilities
  ping(): Promise<number>;
  setToken(token: string): void;
}

// ============================================
// React Hook Types
// ============================================

export interface UseWebSocketOptions {
  url: string;
  token?: string;
  autoConnect?: boolean;
  reconnect?: boolean;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Error) => void;
  onMessage?: (message: WebSocketMessage) => void;
}

export interface UseWebSocketReturn {
  isConnected: boolean;
  status: ConnectionStatus;
  latency: number | null;
  error: Error | null;
  connect: () => Promise<void>;
  disconnect: () => void;
  send: (message: WebSocketMessage) => Promise<void>;
  subscribe: (subscription: ChannelSubscription) => () => void;
}

export interface UseSubscriptionOptions<T = unknown> {
  channel: string;
  filters?: Record<string, unknown>;
  onMessage?: (data: T) => void;
  enabled?: boolean;
}

export interface UseRealtimeEntityOptions<T = unknown> {
  entity: EntityType;
  entityId?: string;
  initialData?: T;
  onUpdate?: (data: T) => void;
  onCreate?: (data: T) => void;
  onDelete?: (id: string) => void;
}

// ============================================
// Utility Functions
// ============================================

export const generateMessageId = (): string => {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
};

export const createMessage = <T extends WebSocketMessage>(
  type: T['type'],
  data: Omit<T, 'id' | 'type' | 'timestamp'>
): T => {
  return {
    id: generateMessageId(),
    type,
    timestamp: Date.now(),
    ...data,
  } as T;
};

export const parseCloseCode = (code: number): CloseReason => {
  const codeMap: Record<number, CloseReason> = {
    1000: 'normal',
    1001: 'going_away',
    1002: 'protocol_error',
    1003: 'unsupported_data',
    1005: 'no_status',
    1006: 'abnormal',
    1007: 'invalid_payload',
    1008: 'policy_violation',
    1009: 'message_too_big',
    1011: 'internal_error',
    1012: 'service_restart',
    1013: 'try_again_later',
  };
  return codeMap[code] || 'unknown';
};

export const isRecoverableError = (code: number): boolean => {
  // Normal closure or abnormal closures that might recover
  return [1000, 1001, 1006, 1012, 1013].includes(code);
};

export const calculateBackoff = (
  attempt: number,
  baseDelay: number,
  maxDelay: number,
  multiplier: number = 2
): number => {
  const delay = baseDelay * Math.pow(multiplier, attempt);
  // Add jitter (±10%)
  const jitter = delay * 0.1 * (Math.random() * 2 - 1);
  return Math.min(delay + jitter, maxDelay);
};

// ============================================
// Default Configuration
// ============================================

export const DEFAULT_CONNECTION_CONFIG: Required<ConnectionConfig> = {
  url: '',
  token: '',
  protocols: [],
  autoReconnect: true,
  maxReconnectAttempts: 10,
  reconnectDelay: 1000,
  maxReconnectDelay: 30000,
  reconnectDelayMultiplier: 2,
  heartbeatInterval: 30000,
  connectionTimeout: 10000,
  messageQueueSize: 100,
  debug: false,
};

export const DEFAULT_QUEUE_CONFIG: MessageQueueConfig = {
  maxSize: 100,
  maxRetries: 3,
  retryDelay: 1000,
  priorityLevels: 3,
};
