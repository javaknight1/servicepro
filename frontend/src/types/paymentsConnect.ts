/**
 * Stripe Connect payout status for an organization (T169):
 * GET /tenants/:id/payments/status and POST /tenants/:id/payments/connect.
 */

/** Onboarding summary computed by the API. */
export type PaymentsState =
  'not_connected' | 'onboarding' | 'action_required' | 'restricted' | 'active';

/** ServicePro's fees on top of Stripe's (D10). Percent values like "0.4". */
export interface PaymentFees {
  card_percent: string;
  ach_percent: string;
  ach_cap: string;
  currency: string;
}

export interface PaymentsStatus {
  connected: boolean;
  state: PaymentsState;
  /** Platform-wide switch (INVOICE_PAYMENTS_ENABLED). */
  online_payments_enabled: boolean;
  /** Invoice emails carry a pay link for this organization. */
  pay_links_active: boolean;
  /** Stripe capability statuses: '' | active | pending | restricted | rejected | unsupported. */
  card_payments_status: string;
  ach_payments_status: string;
  payouts_status: string;
  /** '' | eventually_due | currently_due | past_due */
  requirements_status: string;
  livemode: boolean;
  synced_at?: string;
  fees: PaymentFees;
}

export interface PaymentsConnectResponse {
  /** Single-use Stripe-hosted onboarding link. */
  url: string;
}
