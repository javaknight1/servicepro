import { cn } from '../../utils/cn';
import type { PaymentResultProps } from '../../types/payment';
import { formatCurrency, formatCardBrand } from '../../types/payment';
import {
  CheckCircle,
  XCircle,
  AlertCircle,
  RefreshCw,
  Download,
  ArrowRight,
  CreditCard,
  Clock,
} from 'lucide-react';

// =============================================================================
// Status Icons
// =============================================================================

const StatusIcon = {
  succeeded: CheckCircle,
  failed: XCircle,
  processing: Clock,
  requires_action: AlertCircle,
  canceled: XCircle,
  idle: Clock,
};

const StatusColors = {
  succeeded: {
    bg: 'bg-success-50',
    border: 'border-success-200',
    icon: 'text-success-500',
    text: 'text-success-700',
  },
  failed: {
    bg: 'bg-error-50',
    border: 'border-error-200',
    icon: 'text-error-500',
    text: 'text-error-700',
  },
  processing: {
    bg: 'bg-warning-50',
    border: 'border-warning-200',
    icon: 'text-warning-500',
    text: 'text-warning-700',
  },
  requires_action: {
    bg: 'bg-warning-50',
    border: 'border-warning-200',
    icon: 'text-warning-500',
    text: 'text-warning-700',
  },
  canceled: {
    bg: 'bg-neutral-50',
    border: 'border-neutral-200',
    icon: 'text-neutral-500',
    text: 'text-neutral-700',
  },
  idle: {
    bg: 'bg-neutral-50',
    border: 'border-neutral-200',
    icon: 'text-neutral-500',
    text: 'text-neutral-700',
  },
};

const StatusMessages = {
  succeeded: 'Payment Successful',
  failed: 'Payment Failed',
  processing: 'Processing Payment',
  requires_action: 'Action Required',
  canceled: 'Payment Canceled',
  idle: 'Pending',
};

// =============================================================================
// Payment Result Component
// =============================================================================

export function PaymentResult({
  result,
  onRetry,
  onClose,
  successMessage = 'Your payment has been processed successfully. A confirmation email will be sent to your address.',
  failureMessage = 'We were unable to process your payment. Please check your card details and try again.',
  className,
}: PaymentResultProps) {
  const { status, paymentIntentId, amount, error, receiptUrl, last4, brand } =
    result;

  const Icon = StatusIcon[status];
  const colors = StatusColors[status];
  const statusMessage = StatusMessages[status];

  const errorMessage =
    typeof error === 'string'
      ? error
      : error instanceof Error
        ? error.message
        : error?.message || 'An unknown error occurred';

  const isSuccess = status === 'succeeded';
  const isFailure = status === 'failed';
  const isProcessing = status === 'processing';
  const requiresAction = status === 'requires_action';

  return (
    <div
      className={cn(
        'rounded-xl border p-6 text-center',
        colors.bg,
        colors.border,
        className
      )}
      role="status"
      aria-live="polite"
      aria-label={`Payment ${statusMessage}`}
    >
      {/* Status Icon */}
      <div
        className={cn(
          'mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full',
          isSuccess && 'bg-success-100',
          isFailure && 'bg-error-100',
          isProcessing && 'bg-warning-100',
          requiresAction && 'bg-warning-100',
          status === 'canceled' && 'bg-neutral-100'
        )}
      >
        <Icon
          className={cn('h-8 w-8', colors.icon, isProcessing && 'animate-spin')}
          aria-hidden="true"
        />
      </div>

      {/* Status Message */}
      <h2 className={cn('text-xl font-semibold mb-2', colors.text)}>
        {statusMessage}
      </h2>

      {/* Description */}
      <p className="text-neutral-600 mb-4 max-w-sm mx-auto">
        {isSuccess && successMessage}
        {isFailure && failureMessage}
        {isProcessing && 'Please wait while we process your payment...'}
        {requiresAction &&
          'Additional verification is required to complete your payment.'}
        {status === 'canceled' && 'Your payment has been canceled.'}
      </p>

      {/* Error Details */}
      {isFailure && errorMessage && (
        <div
          className="mb-4 p-3 bg-error-100 border border-error-200 rounded-lg text-left"
          role="alert"
        >
          <p className="text-sm text-error-700">
            <span className="font-medium">Error: </span>
            {errorMessage}
          </p>
        </div>
      )}

      {/* Payment Details (Success) */}
      {isSuccess && amount && (
        <div className="mb-6 p-4 bg-white rounded-lg border border-neutral-200 text-left">
          <h3 className="text-sm font-medium text-neutral-500 mb-3">
            Payment Details
          </h3>
          <dl className="space-y-2">
            <div className="flex justify-between items-center">
              <dt className="text-sm text-neutral-600">Amount Paid</dt>
              <dd className="text-lg font-semibold text-neutral-900">
                {amount.displayAmount ||
                  formatCurrency(amount.amount, amount.currency)}
              </dd>
            </div>
            {last4 && brand && (
              <div className="flex justify-between items-center">
                <dt className="text-sm text-neutral-600 flex items-center gap-1.5">
                  <CreditCard className="h-4 w-4" aria-hidden="true" />
                  Card
                </dt>
                <dd className="text-sm text-neutral-900">
                  {formatCardBrand(brand)} ending in {last4}
                </dd>
              </div>
            )}
            {paymentIntentId && (
              <div className="flex justify-between items-center">
                <dt className="text-sm text-neutral-600">Reference</dt>
                <dd className="text-xs text-neutral-500 font-mono">
                  {paymentIntentId.slice(0, 20)}...
                </dd>
              </div>
            )}
          </dl>
        </div>
      )}

      {/* Actions */}
      <div className="flex flex-col sm:flex-row gap-3 justify-center">
        {/* Retry Button (Failure) */}
        {isFailure && onRetry && (
          <button
            onClick={onRetry}
            className={cn(
              'inline-flex items-center justify-center gap-2 px-4 py-2.5',
              'bg-primary-600 text-white rounded-lg font-medium',
              'hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
              'transition-colors'
            )}
          >
            <RefreshCw className="h-4 w-4" aria-hidden="true" />
            Try Again
          </button>
        )}

        {/* Download Receipt (Success) */}
        {isSuccess && receiptUrl && (
          <a
            href={receiptUrl}
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              'inline-flex items-center justify-center gap-2 px-4 py-2.5',
              'bg-white text-neutral-700 border border-neutral-300 rounded-lg font-medium',
              'hover:bg-neutral-50 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
              'transition-colors'
            )}
          >
            <Download className="h-4 w-4" aria-hidden="true" />
            Download Receipt
          </a>
        )}

        {/* Continue/Close Button */}
        {onClose && (
          <button
            onClick={onClose}
            className={cn(
              'inline-flex items-center justify-center gap-2 px-4 py-2.5',
              'rounded-lg font-medium transition-colors',
              'focus:outline-none focus:ring-2 focus:ring-offset-2',
              isSuccess
                ? 'bg-success-600 text-white hover:bg-success-700 focus:ring-success-500'
                : 'bg-white text-neutral-700 border border-neutral-300 hover:bg-neutral-50 focus:ring-neutral-500'
            )}
          >
            {isSuccess ? (
              <>
                Continue
                <ArrowRight className="h-4 w-4" aria-hidden="true" />
              </>
            ) : (
              'Close'
            )}
          </button>
        )}
      </div>

      {/* Processing Indicator */}
      {isProcessing && (
        <div className="mt-4 flex items-center justify-center gap-2 text-sm text-warning-600">
          <div className="h-2 w-2 bg-warning-500 rounded-full animate-pulse" />
          <span>This may take a moment...</span>
        </div>
      )}
    </div>
  );
}

PaymentResult.displayName = 'PaymentResult';

export default PaymentResult;
