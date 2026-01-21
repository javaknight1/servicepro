import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { StripeCardElementChangeEvent } from '@stripe/stripe-js';

// Mock Stripe Elements
vi.mock('@stripe/react-stripe-js', () => ({
  CardElement: ({ onChange, onFocus, onBlur, onReady, options, id }: any) => (
    <div
      data-testid="stripe-card-element"
      id={id}
      data-disabled={options?.disabled}
      onClick={() => onFocus?.()}
      onBlur={() => onBlur?.()}
    >
      <button
        data-testid="card-complete"
        onClick={() =>
          onChange?.({
            complete: true,
            empty: false,
            error: undefined,
          } as StripeCardElementChangeEvent)
        }
      >
        Complete Card
      </button>
      <button
        data-testid="card-error"
        onClick={() =>
          onChange?.({
            complete: false,
            empty: false,
            error: { message: 'Invalid card number' },
          } as StripeCardElementChangeEvent)
        }
      >
        Card Error
      </button>
      <button data-testid="card-ready" onClick={() => onReady?.()}>
        Ready
      </button>
    </div>
  ),
}));

import { CreditCardInput } from '../CreditCardInput';

describe('CreditCardInput', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the card element', () => {
    render(<CreditCardInput />);

    expect(screen.getByTestId('stripe-card-element')).toBeInTheDocument();
  });

  it('displays card details label', () => {
    render(<CreditCardInput />);

    expect(screen.getByText('Card Details')).toBeInTheDocument();
  });

  it('displays security notice', () => {
    render(<CreditCardInput />);

    expect(
      screen.getByText('Your payment info is encrypted and secure')
    ).toBeInTheDocument();
  });

  it('calls onChange when card is completed', () => {
    const onChange = vi.fn();
    render(<CreditCardInput onChange={onChange} />);

    fireEvent.click(screen.getByTestId('card-complete'));

    expect(onChange).toHaveBeenCalledWith({
      complete: true,
      empty: false,
      error: undefined,
    });
  });

  it('calls onChange with error when card has error', () => {
    const onChange = vi.fn();
    render(<CreditCardInput onChange={onChange} />);

    fireEvent.click(screen.getByTestId('card-error'));

    expect(onChange).toHaveBeenCalledWith({
      complete: false,
      empty: false,
      error: { message: 'Invalid card number' },
    });
  });

  it('displays error message from onChange event', () => {
    render(<CreditCardInput />);

    fireEvent.click(screen.getByTestId('card-error'));

    expect(screen.getByText('Invalid card number')).toBeInTheDocument();
  });

  it('displays error prop message', () => {
    render(<CreditCardInput error="Please enter your card details" />);

    expect(
      screen.getByText('Please enter your card details')
    ).toBeInTheDocument();
  });

  it('calls onFocus when card element is focused', () => {
    const onFocus = vi.fn();
    render(<CreditCardInput onFocus={onFocus} />);

    fireEvent.click(screen.getByTestId('stripe-card-element'));

    expect(onFocus).toHaveBeenCalled();
  });

  it('calls onBlur when card element loses focus', () => {
    const onBlur = vi.fn();
    render(<CreditCardInput onBlur={onBlur} />);

    fireEvent.blur(screen.getByTestId('stripe-card-element'));

    expect(onBlur).toHaveBeenCalled();
  });

  it('calls onReady when card element is ready', () => {
    const onReady = vi.fn();
    render(<CreditCardInput onReady={onReady} />);

    fireEvent.click(screen.getByTestId('card-ready'));

    expect(onReady).toHaveBeenCalled();
  });

  it('passes disabled prop to card element', () => {
    render(<CreditCardInput disabled />);

    const cardElement = screen.getByTestId('stripe-card-element');
    expect(cardElement).toHaveAttribute('data-disabled', 'true');
  });

  it('shows completion indicator when card is complete', () => {
    render(<CreditCardInput />);

    fireEvent.click(screen.getByTestId('card-complete'));

    // The checkmark icon should appear
    const container = screen.getByRole('group');
    expect(container.querySelector('svg')).toBeInTheDocument();
  });

  it('applies custom className', () => {
    const { container } = render(<CreditCardInput className="custom-class" />);

    expect(container.firstChild).toHaveClass('custom-class');
  });

  it('has proper aria attributes', () => {
    render(<CreditCardInput />);

    const group = screen.getByRole('group');
    expect(group).toHaveAttribute('aria-label', 'Credit card input');
  });

  it('has aria-invalid when there is an error', () => {
    render(<CreditCardInput error="Error message" />);

    const group = screen.getByRole('group');
    expect(group).toHaveAttribute('aria-invalid', 'true');
  });

  it('has aria-describedby when there is an error', () => {
    render(<CreditCardInput error="Error message" />);

    const group = screen.getByRole('group');
    expect(group).toHaveAttribute('aria-describedby', 'card-error');

    const errorElement = document.getElementById('card-error');
    expect(errorElement).toHaveTextContent('Error message');
  });

  it('error message has alert role', () => {
    render(<CreditCardInput error="Error message" />);

    expect(screen.getByRole('alert')).toBeInTheDocument();
  });

  it('applies disabled styling when disabled', () => {
    const { container } = render(<CreditCardInput disabled />);

    const wrapper = container.querySelector('[role="group"]');
    expect(wrapper).toHaveClass('opacity-60');
  });

  it('prioritizes error prop over card error', () => {
    render(<CreditCardInput error="Prop error" />);

    // Trigger card error
    fireEvent.click(screen.getByTestId('card-error'));

    // Prop error should still be displayed
    expect(screen.getByText('Prop error')).toBeInTheDocument();
  });
});
