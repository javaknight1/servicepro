import api from './api';

/** A configured tax rate (`models.TaxRate`): the tenant's own, or a global
 * default when `tenant_id` is absent. */
export interface TaxRate {
  id: string;
  tenant_id?: string;
  name: string;
  /** Fraction as a decimal string ("0.0825" = 8.25%). */
  rate: string | number;
  tax_type: string;
  region?: string;
  is_active: boolean;
}

class TaxRateService {
  private basePath = '/v1/tax-rates';

  /** GET /tax-rates (needs tax_rates.read): the tenant's rates plus the
   * global defaults, ordered by name. */
  async listTaxRates(activeOnly = true): Promise<TaxRate[]> {
    const response = await api.get<TaxRate[]>(this.basePath, {
      params: activeOnly ? { active_only: true } : undefined,
    });
    return response.data;
  }
}

export const taxRateService = new TaxRateService();
