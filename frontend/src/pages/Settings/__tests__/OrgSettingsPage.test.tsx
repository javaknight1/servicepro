import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import type { ReactNode } from 'react';
import type { Tenant } from '@/types/tenant';

vi.mock('@components/layout', () => ({
  DashboardLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock('@components/membership', () => ({
  MembershipTab: () => <div />,
}));

const currentTenant: Tenant = {
  id: 'tenant-1',
  name: 'Acme HVAC',
  slug: 'acme',
  owner_id: 'owner-1',
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  settings: { timezone: 'America/Chicago' },
};

vi.mock('@store', () => ({
  useTenantStore: () => ({
    currentTenant,
    isLoading: false,
    error: null,
    updateTenant: vi.fn(),
    deleteTenant: vi.fn(),
    getMembers: vi.fn(),
    removeMember: vi.fn(),
    clearError: vi.fn(),
  }),
}));

vi.mock('@services/tenantService', () => ({ tenantApi: {} }));
vi.mock('@services/roleApi', () => ({ roleApi: { getRoles: vi.fn() } }));
vi.mock('@services/paymentReminderService', () => ({
  paymentReminderService: { getReminderSettings: vi.fn() },
}));
vi.mock('@/utils/tenantPermissions', () => ({
  useTenantPermissions: () => ({
    can: () => true,
    canManageMembers: false,
    canUpdateTenant: true,
    canChangePlan: false,
    canManageBilling: false,
    canDeleteTenant: false,
    isLoading: false,
  }),
}));

import { OrgSettingsPage } from '../OrgSettingsPage';

// T127: business hours are saved but no backend code reads them yet (T099),
// so the page says so instead of implying they take effect.
describe('OrgSettingsPage business hours', () => {
  it('labels operating hours as coming soon and explains the timezone', async () => {
    render(
      <MemoryRouter>
        <OrgSettingsPage />
      </MemoryRouter>
    );

    await userEvent.click(
      screen.getByRole('button', { name: /business hours/i })
    );

    expect(screen.getByText('Coming soon')).toBeInTheDocument();
    expect(
      screen.getByText(/aren't checked against these hours yet/i)
    ).toBeInTheDocument();
    expect(screen.getByText(/used for invoice due dates/i)).toBeInTheDocument();
  });
});
