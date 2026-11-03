import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { Tenant, UpdateTenantRequest } from '@/types/tenant';
import type { PaymentTerm } from '@services/paymentTermService';
import type { TaxRate } from '@services/taxRateService';

const updateTenant =
  vi.fn<(id: string, data: UpdateTenantRequest) => Promise<Tenant>>();
vi.mock('@store', () => ({
  useTenantStore: () => ({ updateTenant }),
}));

const listPaymentTerms = vi.fn<() => Promise<PaymentTerm[]>>();
vi.mock('@services/paymentTermService', () => ({
  paymentTermService: { listPaymentTerms: () => listPaymentTerms() },
  paymentTermsQueryKey: ['payment-terms', 'active'],
}));
const listTaxRates = vi.fn<(activeOnly: boolean) => Promise<TaxRate[]>>();
vi.mock('@services/taxRateService', () => ({
  taxRateService: { listTaxRates: (a: boolean) => listTaxRates(a) },
  taxRatesQueryKey: ['tax-rates', 'all'],
}));

import { BillingDefaultsCard } from '../BillingDefaultsCard';

const tenant: Tenant = {
  id: 'tenant-1',
  name: 'Brightline Facility Services',
  slug: 'brightline',
  owner_id: 'owner-1',
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  settings: { timezone: 'America/New_York', businessCity: 'Columbus' },
};

const net30: PaymentTerm = {
  id: 'term-30',
  name: 'Net 30',
  term_type: 'net_30',
  days_until_due: 30,
  is_default: false,
  is_active: true,
};
const ohio: TaxRate = {
  id: 'rate-oh',
  tenant_id: 'tenant-1',
  name: 'Ohio',
  rate: '0.08',
  tax_type: 'sales_tax',
  is_active: true,
};

/** An axios-shaped 400 with the API's field errors. */
function validationError(fields: Record<string, string>) {
  return Object.assign(new Error('Request failed with status code 400'), {
    response: {
      status: 400,
      data: { error: 'validation_failed', message: 'Invalid', fields },
    },
  });
}

function renderCard(canEdit = true) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <BillingDefaultsCard tenant={tenant} canEdit={canEdit} />
    </QueryClientProvider>
  );
}

// T235: Settings -> Billing defaults.
describe('BillingDefaultsCard', () => {
  beforeEach(() => {
    updateTenant.mockReset();
    listPaymentTerms.mockResolvedValue([net30]);
    listTaxRates.mockResolvedValue([ohio]);
  });

  it('saves currency, prefixes and defaults merged into the settings', async () => {
    updateTenant.mockResolvedValue(tenant);
    renderCard();
    const user = userEvent.setup();

    await screen.findByRole('option', { name: 'Net 30' });
    await screen.findByRole('option', { name: 'Ohio · 8%' });
    await user.selectOptions(screen.getByLabelText('Currency'), 'CAD');
    await user.type(screen.getByLabelText('Invoice prefix'), 'BF-');
    await user.selectOptions(
      screen.getByLabelText('Default payment terms'),
      'term-30'
    );
    await user.selectOptions(
      screen.getByLabelText('Default tax rate'),
      'rate-oh'
    );
    await user.click(
      screen.getByRole('button', { name: 'Save billing defaults' })
    );

    await waitFor(() => expect(updateTenant).toHaveBeenCalledTimes(1));
    const [id, body] = updateTenant.mock.calls[0];
    expect(id).toBe('tenant-1');
    expect(body.settings).toEqual({
      timezone: 'America/New_York',
      businessCity: 'Columbus',
      currency: 'CAD',
      invoicePrefix: 'BF-',
      quotePrefix: undefined,
      jobPrefix: undefined,
      defaultPaymentTermId: 'term-30',
      defaultTaxRateId: 'rate-oh',
      laborRate: undefined,
    });
    expect(
      await screen.findByText('Billing defaults saved')
    ).toBeInTheDocument();
  });

  it('saves the hourly labor rate for job invoices (T030)', async () => {
    updateTenant.mockResolvedValue(tenant);
    renderCard();
    const user = userEvent.setup();

    await user.type(screen.getByLabelText('Labor rate (per hour)'), '85.50');
    await user.click(
      screen.getByRole('button', { name: 'Save billing defaults' })
    );

    await waitFor(() => expect(updateTenant).toHaveBeenCalledTimes(1));
    expect(updateTenant.mock.calls[0][1].settings?.laborRate).toBe('85.50');
  });

  it('rejects a labor rate that is not an amount before calling the API', async () => {
    renderCard();
    const user = userEvent.setup();

    await user.type(screen.getByLabelText('Labor rate (per hour)'), '85.125');
    await user.click(
      screen.getByRole('button', { name: 'Save billing defaults' })
    );

    expect(
      await screen.findByText('Enter an hourly amount like 85 or 85.50')
    ).toBeInTheDocument();
    expect(updateTenant).not.toHaveBeenCalled();
  });

  it('shows the API field errors under each control', async () => {
    updateTenant.mockRejectedValue(
      validationError({
        currency: 'must be an ISO 4217 currency code, like USD',
        defaultTaxRateId: 'was not found',
      })
    );
    renderCard();
    const user = userEvent.setup();

    await user.click(
      screen.getByRole('button', { name: 'Save billing defaults' })
    );

    expect(
      await screen.findByText('must be an ISO 4217 currency code, like USD')
    ).toBeInTheDocument();
    expect(screen.getByText('was not found')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Some settings need fixing.'
    );
  });

  it('rejects a too-long prefix before calling the API', async () => {
    renderCard();
    const user = userEvent.setup();

    await user.type(screen.getByLabelText('Quote prefix'), 'X'.repeat(21));
    await user.click(
      screen.getByRole('button', { name: 'Save billing defaults' })
    );

    expect(
      await screen.findByText('Use at most 20 characters')
    ).toBeInTheDocument();
    expect(updateTenant).not.toHaveBeenCalled();
  });

  it('is read-only without tenants.update', async () => {
    renderCard(false);
    expect(screen.getByLabelText('Currency')).toBeDisabled();
    expect(
      screen.queryByRole('button', { name: 'Save billing defaults' })
    ).not.toBeInTheDocument();
  });
});
