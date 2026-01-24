/* eslint-disable react-refresh/only-export-components */
/**
 * =============================================================================
 * Service Worker Registration
 * =============================================================================
 * Handles service worker lifecycle and updates
 */

import { registerSW } from 'virtual:pwa-register';

// =============================================================================
// Types
// =============================================================================

interface ServiceWorkerConfig {
  onNeedRefresh?: () => void;
  onOfflineReady?: () => void;
  onRegistered?: (registration: ServiceWorkerRegistration | undefined) => void;
  onRegisterError?: (error: Error) => void;
}

interface ServiceWorkerState {
  isRegistered: boolean;
  needsRefresh: boolean;
  isOfflineReady: boolean;
  registration: ServiceWorkerRegistration | null;
}

// =============================================================================
// State
// =============================================================================

const swState: ServiceWorkerState = {
  isRegistered: false,
  needsRefresh: false,
  isOfflineReady: false,
  registration: null,
};

let updateSW: ((reloadPage?: boolean) => Promise<void>) | null = null;

// =============================================================================
// Registration
// =============================================================================

/**
 * Register the service worker with callbacks
 */
export function registerServiceWorker(
  config: ServiceWorkerConfig = {}
): () => void {
  const { onNeedRefresh, onOfflineReady, onRegistered, onRegisterError } =
    config;

  // Skip in development without explicit enable
  if (import.meta.env.DEV && !import.meta.env.VITE_ENABLE_SW) {
    // eslint-disable-next-line no-console
    console.log('Service Worker: Skipped in development');
    return () => {};
  }

  // Check for service worker support
  if (!('serviceWorker' in navigator)) {
    console.warn('Service Worker: Not supported in this browser');
    return () => {};
  }

  // Register service worker
  updateSW = registerSW({
    immediate: true,
    onNeedRefresh() {
      swState.needsRefresh = true;
      // eslint-disable-next-line no-console
      console.log('Service Worker: New content available');
      onNeedRefresh?.();
    },
    onOfflineReady() {
      swState.isOfflineReady = true;
      // eslint-disable-next-line no-console
      console.log('Service Worker: App ready for offline use');
      onOfflineReady?.();
    },
    onRegistered(registration) {
      swState.isRegistered = true;
      swState.registration = registration ?? null;
      // eslint-disable-next-line no-console
      console.log('Service Worker: Registered', registration);
      onRegistered?.(registration);

      // Check for updates periodically
      if (registration) {
        setInterval(
          () => {
            registration.update();
          },
          60 * 60 * 1000
        ); // Every hour
      }
    },
    onRegisterError(error) {
      // eslint-disable-next-line no-console
      console.error('Service Worker: Registration failed', error);
      onRegisterError?.(error);
    },
  });

  // Return cleanup function
  return () => {
    // Unregister if needed
  };
}

// =============================================================================
// Update Functions
// =============================================================================

/**
 * Apply pending service worker update
 */
export async function applyUpdate(): Promise<void> {
  if (updateSW && swState.needsRefresh) {
    await updateSW(true);
  }
}

/**
 * Skip waiting and activate new service worker
 */
export async function skipWaiting(): Promise<void> {
  if (swState.registration?.waiting) {
    swState.registration.waiting.postMessage({ type: 'SKIP_WAITING' });
  }
}

/**
 * Check for service worker updates
 */
export async function checkForUpdates(): Promise<boolean> {
  if (swState.registration) {
    await swState.registration.update();
    return swState.needsRefresh;
  }
  return false;
}

// =============================================================================
// Cache Management
// =============================================================================

/**
 * Clear all service worker caches
 */
export async function clearServiceWorkerCache(): Promise<void> {
  if (swState.registration?.active) {
    swState.registration.active.postMessage({ type: 'CLEAR_CACHE' });
  }

  // Also clear via caches API
  const cacheNames = await caches.keys();
  await Promise.all(cacheNames.map((name) => caches.delete(name)));
}

/**
 * Clear API cache only
 */
export async function clearApiCache(): Promise<void> {
  if (swState.registration?.active) {
    swState.registration.active.postMessage({ type: 'CLEAR_API_CACHE' });
  }
}

/**
 * Invalidate cache for specific patterns
 */
export async function invalidateCachePatterns(
  patterns: string[]
): Promise<void> {
  if (swState.registration?.active) {
    swState.registration.active.postMessage({
      type: 'INVALIDATE_CACHE',
      patterns,
    });
  }
}

/**
 * Get cache statistics from service worker
 */
export async function getCacheStats(): Promise<{
  caches: Record<string, { count: number; size: number }>;
  totalSize: number;
} | null> {
  if (!swState.registration?.active) {
    return null;
  }

  return new Promise((resolve) => {
    const messageChannel = new MessageChannel();

    messageChannel.port1.onmessage = (event) => {
      resolve(event.data);
    };

    swState.registration!.active!.postMessage({ type: 'GET_CACHE_STATS' }, [
      messageChannel.port2,
    ]);

    // Timeout after 5 seconds
    setTimeout(() => resolve(null), 5000);
  });
}

// =============================================================================
// State Accessors
// =============================================================================

/**
 * Get current service worker state
 */
export function getServiceWorkerState(): ServiceWorkerState {
  return { ...swState };
}

/**
 * Check if service worker is registered
 */
export function isServiceWorkerRegistered(): boolean {
  return swState.isRegistered;
}

/**
 * Check if update is available
 */
export function isUpdateAvailable(): boolean {
  return swState.needsRefresh;
}

/**
 * Check if app is ready for offline use
 */
export function isOfflineReady(): boolean {
  return swState.isOfflineReady;
}

// =============================================================================
// React Hook for Service Worker
// =============================================================================

import { useState, useEffect, useCallback } from 'react';

interface UseServiceWorkerReturn {
  isRegistered: boolean;
  needsRefresh: boolean;
  isOfflineReady: boolean;
  applyUpdate: () => Promise<void>;
  checkForUpdates: () => Promise<boolean>;
  clearCache: () => Promise<void>;
}

export function useServiceWorker(): UseServiceWorkerReturn {
  const [state, setState] = useState<ServiceWorkerState>({ ...swState });

  useEffect(() => {
    const cleanup = registerServiceWorker({
      onNeedRefresh: () => setState((s) => ({ ...s, needsRefresh: true })),
      onOfflineReady: () => setState((s) => ({ ...s, isOfflineReady: true })),
      onRegistered: (registration) =>
        setState((s) => ({
          ...s,
          isRegistered: true,
          registration: registration ?? null,
        })),
    });

    return cleanup;
  }, []);

  const handleApplyUpdate = useCallback(async () => {
    await applyUpdate();
  }, []);

  const handleCheckForUpdates = useCallback(async () => {
    return checkForUpdates();
  }, []);

  const handleClearCache = useCallback(async () => {
    await clearServiceWorkerCache();
  }, []);

  return {
    isRegistered: state.isRegistered,
    needsRefresh: state.needsRefresh,
    isOfflineReady: state.isOfflineReady,
    applyUpdate: handleApplyUpdate,
    checkForUpdates: handleCheckForUpdates,
    clearCache: handleClearCache,
  };
}

// =============================================================================
// Update Prompt Component
// =============================================================================

import React from 'react';

interface UpdatePromptProps {
  onUpdate: () => void;
  onDismiss: () => void;
}

export function UpdatePrompt({
  onUpdate,
  onDismiss,
}: UpdatePromptProps): JSX.Element | null {
  const { needsRefresh } = useServiceWorker();

  if (!needsRefresh) return null;

  return (
    <div className="fixed bottom-4 right-4 bg-white rounded-lg shadow-xl border border-gray-200 p-4 max-w-sm z-50">
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0">
          <svg
            className="w-6 h-6 text-blue-500"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
        </div>
        <div className="flex-1">
          <h3 className="font-semibold text-gray-900">Update Available</h3>
          <p className="text-sm text-gray-600 mt-1">
            A new version of the app is available. Reload to update?
          </p>
          <div className="flex gap-2 mt-3">
            <button
              onClick={() => {
                onUpdate();
                applyUpdate();
              }}
              className="px-3 py-1.5 bg-blue-500 text-white text-sm rounded-md hover:bg-blue-600 transition-colors"
            >
              Reload
            </button>
            <button
              onClick={onDismiss}
              className="px-3 py-1.5 bg-gray-100 text-gray-700 text-sm rounded-md hover:bg-gray-200 transition-colors"
            >
              Later
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// =============================================================================
// Exports
// =============================================================================

export default registerServiceWorker;
