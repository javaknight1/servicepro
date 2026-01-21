/**
 * =============================================================================
 * Cache Configuration
 * =============================================================================
 * Central configuration for all caching strategies
 */

// =============================================================================
// Cache Time Constants (in milliseconds)
// =============================================================================

export const CACHE_TIMES = {
  // Short-lived cache for frequently changing data
  SHORT: 1 * 60 * 1000, // 1 minute

  // Default cache duration
  DEFAULT: 5 * 60 * 1000, // 5 minutes

  // Medium cache for semi-static data
  MEDIUM: 15 * 60 * 1000, // 15 minutes

  // Long cache for rarely changing data
  LONG: 30 * 60 * 1000, // 30 minutes

  // Extended cache for static reference data
  EXTENDED: 60 * 60 * 1000, // 1 hour

  // Permanent cache (until manual invalidation)
  PERMANENT: Infinity,
} as const;

// =============================================================================
// React Query Configuration
// =============================================================================

export const QUERY_CONFIG = {
  // Default options for all queries
  defaultOptions: {
    queries: {
      // Data considered fresh for this duration
      staleTime: CACHE_TIMES.DEFAULT,

      // Cache kept for this duration after becoming unused
      gcTime: CACHE_TIMES.LONG, // formerly cacheTime

      // Retry failed requests
      retry: 3,
      retryDelay: (attemptIndex: number) =>
        Math.min(1000 * 2 ** attemptIndex, 30000),

      // Refetch behaviors
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
      refetchOnMount: true,

      // Network mode
      networkMode: 'offlineFirst' as const,
    },
    mutations: {
      // Retry mutations once
      retry: 1,

      // Network mode for mutations
      networkMode: 'online' as const,
    },
  },
} as const;

// =============================================================================
// Per-Endpoint Cache Policies
// =============================================================================

export type CachePolicy = {
  staleTime: number;
  gcTime: number;
  refetchOnWindowFocus?: boolean;
  refetchInterval?: number | false;
};

export const ENDPOINT_CACHE_POLICIES: Record<string, CachePolicy> = {
  // User/Auth - short cache, frequent updates
  'auth.currentUser': {
    staleTime: CACHE_TIMES.SHORT,
    gcTime: CACHE_TIMES.DEFAULT,
    refetchOnWindowFocus: true,
  },
  'auth.permissions': {
    staleTime: CACHE_TIMES.MEDIUM,
    gcTime: CACHE_TIMES.LONG,
    refetchOnWindowFocus: false,
  },

  // Customers - medium cache
  'customers.list': {
    staleTime: CACHE_TIMES.DEFAULT,
    gcTime: CACHE_TIMES.LONG,
    refetchOnWindowFocus: true,
  },
  'customers.detail': {
    staleTime: CACHE_TIMES.DEFAULT,
    gcTime: CACHE_TIMES.LONG,
    refetchOnWindowFocus: true,
  },
  'customers.search': {
    staleTime: CACHE_TIMES.SHORT,
    gcTime: CACHE_TIMES.DEFAULT,
    refetchOnWindowFocus: false,
  },

  // Jobs - short cache, frequently updated
  'jobs.list': {
    staleTime: CACHE_TIMES.SHORT,
    gcTime: CACHE_TIMES.DEFAULT,
    refetchOnWindowFocus: true,
  },
  'jobs.detail': {
    staleTime: CACHE_TIMES.SHORT,
    gcTime: CACHE_TIMES.DEFAULT,
    refetchOnWindowFocus: true,
  },
  'jobs.schedule': {
    staleTime: CACHE_TIMES.SHORT,
    gcTime: CACHE_TIMES.DEFAULT,
    refetchOnWindowFocus: true,
    refetchInterval: 60000, // Refresh every minute
  },

  // Invoices - medium cache
  'invoices.list': {
    staleTime: CACHE_TIMES.DEFAULT,
    gcTime: CACHE_TIMES.LONG,
    refetchOnWindowFocus: true,
  },
  'invoices.detail': {
    staleTime: CACHE_TIMES.DEFAULT,
    gcTime: CACHE_TIMES.LONG,
    refetchOnWindowFocus: true,
  },

  // Quotes - medium cache
  'quotes.list': {
    staleTime: CACHE_TIMES.DEFAULT,
    gcTime: CACHE_TIMES.LONG,
    refetchOnWindowFocus: true,
  },
  'quotes.detail': {
    staleTime: CACHE_TIMES.DEFAULT,
    gcTime: CACHE_TIMES.LONG,
    refetchOnWindowFocus: true,
  },

  // Dashboard - short cache, needs fresh data
  'dashboard.summary': {
    staleTime: CACHE_TIMES.SHORT,
    gcTime: CACHE_TIMES.DEFAULT,
    refetchOnWindowFocus: true,
    refetchInterval: 120000, // Refresh every 2 minutes
  },
  'dashboard.revenue': {
    staleTime: CACHE_TIMES.DEFAULT,
    gcTime: CACHE_TIMES.LONG,
    refetchOnWindowFocus: true,
  },
  'dashboard.metrics': {
    staleTime: CACHE_TIMES.SHORT,
    gcTime: CACHE_TIMES.DEFAULT,
    refetchOnWindowFocus: true,
  },

  // Reference data - long cache
  'settings.general': {
    staleTime: CACHE_TIMES.LONG,
    gcTime: CACHE_TIMES.EXTENDED,
    refetchOnWindowFocus: false,
  },
  'settings.templates': {
    staleTime: CACHE_TIMES.LONG,
    gcTime: CACHE_TIMES.EXTENDED,
    refetchOnWindowFocus: false,
  },
  'roles.list': {
    staleTime: CACHE_TIMES.LONG,
    gcTime: CACHE_TIMES.EXTENDED,
    refetchOnWindowFocus: false,
  },
  'users.list': {
    staleTime: CACHE_TIMES.MEDIUM,
    gcTime: CACHE_TIMES.LONG,
    refetchOnWindowFocus: true,
  },
};

// =============================================================================
// Service Worker Cache Configuration
// =============================================================================

export const SW_CACHE_CONFIG = {
  // Cache names
  cacheNames: {
    static: 'servicepro-static-v1',
    api: 'servicepro-api-v1',
    images: 'servicepro-images-v1',
    fonts: 'servicepro-fonts-v1',
  },

  // Static assets - cache first
  staticAssets: {
    maxEntries: 100,
    maxAgeSeconds: 30 * 24 * 60 * 60, // 30 days
  },

  // API responses - network first with cache fallback
  apiResponses: {
    maxEntries: 200,
    maxAgeSeconds: 5 * 60, // 5 minutes
  },

  // Images - cache first with expiration
  images: {
    maxEntries: 100,
    maxAgeSeconds: 7 * 24 * 60 * 60, // 7 days
  },

  // Fonts - cache first, long expiration
  fonts: {
    maxEntries: 30,
    maxAgeSeconds: 365 * 24 * 60 * 60, // 1 year
  },
} as const;

// =============================================================================
// Local Storage Cache Configuration
// =============================================================================

export const LOCAL_CACHE_CONFIG = {
  // Prefix for all cache keys
  prefix: 'sp_cache_',

  // Default expiry for cached items
  defaultExpiry: CACHE_TIMES.DEFAULT,

  // Maximum size of individual cached items (in characters)
  maxItemSize: 500 * 1024, // 500KB

  // Maximum total cache size
  maxTotalSize: 5 * 1024 * 1024, // 5MB

  // Keys to persist (not cleared on logout)
  persistentKeys: ['theme', 'locale', 'preferences'],

  // Keys with custom expiry
  customExpiry: {
    'user.preferences': CACHE_TIMES.EXTENDED,
    'app.version': CACHE_TIMES.PERMANENT,
    'feature.flags': CACHE_TIMES.LONG,
  },
} as const;

// =============================================================================
// Cache Invalidation Rules
// =============================================================================

export const CACHE_INVALIDATION_RULES: Record<string, string[]> = {
  // When a customer is created/updated/deleted
  'customers.mutate': [
    'customers.list',
    'customers.search',
    'dashboard.summary',
  ],

  // When a job is created/updated/deleted
  'jobs.mutate': [
    'jobs.list',
    'jobs.schedule',
    'dashboard.summary',
    'dashboard.metrics',
  ],

  // When an invoice is created/updated/deleted
  'invoices.mutate': [
    'invoices.list',
    'dashboard.summary',
    'dashboard.revenue',
  ],

  // When a quote is created/updated/deleted
  'quotes.mutate': ['quotes.list', 'dashboard.summary'],

  // When a payment is recorded
  'payments.mutate': [
    'invoices.list',
    'invoices.detail',
    'dashboard.revenue',
    'dashboard.summary',
  ],

  // When user settings change
  'settings.mutate': ['settings.general', 'auth.currentUser'],
};

// =============================================================================
// Utility Functions
// =============================================================================

/**
 * Get cache policy for a specific endpoint
 */
export function getCachePolicy(endpoint: string): CachePolicy {
  return (
    ENDPOINT_CACHE_POLICIES[endpoint] || {
      staleTime: CACHE_TIMES.DEFAULT,
      gcTime: CACHE_TIMES.LONG,
      refetchOnWindowFocus: true,
    }
  );
}

/**
 * Get keys to invalidate for a mutation
 */
export function getInvalidationKeys(mutationType: string): string[] {
  return CACHE_INVALIDATION_RULES[mutationType] || [];
}

/**
 * Check if a cache key should persist across sessions
 */
export function isPersistentKey(key: string): boolean {
  return LOCAL_CACHE_CONFIG.persistentKeys.some((persistentKey) =>
    key.includes(persistentKey)
  );
}

/**
 * Get custom expiry for a cache key
 */
export function getCustomExpiry(key: string): number {
  for (const [pattern, expiry] of Object.entries(
    LOCAL_CACHE_CONFIG.customExpiry
  )) {
    if (key.includes(pattern)) {
      return expiry;
    }
  }
  return LOCAL_CACHE_CONFIG.defaultExpiry;
}
