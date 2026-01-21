/**
 * =============================================================================
 * Hooks Index
 * =============================================================================
 * Export all custom hooks
 */

// Cache hooks
export {
  useCachedQuery,
  useCachedMutation,
  useLocalCache,
  useSessionCache,
  useCacheStats,
  useOfflineStatus,
  usePrefetch,
  useServiceWorkerCache,
  useQueryInvalidation,
  CACHE_TIMES,
  getCachePolicy,
} from './useCache';

// Roles hook
export { useRoles } from './useRoles';

// WebSocket hook
export { useWebSocket } from './useWebSocket';
