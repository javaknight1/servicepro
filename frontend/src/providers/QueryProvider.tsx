/**
 * =============================================================================
 * React Query Provider
 * =============================================================================
 * Configures React Query with caching, persistence, and offline support
 */

import React, { useEffect, useState } from 'react';
import {
  QueryClient,
  QueryClientProvider,
  QueryCache,
  MutationCache,
  onlineManager,
  focusManager,
} from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { persistQueryClient } from '@tanstack/react-query-persist-client';
import { createSyncStoragePersister } from '@tanstack/query-sync-storage-persister';
import { QUERY_CONFIG, CACHE_TIMES } from '../config/cache-config';
import { cache } from '../utils/cache';

// =============================================================================
// Query Client Configuration
// =============================================================================

// Error handler for query cache
const queryCache = new QueryCache({
  onError: (error, query) => {
    // Only show error notification for user-initiated queries
    if (query.state.data !== undefined) {
      // eslint-disable-next-line no-console
      console.error('Query error:', error);
      // You could dispatch to a toast/notification system here
    }
  },
  onSuccess: (data, query) => {
    // Log successful queries in development
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.debug('Query success:', query.queryKey);
    }
  },
});

// Mutation cache with error handling
const mutationCache = new MutationCache({
  onError: (error, _variables, _context, _mutation) => {
    // eslint-disable-next-line no-console
    console.error('Mutation error:', error);

    // Handle specific error types
    if (error instanceof Error) {
      if (error.message.includes('401')) {
        // Handle unauthorized - could redirect to login
        console.warn('Unauthorized - session may have expired');
      } else if (error.message.includes('NetworkError')) {
        // Handle offline mutations
        console.warn('Network error - mutation will be retried when online');
      }
    }
  },
  onSuccess: (_data, _variables, _context, mutation) => {
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.debug('Mutation success:', mutation.options.mutationKey);
    }
  },
});

// Create query client with configuration
function createQueryClient(): QueryClient {
  return new QueryClient({
    queryCache,
    mutationCache,
    defaultOptions: QUERY_CONFIG.defaultOptions,
  });
}

// =============================================================================
// Storage Persister
// =============================================================================

// Create persister for offline support
const persister = createSyncStoragePersister({
  storage: {
    getItem: (key) => {
      const value = cache.get<string>(key);
      return value ?? null;
    },
    setItem: (key, value) => {
      cache.set(key, value, CACHE_TIMES.EXTENDED);
    },
    removeItem: (key) => {
      cache.delete(key);
    },
  },
  serialize: (data) => JSON.stringify(data),
  deserialize: (data) => JSON.parse(data),
  key: 'react-query-cache',
  throttleTime: 1000,
});

// =============================================================================
// Provider Component
// =============================================================================

interface QueryProviderProps {
  children: React.ReactNode;
  enableDevtools?: boolean;
  enablePersistence?: boolean;
}

export function QueryProvider({
  children,
  enableDevtools = import.meta.env.DEV,
  enablePersistence = true,
}: QueryProviderProps) {
  const [queryClient] = useState(() => createQueryClient());
  const [isRestored, setIsRestored] = useState(!enablePersistence);

  // Setup persistence
  useEffect(() => {
    if (enablePersistence) {
      const [unsubscribe, restorePromise] = persistQueryClient({
        queryClient,
        persister,
        maxAge: CACHE_TIMES.EXTENDED,
        buster: import.meta.env.VITE_APP_VERSION || '1.0.0',
        dehydrateOptions: {
          shouldDehydrateQuery: (query) => {
            // Only persist successful queries
            return query.state.status === 'success';
          },
        },
      });

      restorePromise.then(() => {
        setIsRestored(true);
      });

      return () => {
        unsubscribe();
      };
    }
  }, [queryClient, enablePersistence]);

  // Setup online/offline handling
  useEffect(() => {
    // Listen for online status changes
    const handleOnline = () => {
      onlineManager.setOnline(true);
      // Refetch stale queries when coming back online
      queryClient.refetchQueries({
        predicate: (query) => query.isStale(),
      });
    };

    const handleOffline = () => {
      onlineManager.setOnline(false);
    };

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    // Set initial online status
    onlineManager.setOnline(navigator.onLine);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, [queryClient]);

  // Setup focus handling
  useEffect(() => {
    // Custom focus handler for tab visibility
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        focusManager.setFocused(true);
      } else {
        focusManager.setFocused(false);
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, []);

  // Show loading state while restoring from persistence
  if (!isRestored) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      {enableDevtools && (
        <ReactQueryDevtools
          initialIsOpen={false}
          position="bottom"
          buttonPosition="bottom-right"
        />
      )}
    </QueryClientProvider>
  );
}

// =============================================================================
// Utility Components
// =============================================================================

/**
 * Component to display offline status
 */
export function OfflineIndicator() {
  const [isOnline, setIsOnline] = useState(navigator.onLine);

  useEffect(() => {
    const handleOnline = () => setIsOnline(true);
    const handleOffline = () => setIsOnline(false);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  if (isOnline) return null;

  return (
    <div className="fixed bottom-4 left-4 bg-yellow-500 text-white px-4 py-2 rounded-lg shadow-lg z-50 flex items-center gap-2">
      <svg
        className="w-5 h-5"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M18.364 5.636a9 9 0 010 12.728m0 0l-2.829-2.829m2.829 2.829L21 21M15.536 8.464a5 5 0 010 7.072m0 0l-2.829-2.829m-4.243 2.829a4.978 4.978 0 01-1.414-2.83m-1.414 5.658a9 9 0 01-2.167-9.238m7.824 2.167a1 1 0 111.414 1.414m-1.414-1.414L3 3m8.293 8.293l1.414 1.414"
        />
      </svg>
      <span>You're offline</span>
    </div>
  );
}

/**
 * Component to show pending mutations count
 */
export function PendingMutationsIndicator() {
  const [pendingCount, setPendingCount] = useState(0);

  useEffect(() => {
    // This would need to track pending background sync mutations
    // For now, just a placeholder
    const checkPending = async () => {
      if ('serviceWorker' in navigator && 'SyncManager' in window) {
        // Check for pending background sync
        const _registration = await navigator.serviceWorker.ready;
        // Note: SyncManager API is limited, this is simplified
        setPendingCount(0);
      }
    };

    checkPending();
    const interval = setInterval(checkPending, 5000);

    return () => clearInterval(interval);
  }, []);

  if (pendingCount === 0) return null;

  return (
    <div className="fixed bottom-4 right-4 bg-blue-500 text-white px-4 py-2 rounded-lg shadow-lg z-50">
      {pendingCount} pending changes will sync when online
    </div>
  );
}

// =============================================================================
// Cache Management Component
// =============================================================================

interface CacheManagerProps {
  onClearCache?: () => void;
}

export function CacheManager({ onClearCache }: CacheManagerProps) {
  const [stats, setStats] = useState(cache.getStats());

  useEffect(() => {
    const interval = setInterval(() => {
      setStats(cache.getStats());
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  const handleClearCache = () => {
    cache.clear();
    onClearCache?.();
    setStats(cache.getStats());
  };

  const handleClearExpired = () => {
    const cleared = cache.clearExpired();
    // eslint-disable-next-line no-console
    console.log(`Cleared ${cleared} expired items`);
    setStats(cache.getStats());
  };

  return (
    <div className="p-4 bg-gray-100 rounded-lg">
      <h3 className="font-semibold mb-2">Cache Statistics</h3>
      <div className="grid grid-cols-2 gap-2 text-sm mb-4">
        <div>Items: {stats.itemCount}</div>
        <div>Size: {(stats.size / 1024).toFixed(2)} KB</div>
        <div>Hits: {stats.hits}</div>
        <div>Misses: {stats.misses}</div>
        <div className="col-span-2">Hit Rate: {stats.hitRate.toFixed(1)}%</div>
      </div>
      <div className="flex gap-2">
        <button
          onClick={handleClearExpired}
          className="px-3 py-1 bg-yellow-500 text-white rounded text-sm"
        >
          Clear Expired
        </button>
        <button
          onClick={handleClearCache}
          className="px-3 py-1 bg-red-500 text-white rounded text-sm"
        >
          Clear All
        </button>
      </div>
    </div>
  );
}

// =============================================================================
// Exports
// =============================================================================

export default QueryProvider;
