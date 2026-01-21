import React, { useMemo } from 'react';
import { LineItem } from '../../types/quote';

interface TotalCalculatorProps {
  items: LineItem[];
  taxRate: number;
  className?: string;
  showBreakdown?: boolean;
}

/**
 * TotalCalculator Component
 *
 * Displays real-time calculation of quote totals including:
 * - Subtotal (sum of all line items)
 * - Tax amount (subtotal * tax rate)
 * - Grand total (subtotal + tax)
 *
 * @param items - Array of line items
 * @param taxRate - Tax rate as decimal (e.g., 0.0825 for 8.25%)
 * @param className - Optional CSS classes
 * @param showBreakdown - Show detailed breakdown of each item
 */
export const TotalCalculator: React.FC<TotalCalculatorProps> = ({
  items,
  taxRate,
  className = '',
  showBreakdown = false,
}) => {
  // Calculate totals with memoization for performance
  const totals = useMemo(() => {
    const subtotal = items.reduce((sum, item) => {
      const itemTotal = Number(item.quantity) * Number(item.unit_price);
      return sum + itemTotal;
    }, 0);

    const taxAmount = subtotal * taxRate;
    const total = subtotal + taxAmount;

    return {
      subtotal: subtotal.toFixed(2),
      taxAmount: taxAmount.toFixed(2),
      total: total.toFixed(2),
      taxPercentage: (taxRate * 100).toFixed(2),
    };
  }, [items, taxRate]);

  const formatCurrency = (amount: string): string => {
    return `$${amount}`;
  };

  return (
    <div
      className={`bg-white rounded-lg shadow-sm border border-gray-200 p-6 ${className}`}
    >
      {/* Breakdown Section */}
      {showBreakdown && items.length > 0 && (
        <div className="mb-6 space-y-2">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">
            Breakdown
          </h3>
          <div className="space-y-1 max-h-64 overflow-y-auto">
            {items.map((item, index) => (
              <div
                key={item.id || index}
                className="flex justify-between text-sm text-gray-600 py-1 border-b border-gray-100"
              >
                <span className="truncate mr-4 flex-1">
                  {item.quantity} × {item.description}
                </span>
                <span className="font-medium whitespace-nowrap">
                  {formatCurrency(
                    (Number(item.quantity) * Number(item.unit_price)).toFixed(2)
                  )}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Totals Section */}
      <div className="space-y-3">
        {/* Subtotal */}
        <div className="flex justify-between items-center text-base">
          <span className="text-gray-700">Subtotal</span>
          <span className="font-medium text-gray-900">
            {formatCurrency(totals.subtotal)}
          </span>
        </div>

        {/* Tax */}
        <div className="flex justify-between items-center text-base">
          <span className="text-gray-700">Tax ({totals.taxPercentage}%)</span>
          <span className="font-medium text-gray-900">
            {formatCurrency(totals.taxAmount)}
          </span>
        </div>

        {/* Divider */}
        <div className="border-t-2 border-gray-300 my-2"></div>

        {/* Grand Total */}
        <div className="flex justify-between items-center">
          <span className="text-lg font-bold text-gray-900">Total</span>
          <span className="text-2xl font-bold text-blue-600">
            {formatCurrency(totals.total)}
          </span>
        </div>

        {/* Item Count */}
        {items.length > 0 && (
          <div className="text-sm text-gray-500 text-right mt-2">
            {items.length} {items.length === 1 ? 'item' : 'items'}
          </div>
        )}
      </div>

      {/* Empty State */}
      {items.length === 0 && (
        <div className="text-center py-8">
          <svg
            className="mx-auto h-12 w-12 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z"
            />
          </svg>
          <p className="mt-2 text-sm text-gray-500">No items added yet</p>
          <p className="text-xs text-gray-400 mt-1">
            Add line items to calculate total
          </p>
        </div>
      )}
    </div>
  );
};

export default TotalCalculator;
