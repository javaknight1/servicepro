import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import type { Tenant, TenantMember } from '@/types/tenant';

vi.mock('@components/layout', () => ({
  DashboardLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

const getMembers = vi.fn();
let currentTenant: Tenant | null = null;
vi.mock('@store', () => ({
  useTenantStore: () => ({
    currentTenant,
    getMembers,
    removeMember: vi.fn(),
    updateMemberRole: vi.fn(),
  }),
}));

vi.mock('@services/tenantService', () => ({
  tenantApi: {
    getMyAccess: vi.fn(),
    getPendingInvitations: vi.fn(),
    inviteMember: vi.fn(),
    cancelInvitation: vi.fn(),
    resendInvitation: vi.fn(),
  },
}));

vi.mock('@services/roleApi', () => ({
  roleApi: { getRoles: vi.fn() },
}));

import { tenantApi } from '@services/tenantService';
import { roleApi } from '@services/roleApi';
import { TeamMembersPage } from '../TeamMembersPage';

const tenant = (role?: string): Tenant => ({
  id: 'tenant-1',
  name: 'Acme HVAC',
  slug: 'acme',
  owner_id: 'owner-1',
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  role,
});

const member: TenantMember = {
  id: 'm-1',
  user_id: 'u-1',
  email: 'tech@example.com',
  role_id: 'r-user',
  role_name: 'user',
  is_active: true,
  joined_at: '2026-01-01T00:00:00Z',
};

const renderPage = () =>
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <MemoryRouter>
        <TeamMembersPage />
      </MemoryRouter>
    </QueryClientProvider>
  );

/** GET /tenants/:id/me answers with these permissions. */
const grant = (role: string, permissions: string[]) =>
  vi.mocked(tenantApi.getMyAccess).mockResolvedValue({
    data: {
      tenant_id: 'tenant-1',
      role_id: `r-${role}`,
      role,
      role_level: 50,
      permissions,
    },
  } as unknown as Awaited<ReturnType<typeof tenantApi.getMyAccess>>);

describe('TeamMembersPage permissions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(tenantApi.getPendingInvitations).mockResolvedValue({
      data: [],
    } as unknown as Awaited<
      ReturnType<typeof tenantApi.getPendingInvitations>
    >);
    vi.mocked(roleApi.getRoles).mockResolvedValue({
      data: [],
    } as unknown as Awaited<ReturnType<typeof roleApi.getRoles>>);
  });

  it('shows management actions to admins', async () => {
    currentTenant = tenant('admin');
    grant('admin', ['tenants.read', 'tenants.manage']);
    getMembers.mockResolvedValue([member]);
    renderPage();

    expect(await screen.findByText('tech@example.com')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /invite member/i })
    ).toBeInTheDocument();
    expect(screen.getByTitle('Remove member')).toBeInTheDocument();
    expect(tenantApi.getPendingInvitations).toHaveBeenCalledWith('tenant-1');
  });

  it('hides management actions from roles without tenants.manage', async () => {
    currentTenant = tenant('user');
    grant('user', ['tenants.read']);
    getMembers.mockResolvedValue([member]);
    renderPage();

    expect(await screen.findByText('tech@example.com')).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /invite member/i })
    ).not.toBeInTheDocument();
    expect(screen.queryByTitle('Remove member')).not.toBeInTheDocument();
    expect(screen.queryByTitle('Change role')).not.toBeInTheDocument();
    // Invitations need tenants.manage; don't request them just to get a 403
    expect(tenantApi.getPendingInvitations).not.toHaveBeenCalled();
  });

  it('shows management actions to a custom role granted tenants.manage', async () => {
    currentTenant = tenant('office_lead');
    grant('office_lead', ['tenants.read', 'tenants.manage']);
    getMembers.mockResolvedValue([member]);
    renderPage();

    expect(await screen.findByText('tech@example.com')).toBeInTheDocument();
    expect(
      await screen.findByRole('button', { name: /invite member/i })
    ).toBeInTheDocument();
    expect(tenantApi.getMyAccess).toHaveBeenCalledWith('tenant-1');
  });

  it('follows the permission list over the role name', async () => {
    // Named "admin" but the API grants no tenants.manage in this tenant.
    currentTenant = tenant('admin');
    grant('admin', ['tenants.read']);
    getMembers.mockResolvedValue([member]);
    renderPage();

    expect(await screen.findByText('tech@example.com')).toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.queryByRole('button', { name: /invite member/i })
      ).not.toBeInTheDocument()
    );
    expect(screen.queryByTitle('Remove member')).not.toBeInTheDocument();
  });

  it('renders a permission message when the API answers 403', async () => {
    currentTenant = tenant();
    vi.mocked(tenantApi.getMyAccess).mockRejectedValue({
      response: { status: 403, data: { error: 'tenant_access_denied' } },
    });
    getMembers.mockRejectedValue({
      response: {
        status: 403,
        data: {
          error: 'tenant_access_denied',
          message: 'You do not have access to this organization',
        },
      },
    });
    renderPage();

    expect(
      await screen.findByText("You don't have access to this organization.")
    ).toBeInTheDocument();
    expect(screen.queryByText(/no members found/i)).not.toBeInTheDocument();
  });
});
