import { cn } from '../../utils/cn';
import type { PaymentSummaryProps } from '../../types/payment';
import { formatCurrency } from '../../types/payment';
import { Receipt, Tag, Percent } from 'lucide-react';

// =============================================================================
// Loading Skeleton
// =============================================================================

function LoadingSkeleton() {
  return (
    <div
      className="animate-pulse space-y-4"
      aria-busy="true"
      aria-label="Loading payment summary"
    >
      <div className="flex justify-between">
        <div className="h-4 w-24 bg-neutral-200 rounded" />
        <div className="h-4 w-16 bg-neutral-200 rounded" />
      </div>
      <div className="flex justify-between">
        <div className="h-4 w-20 bg-neutral-200 rounded" />
        <div className="h-4 w-14 bg-neutral-200 rounded" />
      </div>
      <div className="border-t pt-4">
        <div className="flex justify-between">
          <div className="h-5 w-16 bg-neutral-200 rounded" />
          <div className="h-5 w-20 bg-neutral-200 rounded" />
        </div>
      </div>
    </div>
  );
}

// =============================================================================
// Line Item Row
// =============================================================================

interface LineItemRowProps {
  description: string;
  quantity: number;
  unitPrice: number;
  totalPrice: number;
  currency: string;
}

function LineItemRow({
  description,
  quantity,
  unitPrice,
  totalPrice,
  currency,
}: LineItemRowProps) {
  return (
    <div className="flex items-start justify-between py-2 text-sm">
      <div className="flex-1 pr-4">
        <p className="text-neutral-900">{description}</p>
        {quantity > 1 && (
          <p className="text-neutral-500 text-xs">
            {quantity} x {formatCurrency(unitPrice, currency)}
          </p>
        )}
      </div>
      <p className="text-neutral-900 font-medium whitespace-nowrap">
        {formatCurrency(totalPrice, currency)}
      </p>
    </div>
  );
}

// =============================================================================
// Summary Row
// =============================================================================

interface SummaryRowProps {
  label: string;
  value: string;
  isDiscount?: boolean;
  isBold?: boolean;
  isLarge?: boolean;
}

function SummaryRow({
  label,
  value,
  isDiscount = false,
  isBold = false,
  isLarge = false,
}: SummaryRowProps) {
  return (
    <div
      className={cn(
        'flex justify-between items-center',
        isLarge ? 'py-3' : 'py-1.5'
      )}
    >
      <span
        className={cn(
          'text-neutral-600',
          isBold && 'font-semibold text-neutral-900',
          isLarge && 'text-lg'
        )}
      >
        {label}
      </span>
      <span
        className={cn(
          'text-neutral-900',
          isDiscount && 'text-success-600',
          isBold && 'font-semibold',
          isLarge && 'text-lg font-bold'
        )}
      >
        {isDiscount ? `-${value}` : value}
      </span>
    </div>
  );
}

// =============================================================================
// Payment Summary Component
// =============================================================================

export function PaymentSummary({
  summary,
  showLineItems = true,
  isLoading = false,
  className,
}: PaymentSummaryProps) {
  if (isLoading) {
    return (
      <div
        className={cn(
          'bg-neutral-50 rounded-lg border border-neutral-200 p-4',
          className
        )}
      >
        <LoadingSkeleton />
      </div>
    );
  }

  const {
    lineItems,
    subtotal,
    tax,
    taxRate,
    discount,
    discountCode,
    total,
    currency,
  } = summary;

  return (
    <div
      className={cn(
        'bg-neutral-50 rounded-lg border border-neutral-200',
        className
      )}
      role="region"
      aria-label="Payment summary"
    >
      {/* Header */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-neutral-200">
        <Receipt className="h-4 w-4 text-neutral-500" aria-hidden="true" />
        <h3 className="text-sm font-semibold text-neutral-700">
          Order Summary
        </h3>
      </div>

      <div className="p-4 space-y-3">
        {/* Line Items */}
        {showLineItems && lineItems.length > 0 && (
          <div className="space-y-1 pb-3 border-b border-neutral-200">
            {lineItems.map((item) => (
              <LineItemRow
                key={item.id}
                description={item.description}
                quantity={item.quantity}
                unitPrice={item.unitPrice}
                totalPrice={item.totalPrice}
                currency={currency}
              />
            ))}
          </div>
        )}

        {/* Subtotal */}
        <SummaryRow
          label="Subtotal"
          value={formatCurrency(subtotal, currency)}
        />

        {/* Discount */}
        {discount && discount > 0 && (
          <div className="flex items-center gap-1.5">
            <Tag className="h-3.5 w-3.5 text-success-600" aria-hidden="true" />
            <SummaryRow
              label={discountCode ? `Discount (${discountCode})` : 'Discount'}
              value={formatCurrency(discount, currency)}
              isDiscount
            />
          </div>
        )}

        {/* Tax */}
        <div className="flex items-center gap-1.5">
          <Percent
            className="h-3.5 w-3.5 text-neutral-400"
            aria-hidden="true"
          />
          <SummaryRow
            label={taxRate ? `Tax (${taxRate}%)` : 'Tax'}
            value={formatCurrency(tax, currency)}
          />
        </div>

        {/* Total */}
        <div className="pt-3 border-t border-neutral-300">
          <SummaryRow
            label="Total"
            value={formatCurrency(total, currency)}
            isBold
            isLarge
          />
        </div>

        {/* Currency Notice */}
        <p className="text-xs text-neutral-500 text-center pt-2">
          All amounts in {currency.toUpperCase()}
        </p>
      </div>
    </div>
  );
}

PaymentSummary.displayName = 'PaymentSummary';

export default PaymentSummary;
