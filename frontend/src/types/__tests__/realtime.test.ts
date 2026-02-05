import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  generateMessageId,
  createMessage,
  parseCloseCode,
  isRecoverableError,
  calculateBackoff,
  DEFAULT_CONNECTION_CONFIG,
  DEFAULT_QUEUE_CONFIG,
} from '../realtime';

describe('realtime utility functions', () => {
  describe('generateMessageId', () => {
    it('should generate a unique message ID', () => {
      const id1 = generateMessageId();
      const id2 = generateMessageId();

      expect(id1).toBeTruthy();
      expect(id2).toBeTruthy();
      expect(id1).not.toBe(id2);
    });

    it('should generate ID with timestamp prefix', () => {
      const before = Date.now();
      const id = generateMessageId();
      const after = Date.now();

      // Extract timestamp from ID (format: timestamp-random)
      const timestamp = parseInt(id.split('-')[0], 10);
      expect(timestamp).toBeGreaterThanOrEqual(before);
      expect(timestamp).toBeLessThanOrEqual(after);
    });

    it('should generate ID with alphanumeric suffix', () => {
      const id = generateMessageId();
      const parts = id.split('-');

      expect(parts.length).toBe(2);
      expect(parts[1]).toMatch(/^[a-z0-9]+$/);
    });
  });

  describe('createMessage', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2024-01-01T00:00:00.000Z'));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('should create a ping message', () => {
      const message = createMessage('ping', {});

      expect(message.type).toBe('ping');
      expect(message.id).toBeTruthy();
      expect(message.timestamp).toBe(Date.now());
    });

    it('should create a subscribe message with data', () => {
      const message = createMessage('subscribe', {
        channel: 'jobs',
        filters: { status: 'active' },
      });

      expect(message.type).toBe('subscribe');
      expect((message as { channel: string }).channel).toBe('jobs');
      expect((message as { filters: object }).filters).toEqual({
        status: 'active',
      });
    });

    it('should create an error message', () => {
      const message = createMessage('error', {
        code: 'AUTH_ERROR',
        message: 'Authentication failed',
      });

      expect(message.type).toBe('error');
      expect((message as { code: string }).code).toBe('AUTH_ERROR');
      expect((message as { message: string }).message).toBe(
        'Authentication failed'
      );
    });

    it('should auto-generate timestamp', () => {
      const expectedTime = Date.now();
      const message = createMessage('pong', {});

      expect(message.timestamp).toBe(expectedTime);
    });
  });

  describe('parseCloseCode', () => {
    it('should parse normal close (1000)', () => {
      expect(parseCloseCode(1000)).toBe('normal');
    });

    it('should parse going_away close (1001)', () => {
      expect(parseCloseCode(1001)).toBe('going_away');
    });

    it('should parse protocol_error close (1002)', () => {
      expect(parseCloseCode(1002)).toBe('protocol_error');
    });

    it('should parse unsupported_data close (1003)', () => {
      expect(parseCloseCode(1003)).toBe('unsupported_data');
    });

    it('should parse no_status close (1005)', () => {
      expect(parseCloseCode(1005)).toBe('no_status');
    });

    it('should parse abnormal close (1006)', () => {
      expect(parseCloseCode(1006)).toBe('abnormal');
    });

    it('should parse invalid_payload close (1007)', () => {
      expect(parseCloseCode(1007)).toBe('invalid_payload');
    });

    it('should parse policy_violation close (1008)', () => {
      expect(parseCloseCode(1008)).toBe('policy_violation');
    });

    it('should parse message_too_big close (1009)', () => {
      expect(parseCloseCode(1009)).toBe('message_too_big');
    });

    it('should parse internal_error close (1011)', () => {
      expect(parseCloseCode(1011)).toBe('internal_error');
    });

    it('should parse service_restart close (1012)', () => {
      expect(parseCloseCode(1012)).toBe('service_restart');
    });

    it('should parse try_again_later close (1013)', () => {
      expect(parseCloseCode(1013)).toBe('try_again_later');
    });

    it('should return unknown for unmapped codes', () => {
      expect(parseCloseCode(9999)).toBe('unknown');
      expect(parseCloseCode(4000)).toBe('unknown');
      expect(parseCloseCode(1234)).toBe('unknown');
    });
  });

  describe('isRecoverableError', () => {
    it('should return true for normal close (1000)', () => {
      expect(isRecoverableError(1000)).toBe(true);
    });

    it('should return true for going_away (1001)', () => {
      expect(isRecoverableError(1001)).toBe(true);
    });

    it('should return true for abnormal close (1006)', () => {
      expect(isRecoverableError(1006)).toBe(true);
    });

    it('should return true for service_restart (1012)', () => {
      expect(isRecoverableError(1012)).toBe(true);
    });

    it('should return true for try_again_later (1013)', () => {
      expect(isRecoverableError(1013)).toBe(true);
    });

    it('should return false for protocol_error (1002)', () => {
      expect(isRecoverableError(1002)).toBe(false);
    });

    it('should return false for unsupported_data (1003)', () => {
      expect(isRecoverableError(1003)).toBe(false);
    });

    it('should return false for policy_violation (1008)', () => {
      expect(isRecoverableError(1008)).toBe(false);
    });

    it('should return false for unknown codes', () => {
      expect(isRecoverableError(4000)).toBe(false);
      expect(isRecoverableError(9999)).toBe(false);
    });
  });

  describe('calculateBackoff', () => {
    beforeEach(() => {
      // Mock Math.random for predictable jitter
      vi.spyOn(Math, 'random').mockReturnValue(0.5);
    });

    afterEach(() => {
      vi.restoreAllMocks();
    });

    it('should calculate backoff for first attempt', () => {
      // With random = 0.5, jitter = delay * 0.1 * 0 = 0
      const delay = calculateBackoff(0, 1000, 30000, 2);
      expect(delay).toBe(1000); // 1000 * 2^0 = 1000
    });

    it('should calculate exponential backoff for subsequent attempts', () => {
      // Attempt 1: 1000 * 2^1 = 2000
      const delay1 = calculateBackoff(1, 1000, 30000, 2);
      expect(delay1).toBe(2000);

      // Attempt 2: 1000 * 2^2 = 4000
      const delay2 = calculateBackoff(2, 1000, 30000, 2);
      expect(delay2).toBe(4000);

      // Attempt 3: 1000 * 2^3 = 8000
      const delay3 = calculateBackoff(3, 1000, 30000, 2);
      expect(delay3).toBe(8000);
    });

    it('should cap delay at maxDelay', () => {
      // Attempt 10: 1000 * 2^10 = 1024000, but capped at 30000
      const delay = calculateBackoff(10, 1000, 30000, 2);
      expect(delay).toBeLessThanOrEqual(30000);
    });

    it('should use default multiplier of 2', () => {
      const delay = calculateBackoff(1, 1000, 30000);
      expect(delay).toBe(2000); // 1000 * 2^1 = 2000
    });

    it('should support custom multiplier', () => {
      const delay = calculateBackoff(1, 1000, 30000, 3);
      expect(delay).toBe(3000); // 1000 * 3^1 = 3000
    });

    it('should add jitter within ±10% range', () => {
      vi.restoreAllMocks();

      // Test with actual random values
      const delays: number[] = [];
      for (let i = 0; i < 100; i++) {
        delays.push(calculateBackoff(1, 1000, 30000, 2));
      }

      // All delays should be within ±10% of 2000
      delays.forEach((delay) => {
        expect(delay).toBeGreaterThanOrEqual(1800); // 2000 - 10%
        expect(delay).toBeLessThanOrEqual(2200); // 2000 + 10%
      });
    });
  });

  describe('DEFAULT_CONNECTION_CONFIG', () => {
    it('should have correct default values', () => {
      expect(DEFAULT_CONNECTION_CONFIG.url).toBe('');
      expect(DEFAULT_CONNECTION_CONFIG.token).toBe('');
      expect(DEFAULT_CONNECTION_CONFIG.protocols).toEqual([]);
      expect(DEFAULT_CONNECTION_CONFIG.autoReconnect).toBe(true);
      expect(DEFAULT_CONNECTION_CONFIG.maxReconnectAttempts).toBe(10);
      expect(DEFAULT_CONNECTION_CONFIG.reconnectDelay).toBe(1000);
      expect(DEFAULT_CONNECTION_CONFIG.maxReconnectDelay).toBe(30000);
      expect(DEFAULT_CONNECTION_CONFIG.reconnectDelayMultiplier).toBe(2);
      expect(DEFAULT_CONNECTION_CONFIG.heartbeatInterval).toBe(30000);
      expect(DEFAULT_CONNECTION_CONFIG.connectionTimeout).toBe(10000);
      expect(DEFAULT_CONNECTION_CONFIG.messageQueueSize).toBe(100);
      expect(DEFAULT_CONNECTION_CONFIG.debug).toBe(false);
    });
  });

  describe('DEFAULT_QUEUE_CONFIG', () => {
    it('should have correct default values', () => {
      expect(DEFAULT_QUEUE_CONFIG.maxSize).toBe(100);
      expect(DEFAULT_QUEUE_CONFIG.maxRetries).toBe(3);
      expect(DEFAULT_QUEUE_CONFIG.retryDelay).toBe(1000);
      expect(DEFAULT_QUEUE_CONFIG.priorityLevels).toBe(3);
    });
  });
});
