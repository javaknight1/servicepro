import api from './api';

/** A payment term (`models.PaymentTerm`), global or the tenant's own. */
export interface PaymentTerm {
  id: string;
  name: string;
  term_type: string;
  days_until_due: number;
  is_default: boolean;
  is_active: boolean;
}

class PaymentTermService {
  private basePath = '/v1/payment-terms';

  /** GET /payment-terms/{id} (needs payment_terms.read). */
  async getPaymentTerm(id: string): Promise<PaymentTerm> {
    const response = await api.get<PaymentTerm>(`${this.basePath}/${id}`);
    return response.data;
  }
}

export const paymentTermService = new PaymentTermService();
