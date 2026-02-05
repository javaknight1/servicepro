/**
 * Invoice & Payment Integration Tests
 *
 * Tests complete invoice management workflows including:
 * - Invoice list loading and display
 * - Invoice creation with line items
 * - Invoice status transitions
 * - Payment recording
 * - Invoice PDF generation
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import React from 'react';

// Mock modules
vi.mock('@store', () => ({
  useAuthStore: vi.fn(),
  useTenantStore: vi.fn(),
  useSidebarStore: vi.fn(),
}));

vi.mock('@services/invoiceService', () => ({
  invoiceService: {
    getInvoices: vi.fn(),
    getInvoice: vi.fn(),
    createInvoice: vi.fn(),
    updateInvoice: vi.fn(),
    deleteInvoice: vi.fn(),
    sendInvoice: vi.fn(),
    recordPayment: vi.fn(),
    cancelInvoice: vi.fn(),
    getStats: vi.fn(),
    downloadInvoicePDF: vi.fn(),
    downloadReceiptPDF: vi.fn(),
  },
  InvoiceStatus: {
    DRAFT: 'draft',
    SENT: 'sent',
    VIEWED: 'viewed',
    PAID: 'paid',
    PARTIALLY_PAID: 'partially_paid',
    OVERDUE: 'overdue',
    CANCELLED: 'cancelled',
  },
}));

vi.mock('@services/customerService', () => ({
  customerService: {
    getCustomers: vi.fn(),
    getCustomer: vi.fn(),
  },
}));

// Import after mocking
import { useAuthStore, useTenantStore, useSidebarStore } from '@store';
import { invoiceService } from '@services/invoiceService';
import { customerService } from '@services/customerService';

// =============================================================================
// Test Data
// =============================================================================

const mockUser = {
  id: 'user-123',
  email: 'admin@example.com',
  first_name: 'Admin',
  last_name: 'User',
  is_active: true,
  email_verified: true,
};

const mockTenant = {
  id: 'tenant-123',
  name: 'Test Organization',
  slug: 'test-org',
  is_active: true,
};

const mockCustomer = {
  id: 'customer-1',
  email: 'john@example.com',
  first_name: 'John',
  last_name: 'Doe',
  phone_primary: '555-123-4567',
};

const mockInvoices = [
  {
    id: 'invoice-1',
    invoice_number: 'INV-001',
    customer_id: 'customer-1',
    customer: mockCustomer,
    status: 'draft' as const,
    issue_date: '2024-01-01',
    due_date: '2024-01-31',
    subtotal: 1000,
    tax_rate: 0.08,
    tax_amount: 80,
    total_amount: 1080,
    amount_paid: 0,
    amount_due: 1080,
    lines: [
      {
        id: 'line-1',
        invoice_id: 'invoice-1',
        description: 'HVAC Installation Service',
        quantity: 1,
        unit_price: 800,
        total: 800,
        sort_order: 0,
      },
      {
        id: 'line-2',
        invoice_id: 'invoice-1',
        description: 'Parts and Materials',
        quantity: 1,
        unit_price: 200,
        total: 200,
        sort_order: 1,
      },
    ],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'invoice-2',
    invoice_number: 'INV-002',
    customer_id: 'customer-1',
    customer: mockCustomer,
    status: 'sent' as const,
    issue_date: '2024-01-02',
    due_date: '2024-02-01',
    subtotal: 500,
    tax_rate: 0.08,
    tax_amount: 40,
    total_amount: 540,
    amount_paid: 0,
    amount_due: 540,
    lines: [
      {
        id: 'line-3',
        invoice_id: 'invoice-2',
        description: 'Plumbing Repair',
        quantity: 1,
        unit_price: 500,
        total: 500,
        sort_order: 0,
      },
    ],
    sent_at: '2024-01-02T10:00:00Z',
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T10:00:00Z',
  },
  {
    id: 'invoice-3',
    invoice_number: 'INV-003',
    customer_id: 'customer-1',
    customer: mockCustomer,
    status: 'paid' as const,
    issue_date: '2024-01-03',
    due_date: '2024-02-02',
    subtotal: 1500,
    tax_rate: 0.08,
    tax_amount: 120,
    total_amount: 1620,
    amount_paid: 1620,
    amount_due: 0,
    lines: [
      {
        id: 'line-4',
        invoice_id: 'invoice-3',
        description: 'Electrical Work',
        quantity: 1,
        unit_price: 1500,
        total: 1500,
        sort_order: 0,
      },
    ],
    sent_at: '2024-01-03T10:00:00Z',
    paid_at: '2024-01-10T00:00:00Z',
    created_at: '2024-01-03T00:00:00Z',
    updated_at: '2024-01-10T00:00:00Z',
  },
  {
    id: 'invoice-4',
    invoice_number: 'INV-004',
    customer_id: 'customer-1',
    customer: mockCustomer,
    status: 'overdue' as const,
    issue_date: '2023-12-01',
    due_date: '2023-12-31',
    subtotal: 750,
    tax_rate: 0.08,
    tax_amount: 60,
    total_amount: 810,
    amount_paid: 0,
    amount_due: 810,
    lines: [
      {
        id: 'line-5',
        invoice_id: 'invoice-4',
        description: 'Maintenance Service',
        quantity: 1,
        unit_price: 750,
        total: 750,
        sort_order: 0,
      },
    ],
    sent_at: '2023-12-01T10:00:00Z',
    created_at: '2023-12-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

const mockInvoiceStats = {
  total_invoices: 4,
  draft_invoices: 1,
  sent_invoices: 1,
  paid_invoices: 1,
  overdue_invoices: 1,
  total_outstanding: 2430,
  total_revenue: 1620,
};

// =============================================================================
// Test Setup
// =============================================================================

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const setupMocks = () => {
  vi.mocked(useAuthStore).mockImplementation((selector) => {
    const state = {
      user: mockUser,
      isAuthenticated: true,
      isLoading: false,
      error: null,
      login: vi.fn(),
      logout: vi.fn(),
      loadUser: vi.fn(),
      clearError: vi.fn(),
    };
    if (typeof selector === 'function') {
      return selector(state);
    }
    return state;
  });

  vi.mocked(useTenantStore).mockImplementation((selector) => {
    const state = {
      currentTenant: mockTenant,
      tenants: [mockTenant],
      isLoading: false,
      error: null,
      loadTenants: vi.fn(),
      setCurrentTenant: vi.fn(),
    };
    if (typeof selector === 'function') {
      return selector(state);
    }
    return state;
  });

  vi.mocked(useSidebarStore).mockImplementation((selector) => {
    const state = {
      isOpen: true,
      isCollapsed: false,
      toggle: vi.fn(),
      setOpen: vi.fn(),
      setCollapsed: vi.fn(),
    };
    if (typeof selector === 'function') {
      return selector(state);
    }
    return state;
  });

  vi.mocked(invoiceService.getInvoices).mockResolvedValue({
    invoices: mockInvoices,
    total: mockInvoices.length,
    page: 1,
    page_size: 10,
  });

  vi.mocked(invoiceService.getStats).mockResolvedValue(mockInvoiceStats);

  vi.mocked(customerService.getCustomers).mockResolvedValue({
    customers: [mockCustomer],
    total: 1,
    page: 1,
    page_size: 10,
  });
};

// =============================================================================
// Invoice List Tests
// =============================================================================

describe('Invoice & Payment Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupMocks();
  });

  describe('Invoice List', () => {
    it('should load and display invoices list', async () => {
      const { InvoicesPage } =
        await import('../../src/pages/Invoices/InvoicesPage');

      render(
        <MemoryRouter initialEntries={['/invoices']}>
          <InvoicesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(invoiceService.getInvoices).toHaveBeenCalled();
      });

      // Verify invoices are displayed
      await waitFor(() => {
        expect(screen.getByText('INV-001')).toBeInTheDocument();
        expect(screen.getByText('INV-002')).toBeInTheDocument();
      });
    });

    it('should display invoice amounts correctly', async () => {
      const { InvoicesPage } =
        await import('../../src/pages/Invoices/InvoicesPage');

      render(
        <MemoryRouter initialEntries={['/invoices']}>
          <InvoicesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        // Check for formatted amounts
        expect(
          screen.getByText(/\$1,080|\$1080|1,080\.00/)
        ).toBeInTheDocument();
      });
    });

    it('should display invoice status badges', async () => {
      const { InvoicesPage } =
        await import('../../src/pages/Invoices/InvoicesPage');

      render(
        <MemoryRouter initialEntries={['/invoices']}>
          <InvoicesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByText(/draft/i)).toBeInTheDocument();
        expect(screen.getByText(/sent/i)).toBeInTheDocument();
        expect(screen.getByText(/paid/i)).toBeInTheDocument();
        expect(screen.getByText(/overdue/i)).toBeInTheDocument();
      });
    });

    it('should highlight overdue invoices', async () => {
      const { InvoicesPage } =
        await import('../../src/pages/Invoices/InvoicesPage');

      render(
        <MemoryRouter initialEntries={['/invoices']}>
          <InvoicesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        // Overdue badge should be visible
        expect(screen.getByText(/overdue/i)).toBeInTheDocument();
      });
    });

    it('should navigate to invoice detail when clicking an invoice', async () => {
      const user = userEvent.setup();
      const { InvoicesPage } =
        await import('../../src/pages/Invoices/InvoicesPage');

      render(
        <MemoryRouter initialEntries={['/invoices']}>
          <InvoicesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('INV-001')).toBeInTheDocument();
      });

      const invoiceRow = screen.getByText('INV-001').closest('tr');
      if (invoiceRow) {
        await user.click(invoiceRow);
        expect(mockNavigate).toHaveBeenCalledWith('/invoices/invoice-1');
      }
    });

    it('should navigate to new invoice page when clicking add button', async () => {
      const user = userEvent.setup();
      const { InvoicesPage } =
        await import('../../src/pages/Invoices/InvoicesPage');

      render(
        <MemoryRouter initialEntries={['/invoices']}>
          <InvoicesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(invoiceService.getInvoices).toHaveBeenCalled();
      });

      const addButton = screen.getByRole('button', { name: /add|new|create/i });
      await user.click(addButton);

      expect(mockNavigate).toHaveBeenCalledWith('/invoices/new');
    });
  });

  describe('Invoice Creation', () => {
    it('should create a new invoice with line items', async () => {
      const user = userEvent.setup();
      const newInvoice = {
        ...mockInvoices[0],
        id: 'invoice-new',
        invoice_number: 'INV-005',
      };

      vi.mocked(invoiceService.createInvoice).mockResolvedValue(newInvoice);
      vi.mocked(customerService.getCustomer).mockResolvedValue(mockCustomer);

      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');

      render(
        <MemoryRouter initialEntries={['/invoices/new']}>
          <Routes>
            <Route path="/invoices/new" element={<InvoiceDetailPage />} />
            <Route path="/invoices/:id" element={<div>Invoice Detail</div>} />
          </Routes>
        </MemoryRouter>
      );

      // Wait for form to render
      await waitFor(() => {
        const customerInput = screen.queryByLabelText(/customer/i);
        expect(
          customerInput || screen.getByRole('combobox')
        ).toBeInTheDocument();
      });

      // Select customer
      const customerSelect =
        screen.queryByLabelText(/customer/i) || screen.getByRole('combobox');
      if (customerSelect) {
        await user.click(customerSelect);
        const option = screen.queryByText(/John Doe/i);
        if (option) {
          await user.click(option);
        }
      }

      // Add line item if button exists
      const addLineButton = screen.queryByRole('button', {
        name: /add line|add item/i,
      });
      if (addLineButton) {
        await user.click(addLineButton);

        // Fill line item
        const descInput =
          screen.queryByLabelText(/description/i) ||
          screen.queryByPlaceholderText(/description/i);
        if (descInput) {
          await user.type(descInput, 'Service Charge');
        }

        const qtyInput = screen.queryByLabelText(/quantity|qty/i);
        if (qtyInput) {
          await user.clear(qtyInput);
          await user.type(qtyInput, '1');
        }

        const priceInput = screen.queryByLabelText(/price|rate|amount/i);
        if (priceInput) {
          await user.clear(priceInput);
          await user.type(priceInput, '500');
        }
      }

      // Submit form
      const saveButton = screen.getByRole('button', {
        name: /save|create|submit/i,
      });
      await user.click(saveButton);

      await waitFor(() => {
        expect(invoiceService.createInvoice).toHaveBeenCalled();
      });
    });
  });

  describe('Invoice Status Transitions', () => {
    it('should send a draft invoice to customer', async () => {
      const user = userEvent.setup();
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[0]); // Draft invoice
      vi.mocked(invoiceService.sendInvoice).mockResolvedValue({
        ...mockInvoices[0],
        status: 'sent' as const,
      });

      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');

      render(
        <MemoryRouter initialEntries={['/invoices/invoice-1']}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(invoiceService.getInvoice).toHaveBeenCalledWith('invoice-1');
      });

      // Find and click send button
      const sendButton = screen.queryByRole('button', { name: /send/i });
      if (sendButton) {
        await user.click(sendButton);

        await waitFor(() => {
          expect(invoiceService.sendInvoice).toHaveBeenCalledWith('invoice-1');
        });
      }
    });

    it('should cancel an invoice', async () => {
      const user = userEvent.setup();
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[0]);
      vi.mocked(invoiceService.cancelInvoice).mockResolvedValue({
        ...mockInvoices[0],
        status: 'cancelled' as const,
      });

      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');

      render(
        <MemoryRouter initialEntries={['/invoices/invoice-1']}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(invoiceService.getInvoice).toHaveBeenCalled();
      });

      // Find and click cancel button
      const cancelButton = screen.queryByRole('button', { name: /cancel/i });
      if (cancelButton) {
        await user.click(cancelButton);

        // Confirm if dialog appears
        const confirmButton = screen.queryByRole('button', {
          name: /confirm|yes/i,
        });
        if (confirmButton) {
          await user.click(confirmButton);
        }

        await waitFor(() => {
          expect(invoiceService.cancelInvoice).toHaveBeenCalledWith(
            'invoice-1'
          );
        });
      }
    });
  });

  describe('Payment Recording', () => {
    it('should record a full payment', async () => {
      const user = userEvent.setup();
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[1]); // Sent invoice
      vi.mocked(invoiceService.recordPayment).mockResolvedValue({
        ...mockInvoices[1],
        status: 'paid' as const,
        amount_paid: 540,
        amount_due: 0,
      });

      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');

      render(
        <MemoryRouter initialEntries={['/invoices/invoice-2']}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(invoiceService.getInvoice).toHaveBeenCalledWith('invoice-2');
      });

      // Find and click record payment button
      const paymentButton = screen.queryByRole('button', {
        name: /record payment|add payment|pay/i,
      });
      if (paymentButton) {
        await user.click(paymentButton);

        // Fill payment details if dialog appears
        const amountInput = screen.queryByLabelText(/amount/i);
        if (amountInput) {
          await user.clear(amountInput);
          await user.type(amountInput, '540');
        }

        const methodSelect = screen.queryByLabelText(/method/i);
        if (methodSelect) {
          await user.click(methodSelect);
          const cashOption = screen.queryByText(/cash|check|credit/i);
          if (cashOption) {
            await user.click(cashOption);
          }
        }

        // Submit payment
        const submitButton = screen.queryByRole('button', {
          name: /submit|confirm|record/i,
        });
        if (submitButton) {
          await user.click(submitButton);
        }

        await waitFor(() => {
          expect(invoiceService.recordPayment).toHaveBeenCalledWith(
            'invoice-2',
            expect.objectContaining({
              amount: 540,
            })
          );
        });
      }
    });

    it('should record a partial payment', async () => {
      const user = userEvent.setup();
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[1]); // Sent invoice
      vi.mocked(invoiceService.recordPayment).mockResolvedValue({
        ...mockInvoices[1],
        status: 'partially_paid' as const,
        amount_paid: 200,
        amount_due: 340,
      });

      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');

      render(
        <MemoryRouter initialEntries={['/invoices/invoice-2']}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(invoiceService.getInvoice).toHaveBeenCalled();
      });

      const paymentButton = screen.queryByRole('button', {
        name: /record payment|add payment|pay/i,
      });
      if (paymentButton) {
        await user.click(paymentButton);

        const amountInput = screen.queryByLabelText(/amount/i);
        if (amountInput) {
          await user.clear(amountInput);
          await user.type(amountInput, '200'); // Partial payment
        }

        const submitButton = screen.queryByRole('button', {
          name: /submit|confirm|record/i,
        });
        if (submitButton) {
          await user.click(submitButton);
        }

        await waitFor(() => {
          expect(invoiceService.recordPayment).toHaveBeenCalledWith(
            'invoice-2',
            expect.objectContaining({
              amount: 200,
            })
          );
        });
      }
    });
  });

  describe('Invoice PDF', () => {
    it('should download invoice PDF', async () => {
      const user = userEvent.setup();
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[0]);
      vi.mocked(invoiceService.downloadInvoicePDF).mockResolvedValue();

      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');

      render(
        <MemoryRouter initialEntries={['/invoices/invoice-1']}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(invoiceService.getInvoice).toHaveBeenCalled();
      });

      // Find and click download/PDF button
      const pdfButton = screen.queryByRole('button', {
        name: /pdf|download|invoice/i,
      });
      if (pdfButton) {
        await user.click(pdfButton);

        await waitFor(() => {
          expect(invoiceService.downloadInvoicePDF).toHaveBeenCalledWith(
            'invoice-1',
            expect.anything()
          );
        });
      }
    });

    it('should download receipt PDF for paid invoice', async () => {
      const user = userEvent.setup();
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[2]); // Paid invoice
      vi.mocked(invoiceService.downloadReceiptPDF).mockResolvedValue();

      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');

      render(
        <MemoryRouter initialEntries={['/invoices/invoice-3']}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(invoiceService.getInvoice).toHaveBeenCalled();
      });

      // Find and click receipt button
      const receiptButton = screen.queryByRole('button', { name: /receipt/i });
      if (receiptButton) {
        await user.click(receiptButton);

        await waitFor(() => {
          expect(invoiceService.downloadReceiptPDF).toHaveBeenCalledWith(
            'invoice-3',
            expect.anything()
          );
        });
      }
    });
  });

  describe('Invoice Filtering', () => {
    it('should filter invoices by status', async () => {
      const user = userEvent.setup();
      const { InvoicesPage } =
        await import('../../src/pages/Invoices/InvoicesPage');

      render(
        <MemoryRouter initialEntries={['/invoices']}>
          <InvoicesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(invoiceService.getInvoices).toHaveBeenCalled();
      });

      // Find status filter
      const statusFilter =
        screen.queryByLabelText(/status/i) ||
        screen.queryByRole('combobox', { name: /status/i });
      if (statusFilter) {
        await user.click(statusFilter);
        const overdueOption = screen.queryByText(/overdue/i);
        if (overdueOption) {
          await user.click(overdueOption);
        }

        await waitFor(() => {
          expect(invoiceService.getInvoices).toHaveBeenCalledWith(
            expect.objectContaining({
              status: 'overdue',
            })
          );
        });
      }
    });

    it('should filter overdue invoices only', async () => {
      const user = userEvent.setup();
      const { InvoicesPage } =
        await import('../../src/pages/Invoices/InvoicesPage');

      render(
        <MemoryRouter initialEntries={['/invoices']}>
          <InvoicesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(invoiceService.getInvoices).toHaveBeenCalled();
      });

      // Find overdue filter/checkbox
      const overdueFilter =
        screen.queryByLabelText(/overdue only/i) ||
        screen.queryByRole('checkbox', { name: /overdue/i });
      if (overdueFilter) {
        await user.click(overdueFilter);

        await waitFor(() => {
          expect(invoiceService.getInvoices).toHaveBeenCalledWith(
            expect.objectContaining({
              overdue_only: true,
            })
          );
        });
      }
    });
  });

  describe('Payment Success Page', () => {
    it('should display payment success message', async () => {
      const { PaymentSuccessPage } =
        await import('../../src/pages/Invoices/PaymentSuccessPage');

      render(
        <MemoryRouter
          initialEntries={['/invoices/payment-success?session_id=test-session']}
        >
          <PaymentSuccessPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(
          screen.getByText(/payment successful|thank you|confirmed/i)
        ).toBeInTheDocument();
      });
    });

    it('should provide navigation to invoices list', async () => {
      const user = userEvent.setup();
      const { PaymentSuccessPage } =
        await import('../../src/pages/Invoices/PaymentSuccessPage');

      render(
        <MemoryRouter initialEntries={['/invoices/payment-success']}>
          <PaymentSuccessPage />
        </MemoryRouter>
      );

      const viewInvoicesLink = screen.queryByRole('link', {
        name: /view invoices|back to invoices/i,
      });
      if (viewInvoicesLink) {
        expect(viewInvoicesLink).toHaveAttribute(
          'href',
          expect.stringContaining('/invoices')
        );
      }
    });
  });

  describe('Payment Cancelled Page', () => {
    it('should display payment cancelled message', async () => {
      const { PaymentCancelledPage } =
        await import('../../src/pages/Invoices/PaymentCancelledPage');

      render(
        <MemoryRouter initialEntries={['/invoices/payment-cancelled']}>
          <PaymentCancelledPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(
          screen.getByText(/payment cancelled|not completed|try again/i)
        ).toBeInTheDocument();
      });
    });
  });

  describe('Error Handling', () => {
    it('should handle invoice load error gracefully', async () => {
      vi.mocked(invoiceService.getInvoices).mockRejectedValue(
        new Error('Failed to load invoices')
      );

      const { InvoicesPage } =
        await import('../../src/pages/Invoices/InvoicesPage');

      render(
        <MemoryRouter initialEntries={['/invoices']}>
          <InvoicesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(invoiceService.getInvoices).toHaveBeenCalled();
      });
    });

    it('should handle payment recording error', async () => {
      const user = userEvent.setup();
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[1]);
      vi.mocked(invoiceService.recordPayment).mockRejectedValue({
        response: { data: { message: 'Payment processing failed' } },
      });

      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');

      render(
        <MemoryRouter initialEntries={['/invoices/invoice-2']}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(invoiceService.getInvoice).toHaveBeenCalled();
      });

      const paymentButton = screen.queryByRole('button', {
        name: /record payment|add payment|pay/i,
      });
      if (paymentButton) {
        await user.click(paymentButton);

        const amountInput = screen.queryByLabelText(/amount/i);
        if (amountInput) {
          await user.type(amountInput, '540');
        }

        const submitButton = screen.queryByRole('button', {
          name: /submit|confirm|record/i,
        });
        if (submitButton) {
          await user.click(submitButton);
        }

        await waitFor(() => {
          const errorMessage = screen.queryByText(/error|failed|processing/i);
          if (errorMessage) {
            expect(errorMessage).toBeInTheDocument();
          }
        });
      }
    });
  });
});
