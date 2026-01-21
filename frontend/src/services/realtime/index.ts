/**
 * Real-time Services - WebSocket connection and live updates
 */

// Core services
export {
  WebSocketServiceImpl,
  createWebSocketService,
  getWebSocketService,
} from './WebSocketService';

export {
  UpdateHandlerManager,
  OptimisticUpdateManager,
  createEntityHandler,
  getUpdateHandlerManager,
  resetUpdateHandlerManager,
} from './UpdateHandlers';

export {
  ConnectionManager,
  createConnectionManager,
  getConnectionManager,
} from './ConnectionManager';

// React hooks
export {
  WebSocketProvider,
  useWebSocketContext,
  useWebSocket,
  useSubscription,
  useRealtimeEntity,
  usePresence,
  useNotifications,
  useConnectionHealth,
  useRealtimeSync,
  useLiveUpdates,
} from './hooks';

// Types
export type {
  // Connection types
  ConnectionStatus,
  ConnectionState,
  ConnectionConfig,
  ConnectionMetrics,
  CloseReason,
  // Message types
  MessageType,
  EntityType,
  BaseMessage,
  WebSocketMessage,
  PingMessage,
  PongMessage,
  SubscribeMessage,
  UnsubscribeMessage,
  PublishMessage,
  AckMessage,
  ErrorMessage,
  NotificationMessage,
  EntityUpdateMessage,
  BatchMessage,
  SyncMessage,
  PresenceMessage,
  // Channel types
  Channel,
  ChannelSubscription,
  // Handler types
  MessageHandler,
  ConnectionHandler,
  ErrorHandler as MessageErrorHandler,
  MessageHandlerMap,
  // Event types
  WebSocketEventType,
  WebSocketEvent,
  WebSocketEventHandler,
  // Queue types
  QueuedMessage,
  MessageQueueConfig,
  // Service interface
  WebSocketService,
  // Hook types
  UseWebSocketOptions,
  UseWebSocketReturn,
  UseSubscriptionOptions,
  UseRealtimeEntityOptions,
} from '../../types/realtime';

// Utility functions
export {
  generateMessageId,
  createMessage,
  parseCloseCode,
  isRecoverableError,
  calculateBackoff,
  DEFAULT_CONNECTION_CONFIG,
  DEFAULT_QUEUE_CONFIG,
} from '../../types/realtime';

// Handler types from UpdateHandlers
export type {
  EntityUpdateHandler,
  NotificationHandler,
  PresenceHandler,
  ErrorHandler,
  SyncHandler,
} from './UpdateHandlers';

// Connection manager types
export type {
  ConnectionHealth,
  ConnectionManagerConfig,
  ConnectionPool,
} from './ConnectionManager';
