import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosError, AxiosHeaders } from 'axios';
import type { ReactElement } from 'react';

vi.mock('@services/siteService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@services/siteService')>();
  return {
    ...actual,
    siteService: { list: vi.fn(), create: vi.fn(), update: vi.fn() },
  };
});
vi.mock('@services/purchaseOrderService', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('@services/purchaseOrderService')>();
  return {
    ...actual,
    purchaseOrderService: { create: vi.fn(), update: vi.fn() },
  };
});
vi.mock('@services/customerService', () => ({
  customerService: { updateCustomer: vi.fn() },
}));

import { siteService, type Site } from '@services/siteService';
import { customerService, type Customer } from '@services/customerService';
import { BillingRulesCards } from '../BillingRulesCards';
import { BillingRulesModal } from '../BillingRulesModal';
import { SiteFormModal } from '../SiteFormModal';
import { PurchaseOrderFormModal } from '../PurchaseOrderFormModal';
import {
  purchaseOrderService,
  type PurchaseOrder,
} from '@services/purchaseOrderService';

function renderWithClient(ui: ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>
  );
}

const customer = {
  id: 'c-1',
  email: 'pm@example.com',
  first_name: 'Karen',
  last_name: 'Doyle',
  phone_primary: '555-0100',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
} as unknown as Customer;

const site: Site = {
  id: 's-1',
  customer_id: 'c-1',
  name: 'Elm Street Plaza',
  address_street: '410 Elm St',
  address_city: 'Columbus',
  address_state: 'OH',
  address_zip: '43215',
  address_full: '410 Elm St, Columbus, OH 43215',
  access_notes: 'Side door',
  contact_name: 'Ray Burke',
  contact_phone: null,
  contact_email: 'ray@example.com',
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

function validationError(fields: Record<string, string>) {
  return new AxiosError(
    'Bad Request',
    'ERR_BAD_REQUEST',
    undefined,
    undefined,
    {
      data: { error: 'validation_error', message: 'Invalid', fields },
      status: 400,
      statusText: 'Bad Request',
      headers: {},
      config: { headers: new AxiosHeaders() },
    }
  );
}

describe('BillingRulesCards', () => {
  it('shows the defaults when the customer has no rules', () => {
    render(<BillingRulesCards customer={customer} />);
    expect(screen.getByText('Per job')).toBeInTheDocument();
    expect(screen.getByText('PO optional')).toBeInTheDocument();
    expect(screen.getByText('No approver')).toBeInTheDocument();
    expect(screen.getByText('No limit')).toBeInTheDocument();
  });

  it('keeps cents on a limit that has them', () => {
    render(
      <BillingRulesCards customer={{ ...customer, nte_limit: '750.50' }} />
    );
    expect(screen.getByText('$750.50 not-to-exceed')).toBeInTheDocument();
  });
});

describe('BillingRulesModal', () => {
  beforeEach(() => vi.clearAllMocks());

  it('sends the NTE limit and saved rules', async () => {
    const user = userEvent.setup();
    const onSaved = vi.fn();
    vi.mocked(customerService.updateCustomer).mockResolvedValue(customer);
    renderWithClient(
      <BillingRulesModal
        customer={customer}
        isOpen
        onClose={vi.fn()}
        onSaved={onSaved}
      />
    );
    await user.click(screen.getByLabelText(/po required/i));
    await user.type(screen.getByLabelText(/extra work limit/i), '500');
    await user.click(screen.getByRole('button', { name: /save rules/i }));
    await waitFor(() =>
      expect(customerService.updateCustomer).toHaveBeenCalledWith('c-1', {
        consolidated_billing: false,
        po_required: true,
        default_po_number: '',
        billing_approver_name: '',
        billing_approver_email: '',
        nte_limit: '500',
      })
    );
    expect(onSaved).toHaveBeenCalledWith(customer);
  });

  it('shows field errors from the API', async () => {
    const user = userEvent.setup();
    vi.mocked(customerService.updateCustomer).mockRejectedValue(
      validationError({
        billing_approver_email: 'must be a valid email address',
      })
    );
    renderWithClient(
      <BillingRulesModal
        customer={customer}
        isOpen
        onClose={vi.fn()}
        onSaved={vi.fn()}
      />
    );
    await user.click(screen.getByRole('button', { name: /save rules/i }));
    expect(
      await screen.findByText('must be a valid email address')
    ).toBeInTheDocument();
  });
});

describe('SiteFormModal', () => {
  beforeEach(() => vi.clearAllMocks());

  it('edits a site and clears an emptied field', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    vi.mocked(siteService.update).mockResolvedValue(site);
    renderWithClient(
      <SiteFormModal customerId="c-1" site={site} isOpen onClose={onClose} />
    );
    const dialog = screen.getByRole('dialog');
    expect(within(dialog).getByLabelText(/site name/i)).toHaveValue(
      'Elm Street Plaza'
    );
    await user.clear(within(dialog).getByLabelText(/access notes/i));
    await user.click(
      within(dialog).getByRole('button', { name: /save site/i })
    );
    await waitFor(() =>
      expect(siteService.update).toHaveBeenCalledWith(
        'c-1',
        's-1',
        expect.objectContaining({
          name: 'Elm Street Plaza',
          access_notes: '',
          contact_email: 'ray@example.com',
        })
      )
    );
    expect(onClose).toHaveBeenCalled();
  });
});

describe('PurchaseOrderFormModal', () => {
  beforeEach(() => vi.clearAllMocks());

  const po: PurchaseOrder = {
    id: 'po-1',
    customer_id: 'c-1',
    po_number: 'PO-7731-HV',
    amount: '30000.00',
    billed: '1000.00',
    remaining: '29000.00',
    invoice_count: 1,
    expires_on: '2027-09-30',
    expired: false,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };

  it('edits the amount and clears the expiry (T187)', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    vi.mocked(purchaseOrderService.update).mockResolvedValue(po);
    renderWithClient(
      <PurchaseOrderFormModal
        customerId="c-1"
        purchaseOrder={po}
        isOpen
        onClose={onClose}
      />
    );
    const dialog = screen.getByRole('dialog');
    const amount = within(dialog).getByLabelText(/authorised amount/i);
    expect(amount).toHaveValue(30000);
    await user.clear(amount);
    await user.type(amount, '35000');
    await user.clear(within(dialog).getByLabelText(/expires/i));
    await user.click(within(dialog).getByRole('button', { name: /save po/i }));
    await waitFor(() =>
      expect(purchaseOrderService.update).toHaveBeenCalledWith('c-1', 'po-1', {
        po_number: 'PO-7731-HV',
        amount: '35000',
        notes: '',
        clear_expires_on: true,
      })
    );
    expect(onClose).toHaveBeenCalled();
  });

  it('shows a duplicate PO number as an error', async () => {
    const user = userEvent.setup();
    vi.mocked(purchaseOrderService.create).mockRejectedValue(
      new AxiosError('Conflict', 'ERR_BAD_REQUEST', undefined, undefined, {
        data: {
          error: 'purchase_order_exists',
          message:
            'This customer already has a purchase order with that number',
        },
        status: 409,
        statusText: 'Conflict',
        headers: {},
        config: { headers: new AxiosHeaders() },
      })
    );
    renderWithClient(
      <PurchaseOrderFormModal customerId="c-1" isOpen onClose={vi.fn()} />
    );
    const dialog = screen.getByRole('dialog');
    await user.type(within(dialog).getByLabelText(/po number/i), 'PO-1');
    await user.type(within(dialog).getByLabelText(/authorised amount/i), '50');
    await user.click(within(dialog).getByRole('button', { name: /add po/i }));
    expect(
      await within(dialog).findByText(/already has a purchase order/i)
    ).toBeInTheDocument();
  });
});
