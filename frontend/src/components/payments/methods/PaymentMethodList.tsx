import type {
  PaymentMethodListProps,
  StoredPaymentMethod,
} from '../../../types/payment';
import { cn } from '../../../utils/cn';
import { PaymentMethodCard } from './PaymentMethodCard';

// Loading skeleton for payment method card
function PaymentMethodSkeleton() {
  return (
    <div className="animate-pulse rounded-lg border border-neutral-200 bg-white p-4">
      <div className="flex items-start gap-3">
        <div className="h-5 w-8 rounded bg-neutral-200" />
        <div className="flex-1">
          <div className="h-4 w-32 rounded bg-neutral-200" />
          <div className="mt-2 h-3 w-48 rounded bg-neutral-200" />
        </div>
      </div>
    </div>
  );
}

// Empty state component
function EmptyState({
  message,
  onAdd,
}: {
  message: string;
  onAdd?: () => void;
}) {
  return (
    <div className="rounded-lg border-2 border-dashed border-neutral-300 bg-neutral-50 p-8 text-center">
      <svg
        className="mx-auto h-12 w-12 text-neutral-400"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 48 48"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M4 8h40M4 8v28a4 4 0 004 4h32a4 4 0 004-4V8M4 8l4-4h32l4 4M16 20h16M16 28h8"
        />
      </svg>
      <p className="mt-4 text-sm text-neutral-600">{message}</p>
      {onAdd && (
        <button
          type="button"
          onClick={onAdd}
          className={cn(
            'mt-4 inline-flex items-center gap-2 rounded-lg',
            'bg-primary-600 px-4 py-2 text-sm font-medium text-white',
            'hover:bg-primary-700',
            'focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2'
          )}
        >
          <svg
            className="h-4 w-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 4v16m8-8H4"
            />
          </svg>
          Add Payment Method
        </button>
      )}
    </div>
  );
}

// Error state component
function ErrorState({
  message,
  onRetry,
}: {
  message: string;
  onRetry?: () => void;
}) {
  return (
    <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-center">
      <svg
        className="mx-auto h-10 w-10 text-red-500"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
        />
      </svg>
      <p className="mt-3 text-sm text-red-700">{message}</p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className={cn(
            'mt-4 inline-flex items-center gap-2 rounded-lg',
            'border border-red-300 bg-white px-4 py-2 text-sm font-medium text-red-700',
            'hover:bg-red-50',
            'focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2'
          )}
        >
          <svg
            className="h-4 w-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
          Try Again
        </button>
      )}
    </div>
  );
}

export function PaymentMethodList({
  paymentMethods,
  selectedId,
  onSelect,
  onAdd,
  onEdit,
  onDelete,
  onSetDefault,
  isLoading = false,
  error,
  emptyMessage = 'No payment methods found. Add one to get started.',
  showAddButton = true,
  className,
}: PaymentMethodListProps) {
  // Sort payment methods: default first, then by creation date (newest first)
  const sortedPaymentMethods = [...paymentMethods].sort((a, b) => {
    if (a.isDefault && !b.isDefault) return -1;
    if (!a.isDefault && b.isDefault) return 1;
    return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
  });

  // Group by active status
  const activePaymentMethods = sortedPaymentMethods.filter((pm) => pm.isActive);
  const inactivePaymentMethods = sortedPaymentMethods.filter(
    (pm) => !pm.isActive
  );

  // Handle selection
  const handleSelect = (paymentMethod: StoredPaymentMethod) => {
    if (onSelect && paymentMethod.isActive && !paymentMethod.isExpired) {
      onSelect(paymentMethod);
    }
  };

  if (isLoading) {
    return (
      <div
        className={cn('space-y-3', className)}
        role="status"
        aria-label="Loading payment methods"
      >
        <PaymentMethodSkeleton />
        <PaymentMethodSkeleton />
        <PaymentMethodSkeleton />
      </div>
    );
  }

  if (error) {
    return (
      <div className={className}>
        <ErrorState message={error} />
      </div>
    );
  }

  if (paymentMethods.length === 0) {
    return (
      <div className={className}>
        <EmptyState
          message={emptyMessage}
          onAdd={showAddButton ? onAdd : undefined}
        />
      </div>
    );
  }

  return (
    <div className={cn('space-y-4', className)}>
      {/* Header with Add Button */}
      {showAddButton && onAdd && (
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-neutral-700">
            Payment Methods ({paymentMethods.length})
          </h3>
          <button
            type="button"
            onClick={onAdd}
            className={cn(
              'inline-flex items-center gap-1.5 rounded-lg',
              'bg-primary-600 px-3 py-1.5 text-sm font-medium text-white',
              'hover:bg-primary-700',
              'focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2'
            )}
          >
            <svg
              className="h-4 w-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 4v16m8-8H4"
              />
            </svg>
            Add
          </button>
        </div>
      )}

      {/* Active Payment Methods */}
      <div className="space-y-2" role="radiogroup" aria-label="Payment methods">
        {activePaymentMethods.map((paymentMethod) => (
          <PaymentMethodCard
            key={paymentMethod.id}
            paymentMethod={paymentMethod}
            isSelected={selectedId === paymentMethod.id}
            onSelect={onSelect ? handleSelect : undefined}
            onEdit={onEdit}
            onDelete={onDelete}
            onSetDefault={onSetDefault}
            disabled={paymentMethod.isExpired}
          />
        ))}
      </div>

      {/* Inactive Payment Methods */}
      {inactivePaymentMethods.length > 0 && (
        <div className="mt-6">
          <h4 className="mb-2 text-sm font-medium text-neutral-500">
            Inactive ({inactivePaymentMethods.length})
          </h4>
          <div className="space-y-2">
            {inactivePaymentMethods.map((paymentMethod) => (
              <PaymentMethodCard
                key={paymentMethod.id}
                paymentMethod={paymentMethod}
                isSelected={false}
                onEdit={onEdit}
                onDelete={onDelete}
                disabled
              />
            ))}
          </div>
        </div>
      )}

      {/* Summary */}
      {paymentMethods.length > 0 && (
        <div className="rounded-lg bg-neutral-50 px-4 py-3 text-xs text-neutral-500">
          <div className="flex items-center gap-4">
            {activePaymentMethods.some((pm) => pm.isExpired) && (
              <span className="flex items-center gap-1 text-red-600">
                <svg
                  className="h-3.5 w-3.5"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                    clipRule="evenodd"
                  />
                </svg>
                {activePaymentMethods.filter((pm) => pm.isExpired).length}{' '}
                expired
              </span>
            )}
            {activePaymentMethods.some(
              (pm) => pm.isExpiringSoon && !pm.isExpired
            ) && (
              <span className="flex items-center gap-1 text-amber-600">
                <svg
                  className="h-3.5 w-3.5"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                    clipRule="evenodd"
                  />
                </svg>
                {
                  activePaymentMethods.filter(
                    (pm) => pm.isExpiringSoon && !pm.isExpired
                  ).length
                }{' '}
                expiring soon
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
