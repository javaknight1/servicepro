import api from './api';
import type {
  SavedPaymentMethod,
  SavedPaymentMethodsResponse,
  AddPaymentMethodRequest,
  SetupIntentResponse,
  BillingEventsResponse,
  BillingConfig,
} from '@/types/payment';

// Billing API endpoints
export const billingApi = {
  // Get billing configuration (public)
  getConfig: () => api.get<BillingConfig>('/v1/billing/config'),

  // Create a setup intent for adding a payment method
  createSetupIntent: (tenantId: string) =>
    api.post<SetupIntentResponse>(
      `/v1/tenants/${tenantId}/billing/setup-intent`
    ),

  // Get all saved payment methods for a tenant
  getPaymentMethods: (tenantId: string) =>
    api.get<SavedPaymentMethodsResponse>(
      `/v1/tenants/${tenantId}/billing/payment-methods`
    ),

  // Add a new payment method
  addPaymentMethod: (tenantId: string, data: AddPaymentMethodRequest) =>
    api.post<SavedPaymentMethod>(
      `/v1/tenants/${tenantId}/billing/payment-methods`,
      data
    ),

  // Set a payment method as default
  setDefaultPaymentMethod: (tenantId: string, paymentMethodId: string) =>
    api.put(
      `/v1/tenants/${tenantId}/billing/payment-methods/${paymentMethodId}/default`
    ),

  // Delete a payment method
  deletePaymentMethod: (tenantId: string, paymentMethodId: string) =>
    api.delete(
      `/v1/tenants/${tenantId}/billing/payment-methods/${paymentMethodId}`
    ),

  // Get billing history
  getBillingHistory: (tenantId: string, page = 1, pageSize = 20) =>
    api.get<BillingEventsResponse>(`/v1/tenants/${tenantId}/billing/history`, {
      params: { page, page_size: pageSize },
    }),
};

export default billingApi;
