import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { QuoteList } from '../QuoteList';
import { quoteService } from '../../../services/quoteService';
import { QuoteStatus, Quote } from '../../../types/quote';

// Unmock @tanstack/react-table since QuoteList needs the real implementation
vi.unmock('@tanstack/react-table');

// Mock the quote service
vi.mock('../../../services/quoteService', () => ({
  quoteService: {
    getQuotes: vi.fn(),
    sendQuote: vi.fn(),
  },
}));

const mockQuotes: Quote[] = [
  {
    id: '1',
    quote_number: 'Q-2024-001',
    customer_id: 'cust-1',
    customer_name: 'John Doe',
    customer_email: 'john@example.com',
    status: QuoteStatus.DRAFT,
    subtotal: '1000.00',
    tax_rate: '0.0825',
    tax_amount: '82.50',
    total: '1082.50',
    valid_until: '2024-12-31T23:59:59Z',
    notes: '',
    items: [],
    created_at: '2024-01-15T10:00:00Z',
    updated_at: '2024-01-15T10:00:00Z',
  },
  {
    id: '2',
    quote_number: 'Q-2024-002',
    customer_id: 'cust-2',
    customer_name: 'Jane Smith',
    customer_email: 'jane@example.com',
    status: QuoteStatus.SENT,
    subtotal: '2000.00',
    tax_rate: '0.0825',
    tax_amount: '165.00',
    total: '2165.00',
    valid_until: '2024-12-31T23:59:59Z',
    notes: '',
    items: [],
    created_at: '2024-01-16T10:00:00Z',
    updated_at: '2024-01-16T10:00:00Z',
  },
  {
    id: '3',
    quote_number: 'Q-2024-003',
    customer_id: 'cust-3',
    customer_name: 'Bob Johnson',
    customer_email: 'bob@example.com',
    status: QuoteStatus.ACCEPTED,
    subtotal: '3000.00',
    tax_rate: '0.0825',
    tax_amount: '247.50',
    total: '3247.50',
    valid_until: '2024-12-31T23:59:59Z',
    notes: '',
    items: [],
    created_at: '2024-01-17T10:00:00Z',
    updated_at: '2024-01-17T10:00:00Z',
  },
];

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('QuoteList', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Loading State', () => {
    it('should display loading spinner while fetching data', () => {
      vi.mocked(quoteService.getQuotes).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      const { container } = render(<QuoteList />, { wrapper: createWrapper() });

      expect(screen.getByText('Loading quotes...')).toBeInTheDocument();
      // Check for spinner by its class instead of role
      expect(container.querySelector('.animate-spin')).toBeInTheDocument();
    });
  });

  describe('Error State', () => {
    it('should display error message when fetch fails', async () => {
      const errorMessage = 'Failed to fetch quotes';
      vi.mocked(quoteService.getQuotes).mockRejectedValue(
        new Error(errorMessage)
      );

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Error loading quotes')).toBeInTheDocument();
        expect(screen.getByText(errorMessage)).toBeInTheDocument();
      });
    });

    it('should allow retry after error', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuotes)
        .mockRejectedValueOnce(new Error('Network error'))
        .mockResolvedValueOnce({
          quotes: mockQuotes,
          pagination: {
            page: 1,
            page_size: 25,
            total: mockQuotes.length,
            total_pages: 1,
          },
        });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Error loading quotes')).toBeInTheDocument();
      });

      const retryButton = screen.getByRole('button', { name: /try again/i });
      await user.click(retryButton);

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });
    });
  });

  describe('Empty State', () => {
    it('should display "No quotes found" when list is empty', async () => {
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: [],
        pagination: {
          page: 1,
          page_size: 25,
          total: 0,
          total_pages: 0,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('No quotes found')).toBeInTheDocument();
      });
    });
  });

  describe('Rendering Quotes', () => {
    it('should render all quotes in the list', async () => {
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
        expect(screen.getByText('Q-2024-002')).toBeInTheDocument();
        expect(screen.getByText('Q-2024-003')).toBeInTheDocument();
      });
    });

    it('should display customer names and emails', async () => {
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('John Doe')).toBeInTheDocument();
        expect(screen.getByText('john@example.com')).toBeInTheDocument();
        expect(screen.getByText('Jane Smith')).toBeInTheDocument();
        expect(screen.getByText('jane@example.com')).toBeInTheDocument();
      });
    });

    it('should format amounts with currency', async () => {
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('$1,082.50')).toBeInTheDocument();
        expect(screen.getByText('$2,165.00')).toBeInTheDocument();
        expect(screen.getByText('$3,247.50')).toBeInTheDocument();
      });
    });

    it('should display status badges with correct colors', async () => {
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      const { container } = render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        // Use getAllByText and find the badges (span elements with rounded-full class)
        const draftBadges = screen.getAllByText('Draft');
        const draftBadge = draftBadges.find(
          (el) => el.tagName === 'SPAN' && el.classList.contains('rounded-full')
        );

        const sentBadges = screen.getAllByText('Sent');
        const sentBadge = sentBadges.find(
          (el) => el.tagName === 'SPAN' && el.classList.contains('rounded-full')
        );

        const acceptedBadges = screen.getAllByText('Accepted');
        const acceptedBadge = acceptedBadges.find(
          (el) => el.tagName === 'SPAN' && el.classList.contains('rounded-full')
        );

        expect(draftBadge).toHaveClass('bg-gray-100', 'text-gray-800');
        expect(sentBadge).toHaveClass('bg-blue-100', 'text-blue-800');
        expect(acceptedBadge).toHaveClass('bg-green-100', 'text-green-800');
      });
    });
  });

  describe('Filtering', () => {
    it('should filter by draft status', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      const draftButton = screen.getByRole('button', { name: 'Draft' });
      await user.click(draftButton);

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenCalledWith(
          expect.objectContaining({
            status: QuoteStatus.DRAFT,
            page: 1,
          })
        );
      });
    });

    it('should filter by sent status', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      const sentButton = screen.getByRole('button', { name: 'Sent' });
      await user.click(sentButton);

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenCalledWith(
          expect.objectContaining({
            status: QuoteStatus.SENT,
            page: 1,
          })
        );
      });
    });

    it('should show all quotes when "All" filter is selected', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      // Click Draft filter first
      const draftButton = screen.getByRole('button', { name: 'Draft' });
      await user.click(draftButton);

      // Then click All filter
      const allButton = screen.getByRole('button', { name: 'All' });
      await user.click(allButton);

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenLastCalledWith(
          expect.objectContaining({
            status: undefined,
          })
        );
      });
    });
  });

  describe('Pagination', () => {
    it('should display pagination information', async () => {
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: 100,
          total_pages: 4,
        },
      });

      const { container } = render(<QuoteList />, { wrapper: createWrapper() });

      // First wait for data to load
      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      // Pagination text is in a p.text-sm.text-gray-700 element
      const paginationText = container.querySelector('p.text-sm.text-gray-700');
      expect(paginationText).toBeInTheDocument();
      expect(paginationText?.textContent).toContain('Showing');
      expect(paginationText?.textContent).toContain('100');
      expect(paginationText?.textContent).toContain('results');
    });

    it('should navigate to next page', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: 100,
          total_pages: 4,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      const nextButtons = screen.getAllByRole('button', { name: /next/i });
      await user.click(nextButtons[0]);

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenCalledWith(
          expect.objectContaining({
            page: 2,
          })
        );
      });
    });

    it('should navigate to previous page', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 2,
          page_size: 25,
          total: 100,
          total_pages: 4,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      const prevButtons = screen.getAllByRole('button', { name: /previous/i });
      await user.click(prevButtons[0]);

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenCalledWith(
          expect.objectContaining({
            page: 1,
          })
        );
      });
    });

    it('should disable previous button on first page', async () => {
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: 100,
          total_pages: 4,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        const prevButtons = screen.getAllByRole('button', {
          name: /previous/i,
        });
        prevButtons.forEach((button) => {
          expect(button).toBeDisabled();
        });
      });
    });

    it('should disable next button on last page', async () => {
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 4,
          page_size: 25,
          total: 100,
          total_pages: 4,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        const nextButtons = screen.getAllByRole('button', { name: /next/i });
        nextButtons.forEach((button) => {
          expect(button).toBeDisabled();
        });
      });
    });
  });

  describe('Page Size', () => {
    it('should change page size to 50', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      const select = screen.getByRole('combobox');
      await user.selectOptions(select, '50');

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenCalledWith(
          expect.objectContaining({
            page_size: 50,
            page: 1,
          })
        );
      });
    });

    it('should change page size to 100', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      const select = screen.getByRole('combobox');
      await user.selectOptions(select, '100');

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenCalledWith(
          expect.objectContaining({
            page_size: 100,
            page: 1,
          })
        );
      });
    });
  });

  describe('Quick Actions Menu', () => {
    it('should open actions menu when button is clicked', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList onEdit={vi.fn()} onDelete={vi.fn()} />, {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      const actionButtons = screen.getAllByRole('button', { name: 'Actions' });
      await user.click(actionButtons[0]);

      await waitFor(() => {
        expect(screen.getByText('Edit Quote')).toBeInTheDocument();
        expect(screen.getByText('Delete Quote')).toBeInTheDocument();
      });
    });

    it('should call onEdit when edit is clicked', async () => {
      const user = userEvent.setup();
      const onEdit = vi.fn();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList onEdit={onEdit} />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      const actionButtons = screen.getAllByRole('button', { name: 'Actions' });
      await user.click(actionButtons[0]);

      const editButton = screen.getByText('Edit Quote');
      await user.click(editButton);

      expect(onEdit).toHaveBeenCalledWith(mockQuotes[0]);
    });

    it('should call onDelete when delete is clicked', async () => {
      const user = userEvent.setup();
      const onDelete = vi.fn();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList onDelete={onDelete} />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      const actionButtons = screen.getAllByRole('button', { name: 'Actions' });
      await user.click(actionButtons[0]);

      const deleteButton = screen.getByText('Delete Quote');
      await user.click(deleteButton);

      expect(onDelete).toHaveBeenCalledWith(mockQuotes[0]);
    });

    it('should show "Convert to Job" for accepted quotes', async () => {
      const user = userEvent.setup();
      const onConvert = vi.fn();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList onConvert={onConvert} />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-003')).toBeInTheDocument();
      });

      // Click on actions for the accepted quote (third row)
      const actionButtons = screen.getAllByRole('button', { name: 'Actions' });
      await user.click(actionButtons[2]);

      expect(screen.getByText('Convert to Job')).toBeInTheDocument();
    });

    it('should show "Send to Customer" for draft quotes', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });
      vi.mocked(quoteService.sendQuote).mockResolvedValue(mockQuotes[0]);

      render(<QuoteList />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      // Click on actions for the draft quote (first row)
      const actionButtons = screen.getAllByRole('button', { name: 'Actions' });
      await user.click(actionButtons[0]);

      expect(screen.getByText('Send to Customer')).toBeInTheDocument();
    });

    it('should close menu when clicking outside', async () => {
      const user = userEvent.setup();
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      const { container } = render(
        <QuoteList onEdit={vi.fn()} onDelete={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('Q-2024-001')).toBeInTheDocument();
      });

      const actionButtons = screen.getAllByRole('button', { name: 'Actions' });
      await user.click(actionButtons[0]);

      await waitFor(() => {
        expect(screen.getByText('Edit Quote')).toBeInTheDocument();
      });

      // Click on the backdrop overlay (fixed inset-0 div) which closes the menu
      const backdrop = container.querySelector('.fixed.inset-0.z-10');
      if (backdrop) {
        await user.click(backdrop);
      }

      await waitFor(() => {
        expect(screen.queryByText('Edit Quote')).not.toBeInTheDocument();
      });
    });
  });

  describe('Customer Filtering', () => {
    it('should filter by customer ID when provided', async () => {
      const customerId = 'cust-123';
      vi.mocked(quoteService.getQuotes).mockResolvedValue({
        quotes: mockQuotes,
        pagination: {
          page: 1,
          page_size: 25,
          total: mockQuotes.length,
          total_pages: 1,
        },
      });

      render(<QuoteList customerId={customerId} />, {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(quoteService.getQuotes).toHaveBeenCalledWith(
          expect.objectContaining({
            customer_id: customerId,
          })
        );
      });
    });
  });
});
