import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { PaymentResult } from '../PaymentResult';
import type { PaymentResultData } from '../../../types/payment';

describe('PaymentResult', () => {
  const successResult: PaymentResultData = {
    status: 'succeeded',
    paymentIntentId: 'pi_test_123456789',
    amount: {
      amount: 9999,
      currency: 'usd',
      displayAmount: '$99.99',
    },
    last4: '4242',
    brand: 'visa',
  };

  const failedResult: PaymentResultData = {
    status: 'failed',
    error: 'Your card was declined.',
  };

  const processingResult: PaymentResultData = {
    status: 'processing',
    paymentIntentId: 'pi_test_123456789',
  };

  const requiresActionResult: PaymentResultData = {
    status: 'requires_action',
    paymentIntentId: 'pi_test_123456789',
  };

  describe('Success State', () => {
    it('displays success message', () => {
      render(<PaymentResult result={successResult} />);

      expect(screen.getByText('Payment Successful')).toBeInTheDocument();
    });

    it('displays custom success message', () => {
      render(
        <PaymentResult
          result={successResult}
          successMessage="Thank you for your purchase!"
        />
      );

      expect(
        screen.getByText('Thank you for your purchase!')
      ).toBeInTheDocument();
    });

    it('displays payment amount', () => {
      render(<PaymentResult result={successResult} />);

      expect(screen.getByText('$99.99')).toBeInTheDocument();
    });

    it('displays card details', () => {
      render(<PaymentResult result={successResult} />);

      expect(screen.getByText('Visa ending in 4242')).toBeInTheDocument();
    });

    it('displays truncated payment reference', () => {
      render(<PaymentResult result={successResult} />);

      expect(screen.getByText(/pi_test_123456789/)).toBeInTheDocument();
    });

    it('shows continue button when onClose is provided', () => {
      const onClose = vi.fn();
      render(<PaymentResult result={successResult} onClose={onClose} />);

      expect(
        screen.getByRole('button', { name: /continue/i })
      ).toBeInTheDocument();
    });

    it('calls onClose when continue button is clicked', () => {
      const onClose = vi.fn();
      render(<PaymentResult result={successResult} onClose={onClose} />);

      fireEvent.click(screen.getByRole('button', { name: /continue/i }));
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('shows download receipt link when receiptUrl is provided', () => {
      const resultWithReceipt: PaymentResultData = {
        ...successResult,
        receiptUrl: 'https://stripe.com/receipt/123',
      };

      render(<PaymentResult result={resultWithReceipt} />);

      const receiptLink = screen.getByRole('link', {
        name: /download receipt/i,
      });
      expect(receiptLink).toHaveAttribute(
        'href',
        'https://stripe.com/receipt/123'
      );
      expect(receiptLink).toHaveAttribute('target', '_blank');
    });
  });

  describe('Failure State', () => {
    it('displays failure message', () => {
      render(<PaymentResult result={failedResult} />);

      expect(screen.getByText('Payment Failed')).toBeInTheDocument();
    });

    it('displays custom failure message', () => {
      render(
        <PaymentResult
          result={failedResult}
          failureMessage="Please contact support."
        />
      );

      expect(screen.getByText('Please contact support.')).toBeInTheDocument();
    });

    it('displays error details', () => {
      render(<PaymentResult result={failedResult} />);

      expect(screen.getByText(/your card was declined/i)).toBeInTheDocument();
    });

    it('shows retry button when onRetry is provided', () => {
      const onRetry = vi.fn();
      render(<PaymentResult result={failedResult} onRetry={onRetry} />);

      expect(
        screen.getByRole('button', { name: /try again/i })
      ).toBeInTheDocument();
    });

    it('calls onRetry when retry button is clicked', () => {
      const onRetry = vi.fn();
      render(<PaymentResult result={failedResult} onRetry={onRetry} />);

      fireEvent.click(screen.getByRole('button', { name: /try again/i }));
      expect(onRetry).toHaveBeenCalledTimes(1);
    });

    it('handles Error object', () => {
      const resultWithErrorObject: PaymentResultData = {
        status: 'failed',
        error: new Error('Network error'),
      };

      render(<PaymentResult result={resultWithErrorObject} />);

      expect(screen.getByText(/network error/i)).toBeInTheDocument();
    });

    it('handles Stripe error object', () => {
      const resultWithStripeError: PaymentResultData = {
        status: 'failed',
        error: {
          type: 'card_error',
          message: 'Insufficient funds',
        } as any,
      };

      render(<PaymentResult result={resultWithStripeError} />);

      expect(screen.getByText(/insufficient funds/i)).toBeInTheDocument();
    });
  });

  describe('Processing State', () => {
    it('displays processing message', () => {
      render(<PaymentResult result={processingResult} />);

      expect(screen.getByText('Processing Payment')).toBeInTheDocument();
    });

    it('displays processing indicator', () => {
      render(<PaymentResult result={processingResult} />);

      expect(screen.getByText('This may take a moment...')).toBeInTheDocument();
    });
  });

  describe('Requires Action State', () => {
    it('displays requires action message', () => {
      render(<PaymentResult result={requiresActionResult} />);

      expect(screen.getByText('Action Required')).toBeInTheDocument();
    });

    it('displays action required description', () => {
      render(<PaymentResult result={requiresActionResult} />);

      expect(
        screen.getByText(/additional verification is required/i)
      ).toBeInTheDocument();
    });
  });

  describe('Canceled State', () => {
    it('displays canceled message', () => {
      const canceledResult: PaymentResultData = {
        status: 'canceled',
      };

      render(<PaymentResult result={canceledResult} />);

      expect(screen.getByText('Payment Canceled')).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('has status role with aria-live', () => {
      render(<PaymentResult result={successResult} />);

      const statusElement = screen.getByRole('status');
      expect(statusElement).toHaveAttribute('aria-live', 'polite');
    });

    it('has accessible label for status', () => {
      render(<PaymentResult result={successResult} />);

      expect(screen.getByRole('status')).toHaveAttribute(
        'aria-label',
        'Payment Payment Successful'
      );
    });

    it('error alert has role="alert"', () => {
      render(<PaymentResult result={failedResult} />);

      expect(screen.getByRole('alert')).toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('applies custom className', () => {
      const { container } = render(
        <PaymentResult result={successResult} className="custom-class" />
      );

      expect(container.firstChild).toHaveClass('custom-class');
    });

    it('applies success styling for succeeded status', () => {
      const { container } = render(<PaymentResult result={successResult} />);

      expect(container.firstChild).toHaveClass('bg-success-50');
    });

    it('applies error styling for failed status', () => {
      const { container } = render(<PaymentResult result={failedResult} />);

      expect(container.firstChild).toHaveClass('bg-error-50');
    });
  });
});
