import { useState, useCallback, useEffect, useRef } from 'react';
import { loadStripe } from '@stripe/stripe-js';
import {
  Elements,
  CardElement,
  useStripe,
  useElements,
} from '@stripe/react-stripe-js';
import type { Stripe, StripeCardElementChangeEvent } from '@stripe/stripe-js';
import type {
  AddPaymentMethodProps,
  StoredPaymentMethod,
} from '../../../types/payment';
import { cn } from '../../../utils/cn';
import { DefaultPaymentToggle } from './DefaultPaymentToggle';

// Inner form component that uses Stripe hooks
interface AddPaymentMethodFormProps {
  customerId: string;
  allowSetDefault?: boolean;
  onSuccess?: (paymentMethod: StoredPaymentMethod) => void;
  onError?: (error: Error) => void;
  onCancel?: () => void;
}

function AddPaymentMethodForm({
  customerId,
  allowSetDefault = true,
  onSuccess,
  onError,
  onCancel,
}: AddPaymentMethodFormProps) {
  const stripe = useStripe();
  const elements = useElements();

  const [isProcessing, setIsProcessing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [cardComplete, setCardComplete] = useState(false);
  const [nickname, setNickname] = useState('');
  const [setAsDefault, setSetAsDefault] = useState(false);

  const handleCardChange = useCallback(
    (event: StripeCardElementChangeEvent) => {
      setCardComplete(event.complete);
      if (event.error) {
        setError(event.error.message);
      } else {
        setError(null);
      }
    },
    []
  );

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!stripe || !elements) {
      setError('Payment system not initialized');
      return;
    }

    const cardElement = elements.getElement(CardElement);
    if (!cardElement) {
      setError('Card element not found');
      return;
    }

    setIsProcessing(true);
    setError(null);

    try {
      // Create the payment method with Stripe
      const { paymentMethod, error: stripeError } =
        await stripe.createPaymentMethod({
          type: 'card',
          card: cardElement,
        });

      if (stripeError) {
        throw new Error(stripeError.message);
      }

      if (!paymentMethod) {
        throw new Error('Failed to create payment method');
      }

      // Send to backend to attach to customer
      const response = await fetch(
        `/api/customers/${customerId}/payment-methods`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            stripe_payment_method_id: paymentMethod.id,
            nickname: nickname || undefined,
            set_as_default: setAsDefault,
          }),
        }
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || 'Failed to save payment method');
      }

      const savedPaymentMethod: StoredPaymentMethod = await response.json();

      if (onSuccess) {
        onSuccess(savedPaymentMethod);
      }
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : 'An unexpected error occurred';
      setError(errorMessage);
      if (onError) {
        onError(err instanceof Error ? err : new Error(errorMessage));
      }
    } finally {
      setIsProcessing(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Card Element */}
      <div>
        <label
          htmlFor="card-element"
          className="block text-sm font-medium text-neutral-700 mb-2"
        >
          Card Information
        </label>
        <div
          className={cn(
            'rounded-lg border bg-white p-4 transition-colors',
            error ? 'border-red-300' : 'border-neutral-300',
            'focus-within:border-primary-500 focus-within:ring-1 focus-within:ring-primary-500'
          )}
        >
          <CardElement
            id="card-element"
            options={{
              style: {
                base: {
                  fontSize: '16px',
                  color: '#1f2937',
                  fontFamily: 'system-ui, -apple-system, sans-serif',
                  '::placeholder': {
                    color: '#9ca3af',
                  },
                },
                invalid: {
                  color: '#dc2626',
                  iconColor: '#dc2626',
                },
              },
              hidePostalCode: false,
            }}
            onChange={handleCardChange}
          />
        </div>
        {error && (
          <p className="mt-2 text-sm text-red-600" role="alert">
            {error}
          </p>
        )}
      </div>

      {/* Nickname */}
      <div>
        <label
          htmlFor="nickname"
          className="block text-sm font-medium text-neutral-700 mb-2"
        >
          Nickname{' '}
          <span className="text-neutral-500 font-normal">(optional)</span>
        </label>
        <input
          type="text"
          id="nickname"
          value={nickname}
          onChange={(e) => setNickname(e.target.value)}
          placeholder="e.g., Personal Card, Business Card"
          maxLength={100}
          className={cn(
            'block w-full rounded-lg border border-neutral-300 px-4 py-2.5',
            'text-neutral-900 placeholder:text-neutral-400',
            'focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500',
            'disabled:bg-neutral-50 disabled:text-neutral-500'
          )}
          disabled={isProcessing}
        />
      </div>

      {/* Set as Default */}
      {allowSetDefault && (
        <DefaultPaymentToggle
          isDefault={setAsDefault}
          onChange={setSetAsDefault}
          disabled={isProcessing}
          size="md"
        />
      )}

      {/* Security notice */}
      <div className="flex items-center gap-2 rounded-lg bg-neutral-50 px-4 py-3 text-sm text-neutral-600">
        <svg
          className="h-5 w-5 text-neutral-400 flex-shrink-0"
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path
            fillRule="evenodd"
            d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z"
            clipRule="evenodd"
          />
        </svg>
        <span>
          Your card information is encrypted and securely processed by Stripe.
          We never store your full card number.
        </span>
      </div>

      {/* Actions */}
      <div className="flex gap-3 pt-2">
        <button
          type="submit"
          disabled={!stripe || isProcessing || !cardComplete}
          className={cn(
            'flex-1 rounded-lg px-4 py-2.5 text-sm font-medium transition-colors',
            'bg-primary-600 text-white',
            'hover:bg-primary-700',
            'focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            'disabled:bg-neutral-300 disabled:cursor-not-allowed'
          )}
        >
          {isProcessing ? (
            <span className="flex items-center justify-center gap-2">
              <svg
                className="h-4 w-4 animate-spin"
                fill="none"
                viewBox="0 0 24 24"
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
              Adding Card...
            </span>
          ) : (
            'Add Card'
          )}
        </button>

        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            disabled={isProcessing}
            className={cn(
              'rounded-lg border border-neutral-300 bg-white px-4 py-2.5 text-sm font-medium text-neutral-700',
              'hover:bg-neutral-50',
              'focus:outline-none focus:ring-2 focus:ring-neutral-500 focus:ring-offset-2',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            Cancel
          </button>
        )}
      </div>
    </form>
  );
}

// Main component with modal and Stripe provider
export function AddPaymentMethod({
  stripePublishableKey,
  customerId,
  onSuccess,
  onError,
  onCancel,
  isOpen,
  allowSetDefault = true,
  className,
}: AddPaymentMethodProps) {
  const [stripePromise, setStripePromise] =
    useState<Promise<Stripe | null> | null>(null);
  const modalRef = useRef<HTMLDivElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);

  // Initialize Stripe
  useEffect(() => {
    if (stripePublishableKey) {
      setStripePromise(loadStripe(stripePublishableKey));
    }
  }, [stripePublishableKey]);

  // Handle modal focus management
  useEffect(() => {
    if (isOpen) {
      previousFocusRef.current = document.activeElement as HTMLElement;
      modalRef.current?.focus();
    } else if (previousFocusRef.current) {
      previousFocusRef.current.focus();
    }
  }, [isOpen]);

  // Handle escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen && onCancel) {
        onCancel();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onCancel]);

  // Prevent body scroll when modal is open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  if (!isOpen) return null;

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget && onCancel) {
      onCancel();
    }
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="add-payment-method-title"
      className={cn(
        'fixed inset-0 z-50 flex items-center justify-center p-4',
        'bg-black/50 backdrop-blur-sm'
      )}
      onClick={handleBackdropClick}
    >
      <div
        ref={modalRef}
        tabIndex={-1}
        className={cn(
          'relative w-full max-w-md rounded-xl bg-white shadow-xl',
          'transform transition-all',
          'max-h-[90vh] overflow-y-auto',
          className
        )}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="sticky top-0 z-10 flex items-center justify-between border-b border-neutral-200 bg-white px-6 py-4">
          <h2
            id="add-payment-method-title"
            className="text-lg font-semibold text-neutral-900"
          >
            Add Payment Method
          </h2>
          {onCancel && (
            <button
              type="button"
              onClick={onCancel}
              className="rounded-lg p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 focus:outline-none focus:ring-2 focus:ring-primary-500"
              aria-label="Close"
            >
              <svg
                className="h-5 w-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          )}
        </div>

        {/* Content */}
        <div className="p-6">
          {stripePromise ? (
            <Elements stripe={stripePromise}>
              <AddPaymentMethodForm
                customerId={customerId}
                allowSetDefault={allowSetDefault}
                onSuccess={onSuccess}
                onError={onError}
                onCancel={onCancel}
              />
            </Elements>
          ) : (
            <div className="flex items-center justify-center py-8">
              <svg
                className="h-8 w-8 animate-spin text-primary-600"
                fill="none"
                viewBox="0 0 24 24"
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
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
