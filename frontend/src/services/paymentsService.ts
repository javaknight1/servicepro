import api from './api';
import type {
  PaymentsConnectResponse,
  PaymentsStatus,
} from '@/types/paymentsConnect';

/** Stripe Connect payouts for an organization (T169). */
export const paymentsApi = {
  /** Whether the organization can take online payments, and its fees. */
  getStatus: (tenantId: string) =>
    api.get<PaymentsStatus>(`/v1/tenants/${tenantId}/payments/status`),

  /**
   * Creates the organization's Stripe account on first use and returns a
   * single-use onboarding link to redirect to.
   */
  connect: (tenantId: string) =>
    api.post<PaymentsConnectResponse>(
      `/v1/tenants/${tenantId}/payments/connect`
    ),
};

/** React Query key for an organization's payments status. */
export const paymentsStatusQueryKey = (tenantId: string | null | undefined) =>
  ['payments-status', tenantId ?? null] as const;
