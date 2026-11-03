import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
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
  useAuthStore: (select: (s: { user: { id: string } }) => unknown) =>
    select({ user: { id: 'owner-1' } }),
  useTenantStore: () => ({
    currentTenant,
    getMembers,
    removeMember: vi.fn(),
    updateMemberRole,
  }),
}));
const updateMemberRole = vi.fn();

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
      screen.getByRole('button', { name: /invite teammate/i })
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
      screen.queryByRole('button', { name: /invite teammate/i })
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
      await screen.findByRole('button', { name: /invite teammate/i })
    ).toBeInTheDocument();
    expect(tenantApi.getMyAccess).toHaveBeenCalledWith('tenant-1');
  });

  it('follows the permission list over the role name', async () => {
    // Named "admin" but the API grants no tenants.manage in this tenant.
    currentTenant = tenant('admin');
    grant('admin', ['tenants.read']);
    getMembers.mockResolvedValue([member]);
    renderPage();

    // The role-name fallback briefly allows management, so the list loads
    // twice (React 19 commits the first load before the reload starts):
    // wait for the settled state rather than the first match.
    await waitFor(() =>
      expect(
        screen.queryByRole('button', { name: /invite teammate/i })
      ).not.toBeInTheDocument()
    );
    expect(await screen.findByText('tech@example.com')).toBeInTheDocument();
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

const role = (id: string, name: string, level: number) => ({
  id,
  name,
  description: `${name} role`,
  hierarchy_level: level,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
});

const allRoles = [
  role('r-super', 'super_admin', 100),
  role('r-admin', 'admin', 80),
  role('r-manager', 'manager', 60),
  role('r-user', 'user', 40),
  role('r-tech', 'technician', 30),
  role('r-guest', 'guest', 20),
];

const ownerMember: TenantMember = {
  ...member,
  id: 'm-owner',
  user_id: 'owner-1',
  email: 'dana@example.com',
  role_id: 'r-admin',
  role_name: 'admin',
};

const techMember: TenantMember = {
  ...member,
  id: 'm-tech',
  user_id: 'u-tech',
  email: 'marcus@example.com',
  role_id: 'r-tech',
  role_name: 'technician',
};

describe('TeamMembersPage roles (T035)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    currentTenant = tenant('admin');
    grant('admin', ['tenants.read', 'tenants.manage']);
    getMembers.mockResolvedValue([ownerMember, techMember]);
    vi.mocked(tenantApi.getPendingInvitations).mockResolvedValue({
      data: [],
    } as unknown as Awaited<
      ReturnType<typeof tenantApi.getPendingInvitations>
    >);
    vi.mocked(roleApi.getRoles).mockResolvedValue({
      data: allRoles,
    } as unknown as Awaited<ReturnType<typeof roleApi.getRoles>>);
    vi.mocked(tenantApi.inviteMember).mockResolvedValue(
      {} as unknown as Awaited<ReturnType<typeof tenantApi.inviteMember>>
    );
  });

  it('labels members Owner / Technician and shows what each role can see', async () => {
    renderPage();
    expect(await screen.findByText('marcus@example.com')).toBeInTheDocument();
    const memberRow = (email: string) => {
      const row = screen.getByText(email).closest<HTMLElement>('div.ml-4');
      if (!row) throw new Error(`no row for ${email}`);
      return row;
    };
    expect(
      within(memberRow('dana@example.com')).getByText('Owner')
    ).toBeInTheDocument();
    expect(
      within(memberRow('marcus@example.com')).getByText('Technician')
    ).toBeInTheDocument();
    expect(screen.getByText('What each role can see')).toBeInTheDocument();
    expect(screen.getByText('Technicians never see money')).toBeInTheDocument();
  });

  it('invites a technician from the three role pills', async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(
      await screen.findByRole('button', { name: /invite teammate/i })
    );
    const pills = await screen.findByRole('radiogroup', { name: 'Role' });
    expect(
      within(pills)
        .getAllByRole('radio')
        .map((r) => r.textContent)
    ).toEqual(['Owner', 'Office manager', 'Technician']);

    await user.type(screen.getByLabelText(/email address/i), 'new@example.com');
    await user.click(within(pills).getByRole('radio', { name: 'Technician' }));
    expect(
      within(pills).getByRole('radio', { name: 'Technician' })
    ).toHaveAttribute('aria-checked', 'true');
    expect(screen.getByText(/never sees prices/i)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /send invitation/i }));

    await waitFor(() =>
      expect(tenantApi.inviteMember).toHaveBeenCalledWith('tenant-1', {
        email: 'new@example.com',
        role_id: 'r-tech',
      })
    );
  });

  it('sets a member as Technician, but never offers a change of your own role', async () => {
    const user = userEvent.setup();
    getMembers.mockResolvedValue([
      ownerMember,
      { ...member, role_id: 'r-manager', role_name: 'manager' },
    ]);
    updateMemberRole.mockResolvedValue(undefined);
    renderPage();

    await screen.findByText('tech@example.com');
    // Only the other member has a Change role button (the caller is owner-1).
    const changeButtons = await screen.findAllByTitle('Change role');
    expect(changeButtons).toHaveLength(1);
    await user.click(changeButtons[0]);

    const pills = await screen.findByRole('radiogroup', { name: 'New role' });
    expect(
      within(pills).getByRole('radio', { name: 'Office manager' })
    ).toHaveAttribute('aria-checked', 'true');
    await user.click(within(pills).getByRole('radio', { name: 'Technician' }));
    await user.click(screen.getByRole('button', { name: /update role/i }));

    await waitFor(() =>
      expect(updateMemberRole).toHaveBeenCalledWith('tenant-1', 'u-1', 'r-tech')
    );
  });
});
