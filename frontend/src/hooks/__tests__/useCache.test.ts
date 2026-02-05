import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import {
  useOfflineStatus,
  useCacheStats,
  useLocalCache,
  useSessionCache,
  usePrefetch,
  useServiceWorkerCache,
  useQueryInvalidation,
} from '../useCache';

// Mock the cache module
vi.mock('../../utils/cache', () => ({
  cache: {
    get: vi.fn(),
    set: vi.fn(() => true),
    delete: vi.fn(() => true),
    subscribe: vi.fn(() => vi.fn()),
    getStats: vi.fn(() => ({
      hits: 10,
      misses: 5,
      size: 1024,
      itemCount: 3,
      hitRate: 66.67,
    })),
    invalidate: vi.fn(),
    clear: vi.fn(),
  },
  sessionCache: {
    get: vi.fn(),
    set: vi.fn(() => true),
    delete: vi.fn(() => true),
  },
}));

// Mock the cache-config module
vi.mock('../../config/cache-config', () => ({
  getCachePolicy: vi.fn(() => ({
    staleTime: 60000,
    gcTime: 300000,
    refetchOnWindowFocus: true,
    refetchInterval: false,
  })),
  getInvalidationKeys: vi.fn(() => ['test-key']),
  CACHE_TIMES: {
    SHORT: 60000,
    MEDIUM: 300000,
    LONG: 600000,
  },
}));

import { cache, sessionCache } from '../../utils/cache';

// Create wrapper with QueryClient
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useCache hooks', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('useOfflineStatus', () => {
    let originalNavigator: boolean;

    beforeEach(() => {
      originalNavigator = navigator.onLine;
      // Mock navigator.onLine
      Object.defineProperty(navigator, 'onLine', {
        writable: true,
        value: true,
      });
    });

    afterEach(() => {
      Object.defineProperty(navigator, 'onLine', {
        writable: true,
        value: originalNavigator,
      });
    });

    it('should return initial online status', () => {
      const { result } = renderHook(() => useOfflineStatus());

      expect(result.current.isOnline).toBe(true);
      expect(result.current.wasOffline).toBe(false);
    });

    it('should detect offline status', () => {
      const { result } = renderHook(() => useOfflineStatus());

      act(() => {
        Object.defineProperty(navigator, 'onLine', {
          writable: true,
          value: false,
        });
        window.dispatchEvent(new Event('offline'));
      });

      expect(result.current.isOnline).toBe(false);
      expect(result.current.wasOffline).toBe(true);
    });

    it('should detect coming back online', () => {
      const { result } = renderHook(() => useOfflineStatus());

      // Go offline first
      act(() => {
        Object.defineProperty(navigator, 'onLine', {
          writable: true,
          value: false,
        });
        window.dispatchEvent(new Event('offline'));
      });

      expect(result.current.isOnline).toBe(false);
      expect(result.current.wasOffline).toBe(true);

      // Come back online
      act(() => {
        Object.defineProperty(navigator, 'onLine', {
          writable: true,
          value: true,
        });
        window.dispatchEvent(new Event('online'));
      });

      expect(result.current.isOnline).toBe(true);
      expect(result.current.wasOffline).toBe(false);
    });

    it('should clean up event listeners on unmount', () => {
      const addEventListenerSpy = vi.spyOn(window, 'addEventListener');
      const removeEventListenerSpy = vi.spyOn(window, 'removeEventListener');

      const { unmount } = renderHook(() => useOfflineStatus());

      expect(addEventListenerSpy).toHaveBeenCalledWith(
        'online',
        expect.any(Function)
      );
      expect(addEventListenerSpy).toHaveBeenCalledWith(
        'offline',
        expect.any(Function)
      );

      unmount();

      expect(removeEventListenerSpy).toHaveBeenCalledWith(
        'online',
        expect.any(Function)
      );
      expect(removeEventListenerSpy).toHaveBeenCalledWith(
        'offline',
        expect.any(Function)
      );
    });
  });

  describe('useCacheStats', () => {
    it('should return initial cache stats', () => {
      const { result } = renderHook(() => useCacheStats());

      expect(result.current).toEqual({
        hits: 10,
        misses: 5,
        size: 1024,
        itemCount: 3,
        hitRate: 66.67,
      });
    });

    it('should refresh stats at specified interval', () => {
      vi.mocked(cache.getStats)
        .mockReturnValueOnce({
          hits: 10,
          misses: 5,
          size: 1024,
          itemCount: 3,
          hitRate: 66.67,
        })
        .mockReturnValue({
          hits: 15,
          misses: 7,
          size: 2048,
          itemCount: 5,
          hitRate: 68.18,
        });

      const { result } = renderHook(() => useCacheStats(1000));

      expect(result.current.hits).toBe(10);

      // Advance timer to trigger interval
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      // After advancing time, stats should update
      expect(cache.getStats).toHaveBeenCalledTimes(2);
    });

    it('should use default refresh interval', () => {
      renderHook(() => useCacheStats());

      expect(cache.getStats).toHaveBeenCalledTimes(1);

      // Default interval is 5000ms
      act(() => {
        vi.advanceTimersByTime(5000);
      });

      expect(cache.getStats).toHaveBeenCalledTimes(2);
    });

    it('should clean up interval on unmount', () => {
      const clearIntervalSpy = vi.spyOn(global, 'clearInterval');

      const { unmount } = renderHook(() => useCacheStats());

      unmount();

      expect(clearIntervalSpy).toHaveBeenCalled();
    });
  });

  describe('useLocalCache', () => {
    it('should return default value when cache is empty', () => {
      vi.mocked(cache.get).mockReturnValue(null);

      const { result } = renderHook(
        () => useLocalCache('test-key', 'default-value'),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      expect(value).toBe('default-value');
    });

    it('should return cached value when available', () => {
      vi.mocked(cache.get).mockReturnValue('cached-value');

      const { result } = renderHook(
        () => useLocalCache('test-key', 'default-value'),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      expect(value).toBe('cached-value');
    });

    it('should set value in cache', () => {
      vi.mocked(cache.get).mockReturnValue(null);

      const { result } = renderHook(
        () => useLocalCache('test-key', 'default-value'),
        { wrapper: createWrapper() }
      );

      const [, setValue] = result.current;

      act(() => {
        setValue('new-value');
      });

      expect(cache.set).toHaveBeenCalledWith(
        'test-key',
        'new-value',
        undefined
      );
    });

    it('should set value with expiry option', () => {
      vi.mocked(cache.get).mockReturnValue(null);

      const { result } = renderHook(
        () => useLocalCache('test-key', 'default', { expiry: 60000 }),
        { wrapper: createWrapper() }
      );

      const [, setValue] = result.current;

      act(() => {
        setValue('new-value');
      });

      expect(cache.set).toHaveBeenCalledWith('test-key', 'new-value', 60000);
    });

    it('should remove value from cache', () => {
      vi.mocked(cache.get).mockReturnValue('cached-value');

      const { result } = renderHook(
        () => useLocalCache('test-key', 'default-value'),
        { wrapper: createWrapper() }
      );

      const [, , removeValue] = result.current;

      act(() => {
        removeValue();
      });

      expect(cache.delete).toHaveBeenCalledWith('test-key');
    });

    it('should handle function updater for setValue', () => {
      vi.mocked(cache.get).mockReturnValue(10);

      const { result } = renderHook(() => useLocalCache('counter', 0), {
        wrapper: createWrapper(),
      });

      const [, setValue] = result.current;

      act(() => {
        setValue((prev) => prev + 1);
      });

      expect(cache.set).toHaveBeenCalledWith('counter', 11, undefined);
    });

    it('should subscribe to cache changes', () => {
      vi.mocked(cache.get).mockReturnValue(null);

      renderHook(() => useLocalCache('test-key', 'default'), {
        wrapper: createWrapper(),
      });

      expect(cache.subscribe).toHaveBeenCalledWith(
        'test-key',
        expect.any(Function)
      );
    });
  });

  describe('useSessionCache', () => {
    it('should return default value when session cache is empty', () => {
      vi.mocked(sessionCache.get).mockReturnValue(null);

      const { result } = renderHook(
        () => useSessionCache('session-key', 'default'),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      expect(value).toBe('default');
    });

    it('should return cached value from session storage', () => {
      vi.mocked(sessionCache.get).mockReturnValue('session-cached');

      const { result } = renderHook(
        () => useSessionCache('session-key', 'default'),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      expect(value).toBe('session-cached');
    });

    it('should set value in session cache', () => {
      vi.mocked(sessionCache.get).mockReturnValue(null);

      const { result } = renderHook(
        () => useSessionCache('session-key', 'default'),
        { wrapper: createWrapper() }
      );

      const [, setValue] = result.current;

      act(() => {
        setValue('new-session-value');
      });

      expect(sessionCache.set).toHaveBeenCalledWith(
        'session-key',
        'new-session-value'
      );
    });

    it('should remove value from session cache', () => {
      vi.mocked(sessionCache.get).mockReturnValue('value');

      const { result } = renderHook(
        () => useSessionCache('session-key', 'default'),
        { wrapper: createWrapper() }
      );

      const [, , removeValue] = result.current;

      act(() => {
        removeValue();
      });

      expect(sessionCache.delete).toHaveBeenCalledWith('session-key');
    });

    it('should handle complex objects', () => {
      const complexObject = { user: { name: 'John', age: 30 } };
      vi.mocked(sessionCache.get).mockReturnValue(complexObject);

      const { result } = renderHook(
        () =>
          useSessionCache<typeof complexObject>('user-data', {
            user: { name: '', age: 0 },
          }),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      expect(value).toEqual(complexObject);
    });
  });

  describe('useOfflineStatus edge cases', () => {
    it('should handle multiple offline/online transitions', () => {
      const { result } = renderHook(() => useOfflineStatus());

      // Go offline
      act(() => {
        Object.defineProperty(navigator, 'onLine', {
          writable: true,
          value: false,
        });
        window.dispatchEvent(new Event('offline'));
      });
      expect(result.current.isOnline).toBe(false);

      // Go online
      act(() => {
        Object.defineProperty(navigator, 'onLine', {
          writable: true,
          value: true,
        });
        window.dispatchEvent(new Event('online'));
      });
      expect(result.current.isOnline).toBe(true);

      // Go offline again
      act(() => {
        Object.defineProperty(navigator, 'onLine', {
          writable: true,
          value: false,
        });
        window.dispatchEvent(new Event('offline'));
      });
      expect(result.current.isOnline).toBe(false);
    });
  });

  describe('useCacheStats with custom interval', () => {
    it('should support very short refresh intervals', () => {
      const { result } = renderHook(() => useCacheStats(100));

      expect(result.current).toBeDefined();
      expect(cache.getStats).toHaveBeenCalled();
    });
  });

  describe('useLocalCache edge cases', () => {
    it('should handle null default value', () => {
      vi.mocked(cache.get).mockReturnValue(null);

      const { result } = renderHook(
        () => useLocalCache<string | null>('nullable-key', null),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      expect(value).toBeNull();
    });

    it('should handle undefined values', () => {
      vi.mocked(cache.get).mockReturnValue(undefined);

      const { result } = renderHook(
        () => useLocalCache('undefined-key', 'default'),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      // When cache returns undefined, should fall back to default
      expect(value).toBe('default');
    });

    it('should handle array values', () => {
      const arrayValue = [1, 2, 3, 4, 5];
      vi.mocked(cache.get).mockReturnValue(arrayValue);

      const { result } = renderHook(
        () => useLocalCache<number[]>('array-key', []),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      expect(value).toEqual(arrayValue);
    });

    it('should handle nested object updates', () => {
      vi.mocked(cache.get).mockReturnValue({ nested: { value: 'old' } });

      const { result } = renderHook(
        () => useLocalCache('nested-key', { nested: { value: '' } }),
        { wrapper: createWrapper() }
      );

      const [, setValue] = result.current;

      act(() => {
        setValue({ nested: { value: 'new' } });
      });

      expect(cache.set).toHaveBeenCalledWith(
        'nested-key',
        { nested: { value: 'new' } },
        undefined
      );
    });
  });

  describe('usePrefetch', () => {
    it('should return prefetch and prefetchOnHover functions', () => {
      const { result } = renderHook(() => usePrefetch(), {
        wrapper: createWrapper(),
      });

      expect(result.current.prefetch).toBeDefined();
      expect(result.current.prefetchOnHover).toBeDefined();
      expect(typeof result.current.prefetch).toBe('function');
      expect(typeof result.current.prefetchOnHover).toBe('function');
    });

    it('should call prefetch with endpoint and queryFn', async () => {
      const { result } = renderHook(() => usePrefetch(), {
        wrapper: createWrapper(),
      });

      const mockQueryFn = vi.fn().mockResolvedValue({ data: 'test' });

      await act(async () => {
        await result.current.prefetch('/api/test', mockQueryFn);
      });

      // The prefetch should have been called (we can't easily verify the internal queryClient call)
      expect(result.current.prefetch).toBeDefined();
    });

    it('should return hover handlers from prefetchOnHover', () => {
      const { result } = renderHook(() => usePrefetch(), {
        wrapper: createWrapper(),
      });

      const mockQueryFn = vi.fn().mockResolvedValue({ data: 'test' });
      const handlers = result.current.prefetchOnHover(
        '/api/test',
        mockQueryFn,
        50
      );

      expect(handlers.onMouseEnter).toBeDefined();
      expect(handlers.onMouseLeave).toBeDefined();
      expect(typeof handlers.onMouseEnter).toBe('function');
      expect(typeof handlers.onMouseLeave).toBe('function');
    });

    it('should handle prefetchOnHover mouse enter and leave', () => {
      const { result } = renderHook(() => usePrefetch(), {
        wrapper: createWrapper(),
      });

      const mockQueryFn = vi.fn().mockResolvedValue({ data: 'test' });
      const handlers = result.current.prefetchOnHover(
        '/api/test',
        mockQueryFn,
        100
      );

      // Trigger mouse enter
      act(() => {
        handlers.onMouseEnter();
      });

      // Trigger mouse leave before timeout
      act(() => {
        handlers.onMouseLeave();
      });

      // The timeout should have been cleared
      expect(mockQueryFn).not.toHaveBeenCalled();
    });
  });

  describe('useServiceWorkerCache', () => {
    let originalNavigator: ServiceWorkerContainer | undefined;

    beforeEach(() => {
      originalNavigator = navigator.serviceWorker;
    });

    afterEach(() => {
      if (originalNavigator) {
        Object.defineProperty(navigator, 'serviceWorker', {
          value: originalNavigator,
          configurable: true,
        });
      }
    });

    it('should return cache management functions', () => {
      const { result } = renderHook(() => useServiceWorkerCache());

      expect(result.current.cacheStats).toBeNull();
      expect(result.current.getCacheStats).toBeDefined();
      expect(result.current.clearCache).toBeDefined();
      expect(result.current.clearApiCache).toBeDefined();
      expect(result.current.invalidatePatterns).toBeDefined();
    });

    it('should return null from getCacheStats when no service worker', async () => {
      Object.defineProperty(navigator, 'serviceWorker', {
        value: { controller: null },
        configurable: true,
      });

      const { result } = renderHook(() => useServiceWorkerCache());

      let stats: unknown;
      await act(async () => {
        stats = await result.current.getCacheStats();
      });

      expect(stats).toBeNull();
    });

    it('should call postMessage on clearCache', async () => {
      const mockPostMessage = vi.fn();
      Object.defineProperty(navigator, 'serviceWorker', {
        value: {
          controller: { postMessage: mockPostMessage },
        },
        configurable: true,
      });

      const { result } = renderHook(() => useServiceWorkerCache());

      await act(async () => {
        await result.current.clearCache();
      });

      expect(mockPostMessage).toHaveBeenCalledWith({ type: 'CLEAR_CACHE' });
    });

    it('should call postMessage on clearApiCache', async () => {
      const mockPostMessage = vi.fn();
      Object.defineProperty(navigator, 'serviceWorker', {
        value: {
          controller: { postMessage: mockPostMessage },
        },
        configurable: true,
      });

      const { result } = renderHook(() => useServiceWorkerCache());

      await act(async () => {
        await result.current.clearApiCache();
      });

      expect(mockPostMessage).toHaveBeenCalledWith({ type: 'CLEAR_API_CACHE' });
    });

    it('should call postMessage on invalidatePatterns', async () => {
      const mockPostMessage = vi.fn();
      Object.defineProperty(navigator, 'serviceWorker', {
        value: {
          controller: { postMessage: mockPostMessage },
        },
        configurable: true,
      });

      const { result } = renderHook(() => useServiceWorkerCache());

      await act(async () => {
        await result.current.invalidatePatterns(['/api/users', '/api/jobs']);
      });

      expect(mockPostMessage).toHaveBeenCalledWith({
        type: 'INVALIDATE_CACHE',
        patterns: ['/api/users', '/api/jobs'],
      });
    });
  });

  describe('useQueryInvalidation', () => {
    it('should return invalidation functions', () => {
      const { result } = renderHook(() => useQueryInvalidation(), {
        wrapper: createWrapper(),
      });

      expect(result.current.invalidateQueries).toBeDefined();
      expect(result.current.invalidateAll).toBeDefined();
      expect(result.current.invalidateStale).toBeDefined();
    });

    it('should invalidate single query key', () => {
      const { result } = renderHook(() => useQueryInvalidation(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.invalidateQueries('users');
      });

      // Verify cache.invalidate was called
      expect(cache.invalidate).toHaveBeenCalledWith(['users']);
    });

    it('should invalidate multiple query keys', () => {
      const { result } = renderHook(() => useQueryInvalidation(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.invalidateQueries(['users', 'jobs', 'tenants']);
      });

      expect(cache.invalidate).toHaveBeenCalledWith([
        'users',
        'jobs',
        'tenants',
      ]);
    });

    it('should invalidate all queries and clear cache', () => {
      const { result } = renderHook(() => useQueryInvalidation(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.invalidateAll();
      });

      expect(cache.clear).toHaveBeenCalled();
    });

    it('should invalidate stale queries', () => {
      const { result } = renderHook(() => useQueryInvalidation(), {
        wrapper: createWrapper(),
      });

      // This tests that invalidateStale can be called without error
      act(() => {
        result.current.invalidateStale();
      });

      expect(result.current.invalidateStale).toBeDefined();
    });
  });
});
