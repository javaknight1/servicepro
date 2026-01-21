import { useCallback, useState } from 'react';
import { CardElement } from '@stripe/react-stripe-js';
import type { StripeCardElementChangeEvent } from '@stripe/stripe-js';
import { cn } from '../../utils/cn';
import type { CreditCardInputProps } from '../../types/payment';
import { CreditCard, Lock } from 'lucide-react';

// =============================================================================
// Stripe Card Element Styles
// =============================================================================

const CARD_ELEMENT_OPTIONS = {
  style: {
    base: {
      fontSize: '16px',
      fontFamily: 'ui-sans-serif, system-ui, sans-serif',
      fontSmoothing: 'antialiased',
      color: '#1f2937',
      '::placeholder': {
        color: '#9ca3af',
      },
      iconColor: '#6b7280',
    },
    invalid: {
      color: '#dc2626',
      iconColor: '#dc2626',
    },
  },
  hidePostalCode: true,
};

// =============================================================================
// Credit Card Input Component
// =============================================================================

export function CreditCardInput({
  disabled = false,
  error,
  onChange,
  onReady,
  onFocus,
  onBlur,
  className,
}: CreditCardInputProps) {
  const [isFocused, setIsFocused] = useState(false);
  const [cardComplete, setCardComplete] = useState(false);
  const [cardError, setCardError] = useState<string | undefined>(undefined);

  const handleChange = useCallback(
    (event: StripeCardElementChangeEvent) => {
      setCardComplete(event.complete);
      setCardError(event.error?.message);

      onChange?.({
        complete: event.complete,
        empty: event.empty,
        error: event.error,
      });
    },
    [onChange]
  );

  const handleFocus = useCallback(() => {
    setIsFocused(true);
    onFocus?.();
  }, [onFocus]);

  const handleBlur = useCallback(() => {
    setIsFocused(false);
    onBlur?.();
  }, [onBlur]);

  const handleReady = useCallback(() => {
    onReady?.();
  }, [onReady]);

  const displayError = error || cardError;

  return (
    <div className={cn('flex flex-col', className)}>
      <label
        htmlFor="card-element"
        className="mb-1.5 text-sm font-medium text-neutral-700 flex items-center gap-2"
      >
        <CreditCard className="h-4 w-4" aria-hidden="true" />
        Card Details
      </label>

      <div
        className={cn(
          'relative px-3 py-3 border rounded-lg bg-white transition-all',
          'focus-within:ring-2 focus-within:ring-offset-0',
          disabled && 'bg-neutral-100 cursor-not-allowed opacity-60',
          displayError
            ? 'border-error-500 focus-within:ring-error-500 focus-within:border-error-500'
            : isFocused
              ? 'border-primary-500 ring-2 ring-primary-500'
              : 'border-neutral-300 focus-within:ring-primary-500 focus-within:border-primary-500'
        )}
        role="group"
        aria-label="Credit card input"
        aria-invalid={!!displayError}
        aria-describedby={displayError ? 'card-error' : undefined}
      >
        <CardElement
          id="card-element"
          options={{
            ...CARD_ELEMENT_OPTIONS,
            disabled,
          }}
          onChange={handleChange}
          onFocus={handleFocus}
          onBlur={handleBlur}
          onReady={handleReady}
        />

        {/* Card completion indicator */}
        {cardComplete && !displayError && (
          <div
            className="absolute right-3 top-1/2 -translate-y-1/2 text-success-500"
            aria-hidden="true"
          >
            <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                clipRule="evenodd"
              />
            </svg>
          </div>
        )}
      </div>

      {/* Error message */}
      {displayError && (
        <p
          id="card-error"
          className="mt-1.5 text-sm text-error-600 flex items-center gap-1"
          role="alert"
        >
          <svg
            className="h-4 w-4 flex-shrink-0"
            fill="currentColor"
            viewBox="0 0 20 20"
            aria-hidden="true"
          >
            <path
              fillRule="evenodd"
              d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
          {displayError}
        </p>
      )}

      {/* Security notice */}
      <div className="mt-2 flex items-center gap-1.5 text-xs text-neutral-500">
        <Lock className="h-3 w-3" aria-hidden="true" />
        <span>Your payment info is encrypted and secure</span>
      </div>
    </div>
  );
}

CreditCardInput.displayName = 'CreditCardInput';

export default CreditCardInput;
