/** A way to pay on the public pay page (T170). */
export type PayMethod = 'bank' | 'card';

export interface PublicPayMethod {
  method: PayMethod;
  /** ServicePro's fee the business pays for this method, decimal dollars. */
  business_fee: string;
}

/** GET /v1/public/invoices/details/:token — the pay page (mockup 17). */
export interface PublicInvoice {
  invoice_number: string;
  issue_date: string;
  due_date: string;
  status: string;
  subtotal: string;
  tax_amount: string;
  total_amount: string;
  amount_paid: string;
  amount_due: string;
  currency: string;
  customer_name?: string;
  po_number?: string;
  business_name: string;
  business_location?: string;
  /** Bank first; empty when the invoice can't be paid online. */
  pay_methods: PublicPayMethod[];
  lines: Array<{
    description: string;
    quantity: string;
    unit_price: string;
    total: string;
  }>;
}
