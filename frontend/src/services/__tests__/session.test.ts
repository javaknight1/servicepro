import { describe, it, expect, beforeEach, vi } from 'vitest';
import { QueryClient } from '@tanstack/react-query';

vi.mock('@/services/tenantService', () => ({
  tenantApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    getMembers: vi.fn(),
    addMember: vi.fn(),
    removeMember: vi.fn(),
    updateMemberRole: vi.fn(),
  },
}));

import api, { authApi, userApi } from '../api';
import { tenantApi } from '@/services/tenantService';
import {
  REACT_QUERY_PERSIST_KEY,
  getQueryPersistBuster,
  registerQueryClient,
} from '../queryCacheRegistry';
import { cache } from '@/utils/cache';
import { useAuthStore } from '@/store/authStore';
import { useTenantStore } from '@/store/tenantStore';
import type { Tenant } from '@/types/tenant';
import type { User } from '@app-types';

const tenantA: Tenant = {
  id: 'tenant-a',
  name: 'Org A',
  slug: 'org-a',
  is_active: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const tenantB: Tenant = { ...tenantA, id: 'tenant-b', name: 'Org B' };

const userA = { id: 'user-a', email: 'a@example.com' } as unknown as User;
const userB = { id: 'user-b', email: 'b@example.com' } as unknown as User;

function seedSession(client: QueryClient): void {
  client.setQueryData(['customers'], [{ id: 'secret-customer' }]);
  cache.set(REACT_QUERY_PERSIST_KEY, '{"clientState":{}}');
  localStorage.setItem('current_tenant_id', tenantA.id);
  useTenantStore.setState({ currentTenant: tenantA, tenants: [tenantA] });
  api.defaults.headers.common['X-Tenant-ID'] = tenantA.id;
}

describe('session isolation', () => {
  let client: QueryClient;

  beforeEach(() => {
    vi.restoreAllMocks();
    client = new QueryClient();
    registerQueryClient(client);
    useAuthStore.setState({ user: userA, isAuthenticated: true });
    useTenantStore.setState({ currentTenant: null, tenants: [] });
  });

  it('logout clears query cache, persisted cache, tenant state and header', async () => {
    vi.spyOn(authApi, 'logout').mockResolvedValue(
      {} as Awaited<ReturnType<typeof authApi.logout>>
    );
    seedSession(client);

    await useAuthStore.getState().logout();

    expect(client.getQueryData(['customers'])).toBeUndefined();
    expect(cache.get(REACT_QUERY_PERSIST_KEY)).toBeNull();
    expect(localStorage.getItem('current_tenant_id')).toBeNull();
    expect(useTenantStore.getState().currentTenant).toBeNull();
    expect(useTenantStore.getState().tenants).toEqual([]);
    expect(api.defaults.headers.common['X-Tenant-ID']).toBeUndefined();
    expect(useAuthStore.getState().user).toBeNull();
  });

  it('logout still clears local state when the API call fails', async () => {
    vi.spyOn(authApi, 'logout').mockRejectedValue(new Error('network'));
    vi.spyOn(console, 'error').mockImplementation(() => undefined);
    seedSession(client);

    await useAuthStore.getState().logout();

    expect(client.getQueryData(['customers'])).toBeUndefined();
    expect(localStorage.getItem('current_tenant_id')).toBeNull();
  });

  it('switching tenant clears cached data from the previous tenant', async () => {
    seedSession(client);
    useTenantStore.setState({ tenants: [tenantA, tenantB] });

    await useTenantStore.getState().switchTenant(tenantB.id);

    expect(client.getQueryData(['customers'])).toBeUndefined();
    expect(cache.get(REACT_QUERY_PERSIST_KEY)).toBeNull();
    expect(localStorage.getItem('current_tenant_id')).toBe(tenantB.id);
  });

  it('re-selecting the same tenant keeps the cache', async () => {
    seedSession(client);

    await useTenantStore.getState().switchTenant(tenantA.id);

    expect(client.getQueryData(['customers'])).toEqual([
      { id: 'secret-customer' },
    ]);
  });

  it('loadTenants drops a persisted tenant the user is not a member of', async () => {
    seedSession(client);
    vi.mocked(tenantApi.list).mockResolvedValue({
      data: { tenants: [tenantB] },
    } as Awaited<ReturnType<typeof tenantApi.list>>);

    await useTenantStore.getState().loadTenants();

    expect(useTenantStore.getState().currentTenant?.id).toBe(tenantB.id);
    expect(localStorage.getItem('current_tenant_id')).toBe(tenantB.id);
    expect(client.getQueryData(['customers'])).toBeUndefined();
  });

  it('loadTenants persists the default tenant id so requests carry the header', async () => {
    vi.mocked(tenantApi.list).mockResolvedValue({
      data: { tenants: [tenantA, tenantB] },
    } as Awaited<ReturnType<typeof tenantApi.list>>);

    await useTenantStore.getState().loadTenants();

    expect(localStorage.getItem('current_tenant_id')).toBe(tenantA.id);
  });

  it('checkAuth for a different user than the persisted one clears session data', async () => {
    seedSession(client);
    vi.spyOn(userApi, 'getCurrentUser').mockResolvedValue({
      data: userB,
    } as Awaited<ReturnType<typeof userApi.getCurrentUser>>);

    await useAuthStore.getState().checkAuth();

    expect(client.getQueryData(['customers'])).toBeUndefined();
    expect(localStorage.getItem('current_tenant_id')).toBeNull();
    expect(useAuthStore.getState().user?.id).toBe(userB.id);
  });

  it('scopes the persistence buster to user and tenant', () => {
    const a = getQueryPersistBuster('user-a', 'tenant-a');
    expect(a).toContain('user-a');
    expect(a).toContain('tenant-a');
    expect(getQueryPersistBuster('user-b', 'tenant-a')).not.toBe(a);
    expect(getQueryPersistBuster('user-a', 'tenant-b')).not.toBe(a);
    expect(getQueryPersistBuster(null, null)).toContain('anon');
  });
});
