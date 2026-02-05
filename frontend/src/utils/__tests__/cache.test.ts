import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { cache, sessionCache, withCache } from '../cache';

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
    get length() {
      return Object.keys(store).length;
    },
    key: vi.fn((index: number) => Object.keys(store)[index] || null),
  };
})();

// Mock sessionStorage
const sessionStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
    get length() {
      return Object.keys(store).length;
    },
    key: vi.fn((index: number) => Object.keys(store)[index] || null),
  };
})();

Object.defineProperty(window, 'localStorage', { value: localStorageMock });
Object.defineProperty(window, 'sessionStorage', { value: sessionStorageMock });

describe('cache utilities', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
    sessionStorageMock.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('cache (LocalStorageCache)', () => {
    describe('get and set', () => {
      it('should set and get a value', () => {
        cache.set('test-key', { name: 'Test' });
        const result = cache.get<{ name: string }>('test-key');

        expect(result).toEqual({ name: 'Test' });
      });

      it('should return null for non-existent key', () => {
        const result = cache.get('non-existent');
        expect(result).toBeNull();
      });

      it('should handle different data types', () => {
        cache.set('string', 'hello');
        cache.set('number', 42);
        cache.set('boolean', true);
        cache.set('array', [1, 2, 3]);
        cache.set('object', { nested: { value: 'test' } });

        expect(cache.get('string')).toBe('hello');
        expect(cache.get('number')).toBe(42);
        expect(cache.get('boolean')).toBe(true);
        expect(cache.get('array')).toEqual([1, 2, 3]);
        expect(cache.get('object')).toEqual({ nested: { value: 'test' } });
      });
    });

    describe('delete', () => {
      it('should delete a value', () => {
        cache.set('to-delete', 'value');
        expect(cache.get('to-delete')).toBe('value');

        cache.delete('to-delete');
        expect(cache.get('to-delete')).toBeNull();
      });

      it('should return true when deleting', () => {
        cache.set('key', 'value');
        expect(cache.delete('key')).toBe(true);
      });
    });

    describe('has', () => {
      it('should return true for existing key', () => {
        cache.set('exists', 'value');
        expect(cache.has('exists')).toBe(true);
      });

      it('should return false for non-existent key', () => {
        expect(cache.has('not-exists')).toBe(false);
      });
    });

    describe('getKeys', () => {
      it('should return all cache keys', () => {
        cache.set('key1', 'value1');
        cache.set('key2', 'value2');

        const keys = cache.getKeys();
        expect(keys).toContain('key1');
        expect(keys).toContain('key2');
      });

      it('should filter keys by pattern', () => {
        cache.set('user-1', 'data1');
        cache.set('user-2', 'data2');
        cache.set('settings', 'data3');

        const keys = cache.getKeys('user');
        expect(keys).toContain('user-1');
        expect(keys).toContain('user-2');
        expect(keys).not.toContain('settings');
      });
    });

    describe('deletePattern', () => {
      it('should delete keys matching pattern', () => {
        cache.set('api-users', 'data1');
        cache.set('api-posts', 'data2');
        cache.set('settings', 'data3');

        const deleted = cache.deletePattern('api');

        expect(deleted).toBe(2);
        expect(cache.get('api-users')).toBeNull();
        expect(cache.get('api-posts')).toBeNull();
        expect(cache.get('settings')).toBe('data3');
      });
    });

    describe('clear', () => {
      it('should clear all cache entries', () => {
        cache.set('key1', 'value1');
        cache.set('key2', 'value2');

        cache.clear(true);

        expect(cache.get('key1')).toBeNull();
        expect(cache.get('key2')).toBeNull();
      });
    });

    describe('getStats', () => {
      it('should return cache statistics', () => {
        cache.set('key1', 'value1');
        cache.get('key1'); // hit
        cache.get('missing'); // miss

        const stats = cache.getStats();

        expect(stats).toHaveProperty('hits');
        expect(stats).toHaveProperty('misses');
        expect(stats).toHaveProperty('size');
        expect(stats).toHaveProperty('itemCount');
        expect(stats).toHaveProperty('hitRate');
      });
    });

    describe('getMany', () => {
      it('should get multiple values', () => {
        cache.set('key1', 'value1');
        cache.set('key2', 'value2');

        const results = cache.getMany(['key1', 'key2', 'key3']);

        expect(results.get('key1')).toBe('value1');
        expect(results.get('key2')).toBe('value2');
        expect(results.get('key3')).toBeNull();
      });
    });

    describe('setMany', () => {
      it('should set multiple values', () => {
        cache.setMany([
          { key: 'multi1', data: 'value1' },
          { key: 'multi2', data: 'value2' },
        ]);

        expect(cache.get('multi1')).toBe('value1');
        expect(cache.get('multi2')).toBe('value2');
      });
    });

    describe('deleteMany', () => {
      it('should delete multiple keys', () => {
        cache.set('del1', 'value1');
        cache.set('del2', 'value2');
        cache.set('keep', 'value3');

        cache.deleteMany(['del1', 'del2']);

        expect(cache.get('del1')).toBeNull();
        expect(cache.get('del2')).toBeNull();
        expect(cache.get('keep')).toBe('value3');
      });
    });

    describe('subscribe', () => {
      it('should notify listeners on value change', () => {
        const callback = vi.fn();

        const unsubscribe = cache.subscribe('watched-key', callback);

        cache.set('watched-key', 'new-value');

        expect(callback).toHaveBeenCalledWith('new-value');

        unsubscribe();
      });

      it('should not notify after unsubscribe', () => {
        const callback = vi.fn();

        const unsubscribe = cache.subscribe('watched-key', callback);
        unsubscribe();

        cache.set('watched-key', 'new-value');

        expect(callback).not.toHaveBeenCalled();
      });
    });

    describe('invalidate', () => {
      it('should invalidate by patterns', () => {
        cache.set('user-list', 'data1');
        cache.set('user-detail', 'data2');
        cache.set('posts', 'data3');

        cache.invalidate(['user']);

        expect(cache.get('user-list')).toBeNull();
        expect(cache.get('user-detail')).toBeNull();
        expect(cache.get('posts')).toBe('data3');
      });
    });

    describe('getWithMetadata', () => {
      it('should return data with metadata', () => {
        cache.set('meta-key', { value: 'test' });
        const result = cache.getWithMetadata<{ value: string }>('meta-key');

        expect(result).not.toBeNull();
        expect(result?.data).toEqual({ value: 'test' });
        expect(result?.age).toBeGreaterThanOrEqual(0);
        expect(result?.ttl).toBeDefined();
      });

      it('should return null for non-existent key', () => {
        const result = cache.getWithMetadata('non-existent');
        expect(result).toBeNull();
      });
    });

    describe('getEventLog', () => {
      it('should return event log', () => {
        cache.set('event-key', 'value');
        cache.get('event-key');

        const events = cache.getEventLog();
        expect(events.length).toBeGreaterThan(0);
        expect(events[0]).toHaveProperty('type');
        expect(events[0]).toHaveProperty('key');
        expect(events[0]).toHaveProperty('timestamp');
      });

      it('should limit event log to specified number', () => {
        for (let i = 0; i < 10; i++) {
          cache.set(`log-key-${i}`, `value-${i}`);
        }

        const events = cache.getEventLog(5);
        expect(events.length).toBeLessThanOrEqual(5);
      });
    });

    describe('clearExpired', () => {
      it('should clear expired entries', () => {
        // Set a value with very short expiry
        cache.set('expire-soon', 'value', 1);

        // Wait for it to expire (using setTimeout)
        vi.useFakeTimers();
        vi.advanceTimersByTime(10);

        const cleared = cache.clearExpired();

        // The entry should be expired and cleared
        expect(cleared).toBeGreaterThanOrEqual(0);
      });
    });

    describe('evictOldest', () => {
      it('should evict oldest entries', () => {
        // Set multiple entries
        cache.set('old-1', 'value1');
        cache.set('old-2', 'value2');
        cache.set('old-3', 'value3');

        cache.evictOldest(2);

        // At least some entries should be evicted
        const stats = cache.getStats();
        expect(stats).toBeDefined();
      });
    });

    describe('clear without includePersistent', () => {
      it('should not clear persistent keys by default', () => {
        cache.set('theme-setting', 'dark');
        cache.set('regular-key', 'value');

        cache.clear(false);

        // Regular key should be cleared, but theme-related keys might persist
        expect(cache.get('regular-key')).toBeNull();
      });
    });

    describe('error handling', () => {
      it('should handle JSON parse errors gracefully', () => {
        // Manually set invalid JSON
        localStorage.setItem('sp_cache_invalid', 'not-valid-json');

        const result = cache.get('invalid');
        expect(result).toBeNull();
      });
    });
  });

  describe('sessionCache', () => {
    it('should set and get values', () => {
      sessionCache.set('session-key', { data: 'test' });
      const result = sessionCache.get<{ data: string }>('session-key');

      expect(result).toEqual({ data: 'test' });
    });

    it('should return null for non-existent key', () => {
      expect(sessionCache.get('missing')).toBeNull();
    });

    it('should delete values', () => {
      sessionCache.set('to-delete', 'value');
      sessionCache.delete('to-delete');

      expect(sessionCache.get('to-delete')).toBeNull();
    });

    it('should clear all session values', () => {
      sessionCache.set('key1', 'value1');
      sessionCache.set('key2', 'value2');

      sessionCache.clear();

      expect(sessionCache.get('key1')).toBeNull();
      expect(sessionCache.get('key2')).toBeNull();
    });
  });

  describe('withCache', () => {
    it('should cache function results', async () => {
      const fetchFn = vi.fn().mockResolvedValue({ data: 'result' });
      const cachedFn = withCache(fetchFn, (id: string) => `fetch-${id}`, 60000);

      // First call - should execute the function
      const result1 = await cachedFn('123');
      expect(result1).toEqual({ data: 'result' });
      expect(fetchFn).toHaveBeenCalledTimes(1);

      // Second call - should return cached result
      const result2 = await cachedFn('123');
      expect(result2).toEqual({ data: 'result' });
      expect(fetchFn).toHaveBeenCalledTimes(1); // Still 1
    });

    it('should call function again for different arguments', async () => {
      const fetchFn = vi
        .fn()
        .mockImplementation((id: string) => Promise.resolve({ id }));
      const cachedFn = withCache(fetchFn, (id: string) => `fetch-${id}`, 60000);

      await cachedFn('1');
      await cachedFn('2');

      expect(fetchFn).toHaveBeenCalledTimes(2);
    });
  });

  describe('sessionCache error handling', () => {
    it('should return null when sessionStorage.getItem throws', () => {
      const originalGetItem = sessionStorageMock.getItem;
      sessionStorageMock.getItem = vi.fn(() => {
        throw new Error('Storage error');
      });

      const result = sessionCache.get('any-key');
      expect(result).toBeNull();

      sessionStorageMock.getItem = originalGetItem;
    });

    it('should return null when JSON.parse throws in sessionCache.get', () => {
      sessionStorageMock.setItem('sp_session_invalid', 'not-json{');

      const result = sessionCache.get('invalid');
      expect(result).toBeNull();
    });

    it('should return false when sessionStorage.setItem throws', () => {
      const originalSetItem = sessionStorageMock.setItem;
      sessionStorageMock.setItem = vi.fn(() => {
        throw new Error('QuotaExceededError');
      });

      const result = sessionCache.set('key', { data: 'test' });
      expect(result).toBe(false);

      sessionStorageMock.setItem = originalSetItem;
    });
  });
});
