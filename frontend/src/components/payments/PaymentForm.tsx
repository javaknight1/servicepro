import { useState, useCallback, FormEvent } from 'react';
import {
  useStripe,
  useElements,
  CardElement,
  Elements,
} from '@stripe/react-stripe-js';
import { loadStripe, StripeError } from '@stripe/stripe-js';
import { cn } from '../../utils/cn';
import type {
  PaymentFormProps,
  BillingAddressData,
  BillingAddressErrors,
  PaymentResultData,
  PaymentStatus,
} from '../../types/payment';
import { formatCurrency, DEFAULT_BILLING_ADDRESS } from '../../types/payment';
import { CreditCardInput } from './CreditCardInput';
import { BillingAddress } from './BillingAddress';
import { PaymentSummary } from './PaymentSummary';
import { PaymentResult } from './PaymentResult';
import { Lock, ShieldCheck } from 'lucide-react';

// =============================================================================
// Stripe Promise (lazy load)
// =============================================================================

// Initialize Stripe with your publishable key
// This should come from environment variables in production
const stripePromise = loadStripe(
  import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY || 'pk_test_placeholder'
);

// =============================================================================
// Form Validation
// =============================================================================

function validateEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
}

function validateBillingAddress(
  address: BillingAddressData
): BillingAddressErrors {
  const errors: BillingAddressErrors = {};

  if (!address.name.trim()) {
    errors.name = 'Name is required';
  } else if (address.name.trim().length < 2) {
    errors.name = 'Name must be at least 2 characters';
  }

  if (!address.email.trim()) {
    errors.email = 'Email is required';
  } else if (!validateEmail(address.email)) {
    errors.email = 'Please enter a valid email address';
  }

  if (address.phone && !/^[\d\s\-+()]+$/.test(address.phone)) {
    errors.phone = 'Please enter a valid phone number';
  }

  if (!address.line1.trim()) {
    errors.line1 = 'Street address is required';
  }

  if (!address.city.trim()) {
    errors.city = 'City is required';
  }

  if (!address.state.trim()) {
    errors.state = 'State is required';
  }

  if (!address.postalCode.trim()) {
    errors.postalCode = 'ZIP code is required';
  } else if (
    address.country === 'US' &&
    !/^\d{5}(-\d{4})?$/.test(address.postalCode)
  ) {
    errors.postalCode = 'Please enter a valid ZIP code';
  }

  if (!address.country) {
    errors.country = 'Country is required';
  }

  return errors;
}

// =============================================================================
// Inner Payment Form (uses Stripe hooks)
// =============================================================================

interface InnerPaymentFormProps extends Omit<PaymentFormProps, 'clientSecret'> {
  clientSecret: string;
}

function InnerPaymentForm({
  amount,
  currency,
  summary,
  clientSecret,
  onSuccess,
  onError,
  onCancel,
  collectBillingAddress = true,
  defaultBillingAddress,
  submitButtonText,
  allowSavePaymentMethod = false,
  className,
}: InnerPaymentFormProps) {
  const stripe = useStripe();
  const elements = useElements();

  // Form state
  const [billingAddress, setBillingAddress] = useState<BillingAddressData>({
    ...DEFAULT_BILLING_ADDRESS,
    ...defaultBillingAddress,
  });
  const [addressErrors, setAddressErrors] = useState<BillingAddressErrors>({});
  const [savePaymentMethod, setSavePaymentMethod] = useState(false);

  // Card state
  const [cardComplete, setCardComplete] = useState(false);
  const [cardError, setCardError] = useState<string | undefined>();

  // Payment state
  const [status, setStatus] = useState<PaymentStatus>('idle');
  const [result, setResult] = useState<PaymentResultData | null>(null);

  const isProcessing = status === 'processing';
  const isComplete = status === 'succeeded' || status === 'failed';

  const handleCardChange = useCallback(
    (event: { complete: boolean; empty: boolean; error?: StripeError }) => {
      setCardComplete(event.complete);
      setCardError(event.error?.message);
    },
    []
  );

  const handleSubmit = useCallback(
    async (e: FormEvent) => {
      e.preventDefault();

      if (!stripe || !elements) {
        return;
      }

      // Validate billing address if collecting
      if (collectBillingAddress) {
        const errors = validateBillingAddress(billingAddress);
        setAddressErrors(errors);
        if (Object.keys(errors).length > 0) {
          return;
        }
      }

      // Validate card
      if (!cardComplete) {
        setCardError('Please enter your card details');
        return;
      }

      setStatus('processing');
      setCardError(undefined);

      try {
        const cardElement = elements.getElement(CardElement);
        if (!cardElement) {
          throw new Error('Card element not found');
        }

        const { error, paymentIntent } = await stripe.confirmCardPayment(
          clientSecret,
          {
            payment_method: {
              card: cardElement,
              billing_details: collectBillingAddress
                ? {
                    name: billingAddress.name,
                    email: billingAddress.email,
                    phone: billingAddress.phone || undefined,
                    address: {
                      line1: billingAddress.line1,
                      line2: billingAddress.line2 || undefined,
                      city: billingAddress.city,
                      state: billingAddress.state,
                      postal_code: billingAddress.postalCode,
                      country: billingAddress.country,
                    },
                  }
                : undefined,
            },
            setup_future_usage: savePaymentMethod ? 'off_session' : undefined,
          }
        );

        if (error) {
          setStatus('failed');
          const failedResult: PaymentResultData = {
            status: 'failed',
            error,
          };
          setResult(failedResult);
          onError?.(error);
        } else if (paymentIntent) {
          const successStatus =
            paymentIntent.status === 'succeeded'
              ? 'succeeded'
              : paymentIntent.status === 'requires_action'
                ? 'requires_action'
                : 'processing';

          setStatus(successStatus);

          const successResult: PaymentResultData = {
            status: successStatus,
            paymentIntentId: paymentIntent.id,
            amount: {
              amount: paymentIntent.amount,
              currency: paymentIntent.currency,
              displayAmount: formatCurrency(
                paymentIntent.amount,
                paymentIntent.currency
              ),
            },
            last4: (
              paymentIntent.payment_method as { card?: { last4: string } }
            )?.card?.last4,
            brand: (
              paymentIntent.payment_method as { card?: { brand: string } }
            )?.card?.brand,
          };

          setResult(successResult);

          if (successStatus === 'succeeded') {
            onSuccess?.(successResult);
          }
        }
      } catch (err) {
        setStatus('failed');
        const error = err instanceof Error ? err : new Error('Payment failed');
        const failedResult: PaymentResultData = {
          status: 'failed',
          error,
        };
        setResult(failedResult);
        onError?.(error);
      }
    },
    [
      stripe,
      elements,
      clientSecret,
      billingAddress,
      cardComplete,
      collectBillingAddress,
      savePaymentMethod,
      onSuccess,
      onError,
    ]
  );

  const handleRetry = useCallback(() => {
    setStatus('idle');
    setResult(null);
    setCardError(undefined);
  }, []);

  const handleCancel = useCallback(() => {
    onCancel?.();
  }, [onCancel]);

  // Show result screen when complete
  if (isComplete && result) {
    return (
      <div className={className}>
        <PaymentResult
          result={result}
          onRetry={status === 'failed' ? handleRetry : undefined}
          onClose={status === 'succeeded' ? handleCancel : handleRetry}
        />
      </div>
    );
  }

  const buttonText =
    submitButtonText || `Pay ${formatCurrency(amount, currency)}`;

  return (
    <form
      onSubmit={handleSubmit}
      className={cn('space-y-6', className)}
      noValidate
    >
      {/* Payment Summary */}
      {summary && <PaymentSummary summary={summary} isLoading={false} />}

      {/* Billing Address */}
      {collectBillingAddress && (
        <BillingAddress
          value={billingAddress}
          errors={addressErrors}
          onChange={setBillingAddress}
          disabled={isProcessing}
        />
      )}

      {/* Card Input */}
      <CreditCardInput
        disabled={isProcessing}
        error={cardError}
        onChange={handleCardChange}
      />

      {/* Save Payment Method */}
      {allowSavePaymentMethod && (
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={savePaymentMethod}
            onChange={(e) => setSavePaymentMethod(e.target.checked)}
            disabled={isProcessing}
            className={cn(
              'h-4 w-4 rounded border-neutral-300 text-primary-600',
              'focus:ring-primary-500 focus:ring-2 focus:ring-offset-0',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          />
          <span className="text-sm text-neutral-700">
            Save this card for future payments
          </span>
        </label>
      )}

      {/* Security Notice */}
      <div className="flex items-center justify-center gap-2 p-3 bg-neutral-50 rounded-lg border border-neutral-200">
        <ShieldCheck className="h-5 w-5 text-success-500" aria-hidden="true" />
        <span className="text-sm text-neutral-600">
          Your payment is secured with 256-bit SSL encryption
        </span>
      </div>

      {/* Submit Button */}
      <button
        type="submit"
        disabled={isProcessing || !stripe}
        className={cn(
          'w-full inline-flex items-center justify-center gap-2 px-6 py-3',
          'bg-primary-600 text-white rounded-lg font-semibold text-lg',
          'hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
          'disabled:opacity-50 disabled:cursor-not-allowed',
          'transition-colors'
        )}
        aria-busy={isProcessing}
      >
        {isProcessing ? (
          <>
            <svg
              className="animate-spin h-5 w-5"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
              />
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              />
            </svg>
            Processing...
          </>
        ) : (
          <>
            <Lock className="h-5 w-5" aria-hidden="true" />
            {buttonText}
          </>
        )}
      </button>

      {/* Cancel Link */}
      {onCancel && (
        <button
          type="button"
          onClick={handleCancel}
          disabled={isProcessing}
          className={cn(
            'w-full text-center text-sm text-neutral-600 hover:text-neutral-800',
            'focus:outline-none focus:underline',
            'disabled:opacity-50 disabled:cursor-not-allowed'
          )}
        >
          Cancel payment
        </button>
      )}
    </form>
  );
}

// =============================================================================
// Payment Form Wrapper (provides Stripe context)
// =============================================================================

export function PaymentForm(props: PaymentFormProps) {
  return (
    <Elements
      stripe={stripePromise}
      options={{
        clientSecret: props.clientSecret,
        appearance: {
          theme: 'stripe',
          variables: {
            colorPrimary: '#4f46e5',
            colorBackground: '#ffffff',
            colorText: '#1f2937',
            colorDanger: '#dc2626',
            fontFamily: 'ui-sans-serif, system-ui, sans-serif',
            borderRadius: '8px',
          },
        },
      }}
    >
      <InnerPaymentForm {...props} />
    </Elements>
  );
}

PaymentForm.displayName = 'PaymentForm';

export default PaymentForm;
