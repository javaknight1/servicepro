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

/** Body of POST /tax-rates and PUT /tax-rates/{id} (omitted = unchanged on
 * update). `rate` is a fraction: 0.0825 = 8.25%. */
export interface TaxRateInput {
  name?: string;
  rate?: number;
  region?: string;
  is_active?: boolean;
}

/** React Query key for the tax-rate list (all rates, active or not). */
export const taxRatesQueryKey = ['tax-rates', 'all'] as const;

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

  /** POST /tax-rates (needs tax_rates.manage). */
  async createTaxRate(input: TaxRateInput): Promise<TaxRate> {
    const response = await api.post<TaxRate>(this.basePath, input);
    return response.data;
  }

  /** PUT /tax-rates/{id} (needs tax_rates.manage); global rates are 403. */
  async updateTaxRate(id: string, input: TaxRateInput): Promise<TaxRate> {
    const response = await api.put<TaxRate>(`${this.basePath}/${id}`, input);
    return response.data;
  }
}

export const taxRateService = new TaxRateService();
