import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router';
import type { AxiosResponse } from 'axios';
import type { PublicInvoice } from '@/types/publicInvoice';

const getDetails = vi.hoisted(() => vi.fn());

vi.mock('@services/publicInvoiceService', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('@services/publicInvoiceService')>();
  return { ...actual, publicInvoiceApi: { getDetails } };
});

import { PayInvoicePage } from '../PayInvoicePage';

function invoice(overrides: Partial<PublicInvoice> = {}): PublicInvoice {
  return {
    invoice_number: 'INV-1042',
    issue_date: '2026-10-31',
    due_date: '2026-12-01',
    status: 'sent',
    subtotal: '9882.00',
    tax_amount: '0.00',
    total_amount: '9882.00',
    amount_paid: '0.00',
    amount_due: '9882.00',
    currency: 'USD',
    customer_name: 'Harborview Property Mgmt',
    po_number: '7731-HV',
    business_name: 'Brightline Facility Services',
    business_location: 'Columbus, OH',
    pay_methods: [
      { method: 'bank', business_fee: '10.00' },
      { method: 'card', business_fee: '39.53' },
    ],
    lines: [],
    payment_pending: false,
    ...overrides,
  };
}

function renderPage(data: PublicInvoice) {
  getDetails.mockResolvedValue({ data } as AxiosResponse<PublicInvoice>);
  render(
    <MemoryRouter initialEntries={['/pay/tok_abc']}>
      <Routes>
        <Route path="/pay/:token" element={<PayInvoicePage />} />
      </Routes>
    </MemoryRouter>
  );
}

describe('PayInvoicePage (T170, mockup 17)', () => {
  beforeEach(() => {
    getDetails.mockReset();
  });

  it('shows the invoice and preselects bank as recommended', async () => {
    renderPage(invoice());

    expect(
      await screen.findByText('Brightline Facility Services')
    ).toBeInTheDocument();
    expect(getDetails).toHaveBeenCalledWith('tok_abc');
    expect(screen.getByText('Columbus, OH')).toBeInTheDocument();
    expect(
      screen.getByText('Invoice INV-1042 · October 2026')
    ).toBeInTheDocument();
    expect(screen.getByText('$9,882.00')).toBeInTheDocument();
    expect(screen.getByText('Due December 1, 2026')).toBeInTheDocument();
    expect(screen.getByText('Harborview Property Mgmt')).toBeInTheDocument();
    expect(screen.getByText('7731-HV')).toBeInTheDocument();

    expect(screen.getByRole('radio', { name: /pay by bank/i })).toBeChecked();
    expect(screen.getByText('Recommended')).toBeInTheDocument();
    expect(
      screen.getByText('$10.00 bank fee paid by Brightline Facility Services')
    ).toBeInTheDocument();

    const button = screen.getByRole('link', { name: 'Pay $9,882.00 by bank' });
    expect(button).toHaveAttribute(
      'href',
      '/api/v1/public/invoices/pay/tok_abc?method=bank'
    );
    expect(
      screen.getByText('Secure payment by Stripe · Powered by ServicePro')
    ).toBeInTheDocument();
  });

  it('switches the button to card when card is chosen', async () => {
    renderPage(invoice());

    await userEvent.click(
      await screen.findByRole('radio', { name: /pay by card/i })
    );

    const button = screen.getByRole('link', { name: 'Pay $9,882.00 by card' });
    expect(button).toHaveAttribute(
      'href',
      '/api/v1/public/invoices/pay/tok_abc?method=card'
    );
    expect(
      screen.getByText('$39.53 card fee paid by Brightline Facility Services')
    ).toBeInTheDocument();
  });

  it('hides a method the business cannot take', async () => {
    renderPage(
      invoice({ pay_methods: [{ method: 'card', business_fee: '39.53' }] })
    );

    expect(
      await screen.findByRole('radio', { name: /pay by card/i })
    ).toBeChecked();
    expect(
      screen.queryByRole('radio', { name: /pay by bank/i })
    ).not.toBeInTheDocument();
    expect(screen.queryByText('Recommended')).not.toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: 'Pay $9,882.00 by card' })
    ).toBeInTheDocument();
  });

  it('offers no payment when the invoice cannot be paid online', async () => {
    renderPage(invoice({ pay_methods: [] }));

    expect(
      await screen.findByText(/online payment isn't available/i)
    ).toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('says so when the invoice is already paid', async () => {
    renderPage(
      invoice({ status: 'paid', amount_due: '0.00', pay_methods: [] })
    );

    expect(await screen.findByText('This invoice is paid')).toBeInTheDocument();
    expect(screen.queryByRole('radio')).not.toBeInTheDocument();
  });

  it('shows a pending bank payment instead of a second payment', async () => {
    renderPage(invoice({ payment_pending: true, pay_methods: [] }));

    expect(
      await screen.findByText('Your bank payment is processing')
    ).toBeInTheDocument();
    expect(screen.queryByRole('radio')).not.toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('explains an invalid link', async () => {
    getDetails.mockRejectedValue({ response: { status: 404, data: {} } });
    render(
      <MemoryRouter initialEntries={['/pay/nope']}>
        <Routes>
          <Route path="/pay/:token" element={<PayInvoicePage />} />
        </Routes>
      </MemoryRouter>
    );

    expect(
      await screen.findByText(/this payment link isn't valid/i)
    ).toBeInTheDocument();
  });
});
