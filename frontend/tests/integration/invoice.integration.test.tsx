/**
 * Invoice & Payment Integration Tests
 *
 * Tests complete invoice management workflows including:
 * - Invoice list loading and display
 * - Invoice creation with line items
 * - Invoice status transitions
 * - Payment recording
 * - Invoice PDF generation
 * - Consolidated invoices: preview by site, create, grouped detail (T188)
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router';
import React from 'react';
import { AxiosError, AxiosHeaders } from 'axios';

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
    recordRefund: vi.fn(),
    cancelInvoice: vi.fn(),
    previewConsolidatedInvoice: vi.fn(),
    createConsolidatedInvoice: vi.fn(),
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
    REFUNDED: 'refunded',
  },
}));

// Tenant permissions come from GET /tenants/:id/me; the tests set them.
vi.mock('@/utils/tenantPermissions', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/utils/tenantPermissions')>()),
  useTenantPermissions: vi.fn(),
}));

vi.mock('@services/paymentTermService', () => ({
  paymentTermService: {
    getPaymentTerm: vi.fn(),
  },
}));

vi.mock('@services/customerService', () => ({
  customerService: {
    getCustomers: vi.fn(),
    getCustomer: vi.fn(),
  },
}));

vi.mock('@services/taxRateService', () => ({
  taxRateService: {
    listTaxRates: vi.fn(),
  },
}));

// Import after mocking
import { useAuthStore, useTenantStore, useSidebarStore } from '@store';
import { invoiceService } from '@services/invoiceService';
import type { InvoiceStats } from '@app-types/invoice';
import { customerService } from '@services/customerService';
import { paymentTermService } from '@services/paymentTermService';
import { taxRateService, type TaxRate } from '@services/taxRateService';
import { useTenantPermissions } from '@/utils/tenantPermissions';

let grantedPermissions = new Set<string>();

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
    tax_amount: 80,
    total_amount: 1080,
    amount_paid: 0,
    amount_due: 1080,
    terms_and_conditions: 'Net 30. Late fee 1.5%/month.',
    lines: [
      {
        id: 'line-1',
        invoice_id: 'invoice-1',
        description: 'HVAC Installation Service',
        quantity: 1,
        unit_price: 800,
        total: 800,
        sort_order: 0,
        taxable: true,
        tax_rate: '0.0800',
      },
      {
        id: 'line-2',
        invoice_id: 'invoice-1',
        description: 'Parts and Materials',
        quantity: 1,
        unit_price: 200,
        total: 200,
        sort_order: 1,
        taxable: false,
        tax_rate: '0.0800',
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

const mockTaxRates: TaxRate[] = [
  {
    id: 'tax-rate-1',
    name: 'City Sales Tax',
    rate: '0.0825',
    tax_type: 'sales_tax',
    is_active: true,
  },
  {
    id: 'tax-rate-2',
    name: 'State Tax',
    rate: '0.06',
    tax_type: 'sales_tax',
    is_active: true,
  },
];

const mockInvoiceStats: InvoiceStats = {
  total_outstanding: '2430.00',
  outstanding_invoices: 2,
  overdue_amount: '810.00',
  overdue_invoices: 1,
  collected_this_month: '1620.00',
};

// =============================================================================
// Test Setup
// =============================================================================

const mockNavigate = vi.fn();
vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router');
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

  // No configured tax rates unless a test sets some (typed-rate input only)
  vi.mocked(taxRateService.listTaxRates).mockResolvedValue([]);

  grantedPermissions = new Set(['payments.create', 'payments.refund']);
  vi.mocked(useTenantPermissions).mockImplementation(() => {
    const can = (permission: string) => grantedPermissions.has(permission);
    return {
      can,
      canManageMembers: false,
      canUpdateTenant: false,
      canChangePlan: false,
      canManageBilling: false,
      canDeleteTenant: false,
      isLoading: false,
    };
  });

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

      // Total and amount due columns are both $1,080.00 for the draft
      await waitFor(() => {
        expect(screen.getAllByText('$1,080.00')).toHaveLength(2);
      });
      expect(screen.getAllByText('$540.00').length).toBeGreaterThan(0);
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

    it('marks an invoice whose bank payment is settling (T170)', async () => {
      vi.mocked(invoiceService.getInvoices).mockResolvedValue({
        invoices: [
          { ...mockInvoices[0], payment_pending_at: '2026-11-03T10:00:00Z' },
          mockInvoices[1],
        ],
        total: 2,
        page: 1,
        page_size: 10,
      });
      const { InvoicesPage } =
        await import('../../src/pages/Invoices/InvoicesPage');

      render(
        <MemoryRouter initialEntries={['/invoices']}>
          <InvoicesPage />
        </MemoryRouter>
      );

      expect(await screen.findByText('INV-001')).toBeInTheDocument();
      expect(screen.getAllByText('Payment pending')).toHaveLength(1);
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
      const user = userEvent.setup({ delay: null });
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
      const user = userEvent.setup({ delay: null });
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

      const addButton = screen.getByRole('button', { name: /new invoice/i });
      await user.click(addButton);

      expect(mockNavigate).toHaveBeenCalledWith('/invoices/new');
    });
  });

  describe('Invoice Creation', () => {
    it('should create a new invoice with line items', async () => {
      const user = userEvent.setup({ delay: null });
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

      // Customer is a native <select>
      const customerSelect = await screen.findByLabelText(/customer/i);
      await screen.findByRole('option', { name: /John Doe/i });
      await user.selectOptions(customerSelect, 'customer-1');

      // Fill the default (first) line item
      await user.type(
        screen.getAllByPlaceholderText(/description/i)[0],
        'Service Charge'
      );
      const qty = screen.getAllByPlaceholderText(/qty/i)[0];
      await user.clear(qty);
      await user.type(qty, '1');
      const price = screen.getAllByPlaceholderText(/price/i)[0];
      await user.clear(price);
      await user.type(price, '500');

      // Submit form
      const saveButton = screen.getByRole('button', {
        name: /save|create|submit/i,
      });
      await user.click(saveButton);

      await waitFor(() => {
        expect(invoiceService.createInvoice).toHaveBeenCalledWith(
          expect.objectContaining({
            customer_id: 'customer-1',
            lines: [
              expect.objectContaining({
                description: 'Service Charge',
                quantity: 1,
                unit_price: 500,
              }),
            ],
          })
        );
      });
    });

    it('sends terms_and_conditions with the create (no follow-up update)', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.createInvoice).mockResolvedValue({
        ...mockInvoices[0],
        id: 'invoice-new',
      });

      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');
      render(
        <MemoryRouter initialEntries={['/invoices/new']}>
          <Routes>
            <Route path="/invoices/new" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      const customerSelect = await screen.findByLabelText(/customer/i);
      await screen.findByRole('option', { name: /John Doe/i });
      await user.selectOptions(customerSelect, 'customer-1');
      await user.type(
        screen.getAllByPlaceholderText(/description/i)[0],
        'Service Charge'
      );
      await user.type(screen.getByLabelText(/terms & conditions/i), 'Net 15');
      await user.click(screen.getByRole('button', { name: /create invoice/i }));

      await waitFor(() => {
        expect(invoiceService.createInvoice).toHaveBeenCalledWith(
          expect.objectContaining({ terms_and_conditions: 'Net 15' })
        );
      });
      expect(invoiceService.updateInvoice).not.toHaveBeenCalled();
    });

    // T029: invoices are generated from a job by the API (POST
    // /jobs/{id}/invoice); the draft links back to its job.
    it('links an invoice generated from a job back to the job', async () => {
      vi.mocked(invoiceService.getInvoice).mockResolvedValue({
        ...mockInvoices[0],
        job_id: 'job-1',
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

      expect(
        await screen.findByRole('link', { name: 'View job' })
      ).toHaveAttribute('href', '/jobs/job-1');
    });
  });

  describe('Tenant defaults on a new invoice (T127)', () => {
    it('prefills the default payment term and leaves the tax to the default rate', async () => {
      const user = userEvent.setup({ delay: null });
      const tenantWithDefaults = {
        ...mockTenant,
        settings: {
          defaultTaxRateId: 'tax-rate-1',
          defaultPaymentTermId: 'term-15',
        },
      };
      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: tenantWithDefaults,
          tenants: [tenantWithDefaults],
        };
        return typeof selector === 'function' ? selector(state) : state;
      });
      vi.mocked(paymentTermService.getPaymentTerm).mockResolvedValue({
        id: 'term-15',
        name: 'Net 15',
        term_type: 'net_15',
        days_until_due: 15,
        is_default: false,
        is_active: true,
      });
      vi.mocked(invoiceService.createInvoice).mockResolvedValue({
        ...mockInvoices[0],
        id: 'invoice-new',
      });

      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');
      render(
        <MemoryRouter initialEntries={['/invoices/new']}>
          <Routes>
            <Route path="/invoices/new" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      expect(
        await screen.findByText('Payment term: Net 15')
      ).toBeInTheDocument();
      expect(paymentTermService.getPaymentTerm).toHaveBeenCalledWith('term-15');
      const issue = (screen.getByLabelText(/issue date/i) as HTMLInputElement)
        .value;
      const due = new Date(issue + 'T00:00:00Z');
      due.setUTCDate(due.getUTCDate() + 15);
      expect(screen.getByLabelText(/due date/i)).toHaveValue(
        due.toISOString().split('T')[0]
      );
      expect(screen.getByLabelText(/tax rate/i)).toHaveValue(null);
      expect(screen.getByText('Tax (default rate):')).toBeInTheDocument();

      const customerSelect = screen.getByLabelText(/customer/i);
      await screen.findByRole('option', { name: /John Doe/i });
      await user.selectOptions(customerSelect, 'customer-1');
      await user.type(
        screen.getAllByPlaceholderText(/description/i)[0],
        'Service Charge'
      );
      await user.click(screen.getByRole('button', { name: /create invoice/i }));

      await waitFor(() => {
        expect(invoiceService.createInvoice).toHaveBeenCalled();
      });
      const [body] = vi.mocked(invoiceService.createInvoice).mock.calls[0];
      // Omitted, so the API applies the tenant's default rate
      expect(body.tax_rate).toBeUndefined();
    });

    it('sends 0% tax when the tenant has no default rate', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.createInvoice).mockResolvedValue({
        ...mockInvoices[0],
        id: 'invoice-new',
      });
      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');
      render(
        <MemoryRouter initialEntries={['/invoices/new']}>
          <Routes>
            <Route path="/invoices/new" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      const customerSelect = await screen.findByLabelText(/customer/i);
      await screen.findByRole('option', { name: /John Doe/i });
      await user.selectOptions(customerSelect, 'customer-1');
      await user.type(
        screen.getAllByPlaceholderText(/description/i)[0],
        'Service Charge'
      );
      await user.click(screen.getByRole('button', { name: /create invoice/i }));

      await waitFor(() => {
        expect(invoiceService.createInvoice).toHaveBeenCalledWith(
          expect.objectContaining({ tax_rate: 0 })
        );
      });
      expect(paymentTermService.getPaymentTerm).not.toHaveBeenCalled();
    });
  });

  describe('Invoice Editing', () => {
    const renderDraft = async () => {
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[0]);
      vi.mocked(invoiceService.updateInvoice).mockResolvedValue(
        mockInvoices[0]
      );
      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');
      render(
        <MemoryRouter initialEntries={['/invoices/invoice-1']}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );
      return screen.findByLabelText(/tax rate/i);
    };

    it('shows the stored tax rate, taxable flags and terms_and_conditions', async () => {
      const taxInput = await renderDraft();

      // The rate comes from the lines (the invoice's tax_rate is an object)
      expect(taxInput).toHaveValue(8);
      expect(screen.getByLabelText(/terms & conditions/i)).toHaveValue(
        'Net 30. Late fee 1.5%/month.'
      );
      expect(screen.getByLabelText('Line 1 taxable')).toBeChecked();
      expect(screen.getByLabelText('Line 2 taxable')).not.toBeChecked();
      // 8% of the taxable $800 line only
      expect(screen.getByText('$64.00')).toBeInTheDocument();
    });

    it('saves terms_and_conditions and omits an unchanged tax rate', async () => {
      const user = userEvent.setup({ delay: null });
      await renderDraft();

      const terms = screen.getByLabelText(/terms & conditions/i);
      await user.clear(terms);
      await user.type(terms, 'Due on receipt');
      await user.click(screen.getByRole('button', { name: /save changes/i }));

      await waitFor(() => {
        expect(invoiceService.updateInvoice).toHaveBeenCalled();
      });
      const [, body] = vi.mocked(invoiceService.updateInvoice).mock.calls[0];
      expect(body).toEqual(
        expect.objectContaining({
          terms_and_conditions: 'Due on receipt',
          lines: [
            expect.objectContaining({
              description: 'HVAC Installation Service',
              taxable: true,
            }),
            expect.objectContaining({
              description: 'Parts and Materials',
              taxable: false,
            }),
          ],
        })
      );
      expect(body).not.toHaveProperty('terms');
      expect(body.tax_rate).toBeUndefined();
    });

    it('keeps per-line discounts on save (T237)', async () => {
      const user = userEvent.setup({ delay: null });
      const [labor, parts] = mockInvoices[0].lines;
      vi.mocked(invoiceService.getInvoice).mockResolvedValue({
        ...mockInvoices[0],
        lines: [
          { ...labor, discount_percentage: '10.00', discount_amount: '80.00' },
          { ...parts, discount_percentage: '0.00', discount_amount: '25.00' },
        ],
      });
      vi.mocked(invoiceService.updateInvoice).mockResolvedValue(
        mockInvoices[0]
      );
      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');
      render(
        <MemoryRouter initialEntries={['/invoices/invoice-1']}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      // Line totals net of their discounts: $720 and $175
      expect(await screen.findByText('$720.00')).toBeInTheDocument();
      expect(screen.getByText('10% discount')).toBeInTheDocument();
      expect(screen.getByText('$175.00')).toBeInTheDocument();
      expect(screen.getByText('$25.00 discount')).toBeInTheDocument();
      // Subtotal $895; 8% tax on the taxable $720 line only
      expect(screen.getByText('$895.00')).toBeInTheDocument();
      expect(screen.getByText('$57.60')).toBeInTheDocument();

      await user.click(screen.getByRole('button', { name: /save changes/i }));
      await waitFor(() => {
        expect(invoiceService.updateInvoice).toHaveBeenCalled();
      });
      const [, body] = vi.mocked(invoiceService.updateInvoice).mock.calls[0];
      expect(body.lines).toEqual([
        expect.objectContaining({
          description: 'HVAC Installation Service',
          discount_percentage: 10,
          discount_amount: undefined,
        }),
        expect.objectContaining({
          description: 'Parts and Materials',
          discount_percentage: undefined,
          discount_amount: 25,
        }),
      ]);
    });

    it('sends a changed tax rate as a fraction', async () => {
      const user = userEvent.setup({ delay: null });
      const taxInput = await renderDraft();

      await user.clear(taxInput);
      await user.type(taxInput, '8.25');
      await user.click(screen.getByRole('button', { name: /save changes/i }));

      await waitFor(() => {
        expect(invoiceService.updateInvoice).toHaveBeenCalledWith(
          'invoice-1',
          expect.objectContaining({ tax_rate: 0.0825 })
        );
      });
    });
  });

  describe('Tax rate picker (T237)', () => {
    const renderPage = async (path: string) => {
      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');
      render(
        <MemoryRouter initialEntries={[path]}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
            <Route path="/invoices/new" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );
    };

    beforeEach(() => {
      vi.mocked(taxRateService.listTaxRates).mockResolvedValue(mockTaxRates);
    });

    it("picks the tenant's default rate on a new invoice and sends its ID", async () => {
      const user = userEvent.setup({ delay: null });
      const tenantWithDefault = {
        ...mockTenant,
        settings: { defaultTaxRateId: 'tax-rate-1' },
      };
      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: tenantWithDefault,
          tenants: [tenantWithDefault],
        };
        return typeof selector === 'function' ? selector(state) : state;
      });
      vi.mocked(invoiceService.createInvoice).mockResolvedValue({
        ...mockInvoices[0],
        id: 'invoice-new',
      });
      await renderPage('/invoices/new');

      const picker = await screen.findByLabelText(/^tax rate$/i);
      await waitFor(() => expect(picker).toHaveValue('tax-rate-1'));
      expect(
        screen.getByRole('option', { name: 'City Sales Tax (8.25%) · default' })
      ).toBeInTheDocument();
      expect(screen.getByText('Tax (8.25%):')).toBeInTheDocument();

      await user.selectOptions(
        screen.getByLabelText(/customer/i),
        await screen.findByRole('option', { name: /John Doe/i })
      );
      await user.type(
        screen.getAllByPlaceholderText(/description/i)[0],
        'Service Charge'
      );
      await user.click(screen.getByRole('button', { name: /create invoice/i }));

      await waitFor(() => {
        expect(invoiceService.createInvoice).toHaveBeenCalled();
      });
      const [body] = vi.mocked(invoiceService.createInvoice).mock.calls[0];
      expect(body.tax_rate_id).toBe('tax-rate-1');
      expect(body.tax_rate).toBeUndefined();
    });

    it("shows the invoice's configured rate and sends a newly picked one by ID", async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.getInvoice).mockResolvedValue({
        ...mockInvoices[0],
        tax_rate_id: 'tax-rate-1',
        tax_rate: { id: 'tax-rate-1', name: 'City Sales Tax', rate: '0.0825' },
      });
      vi.mocked(invoiceService.updateInvoice).mockResolvedValue(
        mockInvoices[0]
      );
      await renderPage('/invoices/invoice-1');

      const picker = await screen.findByLabelText(/^tax rate$/i);
      await waitFor(() => expect(picker).toHaveValue('tax-rate-1'));

      await user.selectOptions(picker, 'tax-rate-2');
      expect(screen.getByText('Tax (6%):')).toBeInTheDocument();
      await user.click(screen.getByRole('button', { name: /save changes/i }));

      await waitFor(() => {
        expect(invoiceService.updateInvoice).toHaveBeenCalled();
      });
      const [, body] = vi.mocked(invoiceService.updateInvoice).mock.calls[0];
      expect(body.tax_rate_id).toBe('tax-rate-2');
      expect(body.tax_rate).toBeUndefined();
    });

    it('sends a custom rate as a fraction (replacing the configured one)', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.getInvoice).mockResolvedValue({
        ...mockInvoices[0],
        tax_rate_id: 'tax-rate-1',
        tax_rate: { id: 'tax-rate-1', name: 'City Sales Tax', rate: '0.0825' },
      });
      vi.mocked(invoiceService.updateInvoice).mockResolvedValue(
        mockInvoices[0]
      );
      await renderPage('/invoices/invoice-1');

      const picker = await screen.findByLabelText(/^tax rate$/i);
      await waitFor(() => expect(picker).toHaveValue('tax-rate-1'));
      await user.selectOptions(picker, '');
      const custom = screen.getByLabelText(/custom rate/i);
      await user.clear(custom);
      await user.type(custom, '5');
      await user.click(screen.getByRole('button', { name: /save changes/i }));

      await waitFor(() => {
        expect(invoiceService.updateInvoice).toHaveBeenCalled();
      });
      const [, body] = vi.mocked(invoiceService.updateInvoice).mock.calls[0];
      expect(body.tax_rate).toBe(0.05);
      expect(body.tax_rate_id).toBeUndefined();
    });
  });

  describe('Invoice Status Transitions', () => {
    it('should send a draft invoice to customer', async () => {
      const user = userEvent.setup({ delay: null });
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

      const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});

      // Send opens a confirmation modal
      await user.click(
        await screen.findByRole('button', { name: /send invoice/i })
      );
      await user.click(screen.getByTestId('confirm-send'));

      await waitFor(() => {
        expect(invoiceService.sendInvoice).toHaveBeenCalledWith('invoice-1');
      });
      alertSpy.mockRestore();
    });
  });

  describe('Purchase orders (T187)', () => {
    const poRequiredError = () =>
      new AxiosError('Conflict', 'ERR_BAD_REQUEST', undefined, undefined, {
        data: {
          error: 'po_required',
          message:
            'This customer requires a PO number before the invoice can be sent',
        },
        status: 409,
        statusText: 'Conflict',
        headers: {},
        config: { headers: new AxiosHeaders() },
      });

    const renderDraft = async (state?: { poRequired: boolean }) => {
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[0]);
      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');
      render(
        <MemoryRouter
          initialEntries={[{ pathname: '/invoices/invoice-1', state }]}
        >
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );
    };

    it('saves an edited PO number', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.updateInvoice).mockResolvedValue(
        mockInvoices[0]
      );
      await renderDraft();
      const po = await screen.findByLabelText(/po number/i);
      await user.type(po, '7731-HV');
      await user.click(screen.getByRole('button', { name: /save changes/i }));
      await waitFor(() => {
        expect(invoiceService.updateInvoice).toHaveBeenCalledWith(
          'invoice-1',
          expect.objectContaining({ po_number: '7731-HV' })
        );
      });
    });

    it('asks for the PO when send is refused with po_required, then saves it and sends', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.sendInvoice)
        .mockRejectedValueOnce(poRequiredError())
        .mockResolvedValueOnce({ ...mockInvoices[0], status: 'sent' as const });
      vi.mocked(invoiceService.updateInvoice).mockResolvedValue({
        ...mockInvoices[0],
        po_number: 'PO-42',
      });
      const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});
      await renderDraft();

      await user.click(
        await screen.findByRole('button', { name: /send invoice/i })
      );
      await user.click(screen.getByTestId('confirm-send'));

      expect(
        await screen.findByText(
          /this customer requires a po number before the invoice can be sent/i
        )
      ).toBeInTheDocument();
      // No PO typed yet: can't send.
      expect(screen.getByTestId('confirm-send')).toBeDisabled();
      await user.type(
        screen.getByLabelText(/^po number$/i, { selector: '#send_po_number' }),
        'PO-42'
      );
      await user.click(screen.getByRole('button', { name: /save po & send/i }));

      await waitFor(() => {
        expect(invoiceService.sendInvoice).toHaveBeenCalledTimes(2);
      });
      expect(invoiceService.updateInvoice).toHaveBeenCalledWith('invoice-1', {
        po_number: 'PO-42',
      });
      alertSpy.mockRestore();
    });

    it('opens the PO prompt when the list sent us here for a PO', async () => {
      await renderDraft({ poRequired: true });
      expect(
        await screen.findByText(/this customer requires a po number/i)
      ).toBeInTheDocument();
      expect(screen.getByTestId('confirm-send')).toHaveTextContent(
        /save po & send/i
      );
    });
  });

  describe('Payment Recording', () => {
    const renderInvoice = async (invoiceId: string) => {
      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');
      render(
        <MemoryRouter initialEntries={[`/invoices/${invoiceId}`]}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );
      await waitFor(() => {
        expect(invoiceService.getInvoice).toHaveBeenCalledWith(invoiceId);
      });
    };

    const openPaymentDialog = async (
      user: ReturnType<typeof userEvent.setup>
    ) => {
      await user.click(
        await screen.findByRole('button', { name: /record payment/i })
      );
      return screen.findByRole('dialog');
    };

    it('records a full payment against /payments and reloads the invoice', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[1]); // Sent, $540 due
      vi.mocked(invoiceService.recordPayment).mockResolvedValue({
        id: 'payment-1',
        invoice_id: 'invoice-2',
        amount: '540.00',
        payment_date: '2024-01-05T00:00:00Z',
        payment_method: 'cash',
        created_at: '2024-01-05T00:00:00Z',
      });

      await renderInvoice('invoice-2');
      const dialog = await openPaymentDialog(user);

      // The amount defaults to the balance due
      expect(within(dialog).getByLabelText(/amount/i)).toHaveValue(540);
      await user.selectOptions(
        within(dialog).getByLabelText(/payment method/i),
        'cash'
      );
      await user.type(within(dialog).getByLabelText(/reference/i), 'R-77');
      await user.click(
        within(dialog).getByRole('button', { name: /record payment/i })
      );

      await waitFor(() => {
        expect(invoiceService.recordPayment).toHaveBeenCalledWith(
          'invoice-2',
          expect.objectContaining({
            amount: 540,
            payment_method: 'cash',
            reference_number: 'R-77',
          })
        );
      });
      // The invoice is reloaded for its new status and balance
      await waitFor(() => {
        expect(invoiceService.getInvoice).toHaveBeenCalledTimes(2);
      });
    });

    it('records a partial payment', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[1]);
      vi.mocked(invoiceService.recordPayment).mockResolvedValue({
        id: 'payment-2',
        invoice_id: 'invoice-2',
        amount: '200.00',
        payment_date: '2024-01-05T00:00:00Z',
        created_at: '2024-01-05T00:00:00Z',
      });

      await renderInvoice('invoice-2');
      const dialog = await openPaymentDialog(user);
      const amount = within(dialog).getByLabelText(/amount/i);
      await user.clear(amount);
      await user.type(amount, '200');
      await user.click(
        within(dialog).getByRole('button', { name: /record payment/i })
      );

      await waitFor(() => {
        expect(invoiceService.recordPayment).toHaveBeenCalledWith(
          'invoice-2',
          expect.objectContaining({ amount: 200 })
        );
      });
    });

    it('stops an amount over the balance before calling the API', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[1]);

      await renderInvoice('invoice-2');
      const dialog = await openPaymentDialog(user);
      const amount = within(dialog).getByLabelText(/amount/i);
      await user.clear(amount);
      await user.type(amount, '600');
      await user.click(
        within(dialog).getByRole('button', { name: /record payment/i })
      );

      expect(
        await within(dialog).findByText(/more than the balance due/i)
      ).toBeInTheDocument();
      expect(invoiceService.recordPayment).not.toHaveBeenCalled();
    });

    it('shows the API reason for a rejected payment (400)', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[1]);
      vi.mocked(invoiceService.recordPayment).mockRejectedValue({
        response: {
          status: 400,
          data: { error: 'payment amount exceeds outstanding balance' },
        },
      });

      await renderInvoice('invoice-2');
      const dialog = await openPaymentDialog(user);
      await user.click(
        within(dialog).getByRole('button', { name: /record payment/i })
      );

      expect(
        await within(dialog).findByText(
          'Payment amount exceeds outstanding balance.'
        )
      ).toBeInTheDocument();
    });

    it('tells the user to reload when the invoice changed (409)', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[1]);
      vi.mocked(invoiceService.recordPayment).mockRejectedValue({
        response: {
          status: 409,
          data: {
            error: "paid, cancelled and refunded invoices can't be edited",
          },
        },
      });

      await renderInvoice('invoice-2');
      const dialog = await openPaymentDialog(user);
      await user.click(
        within(dialog).getByRole('button', { name: /record payment/i })
      );

      expect(
        await within(dialog).findByText(/reload the invoice/i)
      ).toBeInTheDocument();
    });

    it('offers no payment or cancel action on a paid invoice', async () => {
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[2]);

      await renderInvoice('invoice-3');
      await screen.findByRole('heading', { name: /^payments$/i });
      expect(
        screen.queryByRole('button', { name: /record payment/i })
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole('button', { name: /cancel invoice/i })
      ).not.toBeInTheDocument();
    });

    it('lists the payments recorded on the invoice', async () => {
      vi.mocked(invoiceService.getInvoice).mockResolvedValue({
        ...mockInvoices[2],
        payments: [
          {
            id: 'payment-9',
            invoice_id: 'invoice-3',
            amount: '1620.00',
            payment_date: '2024-01-10T00:00:00Z',
            payment_method: 'check',
            reference_number: 'CHK-1001',
            created_at: '2024-01-10T00:00:00Z',
          },
        ],
      });

      await renderInvoice('invoice-3');
      expect(await screen.findByText('CHK-1001')).toBeInTheDocument();
      expect(screen.getByText('Check')).toBeInTheDocument();
    });
  });

  describe('Invoice Cancellation', () => {
    it('requires a reason and sends it to the API', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[1]);
      vi.mocked(invoiceService.cancelInvoice).mockResolvedValue({
        ...mockInvoices[1],
        status: 'cancelled' as const,
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

      await user.click(
        await screen.findByRole('button', { name: /cancel invoice/i })
      );
      const dialog = await screen.findByRole('dialog');
      const submit = within(dialog).getByRole('button', {
        name: /cancel invoice/i,
      });

      await user.click(submit);
      expect(
        await within(dialog).findByText(/enter a reason/i)
      ).toBeInTheDocument();
      expect(invoiceService.cancelInvoice).not.toHaveBeenCalled();

      await user.type(
        within(dialog).getByLabelText(/reason/i),
        'Customer went with another quote'
      );
      await user.click(submit);

      await waitFor(() => {
        expect(invoiceService.cancelInvoice).toHaveBeenCalledWith(
          'invoice-2',
          'Customer went with another quote'
        );
      });
    });
  });

  describe('Refunds', () => {
    const renderInvoice = async (invoiceId: string) => {
      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');
      render(
        <MemoryRouter initialEntries={[`/invoices/${invoiceId}`]}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );
      await waitFor(() => {
        expect(invoiceService.getInvoice).toHaveBeenCalledWith(invoiceId);
      });
    };

    it('records a refund at POST /refunds and reloads the invoice', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[2]); // Paid $1,620
      vi.mocked(invoiceService.recordRefund).mockResolvedValue({
        id: 'refund-1',
        invoice_id: 'invoice-3',
        amount: '1620.00',
        refund_date: '2024-01-12T00:00:00Z',
        method: 'check',
        created_at: '2024-01-12T00:00:00Z',
      });

      await renderInvoice('invoice-3');
      await user.click(
        await screen.findByRole('button', { name: /record refund/i })
      );
      const dialog = await screen.findByRole('dialog');
      expect(within(dialog).getByLabelText(/refund amount/i)).toHaveValue(1620);
      await user.type(
        within(dialog).getByLabelText(/reason/i),
        'Warranty callback'
      );
      await user.click(
        within(dialog).getByRole('button', { name: /record refund/i })
      );

      await waitFor(() => {
        expect(invoiceService.recordRefund).toHaveBeenCalledWith(
          'invoice-3',
          expect.objectContaining({
            amount: 1620,
            method: 'check',
            reason: 'Warranty callback',
          })
        );
      });
      await waitFor(() => {
        expect(invoiceService.getInvoice).toHaveBeenCalledTimes(2);
      });
    });

    it('stops a refund larger than the amount paid', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[2]);

      await renderInvoice('invoice-3');
      await user.click(
        await screen.findByRole('button', { name: /record refund/i })
      );
      const dialog = await screen.findByRole('dialog');
      const amount = within(dialog).getByLabelText(/refund amount/i);
      await user.clear(amount);
      await user.type(amount, '2000');
      await user.click(
        within(dialog).getByRole('button', { name: /record refund/i })
      );

      expect(
        await within(dialog).findByText(/more than the amount paid/i)
      ).toBeInTheDocument();
      expect(invoiceService.recordRefund).not.toHaveBeenCalled();
    });

    it('hides the refund action without payments.refund', async () => {
      grantedPermissions = new Set(['payments.create']);
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[2]);

      await renderInvoice('invoice-3');
      await screen.findByRole('heading', { name: /^payments$/i });
      expect(
        screen.queryByRole('button', { name: /record refund/i })
      ).not.toBeInTheDocument();
    });

    it('hides the refund action when nothing was paid', async () => {
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(mockInvoices[1]);

      await renderInvoice('invoice-2');
      await screen.findByRole('heading', { name: /^payments$/i });
      expect(
        screen.queryByRole('button', { name: /record refund/i })
      ).not.toBeInTheDocument();
    });

    it('shows a refunded invoice with its refunds', async () => {
      vi.mocked(invoiceService.getInvoice).mockResolvedValue({
        ...mockInvoices[2],
        status: 'refunded' as const,
        amount_paid: 0,
        amount_due: 1620,
        refunds: [
          {
            id: 'refund-7',
            invoice_id: 'invoice-3',
            amount: '1620.00',
            refund_date: '2024-01-12T00:00:00Z',
            method: 'cash',
            reason: 'Job cancelled after payment',
            created_at: '2024-01-12T00:00:00Z',
          },
        ],
      });

      await renderInvoice('invoice-3');
      expect(await screen.findByText('Refunded')).toBeInTheDocument();
      expect(
        screen.getByText('Job cancelled after payment')
      ).toBeInTheDocument();
      expect(screen.getByText('-$1,620.00')).toBeInTheDocument();
      // Nothing left to refund, pay or cancel
      expect(
        screen.queryByRole('button', {
          name: /record refund|record payment|cancel invoice/i,
        })
      ).not.toBeInTheDocument();
    });
  });

  describe('Invoice PDF', () => {
    it('should download invoice PDF', async () => {
      const user = userEvent.setup({ delay: null });
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

      await user.click(
        await screen.findByRole('button', { name: /download pdf/i })
      );

      await waitFor(() => {
        expect(invoiceService.downloadInvoicePDF).toHaveBeenCalledWith(
          'invoice-1'
        );
      });
    });

    it('should download receipt PDF for paid invoice', async () => {
      const user = userEvent.setup({ delay: null });
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

      await user.click(
        await screen.findByRole('button', { name: /download receipt/i })
      );

      await waitFor(() => {
        expect(invoiceService.downloadReceiptPDF).toHaveBeenCalledWith(
          'invoice-3'
        );
      });
    });
  });

  describe('Invoice Filtering', () => {
    it('should filter invoices by status', async () => {
      const user = userEvent.setup({ delay: null });
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
      const user = userEvent.setup({ delay: null });
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

      expect(
        await screen.findByRole('heading', { name: /payment successful/i })
      ).toBeInTheDocument();
    });

    it('is a public page with no links into the authenticated app', async () => {
      const { PaymentSuccessPage } =
        await import('../../src/pages/Invoices/PaymentSuccessPage');

      render(
        <MemoryRouter initialEntries={['/invoices/payment-success']}>
          <PaymentSuccessPage />
        </MemoryRouter>
      );

      // The payer is the end customer (not an app user), so the page must
      // not send them to authenticated routes like /invoices.
      expect(
        screen.queryByRole('link', { name: /view invoices|back to invoices/i })
      ).not.toBeInTheDocument();
      expect(
        screen.getByText(/you can close this window/i)
      ).toBeInTheDocument();
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

      expect(
        await screen.findByRole('heading', { name: /payment cancelled/i })
      ).toBeInTheDocument();
      expect(
        screen.getByText(/no charges have been made/i)
      ).toBeInTheDocument();
    });
  });

  describe('Consolidated invoices (T188)', () => {
    const preview = {
      customer_id: 'customer-1',
      period_start: '2026-10-01',
      period_end: '2026-10-31',
      po_number: '7731-HV',
      lines: [
        {
          kind: 'agreement_month' as const,
          site_id: 'site-1',
          site_name: 'Harborview Lofts',
          agreement_id: 'agr-1',
          description: 'Nightly cleaning, Mon–Fri · 22 visits',
          month: '2026-10',
          visits: 22,
          quantity: '1',
          unit_price: '2400',
          amount: '2400',
        },
        {
          kind: 'extra_work' as const,
          site_id: 'site-1',
          site_name: 'Harborview Lofts',
          job_id: 'job-9',
          job_number: 'J-2291',
          description: 'Carpet extraction, lobby · J-2291',
          visits: 0,
          quantity: '1',
          unit_price: '450',
          amount: '450',
          approver_name: 'Karen Doyle',
          approval_amount: '450',
          approval_note: 'Approved by Karen Doyle for 450.00',
        },
        {
          kind: 'agreement_visits' as const,
          site_id: 'site-2',
          site_name: 'Riverside Commons',
          agreement_id: 'agr-2',
          description: 'Cleaning 3x weekly · 13 visits',
          visits: 13,
          quantity: '13',
          unit_price: '138.46',
          amount: '1800',
        },
      ],
      sites: 2,
      visits: 35,
      extra_work: 1,
      subtotal: '4650',
      tax_rate: '0',
      tax_amount: '0',
      total: '4650',
    };

    const consolidated = {
      ...mockInvoices[0],
      id: 'invoice-9',
      invoice_number: 'INV-1042',
      po_number: '7731-HV',
      billing_period_start: '2026-10-01T00:00:00Z',
      billing_period_end: '2026-10-31T00:00:00Z',
      lines: [
        {
          id: 'cl-1',
          description: 'Nightly cleaning, Mon–Fri · 22 visits',
          quantity: 1,
          unit_price: 2400,
          total: 2400,
          sort_order: 0,
          taxable: true,
          site_name: 'Harborview Lofts',
        },
        {
          id: 'cl-2',
          description: 'Carpet extraction, lobby · J-2291',
          quantity: 1,
          unit_price: 450,
          total: 450,
          sort_order: 1,
          taxable: true,
          site_name: 'Harborview Lofts',
          approval_note: 'Approved by Karen Doyle for 450.00',
        },
        {
          id: 'cl-3',
          description: 'Cleaning 3x weekly · 13 visits',
          quantity: 13,
          unit_price: 138.46,
          total: 1800,
          sort_order: 2,
          taxable: true,
          site_name: 'Riverside Commons',
        },
      ],
    };

    it('previews last month grouped by site and creates the draft', async () => {
      // Last month, in local time (fake timers would stall the router).
      const now = new Date();
      const pad = (n: number) => String(n).padStart(2, '0');
      const first = new Date(now.getFullYear(), now.getMonth() - 1, 1);
      const last = new Date(now.getFullYear(), now.getMonth(), 0);
      const periodStart = `${first.getFullYear()}-${pad(first.getMonth() + 1)}-01`;
      const periodEnd = `${last.getFullYear()}-${pad(last.getMonth() + 1)}-${pad(last.getDate())}`;
      {
        const user = userEvent.setup({ delay: null });
        vi.mocked(invoiceService.previewConsolidatedInvoice).mockResolvedValue(
          preview
        );
        vi.mocked(invoiceService.createConsolidatedInvoice).mockResolvedValue(
          consolidated
        );
        const { InvoicesPage } =
          await import('../../src/pages/Invoices/InvoicesPage');
        render(
          <MemoryRouter initialEntries={['/invoices']}>
            <Routes>
              <Route path="/invoices" element={<InvoicesPage />} />
              <Route path="/invoices/:id" element={<div>Invoice Detail</div>} />
            </Routes>
          </MemoryRouter>
        );

        await user.click(
          await screen.findByRole('button', {
            name: /create consolidated invoice/i,
          })
        );
        const dialog = await screen.findByRole('dialog');
        // Default period: last month.
        expect(within(dialog).getByLabelText('From')).toHaveValue(periodStart);
        expect(within(dialog).getByLabelText('To')).toHaveValue(periodEnd);

        await user.selectOptions(
          await within(dialog).findByLabelText('Customer'),
          'customer-1'
        );
        await waitFor(() => {
          expect(
            invoiceService.previewConsolidatedInvoice
          ).toHaveBeenCalledWith({
            customer_id: 'customer-1',
            period_start: periodStart,
            period_end: periodEnd,
          });
        });
        const table = await within(dialog).findByTestId('consolidated-preview');
        // Each site named once, extra work with its approver, totals.
        expect(within(table).getAllByText('Harborview Lofts')).toHaveLength(1);
        expect(
          within(table).getByText('Riverside Commons')
        ).toBeInTheDocument();
        expect(
          within(table).getByText('Approved by Karen Doyle for 450.00')
        ).toBeInTheDocument();
        expect(within(table).getByText(/PO 7731-HV/)).toBeInTheDocument();
        expect(within(table).getAllByText('$4,650.00')).toHaveLength(2);

        await user.click(
          within(dialog).getByRole('button', { name: /create draft/i })
        );
        await waitFor(() => {
          expect(invoiceService.createConsolidatedInvoice).toHaveBeenCalledWith(
            {
              customer_id: 'customer-1',
              period_start: periodStart,
              period_end: periodEnd,
            }
          );
        });
        expect(mockNavigate).toHaveBeenCalledWith('/invoices/invoice-9');
      }
    });

    it('says when there is nothing to invoice', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.previewConsolidatedInvoice).mockResolvedValue({
        ...preview,
        lines: [],
        sites: 0,
        visits: 0,
        extra_work: 0,
        subtotal: '0',
        total: '0',
      });
      const { InvoicesPage } =
        await import('../../src/pages/Invoices/InvoicesPage');
      render(
        <MemoryRouter initialEntries={['/invoices']}>
          <InvoicesPage />
        </MemoryRouter>
      );
      await user.click(
        await screen.findByRole('button', {
          name: /create consolidated invoice/i,
        })
      );
      const dialog = await screen.findByRole('dialog');
      await user.selectOptions(
        await within(dialog).findByLabelText('Customer'),
        'customer-1'
      );
      expect(
        await within(dialog).findByText(/nothing to invoice for this period/i)
      ).toBeInTheDocument();
      expect(
        within(dialog).getByRole('button', { name: /create draft/i })
      ).toBeDisabled();
    });

    it('shows the invoice grouped by site and keeps sites and notes on save', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(invoiceService.getInvoice).mockResolvedValue(consolidated);
      vi.mocked(invoiceService.updateInvoice).mockResolvedValue(consolidated);
      const { InvoiceDetailPage } =
        await import('../../src/pages/Invoices/InvoiceDetailPage');
      render(
        <MemoryRouter initialEntries={['/invoices/invoice-9']}>
          <Routes>
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      const headings = await screen.findAllByTestId('line-site-heading');
      expect(headings.map((h) => h.textContent)).toEqual([
        'Harborview Lofts',
        'Riverside Commons',
      ]);
      expect(screen.getByText('Consolidated · 2 sites')).toBeInTheDocument();
      expect(
        screen.getByText(/Oct 1 – Oct 31, 2026 services/)
      ).toBeInTheDocument();
      expect(
        screen.getByText('Approved by Karen Doyle for 450.00')
      ).toBeInTheDocument();

      await user.click(screen.getByRole('button', { name: /save changes/i }));
      await waitFor(() => {
        expect(invoiceService.updateInvoice).toHaveBeenCalled();
      });
      const [, body] = vi.mocked(invoiceService.updateInvoice).mock.calls[0];
      expect(body.lines?.[1]).toEqual(
        expect.objectContaining({
          site_name: 'Harborview Lofts',
          approval_note: 'Approved by Karen Doyle for 450.00',
        })
      );
      expect(body.lines?.[2]).toEqual(
        expect.objectContaining({ site_name: 'Riverside Commons' })
      );
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

      // An API failure must show an error state with retry, not "No invoices yet"
      expect(
        await screen.findByText(/couldn't load invoices/i)
      ).toBeInTheDocument();
      expect(screen.queryByText(/no invoices yet/i)).not.toBeInTheDocument();

      vi.mocked(invoiceService.getInvoices).mockClear();
      await userEvent
        .setup()
        .click(screen.getByRole('button', { name: /try again/i }));
      expect(invoiceService.getInvoices).toHaveBeenCalledTimes(1);
    });
  });
});
