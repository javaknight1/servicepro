import api from './api';
import { API_BASE_URL } from '@config/endpoints';
import type { PayMethod, PublicInvoice } from '@/types/publicInvoice';

/** The public pay link (T170): no login, the token is the credential. */
export const publicInvoiceApi = {
  /** The invoice, the business that sent it and how it can be paid. */
  getDetails: (token: string) =>
    api.get<PublicInvoice>(
      `/v1/public/invoices/details/${encodeURIComponent(token)}`
    ),
};

/**
 * The backend route that creates the Stripe Checkout session for the chosen
 * method and redirects to it (a full-page navigation, not an XHR).
 */
export function payInvoiceUrl(token: string, method: PayMethod): string {
  return `${API_BASE_URL}/v1/public/invoices/pay/${encodeURIComponent(token)}?method=${method}`;
}
