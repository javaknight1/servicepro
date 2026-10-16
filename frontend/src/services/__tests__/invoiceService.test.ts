import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../api', () => ({
  default: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));

import api from '../api';
import { invoiceService } from '../invoiceService';

// T232: these calls must match the Phase 2 money API routes and bodies.
describe('invoiceService money calls', () => {
  beforeEach(() => {
    vi.mocked(api.post).mockReset();
    vi.mocked(api.post).mockResolvedValue({ data: {} });
  });

  it('records a payment at POST /invoices/{id}/payments', async () => {
    await invoiceService.recordPayment('inv-1', {
      amount: 25.5,
      payment_method: 'check',
      reference_number: '1001',
    });
    expect(api.post).toHaveBeenCalledWith('/v1/invoices/inv-1/payments', {
      amount: 25.5,
      payment_method: 'check',
      reference_number: '1001',
    });
  });

  it('sends the required reason when cancelling', async () => {
    await invoiceService.cancelInvoice('inv-1', 'Duplicate invoice');
    expect(api.post).toHaveBeenCalledWith('/v1/invoices/inv-1/cancel', {
      reason: 'Duplicate invoice',
    });
  });
});
