import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BillingSchedule } from '../../../src/components/subscriptions/BillingSchedule';
import type {
  BillingSchedule as BillingScheduleType,
  Invoice,
} from '../../../src/types/subscription';

// Mock billing schedule
const createMockSchedule = (
  overrides: Partial<BillingScheduleType> = {}
): BillingScheduleType => {
  const now = new Date();
  const thirtyDaysLater = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);
  const sixtyDaysLater = new Date(now.getTime() + 60 * 24 * 60 * 60 * 1000);
  const ninetyDaysLater = new Date(now.getTime() + 90 * 24 * 60 * 60 * 1000);

  return {
    currentPeriod: {
      startDate: now.toISOString(),
      endDate: thirtyDaysLater.toISOString(),
      amount: 2900,
      isCurrent: true,
      isPaid: true,
    },
    upcomingPeriods: [
      {
        startDate: thirtyDaysLater.toISOString(),
        endDate: sixtyDaysLater.toISOString(),
        amount: 2900,
        isCurrent: false,
        isPaid: false,
      },
      {
        startDate: sixtyDaysLater.toISOString(),
        endDate: ninetyDaysLater.toISOString(),
        amount: 2900,
        isCurrent: false,
        isPaid: false,
      },
    ],
    billingCycle: 'monthly',
    nextBillingDate: thirtyDaysLater.toISOString(),
    amountDue: 2900,
    currency: 'USD',
    ...overrides,
  };
};

// Mock invoices
const createMockInvoices = (): Invoice[] => {
  const now = new Date();
  return [
    {
      id: 'inv-1',
      subscriptionId: 'sub-123',
      customerId: 'cust-456',
      stripeInvoiceId: 'in_abc123',
      subtotal: 2900,
      tax: 0,
      total: 2900,
      amountDue: 0,
      amountPaid: 2900,
      status: 'paid',
      periodStart: new Date(
        now.getTime() - 30 * 24 * 60 * 60 * 1000
      ).toISOString(),
      periodEnd: now.toISOString(),
      paidAt: now.toISOString(),
      hostedInvoiceUrl: 'https://stripe.com/invoice/123',
      invoicePdf: 'https://stripe.com/invoice/123/pdf',
      lineItems: [],
      createdAt: now.toISOString(),
    },
    {
      id: 'inv-2',
      subscriptionId: 'sub-123',
      customerId: 'cust-456',
      stripeInvoiceId: 'in_def456',
      subtotal: 2900,
      tax: 0,
      total: 2900,
      amountDue: 0,
      amountPaid: 2900,
      status: 'paid',
      periodStart: new Date(
        now.getTime() - 60 * 24 * 60 * 60 * 1000
      ).toISOString(),
      periodEnd: new Date(
        now.getTime() - 30 * 24 * 60 * 60 * 1000
      ).toISOString(),
      paidAt: new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000).toISOString(),
      hostedInvoiceUrl: 'https://stripe.com/invoice/456',
      invoicePdf: 'https://stripe.com/invoice/456/pdf',
      lineItems: [],
      createdAt: new Date(
        now.getTime() - 30 * 24 * 60 * 60 * 1000
      ).toISOString(),
    },
  ];
};

describe('BillingSchedule Component', () => {
  const mockOnViewInvoice = vi.fn();
  const mockOnDownloadInvoice = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render next billing section', () => {
      const schedule = createMockSchedule();
      render(<BillingSchedule schedule={schedule} />);

      expect(screen.getByText('Next Billing')).toBeInTheDocument();
      expect(screen.getByText('$29.00')).toBeInTheDocument();
    });

    it('should render billing cycle label', () => {
      const schedule = createMockSchedule();
      render(<BillingSchedule schedule={schedule} />);

      expect(screen.getByText('Monthly subscription')).toBeInTheDocument();
    });

    it('should render annual billing cycle correctly', () => {
      const schedule = createMockSchedule({
        billingCycle: 'annual',
        amountDue: 29000,
      });
      render(<BillingSchedule schedule={schedule} />);

      expect(screen.getByText('Annual subscription')).toBeInTheDocument();
    });

    it('should render current period section', () => {
      const schedule = createMockSchedule();
      render(<BillingSchedule schedule={schedule} />);

      expect(screen.getByText('Current Period')).toBeInTheDocument();
      expect(screen.getByText('Active')).toBeInTheDocument();
    });

    it('should render progress bar for current period', () => {
      const schedule = createMockSchedule();
      const { container } = render(<BillingSchedule schedule={schedule} />);

      // Check for progress bar element
      const progressBar = container.querySelector('.bg-primary-500');
      expect(progressBar).toBeInTheDocument();
    });

    it('should render upcoming periods', () => {
      const schedule = createMockSchedule();
      render(<BillingSchedule schedule={schedule} />);

      expect(screen.getByText('Upcoming Billing')).toBeInTheDocument();
    });

    it('should not render upcoming periods section if empty', () => {
      const schedule = createMockSchedule({
        upcomingPeriods: [],
      });
      render(<BillingSchedule schedule={schedule} />);

      expect(screen.queryByText('Upcoming Billing')).not.toBeInTheDocument();
    });
  });

  describe('Invoice History', () => {
    it('should render invoice history', () => {
      const schedule = createMockSchedule();
      const invoices = createMockInvoices();

      render(<BillingSchedule schedule={schedule} invoices={invoices} />);

      expect(screen.getByText('Invoice History')).toBeInTheDocument();
    });

    it('should display invoice status badges', () => {
      const schedule = createMockSchedule();
      const invoices = createMockInvoices();

      render(<BillingSchedule schedule={schedule} invoices={invoices} />);

      const paidBadges = screen.getAllByText('Paid');
      expect(paidBadges.length).toBeGreaterThan(0);
    });

    it('should show empty state when no invoices', () => {
      const schedule = createMockSchedule();

      render(<BillingSchedule schedule={schedule} invoices={[]} />);

      expect(screen.getByText('No invoices yet')).toBeInTheDocument();
    });

    it('should not show invoice history when showPastInvoices is false', () => {
      const schedule = createMockSchedule();
      const invoices = createMockInvoices();

      render(
        <BillingSchedule
          schedule={schedule}
          invoices={invoices}
          showPastInvoices={false}
        />
      );

      expect(screen.queryByText('Invoice History')).not.toBeInTheDocument();
    });

    it('should limit displayed invoices', () => {
      const schedule = createMockSchedule();
      const manyInvoices: Invoice[] = Array.from({ length: 10 }, (_, i) => ({
        ...createMockInvoices()[0],
        id: `inv-${i}`,
        stripeInvoiceId: `in_test${i}`,
      }));

      render(
        <BillingSchedule
          schedule={schedule}
          invoices={manyInvoices}
          maxInvoices={3}
        />
      );

      // Should show "View all" button when more invoices exist
      expect(screen.getByText('View all (10)')).toBeInTheDocument();
    });
  });

  describe('Invoice Actions', () => {
    it('should show view button when onViewInvoice provided', () => {
      const schedule = createMockSchedule();
      const invoices = createMockInvoices();

      render(
        <BillingSchedule
          schedule={schedule}
          invoices={invoices}
          onViewInvoice={mockOnViewInvoice}
        />
      );

      const viewButtons = screen.getAllByTitle('View invoice');
      expect(viewButtons.length).toBeGreaterThan(0);
    });

    it('should show download button when onDownloadInvoice provided', () => {
      const schedule = createMockSchedule();
      const invoices = createMockInvoices();

      render(
        <BillingSchedule
          schedule={schedule}
          invoices={invoices}
          onDownloadInvoice={mockOnDownloadInvoice}
        />
      );

      const downloadButtons = screen.getAllByTitle('Download PDF');
      expect(downloadButtons.length).toBeGreaterThan(0);
    });

    it('should call onViewInvoice when view button clicked', async () => {
      const user = userEvent.setup();
      const schedule = createMockSchedule();
      const invoices = createMockInvoices();

      render(
        <BillingSchedule
          schedule={schedule}
          invoices={invoices}
          onViewInvoice={mockOnViewInvoice}
        />
      );

      const viewButtons = screen.getAllByTitle('View invoice');
      await user.click(viewButtons[0]);

      expect(mockOnViewInvoice).toHaveBeenCalledWith(
        expect.objectContaining({
          id: invoices[0].id,
        })
      );
    });

    it('should call onDownloadInvoice when download button clicked', async () => {
      const user = userEvent.setup();
      const schedule = createMockSchedule();
      const invoices = createMockInvoices();

      render(
        <BillingSchedule
          schedule={schedule}
          invoices={invoices}
          onDownloadInvoice={mockOnDownloadInvoice}
        />
      );

      const downloadButtons = screen.getAllByTitle('Download PDF');
      await user.click(downloadButtons[0]);

      expect(mockOnDownloadInvoice).toHaveBeenCalledWith(
        expect.objectContaining({
          id: invoices[0].id,
        })
      );
    });
  });

  describe('Loading State', () => {
    it('should render loading skeleton', () => {
      const schedule = createMockSchedule();
      const { container } = render(
        <BillingSchedule schedule={schedule} isLoading />
      );

      const skeletons = container.querySelectorAll('.animate-pulse');
      expect(skeletons.length).toBeGreaterThan(0);
    });
  });

  describe('Invoice Status Badges', () => {
    it('should show different status badges', () => {
      const schedule = createMockSchedule();
      const invoicesWithStatuses: Invoice[] = [
        { ...createMockInvoices()[0], id: 'inv-1', status: 'paid' },
        {
          ...createMockInvoices()[0],
          id: 'inv-2',
          stripeInvoiceId: 'test2',
          status: 'open',
        },
        {
          ...createMockInvoices()[0],
          id: 'inv-3',
          stripeInvoiceId: 'test3',
          status: 'draft',
        },
      ];

      render(
        <BillingSchedule schedule={schedule} invoices={invoicesWithStatuses} />
      );

      expect(screen.getByText('Paid')).toBeInTheDocument();
      expect(screen.getByText('Open')).toBeInTheDocument();
      expect(screen.getByText('Draft')).toBeInTheDocument();
    });
  });

  describe('Payment Method Reminder', () => {
    it('should show payment method reminder', () => {
      const schedule = createMockSchedule();
      render(<BillingSchedule schedule={schedule} />);

      expect(
        screen.getByText(/Payments are automatically charged/i)
      ).toBeInTheDocument();
    });
  });

  describe('Custom ClassName', () => {
    it('should apply custom className', () => {
      const schedule = createMockSchedule();
      const { container } = render(
        <BillingSchedule schedule={schedule} className="custom-class" />
      );

      expect(container.firstChild).toHaveClass('custom-class');
    });
  });

  describe('Currency Formatting', () => {
    it('should format amounts correctly', () => {
      const schedule = createMockSchedule({
        amountDue: 19900, // $199.00
      });

      render(<BillingSchedule schedule={schedule} />);

      expect(screen.getByText('$199.00')).toBeInTheDocument();
    });
  });
});
