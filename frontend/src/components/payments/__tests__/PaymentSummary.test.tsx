import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { PaymentSummary } from '../PaymentSummary';
import type { PaymentSummaryData } from '../../../types/payment';

describe('PaymentSummary', () => {
  const baseSummary: PaymentSummaryData = {
    lineItems: [
      {
        id: '1',
        description: 'Service Plan - Pro',
        quantity: 1,
        unitPrice: 9900,
        totalPrice: 9900,
      },
    ],
    subtotal: 9900,
    tax: 792,
    taxRate: 8,
    total: 10692,
    currency: 'usd',
  };

  it('renders order summary header', () => {
    render(<PaymentSummary summary={baseSummary} />);

    expect(screen.getByText('Order Summary')).toBeInTheDocument();
  });

  it('displays line items when showLineItems is true', () => {
    render(<PaymentSummary summary={baseSummary} showLineItems />);

    expect(screen.getByText('Service Plan - Pro')).toBeInTheDocument();
    // $99.00 appears for both line item and subtotal
    const prices = screen.getAllByText('$99.00');
    expect(prices.length).toBeGreaterThan(0);
  });

  it('hides line items when showLineItems is false', () => {
    render(<PaymentSummary summary={baseSummary} showLineItems={false} />);

    expect(screen.queryByText('Service Plan - Pro')).not.toBeInTheDocument();
  });

  it('displays subtotal', () => {
    render(<PaymentSummary summary={baseSummary} />);

    expect(screen.getByText('Subtotal')).toBeInTheDocument();
    // Find at least one $99.00 (subtotal value)
    const prices = screen.getAllByText('$99.00');
    expect(prices.length).toBeGreaterThan(0);
  });

  it('displays tax with rate', () => {
    render(<PaymentSummary summary={baseSummary} />);

    expect(screen.getByText('Tax (8%)')).toBeInTheDocument();
    expect(screen.getByText('$7.92')).toBeInTheDocument();
  });

  it('displays tax without rate when not provided', () => {
    const summaryWithoutRate = { ...baseSummary, taxRate: undefined };
    render(<PaymentSummary summary={summaryWithoutRate} />);

    expect(screen.getByText('Tax')).toBeInTheDocument();
  });

  it('displays total amount', () => {
    render(<PaymentSummary summary={baseSummary} />);

    expect(screen.getByText('Total')).toBeInTheDocument();
    expect(screen.getByText('$106.92')).toBeInTheDocument();
  });

  it('displays discount when provided', () => {
    const summaryWithDiscount: PaymentSummaryData = {
      ...baseSummary,
      discount: 1000,
      discountCode: 'SAVE10',
      total: 9692,
    };

    render(<PaymentSummary summary={summaryWithDiscount} />);

    expect(screen.getByText('Discount (SAVE10)')).toBeInTheDocument();
    expect(screen.getByText('-$10.00')).toBeInTheDocument();
  });

  it('displays discount without code', () => {
    const summaryWithDiscount: PaymentSummaryData = {
      ...baseSummary,
      discount: 500,
      total: 10192,
    };

    render(<PaymentSummary summary={summaryWithDiscount} />);

    expect(screen.getByText('Discount')).toBeInTheDocument();
    expect(screen.getByText('-$5.00')).toBeInTheDocument();
  });

  it('shows quantity and unit price for multi-quantity items', () => {
    const summaryWithMultiQuantity: PaymentSummaryData = {
      ...baseSummary,
      lineItems: [
        {
          id: '1',
          description: 'Widget',
          quantity: 3,
          unitPrice: 1000,
          totalPrice: 3000,
        },
      ],
      subtotal: 3000,
      tax: 240,
      total: 3240,
    };

    render(<PaymentSummary summary={summaryWithMultiQuantity} showLineItems />);

    expect(screen.getByText('3 x $10.00')).toBeInTheDocument();
    // $30.00 appears for both line item and subtotal
    const prices = screen.getAllByText('$30.00');
    expect(prices.length).toBeGreaterThan(0);
  });

  it('displays currency notice', () => {
    render(<PaymentSummary summary={baseSummary} />);

    expect(screen.getByText('All amounts in USD')).toBeInTheDocument();
  });

  it('renders loading skeleton when isLoading is true', () => {
    render(<PaymentSummary summary={baseSummary} isLoading />);

    expect(
      screen.getByLabelText('Loading payment summary')
    ).toBeInTheDocument();
    expect(screen.queryByText('Order Summary')).not.toBeInTheDocument();
  });

  it('handles multiple line items', () => {
    const summaryWithMultipleItems: PaymentSummaryData = {
      lineItems: [
        {
          id: '1',
          description: 'Basic Plan',
          quantity: 1,
          unitPrice: 2900,
          totalPrice: 2900,
        },
        {
          id: '2',
          description: 'Add-on: Analytics',
          quantity: 1,
          unitPrice: 900,
          totalPrice: 900,
        },
        {
          id: '3',
          description: 'Add-on: Support',
          quantity: 1,
          unitPrice: 1900,
          totalPrice: 1900,
        },
      ],
      subtotal: 5700,
      tax: 456,
      total: 6156,
      currency: 'usd',
    };

    render(<PaymentSummary summary={summaryWithMultipleItems} showLineItems />);

    expect(screen.getByText('Basic Plan')).toBeInTheDocument();
    expect(screen.getByText('Add-on: Analytics')).toBeInTheDocument();
    expect(screen.getByText('Add-on: Support')).toBeInTheDocument();
  });

  it('handles different currencies', () => {
    const eurSummary: PaymentSummaryData = {
      ...baseSummary,
      currency: 'eur',
    };

    render(<PaymentSummary summary={eurSummary} />);

    expect(screen.getByText('All amounts in EUR')).toBeInTheDocument();
  });

  it('applies custom className', () => {
    const { container } = render(
      <PaymentSummary summary={baseSummary} className="custom-class" />
    );

    expect(container.firstChild).toHaveClass('custom-class');
  });

  it('has accessible region role', () => {
    render(<PaymentSummary summary={baseSummary} />);

    expect(screen.getByRole('region')).toHaveAttribute(
      'aria-label',
      'Payment summary'
    );
  });

  it('handles zero discount gracefully', () => {
    const summaryWithZeroDiscount: PaymentSummaryData = {
      ...baseSummary,
      discount: 0,
    };

    render(<PaymentSummary summary={summaryWithZeroDiscount} />);

    // Zero discount should not show
    expect(screen.queryByText(/discount/i)).not.toBeInTheDocument();
  });
});
