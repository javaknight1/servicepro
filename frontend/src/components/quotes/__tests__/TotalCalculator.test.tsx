import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import TotalCalculator from '../TotalCalculator';
import { LineItem } from '../../../types/quote';

describe('TotalCalculator', () => {
  const mockItems: LineItem[] = [
    {
      id: '1',
      description: 'Item 1',
      quantity: 2,
      unit_price: 50,
      total: 100,
      sort_order: 0,
    },
    {
      id: '2',
      description: 'Item 2',
      quantity: 1,
      unit_price: 75,
      total: 75,
      sort_order: 1,
    },
  ];

  it('should render totals correctly', () => {
    render(<TotalCalculator items={mockItems} taxRate={0.0825} />);

    expect(screen.getByText('Subtotal')).toBeInTheDocument();
    expect(screen.getByText('$175.00')).toBeInTheDocument(); // Subtotal
    expect(screen.getByText('Tax (8.25%)')).toBeInTheDocument();
    expect(screen.getByText('$14.44')).toBeInTheDocument(); // Tax amount
    expect(screen.getByText('Total')).toBeInTheDocument();
    expect(screen.getByText('$189.44')).toBeInTheDocument(); // Grand total
  });

  it('should show item count', () => {
    render(<TotalCalculator items={mockItems} taxRate={0.0825} />);

    expect(screen.getByText('2 items')).toBeInTheDocument();
  });

  it('should show singular "item" for one item', () => {
    const singleItem = [mockItems[0]];
    render(<TotalCalculator items={singleItem} taxRate={0.0825} />);

    expect(screen.getByText('1 item')).toBeInTheDocument();
  });

  it('should show empty state when no items', () => {
    render(<TotalCalculator items={[]} taxRate={0.0825} />);

    expect(screen.getByText('No items added yet')).toBeInTheDocument();
    expect(
      screen.getByText('Add line items to calculate total')
    ).toBeInTheDocument();
  });

  it('should show breakdown when showBreakdown is true', () => {
    render(
      <TotalCalculator items={mockItems} taxRate={0.0825} showBreakdown />
    );

    expect(screen.getByText('Breakdown')).toBeInTheDocument();
    expect(screen.getByText(/2 × Item 1/)).toBeInTheDocument();
    expect(screen.getByText(/1 × Item 2/)).toBeInTheDocument();
  });

  it('should not show breakdown when showBreakdown is false', () => {
    render(
      <TotalCalculator
        items={mockItems}
        taxRate={0.0825}
        showBreakdown={false}
      />
    );

    expect(screen.queryByText('Breakdown')).not.toBeInTheDocument();
  });

  it('should calculate totals correctly with zero tax rate', () => {
    render(<TotalCalculator items={mockItems} taxRate={0} />);

    // When tax rate is 0, subtotal and grand total are both $175.00
    const priceElements = screen.getAllByText('$175.00');
    expect(priceElements.length).toBe(2); // Subtotal and Grand Total
    expect(screen.getByText('$0.00')).toBeInTheDocument(); // Tax amount
  });

  it('should handle decimal quantities and prices', () => {
    const decimalItems: LineItem[] = [
      {
        id: '1',
        description: 'Service Hours',
        quantity: 2.5,
        unit_price: 75.5,
        total: 188.75,
        sort_order: 0,
      },
    ];

    render(<TotalCalculator items={decimalItems} taxRate={0.0825} />);

    expect(screen.getByText('$188.75')).toBeInTheDocument(); // Subtotal
  });

  it('should apply custom className', () => {
    const { container } = render(
      <TotalCalculator
        items={mockItems}
        taxRate={0.0825}
        className="custom-class"
      />
    );

    const calculator = container.querySelector('.custom-class');
    expect(calculator).toBeInTheDocument();
  });

  it('should format tax percentage correctly', () => {
    render(<TotalCalculator items={mockItems} taxRate={0.0825} />);

    expect(screen.getByText('Tax (8.25%)')).toBeInTheDocument();
  });

  it('should handle large numbers correctly', () => {
    const largeItems: LineItem[] = [
      {
        id: '1',
        description: 'Large Item',
        quantity: 100,
        unit_price: 1000,
        total: 100000,
        sort_order: 0,
      },
    ];

    render(<TotalCalculator items={largeItems} taxRate={0.0825} />);

    expect(screen.getByText('$100000.00')).toBeInTheDocument(); // Subtotal
  });
});
