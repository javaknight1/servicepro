import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { AxiosError } from 'axios';
import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import type { Tenant } from '@/types/tenant';
import type {
  PaymentsConnectResponse,
  PaymentsStatus,
} from '@/types/paymentsConnect';

vi.mock('@components/layout', () => ({
  DashboardLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

const currentTenant: Tenant = {
  id: 'tenant-1',
  name: 'Brightline Facility Services',
  slug: 'brightline',
  owner_id: 'owner-1',
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  settings: {},
};

vi.mock('@store', () => ({
  useTenantStore: () => ({ currentTenant }),
}));

const permissions = { canManageBilling: true, isLoading: false };
vi.mock('@/utils/tenantPermissions', () => ({
  useTenantPermissions: () => ({
    can: () => permissions.canManageBilling,
    canManageMembers: false,
    canUpdateTenant: false,
    canChangePlan: false,
    canManageBilling: permissions.canManageBilling,
    canDeleteTenant: false,
    isLoading: permissions.isLoading,
  }),
}));

const getStatus =
  vi.fn<(tenantId: string) => Promise<AxiosResponse<PaymentsStatus>>>();
const connect =
  vi.fn<
    (tenantId: string) => Promise<AxiosResponse<PaymentsConnectResponse>>
  >();
vi.mock('@services/paymentsService', () => ({
  paymentsApi: {
    getStatus: (id: string) => getStatus(id),
    connect: (id: string) => connect(id),
  },
  paymentsStatusQueryKey: (id: string) => ['payments-status', id],
}));

const redirectTo = vi.fn<(url: string) => void>();
vi.mock('@/utils/redirect', () => ({
  redirectTo: (url: string) => redirectTo(url),
}));

import { PaymentsSettingsPage } from '../PaymentsSettingsPage';

const fees = {
  card_percent: '0.4',
  ach_percent: '1',
  ach_cap: '10.00',
  currency: 'USD',
};

function status(overrides: Partial<PaymentsStatus> = {}): PaymentsStatus {
  return {
    connected: false,
    state: 'not_connected',
    online_payments_enabled: true,
    pay_links_active: false,
    card_payments_status: '',
    ach_payments_status: '',
    payouts_status: '',
    requirements_status: '',
    livemode: false,
    fees,
    ...overrides,
  };
}

function ok<T>(data: T): AxiosResponse<T> {
  return { data } as unknown as AxiosResponse<T>;
}

function renderPage(route = '/settings/payments') {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <PaymentsSettingsPage />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

// T169: Settings -> Payments (mockups 04 and 12).
describe('PaymentsSettingsPage', () => {
  beforeEach(() => {
    getStatus.mockReset();
    connect.mockReset();
    redirectTo.mockReset();
    permissions.canManageBilling = true;
    permissions.isLoading = false;
  });

  it('offers to connect Stripe and explains the fees', async () => {
    getStatus.mockResolvedValue(ok(status()));
    renderPage();

    expect(await screen.findByText('Not connected')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Connect payouts with Stripe' })
    ).toBeInTheDocument();
    expect(
      screen.getByText('A $5,000 invoice paid by bank costs you $10.')
    ).toBeInTheDocument();
    expect(screen.getByText('Stripe + 0.4%')).toBeInTheDocument();
    expect(getStatus).toHaveBeenCalledWith('tenant-1');
  });

  it('sends the owner to the Stripe onboarding link', async () => {
    getStatus.mockResolvedValue(ok(status()));
    connect.mockResolvedValue(
      ok({ url: 'https://connect.stripe.com/setup/abc' })
    );
    renderPage();

    await userEvent.click(
      await screen.findByRole('button', { name: 'Connect payouts with Stripe' })
    );

    await waitFor(() =>
      expect(redirectTo).toHaveBeenCalledWith(
        'https://connect.stripe.com/setup/abc'
      )
    );
    expect(connect).toHaveBeenCalledWith('tenant-1');
  });

  it('shows capability statuses and a continue button while onboarding', async () => {
    getStatus.mockResolvedValue(
      ok(
        status({
          connected: true,
          state: 'onboarding',
          card_payments_status: 'pending',
          ach_payments_status: 'pending',
          requirements_status: 'currently_due',
        })
      )
    );
    renderPage('/settings/payments?onboarding=return');

    expect(await screen.findByText('Setup in progress')).toBeInTheDocument();
    expect(screen.getByText('Card payments')).toBeInTheDocument();
    expect(screen.getByText('Bank payments (ACH)')).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveTextContent(
      /stripe is checking your details/i
    );
    expect(
      screen.getByRole('button', { name: 'Continue setup with Stripe' })
    ).toBeInTheDocument();
  });

  it('shows a connected account that customers can pay', async () => {
    getStatus.mockResolvedValue(
      ok(
        status({
          connected: true,
          state: 'active',
          pay_links_active: true,
          card_payments_status: 'active',
          ach_payments_status: 'active',
          payouts_status: 'active',
        })
      )
    );
    renderPage();

    expect(await screen.findByText('Connected')).toBeInTheDocument();
    expect(
      screen.getByText(/customers can pay your invoices online/i)
    ).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: /open stripe dashboard/i })
    ).toHaveAttribute('href', 'https://dashboard.stripe.com');
  });

  it('says when online payment is switched off platform-wide', async () => {
    getStatus.mockResolvedValue(
      ok(
        status({
          connected: true,
          state: 'active',
          online_payments_enabled: false,
          card_payments_status: 'active',
        })
      )
    );
    renderPage();

    expect(
      await screen.findByText(/online invoice payment isn.t switched on yet/i)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/customers can pay your invoices online/i)
    ).not.toBeInTheDocument();
  });

  it('makes a fresh link when Stripe sends the user back to refresh', async () => {
    getStatus.mockResolvedValue(
      ok(status({ connected: true, state: 'onboarding' }))
    );
    connect.mockResolvedValue(
      ok({ url: 'https://connect.stripe.com/setup/new' })
    );
    renderPage('/settings/payments?onboarding=refresh');

    await waitFor(() =>
      expect(redirectTo).toHaveBeenCalledWith(
        'https://connect.stripe.com/setup/new'
      )
    );
    expect(connect).toHaveBeenCalledTimes(1);
  });

  it('shows the API error when Stripe setup is unavailable', async () => {
    getStatus.mockResolvedValue(ok(status()));
    connect.mockRejectedValue(
      new AxiosError(
        'Request failed',
        'ERR_BAD_RESPONSE',
        undefined,
        undefined,
        {
          status: 503,
          statusText: 'Service Unavailable',
          headers: {},
          config: {} as InternalAxiosRequestConfig,
          data: {
            error: 'connect_unavailable',
            message:
              "Connecting to Stripe isn't available yet. Please try again later",
          },
        }
      )
    );
    renderPage();

    await userEvent.click(
      await screen.findByRole('button', { name: 'Connect payouts with Stripe' })
    );
    expect(await screen.findByRole('alert')).toHaveTextContent(
      /connecting to stripe isn.t available yet/i
    );
    expect(redirectTo).not.toHaveBeenCalled();
  });

  it('hides payouts from members without billing permission', async () => {
    permissions.canManageBilling = false;
    renderPage();

    expect(
      screen.getByText(/only owners and admins can see payouts/i)
    ).toBeInTheDocument();
    expect(getStatus).not.toHaveBeenCalled();
  });
});
