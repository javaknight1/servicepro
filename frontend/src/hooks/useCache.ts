/**
 * =============================================================================
 * Cache Hooks
 * =============================================================================
 * React hooks for cache management and offline support
 */

import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import {
  useQuery,
  useMutation,
  useQueryClient,
  UseQueryOptions,
  UseMutationOptions,
} from '@tanstack/react-query';
import { cache, sessionCache } from '../utils/cache';
import {
  getCachePolicy,
  getInvalidationKeys,
  CACHE_TIMES,
} from '../config/cache-config';

// =============================================================================
// Types
// =============================================================================

interface UseCachedQueryOptions<T>
  extends Omit<UseQueryOptions<T>, 'queryKey' | 'queryFn'> {
  cacheKey?: string;
  persistToLocalStorage?: boolean;
  localStorageExpiry?: number;
}

interface UseCachedMutationOptions<TData, TVariables>
  extends UseMutationOptions<TData, Error, TVariables> {
  invalidateKeys?: string[];
  optimisticUpdate?: (variables: TVariables) => void;
  rollback?: () => void;
}

interface CacheStats {
  hits: number;
  misses: number;
  size: number;
  itemCount: number;
  hitRate: number;
}

// =============================================================================
// useCachedQuery
// =============================================================================

/**
 * Hook for cached queries with local storage persistence
 */
export function useCachedQuery<T>(
  endpoint: string,
  queryFn: () => Promise<T>,
  options: UseCachedQueryOptions<T> = {}
) {
  const {
    cacheKey = endpoint,
    persistToLocalStorage = false,
    localStorageExpiry,
    ...queryOptions
  } = options;

  const queryClient = useQueryClient();
  const policy = getCachePolicy(endpoint);

  // Try to get initial data from local storage
  const initialData = useMemo(() => {
    if (persistToLocalStorage) {
      return cache.get<T>(cacheKey) ?? undefined;
    }
    return undefined;
  }, [cacheKey, persistToLocalStorage]);

  const query = useQuery<T>({
    queryKey: [endpoint],
    queryFn,
    staleTime: policy.staleTime,
    gcTime: policy.gcTime,
    refetchOnWindowFocus: policy.refetchOnWindowFocus,
    refetchInterval: policy.refetchInterval,
    initialData,
    ...queryOptions,
  });

  // Persist to local storage when data changes
  useEffect(() => {
    if (persistToLocalStorage && query.data !== undefined) {
      cache.set(cacheKey, query.data, localStorageExpiry ?? policy.gcTime);
    }
  }, [
    query.data,
    cacheKey,
    persistToLocalStorage,
    localStorageExpiry,
    policy.gcTime,
  ]);

  // Invalidate function
  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: [endpoint] });
    if (persistToLocalStorage) {
      cache.delete(cacheKey);
    }
  }, [queryClient, endpoint, cacheKey, persistToLocalStorage]);

  return {
    ...query,
    invalidate,
    isStale: query.dataUpdatedAt
      ? Date.now() - query.dataUpdatedAt > policy.staleTime
      : true,
  };
}

// =============================================================================
// useCachedMutation
// =============================================================================

/**
 * Hook for mutations with automatic cache invalidation
 */
export function useCachedMutation<TData, TVariables>(
  mutationType: string,
  mutationFn: (variables: TVariables) => Promise<TData>,
  options: UseCachedMutationOptions<TData, TVariables> = {}
) {
  const {
    invalidateKeys = getInvalidationKeys(mutationType),
    optimisticUpdate,
    rollback,
    onMutate: _onMutate,
    onError: _onError,
    onSettled: _onSettled,
    ...mutationOptions
  } = options;

  const queryClient = useQueryClient();
  const previousDataRef = useRef<Map<string, unknown>>(new Map());

  const mutation = useMutation<TData, Error, TVariables, { previousData: Map<string, unknown> }>({
    mutationFn,
    onMutate: async (variables): Promise<{ previousData: Map<string, unknown> }> => {
      // Cancel outgoing queries for affected keys
      for (const key of invalidateKeys) {
        await queryClient.cancelQueries({ queryKey: [key] });
      }

      // Store previous data for rollback
      previousDataRef.current.clear();
      for (const key of invalidateKeys) {
        const previousData = queryClient.getQueryData([key]);
        if (previousData !== undefined) {
          previousDataRef.current.set(key, previousData);
        }
      }

      // Apply optimistic update
      if (optimisticUpdate) {
        optimisticUpdate(variables);
      }

      return { previousData: previousDataRef.current };
    },
    onError: (_error, _variables, context) => {
      // Rollback to previous data
      if (context?.previousData) {
        for (const [key, data] of context.previousData.entries()) {
          queryClient.setQueryData([key], data);
        }
      }

      // Call custom rollback
      if (rollback) {
        rollback();
      }
    },
    onSettled: () => {
      // Invalidate affected queries
      for (const key of invalidateKeys) {
        queryClient.invalidateQueries({ queryKey: [key] });
      }

      // Also invalidate local storage cache
      cache.invalidate(invalidateKeys);
    },
    ...mutationOptions,
  });

  return mutation;
}

// =============================================================================
// useLocalCache
// =============================================================================

/**
 * Hook for local storage cache with reactive updates
 */
export function useLocalCache<T>(key: string, defaultValue: T) {
  const [value, setValue] = useState<T>(() => {
    const cached = cache.get<T>(key);
    return cached ?? defaultValue;
  });

  // Subscribe to cache changes
  useEffect(() => {
    const unsubscribe = cache.subscribe<T>(key, (newValue) => {
      setValue(newValue ?? defaultValue);
    });

    return unsubscribe;
  }, [key, defaultValue]);

  // Set value function
  const set = useCallback(
    (newValue: T | ((prev: T) => T), expiry?: number) => {
      const valueToSet =
        typeof newValue === 'function'
          ? (newValue as (prev: T) => T)(value)
          : newValue;

      cache.set(key, valueToSet, expiry);
      setValue(valueToSet);
    },
    [key, value]
  );

  // Remove value function
  const remove = useCallback(() => {
    cache.delete(key);
    setValue(defaultValue);
  }, [key, defaultValue]);

  return [value, set, remove] as const;
}

// =============================================================================
// useSessionCache
// =============================================================================

/**
 * Hook for session storage (cleared on tab close)
 */
export function useSessionCache<T>(key: string, defaultValue: T) {
  const [value, setValue] = useState<T>(() => {
    const cached = sessionCache.get<T>(key);
    return cached ?? defaultValue;
  });

  const set = useCallback(
    (newValue: T | ((prev: T) => T)) => {
      const valueToSet =
        typeof newValue === 'function'
          ? (newValue as (prev: T) => T)(value)
          : newValue;

      sessionCache.set(key, valueToSet);
      setValue(valueToSet);
    },
    [key, value]
  );

  const remove = useCallback(() => {
    sessionCache.delete(key);
    setValue(defaultValue);
  }, [key, defaultValue]);

  return [value, set, remove] as const;
}

// =============================================================================
// useCacheStats
// =============================================================================

/**
 * Hook for monitoring cache statistics
 */
export function useCacheStats(refreshInterval = 5000): CacheStats {
  const [stats, setStats] = useState<CacheStats>(() => cache.getStats());

  useEffect(() => {
    const interval = setInterval(() => {
      setStats(cache.getStats());
    }, refreshInterval);

    return () => clearInterval(interval);
  }, [refreshInterval]);

  return stats;
}

// =============================================================================
// useOfflineStatus
// =============================================================================

/**
 * Hook for monitoring online/offline status
 */
export function useOfflineStatus() {
  const [isOnline, setIsOnline] = useState(navigator.onLine);
  const [wasOffline, setWasOffline] = useState(false);

  useEffect(() => {
    const handleOnline = () => {
      setIsOnline(true);
      if (wasOffline) {
        // Trigger refetch when coming back online
        setWasOffline(false);
      }
    };

    const handleOffline = () => {
      setIsOnline(false);
      setWasOffline(true);
    };

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, [wasOffline]);

  return { isOnline, wasOffline };
}

// =============================================================================
// usePrefetch
// =============================================================================

/**
 * Hook for prefetching data
 */
export function usePrefetch() {
  const queryClient = useQueryClient();

  const prefetch = useCallback(
    async <T>(
      endpoint: string,
      queryFn: () => Promise<T>,
      options?: { staleTime?: number }
    ) => {
      const policy = getCachePolicy(endpoint);

      await queryClient.prefetchQuery({
        queryKey: [endpoint],
        queryFn,
        staleTime: options?.staleTime ?? policy.staleTime,
      });
    },
    [queryClient]
  );

  const prefetchOnHover = useCallback(
    <T>(endpoint: string, queryFn: () => Promise<T>, delay = 100) => {
      let timeoutId: ReturnType<typeof setTimeout>;

      return {
        onMouseEnter: () => {
          timeoutId = setTimeout(() => {
            prefetch(endpoint, queryFn);
          }, delay);
        },
        onMouseLeave: () => {
          clearTimeout(timeoutId);
        },
      };
    },
    [prefetch]
  );

  return { prefetch, prefetchOnHover };
}

// =============================================================================
// useServiceWorkerCache
// =============================================================================

/**
 * Hook for interacting with service worker cache
 */
export function useServiceWorkerCache() {
  const [cacheStats, setCacheStats] = useState<{
    caches: Record<string, { count: number; size: number }>;
    totalSize: number;
  } | null>(null);

  const getCacheStats = useCallback(async () => {
    const controller = 'serviceWorker' in navigator ? navigator.serviceWorker.controller : null;
    if (controller) {
      const messageChannel = new MessageChannel();

      return new Promise<typeof cacheStats>((resolve) => {
        messageChannel.port1.onmessage = (event) => {
          setCacheStats(event.data);
          resolve(event.data);
        };

        controller.postMessage(
          { type: 'GET_CACHE_STATS' },
          [messageChannel.port2]
        );
      });
    }
    return null;
  }, []);

  const clearCache = useCallback(async () => {
    if ('serviceWorker' in navigator && navigator.serviceWorker.controller) {
      navigator.serviceWorker.controller.postMessage({ type: 'CLEAR_CACHE' });
    }
  }, []);

  const clearApiCache = useCallback(async () => {
    if ('serviceWorker' in navigator && navigator.serviceWorker.controller) {
      navigator.serviceWorker.controller.postMessage({
        type: 'CLEAR_API_CACHE',
      });
    }
  }, []);

  const invalidatePatterns = useCallback(async (patterns: string[]) => {
    if ('serviceWorker' in navigator && navigator.serviceWorker.controller) {
      navigator.serviceWorker.controller.postMessage({
        type: 'INVALIDATE_CACHE',
        patterns,
      });
    }
  }, []);

  return {
    cacheStats,
    getCacheStats,
    clearCache,
    clearApiCache,
    invalidatePatterns,
  };
}

// =============================================================================
// useQueryInvalidation
// =============================================================================

/**
 * Hook for query invalidation helpers
 */
export function useQueryInvalidation() {
  const queryClient = useQueryClient();

  const invalidateQueries = useCallback(
    (keys: string | string[]) => {
      const keyArray = Array.isArray(keys) ? keys : [keys];

      for (const key of keyArray) {
        queryClient.invalidateQueries({ queryKey: [key] });
      }

      // Also invalidate local storage
      cache.invalidate(keyArray);
    },
    [queryClient]
  );

  const invalidateAll = useCallback(() => {
    queryClient.invalidateQueries();
    cache.clear();
  }, [queryClient]);

  const invalidateStale = useCallback(() => {
    queryClient.invalidateQueries({
      predicate: (query) => query.isStale(),
    });
  }, [queryClient]);

  return {
    invalidateQueries,
    invalidateAll,
    invalidateStale,
  };
}

// =============================================================================
// Exports
// =============================================================================

export { CACHE_TIMES, getCachePolicy };
