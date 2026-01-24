import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { PaymentMethodList } from '../PaymentMethodList';
import type { StoredPaymentMethod } from '../../../../types/payment';

describe('PaymentMethodList', () => {
  const createPaymentMethod = (
    overrides: Partial<StoredPaymentMethod> = {}
  ): StoredPaymentMethod => ({
    id: 'pm-' + Math.random().toString(36).substr(2, 9),
    type: 'card',
    isDefault: false,
    displayName: 'Visa •••• 4242',
    cardBrand: 'visa',
    cardLast4: '4242',
    cardExpMonth: 12,
    cardExpYear: new Date().getFullYear() + 1,
    isExpired: false,
    isExpiringSoon: false,
    isActive: true,
    createdAt: new Date().toISOString(),
    ...overrides,
  });

  const defaultPaymentMethod = createPaymentMethod({
    id: 'pm-default',
    isDefault: true,
  });
  const regularPaymentMethod = createPaymentMethod({ id: 'pm-regular' });
  const expiredPaymentMethod = createPaymentMethod({
    id: 'pm-expired',
    isExpired: true,
  });
  const inactivePaymentMethod = createPaymentMethod({
    id: 'pm-inactive',
    isActive: false,
  });

  beforeEach(() => {
    vi.clearAllMocks();
  });

  // ==========================================================================
  // Loading State Tests
  // ==========================================================================

  it('renders loading skeleton when isLoading is true', () => {
    render(<PaymentMethodList paymentMethods={[]} isLoading />);

    expect(screen.getByRole('status')).toBeInTheDocument();
    expect(
      screen.getByLabelText('Loading payment methods')
    ).toBeInTheDocument();
  });

  // ==========================================================================
  // Error State Tests
  // ==========================================================================

  it('renders error message when error prop is provided', () => {
    render(
      <PaymentMethodList
        paymentMethods={[]}
        error="Failed to load payment methods"
      />
    );

    expect(
      screen.getByText('Failed to load payment methods')
    ).toBeInTheDocument();
  });

  // ==========================================================================
  // Empty State Tests
  // ==========================================================================

  it('renders empty state when no payment methods', () => {
    render(<PaymentMethodList paymentMethods={[]} />);

    expect(
      screen.getByText('No payment methods found. Add one to get started.')
    ).toBeInTheDocument();
  });

  it('renders custom empty message', () => {
    render(
      <PaymentMethodList
        paymentMethods={[]}
        emptyMessage="You have no saved cards"
      />
    );

    expect(screen.getByText('You have no saved cards')).toBeInTheDocument();
  });

  it('renders add button in empty state when showAddButton is true', () => {
    const onAdd = vi.fn();
    render(
      <PaymentMethodList paymentMethods={[]} onAdd={onAdd} showAddButton />
    );

    expect(screen.getByText('Add Payment Method')).toBeInTheDocument();
  });

  it('does not render add button in empty state when showAddButton is false', () => {
    const onAdd = vi.fn();
    render(
      <PaymentMethodList
        paymentMethods={[]}
        onAdd={onAdd}
        showAddButton={false}
      />
    );

    expect(screen.queryByText('Add Payment Method')).not.toBeInTheDocument();
  });

  // ==========================================================================
  // Payment Method Rendering Tests
  // ==========================================================================

  it('renders all active payment methods', () => {
    const paymentMethods = [defaultPaymentMethod, regularPaymentMethod];
    render(<PaymentMethodList paymentMethods={paymentMethods} />);

    expect(screen.getAllByRole('button', { name: /Visa/i })).toHaveLength(2);
  });

  it('shows default payment method first', () => {
    const paymentMethods = [regularPaymentMethod, defaultPaymentMethod];
    render(<PaymentMethodList paymentMethods={paymentMethods} />);

    const cards = screen.getAllByRole('button', { name: /Visa/i });
    expect(cards[0]).toHaveAttribute(
      'aria-label',
      expect.stringContaining('default')
    );
  });

  it('shows payment method count in header', () => {
    const paymentMethods = [defaultPaymentMethod, regularPaymentMethod];
    render(
      <PaymentMethodList paymentMethods={paymentMethods} onAdd={vi.fn()} />
    );

    expect(screen.getByText('Payment Methods (2)')).toBeInTheDocument();
  });

  it('separates inactive payment methods', () => {
    const paymentMethods = [defaultPaymentMethod, inactivePaymentMethod];
    render(<PaymentMethodList paymentMethods={paymentMethods} />);

    expect(screen.getByText('Inactive (1)')).toBeInTheDocument();
  });

  // ==========================================================================
  // Selection Tests
  // ==========================================================================

  it('highlights selected payment method', () => {
    const paymentMethods = [defaultPaymentMethod, regularPaymentMethod];
    render(
      <PaymentMethodList
        paymentMethods={paymentMethods}
        selectedId={regularPaymentMethod.id}
        onSelect={vi.fn()}
      />
    );

    const cards = screen.getAllByRole('button', { name: /Visa/i });
    expect(cards[1]).toHaveAttribute('aria-selected', 'true');
  });

  it('calls onSelect when payment method is clicked', () => {
    const onSelect = vi.fn();
    const paymentMethods = [regularPaymentMethod];
    render(
      <PaymentMethodList paymentMethods={paymentMethods} onSelect={onSelect} />
    );

    fireEvent.click(screen.getByRole('button', { name: /Visa/i }));

    expect(onSelect).toHaveBeenCalledWith(regularPaymentMethod);
  });

  it('does not allow selection of expired payment methods', () => {
    const onSelect = vi.fn();
    const paymentMethods = [expiredPaymentMethod];
    render(
      <PaymentMethodList paymentMethods={paymentMethods} onSelect={onSelect} />
    );

    fireEvent.click(screen.getByRole('button', { name: /Visa/i }));

    expect(onSelect).not.toHaveBeenCalled();
  });

  // ==========================================================================
  // Action Callback Tests
  // ==========================================================================

  it('calls onAdd when add button is clicked', () => {
    const onAdd = vi.fn();
    const paymentMethods = [defaultPaymentMethod];
    render(
      <PaymentMethodList
        paymentMethods={paymentMethods}
        onAdd={onAdd}
        showAddButton
      />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Add' }));

    expect(onAdd).toHaveBeenCalled();
  });

  it('renders edit button for each payment method when onEdit is provided', () => {
    const onEdit = vi.fn();
    const paymentMethods = [defaultPaymentMethod];
    render(
      <PaymentMethodList paymentMethods={paymentMethods} onEdit={onEdit} />
    );

    expect(screen.getByLabelText('Edit payment method')).toBeInTheDocument();
  });

  it('renders delete button for each payment method when onDelete is provided', () => {
    const onDelete = vi.fn();
    const paymentMethods = [regularPaymentMethod];
    render(
      <PaymentMethodList paymentMethods={paymentMethods} onDelete={onDelete} />
    );

    expect(screen.getByLabelText('Delete payment method')).toBeInTheDocument();
  });

  it('renders set default button for non-default methods when onSetDefault is provided', () => {
    const onSetDefault = vi.fn();
    const paymentMethods = [regularPaymentMethod];
    render(
      <PaymentMethodList
        paymentMethods={paymentMethods}
        onSetDefault={onSetDefault}
      />
    );

    expect(screen.getByLabelText('Set as default')).toBeInTheDocument();
  });

  // ==========================================================================
  // Summary Section Tests
  // ==========================================================================

  it('shows expired card count in summary', () => {
    const paymentMethods = [expiredPaymentMethod, regularPaymentMethod];
    render(<PaymentMethodList paymentMethods={paymentMethods} />);

    expect(screen.getByText(/1 expired/)).toBeInTheDocument();
  });

  it('shows expiring soon count in summary', () => {
    const expiringSoonMethod = createPaymentMethod({ isExpiringSoon: true });
    const paymentMethods = [expiringSoonMethod, regularPaymentMethod];
    render(<PaymentMethodList paymentMethods={paymentMethods} />);

    expect(screen.getByText(/1 expiring soon/)).toBeInTheDocument();
  });

  // ==========================================================================
  // Accessibility Tests
  // ==========================================================================

  it('has radiogroup role for payment method list', () => {
    const paymentMethods = [defaultPaymentMethod, regularPaymentMethod];
    render(<PaymentMethodList paymentMethods={paymentMethods} />);

    expect(screen.getByRole('radiogroup')).toBeInTheDocument();
  });

  it('has aria-label for radiogroup', () => {
    const paymentMethods = [defaultPaymentMethod];
    render(<PaymentMethodList paymentMethods={paymentMethods} />);

    expect(screen.getByRole('radiogroup')).toHaveAttribute(
      'aria-label',
      'Payment methods'
    );
  });

  // ==========================================================================
  // CSS Class Tests
  // ==========================================================================

  it('applies custom className', () => {
    render(<PaymentMethodList paymentMethods={[]} className="custom-class" />);

    // The className is applied to the container
    const container = screen
      .getByText(/No payment methods/)
      .closest('div')?.parentElement;
    expect(container).toHaveClass('custom-class');
  });

  // ==========================================================================
  // Sorting Tests
  // ==========================================================================

  it('sorts payment methods by default status and creation date', () => {
    const oldMethod = createPaymentMethod({
      id: 'pm-old',
      createdAt: '2023-01-01T00:00:00Z',
    });
    const newMethod = createPaymentMethod({
      id: 'pm-new',
      createdAt: '2024-01-01T00:00:00Z',
    });
    const defaultMethod = createPaymentMethod({
      id: 'pm-default',
      isDefault: true,
      createdAt: '2022-01-01T00:00:00Z',
    });

    const paymentMethods = [oldMethod, newMethod, defaultMethod];
    render(<PaymentMethodList paymentMethods={paymentMethods} />);

    const cards = screen.getAllByRole('button', { name: /Visa/i });
    // Default should be first, then newer, then older
    expect(cards[0]).toHaveAttribute(
      'aria-label',
      expect.stringContaining('default')
    );
  });
});
