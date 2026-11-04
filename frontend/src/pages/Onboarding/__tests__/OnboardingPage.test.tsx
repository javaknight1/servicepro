import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import type { AxiosResponse } from 'axios';
import type { Tenant, UpdateTenantRequest } from '@/types/tenant';
import type { PaymentTerm } from '@services/paymentTermService';
import type { TaxRate, TaxRateInput } from '@services/taxRateService';
import type {
  PaymentsConnectResponse,
  PaymentsStatus,
} from '@/types/paymentsConnect';
import type { QuickBooksStatus } from '@/types/quickbooks';

const store = vi.hoisted(() => ({
  currentTenant: null as Tenant | null,
}));
const updateTenant =
  vi.fn<(id: string, data: UpdateTenantRequest) => Promise<Tenant>>();

vi.mock('@store', () => ({
  useTenantStore: () => ({
    currentTenant: store.currentTenant,
    updateTenant,
  }),
  useAuthStore: (select: (s: { user: { email: string } }) => unknown) =>
    select({ user: { email: 'dana@brightlinefs.com' } }),
}));

vi.mock('@/utils/tenantPermissions', () => ({
  useTenantPermissions: () => ({
    can: () => true,
    canManageMembers: true,
    canUpdateTenant: true,
    canChangePlan: true,
    canManageBilling: true,
    canDeleteTenant: false,
    canManageTaxRates: true,
    isLoading: false,
  }),
}));

const net30: PaymentTerm = {
  id: 'term-30',
  name: 'Net 30',
  term_type: 'net_30',
  days_until_due: 30,
  is_default: false,
  is_active: true,
};
const net15: PaymentTerm = {
  ...net30,
  id: 'term-15',
  name: 'Net 15',
  days_until_due: 15,
};
vi.mock('@services/paymentTermService', () => ({
  paymentTermService: {
    listPaymentTerms: () => Promise.resolve([net15, net30]),
  },
  paymentTermsQueryKey: ['payment-terms', 'active'],
}));

const createTaxRate = vi.fn<(input: TaxRateInput) => Promise<TaxRate>>();
vi.mock('@services/taxRateService', () => ({
  taxRateService: {
    listTaxRates: () => Promise.resolve([]),
    createTaxRate: (i: TaxRateInput) => createTaxRate(i),
  },
  taxRatesQueryKey: ['tax-rates', 'all'],
}));

const connect =
  vi.fn<
    (tenantId: string) => Promise<AxiosResponse<PaymentsConnectResponse>>
  >();
const getStatus =
  vi.fn<(tenantId: string) => Promise<AxiosResponse<PaymentsStatus>>>();
vi.mock('@services/paymentsService', () => ({
  paymentsApi: {
    getStatus: (id: string) => getStatus(id),
    connect: (id: string) => connect(id),
  },
  paymentsStatusQueryKey: (id: string) => ['payments-status', id],
}));

const getQuickBooksStatus =
  vi.fn<(tenantId: string) => Promise<AxiosResponse<QuickBooksStatus>>>();
vi.mock('@services/quickbooksService', () => ({
  quickbooksApi: {
    getStatus: (id: string) => getQuickBooksStatus(id),
  },
  quickbooksStatusQueryKey: (id: string) => ['quickbooks-status', id],
}));

function quickbooksStatus(
  overrides: Partial<QuickBooksStatus> = {}
): QuickBooksStatus {
  return {
    configured: false,
    status: 'disconnected',
    counts: { customers: 0, invoices: 0, payments: 0, pending: 0, errors: 0 },
    errors: [],
    recent: [],
    ...overrides,
  };
}

const redirectTo = vi.fn<(url: string) => void>();
vi.mock('@/utils/redirect', () => ({
  redirectTo: (url: string) => redirectTo(url),
}));

import { OnboardingPage } from '../OnboardingPage';

function tenantWith(settings: Tenant['settings']): Tenant {
  return {
    id: 'tenant-1',
    name: 'Brightline',
    slug: 'brightline',
    owner_id: 'owner-1',
    is_active: true,
    created_at: '2026-11-03T00:00:00Z',
    updated_at: '2026-11-03T00:00:00Z',
    settings,
  };
}

function ok<T>(data: T): AxiosResponse<T> {
  return { data } as unknown as AxiosResponse<T>;
}

function apiError(status: number, data: object) {
  return new AxiosError(
    String(status),
    'ERR_BAD_REQUEST',
    undefined,
    undefined,
    {
      status,
      data,
    } as unknown as AxiosResponse
  );
}

/** The last settings sent to the API. */
function lastSave(): UpdateTenantRequest {
  const calls = updateTenant.mock.calls;
  return calls[calls.length - 1][1];
}

function renderWizard(settings: Tenant['settings']) {
  store.currentTenant = tenantWith(settings);
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/onboarding']}>
        <Routes>
          <Route path="/onboarding" element={<OnboardingPage />} />
          <Route path="/dashboard" element={<p>Dashboard page</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

// T235: onboarding wizard (mockups 02-04).
describe('OnboardingPage', () => {
  beforeEach(() => {
    updateTenant.mockReset();
    // Like the store: the saved tenant becomes the current one.
    updateTenant.mockImplementation((_id, data) => {
      const current = store.currentTenant as Tenant;
      store.currentTenant = {
        ...current,
        name: data.name ?? current.name,
        settings: data.settings ?? current.settings,
      };
      return Promise.resolve(store.currentTenant);
    });
    createTaxRate.mockReset();
    connect.mockReset();
    getStatus.mockReset();
    redirectTo.mockReset();
    getQuickBooksStatus.mockReset();
    getQuickBooksStatus.mockResolvedValue(ok(quickbooksStatus()));
    getStatus.mockResolvedValue(
      ok({
        connected: false,
        state: 'not_connected',
        online_payments_enabled: false,
        pay_links_active: false,
        card_payments_status: '',
        ach_payments_status: '',
        payouts_status: '',
        requirements_status: '',
        livemode: false,
        fees: {
          card_percent: '0.4',
          ach_percent: '1',
          ach_cap: '10.00',
          currency: 'USD',
        },
      })
    );
  });

  it('step 1 saves the business and moves to step 2', async () => {
    renderWizard({ onboardingStep: 1, timezone: 'America/Chicago' });
    const user = userEvent.setup();

    expect(
      screen.getByRole('heading', { name: 'Your business' })
    ).toBeInTheDocument();
    expect(screen.getByText(/dana@brightlinefs.com/)).toBeInTheDocument();
    const name = screen.getByLabelText('Business name');
    await user.clear(name);
    await user.type(name, 'Brightline Facility Services');
    await user.click(screen.getByLabelText(/Commercial cleaning/));
    await user.click(screen.getByRole('button', { name: 'More technicians' }));
    await user.type(screen.getByLabelText('City'), 'Columbus');
    await user.type(screen.getByLabelText('State'), 'OH');
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(
      await screen.findByRole('heading', { name: 'How you bill' })
    ).toBeInTheDocument();
    expect(lastSave()).toEqual({
      name: 'Brightline Facility Services',
      settings: {
        onboardingStep: 2,
        timezone: 'America/Chicago',
        trade: 'commercial_cleaning',
        technicianCount: 2,
        businessCity: 'Columbus',
        businessState: 'OH',
      },
    });
  });

  it('skipping a step keeps only the resume point', async () => {
    renderWizard({ onboardingStep: 1 });
    const user = userEvent.setup();

    await user.click(screen.getByRole('button', { name: 'Skip this step' }));

    expect(
      await screen.findByRole('heading', { name: 'How you bill' })
    ).toBeInTheDocument();
    expect(lastSave()).toEqual({ settings: { onboardingStep: 2 } });
  });

  it('resumes at the saved step', () => {
    renderWizard({ onboardingStep: 3 });
    expect(
      screen.getByRole('heading', { name: 'Get paid' })
    ).toBeInTheDocument();
  });

  it('step 2 creates a flat tax rate and saves the billing defaults', async () => {
    createTaxRate.mockResolvedValue({
      id: 'rate-new',
      tenant_id: 'tenant-1',
      name: 'Ohio',
      rate: '0.08',
      tax_type: 'sales_tax',
      is_active: true,
    });
    renderWizard({ onboardingStep: 2 });
    const user = userEvent.setup();

    // Net 30 is preselected once the terms load.
    await waitFor(() =>
      expect(screen.getByLabelText('Default payment terms')).toHaveValue(
        'term-30'
      )
    );
    expect(screen.getByLabelText(/Consolidated monthly/)).toBeChecked();
    await user.selectOptions(screen.getByLabelText('Sales tax'), '__new__');
    const rateName = screen.getByLabelText('Rate name');
    await user.clear(rateName);
    await user.type(rateName, 'Ohio');
    await user.type(screen.getByLabelText('Rate (%)'), '8');
    await user.type(screen.getByLabelText('Invoice numbering'), 'BF-');
    await user.click(
      screen.getByRole('switch', { name: 'Require PO numbers' })
    );
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(
      await screen.findByRole('heading', { name: 'Get paid' })
    ).toBeInTheDocument();
    expect(createTaxRate).toHaveBeenCalledWith({
      name: 'Ohio',
      rate: 0.08,
      is_active: true,
    });
    expect(lastSave().settings).toEqual({
      onboardingStep: 3,
      defaultConsolidatedBilling: true,
      defaultPaymentTermId: 'term-30',
      defaultTaxRateId: 'rate-new',
      invoicePrefix: 'BF-',
      defaultPoRequired: true,
    });
  });

  it('step 2 shows the API field errors inline', async () => {
    renderWizard({ onboardingStep: 2 });
    const user = userEvent.setup();
    updateTenant.mockRejectedValueOnce(
      apiError(400, {
        error: 'validation_failed',
        message: 'Some settings are not valid',
        fields: { defaultPaymentTermId: 'was not found' },
      })
    );

    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(await screen.findByText('was not found')).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'How you bill' })
    ).toBeInTheDocument();
  });

  it('step 3 shows a Connect failure and can still be skipped', async () => {
    connect.mockRejectedValue(
      apiError(409, {
        error: 'connect_unavailable',
        message: 'Stripe Connect is not enabled on this platform',
      })
    );
    renderWizard({ onboardingStep: 3 });
    const user = userEvent.setup();

    expect(await screen.findByText('Stripe + 0.4%')).toBeInTheDocument();
    // Honesty gate: nothing unbuilt is offered as if it worked. Imports
    // are built (T193) and link to the import page; QuickBooks isn't.
    expect(screen.getAllByText('Coming soon')).toHaveLength(1);
    expect(
      screen.getByRole('link', { name: /Import from Jobber/ })
    ).toHaveAttribute('href', '/settings/import?source=jobber');
    expect(screen.getByRole('link', { name: /Upload a CSV/ })).toHaveAttribute(
      'href',
      '/settings/import?source=generic'
    );
    expect(
      screen.getByRole('button', { name: 'Connect QuickBooks' })
    ).toBeDisabled();

    await user.click(
      screen.getByRole('button', { name: 'Connect payouts with Stripe' })
    );
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Stripe Connect is not enabled on this platform'
    );
    expect(redirectTo).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'Skip for now' }));
    expect(await screen.findByText('Dashboard page')).toBeInTheDocument();
    expect(lastSave().settings?.onboardingStep).toBe(3);
    expect(lastSave().settings?.onboardingCompletedAt).toEqual(
      expect.any(String)
    );
  });

  it('step 3 links QuickBooks to Settings once the server offers it (T192)', async () => {
    getQuickBooksStatus.mockResolvedValue(
      ok(quickbooksStatus({ configured: true }))
    );
    renderWizard({ onboardingStep: 3 });

    const link = await screen.findByRole('link', {
      name: 'Connect QuickBooks',
    });
    expect(link).toHaveAttribute(
      'href',
      '/settings/organization?tab=integrations'
    );
    expect(screen.queryByText('Coming soon')).not.toBeInTheDocument();
  });

  it('step 3 sends the owner to Stripe onboarding', async () => {
    connect.mockResolvedValue(ok({ url: 'https://connect.stripe.com/x' }));
    renderWizard({ onboardingStep: 3 });
    const user = userEvent.setup();

    await user.click(
      screen.getByRole('button', { name: 'Connect payouts with Stripe' })
    );
    await waitFor(() =>
      expect(redirectTo).toHaveBeenCalledWith('https://connect.stripe.com/x')
    );
  });

  it('finish setup marks onboarding complete', async () => {
    renderWizard({ onboardingStep: 3 });
    const user = userEvent.setup();

    await user.click(screen.getByRole('button', { name: 'Finish setup' }));

    expect(await screen.findByText('Dashboard page')).toBeInTheDocument();
    expect(lastSave().settings?.onboardingCompletedAt).toEqual(
      expect.any(String)
    );
  });

  it('save and finish later keeps it resumable', async () => {
    renderWizard({ onboardingStep: 2 });
    const user = userEvent.setup();

    await user.click(
      screen.getByRole('button', { name: 'Save and finish later' })
    );

    expect(await screen.findByText('Dashboard page')).toBeInTheDocument();
    expect(lastSave()).toEqual({ settings: { onboardingStep: 2 } });
  });
});
