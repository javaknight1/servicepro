/**
 * Update Handlers - Handle different message types and entity updates
 */

import {
  WebSocketMessage,
  MessageType,
  MessageHandler,
  EntityType,
  EntityUpdateMessage,
  NotificationMessage,
  BatchMessage,
  SyncMessage,
  PresenceMessage,
  ErrorMessage,
  AckMessage,
} from '../../types/realtime';

// ============================================
// Handler Registry
// ============================================

export type EntityUpdateHandler<T = unknown> = (
  entityId: string,
  data: T,
  operation: 'create' | 'update' | 'delete'
) => void | Promise<void>;

export type NotificationHandler = (notification: NotificationMessage) => void;

export type PresenceHandler = (presence: PresenceMessage) => void;

export type ErrorHandler = (error: ErrorMessage) => void;

export type SyncHandler<T = unknown> = (
  entity: EntityType,
  data: T[],
  hasMore: boolean,
  cursor?: string
) => void | Promise<void>;

interface HandlerRegistry {
  entity: Map<EntityType, Set<EntityUpdateHandler>>;
  notification: Set<NotificationHandler>;
  presence: Set<PresenceHandler>;
  error: Set<ErrorHandler>;
  sync: Map<EntityType, Set<SyncHandler>>;
}

// ============================================
// Update Handler Manager
// ============================================

export class UpdateHandlerManager {
  private handlers: HandlerRegistry = {
    entity: new Map(),
    notification: new Set(),
    presence: new Set(),
    error: new Set(),
    sync: new Map(),
  };

  private batchBuffer: WebSocketMessage[] = [];
  private batchTimeout: NodeJS.Timeout | null = null;
  private batchDelay = 50; // ms
  private maxBatchSize = 100;

  // Performance tracking
  private handlerMetrics: Map<
    string,
    {
      calls: number;
      totalTime: number;
      errors: number;
    }
  > = new Map();

  // ============================================
  // Entity Handlers
  // ============================================

  onEntityUpdate<T = unknown>(
    entity: EntityType,
    handler: EntityUpdateHandler<T>
  ): () => void {
    if (!this.handlers.entity.has(entity)) {
      this.handlers.entity.set(entity, new Set());
    }
    this.handlers.entity.get(entity)!.add(handler as EntityUpdateHandler);

    return () => {
      this.handlers.entity.get(entity)?.delete(handler as EntityUpdateHandler);
    };
  }

  offEntityUpdate(entity: EntityType, handler?: EntityUpdateHandler): void {
    if (handler) {
      this.handlers.entity.get(entity)?.delete(handler);
    } else {
      this.handlers.entity.delete(entity);
    }
  }

  // ============================================
  // Notification Handlers
  // ============================================

  onNotification(handler: NotificationHandler): () => void {
    this.handlers.notification.add(handler);
    return () => this.handlers.notification.delete(handler);
  }

  offNotification(handler: NotificationHandler): void {
    this.handlers.notification.delete(handler);
  }

  // ============================================
  // Presence Handlers
  // ============================================

  onPresence(handler: PresenceHandler): () => void {
    this.handlers.presence.add(handler);
    return () => this.handlers.presence.delete(handler);
  }

  offPresence(handler: PresenceHandler): void {
    this.handlers.presence.delete(handler);
  }

  // ============================================
  // Error Handlers
  // ============================================

  onError(handler: ErrorHandler): () => void {
    this.handlers.error.add(handler);
    return () => this.handlers.error.delete(handler);
  }

  offError(handler: ErrorHandler): void {
    this.handlers.error.delete(handler);
  }

  // ============================================
  // Sync Handlers
  // ============================================

  onSync<T = unknown>(entity: EntityType, handler: SyncHandler<T>): () => void {
    if (!this.handlers.sync.has(entity)) {
      this.handlers.sync.set(entity, new Set());
    }
    this.handlers.sync.get(entity)!.add(handler as SyncHandler);

    return () => {
      this.handlers.sync.get(entity)?.delete(handler as SyncHandler);
    };
  }

  offSync(entity: EntityType, handler?: SyncHandler): void {
    if (handler) {
      this.handlers.sync.get(entity)?.delete(handler);
    } else {
      this.handlers.sync.delete(entity);
    }
  }

  // ============================================
  // Message Processing
  // ============================================

  /**
   * Process a single message
   */
  async handleMessage(message: WebSocketMessage): Promise<void> {
    const startTime = performance.now();
    const metricKey = message.type;

    try {
      switch (message.type) {
        case 'create':
        case 'update':
        case 'delete':
          await this.handleEntityUpdate(message as EntityUpdateMessage);
          break;

        case 'notification':
          await this.handleNotification(message as NotificationMessage);
          break;

        case 'presence':
          await this.handlePresence(message as PresenceMessage);
          break;

        case 'error':
          await this.handleError(message as ErrorMessage);
          break;

        case 'sync':
          await this.handleSync(message as SyncMessage);
          break;

        case 'batch':
          await this.handleBatch(message as BatchMessage);
          break;

        case 'ack':
          // ACKs are typically handled by the WebSocketService
          break;

        default:
          console.warn(`Unhandled message type: ${message.type}`);
      }

      this.recordMetric(metricKey, performance.now() - startTime, false);
    } catch (error) {
      this.recordMetric(metricKey, performance.now() - startTime, true);
      console.error(`Error handling message ${message.type}:`, error);
      throw error;
    }
  }

  /**
   * Process messages in batch for performance
   */
  async handleMessageBatch(messages: WebSocketMessage[]): Promise<void> {
    // Group messages by type for optimized processing
    const grouped = this.groupMessagesByType(messages);

    // Process entity updates together
    const entityUpdates = [
      ...(grouped.create || []),
      ...(grouped.update || []),
      ...(grouped.delete || []),
    ] as EntityUpdateMessage[];

    if (entityUpdates.length > 0) {
      await this.handleEntityUpdateBatch(entityUpdates);
    }

    // Process other message types
    for (const notification of (grouped.notification ||
      []) as NotificationMessage[]) {
      await this.handleNotification(notification);
    }

    for (const presence of (grouped.presence || []) as PresenceMessage[]) {
      await this.handlePresence(presence);
    }

    for (const error of (grouped.error || []) as ErrorMessage[]) {
      await this.handleError(error);
    }

    for (const sync of (grouped.sync || []) as SyncMessage[]) {
      await this.handleSync(sync);
    }
  }

  /**
   * Buffer messages for batch processing
   */
  bufferMessage(message: WebSocketMessage): void {
    this.batchBuffer.push(message);

    // Process immediately if buffer is full
    if (this.batchBuffer.length >= this.maxBatchSize) {
      this.flushBuffer();
      return;
    }

    // Set timeout to process buffer
    if (!this.batchTimeout) {
      this.batchTimeout = setTimeout(() => {
        this.flushBuffer();
      }, this.batchDelay);
    }
  }

  private async flushBuffer(): Promise<void> {
    if (this.batchTimeout) {
      clearTimeout(this.batchTimeout);
      this.batchTimeout = null;
    }

    if (this.batchBuffer.length === 0) return;

    const messages = [...this.batchBuffer];
    this.batchBuffer = [];

    await this.handleMessageBatch(messages);
  }

  // ============================================
  // Specific Message Handlers
  // ============================================

  private async handleEntityUpdate(
    message: EntityUpdateMessage
  ): Promise<void> {
    const handlers = this.handlers.entity.get(message.entity);
    if (!handlers || handlers.size === 0) return;

    const operation = message.type as 'create' | 'update' | 'delete';

    const promises = Array.from(handlers).map((handler) =>
      Promise.resolve(handler(message.entityId, message.data, operation))
    );

    await Promise.all(promises);
  }

  private async handleEntityUpdateBatch(
    messages: EntityUpdateMessage[]
  ): Promise<void> {
    // Group by entity type
    const byEntity = new Map<EntityType, EntityUpdateMessage[]>();
    for (const message of messages) {
      if (!byEntity.has(message.entity)) {
        byEntity.set(message.entity, []);
      }
      byEntity.get(message.entity)!.push(message);
    }

    // Process each entity type
    for (const [entityType, entityMessages] of byEntity) {
      const handlers = this.handlers.entity.get(entityType);
      if (!handlers || handlers.size === 0) continue;

      for (const message of entityMessages) {
        const operation = message.type as 'create' | 'update' | 'delete';
        for (const handler of handlers) {
          try {
            await handler(message.entityId, message.data, operation);
          } catch (error) {
            console.error(`Handler error for ${entityType}:`, error);
          }
        }
      }
    }
  }

  private async handleNotification(
    message: NotificationMessage
  ): Promise<void> {
    for (const handler of this.handlers.notification) {
      try {
        handler(message);
      } catch (error) {
        console.error('Notification handler error:', error);
      }
    }
  }

  private async handlePresence(message: PresenceMessage): Promise<void> {
    for (const handler of this.handlers.presence) {
      try {
        handler(message);
      } catch (error) {
        console.error('Presence handler error:', error);
      }
    }
  }

  private async handleError(message: ErrorMessage): Promise<void> {
    for (const handler of this.handlers.error) {
      try {
        handler(message);
      } catch (error) {
        console.error('Error handler error:', error);
      }
    }
  }

  private async handleSync(message: SyncMessage): Promise<void> {
    const handlers = this.handlers.sync.get(message.entity);
    if (!handlers || handlers.size === 0) return;

    for (const handler of handlers) {
      try {
        await handler(
          message.entity,
          message.data,
          message.hasMore,
          message.cursor
        );
      } catch (error) {
        console.error(`Sync handler error for ${message.entity}:`, error);
      }
    }
  }

  private async handleBatch(message: BatchMessage): Promise<void> {
    await this.handleMessageBatch(message.messages);
  }

  // ============================================
  // Utilities
  // ============================================

  private groupMessagesByType(
    messages: WebSocketMessage[]
  ): Partial<Record<MessageType, WebSocketMessage[]>> {
    const grouped: Partial<Record<MessageType, WebSocketMessage[]>> = {};

    for (const message of messages) {
      if (!grouped[message.type]) {
        grouped[message.type] = [];
      }
      grouped[message.type]!.push(message);
    }

    return grouped;
  }

  private recordMetric(key: string, time: number, hasError: boolean): void {
    if (!this.handlerMetrics.has(key)) {
      this.handlerMetrics.set(key, { calls: 0, totalTime: 0, errors: 0 });
    }

    const metric = this.handlerMetrics.get(key)!;
    metric.calls++;
    metric.totalTime += time;
    if (hasError) metric.errors++;
  }

  /**
   * Get handler performance metrics
   */
  getMetrics(): Map<
    string,
    {
      calls: number;
      averageTime: number;
      errors: number;
    }
  > {
    const result = new Map<
      string,
      {
        calls: number;
        averageTime: number;
        errors: number;
      }
    >();

    for (const [key, metric] of this.handlerMetrics) {
      result.set(key, {
        calls: metric.calls,
        averageTime: metric.calls > 0 ? metric.totalTime / metric.calls : 0,
        errors: metric.errors,
      });
    }

    return result;
  }

  /**
   * Reset all metrics
   */
  resetMetrics(): void {
    this.handlerMetrics.clear();
  }

  /**
   * Clear all handlers
   */
  clearAllHandlers(): void {
    this.handlers.entity.clear();
    this.handlers.notification.clear();
    this.handlers.presence.clear();
    this.handlers.error.clear();
    this.handlers.sync.clear();
  }

  /**
   * Set batch processing configuration
   */
  setBatchConfig(config: { delay?: number; maxSize?: number }): void {
    if (config.delay !== undefined) this.batchDelay = config.delay;
    if (config.maxSize !== undefined) this.maxBatchSize = config.maxSize;
  }
}

// ============================================
// Specialized Entity Handlers
// ============================================

/**
 * Create a typed entity handler for a specific entity type
 */
export function createEntityHandler<T>(
  manager: UpdateHandlerManager,
  entity: EntityType,
  callbacks: {
    onCreate?: (id: string, data: T) => void | Promise<void>;
    onUpdate?: (id: string, data: T) => void | Promise<void>;
    onDelete?: (id: string) => void | Promise<void>;
  }
): () => void {
  return manager.onEntityUpdate<T>(entity, (id, data, operation) => {
    switch (operation) {
      case 'create':
        return callbacks.onCreate?.(id, data);
      case 'update':
        return callbacks.onUpdate?.(id, data);
      case 'delete':
        return callbacks.onDelete?.(id);
    }
  });
}

// ============================================
// Optimistic Update Handler
// ============================================

interface OptimisticUpdate<T> {
  id: string;
  entity: EntityType;
  entityId: string;
  optimisticData: T;
  originalData: T | null;
  timestamp: number;
  committed: boolean;
  rolledBack: boolean;
}

export class OptimisticUpdateManager<T = unknown> {
  private updates: Map<string, OptimisticUpdate<T>> = new Map();
  private rollbackCallbacks: Map<string, (data: T | null) => void> = new Map();
  private timeout: number;

  constructor(timeoutMs: number = 10000) {
    this.timeout = timeoutMs;
  }

  /**
   * Apply an optimistic update
   */
  apply(
    entity: EntityType,
    entityId: string,
    optimisticData: T,
    originalData: T | null,
    onRollback?: (data: T | null) => void
  ): string {
    const id = `${entity}-${entityId}-${Date.now()}`;

    this.updates.set(id, {
      id,
      entity,
      entityId,
      optimisticData,
      originalData,
      timestamp: Date.now(),
      committed: false,
      rolledBack: false,
    });

    if (onRollback) {
      this.rollbackCallbacks.set(id, onRollback);
    }

    // Set timeout for auto-rollback
    setTimeout(() => {
      this.rollbackIfPending(id);
    }, this.timeout);

    return id;
  }

  /**
   * Commit an optimistic update (server confirmed)
   */
  commit(updateId: string): boolean {
    const update = this.updates.get(updateId);
    if (!update || update.rolledBack) return false;

    update.committed = true;
    this.cleanup(updateId);
    return true;
  }

  /**
   * Rollback an optimistic update
   */
  rollback(updateId: string): T | null {
    const update = this.updates.get(updateId);
    if (!update || update.committed) return null;

    update.rolledBack = true;
    const callback = this.rollbackCallbacks.get(updateId);
    if (callback) {
      callback(update.originalData);
    }

    this.cleanup(updateId);
    return update.originalData;
  }

  /**
   * Rollback if update is still pending
   */
  private rollbackIfPending(updateId: string): void {
    const update = this.updates.get(updateId);
    if (update && !update.committed && !update.rolledBack) {
      this.rollback(updateId);
    }
  }

  /**
   * Get pending updates for an entity
   */
  getPendingUpdates(
    entity: EntityType,
    entityId: string
  ): OptimisticUpdate<T>[] {
    return Array.from(this.updates.values()).filter(
      (u) =>
        u.entity === entity &&
        u.entityId === entityId &&
        !u.committed &&
        !u.rolledBack
    );
  }

  /**
   * Check if entity has pending updates
   */
  hasPendingUpdates(entity: EntityType, entityId: string): boolean {
    return this.getPendingUpdates(entity, entityId).length > 0;
  }

  private cleanup(updateId: string): void {
    this.updates.delete(updateId);
    this.rollbackCallbacks.delete(updateId);
  }

  /**
   * Clear all updates
   */
  clear(): void {
    this.updates.clear();
    this.rollbackCallbacks.clear();
  }
}

// ============================================
// Singleton Instance
// ============================================

let updateHandlerInstance: UpdateHandlerManager | null = null;

export const getUpdateHandlerManager = (): UpdateHandlerManager => {
  if (!updateHandlerInstance) {
    updateHandlerInstance = new UpdateHandlerManager();
  }
  return updateHandlerInstance;
};

export const resetUpdateHandlerManager = (): void => {
  if (updateHandlerInstance) {
    updateHandlerInstance.clearAllHandlers();
  }
  updateHandlerInstance = null;
};

export default UpdateHandlerManager;
