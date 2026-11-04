import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Sidebar } from '../Sidebar';
import { useTenantStore } from '@store/tenantStore';
import { useMembershipStore } from '@store/membershipStore';
import { useSidebarStore } from '@store/sidebarStore';
import type { Tenant } from '@/types/tenant';
import type { TenantMembership } from '@/types/membership';

vi.mock('@components/tenants', () => ({
  TenantSwitcher: () => <div>Tenant switcher</div>,
}));

vi.mock('@services/tenantService', () => ({
  tenantApi: { getMyAccess: vi.fn() },
}));

import { tenantApi } from '@services/tenantService';

const tenant: Tenant = {
  id: 'tenant-1',
  name: 'Acme HVAC',
  slug: 'acme',
  owner_id: 'owner-1',
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

const teamPlan = {
  subscription: { tenant_id: 'tenant-1', tier_display_name: 'Team' },
  tier: { display_name: 'Team', features: { team_members: -1 } },
} as unknown as TenantMembership;

/** GET /tenants/:id/me answers with these permissions. */
const grant = (role: string, permissions: string[]) =>
  vi.mocked(tenantApi.getMyAccess).mockResolvedValue({
    data: {
      tenant_id: 'tenant-1',
      role_id: `r-${role}`,
      role,
      role_level: 30,
      permissions,
    },
  } as unknown as Awaited<ReturnType<typeof tenantApi.getMyAccess>>);

const renderAt = (path: string) =>
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <MemoryRouter initialEntries={[path]}>
        <Sidebar />
      </MemoryRouter>
    </QueryClientProvider>
  );

const linkLabels = () =>
  screen
    .getAllByRole('link')
    .map((a) => a.textContent)
    .filter((t) => t !== 'ServicePro');

describe('Sidebar (T269)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // The permission list never arrives: the loading fallback applies.
    vi.mocked(tenantApi.getMyAccess).mockReturnValue(new Promise(() => {}));
    useSidebarStore.setState({ isCollapsed: false, isMobileOpen: false });
    useTenantStore.setState({ currentTenant: tenant, tenants: [tenant] });
    useMembershipStore.setState({
      currentMembership: teamPlan,
      loadMembership: vi.fn().mockResolvedValue(undefined),
    });
  });

  it('lists the built modules in the mockup order', () => {
    renderAt('/dashboard');
    expect(linkLabels()).toEqual([
      'Dashboard',
      'My day',
      'Customers & Sites',
      'Agreements',
      'Jobs',
      'Quotes',
      'Invoices',
      'Payments',
      'Team',
      'Settings',
    ]);
    expect(screen.getByRole('button', { name: /Reports/ })).toBeInTheDocument();
  });

  it('hides modules whose pages do not exist yet', () => {
    renderAt('/dashboard');
    for (const name of ['Schedule']) {
      expect(screen.queryByText(name)).not.toBeInTheDocument();
    }
  });

  it('marks the section of a detail page active', () => {
    renderAt('/customers/abc');
    expect(
      screen.getByRole('link', { name: 'Customers & Sites' })
    ).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('link', { name: 'Dashboard' })).not.toHaveAttribute(
      'aria-current'
    );
  });

  it('shows the plan badge from the membership', async () => {
    grant('owner', ['tenants.read', 'jobs.list', 'jobs.view_all']);
    renderAt('/dashboard');
    expect(await screen.findByText('Team plan')).toBeInTheDocument();
    expect(screen.getByText(/unlimited users/)).toBeInTheDocument();
  });

  // T272: no plan request until the permission list says tenants.read.
  it('does not load the plan while the permission list loads', () => {
    useMembershipStore.setState({ currentMembership: null });
    renderAt('/dashboard');
    expect(useMembershipStore.getState().loadMembership).not.toHaveBeenCalled();
  });

  it('loads the plan once tenants.read is confirmed', async () => {
    useMembershipStore.setState({ currentMembership: null });
    grant('owner', ['tenants.read', 'jobs.list', 'jobs.view_all']);
    renderAt('/dashboard');
    await waitFor(() =>
      expect(useMembershipStore.getState().loadMembership).toHaveBeenCalledWith(
        'tenant-1'
      )
    );
  });

  it('shows only Dashboard without an organization', () => {
    useTenantStore.setState({ currentTenant: null, tenants: [] });
    renderAt('/dashboard');
    expect(screen.getByRole('link', { name: 'Dashboard' })).toBeInTheDocument();
    expect(screen.queryByText('Invoices')).not.toBeInTheDocument();
    expect(screen.queryByText('Team plan')).not.toBeInTheDocument();
  });
});

describe('Sidebar for technicians (T035)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useSidebarStore.setState({ isCollapsed: false, isMobileOpen: false });
    useMembershipStore.setState({
      currentMembership: null,
      loadMembership: vi.fn().mockResolvedValue(undefined),
    });
  });

  // T173: technicians land on My day; the office dashboard isn't theirs.
  it('shows only My day and Jobs from the permission list', async () => {
    const tech = { ...tenant, role: 'technician' };
    useTenantStore.setState({ currentTenant: tech, tenants: [tech] });
    grant('technician', ['jobs.read', 'jobs.list', 'jobs.update']);
    renderAt('/my-day');

    await waitFor(() => expect(tenantApi.getMyAccess).toHaveBeenCalled());
    expect(linkLabels()).toEqual(['My day', 'Jobs']);
    expect(screen.getByRole('link', { name: 'My day' })).toHaveAttribute(
      'aria-current',
      'page'
    );
    expect(
      screen.queryByRole('button', { name: /Reports/ })
    ).not.toBeInTheDocument();
  });

  // T272: technicians lack tenants.read, so GET /tenants/:id/membership 403s.
  it('never requests or shows the plan', async () => {
    const tech = { ...tenant, role: 'technician' };
    useTenantStore.setState({ currentTenant: tech, tenants: [tech] });
    useMembershipStore.setState({ currentMembership: teamPlan });
    grant('technician', ['jobs.read', 'jobs.list', 'jobs.update']);
    renderAt('/my-day');

    await waitFor(() => expect(linkLabels()).toEqual(['My day', 'Jobs']));
    expect(useMembershipStore.getState().loadMembership).not.toHaveBeenCalled();
    expect(screen.queryByText('Team plan')).not.toBeInTheDocument();
  });

  it('never flashes money or admin items while the list loads', () => {
    vi.mocked(tenantApi.getMyAccess).mockReturnValue(new Promise(() => {}));
    const tech = { ...tenant, role: 'technician' };
    useTenantStore.setState({ currentTenant: tech, tenants: [tech] });
    renderAt('/dashboard');

    expect(linkLabels()).toEqual(['My day', 'Jobs']);
  });

  it('follows the permission list, not the role name', async () => {
    // A custom role that may see invoices but not reports or the team.
    const custom = { ...tenant, role: 'billing_clerk' };
    useTenantStore.setState({ currentTenant: custom, tenants: [custom] });
    grant('billing_clerk', ['jobs.list', 'invoices.list', 'invoices.read']);
    renderAt('/dashboard');

    await waitFor(() =>
      expect(linkLabels()).toEqual(['Dashboard', 'My day', 'Jobs', 'Invoices'])
    );
  });
});
