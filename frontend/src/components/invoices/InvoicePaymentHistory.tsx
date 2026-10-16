import type { ReactNode } from 'react';
import { CreditCard } from 'lucide-react';
import type { Invoice } from '@services/invoiceService';
import { formatDollars } from '@/utils/money';
import { MANUAL_PAYMENT_METHODS } from './RecordPaymentModal';

function methodLabel(method?: string, stripe?: boolean): string {
  if (stripe) return 'Online (Stripe)';
  if (!method) return '-';
  const known = MANUAL_PAYMENT_METHODS.find((m) => m.value === method);
  return known ? known.label : method.replace(/_/g, ' ');
}

function formatDate(value?: string): string {
  if (!value) return '-';
  // Dates arrive as RFC 3339 (date columns at midnight UTC); show the day.
  const day = value.split('T')[0];
  const parsed = new Date(`${day}T00:00:00`);
  return Number.isNaN(parsed.getTime()) ? day : parsed.toLocaleDateString();
}

interface InvoicePaymentHistoryProps {
  invoice: Pick<
    Invoice,
    'total_amount' | 'amount_paid' | 'amount_due' | 'payments' | 'refunds'
  >;
  /** Action buttons (record payment, refund, cancel) shown in the header. */
  actions?: ReactNode;
}

/** Paid / balance summary and the payments and refunds on an invoice. */
export function InvoicePaymentHistory({
  invoice,
  actions,
}: InvoicePaymentHistoryProps) {
  const payments = invoice.payments ?? [];
  const refunds = invoice.refunds ?? [];

  return (
    <div className="bg-white rounded-lg border border-neutral-200 p-6">
      <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
        <h2 className="text-lg font-semibold text-neutral-900 flex items-center gap-2">
          <CreditCard className="h-5 w-5" />
          Payments
        </h2>
        {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
      </div>

      <dl className="grid grid-cols-3 gap-4 mb-4 text-sm">
        <div>
          <dt className="text-neutral-500">Total</dt>
          <dd className="font-medium">{formatDollars(invoice.total_amount)}</dd>
        </div>
        <div>
          <dt className="text-neutral-500">Paid</dt>
          <dd className="font-medium" data-testid="invoice-amount-paid">
            {formatDollars(invoice.amount_paid)}
          </dd>
        </div>
        <div>
          <dt className="text-neutral-500">Balance due</dt>
          <dd className="font-medium" data-testid="invoice-amount-due">
            {formatDollars(invoice.amount_due)}
          </dd>
        </div>
      </dl>

      {payments.length === 0 ? (
        <p className="py-2 text-sm text-neutral-500">
          No payments recorded yet
        </p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-neutral-200">
                <th className="text-left py-2 px-3 font-medium text-neutral-600">
                  Date
                </th>
                <th className="text-left py-2 px-3 font-medium text-neutral-600">
                  Method
                </th>
                <th className="text-left py-2 px-3 font-medium text-neutral-600">
                  Reference
                </th>
                <th className="text-right py-2 px-3 font-medium text-neutral-600">
                  Amount
                </th>
              </tr>
            </thead>
            <tbody>
              {payments.map((payment) => (
                <tr key={payment.id} className="border-b border-neutral-100">
                  <td className="py-2 px-3">
                    {formatDate(payment.payment_date)}
                  </td>
                  <td className="py-2 px-3 capitalize">
                    {methodLabel(
                      payment.payment_method,
                      !!payment.stripe_payment_intent_id
                    )}
                  </td>
                  <td className="py-2 px-3 text-neutral-600">
                    {payment.reference_number || '-'}
                  </td>
                  <td className="py-2 px-3 text-right font-medium">
                    {formatDollars(payment.amount)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {refunds.length > 0 && (
        <div className="mt-6">
          <h3 className="text-sm font-semibold text-neutral-900 mb-2">
            Refunds
          </h3>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-neutral-200">
                  <th className="text-left py-2 px-3 font-medium text-neutral-600">
                    Date
                  </th>
                  <th className="text-left py-2 px-3 font-medium text-neutral-600">
                    Method
                  </th>
                  <th className="text-left py-2 px-3 font-medium text-neutral-600">
                    Reason
                  </th>
                  <th className="text-right py-2 px-3 font-medium text-neutral-600">
                    Amount
                  </th>
                </tr>
              </thead>
              <tbody>
                {refunds.map((refund) => (
                  <tr key={refund.id} className="border-b border-neutral-100">
                    <td className="py-2 px-3">
                      {formatDate(refund.refund_date)}
                    </td>
                    <td className="py-2 px-3 capitalize">
                      {methodLabel(refund.method, !!refund.stripe_charge_id)}
                    </td>
                    <td className="py-2 px-3 text-neutral-600">
                      {refund.reason || '-'}
                    </td>
                    <td className="py-2 px-3 text-right font-medium">
                      -{formatDollars(refund.amount)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
