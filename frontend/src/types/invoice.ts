export enum InvoiceStatus {
  DRAFT = 'draft',
  SENT = 'sent',
  VIEWED = 'viewed',
  PAID = 'paid',
  PARTIALLY_PAID = 'partially_paid',
  OVERDUE = 'overdue',
  CANCELLED = 'cancelled',
  REFUNDED = 'refunded',
}

export interface InvoiceLineItem {
  id: string;
  invoice_id?: string;
  description: string;
  quantity: number;
  unit_price: number;
  total: number;
  sort_order: number;
}

export interface Invoice {
  id: string;
  invoice_number: string;
  customer_id: string;
  customer?: {
    id: string;
    email: string;
    first_name: string;
    last_name: string;
    company_name?: string;
  };
  job_id?: string;
  quote_id?: string;
  status: InvoiceStatus;
  issue_date: string;
  due_date: string;
  subtotal: number;
  tax_rate_id?: string;
  tax_amount: number;
  total: number;
  amount_paid: number;
  amount_due: number;
  notes?: string;
  terms_and_conditions?: string;
  lines: InvoiceLineItem[];
  // Payment token fields for online invoice payment
  payment_token?: string;
  payment_token_expires_at?: string;
  stripe_checkout_session_id?: string;
  stripe_payment_intent_id?: string;
  created_at: string;
  updated_at: string;
}

export interface InvoiceFilters {
  customer_id?: string;
  job_id?: string;
  status?: InvoiceStatus;
  search?: string;
  overdue_only?: boolean;
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

export interface InvoiceListResponse {
  invoices: Invoice[];
  total: number;
  page: number;
  page_size: number;
}

/** GET /invoices/stats (T257). Amounts are API decimals (strings). */
export interface InvoiceStats {
  /** Unpaid balance of sent, viewed, partially paid and overdue invoices. */
  total_outstanding: string | number;
  outstanding_invoices: number;
  /** The part of total_outstanding past its due date. */
  overdue_amount: string | number;
  overdue_invoices: number;
  /** Payments less refunds dated this calendar month (UTC). */
  collected_this_month: string | number;
}

export const getInvoiceStatusLabel = (status: InvoiceStatus): string => {
  const labels: Record<InvoiceStatus, string> = {
    [InvoiceStatus.DRAFT]: 'Draft',
    [InvoiceStatus.SENT]: 'Sent',
    [InvoiceStatus.VIEWED]: 'Viewed',
    [InvoiceStatus.PAID]: 'Paid',
    [InvoiceStatus.PARTIALLY_PAID]: 'Partially Paid',
    [InvoiceStatus.OVERDUE]: 'Overdue',
    [InvoiceStatus.CANCELLED]: 'Cancelled',
    [InvoiceStatus.REFUNDED]: 'Refunded',
  };
  return labels[status] || status;
};
