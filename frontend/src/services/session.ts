/**
 * Session teardown helpers.
 *
 * On logout (or an expired session) everything that could leak one user's or
 * tenant's data to the next person on the same browser must be cleared:
 * the in-memory and persisted React Query cache, the persisted tenant store,
 * the saved current tenant id and any X-Tenant-ID default header.
 */
import api, { CURRENT_TENANT_KEY, clearTenantHeader } from './api';
import { clearQueryCaches } from './queryCacheRegistry';
import { useTenantStore } from '@/store/tenantStore';

export const TENANT_STORAGE_KEY = 'tenant-storage';

/** Remove all saved tenant context (store, storage keys, request header). */
export function clearTenantContext(): void {
  useTenantStore.getState().reset();
  useTenantStore.persist?.clearStorage();
  localStorage.removeItem(TENANT_STORAGE_KEY);
  localStorage.removeItem(CURRENT_TENANT_KEY);
  clearTenantHeader(api);
}

/** Clear every piece of per-user / per-tenant client state. */
export function clearSessionState(): void {
  clearQueryCaches();
  clearTenantContext();
}
