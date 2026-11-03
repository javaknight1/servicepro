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

const updateTenant = vi.hoisted(() => vi.fn());

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
    updateTenant,
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
    canManageTaxRates: false,
    isLoading: false,
  }),
}));

import { OrgSettingsPage } from '../OrgSettingsPage';

// T259: operating hours are hidden until something reads them (T099); the
// tab keeps only the timezone, which does drive due dates and reminders.
describe('OrgSettingsPage time zone', () => {
  it('shows the timezone and no operating-hours grid', async () => {
    render(
      <MemoryRouter>
        <OrgSettingsPage />
      </MemoryRouter>
    );

    expect(
      screen.queryByRole('button', { name: /business hours/i })
    ).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: /time zone/i }));

    expect(screen.getByText(/used for invoice due dates/i)).toBeInTheDocument();
    expect(screen.getByLabelText('Timezone')).toHaveValue('America/Chicago');
    expect(screen.queryByText('Coming soon')).not.toBeInTheDocument();
    expect(screen.queryByText(/operating hours/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/monday/i)).not.toBeInTheDocument();
  });
});

// T170: the business's city and state appear on the customer pay page; the
// rest of the settings are kept when they are saved.
describe('OrgSettingsPage business location', () => {
  it('saves city and state into the settings without dropping others', async () => {
    render(
      <MemoryRouter>
        <OrgSettingsPage />
      </MemoryRouter>
    );

    await userEvent.type(screen.getByLabelText('City'), 'Columbus');
    await userEvent.type(screen.getByLabelText('State'), 'OH');
    await userEvent.click(
      screen.getByRole('button', { name: /save changes/i })
    );

    expect(updateTenant).toHaveBeenCalledWith(
      'tenant-1',
      expect.objectContaining({
        settings: {
          timezone: 'America/Chicago',
          businessCity: 'Columbus',
          businessState: 'OH',
        },
      })
    );
  });
});
