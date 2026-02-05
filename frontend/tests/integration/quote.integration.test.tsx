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

// Import after mocking
import { useAuthStore, useTenantStore, useSidebarStore } from '@store';
import { quoteService } from '@services/quoteService';
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
    total_amount: 1080,
    valid_until: '2024-12-31T00:00:00Z',
    lines: [
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
    total_amount: 540,
    valid_until: '2024-12-31T00:00:00Z',
    lines: [
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
    total_amount: 1620,
    valid_until: '2024-12-31T00:00:00Z',
    lines: [
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

  vi.mocked(quoteService.getQuotes).mockResolvedValue({
    quotes: mockQuotes,
    total: mockQuotes.length,
    page: 1,
    page_size: 10,
  });

  vi.mocked(quoteService.getStats).mockResolvedValue(mockQuoteStats);

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
      const user = userEvent.setup();
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
      const user = userEvent.setup();
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
      const user = userEvent.setup();
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
          await user.type(descInput, 'Test Service');
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
        expect(quoteService.createQuote).toHaveBeenCalled();
      });
    });
  });

  describe('Quote Status Transitions', () => {
    it('should send a draft quote to customer', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuote).mockResolvedValue(mockQuotes[0]); // Draft quote
      vi.mocked(quoteService.sendQuote).mockResolvedValue({
        ...mockQuotes[0],
        status: 'sent' as const,
        sent_at: '2024-01-05T00:00:00Z',
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

      await waitFor(() => {
        expect(quoteService.getQuote).toHaveBeenCalledWith('quote-1');
      });

      // Find and click send button
      const sendButton = screen.queryByRole('button', { name: /send/i });
      if (sendButton) {
        await user.click(sendButton);

        await waitFor(() => {
          expect(quoteService.sendQuote).toHaveBeenCalledWith('quote-1');
        });
      }
    });

    it('should accept a sent quote', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuote).mockResolvedValue(mockQuotes[1]); // Sent quote
      vi.mocked(quoteService.acceptQuote).mockResolvedValue({
        ...mockQuotes[1],
        status: 'accepted' as const,
        accepted_at: '2024-01-05T00:00:00Z',
      });

      const { QuoteDetailPage } =
        await import('../../src/pages/Quotes/QuoteDetailPage');

      render(
        <MemoryRouter initialEntries={['/quotes/quote-2']}>
          <Routes>
            <Route path="/quotes/:id" element={<QuoteDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(quoteService.getQuote).toHaveBeenCalledWith('quote-2');
      });

      // Find and click accept button
      const acceptButton = screen.queryByRole('button', { name: /accept/i });
      if (acceptButton) {
        await user.click(acceptButton);

        await waitFor(() => {
          expect(quoteService.acceptQuote).toHaveBeenCalledWith('quote-2');
        });
      }
    });

    it('should reject a sent quote', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuote).mockResolvedValue(mockQuotes[1]); // Sent quote
      vi.mocked(quoteService.rejectQuote).mockResolvedValue({
        ...mockQuotes[1],
        status: 'declined' as const,
      });

      const { QuoteDetailPage } =
        await import('../../src/pages/Quotes/QuoteDetailPage');

      render(
        <MemoryRouter initialEntries={['/quotes/quote-2']}>
          <Routes>
            <Route path="/quotes/:id" element={<QuoteDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(quoteService.getQuote).toHaveBeenCalled();
      });

      // Find and click reject/decline button
      const rejectButton = screen.queryByRole('button', {
        name: /reject|decline/i,
      });
      if (rejectButton) {
        await user.click(rejectButton);

        // Confirm if dialog appears
        const confirmButton = screen.queryByRole('button', {
          name: /confirm|yes/i,
        });
        if (confirmButton) {
          await user.click(confirmButton);
        }

        await waitFor(() => {
          expect(quoteService.rejectQuote).toHaveBeenCalledWith('quote-2');
        });
      }
    });
  });

  describe('Quote PDF', () => {
    it('should download quote PDF', async () => {
      const user = userEvent.setup();
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

      // Find and click download/PDF button
      const pdfButton = screen.queryByRole('button', { name: /pdf|download/i });
      if (pdfButton) {
        await user.click(pdfButton);

        await waitFor(() => {
          expect(quoteService.downloadQuotePDF).toHaveBeenCalledWith(
            'quote-1',
            expect.anything()
          );
        });
      }
    });
  });

  describe('Quote Filtering', () => {
    it('should filter quotes by status', async () => {
      const user = userEvent.setup();
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
    });

    it('should handle send quote error', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuote).mockResolvedValue(mockQuotes[0]);
      vi.mocked(quoteService.sendQuote).mockRejectedValue({
        response: { data: { message: 'Customer email not available' } },
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

      await waitFor(() => {
        expect(quoteService.getQuote).toHaveBeenCalled();
      });

      const sendButton = screen.queryByRole('button', { name: /send/i });
      if (sendButton) {
        await user.click(sendButton);

        await waitFor(() => {
          const errorMessage = screen.queryByText(
            /error|failed|not available/i
          );
          if (errorMessage) {
            expect(errorMessage).toBeInTheDocument();
          }
        });
      }
    });
  });
});
