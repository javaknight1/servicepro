import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import type { AxiosResponse } from 'axios';
import type { TaxRate, TaxRateInput } from '@services/taxRateService';

const listTaxRates = vi.fn<(activeOnly: boolean) => Promise<TaxRate[]>>();
const createTaxRate = vi.fn<(input: TaxRateInput) => Promise<TaxRate>>();
const updateTaxRate =
  vi.fn<(id: string, input: TaxRateInput) => Promise<TaxRate>>();
vi.mock('@services/taxRateService', () => ({
  taxRateService: {
    listTaxRates: (a: boolean) => listTaxRates(a),
    createTaxRate: (i: TaxRateInput) => createTaxRate(i),
    updateTaxRate: (id: string, i: TaxRateInput) => updateTaxRate(id, i),
  },
  taxRatesQueryKey: ['tax-rates', 'all'],
}));

import { TaxRatesCard } from '../TaxRatesCard';

const own: TaxRate = {
  id: 'rate-oh',
  tenant_id: 'tenant-1',
  name: 'Ohio',
  rate: '0.0800',
  tax_type: 'sales_tax',
  region: 'Franklin County',
  is_active: true,
};
const global: TaxRate = {
  id: 'rate-none',
  name: 'No Tax',
  rate: '0',
  tax_type: 'sales_tax',
  is_active: true,
};

function renderCard(canManage = true) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <TaxRatesCard canManage={canManage} />
    </QueryClientProvider>
  );
}

// T235: Settings -> Tax rates on the T237 API.
describe('TaxRatesCard', () => {
  beforeEach(() => {
    listTaxRates.mockReset();
    createTaxRate.mockReset();
    updateTaxRate.mockReset();
    listTaxRates.mockResolvedValue([global, own]);
  });

  it('lists own and built-in rates, all of them', async () => {
    renderCard();
    expect(await screen.findByText('Ohio')).toBeInTheDocument();
    expect(screen.getByText('· 8%')).toBeInTheDocument();
    expect(screen.getByText('Built-in')).toBeInTheDocument();
    expect(listTaxRates).toHaveBeenCalledWith(false);
    // Built-in rates are read-only.
    expect(
      screen.queryByRole('button', { name: 'Edit No Tax' })
    ).not.toBeInTheDocument();
  });

  it('creates a rate from a percentage', async () => {
    createTaxRate.mockResolvedValue({ ...own, id: 'rate-new' });
    renderCard();
    const user = userEvent.setup();
    await screen.findByText('Ohio');

    await user.click(screen.getByRole('button', { name: /add tax rate/i }));
    const dialog = await screen.findByRole('dialog');
    await user.type(within(dialog).getByLabelText('Name'), 'Columbus');
    await user.type(within(dialog).getByLabelText('Rate (%)'), '7.5');
    await user.click(
      within(dialog).getByRole('button', { name: 'Save tax rate' })
    );

    await waitFor(() =>
      expect(createTaxRate).toHaveBeenCalledWith({
        name: 'Columbus',
        rate: 0.075,
        region: '',
        is_active: true,
      })
    );
  });

  it('validates the form and shows API errors', async () => {
    createTaxRate.mockRejectedValue(
      new AxiosError('400', 'ERR_BAD_REQUEST', undefined, undefined, {
        status: 400,
        data: { error: 'invalid_tax_rate', message: 'rate is too high' },
      } as unknown as AxiosResponse)
    );
    renderCard();
    const user = userEvent.setup();
    await screen.findByText('Ohio');

    await user.click(screen.getByRole('button', { name: /add tax rate/i }));
    const dialog = await screen.findByRole('dialog');
    await user.click(
      within(dialog).getByRole('button', { name: 'Save tax rate' })
    );
    expect(within(dialog).getByText('Enter a name')).toBeInTheDocument();
    expect(within(dialog).getByText('Enter a percentage')).toBeInTheDocument();
    expect(createTaxRate).not.toHaveBeenCalled();

    await user.type(within(dialog).getByLabelText('Name'), 'Odd');
    await user.type(within(dialog).getByLabelText('Rate (%)'), '9');
    await user.click(
      within(dialog).getByRole('button', { name: 'Save tax rate' })
    );
    expect(
      await within(dialog).findByText(/rate is too high/)
    ).toBeInTheDocument();
  });

  it('edits and deactivates an own rate', async () => {
    updateTaxRate.mockResolvedValue(own);
    renderCard();
    const user = userEvent.setup();
    await screen.findByText('Ohio');

    await user.click(screen.getByRole('button', { name: 'Deactivate Ohio' }));
    await waitFor(() =>
      expect(updateTaxRate).toHaveBeenCalledWith('rate-oh', {
        is_active: false,
      })
    );

    await user.click(screen.getByRole('button', { name: 'Edit Ohio' }));
    const dialog = await screen.findByRole('dialog');
    const percent = within(dialog).getByLabelText('Rate (%)');
    expect(percent).toHaveValue(8);
    await user.clear(percent);
    await user.type(percent, '8.25');
    await user.click(
      within(dialog).getByRole('button', { name: 'Save tax rate' })
    );
    await waitFor(() =>
      expect(updateTaxRate).toHaveBeenCalledWith('rate-oh', {
        name: 'Ohio',
        rate: 0.0825,
        region: 'Franklin County',
      })
    );
  });

  it('hides management actions without tax_rates.manage', async () => {
    renderCard(false);
    await screen.findByText('Ohio');
    expect(
      screen.queryByRole('button', { name: /add tax rate/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Edit Ohio' })
    ).not.toBeInTheDocument();
    expect(
      screen.getByText('Only organization admins can change tax rates.')
    ).toBeInTheDocument();
  });
});
