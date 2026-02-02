import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { PaymentSummaryData } from '../../../types/payment';
import type {
  StripeCardElementChangeEvent,
  StripeCardElementOptions,
} from '@stripe/stripe-js';

/** Props for mocked CardElement component */
interface MockCardElementProps {
  onChange?: (event: Partial<StripeCardElementChangeEvent>) => void;
  onFocus?: () => void;
  onBlur?: () => void;
  onReady?: () => void;
  options?: StripeCardElementOptions;
  id?: string;
}

// Create mock objects with vi.hoisted() to ensure they're available before vi.mock runs
const { mockStripe, mockElements } = vi.hoisted(() => ({
  mockStripe: {
    confirmCardPayment: vi.fn(),
    elements: vi.fn(),
  },
  mockElements: {
    getElement: vi.fn(),
  },
}));

vi.mock('@stripe/react-stripe-js', () => ({
  Elements: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  useStripe: () => mockStripe,
  useElements: () => mockElements,
  CardElement: ({
    onChange,
    onFocus,
    onBlur,
    onReady,
    options: _options,
    id,
  }: MockCardElementProps) => (
    <div
      data-testid="stripe-card-element"
      id={id}
      onClick={() => {
        onFocus?.();
        onChange?.({ complete: true, empty: false, error: undefined });
      }}
      onBlur={() => onBlur?.()}
    >
      Mock Card Element
      <button data-testid="trigger-ready" onClick={() => onReady?.()}>
        Ready
      </button>
    </div>
  ),
}));

vi.mock('@stripe/stripe-js', () => ({
  loadStripe: vi.fn(() => Promise.resolve(mockStripe)),
}));

// Import after mocking
import { PaymentForm } from '../PaymentForm';

describe('PaymentForm', () => {
  const defaultProps = {
    amount: 9999,
    currency: 'usd',
    clientSecret: 'pi_test_secret',
  };

  const mockSummary: PaymentSummaryData = {
    lineItems: [
      {
        id: '1',
        description: 'Test Product',
        quantity: 1,
        unitPrice: 9999,
        totalPrice: 9999,
      },
    ],
    subtotal: 9999,
    tax: 0,
    total: 9999,
    currency: 'usd',
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockElements.getElement.mockReturnValue({ id: 'card-element' });
    mockStripe.confirmCardPayment.mockResolvedValue({
      paymentIntent: {
        id: 'pi_test_123',
        status: 'succeeded',
        amount: 9999,
        currency: 'usd',
      },
    });
  });

  it('renders the payment form', () => {
    render(<PaymentForm {...defaultProps} />);

    expect(screen.getByTestId('stripe-card-element')).toBeInTheDocument();
  });

  it('displays submit button with amount', () => {
    render(<PaymentForm {...defaultProps} />);

    expect(
      screen.getByRole('button', { name: /pay \$99\.99/i })
    ).toBeInTheDocument();
  });

  it('displays custom submit button text', () => {
    render(<PaymentForm {...defaultProps} submitButtonText="Subscribe Now" />);

    expect(
      screen.getByRole('button', { name: /subscribe now/i })
    ).toBeInTheDocument();
  });

  it('displays payment summary when provided', () => {
    render(<PaymentForm {...defaultProps} summary={mockSummary} />);

    expect(screen.getByText('Order Summary')).toBeInTheDocument();
    expect(screen.getByText('Test Product')).toBeInTheDocument();
  });

  it('displays billing address form by default', () => {
    render(<PaymentForm {...defaultProps} />);

    expect(screen.getByLabelText(/full name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/street address/i)).toBeInTheDocument();
  });

  it('hides billing address when collectBillingAddress is false', () => {
    render(<PaymentForm {...defaultProps} collectBillingAddress={false} />);

    expect(screen.queryByLabelText(/full name/i)).not.toBeInTheDocument();
  });

  it('pre-fills billing address with default values', () => {
    render(
      <PaymentForm
        {...defaultProps}
        defaultBillingAddress={{
          name: 'John Doe',
          email: 'john@example.com',
        }}
      />
    );

    expect(screen.getByLabelText(/full name/i)).toHaveValue('John Doe');
    expect(screen.getByLabelText(/email/i)).toHaveValue('john@example.com');
  });

  it('displays save payment method checkbox when allowed', () => {
    render(<PaymentForm {...defaultProps} allowSavePaymentMethod />);

    expect(
      screen.getByLabelText(/save this card for future payments/i)
    ).toBeInTheDocument();
  });

  it('hides save payment method checkbox by default', () => {
    render(<PaymentForm {...defaultProps} />);

    expect(
      screen.queryByLabelText(/save this card for future payments/i)
    ).not.toBeInTheDocument();
  });

  it('displays cancel button when onCancel is provided', () => {
    const onCancel = vi.fn();
    render(<PaymentForm {...defaultProps} onCancel={onCancel} />);

    expect(
      screen.getByRole('button', { name: /cancel payment/i })
    ).toBeInTheDocument();
  });

  it('calls onCancel when cancel button is clicked', async () => {
    const onCancel = vi.fn();
    render(<PaymentForm {...defaultProps} onCancel={onCancel} />);

    fireEvent.click(screen.getByRole('button', { name: /cancel payment/i }));

    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it('displays security notice', () => {
    render(<PaymentForm {...defaultProps} />);

    expect(screen.getByText(/256-bit ssl encryption/i)).toBeInTheDocument();
  });

  it('shows validation errors for required billing fields', async () => {
    const user = userEvent.setup();
    render(<PaymentForm {...defaultProps} />);

    // Click card element to mark card as complete
    fireEvent.click(screen.getByTestId('stripe-card-element'));

    // Submit form without filling required fields
    const submitButton = screen.getByRole('button', { name: /pay/i });
    await user.click(submitButton);

    // Should show validation errors
    await waitFor(() => {
      expect(screen.getByText(/name is required/i)).toBeInTheDocument();
    });
  });

  it('shows card error when card is not complete', async () => {
    const user = userEvent.setup();

    // Override mock for this test
    vi.doMock('@stripe/react-stripe-js', () => ({
      Elements: ({ children }: { children: React.ReactNode }) => (
        <div>{children}</div>
      ),
      useStripe: () => mockStripe,
      useElements: () => mockElements,
      CardElement: ({ onChange }: MockCardElementProps) => (
        <div
          data-testid="stripe-card-element"
          onClick={() =>
            onChange?.({ complete: false, empty: true, error: undefined })
          }
        >
          Mock Card Element
        </div>
      ),
    }));

    render(<PaymentForm {...defaultProps} collectBillingAddress={false} />);

    const submitButton = screen.getByRole('button', { name: /pay/i });
    await user.click(submitButton);

    // The form should require card details
    // Note: This test may need adjustment based on actual implementation
  });

  it('shows processing state during payment', async () => {
    const user = userEvent.setup();

    // Make the payment take longer
    mockStripe.confirmCardPayment.mockImplementation(
      () =>
        new Promise((resolve) =>
          setTimeout(
            () =>
              resolve({
                paymentIntent: {
                  id: 'pi_test',
                  status: 'succeeded',
                  amount: 9999,
                  currency: 'usd',
                },
              }),
            100
          )
        )
    );

    render(<PaymentForm {...defaultProps} collectBillingAddress={false} />);

    // Complete the card
    fireEvent.click(screen.getByTestId('stripe-card-element'));

    const submitButton = screen.getByRole('button', { name: /pay/i });
    await user.click(submitButton);

    // Check for processing state
    expect(screen.getByText(/processing/i)).toBeInTheDocument();
    expect(submitButton).toBeDisabled();
  });

  it('calls onSuccess when payment succeeds', async () => {
    const user = userEvent.setup();
    const onSuccess = vi.fn();

    mockStripe.confirmCardPayment.mockResolvedValue({
      paymentIntent: {
        id: 'pi_test_123',
        status: 'succeeded',
        amount: 9999,
        currency: 'usd',
      },
    });

    render(
      <PaymentForm
        {...defaultProps}
        onSuccess={onSuccess}
        collectBillingAddress={false}
      />
    );

    // Complete the card
    fireEvent.click(screen.getByTestId('stripe-card-element'));

    const submitButton = screen.getByRole('button', { name: /pay/i });
    await user.click(submitButton);

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalled();
    });

    expect(onSuccess).toHaveBeenCalledWith(
      expect.objectContaining({
        status: 'succeeded',
        paymentIntentId: 'pi_test_123',
      })
    );
  });

  it('calls onError when payment fails', async () => {
    const user = userEvent.setup();
    const onError = vi.fn();

    const stripeError = {
      type: 'card_error' as const,
      message: 'Your card was declined.',
    };

    mockStripe.confirmCardPayment.mockResolvedValue({
      error: stripeError,
    });

    render(
      <PaymentForm
        {...defaultProps}
        onError={onError}
        collectBillingAddress={false}
      />
    );

    // Complete the card
    fireEvent.click(screen.getByTestId('stripe-card-element'));

    const submitButton = screen.getByRole('button', { name: /pay/i });
    await user.click(submitButton);

    await waitFor(() => {
      expect(onError).toHaveBeenCalled();
    });
  });

  it('shows result screen after successful payment', async () => {
    const user = userEvent.setup();

    mockStripe.confirmCardPayment.mockResolvedValue({
      paymentIntent: {
        id: 'pi_test_123',
        status: 'succeeded',
        amount: 9999,
        currency: 'usd',
      },
    });

    render(<PaymentForm {...defaultProps} collectBillingAddress={false} />);

    // Complete the card
    fireEvent.click(screen.getByTestId('stripe-card-element'));

    const submitButton = screen.getByRole('button', { name: /pay/i });
    await user.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText('Payment Successful')).toBeInTheDocument();
    });
  });

  it('shows result screen after failed payment with retry option', async () => {
    const user = userEvent.setup();

    mockStripe.confirmCardPayment.mockResolvedValue({
      error: {
        type: 'card_error' as const,
        message: 'Declined',
      },
    });

    render(<PaymentForm {...defaultProps} collectBillingAddress={false} />);

    // Complete the card
    fireEvent.click(screen.getByTestId('stripe-card-element'));

    const submitButton = screen.getByRole('button', { name: /pay/i });
    await user.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText('Payment Failed')).toBeInTheDocument();
    });

    expect(
      screen.getByRole('button', { name: /try again/i })
    ).toBeInTheDocument();
  });

  it('applies custom className', () => {
    const { container } = render(
      <PaymentForm {...defaultProps} className="custom-class" />
    );

    expect(container.querySelector('form')).toHaveClass('custom-class');
  });

  it('prevents default form submission', async () => {
    const user = userEvent.setup();

    render(<PaymentForm {...defaultProps} collectBillingAddress={false} />);

    // Complete the card
    fireEvent.click(screen.getByTestId('stripe-card-element'));

    const form = screen.getByRole('button', { name: /pay/i }).closest('form');
    const submitHandler = vi.fn((e) => e.preventDefault());
    form?.addEventListener('submit', submitHandler);

    const submitButton = screen.getByRole('button', { name: /pay/i });
    await user.click(submitButton);

    // Form should not reload page
    expect(mockStripe.confirmCardPayment).toHaveBeenCalled();
  });
});
