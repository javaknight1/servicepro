import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router';

vi.mock('../../services/invoiceService', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('../../services/invoiceService')>();
  return { ...actual, invoiceService: { sendInvoice: vi.fn() } };
});

import { invoiceService, InvoiceStatus } from '../../services/invoiceService';
import { InvoiceActionsMenu } from './InvoiceActionsMenu';

function DetailProbe() {
  const location = useLocation();
  const state = location.state as { poRequired?: boolean } | null;
  return (
    <p>
      detail {location.pathname} {state?.poRequired ? 'needs PO' : ''}
    </p>
  );
}

describe('InvoiceActionsMenu', () => {
  beforeEach(() => vi.clearAllMocks());

  it('opens the invoice to ask for a PO when send needs one (T187)', async () => {
    const user = userEvent.setup();
    const onInvoiceSent = vi.fn();
    vi.mocked(invoiceService.sendInvoice).mockRejectedValue({
      response: {
        status: 409,
        data: { error: 'po_required', message: 'PO required' },
      },
    });

    render(
      <MemoryRouter initialEntries={['/invoices']}>
        <Routes>
          <Route
            path="/invoices"
            element={
              <InvoiceActionsMenu
                invoiceId="inv-1"
                status={InvoiceStatus.DRAFT}
                onInvoiceSent={onInvoiceSent}
              />
            }
          />
          <Route path="/invoices/:id" element={<DetailProbe />} />
        </Routes>
      </MemoryRouter>
    );

    await user.click(screen.getByRole('button', { name: 'Invoice actions' }));
    await user.click(await screen.findByText('Send Invoice'));

    expect(
      await screen.findByText(/detail \/invoices\/inv-1 needs PO/)
    ).toBeInTheDocument();
    expect(onInvoiceSent).not.toHaveBeenCalled();
  });
});
