import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router';
import type { ReactNode } from 'react';
import type { CreateTenantRequest, Tenant } from '@/types/tenant';

vi.mock('@components/layout', () => ({
  DashboardLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

const createTenant = vi.fn<(data: CreateTenantRequest) => Promise<Tenant>>();
const setCurrentTenant = vi.fn<(tenant: Tenant) => void>();
vi.mock('@store', () => ({
  useTenantStore: () => ({
    createTenant,
    setCurrentTenant,
    isLoading: false,
    error: null,
    clearError: vi.fn(),
  }),
}));

import { NewOrgPage } from '../NewOrgPage';

// T235: a new organization goes straight into the onboarding wizard.
describe('NewOrgPage', () => {
  it('creates the organization with onboarding started and opens the wizard', async () => {
    const created = { id: 'tenant-9', name: 'Brightline' } as Tenant;
    createTenant.mockResolvedValue(created);
    render(
      <MemoryRouter initialEntries={['/settings/organization/new']}>
        <Routes>
          <Route path="/settings/organization/new" element={<NewOrgPage />} />
          <Route path="/onboarding" element={<p>Wizard</p>} />
        </Routes>
      </MemoryRouter>
    );
    const user = userEvent.setup();

    await user.type(screen.getByLabelText('Organization Name'), 'Brightline');
    await user.click(
      screen.getByRole('button', { name: 'Create Organization' })
    );

    expect(await screen.findByText('Wizard')).toBeInTheDocument();
    expect(setCurrentTenant).toHaveBeenCalledWith(created);
    const body = createTenant.mock.calls[0][0];
    expect(body.slug).toBe('brightline');
    expect(body.settings?.onboardingStep).toBe(1);
    expect(body.settings?.timezone).toEqual(expect.any(String));
  });
});
