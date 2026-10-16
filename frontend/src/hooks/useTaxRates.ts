import { useEffect, useState } from 'react';
import { taxRateService, type TaxRate } from '@services/taxRateService';

/**
 * The tenant's active tax rates plus the global defaults, for the invoice
 * and quote tax-rate pickers. `loaded` turns true once the request settles;
 * on failure the list stays empty and the forms fall back to a typed rate.
 */
export function useTaxRates(): { taxRates: TaxRate[]; loaded: boolean } {
  const [taxRates, setTaxRates] = useState<TaxRate[]>([]);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    let cancelled = false;
    taxRateService
      .listTaxRates()
      .then((rates) => {
        if (!cancelled) setTaxRates(Array.isArray(rates) ? rates : []);
      })
      .catch((err: unknown) => {
        console.error('Failed to load tax rates:', err);
      })
      .finally(() => {
        if (!cancelled) setLoaded(true);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return { taxRates, loaded };
}
