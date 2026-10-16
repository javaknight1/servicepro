import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

vi.mock('@services/tenantService', () => ({
  tenantApi: { getMyAccess: vi.fn() },
}));

import { tenantApi } from '@services/tenantService';
import {
  hasTenantPermission,
  TENANT_PERMISSIONS,
  useTenantPermissions,
} from '../tenantPermissions';
import type { TenantAccess } from '@/types/tenant';

const access = (permissions: string[], role = 'custom_role'): TenantAccess => ({
  tenant_id: 'tenant-1',
  role_id: 'role-1',
  role,
  role_level: 50,
  permissions,
});

describe('hasTenantPermission with the API permission list', () => {
  it('is an exact lookup, whatever the role name', () => {
    const office = access(['tenants.manage', 'tenants.read'], 'office_lead');
    expect(hasTenantPermission('tenants.manage', office, 'office_lead')).toBe(
      true
    );
    expect(hasTenantPermission('tenants.update', office, 'office_lead')).toBe(
      false
    );
    // A built-in name does not override the list.
    expect(hasTenantPermission('tenants.manage', access([]), 'admin')).toBe(
      false
    );
    expect(
      hasTenantPermission('tenants.manage', access(['tenants.manage']), 'user')
    ).toBe(true);
  });

  it('shows delete only when tenants.delete is granted', () => {
    expect(
      hasTenantPermission(
        TENANT_PERMISSIONS.delete,
        access(['tenants.manage', 'tenants.update']),
        'admin'
      )
    ).toBe(false);
    expect(
      hasTenantPermission(
        TENANT_PERMISSIONS.delete,
        access(['tenants.delete']),
        'admin'
      )
    ).toBe(true);
  });
});

describe('hasTenantPermission fallback while the list is loading', () => {
  it('hides management only for built-in non-admin roles', () => {
    const manage = TENANT_PERMISSIONS.manageMembers;
    expect(hasTenantPermission(manage, undefined, 'admin')).toBe(true);
    expect(hasTenantPermission(manage, undefined, 'Admin')).toBe(true);
    expect(hasTenantPermission(manage, undefined, 'manager')).toBe(false);
    expect(hasTenantPermission(manage, undefined, 'user')).toBe(false);
    expect(hasTenantPermission(manage, undefined, 'guest')).toBe(false);
    // Unknown or custom roles defer to the API's 403.
    expect(hasTenantPermission(manage, undefined, undefined)).toBe(true);
    expect(hasTenantPermission(manage, undefined, 'office_lead')).toBe(true);
  });

  it('never guesses tenants.delete or other permissions', () => {
    expect(
      hasTenantPermission(TENANT_PERMISSIONS.delete, undefined, 'super_admin')
    ).toBe(false);
    expect(hasTenantPermission(TENANT_PERMISSIONS.delete, null, 'admin')).toBe(
      false
    );
    expect(hasTenantPermission('customers.delete', undefined, 'admin')).toBe(
      false
    );
  });
});

describe('useTenantPermissions', () => {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      {children}
    </QueryClientProvider>
  );

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('uses the fallback while loading, then the API list', async () => {
    let resolve: (value: { data: TenantAccess }) => void = () => {};
    vi.mocked(tenantApi.getMyAccess).mockReturnValue(
      new Promise((r) => {
        resolve = r;
      }) as unknown as ReturnType<typeof tenantApi.getMyAccess>
    );

    const { result } = renderHook(
      () => useTenantPermissions({ id: 'tenant-1', role: 'admin' }),
      { wrapper }
    );
    expect(result.current.isLoading).toBe(true);
    expect(result.current.canManageMembers).toBe(true);
    expect(result.current.canDeleteTenant).toBe(false);

    resolve({
      data: access(['tenants.update', 'tenants.delete'], 'admin'),
    });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(tenantApi.getMyAccess).toHaveBeenCalledWith('tenant-1');
    expect(result.current.canManageMembers).toBe(false);
    expect(result.current.canUpdateTenant).toBe(true);
    expect(result.current.canDeleteTenant).toBe(true);
    expect(result.current.can('tenants.update')).toBe(true);
  });

  it('keeps delete hidden when the list lacks tenants.delete', async () => {
    vi.mocked(tenantApi.getMyAccess).mockResolvedValue({
      data: access(['tenants.manage', 'tenants.update'], 'admin'),
    } as unknown as Awaited<ReturnType<typeof tenantApi.getMyAccess>>);

    const { result } = renderHook(
      () => useTenantPermissions({ id: 'tenant-1', role: 'admin' }),
      { wrapper }
    );
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.canManageMembers).toBe(true);
    expect(result.current.canDeleteTenant).toBe(false);
  });

  it('does not fetch without a tenant', () => {
    const { result } = renderHook(() => useTenantPermissions(null), {
      wrapper,
    });
    expect(tenantApi.getMyAccess).not.toHaveBeenCalled();
    expect(result.current.isLoading).toBe(false);
    expect(result.current.canDeleteTenant).toBe(false);
  });
});
