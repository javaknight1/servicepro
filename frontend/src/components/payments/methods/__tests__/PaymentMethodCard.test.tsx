import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { PaymentMethodCard } from '../PaymentMethodCard';
import type { StoredPaymentMethod } from '../../../../types/payment';

describe('PaymentMethodCard', () => {
  const basePaymentMethod: StoredPaymentMethod = {
    id: 'pm-123',
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
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  // ==========================================================================
  // Rendering Tests
  // ==========================================================================

  it('renders payment method card with display name', () => {
    render(<PaymentMethodCard paymentMethod={basePaymentMethod} />);

    expect(screen.getByText('Visa •••• 4242')).toBeInTheDocument();
  });

  it('renders card brand and last 4 digits', () => {
    render(<PaymentMethodCard paymentMethod={basePaymentMethod} />);

    // The display name contains both brand and last 4 digits
    expect(screen.getByText('Visa •••• 4242')).toBeInTheDocument();
  });

  it('renders expiry date', () => {
    render(<PaymentMethodCard paymentMethod={basePaymentMethod} />);

    const expYear = (basePaymentMethod.cardExpYear ?? 0).toString().slice(-2);
    expect(screen.getByText(`Exp: 12/${expYear}`)).toBeInTheDocument();
  });

  it('renders default badge when isDefault is true', () => {
    const defaultPM = { ...basePaymentMethod, isDefault: true };
    render(<PaymentMethodCard paymentMethod={defaultPM} />);

    expect(screen.getByText('Default')).toBeInTheDocument();
  });

  it('does not render default badge when isDefault is false', () => {
    render(<PaymentMethodCard paymentMethod={basePaymentMethod} />);

    expect(screen.queryByText('Default')).not.toBeInTheDocument();
  });

  it('renders nickname when provided', () => {
    const pmWithNickname = { ...basePaymentMethod, nickname: 'Personal Card' };
    render(<PaymentMethodCard paymentMethod={pmWithNickname} />);

    expect(screen.getByText('Personal Card')).toBeInTheDocument();
  });

  // ==========================================================================
  // Bank Account Tests
  // ==========================================================================

  it('renders bank account details', () => {
    const bankPM: StoredPaymentMethod = {
      ...basePaymentMethod,
      type: 'bank_account',
      cardBrand: undefined,
      cardLast4: undefined,
      cardExpMonth: undefined,
      cardExpYear: undefined,
      bankName: 'Chase',
      bankLast4: '6789',
      displayName: 'Chase •••• 6789',
    };
    render(<PaymentMethodCard paymentMethod={bankPM} />);

    // The display name contains both bank name and last 4 digits
    expect(screen.getByText('Chase •••• 6789')).toBeInTheDocument();
  });

  // ==========================================================================
  // Status Tests
  // ==========================================================================

  it('renders expired warning when card is expired', () => {
    const expiredPM = { ...basePaymentMethod, isExpired: true };
    render(<PaymentMethodCard paymentMethod={expiredPM} />);

    expect(screen.getByText('Card expired')).toBeInTheDocument();
  });

  it('renders expiring soon warning', () => {
    const expiringSoonPM = { ...basePaymentMethod, isExpiringSoon: true };
    render(<PaymentMethodCard paymentMethod={expiringSoonPM} />);

    expect(screen.getByText('Expiring soon')).toBeInTheDocument();
  });

  it('renders inactive status', () => {
    const inactivePM = { ...basePaymentMethod, isActive: false };
    render(<PaymentMethodCard paymentMethod={inactivePM} />);

    expect(screen.getByText('Inactive')).toBeInTheDocument();
  });

  // ==========================================================================
  // Selection Tests
  // ==========================================================================

  it('applies selected styles when isSelected is true', () => {
    render(<PaymentMethodCard paymentMethod={basePaymentMethod} isSelected />);

    const card = screen.getByRole('button');
    expect(card).toHaveAttribute('aria-selected', 'true');
  });

  it('calls onSelect when clicked', () => {
    const onSelect = vi.fn();
    render(
      <PaymentMethodCard
        paymentMethod={basePaymentMethod}
        onSelect={onSelect}
      />
    );

    fireEvent.click(screen.getByRole('button'));

    expect(onSelect).toHaveBeenCalledWith(basePaymentMethod);
  });

  it('calls onSelect when Enter key is pressed', () => {
    const onSelect = vi.fn();
    render(
      <PaymentMethodCard
        paymentMethod={basePaymentMethod}
        onSelect={onSelect}
      />
    );

    fireEvent.keyDown(screen.getByRole('button'), { key: 'Enter' });

    expect(onSelect).toHaveBeenCalledWith(basePaymentMethod);
  });

  it('calls onSelect when Space key is pressed', () => {
    const onSelect = vi.fn();
    render(
      <PaymentMethodCard
        paymentMethod={basePaymentMethod}
        onSelect={onSelect}
      />
    );

    fireEvent.keyDown(screen.getByRole('button'), { key: ' ' });

    expect(onSelect).toHaveBeenCalledWith(basePaymentMethod);
  });

  it('does not call onSelect when disabled', () => {
    const onSelect = vi.fn();
    render(
      <PaymentMethodCard
        paymentMethod={basePaymentMethod}
        onSelect={onSelect}
        disabled
      />
    );

    fireEvent.click(screen.getByRole('button'));

    expect(onSelect).not.toHaveBeenCalled();
  });

  // ==========================================================================
  // Action Button Tests
  // ==========================================================================

  it('renders edit button when onEdit is provided', () => {
    const onEdit = vi.fn();
    render(
      <PaymentMethodCard paymentMethod={basePaymentMethod} onEdit={onEdit} />
    );

    expect(screen.getByLabelText('Edit payment method')).toBeInTheDocument();
  });

  it('calls onEdit when edit button is clicked', () => {
    const onEdit = vi.fn();
    render(
      <PaymentMethodCard paymentMethod={basePaymentMethod} onEdit={onEdit} />
    );

    fireEvent.click(screen.getByLabelText('Edit payment method'));

    expect(onEdit).toHaveBeenCalledWith(basePaymentMethod);
  });

  it('renders delete button when onDelete is provided', () => {
    const onDelete = vi.fn();
    render(
      <PaymentMethodCard
        paymentMethod={basePaymentMethod}
        onDelete={onDelete}
      />
    );

    expect(screen.getByLabelText('Delete payment method')).toBeInTheDocument();
  });

  it('shows delete confirmation when delete button is clicked', () => {
    const onDelete = vi.fn();
    render(
      <PaymentMethodCard
        paymentMethod={basePaymentMethod}
        onDelete={onDelete}
      />
    );

    fireEvent.click(screen.getByLabelText('Delete payment method'));

    expect(screen.getByText('Delete?')).toBeInTheDocument();
    expect(screen.getByText('Yes')).toBeInTheDocument();
    expect(screen.getByText('No')).toBeInTheDocument();
  });

  it('calls onDelete when delete is confirmed', () => {
    const onDelete = vi.fn();
    render(
      <PaymentMethodCard
        paymentMethod={basePaymentMethod}
        onDelete={onDelete}
      />
    );

    fireEvent.click(screen.getByLabelText('Delete payment method'));
    fireEvent.click(screen.getByText('Yes'));

    expect(onDelete).toHaveBeenCalledWith(basePaymentMethod);
  });

  it('cancels delete confirmation when No is clicked', () => {
    const onDelete = vi.fn();
    render(
      <PaymentMethodCard
        paymentMethod={basePaymentMethod}
        onDelete={onDelete}
      />
    );

    fireEvent.click(screen.getByLabelText('Delete payment method'));
    fireEvent.click(screen.getByText('No'));

    expect(onDelete).not.toHaveBeenCalled();
    expect(screen.queryByText('Delete?')).not.toBeInTheDocument();
  });

  it('renders set default button when onSetDefault is provided and not default', () => {
    const onSetDefault = vi.fn();
    render(
      <PaymentMethodCard
        paymentMethod={basePaymentMethod}
        onSetDefault={onSetDefault}
      />
    );

    expect(screen.getByLabelText('Set as default')).toBeInTheDocument();
  });

  it('does not render set default button when already default', () => {
    const onSetDefault = vi.fn();
    const defaultPM = { ...basePaymentMethod, isDefault: true };
    render(
      <PaymentMethodCard
        paymentMethod={defaultPM}
        onSetDefault={onSetDefault}
      />
    );

    expect(screen.queryByLabelText('Set as default')).not.toBeInTheDocument();
  });

  it('calls onSetDefault when set default button is clicked', () => {
    const onSetDefault = vi.fn();
    render(
      <PaymentMethodCard
        paymentMethod={basePaymentMethod}
        onSetDefault={onSetDefault}
      />
    );

    fireEvent.click(screen.getByLabelText('Set as default'));

    expect(onSetDefault).toHaveBeenCalledWith(basePaymentMethod);
  });

  // ==========================================================================
  // Accessibility Tests
  // ==========================================================================

  it('has correct aria-label for card', () => {
    render(<PaymentMethodCard paymentMethod={basePaymentMethod} />);

    const card = screen.getByRole('button');
    expect(card).toHaveAttribute('aria-label', 'Visa •••• 4242');
  });

  it('includes default in aria-label when default', () => {
    const defaultPM = { ...basePaymentMethod, isDefault: true };
    render(<PaymentMethodCard paymentMethod={defaultPM} />);

    const card = screen.getByRole('button');
    expect(card).toHaveAttribute(
      'aria-label',
      'Visa •••• 4242, default payment method'
    );
  });

  it('has aria-disabled when disabled', () => {
    render(<PaymentMethodCard paymentMethod={basePaymentMethod} disabled />);

    const card = screen.getByRole('button');
    expect(card).toHaveAttribute('aria-disabled', 'true');
  });

  // ==========================================================================
  // CSS Class Tests
  // ==========================================================================

  it('applies custom className', () => {
    render(
      <PaymentMethodCard
        paymentMethod={basePaymentMethod}
        className="custom-class"
      />
    );

    const card = screen.getByRole('button');
    expect(card).toHaveClass('custom-class');
  });
});
