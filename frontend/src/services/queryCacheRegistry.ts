/**
 * Query cache registry
 *
 * Holds a reference to the app's QueryClient so non-React code (stores,
 * the axios interceptor) can wipe cached server data on logout, session
 * expiry, or tenant switch. Deliberately has no app imports beyond the
 * storage cache to stay free of import cycles.
 */
import type { QueryClient } from '@tanstack/react-query';
import { cache } from '@/utils/cache';

/** Storage key of the persisted React Query cache (under the `sp_` prefix). */
export const REACT_QUERY_PERSIST_KEY = 'react-query-cache';

let registeredClient: QueryClient | null = null;

export function registerQueryClient(client: QueryClient | null): void {
  registeredClient = client;
}

/** Remove the persisted (localStorage) copy of the query cache. */
export function clearPersistedQueryCache(): void {
  cache.delete(REACT_QUERY_PERSIST_KEY);
}

/** Clear both the in-memory query cache and its persisted copy. */
export function clearQueryCaches(): void {
  registeredClient?.clear();
  clearPersistedQueryCache();
}

/**
 * Persistence buster scoped to the app version, user and tenant. A persisted
 * cache written for a different user or tenant is discarded on restore.
 */
export function getQueryPersistBuster(
  userId: string | null | undefined,
  tenantId: string | null | undefined
): string {
  const version = import.meta.env.VITE_APP_VERSION || '1.0.0';
  return `${version}:${userId ?? 'anon'}:${tenantId ?? 'none'}`;
}
