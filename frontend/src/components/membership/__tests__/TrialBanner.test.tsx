import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import type { TenantMembership } from '@/types/membership';
import type { Tenant, TenantAccess } from '@/types/tenant';
import { useTenantStore } from '@store/tenantStore';

const getTenantMembership =
  vi.fn<(tenantId: string) => Promise<AxiosResponse<TenantMembership>>>();
const getMyAccess =
  vi.fn<(tenantId: string) => Promise<AxiosResponse<TenantAccess>>>();

vi.mock('@/services/membershipService', () => ({
  membershipApi: {
    getTenantMembership: (id: string) => getTenantMembership(id),
  },
  tenantMembershipQueryKey: (id: string) => ['tenant-membership', id],
}));

vi.mock('@services/tenantService', () => ({
  tenantApi: { getMyAccess: (id: string) => getMyAccess(id) },
}));

import { TrialBanner } from '../TrialBanner';

function ok<T>(data: T): AxiosResponse<T> {
  return {
    data,
    status: 200,
    statusText: 'OK',
    headers: {},
    config: { headers: {} } as InternalAxiosRequestConfig,
  };
}

const tenant: Tenant = {
  id: 'tenant-1',
  name: 'Acme Cleaning',
  slug: 'acme',
  owner_id: 'owner-1',
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

const access = (role: string, permissions: string[]): TenantAccess => ({
  tenant_id: 'tenant-1',
  role_id: `r-${role}`,
  role,
  role_level: 30,
  permissions,
});

const trialEnded = {
  trial: { ended: true },
  has_paid_subscription: false,
} as unknown as TenantMembership;

const renderBanner = () =>
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <MemoryRouter>
        <TrialBanner />
      </MemoryRouter>
    </QueryClientProvider>
  );

describe('TrialBanner (T272)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getTenantMembership.mockResolvedValue(ok(trialEnded));
  });

  it('shows the ended-trial banner to members who may read the plan', async () => {
    useTenantStore.setState({ currentTenant: tenant, tenants: [tenant] });
    getMyAccess.mockResolvedValue(ok(access('owner', ['tenants.read'])));
    renderBanner();

    expect(
      await screen.findByText(/Your free trial has ended/)
    ).toBeInTheDocument();
    expect(getTenantMembership).toHaveBeenCalledWith('tenant-1');
  });

  it('never requests the plan for a technician', async () => {
    const tech = { ...tenant, role: 'technician' };
    useTenantStore.setState({ currentTenant: tech, tenants: [tech] });
    getMyAccess.mockResolvedValue(
      ok(access('technician', ['jobs.read', 'jobs.list', 'jobs.update']))
    );
    renderBanner();

    await waitFor(() => expect(getMyAccess).toHaveBeenCalled());
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(getTenantMembership).not.toHaveBeenCalled();
    expect(screen.queryByText(/Your free trial has ended/)).toBeNull();
  });

  it('waits for the permission list before requesting the plan', () => {
    useTenantStore.setState({ currentTenant: tenant, tenants: [tenant] });
    getMyAccess.mockReturnValue(new Promise(() => {}));
    renderBanner();

    expect(getTenantMembership).not.toHaveBeenCalled();
  });
});
