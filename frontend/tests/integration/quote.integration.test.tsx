/**
 * Quote Workflow Integration Tests
 *
 * Tests complete quote management workflows including:
 * - Quote list loading and display
 * - Quote creation with line items
 * - Quote status transitions (draft → sent → accepted/declined)
 * - Quote PDF generation
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router';
import React from 'react';

// Mock modules
vi.mock('@store', () => ({
  useAuthStore: vi.fn(),
  useTenantStore: vi.fn(),
  useSidebarStore: vi.fn(),
}));

vi.mock('@services/quoteService', () => ({
  quoteService: {
    getQuotes: vi.fn(),
    getQuote: vi.fn(),
    createQuote: vi.fn(),
    updateQuote: vi.fn(),
    deleteQuote: vi.fn(),
    sendQuote: vi.fn(),
    acceptQuote: vi.fn(),
    rejectQuote: vi.fn(),
    reviseQuote: vi.fn(),
    getStats: vi.fn(),
    downloadQuotePDF: vi.fn(),
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
import { quoteService } from '@services/quoteService';
import { customerService } from '@services/customerService';
import { taxRateService, type TaxRate } from '@services/taxRateService';

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
    name: 'Reduced Rate',
    rate: '0.08',
    tax_type: 'sales_tax',
    is_active: true,
  },
];

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

const mockQuotes = [
  {
    id: 'quote-1',
    quote_number: 'QT-001',
    customer_id: 'customer-1',
    customer: mockCustomer,
    status: 'draft' as const,
    subtotal: 1000,
    tax_rate: 0.08,
    tax_amount: 80,
    total: 1080,
    valid_until: '2024-12-31T00:00:00Z',
    items: [
      {
        id: 'line-1',
        description: 'HVAC Installation Service',
        quantity: 1,
        unit_price: 800,
        total: 800,
      },
      {
        id: 'line-2',
        description: 'Parts and Materials',
        quantity: 1,
        unit_price: 200,
        total: 200,
      },
    ],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'quote-2',
    quote_number: 'QT-002',
    customer_id: 'customer-1',
    customer: mockCustomer,
    status: 'sent' as const,
    subtotal: 500,
    tax_rate: 0.08,
    tax_amount: 40,
    total: 540,
    valid_until: '2024-12-31T00:00:00Z',
    items: [
      {
        id: 'line-3',
        description: 'Plumbing Repair',
        quantity: 1,
        unit_price: 500,
        total: 500,
      },
    ],
    sent_at: '2024-01-02T00:00:00Z',
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
  },
  {
    id: 'quote-3',
    quote_number: 'QT-003',
    customer_id: 'customer-1',
    customer: mockCustomer,
    status: 'accepted' as const,
    subtotal: 1500,
    tax_rate: 0.08,
    tax_amount: 120,
    total: 1620,
    valid_until: '2024-12-31T00:00:00Z',
    items: [
      {
        id: 'line-4',
        description: 'Electrical Work',
        quantity: 1,
        unit_price: 1500,
        total: 1500,
      },
    ],
    sent_at: '2024-01-03T00:00:00Z',
    accepted_at: '2024-01-04T00:00:00Z',
    created_at: '2024-01-03T00:00:00Z',
    updated_at: '2024-01-04T00:00:00Z',
  },
];

const mockQuoteStats = {
  total_quotes: 3,
  draft_quotes: 1,
  sent_quotes: 1,
  accepted_quotes: 1,
  declined_quotes: 0,
  expired_quotes: 0,
  total_value: 3240,
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

  vi.mocked(quoteService.getQuotes).mockResolvedValue({
    quotes: mockQuotes,
    total: mockQuotes.length,
    page: 1,
    page_size: 10,
  });

  vi.mocked(quoteService.getStats).mockResolvedValue(mockQuoteStats);

  // No configured tax rates unless a test sets some (typed-rate input only)
  vi.mocked(taxRateService.listTaxRates).mockResolvedValue([]);

  vi.mocked(customerService.getCustomers).mockResolvedValue({
    customers: [mockCustomer],
    total: 1,
    page: 1,
    page_size: 10,
  });
};

// =============================================================================
// Quote List Tests
// =============================================================================

describe('Quote Workflow Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupMocks();
  });

  describe('Quote List', () => {
    it('should load and display quotes list', async () => {
      const { QuotesPage } = await import('../../src/pages/Quotes/QuotesPage');

      render(
        <MemoryRouter initialEntries={['/quotes']}>
          <QuotesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenCalled();
      });

      // Verify quotes are displayed
      await waitFor(() => {
        expect(screen.getByText('QT-001')).toBeInTheDocument();
        expect(screen.getByText('QT-002')).toBeInTheDocument();
      });
    });

    it('should display quote amounts correctly', async () => {
      const { QuotesPage } = await import('../../src/pages/Quotes/QuotesPage');

      render(
        <MemoryRouter initialEntries={['/quotes']}>
          <QuotesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        // Check for formatted amounts
        expect(
          screen.getByText(/\$1,080|\$1080|1,080\.00/)
        ).toBeInTheDocument();
      });
    });

    it('should display quote status badges', async () => {
      const { QuotesPage } = await import('../../src/pages/Quotes/QuotesPage');

      render(
        <MemoryRouter initialEntries={['/quotes']}>
          <QuotesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByText(/draft/i)).toBeInTheDocument();
        expect(screen.getByText(/sent/i)).toBeInTheDocument();
        expect(screen.getByText(/accepted/i)).toBeInTheDocument();
      });
    });

    it('should navigate to quote detail when clicking a quote', async () => {
      const user = userEvent.setup({ delay: null });
      const { QuotesPage } = await import('../../src/pages/Quotes/QuotesPage');

      render(
        <MemoryRouter initialEntries={['/quotes']}>
          <QuotesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('QT-001')).toBeInTheDocument();
      });

      const quoteRow = screen.getByText('QT-001').closest('tr');
      if (quoteRow) {
        await user.click(quoteRow);
        expect(mockNavigate).toHaveBeenCalledWith('/quotes/quote-1');
      }
    });

    it('should navigate to new quote page when clicking add button', async () => {
      const user = userEvent.setup({ delay: null });
      const { QuotesPage } = await import('../../src/pages/Quotes/QuotesPage');

      render(
        <MemoryRouter initialEntries={['/quotes']}>
          <QuotesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenCalled();
      });

      const addButton = screen.getByRole('button', { name: /add|new|create/i });
      await user.click(addButton);

      expect(mockNavigate).toHaveBeenCalledWith('/quotes/new');
    });
  });

  describe('Quote Creation', () => {
    it('should create a new quote with line items', async () => {
      const user = userEvent.setup({ delay: null });
      const newQuote = {
        ...mockQuotes[0],
        id: 'quote-new',
        quote_number: 'QT-004',
      };

      vi.mocked(quoteService.createQuote).mockResolvedValue(newQuote);
      vi.mocked(customerService.getCustomer).mockResolvedValue(mockCustomer);

      const { QuoteDetailPage } =
        await import('../../src/pages/Quotes/QuoteDetailPage');

      render(
        <MemoryRouter initialEntries={['/quotes/new']}>
          <Routes>
            <Route path="/quotes/new" element={<QuoteDetailPage />} />
            <Route path="/quotes/:id" element={<div>Quote Detail</div>} />
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
        'Test Service'
      );
      const qty = screen.getAllByPlaceholderText(/qty/i)[0];
      await user.clear(qty);
      await user.type(qty, '2');
      const price = screen.getAllByPlaceholderText(/price/i)[0];
      await user.clear(price);
      await user.type(price, '250');

      // Submit form
      const saveButton = screen.getByRole('button', {
        name: /save|create|submit/i,
      });
      await user.click(saveButton);

      await waitFor(() => {
        expect(quoteService.createQuote).toHaveBeenCalledWith(
          expect.objectContaining({
            customer_id: 'customer-1',
            items: [
              expect.objectContaining({
                description: 'Test Service',
                quantity: 2,
                unit_price: 250,
              }),
            ],
          })
        );
      });
    });
  });

  describe('Tenant default tax rate on a new quote (T127)', () => {
    it('leaves the rate blank and omits it so the API applies the default', async () => {
      const user = userEvent.setup({ delay: null });
      const tenantWithDefaults = {
        ...mockTenant,
        settings: { defaultTaxRateId: 'tax-rate-1' },
      };
      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: tenantWithDefaults,
          tenants: [tenantWithDefaults],
        };
        return typeof selector === 'function' ? selector(state) : state;
      });
      vi.mocked(quoteService.createQuote).mockResolvedValue(mockQuotes[0]);

      const { QuoteDetailPage } =
        await import('../../src/pages/Quotes/QuoteDetailPage');
      render(
        <MemoryRouter initialEntries={['/quotes/new']}>
          <Routes>
            <Route path="/quotes/new" element={<QuoteDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      const taxInput = await screen.findByLabelText(/tax rate/i);
      await waitFor(() => expect(taxInput).toHaveValue(null));
      expect(
        screen.getByText(/left blank, your default tax rate applies/i)
      ).toBeInTheDocument();

      const customerSelect = screen.getByLabelText(/customer/i);
      await screen.findByRole('option', { name: /John Doe/i });
      await user.selectOptions(customerSelect, 'customer-1');
      await user.type(
        screen.getAllByPlaceholderText(/description/i)[0],
        'Service Charge'
      );
      await user.click(screen.getByRole('button', { name: /create quote/i }));

      await waitFor(() => {
        expect(quoteService.createQuote).toHaveBeenCalled();
      });
      const [body] = vi.mocked(quoteService.createQuote).mock.calls[0];
      expect(body.tax_rate).toBeUndefined();
    });
  });

  describe('Tax rate picker (T237)', () => {
    beforeEach(() => {
      vi.mocked(taxRateService.listTaxRates).mockResolvedValue(mockTaxRates);
    });

    it("picks the tenant's default rate on a new quote and sends its rate", async () => {
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
      vi.mocked(quoteService.createQuote).mockResolvedValue(mockQuotes[0]);

      const { QuoteDetailPage } =
        await import('../../src/pages/Quotes/QuoteDetailPage');
      render(
        <MemoryRouter initialEntries={['/quotes/new']}>
          <Routes>
            <Route path="/quotes/new" element={<QuoteDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      const picker = await screen.findByLabelText(/^tax rate$/i);
      await waitFor(() => expect(picker).toHaveValue('tax-rate-1'));
      expect(screen.getByText('Tax (8.25%):')).toBeInTheDocument();

      await user.selectOptions(
        screen.getByLabelText(/customer/i),
        await screen.findByRole('option', { name: /John Doe/i })
      );
      await user.type(
        screen.getAllByPlaceholderText(/description/i)[0],
        'Service Charge'
      );
      await user.click(screen.getByRole('button', { name: /create quote/i }));

      await waitFor(() => {
        expect(quoteService.createQuote).toHaveBeenCalled();
      });
      const [body] = vi.mocked(quoteService.createQuote).mock.calls[0];
      expect(body.tax_rate).toBe(0.0825);
    });

    it("shows the configured rate matching a quote's rate and saves a newly picked one", async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(quoteService.getQuote).mockResolvedValue(mockQuotes[0]); // 8%
      vi.mocked(quoteService.updateQuote).mockResolvedValue(mockQuotes[0]);

      const { QuoteDetailPage } =
        await import('../../src/pages/Quotes/QuoteDetailPage');
      render(
        <MemoryRouter initialEntries={['/quotes/quote-1']}>
          <Routes>
            <Route path="/quotes/:id" element={<QuoteDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      const picker = await screen.findByLabelText(/^tax rate$/i);
      await waitFor(() => expect(picker).toHaveValue('tax-rate-2'));

      await user.selectOptions(picker, 'tax-rate-1');
      await user.click(screen.getByRole('button', { name: /save changes/i }));

      await waitFor(() => {
        expect(quoteService.updateQuote).toHaveBeenCalledWith(
          'quote-1',
          expect.objectContaining({ tax_rate: 0.0825 })
        );
      });
    });
  });

  describe('Quote Status Transitions', () => {
    const renderQuote = async (quoteId: string) => {
      const { QuoteDetailPage } =
        await import('../../src/pages/Quotes/QuoteDetailPage');
      render(
        <MemoryRouter initialEntries={[`/quotes/${quoteId}`]}>
          <Routes>
            <Route path="/quotes/:id" element={<QuoteDetailPage />} />
          </Routes>
        </MemoryRouter>
      );
      await waitFor(() => {
        expect(quoteService.getQuote).toHaveBeenCalledWith(quoteId);
      });
    };

    it('shows the quote number and status', async () => {
      vi.mocked(quoteService.getQuote).mockResolvedValue({
        ...mockQuotes[1],
        quote_number: 'QUO-000042',
      });

      await renderQuote('quote-2');
      expect(
        await screen.findByRole('heading', { name: 'Quote QUO-000042' })
      ).toBeInTheDocument();
      expect(screen.getByText('Sent')).toBeInTheDocument();
    });

    it('sends a draft quote and shows the status the API returned', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(quoteService.getQuote).mockResolvedValue(mockQuotes[0]); // Draft quote
      vi.mocked(quoteService.sendQuote).mockResolvedValue({
        ...mockQuotes[0],
        status: 'sent' as const,
      });

      await renderQuote('quote-1');
      await user.click(
        await screen.findByRole('button', { name: /send quote/i })
      );

      await waitFor(() => {
        expect(quoteService.sendQuote).toHaveBeenCalledWith('quote-1');
      });
      expect(await screen.findByText('Sent')).toBeInTheDocument();
      // A sent quote is read-only until revised
      expect(
        screen.getByRole('button', { name: /save changes/i })
      ).toBeDisabled();
      expect(
        screen.getByRole('button', { name: /send quote/i })
      ).toBeDisabled();
    });

    it('accepts a sent quote', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(quoteService.getQuote).mockResolvedValue(mockQuotes[1]); // Sent quote
      vi.mocked(quoteService.acceptQuote).mockResolvedValue({
        ...mockQuotes[1],
        status: 'accepted' as const,
      });

      await renderQuote('quote-2');
      await user.click(
        await screen.findByRole('button', { name: /^accept$/i })
      );

      await waitFor(() => {
        expect(quoteService.acceptQuote).toHaveBeenCalledWith('quote-2');
      });
      expect(await screen.findByText('Accepted')).toBeInTheDocument();
      // Accepted quotes are final: no accept/decline/revise
      expect(screen.queryByRole('button', { name: /^accept$/i })).toBeNull();
      expect(screen.queryByRole('button', { name: /decline/i })).toBeNull();
      expect(screen.queryByRole('button', { name: /revise/i })).toBeNull();
    });

    it('declines a sent quote', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(quoteService.getQuote).mockResolvedValue(mockQuotes[1]); // Sent quote
      vi.mocked(quoteService.rejectQuote).mockResolvedValue({
        ...mockQuotes[1],
        status: 'declined' as const,
      });

      await renderQuote('quote-2');
      await user.click(await screen.findByRole('button', { name: /decline/i }));

      await waitFor(() => {
        expect(quoteService.rejectQuote).toHaveBeenCalledWith('quote-2');
      });
      expect(await screen.findByText('Declined')).toBeInTheDocument();
      // A declined quote can be revised and sent again
      expect(screen.getByRole('button', { name: /revise/i })).toBeEnabled();
    });

    it("shows the API's reason when accepting is refused", async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(quoteService.getQuote).mockResolvedValue(mockQuotes[1]);
      vi.mocked(quoteService.acceptQuote).mockRejectedValue({
        response: {
          status: 403,
          data: {
            error: 'cannot_accept',
            message: 'Quote cannot be accepted in current status',
          },
        },
      });

      await renderQuote('quote-2');
      await user.click(
        await screen.findByRole('button', { name: /^accept$/i })
      );

      expect(
        await screen.findByText('Quote cannot be accepted in current status')
      ).toBeInTheDocument();
      expect(screen.getByText('Sent')).toBeInTheDocument();
    });

    it('should keep a sent quote read-only until it is revised', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(quoteService.getQuote).mockResolvedValue(mockQuotes[1]); // Sent quote
      vi.mocked(quoteService.reviseQuote).mockResolvedValue({
        ...mockQuotes[1],
        status: 'draft' as const,
      });

      await renderQuote('quote-2');

      const saveButton = await screen.findByRole('button', {
        name: /save changes/i,
      });
      await waitFor(() => expect(saveButton).toBeDisabled());

      await user.click(screen.getByRole('button', { name: /revise/i }));

      await waitFor(() => {
        expect(quoteService.reviseQuote).toHaveBeenCalledWith('quote-2');
      });
      await waitFor(() => expect(saveButton).toBeEnabled());
      expect(screen.queryByRole('button', { name: /revise/i })).toBeNull();
      expect(screen.getByText('Draft')).toBeInTheDocument();
    });
  });

  describe('Quote PDF', () => {
    it('should download quote PDF', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(quoteService.getQuote).mockResolvedValue(mockQuotes[0]);
      vi.mocked(quoteService.downloadQuotePDF).mockResolvedValue();

      const { QuoteDetailPage } =
        await import('../../src/pages/Quotes/QuoteDetailPage');

      render(
        <MemoryRouter initialEntries={['/quotes/quote-1']}>
          <Routes>
            <Route path="/quotes/:id" element={<QuoteDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(quoteService.getQuote).toHaveBeenCalled();
      });

      await user.click(
        await screen.findByRole('button', { name: /pdf|download/i })
      );

      await waitFor(() => {
        expect(quoteService.downloadQuotePDF).toHaveBeenCalledWith('quote-1');
      });
    });
  });

  describe('Quote List search and sort (T228)', () => {
    it('sends the typed search as q (debounced) and resets to page 1', async () => {
      const user = userEvent.setup({ delay: null });
      const { QuotesPage } = await import('../../src/pages/Quotes/QuotesPage');

      render(
        <MemoryRouter initialEntries={['/quotes']}>
          <QuotesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenCalledTimes(1);
      });

      await user.type(screen.getByPlaceholderText(/search quotes/i), ' acme ');

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenLastCalledWith(
          expect.objectContaining({ q: 'acme', page: 1 })
        );
      });
      // One request for the whole burst of keystrokes, not one per key.
      expect(quoteService.getQuotes).toHaveBeenCalledTimes(2);
    });

    it('sends sort_by/sort_order when a sortable column is clicked', async () => {
      const user = userEvent.setup({ delay: null });
      const { QuotesPage } = await import('../../src/pages/Quotes/QuotesPage');

      render(
        <MemoryRouter initialEntries={['/quotes']}>
          <QuotesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('QT-001')).toBeInTheDocument();
      });

      await user.click(screen.getByText('Total'));

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenLastCalledWith(
          expect.objectContaining({ sort_by: 'total' })
        );
      });
    });
  });

  describe('Quote Filtering', () => {
    it('should filter quotes by status', async () => {
      const user = userEvent.setup({ delay: null });
      const { QuotesPage } = await import('../../src/pages/Quotes/QuotesPage');

      render(
        <MemoryRouter initialEntries={['/quotes']}>
          <QuotesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenCalled();
      });

      // Find status filter
      const statusFilter =
        screen.queryByLabelText(/status/i) ||
        screen.queryByRole('combobox', { name: /status/i });
      if (statusFilter) {
        await user.click(statusFilter);
        const acceptedOption = screen.queryByText(/accepted/i);
        if (acceptedOption) {
          await user.click(acceptedOption);
        }

        await waitFor(() => {
          expect(quoteService.getQuotes).toHaveBeenCalledWith(
            expect.objectContaining({
              status: 'accepted',
            })
          );
        });
      }
    });
  });

  describe('Error Handling', () => {
    it('should handle quote load error gracefully', async () => {
      vi.mocked(quoteService.getQuotes).mockRejectedValue(
        new Error('Failed to load quotes')
      );

      const { QuotesPage } = await import('../../src/pages/Quotes/QuotesPage');

      render(
        <MemoryRouter initialEntries={['/quotes']}>
          <QuotesPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenCalled();
      });

      // An API failure must show an error state with retry, not "No quotes yet"
      expect(
        await screen.findByText(/couldn't load quotes/i)
      ).toBeInTheDocument();
      expect(screen.queryByText(/no quotes yet/i)).not.toBeInTheDocument();

      vi.mocked(quoteService.getQuotes).mockClear();
      await userEvent
        .setup()
        .click(screen.getByRole('button', { name: /try again/i }));
      expect(quoteService.getQuotes).toHaveBeenCalledTimes(1);
    });

    it('should handle send quote error', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(quoteService.getQuote).mockResolvedValue(mockQuotes[0]);
      vi.mocked(quoteService.sendQuote).mockRejectedValue({
        response: {
          status: 403,
          data: {
            error: 'cannot_send',
            message: 'Quote cannot be sent in current status',
          },
        },
      });

      const { QuoteDetailPage } =
        await import('../../src/pages/Quotes/QuoteDetailPage');

      render(
        <MemoryRouter initialEntries={['/quotes/quote-1']}>
          <Routes>
            <Route path="/quotes/:id" element={<QuoteDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await user.click(
        await screen.findByRole('button', { name: /send quote/i })
      );

      expect(
        await screen.findByText('Quote cannot be sent in current status')
      ).toBeInTheDocument();
      expect(screen.getByText('Draft')).toBeInTheDocument();
    });

    it('shows a refused list action inline instead of window.alert', async () => {
      const user = userEvent.setup({ delay: null });
      const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});
      vi.mocked(quoteService.acceptQuote).mockRejectedValue({
        response: {
          status: 403,
          data: {
            error: 'cannot_accept',
            message: 'Quote cannot be accepted in current status',
          },
        },
      });

      const { QuotesPage } = await import('../../src/pages/Quotes/QuotesPage');
      render(
        <MemoryRouter initialEntries={['/quotes']}>
          <QuotesPage />
        </MemoryRouter>
      );

      const row = (await screen.findByText('QT-002')).closest('tr');
      expect(row).not.toBeNull();
      await user.click(
        within(row as HTMLElement).getByRole('button', {
          name: 'Quote actions',
        })
      );
      await user.click(screen.getByRole('menuitem', { name: /accept quote/i }));

      const banner = await screen.findByRole('alert');
      expect(banner).toHaveTextContent(
        'Quote cannot be accepted in current status'
      );
      expect(alertSpy).not.toHaveBeenCalled();

      await user.click(
        within(banner).getByRole('button', { name: /dismiss/i })
      );
      expect(screen.queryByRole('alert')).not.toBeInTheDocument();
      alertSpy.mockRestore();
    });
  });
});
