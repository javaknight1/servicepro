import { useMemo, useEffect } from 'react';
import { UseFormSetValue, UseFormWatch } from 'react-hook-form';
import { LineItem } from '../../../types/quote';
import { QuoteFormData } from '../validation';

/**
 * Hook for real-time quote calculations
 * Automatically calculates subtotal, tax, and total when items or tax rate changes
 */
export const useQuoteCalculations = (
  watch: UseFormWatch<QuoteFormData>,
  setValue: UseFormSetValue<QuoteFormData>
) => {
  // Watch for changes to items and tax information
  const watchedItems = watch('items');
  const items = useMemo(() => watchedItems || [], [watchedItems]);
  const taxRate = watch('tax_rate') || '0';
  const taxExemptionType = watch('tax_exemption_type') || 'none';

  // Calculate subtotal from line items
  const subtotal = useMemo(() => {
    const total = items.reduce((sum, item) => {
      const itemTotal = Number(item.quantity) * Number(item.unit_price);
      return sum + itemTotal;
    }, 0);
    return total;
  }, [items]);

  // Calculate tax amount
  const taxAmount = useMemo(() => {
    // No tax if there's an exemption
    if (taxExemptionType && taxExemptionType !== 'none') {
      return 0;
    }

    const rate = parseFloat(taxRate) || 0;
    return subtotal * rate;
  }, [subtotal, taxRate, taxExemptionType]);

  // Calculate total
  const total = useMemo(() => {
    return subtotal + taxAmount;
  }, [subtotal, taxAmount]);

  // Update form values when calculations change
  useEffect(() => {
    setValue('subtotal', subtotal.toFixed(2), { shouldValidate: false });
    setValue('tax_amount', taxAmount.toFixed(2), { shouldValidate: false });
    setValue('total', total.toFixed(2), { shouldValidate: false });
  }, [subtotal, taxAmount, total, setValue]);

  return {
    subtotal: subtotal.toFixed(2),
    taxAmount: taxAmount.toFixed(2),
    total: total.toFixed(2),
  };
};

/**
 * Hook for calculating line item totals
 */
export const useLineItemCalculation = (
  quantity: number,
  unitPrice: number
): number => {
  return useMemo(() => {
    return quantity * unitPrice;
  }, [quantity, unitPrice]);
};

/**
 * Hook for tax rate lookup by ZIP code
 */
export const useTaxRateLookup = (zipCode: string) => {
  const [isLoading, setIsLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [taxRate, setTaxRate] = React.useState<string>('0');

  useEffect(() => {
    // Validate ZIP code format
    const zipRegex = /^\d{5}(-\d{4})?$/;
    if (!zipCode || !zipRegex.test(zipCode)) {
      setTaxRate('0');
      setError(null);
      return;
    }

    // Lookup tax rate
    const lookupTaxRate = async () => {
      setIsLoading(true);
      setError(null);

      try {
        // This would call the tax service API
        // For now, using a simple lookup table
        const response = await fetch(`/api/v1/tax/rate?zip_code=${zipCode}`);

        if (!response.ok) {
          throw new Error('Failed to fetch tax rate');
        }

        const data = await response.json();
        setTaxRate(data.combined_rate || '0');
      } catch (err) {
        setError(
          err instanceof Error ? err.message : 'Failed to lookup tax rate'
        );
        setTaxRate('0');
      } finally {
        setIsLoading(false);
      }
    };

    // Debounce the lookup
    const timeoutId = setTimeout(() => {
      lookupTaxRate();
    }, 500);

    return () => clearTimeout(timeoutId);
  }, [zipCode]);

  return { taxRate, isLoading, error };
};

/**
 * Hook for managing quote totals display
 */
export const useQuoteTotals = (
  items: LineItem[],
  taxRate: string,
  taxExemptionType?: string
) => {
  return useMemo(() => {
    // Calculate subtotal
    const subtotal = items.reduce((sum, item) => {
      return sum + item.quantity * item.unit_price;
    }, 0);

    // Calculate tax
    const rate = parseFloat(taxRate) || 0;
    const isExempt = taxExemptionType && taxExemptionType !== 'none';
    const tax = isExempt ? 0 : subtotal * rate;

    // Calculate total
    const total = subtotal + tax;

    return {
      subtotal: subtotal.toFixed(2),
      tax: tax.toFixed(2),
      total: total.toFixed(2),
      itemCount: items.length,
      averageItemValue:
        items.length > 0 ? (subtotal / items.length).toFixed(2) : '0.00',
    };
  }, [items, taxRate, taxExemptionType]);
};

/**
 * Import React for useTaxRateLookup
 */
import * as React from 'react';
