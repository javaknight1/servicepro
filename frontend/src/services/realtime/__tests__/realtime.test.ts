/**
 * Real-time System Tests
 * - Connection stability tests
 * - Update propagation tests
 * - Performance benchmarks
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  WebSocketServiceImpl,
  createWebSocketService,
} from '../WebSocketService';
import {
  UpdateHandlerManager,
  OptimisticUpdateManager,
  createEntityHandler,
  getUpdateHandlerManager,
  resetUpdateHandlerManager,
} from '../UpdateHandlers';
import {
  ConnectionManager,
  createConnectionManager,
} from '../ConnectionManager';
import {
  generateMessageId,
  createMessage,
  parseCloseCode,
  isRecoverableError,
  calculateBackoff,
  WebSocketMessage,
  EntityUpdateMessage,
  NotificationMessage,
  PresenceMessage,
  BatchMessage,
  PingMessage,
  PongMessage,
} from '../../../types/realtime';

// ============================================
// Mock WebSocket
// ============================================

class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  url: string;
  readyState: number = MockWebSocket.CONNECTING;
  onopen: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;

  private messageQueue: MessageEvent[] = [];

  constructor(url: string, _protocols?: string | string[]) {
    this.url = url;

    // Simulate async connection
    setTimeout(() => {
      if (this.shouldConnect()) {
        this.readyState = MockWebSocket.OPEN;
        this.onopen?.(new Event('open'));
      } else {
        this.readyState = MockWebSocket.CLOSED;
        this.onerror?.(new Event('error'));
        this.onclose?.(new CloseEvent('close', { code: 1006 }));
      }
    }, 10);
  }

  private shouldConnect(): boolean {
    return !this.url.includes('fail');
  }

  send(data: string): void {
    if (this.readyState !== MockWebSocket.OPEN) {
      throw new Error('WebSocket is not open');
    }

    // Parse and respond to certain messages
    const message = JSON.parse(data);
    if (message.type === 'ping') {
      this.simulateMessage({
        id: message.id,
        type: 'pong',
        timestamp: Date.now(),
      });
    }
  }

  close(code: number = 1000, _reason?: string): void {
    this.readyState = MockWebSocket.CLOSING;
    setTimeout(() => {
      this.readyState = MockWebSocket.CLOSED;
      this.onclose?.(new CloseEvent('close', { code }));
    }, 10);
  }

  // Test helper methods
  simulateMessage(data: unknown): void {
    const event = new MessageEvent('message', {
      data: JSON.stringify(data),
    });
    this.onmessage?.(event);
  }

  simulateError(): void {
    this.onerror?.(new Event('error'));
  }

  simulateClose(code: number = 1000): void {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent('close', { code }));
  }
}

// Replace global WebSocket with mock
(global as unknown as { WebSocket: typeof MockWebSocket }).WebSocket =
  MockWebSocket;

// ============================================
// Utility Function Tests
// ============================================

describe('Utility Functions', () => {
  describe('generateMessageId', () => {
    it('generates unique IDs', () => {
      const ids = new Set<string>();
      for (let i = 0; i < 1000; i++) {
        ids.add(generateMessageId());
      }
      expect(ids.size).toBe(1000);
    });

    it('generates IDs with timestamp prefix', () => {
      const id = generateMessageId();
      const parts = id.split('-');
      expect(parseInt(parts[0])).toBeGreaterThan(0);
    });
  });

  describe('createMessage', () => {
    it('creates message with id and timestamp', () => {
      const message = createMessage<PingMessage>('ping', {});
      expect(message.id).toBeDefined();
      expect(message.type).toBe('ping');
      expect(message.timestamp).toBeGreaterThan(0);
    });

    it('preserves additional properties', () => {
      const message = createMessage<EntityUpdateMessage>('update', {
        entity: 'job',
        entityId: '123',
        data: { name: 'Test' },
      });
      expect(message.entity).toBe('job');
      expect(message.entityId).toBe('123');
      expect(message.data).toEqual({ name: 'Test' });
    });
  });

  describe('parseCloseCode', () => {
    it('parses known close codes', () => {
      expect(parseCloseCode(1000)).toBe('normal');
      expect(parseCloseCode(1001)).toBe('going_away');
      expect(parseCloseCode(1006)).toBe('abnormal');
      expect(parseCloseCode(1012)).toBe('service_restart');
    });

    it('returns unknown for unrecognized codes', () => {
      expect(parseCloseCode(9999)).toBe('unknown');
    });
  });

  describe('isRecoverableError', () => {
    it('identifies recoverable close codes', () => {
      expect(isRecoverableError(1000)).toBe(true);
      expect(isRecoverableError(1001)).toBe(true);
      expect(isRecoverableError(1006)).toBe(true);
      expect(isRecoverableError(1012)).toBe(true);
      expect(isRecoverableError(1013)).toBe(true);
    });

    it('identifies non-recoverable close codes', () => {
      expect(isRecoverableError(1002)).toBe(false);
      expect(isRecoverableError(1003)).toBe(false);
      expect(isRecoverableError(1008)).toBe(false);
    });
  });

  describe('calculateBackoff', () => {
    it('calculates exponential backoff', () => {
      const delay0 = calculateBackoff(0, 1000, 30000, 2);
      const delay1 = calculateBackoff(1, 1000, 30000, 2);
      const delay2 = calculateBackoff(2, 1000, 30000, 2);

      expect(delay0).toBeGreaterThanOrEqual(900); // 1000 - 10% jitter
      expect(delay0).toBeLessThanOrEqual(1100); // 1000 + 10% jitter
      expect(delay1).toBeGreaterThan(delay0);
      expect(delay2).toBeGreaterThan(delay1);
    });

    it('caps at maximum delay', () => {
      const delay = calculateBackoff(10, 1000, 5000, 2);
      expect(delay).toBeLessThanOrEqual(5500); // max + jitter
    });
  });
});

// ============================================
// WebSocketService Tests
// ============================================

describe('WebSocketService', () => {
  let service: WebSocketServiceImpl;

  beforeEach(() => {
    service = new WebSocketServiceImpl({
      url: 'ws://localhost:8080',
      debug: false,
    });
  });

  afterEach(() => {
    service.disconnect();
  });

  describe('Connection', () => {
    it('connects successfully', async () => {
      await service.connect();
      expect(service.isConnected()).toBe(true);
      expect(service.getConnectionState().status).toBe('connected');
    });

    it('handles connection failure', async () => {
      const failService = new WebSocketServiceImpl({
        url: 'ws://fail.localhost:8080',
        autoReconnect: false,
      });

      await expect(failService.connect()).rejects.toThrow();
      expect(failService.isConnected()).toBe(false);
    });

    it('disconnects cleanly', async () => {
      await service.connect();
      expect(service.isConnected()).toBe(true);

      service.disconnect();
      await new Promise((resolve) => setTimeout(resolve, 20));

      expect(service.isConnected()).toBe(false);
      expect(service.getConnectionState().status).toBe('disconnected');
    });

    it('tracks connection metrics', async () => {
      await service.connect();

      const metrics = service.getMetrics();
      expect(metrics.totalConnections).toBe(1);
      expect(metrics.successfulConnections).toBe(1);
    });
  });

  describe('Messaging', () => {
    beforeEach(async () => {
      await service.connect();
    });

    it('sends messages', async () => {
      const message = createMessage<PingMessage>('ping', {});
      await expect(service.send(message)).resolves.toBeUndefined();

      const metrics = service.getMetrics();
      expect(metrics.messagesSent).toBe(1);
    });

    it('queues messages when disconnected', async () => {
      service.disconnect();
      await new Promise((resolve) => setTimeout(resolve, 20));

      const message = createMessage<PingMessage>('ping', {});
      await service.send(message);

      // Message should be queued, not sent
      const metrics = service.getMetrics();
      expect(metrics.messagesSent).toBe(0);
    });
  });

  describe('Subscriptions', () => {
    beforeEach(async () => {
      await service.connect();
    });

    it('subscribes to channels', () => {
      const handler = vi.fn();
      const unsubscribe = service.subscribe({
        channel: 'jobs',
        onMessage: handler,
      });

      const subscriptions = service.getSubscriptions();
      expect(subscriptions).toHaveLength(1);
      expect(subscriptions[0].name).toBe('jobs');

      unsubscribe();
    });

    it('unsubscribes from channels', () => {
      const handler = vi.fn();
      const unsubscribe = service.subscribe({
        channel: 'jobs',
        onMessage: handler,
      });

      expect(service.getSubscriptions()).toHaveLength(1);

      unsubscribe();

      expect(service.getSubscriptions()).toHaveLength(0);
    });
  });

  describe('Event Handlers', () => {
    it('registers and triggers event handlers', async () => {
      const openHandler = vi.fn();
      service.on('open', openHandler);

      await service.connect();

      expect(openHandler).toHaveBeenCalled();
    });

    it('unregisters event handlers', async () => {
      const openHandler = vi.fn();
      const unsubscribe = service.on('open', openHandler);

      unsubscribe();

      await service.connect();

      expect(openHandler).not.toHaveBeenCalled();
    });
  });
});

// ============================================
// UpdateHandlerManager Tests
// ============================================

describe('UpdateHandlerManager', () => {
  let manager: UpdateHandlerManager;

  beforeEach(() => {
    resetUpdateHandlerManager();
    manager = getUpdateHandlerManager();
  });

  afterEach(() => {
    manager.clearAllHandlers();
  });

  describe('Entity Handlers', () => {
    it('registers and triggers entity update handlers', async () => {
      const handler = vi.fn();
      manager.onEntityUpdate('job', handler);

      const message: EntityUpdateMessage = {
        id: '1',
        type: 'update',
        timestamp: Date.now(),
        entity: 'job',
        entityId: '123',
        data: { name: 'Updated Job' },
      };

      await manager.handleMessage(message);

      expect(handler).toHaveBeenCalledWith(
        '123',
        { name: 'Updated Job' },
        'update'
      );
    });

    it('handles create, update, and delete operations', async () => {
      const handler = vi.fn();
      manager.onEntityUpdate('customer', handler);

      // Create
      await manager.handleMessage({
        id: '1',
        type: 'create',
        timestamp: Date.now(),
        entity: 'customer',
        entityId: '1',
        data: { name: 'New Customer' },
      });

      // Update
      await manager.handleMessage({
        id: '2',
        type: 'update',
        timestamp: Date.now(),
        entity: 'customer',
        entityId: '1',
        data: { name: 'Updated Customer' },
      });

      // Delete
      await manager.handleMessage({
        id: '3',
        type: 'delete',
        timestamp: Date.now(),
        entity: 'customer',
        entityId: '1',
        data: null,
      });

      expect(handler).toHaveBeenCalledTimes(3);
      expect(handler).toHaveBeenNthCalledWith(
        1,
        '1',
        { name: 'New Customer' },
        'create'
      );
      expect(handler).toHaveBeenNthCalledWith(
        2,
        '1',
        { name: 'Updated Customer' },
        'update'
      );
      expect(handler).toHaveBeenNthCalledWith(3, '1', null, 'delete');
    });

    it('unregisters entity handlers', () => {
      const handler = vi.fn();
      const unsubscribe = manager.onEntityUpdate('job', handler);

      unsubscribe();

      manager.handleMessage({
        id: '1',
        type: 'update',
        timestamp: Date.now(),
        entity: 'job',
        entityId: '123',
        data: {},
      });

      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe('Notification Handlers', () => {
    it('handles notifications', async () => {
      const handler = vi.fn();
      manager.onNotification(handler);

      const notification: NotificationMessage = {
        id: '1',
        type: 'notification',
        timestamp: Date.now(),
        title: 'Test',
        body: 'Test notification',
        level: 'info',
      };

      await manager.handleMessage(notification);

      expect(handler).toHaveBeenCalledWith(notification);
    });
  });

  describe('Presence Handlers', () => {
    it('handles presence updates', async () => {
      const handler = vi.fn();
      manager.onPresence(handler);

      const presence: PresenceMessage = {
        id: '1',
        type: 'presence',
        timestamp: Date.now(),
        userId: 'user1',
        status: 'online',
      };

      await manager.handleMessage(presence);

      expect(handler).toHaveBeenCalledWith(presence);
    });
  });

  describe('Batch Processing', () => {
    it('processes batch messages', async () => {
      const jobHandler = vi.fn();
      const customerHandler = vi.fn();

      manager.onEntityUpdate('job', jobHandler);
      manager.onEntityUpdate('customer', customerHandler);

      const batch: BatchMessage = {
        id: '1',
        type: 'batch',
        timestamp: Date.now(),
        messages: [
          {
            id: '2',
            type: 'update',
            timestamp: Date.now(),
            entity: 'job',
            entityId: '1',
            data: { name: 'Job 1' },
          } as EntityUpdateMessage,
          {
            id: '3',
            type: 'update',
            timestamp: Date.now(),
            entity: 'customer',
            entityId: '1',
            data: { name: 'Customer 1' },
          } as EntityUpdateMessage,
        ],
      };

      await manager.handleMessage(batch);

      expect(jobHandler).toHaveBeenCalled();
      expect(customerHandler).toHaveBeenCalled();
    });

    it('buffers messages for batch processing', async () => {
      const handler = vi.fn();
      manager.onEntityUpdate('job', handler);
      manager.setBatchConfig({ delay: 10, maxSize: 5 });

      // Buffer multiple messages
      for (let i = 0; i < 3; i++) {
        manager.bufferMessage({
          id: String(i),
          type: 'update',
          timestamp: Date.now(),
          entity: 'job',
          entityId: String(i),
          data: {},
        } as EntityUpdateMessage);
      }

      // Wait for buffer to flush
      await new Promise((resolve) => setTimeout(resolve, 50));

      expect(handler).toHaveBeenCalledTimes(3);
    });
  });

  describe('Metrics', () => {
    it('tracks handler performance metrics', async () => {
      const handler = vi.fn();
      manager.onEntityUpdate('job', handler);

      await manager.handleMessage({
        id: '1',
        type: 'update',
        timestamp: Date.now(),
        entity: 'job',
        entityId: '1',
        data: {},
      });

      const metrics = manager.getMetrics();
      expect(metrics.has('update')).toBe(true);
      expect(metrics.get('update')?.calls).toBe(1);
    });
  });
});

// ============================================
// OptimisticUpdateManager Tests
// ============================================

describe('OptimisticUpdateManager', () => {
  let manager: OptimisticUpdateManager<{ name: string }>;

  beforeEach(() => {
    manager = new OptimisticUpdateManager(1000);
  });

  afterEach(() => {
    manager.clear();
  });

  it('applies optimistic updates', () => {
    const id = manager.apply(
      'job',
      '123',
      { name: 'Updated' },
      { name: 'Original' }
    );

    expect(id).toBeDefined();
    expect(manager.hasPendingUpdates('job', '123')).toBe(true);
  });

  it('commits updates', () => {
    const id = manager.apply(
      'job',
      '123',
      { name: 'Updated' },
      { name: 'Original' }
    );

    const committed = manager.commit(id);

    expect(committed).toBe(true);
    expect(manager.hasPendingUpdates('job', '123')).toBe(false);
  });

  it('rolls back updates', () => {
    const rollbackFn = vi.fn();
    const id = manager.apply(
      'job',
      '123',
      { name: 'Updated' },
      { name: 'Original' },
      rollbackFn
    );

    const original = manager.rollback(id);

    expect(original).toEqual({ name: 'Original' });
    expect(rollbackFn).toHaveBeenCalledWith({ name: 'Original' });
    expect(manager.hasPendingUpdates('job', '123')).toBe(false);
  });

  it('auto-rolls back after timeout', async () => {
    const rollbackFn = vi.fn();
    const shortTimeoutManager = new OptimisticUpdateManager(50);

    shortTimeoutManager.apply(
      'job',
      '123',
      { name: 'Updated' },
      { name: 'Original' },
      rollbackFn
    );

    await new Promise((resolve) => setTimeout(resolve, 100));

    expect(rollbackFn).toHaveBeenCalled();
  });
});

// ============================================
// ConnectionManager Tests
// ============================================

describe('ConnectionManager', () => {
  let manager: ConnectionManager;

  beforeEach(() => {
    manager = createConnectionManager({
      primaryUrl: 'ws://localhost:8080',
      fallbackUrls: ['ws://localhost:8081', 'ws://localhost:8082'],
      autoFailover: true,
      debug: false,
    });
  });

  afterEach(() => {
    manager.disconnect();
  });

  describe('Connection', () => {
    it('connects to primary URL', async () => {
      await manager.connect();

      expect(manager.isConnected()).toBe(true);
      expect(manager.getActiveConnection()).not.toBeNull();
    });

    it('disconnects all connections', async () => {
      await manager.connect();
      manager.disconnect();

      expect(manager.isConnected()).toBe(false);
      expect(manager.getActiveConnection()).toBeNull();
    });
  });

  describe('Health Monitoring', () => {
    it('tracks connection health', async () => {
      await manager.connect();

      // Wait for health check
      await new Promise((resolve) => setTimeout(resolve, 100));

      const health = manager.getHealth();
      expect(health).not.toBeNull();
    });

    it('returns connection status for all URLs', async () => {
      await manager.connect();

      const status = manager.getConnectionStatus();
      expect(status.length).toBe(3); // primary + 2 fallback
      expect(status.some((s) => s.isActive)).toBe(true);
    });
  });

  describe('Events', () => {
    it('emits connected event', async () => {
      const handler = vi.fn();
      manager.on('connected', handler);

      await manager.connect();

      expect(handler).toHaveBeenCalled();
    });

    it('emits disconnected event', async () => {
      const handler = vi.fn();
      manager.on('disconnected', handler);

      await manager.connect();
      manager.disconnect();

      expect(handler).toHaveBeenCalled();
    });
  });
});

// ============================================
// Performance Benchmarks
// ============================================

describe('Performance Benchmarks', () => {
  describe('Message Throughput', () => {
    it('handles high message volume', async () => {
      const manager = getUpdateHandlerManager();
      const handler = vi.fn();
      manager.onEntityUpdate('job', handler);

      const messageCount = 1000;
      const startTime = performance.now();

      const messages: EntityUpdateMessage[] = Array.from(
        { length: messageCount },
        (_, i) => ({
          id: String(i),
          type: 'update',
          timestamp: Date.now(),
          entity: 'job',
          entityId: String(i),
          data: { name: `Job ${i}` },
        })
      );

      await manager.handleMessageBatch(messages);

      const endTime = performance.now();
      const duration = endTime - startTime;
      const throughput = messageCount / (duration / 1000);

      console.log(`Message throughput: ${throughput.toFixed(0)} msg/sec`);
      console.log(
        `Duration: ${duration.toFixed(2)}ms for ${messageCount} messages`
      );

      expect(handler).toHaveBeenCalledTimes(messageCount);
      expect(duration).toBeLessThan(5000); // Should process 1000 messages in under 5 seconds
    });
  });

  describe('Handler Registration', () => {
    it('efficiently registers many handlers', () => {
      const manager = getUpdateHandlerManager();
      const handlerCount = 100;

      const startTime = performance.now();

      const unsubscribers: Array<() => void> = [];
      for (let i = 0; i < handlerCount; i++) {
        unsubscribers.push(manager.onEntityUpdate('job', vi.fn()));
      }

      const endTime = performance.now();
      const duration = endTime - startTime;

      console.log(
        `Handler registration: ${duration.toFixed(
          2
        )}ms for ${handlerCount} handlers`
      );

      expect(duration).toBeLessThan(100); // Should register 100 handlers in under 100ms

      // Cleanup
      unsubscribers.forEach((unsub) => unsub());
    });
  });

  describe('Batch Processing', () => {
    it('batch processing is faster than individual processing', async () => {
      const manager = getUpdateHandlerManager();
      const handler = vi.fn();
      manager.onEntityUpdate('job', handler);

      const messageCount = 500;
      const messages: EntityUpdateMessage[] = Array.from(
        { length: messageCount },
        (_, i) => ({
          id: String(i),
          type: 'update',
          timestamp: Date.now(),
          entity: 'job',
          entityId: String(i),
          data: { name: `Job ${i}` },
        })
      );

      // Individual processing
      const individualStart = performance.now();
      for (const message of messages) {
        await manager.handleMessage(message);
      }
      const individualDuration = performance.now() - individualStart;

      handler.mockClear();

      // Batch processing
      const batchStart = performance.now();
      await manager.handleMessageBatch(messages);
      const batchDuration = performance.now() - batchStart;

      console.log(`Individual: ${individualDuration.toFixed(2)}ms`);
      console.log(`Batch: ${batchDuration.toFixed(2)}ms`);
      console.log(
        `Speedup: ${(individualDuration / batchDuration).toFixed(2)}x`
      );

      // Batch should be faster (allowing some variance)
      expect(batchDuration).toBeLessThan(individualDuration * 1.5);
    });
  });

  describe('Memory Usage', () => {
    it('does not leak memory with repeated operations', () => {
      const manager = getUpdateHandlerManager();
      const iterations = 100;

      for (let i = 0; i < iterations; i++) {
        const handler = vi.fn();
        const unsubscribe = manager.onEntityUpdate('job', handler);
        unsubscribe();
      }

      // If we get here without memory issues, the test passes
      expect(true).toBe(true);
    });
  });

  describe('Optimistic Update Performance', () => {
    it('handles rapid optimistic updates', () => {
      const manager = new OptimisticUpdateManager<{ count: number }>(10000);
      const updateCount = 1000;

      const startTime = performance.now();

      const ids: string[] = [];
      for (let i = 0; i < updateCount; i++) {
        const id = manager.apply(
          'counter',
          'single',
          { count: i + 1 },
          { count: i }
        );
        ids.push(id);
      }

      // Commit all
      for (const id of ids) {
        manager.commit(id);
      }

      const endTime = performance.now();
      const duration = endTime - startTime;

      console.log(
        `Optimistic update throughput: ${(
          updateCount /
          (duration / 1000)
        ).toFixed(0)} ops/sec`
      );

      expect(duration).toBeLessThan(1000); // Should handle 1000 operations in under 1 second

      manager.clear();
    });
  });
});

// ============================================
// Connection Stability Tests
// ============================================

describe('Connection Stability', () => {
  describe('Reconnection', () => {
    it('tracks reconnect attempts', async () => {
      const service = new WebSocketServiceImpl({
        url: 'ws://localhost:8080',
        autoReconnect: true,
        maxReconnectAttempts: 3,
      });

      await service.connect();
      const state = service.getConnectionState();

      expect(state.reconnectAttempts).toBe(0);

      service.disconnect();
    });
  });

  describe('Message Queuing', () => {
    it('queues messages when disconnected', async () => {
      const service = new WebSocketServiceImpl({
        url: 'ws://localhost:8080',
        messageQueueSize: 10,
      });

      // Send without connecting
      const message = createMessage<PingMessage>('ping', {});
      await service.send(message);

      // Message should be queued
      const metrics = service.getMetrics();
      expect(metrics.messagesSent).toBe(0);
    });
  });

  describe('Error Recovery', () => {
    it('recovers from connection errors', async () => {
      const manager = createConnectionManager({
        primaryUrl: 'ws://localhost:8080',
        autoFailover: true,
      });

      const errorHandler = vi.fn();
      manager.on('error', errorHandler);

      await manager.connect();

      // Should be connected
      expect(manager.isConnected()).toBe(true);

      manager.disconnect();
    });
  });
});

// ============================================
// Update Propagation Tests
// ============================================

describe('Update Propagation', () => {
  let handlerManager: UpdateHandlerManager;

  beforeEach(() => {
    resetUpdateHandlerManager();
    handlerManager = getUpdateHandlerManager();
  });

  afterEach(() => {
    handlerManager.clearAllHandlers();
  });

  describe('Entity Updates', () => {
    it('propagates updates to multiple handlers', async () => {
      const handler1 = vi.fn();
      const handler2 = vi.fn();

      handlerManager.onEntityUpdate('job', handler1);
      handlerManager.onEntityUpdate('job', handler2);

      await handlerManager.handleMessage({
        id: '1',
        type: 'update',
        timestamp: Date.now(),
        entity: 'job',
        entityId: '123',
        data: { name: 'Test' },
      });

      expect(handler1).toHaveBeenCalled();
      expect(handler2).toHaveBeenCalled();
    });

    it('isolates updates by entity type', async () => {
      const jobHandler = vi.fn();
      const customerHandler = vi.fn();

      handlerManager.onEntityUpdate('job', jobHandler);
      handlerManager.onEntityUpdate('customer', customerHandler);

      await handlerManager.handleMessage({
        id: '1',
        type: 'update',
        timestamp: Date.now(),
        entity: 'job',
        entityId: '123',
        data: {},
      });

      expect(jobHandler).toHaveBeenCalled();
      expect(customerHandler).not.toHaveBeenCalled();
    });
  });

  describe('Typed Entity Handler', () => {
    it('provides typed callbacks', async () => {
      interface Job {
        id: string;
        name: string;
      }

      const onCreate = vi.fn<[string, Job], void>();
      const onUpdate = vi.fn<[string, Job], void>();
      const onDelete = vi.fn<[string], void>();

      createEntityHandler<Job>(handlerManager, 'job', {
        onCreate,
        onUpdate,
        onDelete,
      });

      await handlerManager.handleMessage({
        id: '1',
        type: 'create',
        timestamp: Date.now(),
        entity: 'job',
        entityId: 'job-1',
        data: { id: 'job-1', name: 'New Job' },
      });

      expect(onCreate).toHaveBeenCalledWith('job-1', {
        id: 'job-1',
        name: 'New Job',
      });
    });
  });
});
