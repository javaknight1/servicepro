import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { Sidebar } from '../Sidebar';
import { useTenantStore } from '@store/tenantStore';
import { useMembershipStore } from '@store/membershipStore';
import { useSidebarStore } from '@store/sidebarStore';
import type { Tenant } from '@/types/tenant';
import type { TenantMembership } from '@/types/membership';

vi.mock('@components/tenants', () => ({
  TenantSwitcher: () => <div>Tenant switcher</div>,
}));

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

const renderAt = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Sidebar />
    </MemoryRouter>
  );

describe('Sidebar (T269)', () => {
  beforeEach(() => {
    useSidebarStore.setState({ isCollapsed: false, isMobileOpen: false });
    useTenantStore.setState({ currentTenant: tenant, tenants: [tenant] });
    useMembershipStore.setState({
      currentMembership: teamPlan,
      loadMembership: vi.fn().mockResolvedValue(undefined),
    });
  });

  it('lists the built modules in the mockup order', () => {
    renderAt('/dashboard');
    const labels = screen
      .getAllByRole('link')
      .map((a) => a.textContent)
      .filter((t) => t !== 'ServicePro');
    expect(labels).toEqual([
      'Dashboard',
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

  it('shows the plan badge from the membership', () => {
    renderAt('/dashboard');
    expect(screen.getByText('Team plan')).toBeInTheDocument();
    expect(screen.getByText(/unlimited users/)).toBeInTheDocument();
  });

  it('shows only Dashboard without an organization', () => {
    useTenantStore.setState({ currentTenant: null, tenants: [] });
    renderAt('/dashboard');
    expect(screen.getByRole('link', { name: 'Dashboard' })).toBeInTheDocument();
    expect(screen.queryByText('Invoices')).not.toBeInTheDocument();
    expect(screen.queryByText('Team plan')).not.toBeInTheDocument();
  });
});
