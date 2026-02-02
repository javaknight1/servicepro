/// <reference lib="webworker" />

/**
 * =============================================================================
 * Service Worker for ServicePro
 * =============================================================================
 * Provides offline support and intelligent caching strategies
 */

import { precacheAndRoute, cleanupOutdatedCaches } from 'workbox-precaching';
import { registerRoute, NavigationRoute } from 'workbox-routing';
import {
  CacheFirst,
  NetworkFirst,
  StaleWhileRevalidate,
  NetworkOnly,
} from 'workbox-strategies';
import { ExpirationPlugin } from 'workbox-expiration';
import { CacheableResponsePlugin } from 'workbox-cacheable-response';
import { BackgroundSyncPlugin } from 'workbox-background-sync';

declare const self: ServiceWorkerGlobalScope;

// =============================================================================
// Configuration
// =============================================================================

const CACHE_NAMES = {
  static: 'servicepro-static-v1',
  api: 'servicepro-api-v1',
  images: 'servicepro-images-v1',
  fonts: 'servicepro-fonts-v1',
};

// =============================================================================
// Precache Static Assets
// =============================================================================

// Precache all static assets from the build
precacheAndRoute(self.__WB_MANIFEST);

// Clean up old caches
cleanupOutdatedCaches();

// =============================================================================
// Navigation Routes (SPA Support)
// =============================================================================

// Handle navigation requests with a Network-First strategy
// Falls back to cached app shell for offline support
const navigationRoute = new NavigationRoute(
  new NetworkFirst({
    cacheName: CACHE_NAMES.static,
    plugins: [
      new CacheableResponsePlugin({
        statuses: [200],
      }),
    ],
  }),
  {
    // Don't handle API routes or specific paths
    denylist: [/\/api\//, /\/auth\//, /\.json$/],
  }
);

registerRoute(navigationRoute);

// =============================================================================
// Static Assets (Cache First)
// =============================================================================

// JavaScript and CSS files
registerRoute(
  ({ request }) =>
    request.destination === 'script' || request.destination === 'style',
  new CacheFirst({
    cacheName: CACHE_NAMES.static,
    plugins: [
      new CacheableResponsePlugin({
        statuses: [0, 200],
      }),
      new ExpirationPlugin({
        maxEntries: 100,
        maxAgeSeconds: 30 * 24 * 60 * 60, // 30 days
      }),
    ],
  })
);

// =============================================================================
// Fonts (Cache First, Long Expiration)
// =============================================================================

registerRoute(
  ({ request }) => request.destination === 'font',
  new CacheFirst({
    cacheName: CACHE_NAMES.fonts,
    plugins: [
      new CacheableResponsePlugin({
        statuses: [0, 200],
      }),
      new ExpirationPlugin({
        maxEntries: 30,
        maxAgeSeconds: 365 * 24 * 60 * 60, // 1 year
      }),
    ],
  })
);

// =============================================================================
// Images (Cache First with Expiration)
// =============================================================================

registerRoute(
  ({ request }) => request.destination === 'image',
  new CacheFirst({
    cacheName: CACHE_NAMES.images,
    plugins: [
      new CacheableResponsePlugin({
        statuses: [0, 200],
      }),
      new ExpirationPlugin({
        maxEntries: 100,
        maxAgeSeconds: 7 * 24 * 60 * 60, // 7 days
        purgeOnQuotaError: true,
      }),
    ],
  })
);

// =============================================================================
// API Routes (Network First with Cache Fallback)
// =============================================================================

// GET requests to API - Network first, cache fallback
registerRoute(
  ({ url, request }) =>
    url.pathname.startsWith('/api/') && request.method === 'GET',
  new NetworkFirst({
    cacheName: CACHE_NAMES.api,
    networkTimeoutSeconds: 10,
    plugins: [
      new CacheableResponsePlugin({
        statuses: [200],
      }),
      new ExpirationPlugin({
        maxEntries: 200,
        maxAgeSeconds: 5 * 60, // 5 minutes
      }),
    ],
  })
);

// Specific endpoints that should be cached longer
const longCacheEndpoints = [
  '/api/v1/settings',
  '/api/v1/roles',
  '/api/v1/templates',
];

registerRoute(
  ({ url }) => longCacheEndpoints.some((ep) => url.pathname.startsWith(ep)),
  new StaleWhileRevalidate({
    cacheName: CACHE_NAMES.api,
    plugins: [
      new CacheableResponsePlugin({
        statuses: [200],
      }),
      new ExpirationPlugin({
        maxEntries: 50,
        maxAgeSeconds: 60 * 60, // 1 hour
      }),
    ],
  })
);

// =============================================================================
// Background Sync for Mutations
// =============================================================================

// Create a background sync queue for failed mutations
const bgSyncPlugin = new BackgroundSyncPlugin('servicepro-mutations', {
  maxRetentionTime: 24 * 60, // Retry for up to 24 hours
  onSync: async ({ queue }) => {
    let entry;
    while ((entry = await queue.shiftRequest())) {
      try {
        await fetch(entry.request.clone());
        // eslint-disable-next-line no-console
        console.log(
          'Background sync: Successfully replayed request',
          entry.request.url
        );
      } catch (error) {
        console.error('Background sync: Failed to replay request', error);
        await queue.unshiftRequest(entry);
        throw error;
      }
    }
  },
});

// POST, PUT, PATCH, DELETE requests with background sync
registerRoute(
  ({ url, request }) =>
    url.pathname.startsWith('/api/') &&
    ['POST', 'PUT', 'PATCH', 'DELETE'].includes(request.method),
  new NetworkOnly({
    plugins: [bgSyncPlugin],
  }),
  'POST'
);

registerRoute(
  ({ url, request }) =>
    url.pathname.startsWith('/api/') && request.method === 'PUT',
  new NetworkOnly({
    plugins: [bgSyncPlugin],
  }),
  'PUT'
);

registerRoute(
  ({ url, request }) =>
    url.pathname.startsWith('/api/') && request.method === 'PATCH',
  new NetworkOnly({
    plugins: [bgSyncPlugin],
  }),
  'PATCH'
);

registerRoute(
  ({ url, request }) =>
    url.pathname.startsWith('/api/') && request.method === 'DELETE',
  new NetworkOnly({
    plugins: [bgSyncPlugin],
  }),
  'DELETE'
);

// =============================================================================
// Skip Caching for Auth Endpoints
// =============================================================================

// Auth endpoints should never be cached
registerRoute(
  ({ url }) =>
    url.pathname.startsWith('/api/v1/auth/') ||
    url.pathname.includes('/login') ||
    url.pathname.includes('/logout') ||
    url.pathname.includes('/refresh'),
  new NetworkOnly()
);

// =============================================================================
// Service Worker Lifecycle
// =============================================================================

// Install event - skip waiting to activate immediately
self.addEventListener('install', (event) => {
  // eslint-disable-next-line no-console
  console.log('Service Worker: Installing...');
  event.waitUntil(self.skipWaiting());
});

// Activate event - claim all clients
self.addEventListener('activate', (event) => {
  // eslint-disable-next-line no-console
  console.log('Service Worker: Activating...');
  event.waitUntil(
    Promise.all([
      // Clean up old caches
      caches.keys().then((cacheNames) => {
        return Promise.all(
          cacheNames
            .filter((name) => !Object.values(CACHE_NAMES).includes(name))
            .map((name) => {
              // eslint-disable-next-line no-console
              console.log('Service Worker: Deleting old cache', name);
              return caches.delete(name);
            })
        );
      }),
      // Claim all clients
      self.clients.claim(),
    ])
  );
});

// =============================================================================
// Message Handling
// =============================================================================

self.addEventListener('message', (event) => {
  if (!event.data) return;

  switch (event.data.type) {
    case 'SKIP_WAITING':
      self.skipWaiting();
      break;

    case 'CLEAR_CACHE':
      event.waitUntil(
        caches.keys().then((cacheNames) => {
          return Promise.all(cacheNames.map((name) => caches.delete(name)));
        })
      );
      break;

    case 'CLEAR_API_CACHE':
      event.waitUntil(caches.delete(CACHE_NAMES.api));
      break;

    case 'GET_CACHE_STATS':
      event.waitUntil(
        getCacheStats().then((stats) => {
          event.ports[0]?.postMessage(stats);
        })
      );
      break;

    case 'INVALIDATE_CACHE':
      if (event.data.patterns) {
        event.waitUntil(invalidateCachePatterns(event.data.patterns));
      }
      break;
  }
});

// =============================================================================
// Utility Functions
// =============================================================================

async function getCacheStats(): Promise<{
  caches: Record<string, { count: number; size: number }>;
  totalSize: number;
}> {
  const stats: Record<string, { count: number; size: number }> = {};
  let totalSize = 0;

  for (const [name, cacheName] of Object.entries(CACHE_NAMES)) {
    const cache = await caches.open(cacheName);
    const keys = await cache.keys();
    let size = 0;

    for (const request of keys) {
      const response = await cache.match(request);
      if (response) {
        const blob = await response.blob();
        size += blob.size;
      }
    }

    stats[name] = { count: keys.length, size };
    totalSize += size;
  }

  return { caches: stats, totalSize };
}

async function invalidateCachePatterns(patterns: string[]): Promise<void> {
  const cache = await caches.open(CACHE_NAMES.api);
  const keys = await cache.keys();

  for (const request of keys) {
    const url = new URL(request.url);

    for (const pattern of patterns) {
      if (url.pathname.includes(pattern)) {
        await cache.delete(request);
        // eslint-disable-next-line no-console
        console.log('Service Worker: Invalidated cache for', url.pathname);
        break;
      }
    }
  }
}

// eslint-disable-next-line no-console
console.log('Service Worker: Registered and ready');
